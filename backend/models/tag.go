package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tag is an attribute a set of people share, even if they don't know each
// other (docs/fork-plan/91-envelope-data-model.md §91.5) — a property of the
// person, distinct from Circle (a social grouping, circle.go). Unlike
// Circle, Tag has a standards home: it projects onto Card.Keywords (vCard
// CATEGORIES / JSContact keywords) — see projectTags in contact_record.go.
//
// UUID-string-primary-key entity, following Household's exact template
// (household.go): ID generated in BeforeCreate, no soft-delete.
type Tag struct {
	ID        string    `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID uint   `gorm:"not null;index" json:"-"`
	Name   string `gorm:"not null" json:"name"`
}

// BeforeCreate generates a UUID for new Tags, mirroring Household's own
// BeforeCreate.
func (t *Tag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// ContactTag is the tagging join (§91.5's "taggings (contact ↔ tag)"),
// following HouseholdMember's exact template (household.go): uint primary
// key, no soft-delete, ContactVCardUID is a Contact.VCardUID (the graph
// invariant), unique per (tag, contact).
type ContactTag struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	TagID  string `gorm:"not null;index;uniqueIndex:idx_contact_tag,priority:1" json:"tag_id"`
	UserID uint   `gorm:"not null;index" json:"-"`

	// Explicit column name, matching CircleMember.MemberVCardUID's own
	// reasoning — GORM's default namer would otherwise split "VCardUID".
	ContactVCardUID string `gorm:"column:contact_vcard_uid;not null;index;uniqueIndex:idx_contact_tag,priority:2" json:"contact_vcard_uid" validate:"required,uuid4"`
}
