package main

import (
	"errors"
	"fmt"
	"strings"

	"mycorrhizal/models"

	"gorm.io/gorm"
)

// migrationOutcome summarizes what migrateOneRelationship did (or, in dry
// run, would do) for one legacy relationship row — used for both the CLI's
// per-row report line and direct assertions in migrate_test.go.
type migrationOutcome struct {
	// Skipped is non-empty when this row was not (and, in dry run, would not
	// be) migrated; its value is the human-readable reason.
	Skipped string

	SourceID           string // resolved Contact.VCardUID for the edge's source (empty if a thin contact is still pending in dry run)
	TargetID           string // resolved Contact.VCardUID for the edge's target (the legacy row's owning contact)
	CreatedThinContact bool
	ThinContactName    string

	MatchedType  string // registry token
	UsedFallback bool   // true if MatchedType == "related_to" (no confident match found)

	// EdgeID is populated only when write=true and the edge was actually
	// persisted (empty in dry run, even on an otherwise-successful outcome).
	EdgeID string
}

// migrateOneRelationship migrates a single legacy models.Relationship row
// into a RelationshipEdge, per docs/fork-plan/92-delivery-roadmap.md's
// WP-81 entry and docs/fork-plan/90-vision-and-reconciliation.md D3.
//
// Direction: legacy Relationship{ContactID: Bob, Name/RelatedContactID:
// Alice, Type: "Mother"} means "Alice is Bob's mother" — the same convention
// vCard RELATED and this codebase's own type registry already use (type
// describes what the *other* party is *to* the holder). So the resulting
// edge is SourceID=Alice (or a new thin Contact for a name-only row),
// TargetID=Bob (the legacy row's own ContactID), Type=the matched token —
// no direction-flipping needed.
//
// write=false performs every read and resolution step, so the report line
// reflects exactly what a real run would do, but makes zero database writes
// — no thin Contact is created, no RelationshipEdge is persisted. Idempotent
// via RelationshipEdge.LegacyRelationshipID: a row already migrated is
// reported as skipped rather than re-migrated, unless force is true.
func migrateOneRelationship(db *gorm.DB, legacy models.Relationship, write, force bool) (migrationOutcome, error) {
	// Looked up regardless of force: when found and forcing, `existing` is
	// reused below as the base for an UPDATE in place, rather than
	// constructing a fresh struct and INSERTing a second edge for the same
	// legacy row — which the legacy_relationship_id unique index (rightly)
	// refuses, and which would also lose the original CreatedAt.
	var existing models.RelationshipEdge
	err := db.Where("legacy_relationship_id = ?", legacy.ID).First(&existing).Error
	switch {
	case err == nil:
		if !force {
			return migrationOutcome{Skipped: fmt.Sprintf("already migrated to edge %s (pass -force to reprocess)", existing.ID)}, nil
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		// Not yet migrated — the common case. `existing` stays the zero
		// value, so the Save() below performs an insert.
	default:
		return migrationOutcome{}, fmt.Errorf("checking idempotency for legacy id=%d: %w", legacy.ID, err)
	}

	if legacy.RelatedContactID != nil && *legacy.RelatedContactID == legacy.ContactID {
		return migrationOutcome{Skipped: "self-referencing relationship (contact_id == related_contact_id)"}, nil
	}

	var owner models.Contact
	if err := db.First(&owner, legacy.ContactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Shouldn't happen — contact_controller.go's cascade delete
			// removes a contact's own Relationship rows when that contact is
			// deleted — but a migration touching irreversible, user-visible
			// data should not assume an invariant it isn't the one enforcing,
			// and should never abort the whole run over one bad row.
			return migrationOutcome{Skipped: fmt.Sprintf("owning contact id=%d no longer exists", legacy.ContactID)}, nil
		}
		return migrationOutcome{}, fmt.Errorf("loading owning contact id=%d: %w", legacy.ContactID, err)
	}

	var outcome migrationOutcome
	var relatedVCardUID string

	if legacy.RelatedContactID != nil {
		var related models.Contact
		if err := db.First(&related, *legacy.RelatedContactID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// A known, pre-existing gap: deleting a contact only cleans
				// up Relationship rows where IT is the owner (contact_id),
				// not rows where it's the target (related_contact_id) — so a
				// dangling pointer here is real, not hypothetical.
				return migrationOutcome{Skipped: fmt.Sprintf("related contact id=%d no longer exists (dangling pointer)", *legacy.RelatedContactID)}, nil
			}
			return migrationOutcome{}, fmt.Errorf("loading related contact id=%d: %w", *legacy.RelatedContactID, err)
		}
		relatedVCardUID = related.VCardUID
	} else {
		name := strings.TrimSpace(legacy.Name)
		if name == "" {
			return migrationOutcome{Skipped: "empty name on a name-only relationship"}, nil
		}

		outcome.ThinContactName = name

		if write {
			// A -force reprocess of an already-migrated name-only row must
			// reuse the thin Contact created on the FIRST pass (found via
			// existing.SourceID, the prior edge's source), refreshing its
			// Firstname/Gender/Birthday from the current legacy row — never
			// create a second Contact for the same legacy row, which would
			// otherwise happen silently on every -force run.
			var thin models.Contact
			reused := false
			if existing.ID != "" && existing.SourceID != "" {
				if err := db.Where("vcard_uid = ?", existing.SourceID).First(&thin).Error; err == nil {
					reused = true
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return migrationOutcome{}, fmt.Errorf("loading previously-created thin contact for legacy id=%d: %w", legacy.ID, err)
				}
				// RecordNotFound (the thin contact was since deleted): fall
				// through and create a fresh one below.
			}

			// D3: "legacy models.Relationship rows that carry only a Name
			// string with no RelatedContactID are migrated by promoting that
			// name into a new thin Contact ... no edge is ever left
			// referencing a bare string." The whole free-text Name goes into
			// Firstname verbatim — splitting "John Smith" into first/last is
			// fragile guessing the spec doesn't ask for, and a single-field
			// name is a valid thin entity. Gender/Birthday move onto the new
			// Contact's own fields, since RelationshipEdge has no home for
			// either — this is exactly why the promotion matters, not an
			// afterthought.
			thin.UserID = legacy.UserID
			thin.Firstname = name
			thin.Gender = legacy.Gender
			thin.Birthday = legacy.Birthday

			if err := db.Save(&thin).Error; err != nil {
				return migrationOutcome{}, fmt.Errorf("saving thin contact %q for legacy id=%d: %w", name, legacy.ID, err)
			}
			relatedVCardUID = thin.VCardUID
			outcome.CreatedThinContact = !reused
		} else {
			outcome.CreatedThinContact = existing.ID == ""
		}
		// In dry run, relatedVCardUID stays "" — nothing was created, so
		// there is no real UID to show, and fabricating one would misleadingly
		// suggest a specific value that a real -write run wouldn't reproduce
		// (VCardUIDs are random).
	}

	token, matched := models.MatchLegacyRelationType(legacy.Type)
	confidence := 1.0
	if !matched {
		// related_to: the relationship's existence is certain (the user
		// asserted it, however it was typed in) — only the specific type
		// category is uncertain. Status stays "confirmed" for exactly that
		// reason; only Confidence reflects the categorization uncertainty.
		token = "related_to"
		confidence = 0.5
	}
	outcome.MatchedType = token
	outcome.UsedFallback = !matched
	outcome.SourceID = relatedVCardUID
	outcome.TargetID = owner.VCardUID

	if !write {
		return outcome, nil
	}

	var metadata map[string]interface{}
	if !matched {
		metadata = map[string]interface{}{"legacy_type": legacy.Type}
	}

	// Start from the previously-loaded row when force-reprocessing (it
	// already carries ID/CreatedAt from the DB) rather than a fresh literal —
	// Save() performs a full-column update, so a fresh literal's zero-value
	// CreatedAt would silently overwrite the real one.
	edge := existing
	legacyID := legacy.ID
	edge.UserID = legacy.UserID
	edge.SourceID = outcome.SourceID
	edge.TargetID = outcome.TargetID
	edge.Type = token
	edge.Directional = !models.IsSymmetricRelationType(token)
	edge.Metadata = metadata
	edge.Source = models.RelationshipSourceImported
	edge.Confidence = confidence
	edge.Status = models.RelationshipStatusConfirmed
	edge.Sensitivity = models.RelationshipSensitivityNormal
	edge.LegacyRelationshipID = &legacyID

	if err := db.Save(&edge).Error; err != nil {
		return migrationOutcome{}, fmt.Errorf("saving relationship edge for legacy id=%d: %w", legacy.ID, err)
	}
	outcome.EdgeID = edge.ID

	return outcome, nil
}
