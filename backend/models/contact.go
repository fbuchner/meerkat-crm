package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Contact struct {
	gorm.Model
	UserID             uint           `gorm:"not null;index" json:"-"`
	Firstname          string         `gorm:"type:text not null COLLATE NOCASE" json:"firstname" validate:"required,min=1,max=100"`
	Lastname           string         `gorm:"type:text COLLATE NOCASE" json:"lastname" validate:"max=100"`
	Nickname           string         `gorm:"type:text COLLATE NOCASE" json:"nickname" validate:"max=50"`
	Gender             string         `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Email              string         `gorm:"type:text COLLATE NOCASE" json:"email" validate:"omitempty,email"`
	Phone              string         `json:"phone" validate:"omitempty,phone"`
	Birthday           string         `json:"birthday" validate:"omitempty,birthday"`
	Photo              string         `json:"photo"`                                     // Path to the profile photo
	PhotoThumbnail     string         `json:"-"`                                         // Base64 data URL of thumbnail (not exposed in JSON directly)
	Relationships      []Relationship `gorm:"foreignKey:ContactID" json:"relationships"` // Has many relationships
	Address            string         `json:"address" validate:"max=500"`                // Full address as a string
	HowWeMet           string         `json:"how_we_met" validate:"max=1000"`            // Text field
	FoodPreference     string         `json:"food_preference" validate:"max=500"`        // Text field
	WorkInformation    string         `json:"work_information" validate:"max=1000"`      // Text field
	ContactInformation string         `json:"contact_information" validate:"max=1000"`   // Additional contact information
	Circles            []string       `gorm:"type:text;serializer:json" json:"circles"`            // Serialize Circles properly
	Activities         []Activity     `gorm:"many2many:activity_contacts;foreignKey:ID;joinForeignKey:ContactID;References:ID;joinReferences:ActivityID" json:"activities,omitempty"`
	Notes              []Note         `json:"notes,omitempty"`     // One-to-many relationship with notes
	Reminders          []Reminder     `json:"reminders,omitempty"` // One-to-many relationship with reminders

	// CardDAV fields
	VCardUID   string `gorm:"column:vcard_uid;index" json:"-"` // Permanent RFC 6350 UID
	VCardExtra string `gorm:"column:vcard_extra" json:"-"` // JSON for unmapped vCard properties
	ETag       string `gorm:"column:etag" json:"-"`        // Sync conflict detection

	// Custom fields (user-defined string fields)
	CustomFields map[string]string `gorm:"type:text;serializer:json" json:"custom_fields"`

	Archived bool `gorm:"default:false" json:"archived"`
}

// BeforeCreate generates VCardUID for new contacts
func (c *Contact) BeforeCreate(tx *gorm.DB) error {
	// Generate VCardUID if not set (required for unique constraint)
	if c.VCardUID == "" {
		c.VCardUID = uuid.New().String()
	}
	return nil
}

// BeforeSave generates ETag before saving contact
func (c *Contact) BeforeSave(tx *gorm.DB) error {
	// ETag format: e-{id}-{updated_at_unix}
	// For new records, ID is 0 and will be set after create
	// We regenerate ETag in AfterSave for new records
	if c.ID != 0 {
		c.ETag = fmt.Sprintf("e-%d-%d", c.ID, c.UpdatedAt.Unix())
	}
	return nil
}

// AfterCreate sets ETag after creating a new contact
func (c *Contact) AfterCreate(tx *gorm.DB) error {
	// Now we have the ID, generate proper ETag
	c.ETag = fmt.Sprintf("e-%d-%d", c.ID, c.UpdatedAt.Unix())
	return tx.Model(c).UpdateColumn("etag", c.ETag).Error
}
