package models

// ContactMergeFieldConflict is one field where the keep and merge contacts
// disagree and a user decision is required. Covers two distinct kinds of
// conflict, kept in separate slices on ContactMergeResolution rather than
// disambiguated by a prefix on Field: a scalar Contact field (Field is the
// fixed snake_case key from contact_merge_service.go's mergeScalarFields,
// e.g. "firstname") or a typed custom-field value (Field is the
// FieldDefinition.ID UUID, since FieldValue rows -- unlike every other
// association -- cannot be unioned: the schema allows only one value per
// field per contact).
type ContactMergeFieldConflict struct {
	Field       string `json:"field"`
	Label       string `json:"label"`
	KeeperValue string `json:"keeper_value"`
	LoserValue  string `json:"loser_value"`
}

// ContactMergeResolution is the pure (no DB access, beyond FieldValueConflicts
// which is computed separately -- see ComputeFieldValueConflicts) computed
// outcome of merging two contacts' field-level data: the multi-valued unions
// (deduplicated), the non-conflicting scalar outcomes, and the conflict sets
// the user must resolve. Produced by services.ComputeContactMergeResolution
// and shared verbatim by the preview and commit endpoints so they can never
// compute a different answer for the same pair.
type ContactMergeResolution struct {
	Emails       []ContactEmail    `json:"emails"`
	Phones       []ContactPhone    `json:"phones"`
	Addresses    []ContactAddress  `json:"addresses"`
	URLs         []ContactURL      `json:"urls"`
	IMPPs        []ContactIMPP     `json:"impps"`
	Circles      []string          `json:"circles"`
	CustomFields map[string]string `json:"custom_fields"`

	// ResolvedScalars holds every scalar field's outcome that did NOT need a
	// user decision: keyed same as Conflicts[i].Field. Includes fields where
	// both sides already agreed (value unchanged) so the map is a complete
	// record of every scalar's post-merge value, not just the changed ones.
	ResolvedScalars map[string]string `json:"resolved_scalars"`

	// Conflicts is empty when nothing needs a user decision -- in a real
	// duplicate pair this is typically a handful of fields and often zero,
	// since duplicates usually arise from one sparse record and one full one.
	Conflicts []ContactMergeFieldConflict `json:"conflicts"`

	// FieldValueConflicts is the FieldValue-specific conflict set (see
	// ContactMergeFieldConflict's doc comment). Computed separately by
	// services.ComputeFieldValueConflicts since it requires a DB read, unlike
	// every other part of this struct.
	FieldValueConflicts []ContactMergeFieldConflict `json:"field_value_conflicts"`
}

// ContactMergeAssociationCounts is how many rows of each association type the
// loser contact holds. Used two ways: the preview response ("this will merge
// N notes, M edges, ...", computed pre-merge) and the commit audit note
// (computed pre-merge inside the same transaction, so it reflects exactly
// what RepointContactAssociations is about to move).
type ContactMergeAssociationCounts struct {
	Notes                int64 `json:"notes"`
	Activities           int64 `json:"activities"`
	Reminders            int64 `json:"reminders"`
	ReminderCompletions  int64 `json:"reminder_completions"`
	RelationshipEdges    int64 `json:"relationship_edges"`
	HouseholdMemberships int64 `json:"household_memberships"`
	CircleMemberships    int64 `json:"circle_memberships"`
	Tags                 int64 `json:"tags"`
	LifeEvents           int64 `json:"life_events"`           // loser is the subject (entity_id)
	LifeEventReferences  int64 `json:"life_event_references"` // loser appears in someone else's RelatedEntityIDs
	FieldValues          int64 `json:"field_values"`
	ContactSyncLinks     int64 `json:"contact_sync_links"` // discarded, not re-pointed
}

// ContactMergeRequest is the DTO for both merge endpoints. KeepID survives;
// MergeID is absorbed and (commit only) soft-deleted. Resolutions is read
// only by the commit endpoint (POST /contacts/merge) -- a field->chosen-value
// map covering both scalar conflicts (keyed by fixed field name) and
// FieldValue conflicts (keyed by FieldDefinition.ID); the two key spaces
// don't collide in practice, so one flat map keeps the frontend simple.
// Ignored (may be omitted) on the preview endpoint.
//
// KeepID == MergeID is checked in the controller, not via a validate tag --
// same reasoning RelationshipEdgeInput's doc comment gives: this codebase has
// no existing precedent for required_without/excluded_with/nefield, and a
// controller-level check gives a much more precise per-field error.
type ContactMergeRequest struct {
	KeepID      uint              `json:"keep_id" validate:"required,gt=0"`
	MergeID     uint              `json:"merge_id" validate:"required,gt=0"`
	Resolutions map[string]string `json:"resolutions,omitempty"`
}

// ContactMergePreviewResponse is the response for POST /contacts/merge/preview.
type ContactMergePreviewResponse struct {
	KeepID            uint                          `json:"keep_id"`
	MergeID           uint                          `json:"merge_id"`
	Resolution        ContactMergeResolution        `json:"resolution"`
	AssociationCounts ContactMergeAssociationCounts `json:"association_counts"`
}
