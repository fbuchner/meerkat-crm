package models

import (
	"gorm.io/gorm"
)

// Note struct to represent notes attached to a contact
type Note struct {
	gorm.Model
	Content   string  `json:"content"`
	Date      Date    `json:"date"`
	ContactID uint    `json:"contact_id"` // Foreign key field for Contact
	Contact   Contact `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"contact,omitempty"`
}
