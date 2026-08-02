package models

import (
	"fmt"
	"os"
	"strings"

	"mycorrhizal/contactmodel"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DefaultPhotoDir is the configured profile-photo directory
// (config.Config.ProfilePhotoDir), read directly from the PROFILE_PHOTO_DIR
// environment variable (the same variable config.LoadConfig() reads) rather
// than threaded in from main.go: BeforeSave is a GORM hook with a fixed
// signature (tx *gorm.DB) error — it has no per-call parameter to receive a
// photoDir through, unlike RecordFromContact/ApplyRecordToContact's own
// explicit photoDir parameter (added for WP-73's photo-bridging
// prerequisite, docs/fork-plan/50-integration-and-rebrand.md). A
// package-level var populated at process-init time is the least-invasive way
// to give BeforeSave the same capability without changing its signature or
// reaching into files outside backend/models' WP-73 file scope (this WP does
// not touch main.go). Environment variables are already present in the OS
// process environment before the Go binary starts (this codebase does not
// load a .env file itself — see config/config.go), so reading it here at var-
// init time is equivalent to config.LoadConfig() reading it moments later in
// main(). Empty ("") is a safe default: RecordFromContact's photo bridging
// degrades gracefully to the base64 PhotoThumbnail fallback (or is skipped
// entirely if neither Photo nor PhotoThumbnail is set), never panics.
var DefaultPhotoDir = os.Getenv("PROFILE_PHOTO_DIR")

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
	UserID             uint       `gorm:"not null;index" json:"-"`
	Firstname          string     `gorm:"type:text not null COLLATE NOCASE" json:"firstname" validate:"required,min=1,max=100"`
	Lastname           string     `gorm:"type:text COLLATE NOCASE" json:"lastname" validate:"max=100"`
	Nickname           string     `gorm:"type:text COLLATE NOCASE" json:"nickname" validate:"max=50"`
	Gender             string     `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Email              string     `gorm:"type:text COLLATE NOCASE" json:"email" validate:"omitempty,email"`
	Phone              string     `json:"phone" validate:"omitempty,phone"`
	Birthday           string     `json:"birthday" validate:"omitempty,birthday"`
	Photo              string     `json:"photo"`                                    // Path to the profile photo
	PhotoThumbnail     string     `json:"-"`                                        // Base64 data URL of thumbnail (not exposed in JSON directly)
	Address            string     `json:"address" validate:"max=500"`               // Full address as a string
	HowWeMet           string     `json:"how_we_met" validate:"max=1000"`           // Text field
	WorkInformation    string     `json:"work_information" validate:"max=1000"`     // Text field
	ContactInformation string     `json:"contact_information" validate:"max=1000"`  // Additional contact information
	Circles            []string   `gorm:"type:text;serializer:json" json:"circles"` // Serialize Circles properly
	Activities         []Activity `gorm:"many2many:activity_contacts;foreignKey:ID;joinForeignKey:ContactID;References:ID;joinReferences:ActivityID" json:"activities,omitempty"`
	Notes              []Note     `json:"notes,omitempty"`     // One-to-many relationship with notes
	Reminders          []Reminder `json:"reminders,omitempty"` // One-to-many relationship with reminders

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

	// Neutral RFC 9553/9554/9555 representation (WP-70, P1 — see
	// docs/fork-plan/50-integration-and-rebrand.md). This is a second,
	// parallel representation of the same data already held in the legacy
	// flat/array fields above: purely additive, nothing existing is removed,
	// renamed, or stops being populated. Populated by RecordFromContact (see
	// contact_record.go) via BeforeSave on every save, and by the one-shot
	// cmd/backfill-contact-records tool for rows that predate this WP.
	// Nothing else reads these fields yet (hence json:"-": exposing them on
	// the wire is P2's job, per WP-71's API/DTO rewrite), so adding them
	// carries no compile or behavior risk to any other package.
	Card        contactmodel.Card        `gorm:"column:card;type:text;serializer:json" json:"-"`
	CRM         contactmodel.CRMEnvelope `gorm:"column:crm;type:text;serializer:json" json:"-"`
	Passthrough contactmodel.Passthrough `gorm:"column:passthrough;type:text;serializer:json" json:"-"`

	// Derived projection scalars with no existing legacy analog
	// (contactmodel.Projection.FN / .Org). Populated the same way as
	// Firstname/Lastname/Email/Phone/Birthday below, via DeriveProjection.
	FN  string `gorm:"column:fn" json:"-"`
	Org string `gorm:"column:org" json:"-"`

	// cardSetDirectly is a transient, in-memory-only marker (unexported, so
	// GORM ignores it entirely — no column, nothing to tag) set by
	// ApplyRecordToContact (contact_record_reverse.go, WP-71/P2) to tell
	// BeforeSave below "Card/CRM/Passthrough were just set directly from an
	// authoritative contactmodel.Record — do not re-derive and overwrite them
	// from the flat legacy fields on this save."
	//
	// Without this, BeforeSave's original (WP-70/P1) unconditional
	// `c.Card = RecordFromContact(c, photoDir).Card` would silently discard
	// any Card-only data with no flat-field home (SpeakToAs, PersonalInfo,
	// SocialProfiles, OtherOnlineServices, Keywords, extra name
	// components, additional Organizations/Titles, RelatedTo, Members,
	// Localizations, ...) on every single save of a contact created/updated
	// through the new nested REST API or the VCF/JSContact import path —
	// defeating the entire point of WP-71 accepting/returning the full
	// neutral Record. Flat-field-only writers (CSV import's
	// BuildContactFromRow, MergeImportedContact's merge-by-flat-fields path,
	// and anything else that never calls ApplyRecordToContact) never set
	// this flag, so BeforeSave's original flat->Card derivation keeps running
	// for them exactly as it did in WP-70 — this is what keeps their Card
	// column in sync at all, since they have no other way to populate it.
	cardSetDirectly bool
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
// with the first entry of their respective JSON arrays, and keeps the neutral
// Card/CRM/Passthrough representation (and its own derived projection
// scalars) in sync with the legacy fields on every create/update.
//
// RecordFromContact + contactmodel.DeriveProjection is now the single source
// of truth for Firstname/Lastname/Email/Phone/Birthday/FN/Org: the old
// ad-hoc "first array entry wins" logic for Email/Phone is superseded by
// DeriveProjection's own (equivalent, Pref-aware) primary-value selection,
// so there is one derivation path, not two competing ones (see
// docs/fork-plan/50-integration-and-rebrand.md WP-70). Address has no
// neutral projection field (Address stays a free-text legacy scalar), so its
// ad-hoc sync from the first Addresses[] entry is kept as-is.
//
// Projection values only overwrite their legacy scalar when non-empty, the
// same "only sync when there's something to sync" semantics the original
// Email/Phone logic had (`if len(c.Emails) > 0 { ... }`) — this matters
// because some existing contacts only ever had the scalar Email/Phone set
// directly, without ever populating the Emails/Phones arrays; DeriveProjection
// on those rows also lands back on the same scalar value (RecordFromContact
// falls back to the scalar when its array is empty — see contact_record.go),
// so this is a no-op for them, not a silent blank-out. FN and Org are new
// columns with no prior value and no back-compat concern, so they are always
// assigned directly.
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

	var record *contactmodel.Record
	if c.cardSetDirectly {
		// Card/CRM/Passthrough were just set directly by ApplyRecordToContact
		// from an authoritative Record (new nested REST input, or a VCF/
		// JSContact import) — use that Record's own values for the derived
		// projection below, but leave c.Card/c.CRM/c.Passthrough untouched
		// rather than truncating them back down to what the (necessarily
		// lossy) flat fields alone could reconstruct. See the cardSetDirectly
		// field doc above.
		record = &contactmodel.Record{Card: c.Card, Envelope: c.CRM, Passthrough: c.Passthrough}
		c.cardSetDirectly = false // one-shot: only guards the save that immediately follows ApplyRecordToContact
	} else {
		record = RecordFromContact(c, DefaultPhotoDir)
		c.Card = record.Card
		c.CRM = record.Envelope
		c.Passthrough = record.Passthrough
	}

	proj := contactmodel.DeriveProjection(record)
	if proj.Firstname != "" {
		c.Firstname = proj.Firstname
	}
	if proj.Lastname != "" {
		c.Lastname = proj.Lastname
	}
	if proj.PrimaryEmail != "" {
		c.Email = proj.PrimaryEmail
	}
	if proj.PrimaryPhone != "" {
		c.Phone = proj.PrimaryPhone
	}
	if proj.Birthday != "" {
		c.Birthday = proj.Birthday
	}
	c.FN = proj.FN
	c.Org = proj.Org

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
