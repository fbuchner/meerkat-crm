package models

import (
	"gorm.io/gorm"
)

type Contact struct {
	gorm.Model
	UserID             uint           `gorm:"not null;index" json:"-"`
	Firstname          string         `gorm:"type:text not null COLLATE NOCASE" json:"firstname" validate:"required,min=1,max=100,safe_string"`
	Lastname           string         `gorm:"type:text COLLATE NOCASE" json:"lastname" validate:"max=100,safe_string"`
	Nickname           string         `gorm:"type:text COLLATE NOCASE" json:"nickname" validate:"max=50,safe_string"`
	Gender             string         `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Email              string         `gorm:"type:text COLLATE NOCASE" json:"email" validate:"omitempty,email"`
	Phone              string         `json:"phone" validate:"omitempty,phone"`
	Birthday           string         `json:"birthday" validate:"omitempty,birthday"`
	Photo              string         `json:"photo"`                                               // Path to the profile photo
	PhotoThumbnail     string         `json:"photo_thumbnail"`                                     // Path to the profile photo thumbnail
	Relationships      []Relationship `gorm:"foreignKey:ContactID" json:"relationships"`           // Has many relationships
	Address            string         `json:"address" validate:"max=500,safe_string"`              // Full address as a string
	HowWeMet           string         `json:"how_we_met" validate:"max=1000,safe_string"`          // Text field
	FoodPreference     string         `json:"food_preference" validate:"max=500,safe_string"`      // Text field
	WorkInformation    string         `json:"work_information" validate:"max=1000,safe_string"`    // Text field
	ContactInformation string         `json:"contact_information" validate:"max=1000,safe_string"` // Additional contact information
	Circles            []string       `gorm:"type:text;serializer:json" json:"circles"`            // Serialize Circles properly
	Activities         []Activity     `gorm:"many2many:activity_contacts;foreignKey:ID;joinForeignKey:ContactID;References:ID;joinReferences:ActivityID" json:"activities,omitempty"`
	Notes              []Note         `json:"notes,omitempty"`     // One-to-many relationship with notes
	Reminders          []Reminder     `json:"reminders,omitempty"` // One-to-many relationship with reminders
}
