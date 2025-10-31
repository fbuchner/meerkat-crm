package models

import (
	"time"

	"gorm.io/gorm"
)

// Activity struct to represent shared activities with one or more contacts
type Activity struct {
	gorm.Model
	Title       string    `json:"title" validate:"required,min=1,max=200,safe_string"`
	Description string    `json:"description" validate:"max=2000,safe_string"`
	Location    string    `json:"location" validate:"max=300,safe_string"`
	Date        time.Time `json:"date" validate:"required"`
	Contacts    []Contact `gorm:"many2many:activity_contacts;foreignKey:ID;joinForeignKey:ActivityID;References:ID;joinReferences:ContactID" json:"contacts,omitempty"`
}
