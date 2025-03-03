package models

import (
	"time"

	"gorm.io/gorm"
)

// Activity struct to represent shared activities with one or more contacts
type Activity struct {
	gorm.Model
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Date        time.Time `json:"date"`
	Contacts    []Contact `gorm:"many2many:activity_contacts;foreignKey:ID;joinForeignKey:ActivityID;References:ID;joinReferences:ContactID" json:"contacts,omitempty"`
}
