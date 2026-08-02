package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category values stored on Preference.Category — the conventional, open set
// from docs/fork-plan/91-envelope-data-model.md §91.9 (trailing "…" there:
// food, drink, clothing_size, hobby, gift, dislike, media, …). Deliberately
// NOT validated with a `oneof` tag, matching the open-classifier reasoning of
// LifeEvent.Type/HouseholdMember.Role: the suggestion engine (and future AI
// layer) must degrade gracefully on a category this switch doesn't know,
// not reject it.
const (
	PreferenceCategoryFood         = "food"
	PreferenceCategoryDrink        = "drink"
	PreferenceCategoryClothingSize = "clothing_size"
	PreferenceCategoryHobby        = "hobby"
	PreferenceCategoryGift         = "gift"
	PreferenceCategoryDislike      = "dislike"
	PreferenceCategoryMedia        = "media"
)

// Source values stored on Preference.Source — provenance, §91.9's own
// closed list (conversation_note, user, ai-suggested, external).
const (
	PreferenceSourceConversationNote = "conversation_note"
	PreferenceSourceUser             = "user"
	PreferenceSourceAISuggested      = "ai-suggested"
	PreferenceSourceExternal         = "external"
)

// Preference is one structured personal fact about an entity
// (docs/fork-plan/91-envelope-data-model.md §91.9) — "important info that
// currently only lives in notes or the single Contact.FoodPreference
// free-text field", generalized: a category/key/value triple with provenance
// and staleness tracking.
//
// UUID-string-primary-key entity, following LifeEvent's exact template
// (life_event.go): ID generated in BeforeCreate, soft-deletes. Preferences
// are user-authored content (a fact about a person), not a graph-adjacent
// join row, so they follow LifeEvent's soft-delete choice — and because they
// have no natural-key unique constraint (a person may hold several food
// preferences, several hobbies, ...), a soft-deleted row never blocks
// re-creation, which is the trap that forces hard-delete on join rows.
type Preference struct {
	ID        string         `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID uint `gorm:"not null;index" json:"-"`

	// EntityID is the subject Contact, referenced by Contact.VCardUID — the
	// same graph invariant every join/reference entity follows (§90 D3).
	EntityID string `gorm:"column:entity_id;not null;index" json:"entity_id" validate:"required,uuid4"`

	// Category is the open classifier above (food, hobby, gift, ...). Key is
	// an optional structured qualifier (e.g. "favorite_coffee"), matching
	// §91.9's key/value pair.
	Category string `gorm:"not null" json:"category" validate:"required,max=100"`
	Key      string `json:"key,omitempty" validate:"omitempty,max=100"`
	Value    string `gorm:"not null" json:"value" validate:"required,max=1000"`

	// Source is provenance (§91.9's closed set above). Confidence/
	// LastConfirmed implement §91.9's "preferences go stale; track when last
	// confirmed".
	Source        string     `json:"source,omitempty" validate:"omitempty,oneof=conversation_note user ai-suggested external"`
	Confidence    *float64   `json:"confidence,omitempty"`
	LastConfirmed *time.Time `json:"last_confirmed,omitempty"`

	// Sensitivity reuses the cross-cutting normal/private/secret set
	// (§91.13), driving the projectPreferences query filter — anything above
	// normal is excluded from Card.PersonalInfo projection in the query, the
	// way projectCustomFields filters field_definitions.
	Sensitivity string `gorm:"not null;default:normal;index" json:"sensitivity" validate:"required,oneof=normal private secret"`
}

// BeforeCreate generates a UUID for new Preferences, mirroring LifeEvent's
// own BeforeCreate.
func (p *Preference) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}
