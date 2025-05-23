package models

import (
	"gorm.io/gorm"
)

type Contact struct {
	gorm.Model
	Firstname          string         `gorm:"type:text not null COLLATE NOCASE" json:"firstname"`
	Lastname           string         `gorm:"type:text COLLATE NOCASE" json:"lastname"`
	Nickname           string         `gorm:"type:text COLLATE NOCASE" json:"nickname"`
	Gender             string         `json:"gender"`
	Email              string         `gorm:"type:text COLLATE NOCASE" json:"email"`
	Phone              string         `json:"phone"`
	Birthday           *Date          `json:"birthday"`
	Photo              string         `json:"photo"`                                     // Path to the profile photo
	PhotoThumbnail     string         `json:"photo_thumnbnail"`                          // Path to the profile photo thumbnail
	Relationships      []Relationship `gorm:"foreignKey:ContactID" json:"relationships"` // Has many relationships
	Address            string         `json:"address"`                                   // Full address as a string
	HowWeMet           string         `json:"how_we_met"`                                // Text field
	FoodPreference     string         `json:"food_preference"`                           // Text field
	WorkInformation    string         `json:"work_information"`                          // Text field
	ContactInformation string         `json:"contact_information"`                       // Additional contact information
	Circles            []string       `gorm:"type:text;serializer:json" json:"circles"`  // Serialize Circles properly
	Activities         []Activity     `gorm:"many2many:activity_contacts;foreignKey:ID;joinForeignKey:ContactID;References:ID;joinReferences:ActivityID" json:"activities,omitempty"`
	Notes              []Note         `json:"notes,omitempty"`     // One-to-many relationship with notes
	Reminders          []Reminder     `json:"reminders,omitempty"` // One-to-many relationship with reminders
}
