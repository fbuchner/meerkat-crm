package models

import (
	"time"

	"gorm.io/gorm"
)

// Note struct to represent notes attached to a contact
type Note struct {
	gorm.Model
	Content   string    `json:"content"`
	Date      time.Time `json:"date"`
	ContactID *uint     `json:"contact_id"`
	Contact   Contact   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"contact,omitempty"`
}
