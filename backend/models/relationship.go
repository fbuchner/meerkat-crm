package models

import (
	"gorm.io/gorm"
)

type Relationship struct {
	gorm.Model
	Name             string   `json:"name" validate:"required,min=1,max=100,safe_string"`                    // Name of the related person
	Type             string   `json:"type" validate:"required,min=1,max=50,safe_string"`                     // Relationship type (e.g., "Child", "Mother")
	Gender           string   `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"` // Gender of the related person
	Birthday         string   `json:"birthday" validate:"omitempty,birthday"`                                // Birthday of the related person
	ContactID        uint     `json:"contact_id" validate:"required"`                                        // Contact this relationship belongs to
	RelatedContactID *uint    `json:"related_contact_id"`                                                    // Optional link to an existing Contact
	RelatedContact   *Contact `gorm:"foreignKey:RelatedContactID" json:"related_contact,omitempty"`          // Linked Contact if exists
}
