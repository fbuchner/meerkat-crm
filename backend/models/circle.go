package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Circle is a social grouping — "how I know them / that they likely know
// each other" (docs/fork-plan/91-envelope-data-model.md §91.5). Distinct
// from Tag (an attribute of the person, tag.go): a circle says something
// about *my* connection/context, not about the contact as an individual, so
// it has no standards projection.
//
// UUID-string-primary-key entity, following Household's exact template
// (household.go): ID generated in BeforeCreate, no soft-delete.
type Circle struct {
	ID        string    `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID uint   `gorm:"not null;index" json:"-"`
	Name   string `gorm:"not null" json:"name"`
}

// BeforeCreate generates a UUID for new Circles, mirroring Household's own
// BeforeCreate.
func (ci *Circle) BeforeCreate(tx *gorm.DB) error {
	if ci.ID == "" {
		ci.ID = uuid.New().String()
	}
	return nil
}

// CircleMember is the membership join (§91.5's "members" — circle_members
// join → entity UUIDs), following HouseholdMember's exact template
// (household.go): uint primary key, no soft-delete, MemberVCardUID is a
// Contact.VCardUID (the graph invariant), unique per (circle, member).
type CircleMember struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	CircleID string `gorm:"not null;index;uniqueIndex:idx_circle_member,priority:1" json:"circle_id"`
	UserID   uint   `gorm:"not null;index" json:"-"`

	// Explicit column name, matching HouseholdMember.MemberVCardUID's own
	// fix — GORM's default namer would otherwise split "VCardUID" into
	// v_card_uid, mismatching the raw-SQL migration's member_vcard_uid-style
	// column (a caught, documented bug from WP-83, not a hypothetical here).
	MemberVCardUID string `gorm:"column:member_vcard_uid;not null;index;uniqueIndex:idx_circle_member,priority:2" json:"member_vcard_uid" validate:"required,uuid4"`
}
