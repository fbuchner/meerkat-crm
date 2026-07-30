package models

import (
	"mycorrhizal/contactmodel"

	"gorm.io/gorm"
)

// ContactSummary is the slim per-item shape for GET /api/v1/contacts (list).
// Per docs/fork-plan/50-integration-and-rebrand.md WP-71 ("Mobile-CRUD-real"
// section), it wraps contactmodel.Projection's own fields (Firstname,
// Lastname, FN, PrimaryEmail, PrimaryPhone, Birthday, Org) plus the record's
// identity (UID) and the existing Photo/PhotoThumbnail — deliberately a new
// controller-layer DTO built FROM a Contact, not contactmodel.Projection
// reused verbatim as a wire type (Projection stays an internal
// persistence-projection helper). This is what a mobile contact-list screen
// calls; it must not require fetching every contact's full nested Card.
//
// ID and Archived are practical additions beyond the doc's literal field
// list: ID is required for any client to address a specific contact (link
// to the detail view, issue further requests), and Archived lets a caller
// that requested include_archived=true (mixed archived/active results)
// distinguish which is which without a second round trip. Both are cheap,
// already-loaded scalar columns — no extra query cost.
//
// Nickname and Circles were added after the fact (frontend migration
// pre-work): ContactsPage's list view renders both per-row, and the old
// fields=-based flat API used to let it request them directly. Both are
// cheap, already-loaded columns like ID/Archived above.
type ContactSummary struct {
	ID             uint     `json:"id"`
	UID            string   `json:"uid"`
	Firstname      string   `json:"firstname"`
	Lastname       string   `json:"lastname"`
	Nickname       string   `json:"nickname"`
	FN             string   `json:"fn"`
	PrimaryEmail   string   `json:"primary_email"`
	PrimaryPhone   string   `json:"primary_phone"`
	Birthday       string   `json:"birthday"`
	Org            string   `json:"org"`
	Photo          string   `json:"photo"`
	PhotoThumbnail string   `json:"photo_thumbnail"`
	Circles        []string `json:"circles"`
	Archived       bool     `json:"archived"`
}

// NewContactSummary builds a ContactSummary directly from a Contact's own
// already-denormalized columns (Firstname/Lastname/FN/Email/Phone/Birthday/
// Org), rather than re-deriving them a third time via
// RecordFromContact+contactmodel.DeriveProjection: WP-70's BeforeSave
// already keeps those columns in sync with the neutral Card on every
// create/update, so reading them directly here is the single derivation
// path, just read from its cached/persisted destination instead of
// recomputed on every list-row render.
func NewContactSummary(c *Contact) ContactSummary {
	return ContactSummary{
		ID:             c.ID,
		UID:            c.VCardUID,
		Firstname:      c.Firstname,
		Lastname:       c.Lastname,
		Nickname:       c.Nickname,
		FN:             c.FN,
		PrimaryEmail:   c.Email,
		PrimaryPhone:   c.Phone,
		Birthday:       c.Birthday,
		Org:            c.Org,
		Photo:          c.Photo,
		PhotoThumbnail: c.PhotoThumbnail,
		Circles:        c.Circles,
		Archived:       c.Archived,
	}
}

// ContactSummaryWithRelations extends ContactSummary with the sub-resource
// arrays requested via includes= (notes/activities/reminders). Per Gap 2's
// binding resolution, this is used only when includes= is non-empty; plain
// ContactSummary is used otherwise. It is intentionally NOT a full Card:
// includes= augments the slim list shape, it does not upgrade it to the
// detail shape.
//
// Relationships was removed from this shape in §3d WP4 alongside the
// includes=relationships removal in contact_controller.go's GetContacts.
type ContactSummaryWithRelations struct {
	ContactSummary
	Notes      []Note     `json:"notes,omitempty"`
	Activities []Activity `json:"activities,omitempty"`
	Reminders  []Reminder `json:"reminders,omitempty"`
}

// NewContactSummaryWithRelations builds the extended list-item shape from a
// Contact whose requested associations have already been Preload()ed by the
// caller (GetContacts in contact_controller.go).
func NewContactSummaryWithRelations(c *Contact) ContactSummaryWithRelations {
	return ContactSummaryWithRelations{
		ContactSummary: NewContactSummary(c),
		Notes:          c.Notes,
		Activities:     c.Activities,
		Reminders:      c.Reminders,
	}
}

