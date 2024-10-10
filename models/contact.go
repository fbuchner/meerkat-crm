package models

import (
	"gorm.io/gorm"
)

// Partner struct to represent the contact's partner
type Partner struct {
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
	Gender   string `json:"gender"`
}

// Relationship struct updated to optionally relate to an existing contact
type Relationship struct {
	Name           string   `json:"name"`                                                  // Name of the related person
	Type           string   `json:"type"`                                                  // Relationship type (e.g., "Child", "Mother")
	Gender         string   `json:"gender"`                                                // Gender of the related person
	Birthday       string   `json:"birthday"`                                              // Birthday of the related person
	ContactID      *uint    `json:"contact_id"`                                            // Optional link to an existing Contact
	RelatedContact *Contact `gorm:"foreignKey:ContactID" json:"related_contact,omitempty"` // Linked Contact if exists
}

// Circle struct to represent a circle of contacts
type Circle struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// Contact struct updated with relationships potentially linking to other contacts
type Contact struct {
	gorm.Model
	Firstname          string         `json:"firstname"`
	Lastname           string         `json:"lastname"`
	Nickname           string         `json:"nickname"`
	Gender             string         `json:"gender"`
	Email              string         `json:"email"`
	Phone              string         `json:"phone"`
	Birthday           Date           `json:"birthday"`
	Photo              string         `json:"photo"`                                                    // Path to the profile photo
	Partner            Partner        `gorm:"embedded" json:"partner"`                                  // Embedded struct for partner info
	Relationships      []Relationship `gorm:"foreignKey:ContactID" json:"relationships"`                // Has many relationships
	Address            string         `json:"address"`                                                  // Full address as a string
	HowWeMet           string         `json:"how_we_met"`                                               // Text field
	FoodPreference     string         `json:"food_preference"`                                          // Text field
	WorkInformation    string         `json:"work_information"`                                         // Text field
	ContactInformation string         `json:"contact_information"`                                      // Additional contact information
	Circles            []Circle       `gorm:"many2many:contact_circles" json:"circles"`                 // Many-to-many relationship with circles
	Activities         []Activity     `gorm:"many2many:activity_contacts;" json:"activities,omitempty"` // Reverse relationship
	Notes              []Note         `json:"notes,omitempty"`                                          // One-to-many relationship with notes
}
