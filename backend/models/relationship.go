package models

import (
	"gorm.io/gorm"
)

type Relationship struct {
	gorm.Model
	Name             string   `json:"name"`                                                         // Name of the related person
	Type             string   `json:"type"`                                                         // Relationship type (e.g., "Child", "Mother")
	Gender           string   `json:"gender"`                                                       // Gender of the related person
	Birthday         *Date    `json:"birthday"`                                                     // Birthday of the related person
	ContactID        uint     `json:"contact_id"`                                                   // Contact this relationship belongs to
	RelatedContactID *uint    `json:"related_contact_id"`                                           // Optional link to an existing Contact
	RelatedContact   *Contact `gorm:"foreignKey:RelatedContactID" json:"related_contact,omitempty"` // Linked Contact if exists
}
