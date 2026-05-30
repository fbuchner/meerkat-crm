package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContactEmail is a single typed email address (vCard EMAIL).
type ContactEmail struct {
	Type  string `json:"type" validate:"max=30"`
	Value string `json:"value" validate:"required,email"`
}

// ContactPhone is a single typed phone number (vCard TEL).
type ContactPhone struct {
	Type  string `json:"type" validate:"max=30"`
	Value string `json:"value" validate:"required,phone"`
}

// ContactURL is a single typed website URL (vCard URL).
type ContactURL struct {
	Type  string `json:"type" validate:"max=30"`
	Value string `json:"value" validate:"required,max=500,safeurl"`
}

// ContactIMPP is a single instant-messaging / social handle (vCard IMPP).
// Type holds the service (e.g. "telegram", "signal"); Value holds the handle.
type ContactIMPP struct {
	Type  string `json:"type" validate:"max=30"`
	Value string `json:"value" validate:"required,max=200,safeurl"`
}

// ContactAddress is a single structured postal address (vCard ADR).
type ContactAddress struct {
	Type    string `json:"type" validate:"max=30"`
	Street  string `json:"street" validate:"max=500"`
	City    string `json:"city" validate:"max=200"`
	Region  string `json:"region" validate:"max=200"`
	Postal  string `json:"postal" validate:"max=30"`
	Country string `json:"country" validate:"max=100"`
}

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
	Circles            []string       `gorm:"type:text;serializer:json" json:"circles"`  // Serialize Circles properly
	Activities         []Activity     `gorm:"many2many:activity_contacts;foreignKey:ID;joinForeignKey:ContactID;References:ID;joinReferences:ActivityID" json:"activities,omitempty"`
	Notes              []Note         `json:"notes,omitempty"`     // One-to-many relationship with notes
	Reminders          []Reminder     `json:"reminders,omitempty"` // One-to-many relationship with reminders

	// Multi-valued vCard fields (stored as JSON arrays). The legacy Email/Phone/Address
	// scalars above are kept in sync (see BeforeSave) as the denormalized "primary" value
	// used for search and list views.
	Emails    []ContactEmail   `gorm:"column:emails;type:text;serializer:json" json:"emails"`
	Phones    []ContactPhone   `gorm:"column:phones;type:text;serializer:json" json:"phones"`
	Addresses []ContactAddress `gorm:"column:addresses;type:text;serializer:json" json:"addresses"`
	URLs      []ContactURL     `gorm:"column:urls;type:text;serializer:json" json:"urls"`
	IMPPs     []ContactIMPP    `gorm:"column:impps;type:text;serializer:json" json:"impps"`

	// Structured name parts (vCard N)
	Prefix     string `gorm:"type:text" json:"prefix" validate:"max=50"`
	MiddleName string `gorm:"type:text" json:"middle_name" validate:"max=100"`
	Suffix     string `gorm:"type:text" json:"suffix" validate:"max=50"`

	// Organizational fields (vCard ORG / TITLE / ROLE)
	Organization string `gorm:"type:text" json:"organization" validate:"max=200"`
	Department   string `gorm:"type:text" json:"department" validate:"max=200"`
	JobTitle     string `gorm:"type:text" json:"job_title" validate:"max=200"`
	Role         string `gorm:"type:text" json:"role" validate:"max=200"`

	// Anniversary date (vCard X-ANNIVERSARY), same format as Birthday
	Anniversary string `json:"anniversary" validate:"omitempty,birthday"`

	// CardDAV fields
	VCardUID   string `gorm:"column:vcard_uid;index" json:"-"` // Permanent RFC 6350 UID
	VCardExtra string `gorm:"column:vcard_extra" json:"-"`     // JSON for unmapped vCard properties
	ETag       string `gorm:"column:etag" json:"-"`            // Sync conflict detection

	// Custom fields (user-defined string fields)
	CustomFields map[string]string `gorm:"type:text;serializer:json" json:"custom_fields"`

	Archived bool `gorm:"default:false" json:"archived"`
}

// renders a structured address as a single human-readable line, used to keep the legacy Address scalar in sync for search/list views.
func FormatAddress(a ContactAddress) string {
	parts := []string{}
	for _, p := range []string{a.Street, a.City, a.Region, a.Postal, a.Country} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}

// BeforeSave keeps the denormalized primary scalars (Email/Phone/Address) in sync
// with the first entry of their respective JSON arrays. Runs on both create and update
func (c *Contact) BeforeSave(tx *gorm.DB) error {
	if len(c.Emails) > 0 {
		c.Email = c.Emails[0].Value
	}
	if len(c.Phones) > 0 {
		c.Phone = c.Phones[0].Value
	}
	if len(c.Addresses) > 0 {
		c.Address = FormatAddress(c.Addresses[0])
	}
	return nil
}

// generates VCardUID for new contacts
func (c *Contact) BeforeCreate(tx *gorm.DB) error {
	// Generate VCardUID if not set (required for unique constraint)
	if c.VCardUID == "" {
		c.VCardUID = uuid.New().String()
	}
	return nil
}

func (c *Contact) AfterCreate(tx *gorm.DB) error {
	// Now we have the ID, generate proper ETag
	c.ETag = fmt.Sprintf("e-%d-%d", c.ID, c.UpdatedAt.Unix())
	return tx.Model(c).UpdateColumn("etag", c.ETag).Error
}

func (c *Contact) AfterSave(tx *gorm.DB) error {
	newETag := fmt.Sprintf("e-%d-%d", c.ID, c.UpdatedAt.Unix())
	if newETag != c.ETag {
		c.ETag = newETag
		return tx.Model(c).UpdateColumn("etag", c.ETag).Error
	}
	return nil
}
