package models

import (
	"gorm.io/gorm"
)

// Activity struct to represent shared activities with one or more contacts
type Activity struct {
	gorm.Model
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Date        Date      `json:"date"`
	Contacts    []Contact `gorm:"many2many:activity_contacts;" json:"contacts,omitempty"` // Define many-to-many relationship
}