// ContactRecordInput is the request body for POST /api/v1/contacts and
// PUT /api/v1/contacts/{id} (WP-71 item 3): the full neutral Card/CRM shape,
// nested — not the old flat ContactInput. Gender rides alongside Card/CRM
// rather than inside either: per RecordFromContact/ApplyRecordToContact's
// own extensive documentation, Gender is a legacy CRM concept with
// deliberately no neutral-model home (it is not vCard GENDER or JSContact
// speakToAs), so it isn't part of contactmodel.Card or contactmodel.
// CRMEnvelope at all — it has to be carried as a sibling field on the
// wire DTO instead.
//
// Validation is deliberately light on Card/CRM/Passthrough: per this WP's
// item 5 (graceful, non-strict validation sourced from the neutral model's
// own degradation policy, docs/fork-plan/00-overview.md §0.5), nothing here
// hard-fails on an unrecognized enum value (e.g. an unexpected
// NameComponent.Kind, Anniversary.Kind, or PersonalInfo.Kind) — contactmodel
// itself (a P0-locked package, out of this WP's scope to modify) carries no
// `validate` struct tags on any of these nested types, so there is nothing
// to loosen here; the previous flat ContactInput's handful of
// `validate:"omitempty,oneof=..."` tags simply have no equivalent
// re-introduced on the new nested shape, by design. Firstname/name
// presence is checked post-conversion in the controller (see CreateContact/
// UpdateContact) rather than via a struct tag, since "at least one name
// component present" is not expressible as a simple field-level validator on
// a nested slice.
type ContactRecordInput struct {
	Gender      string                   `json:"gender" validate:"omitempty,oneof=male female other prefer_not_to_say"`
	Card        contactmodel.Card        `json:"card"`
	CRM         contactmodel.CRMEnvelope `json:"crm"`
	Passthrough contactmodel.Passthrough `json:"passthrough"`
}

// ToRecord builds a contactmodel.Record from the input's Card/CRM/
// Passthrough, ready to hand to ApplyRecordToContact.
func (in *ContactRecordInput) ToRecord() *contactmodel.Record {
	return &contactmodel.Record{Card: in.Card, Envelope: in.CRM, Passthrough: in.Passthrough}
}

// ContactRecordResponse is the full neutral detail-view response shape for
// GET /api/v1/contacts/{id}, and the response returned by POST/PUT
// /api/v1/contacts (a symmetric read/write contract, per WP-71's
// "Mobile-CRUD-real" section). Card/CRM/Passthrough are exposed under their
// own top-level names here rather than by giving models.Contact's existing
// (WP-70) `json:"-"` Card/CRM/Passthrough fields real JSON tags — this keeps
// Contact's own default JSON shape (still relied on by GetContactsRandom,
// Archive/UnarchiveContact, etc., all out of this WP's Gap list) completely
// unchanged, and gives this response a clean place to add
// identity/relationship fields the bare Contact struct doesn't carry the
// same way (UID/ETag surfaced as their own fields rather than the
// Card.UID/etag columns).
type ContactRecordResponse struct {
	ID             uint                     `json:"id"`
	UID            string                   `json:"uid"`
	ETag           string                   `json:"etag"`
	Gender         string                   `json:"gender"`
	Card           contactmodel.Card        `json:"card"`
	CRM            contactmodel.CRMEnvelope `json:"crm"`
	Passthrough    contactmodel.Passthrough `json:"passthrough,omitempty"`
	Photo          string                   `json:"photo"`
	PhotoThumbnail string                   `json:"photo_thumbnail"`
	Archived       bool                     `json:"archived"`

	// Preserved from the existing GetContact/GetContacts preload-all
	// behavior (Gap: "keep the existing preload behavior for backward
	// compat" — see GetContact's doc comment in contact_controller.go for
	// the full reasoning on why these are still included here even though
	// dedicated /contacts/:id/notes-style endpoints also exist).
	//
	// Relationships was removed in §3d WP4 — see GetContact's doc comment.
	Notes      []Note     `json:"notes,omitempty"`
	Activities []Activity `json:"activities,omitempty"`
	Reminders  []Reminder `json:"reminders,omitempty"`
}

// NewContactRecordResponse builds the full detail-view response from a
// Contact, deriving Card/CRM/Passthrough/UID via RecordForContact (not
// RecordFromContact directly) so the response reflects what's actually
// persisted, including data with no flat-field home (SpeakToAs,
// PersonalInfo, ...) — calling RecordFromContact fresh here was a real,
// live bug (found while auditing WP-73's work): it silently dropped exactly
// that data from GET/POST/PUT responses. See RecordForContact's doc
// comment. photoDir (config.Config.ProfilePhotoDir) is forwarded through it
// so the response's Card.Media carries the contact's photo (WP-73's
// photo-bridging prerequisite) in addition to the existing top-level
// Photo/PhotoThumbnail fields below. db is forwarded to RecordForContact for
// relationship-graph projection (WP-80); pass nil to skip it.
func NewContactRecordResponse(c *Contact, photoDir string, db *gorm.DB) ContactRecordResponse {
	record := RecordForContact(c, photoDir, db)
	return ContactRecordResponse{
		ID:             c.ID,
		UID:            record.UID,
		ETag:           record.ETag,
		Gender:         c.Gender,
		Card:           record.Card,
		CRM:            record.Envelope,
		Passthrough:    record.Passthrough,
		Photo:          c.Photo,
		PhotoThumbnail: c.PhotoThumbnail,
		Archived:       c.Archived,
		Notes:          c.Notes,
		Activities:     c.Activities,
		Reminders:      c.Reminders,
	}
}
