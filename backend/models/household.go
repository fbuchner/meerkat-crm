package models

import (
	"time"

	"mycorrhizal/contactmodel"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Type values stored on Household.Type. Drives which relationship-suggestion
// rules apply — see services.GenerateHouseholdSuggestions.
const (
	HouseholdTypeFamilyUnit = "family_unit"
	HouseholdTypeRoommates  = "roommates"
	HouseholdTypeOther      = "other"
)

// Conventional (not validated — see Role's doc comment) values for
// HouseholdMember.Role.
const (
	HouseholdRoleHead     = "head"
	HouseholdRoleChild    = "child"
	HouseholdRolePet      = "pet"
	HouseholdRoleRoommate = "roommate"
)

// Household is a co-residence grouping of people and pets (docs/fork-plan/
// 91-envelope-data-model.md §91.3) — distinct from a Circle (social grouping,
// how I know them) and from the relationship graph itself (a household is a
// node reached via membership, not a dyadic RelationshipEdge; see §91.2's
// "Household is a node, not a relationship edge" section for why the two
// stay separate — relationships outlive households).
//
// Second UUID-string-primary-key entity in this codebase, after
// RelationshipEdge (relationship_edge.go) — same pattern: ID generated in
// BeforeCreate, no soft-delete (matches RelationshipEdge's own reasoning:
// this is a graph-adjacent entity, not a user-facing CRUD resource with
// existing soft-delete expectations).
type Household struct {
	ID        string    `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID uint   `gorm:"not null;index" json:"-"`
	Name   string `gorm:"not null" json:"name"`

	// Type drives the suggestion engine's rules (§91.4) — a closed switch
	// with specific consequences per value, the same role RelationshipEdge.
	// Status plays, so (unlike Role below) it IS validated.
	Type string `gorm:"not null" json:"type" validate:"required,oneof=family_unit roommates other"`

	// Address is an attribute, not identity — members who move keep the
	// household (§91.3). Reuses the neutral Address shape (contactmodel,
	// already used for Contact.Card) rather than a new type; JSON-serialized
	// like Contact.Card/CRM/Passthrough.
	Address *contactmodel.Address `gorm:"type:text;serializer:json" json:"address,omitempty"`
}

// BeforeCreate generates a UUID for new households, mirroring
// RelationshipEdge's own BeforeCreate.
func (h *Household) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" {
		h.ID = uuid.New().String()
	}
	return nil
}

// HouseholdMember is the membership edge (§91.3's "Membership edge" —
// household_id, entity_id, role, since/until). MemberVCardUID is a
// Contact.VCardUID, not Contact.ID — the same graph invariant
// RelationshipEdge.SourceID/TargetID follow (§90 D3: every relationship-graph
// endpoint is referenced by its stable UUID, never a bare string or a
// database-internal integer).
//
// uint primary key, no soft-delete — mirrors ContactSyncLink/
// CalendarEventLink's precedent for join-shaped rows.
type HouseholdMember struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	HouseholdID string `gorm:"not null;index;uniqueIndex:idx_household_member,priority:1" json:"household_id"`
	UserID      uint   `gorm:"not null;index" json:"-"`

	// Explicit column name: GORM's default namer splits "VCardUID" as
	// v_card_uid (unlike Contact.VCardUID itself, which GORM only ever reads
	// via its own struct tag on that model, not re-derived here) — caught by
	// this field's own test running AutoMigrate and producing
	// member_v_card_uid, which would have silently mismatched
	// migrations/000029's real member_vcard_uid column at runtime.
	MemberVCardUID string `gorm:"column:member_vcard_uid;not null;index;uniqueIndex:idx_household_member,priority:2" json:"member_vcard_uid" validate:"required,uuid4"`

	// Role is conventional, not enforced (HouseholdRole* constants above) —
	// deliberately unvalidated, same reasoning as CRMEnvelope.Kind (WP-82):
	// it's an open, descriptive classifier the suggestion engine must
	// degrade gracefully on (anything other than "child" or a pet-kind
	// contact just counts as an adult), not a closed system state.
	Role string `json:"role,omitempty"`

	// Since/Until are optional partial dates, matching Contact.Birthday's
	// existing string-date convention (contactmodel.PartialDate-compatible)
	// rather than a strict time.Time.
	Since string `json:"since,omitempty"`
	Until string `json:"until,omitempty"`
}
