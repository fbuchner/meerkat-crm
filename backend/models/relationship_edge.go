package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Provenance values stored on RelationshipEdge.Source.
const (
	RelationshipSourceUserConfirmed     = "user-confirmed"
	RelationshipSourceHouseholdInferred = "household-inferred"
	RelationshipSourceImported          = "imported"
	RelationshipSourceAISuggested       = "ai-suggested"
)

// Status values stored on RelationshipEdge.Status.
const (
	// RelationshipStatusConfirmed edges are authoritative: they are eligible
	// to project into Card.RelatedTo on export (models/contact_record.go).
	RelationshipStatusConfirmed = "confirmed"
	// RelationshipStatusSuggested edges await user approval (WP-83's
	// household-inference mechanism is the first producer of these) and are
	// never treated as fact — never projected, never read as a hard edge by
	// anything outside a review surface.
	RelationshipStatusSuggested = "suggested"
)

// Sensitivity values stored on RelationshipEdge.Sensitivity.
const (
	RelationshipSensitivityNormal  = "normal"
	RelationshipSensitivityPrivate = "private"
	RelationshipSensitivitySecret  = "secret"
)

// RelationshipEdge is the relationship graph's edge entity (docs/fork-plan/
// 91-envelope-data-model.md §91.2) — the internal source of truth for
// relationships between two Contacts, per docs/fork-plan/90-vision-and-
// reconciliation.md D3. Distinct from the legacy models.Relationship (a flat,
// free-text table this entity is not built on top of and does not yet
// replace — that cutover is WP-81, a separate later work package).
//
// SourceID/TargetID reference Contact.VCardUID (models/contact.go), which is
// already a UUID generated for every Contact unconditionally — no new column
// on Contact was needed for this. Only one direction of an edge is ever
// stored ("parent_of" from A to B, never also "child_of" from B to A); the
// reciprocal is always derived from the type registry (relationship_type_
// registry.go's InverseRelationType), never persisted, so there is nothing
// to keep in sync.
//
// This is the first UUID-string-primary-key entity in this codebase — every
// other model uses gorm.Model's uint autoincrement PK. ID is generated in
// BeforeCreate, mirroring Contact.VCardUID's generation (models/contact.go).
//
// No soft-delete: matches ContactSyncLink/CalendarEventLink's precedent for
// edge/join-shaped rows that don't need one.
type RelationshipEdge struct {
	ID        string    `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID uint `gorm:"not null;index" json:"-"`

	// SourceID/TargetID are Contact.VCardUID values, not Contact.ID — the
	// graph invariant (§90 D3) that every relationship endpoint is an entity
	// referenced by its stable UUID, never a bare string.
	SourceID string `gorm:"not null;index" json:"source_id" validate:"required,uuid4"`
	TargetID string `gorm:"not null;index" json:"target_id" validate:"required,uuid4"`

	// Type is the social role (e.g. "parent_of", "spouse_of") — see
	// relationship_type_registry.go for the full registered set. The real-
	// world nuance (biological/adoptive/step, poly descriptors, custody...)
	// belongs in Metadata, never in a new Type token (§91.2's "type = role,
	// metadata = nature" rule).
	Type string `gorm:"not null;index" json:"type" validate:"required,relation_type"`

	// Directional is per-edge, not derived from the registry: most relation
	// types have one conventional default (spouse_of is normally symmetric,
	// parent_of is normally directional), but §91.2's affinity edges are the
	// explicit exception — conflicts_with defaults symmetric but may be
	// asserted directionally ("Marley is aggressive toward Gimley
	// specifically"). Callers set this at creation time.
	Directional bool `gorm:"not null" json:"directional"`

	// Metadata is free-form (since/until dates, {kind: biological|adoptive|
	// step|chosen}, poly descriptors, affinity strength/context, ...) — the
	// mechanism that lets a small, stable Type vocabulary express real-world
	// variety without a new token per nuance. First map[string]interface{}
	// JSON column in this codebase; uses the same gorm serializer mechanism
	// as the first typed one, Contact.CustomFields (models/contact.go).
	Metadata map[string]interface{} `gorm:"type:text;serializer:json" json:"metadata,omitempty"`

	// Source is provenance: how this edge came to exist. Confidence is 0-1;
	// user-confirmed edges are always 1.0, inferred/suggested ones lower.
	Source     string  `gorm:"not null" json:"source" validate:"required,oneof=user-confirmed household-inferred imported ai-suggested"`
	Confidence float64 `gorm:"not null;default:1" json:"confidence" validate:"gte=0,lte=1"`

	// Status gates authority: only "confirmed" edges are ever projected to
	// Card.RelatedTo or treated as fact by anything outside a suggestion-
	// review surface. See the RelationshipStatus* constants above.
	Status string `gorm:"not null;index" json:"status" validate:"required,oneof=confirmed suggested"`

	// Sensitivity is the cross-cutting marker from §91.13. Anything above
	// "normal" is excluded by default from exports (enforced in models/
	// contact_record.go's projection step), external sync, and shared views.
	// Ships as a column now rather than being retrofitted later.
	Sensitivity string `gorm:"not null;default:normal;index" json:"sensitivity" validate:"required,oneof=normal private secret"`

	// LegacyRelationshipID is WP-81 migration bookkeeping: the legacy
	// models.Relationship.ID this edge was migrated from, or nil for every
	// edge created any other way (user-created, WP-83 household-suggested,
	// ...). Unique where non-null so cmd/backfill-relationship-edges can tell
	// "already migrated" from "not yet" and be safely re-run. Not read by any
	// application code — only by that tool.
	LegacyRelationshipID *uint `gorm:"uniqueIndex" json:"-"`
}

// BeforeCreate generates a UUID for new edges, mirroring Contact.VCardUID's
// generation (models/contact.go's BeforeCreate).
func (r *RelationshipEdge) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}
