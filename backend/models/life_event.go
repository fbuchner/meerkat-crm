package models

import (
	"time"

	"mycorrhizal/contactmodel"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Conventional (not validated — same open-classifier reasoning as
// Activity.Type/InteractionType* above and HouseholdMember.Role) values for
// LifeEvent.Type. §91.6 lists these with a trailing "…", signalling an open,
// extensible set.
const (
	LifeEventTypeMarried    = "married"
	LifeEventTypeGraduated  = "graduated"
	LifeEventTypeJobChange  = "job_change"
	LifeEventTypeHadChild   = "had_child"
	LifeEventTypeAdoptedPet = "adopted_pet"
	LifeEventTypeRetired    = "retired"
	LifeEventTypeMoved      = "moved"
)

// Provenance values stored on LifeEvent.Source, following
// RelationshipEdge.Source's per-entity-local-enum convention.
const (
	LifeEventSourceUser        = "user"
	LifeEventSourceImported    = "imported"
	LifeEventSourceAISuggested = "ai-suggested"
)

// LifeEvent is a permanent fact about an entity's life (docs/fork-plan/
// 91-envelope-data-model.md §91.6) — "what happened in *their* life", as
// opposed to Interaction/Activity ("what happened between *us*", §91.7).
//
// UUID-string-primary-key entity, following Household's exact template
// (household.go): ID generated in BeforeCreate. Soft-deletes (deleted_at)
// added per T5 — LifeEvent is first-class user-authored content, same shape
// as Note, not a graph-adjacent join row.
type LifeEvent struct {
	ID        string         `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID uint `gorm:"not null;index" json:"-"`

	// EntityID is the subject Contact, referenced by Contact.VCardUID — the
	// same graph invariant RelationshipEdge.SourceID/TargetID and
	// HouseholdMember.MemberVCardUID follow (§90 D3).
	EntityID string `gorm:"column:entity_id;not null;index" json:"entity_id" validate:"required,uuid4"`

	// Type is conventional/unvalidated — see the LifeEventType* constants
	// above.
	Type string `json:"type,omitempty"`

	// Date reuses contactmodel.PartialDate per §91.6's own instruction
	// ("life events are often known only to a year"), JSON-serialized like
	// Household.Address.
	Date *contactmodel.PartialDate `gorm:"type:text;serializer:json" json:"date,omitempty"`

	Description string `json:"description,omitempty" validate:"max=2000"`

	Source string `json:"source,omitempty" validate:"omitempty,oneof=user imported ai-suggested"`

	// RelatedEntityIDs holds other Contact.VCardUIDs this event involves —
	// covering both §91.6's "secondary participants" (e.g. both spouses in a
	// married event) and "related_entity_ids" (the new child, the pet
	// adopted, the org joined) with a single JSON array rather than a
	// dedicated join table, since nothing needs to query from the
	// related-entity side yet. Reuses Contact.Circles' own serialization
	// style (models/contact.go).
	RelatedEntityIDs []string `gorm:"type:text;serializer:json" json:"related_entity_ids,omitempty"`

	// Remind, when true, opts this event into automatic yearly reminder
	// generation (T5b). Only meaningful when Date has month/day — year-only
	// events have nothing to anchor a yearly recurrence to.
	Remind bool `gorm:"default:false" json:"remind,omitempty"`
}

// BeforeCreate generates a UUID for new LifeEvents, mirroring Household's own
// BeforeCreate.
func (l *LifeEvent) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}
