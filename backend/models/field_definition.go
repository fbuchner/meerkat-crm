package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Target values stored on FieldDefinition.Target -- which entity kind a
// custom field attaches to. docs/fork-plan/94-custom-fields.md §94.3: "contact
// initially; model allows relationship/household/... later." A closed,
// single-value enum today, widened when a second target actually ships.
const (
	FieldDefinitionTargetContact = "contact"
)

// Type values stored on FieldDefinition.Type -- a closed set (§94.4). Note
// there is no separate "list<T>" token: §94.4 describes list-of-T as "a
// multi-valued variant of any scalar type above ({multi:true})", so
// multi-valuedness is FieldConstraints.Multi on any of these nine, not a
// tenth Type value.
const (
	FieldTypeString   = "string"
	FieldTypeText     = "text"
	FieldTypeNumber   = "number"
	FieldTypeBoolean  = "boolean"
	FieldTypeDate     = "date"
	FieldTypeDateTime = "datetime"
	FieldTypeURI      = "uri"
	FieldTypeEmail    = "email"
	FieldTypePhone    = "phone"
	FieldTypeEnum     = "enum"
)

// FieldConstraints holds the type-dependent validation rules for a
// FieldDefinition (§94.3's "constraints" row: "JSON, type-dependent").
// A concrete typed struct rather than a free-form map -- unlike
// RelationshipEdge.Metadata (a genuine passthrough value bag), every field
// here is actually read and enforced by services.ValidateFieldValue, so a
// real Go type per constraint is more self-documenting and less error-prone
// than interface{} type-assertions at validation time.
type FieldConstraints struct {
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	// Values is the allowed-value list for FieldTypeEnum.
	Values []string `json:"values,omitempty"`
	// Multi marks this field as list-of-<Type> rather than a single scalar
	// (§94.4's list<T>) -- FieldValue.Value is then a JSON array, each
	// element validated against the same scalar rule.
	Multi bool `json:"multi,omitempty"`
}

// FieldDefinition is the schema half of WP-84b's custom-fields system
// (docs/fork-plan/94-custom-fields.md §94.3) -- a user-defined, typed,
// optionally validated and standards-projected property, generalizing the
// existing untyped v1 (User.CustomFieldNames, user.go). Distinct from native
// contactmodel.Card fields: this is the user's escape hatch for concepts the
// standards don't define, not a reimplementation of ones they do (§94.2).
//
// UUID-string-primary-key entity, following Household's exact template
// (household.go): ID generated in BeforeCreate, no soft-delete.
type FieldDefinition struct {
	ID        string    `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID uint   `gorm:"not null;index;uniqueIndex:idx_field_definition_user_key,priority:1" json:"-"`
	Label  string `gorm:"not null" json:"label" validate:"required,max=200"`

	// Key is the stable machine name (the map key / API field name) -- unique
	// per user (see UserID's matching uniqueIndex tag above), since two
	// definitions with the same key would be ambiguous to reference.
	Key string `gorm:"not null;uniqueIndex:idx_field_definition_user_key,priority:2" json:"key" validate:"required,max=100"`

	Target string `gorm:"not null" json:"target" validate:"required,oneof=contact"`
	Type   string `gorm:"not null" json:"type" validate:"required,oneof=string text number boolean date datetime uri email phone enum"`

	Constraints FieldConstraints `gorm:"type:text;serializer:json" json:"constraints,omitempty"`

	// Projection is "internal-only" (default) or "vcard:X-<NAME>" -- see
	// middleware.validateFieldDefinitionProjection and
	// models.projectCustomFields (contact_record.go).
	Projection string `gorm:"not null;default:internal-only" json:"projection" validate:"required,fielddefprojection"`

	// Sensitivity reuses RelationshipEdge's three-value set directly rather
	// than a duplicate local const block: §94.3 explicitly frames this as
	// "the cross-cutting sensitivity rule (91.13)", not an entity-specific
	// classifier like Type/Target above.
	Sensitivity string `gorm:"not null;default:normal;index" json:"sensitivity" validate:"required,oneof=normal private secret"`
}

// BeforeCreate generates a UUID for new FieldDefinitions, mirroring
// Household's own BeforeCreate.
func (fd *FieldDefinition) BeforeCreate(tx *gorm.DB) error {
	if fd.ID == "" {
		fd.ID = uuid.New().String()
	}
	return nil
}

// FieldValue is the data half (§94.3) -- one typed value of a FieldDefinition
// for one entity. Value is stored as json.RawMessage (matching
// RelationshipEdge.Metadata/contactmodel.JCardProp.Value's existing
// precedent for a JSON column holding a variably-typed value) so
// number/bool/array/string values round-trip in their real JSON shape
// rather than being stringified or re-decoded through interface{} (which
// would turn every number into float64).
//
// uint primary key, no soft-delete -- mirrors HouseholdMember/CircleMember/
// ContactTag's precedent for join-shaped rows.
type FieldValue struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	FieldDefinitionID string `gorm:"not null;index;uniqueIndex:idx_field_value_definition_entity,priority:1" json:"field_definition_id"`
	UserID            uint   `gorm:"not null;index" json:"-"`

	// EntityID is a Contact.VCardUID -- the graph invariant every join
	// entity has followed since WP-80 (never a bare Contact.ID).
	EntityID string `gorm:"column:entity_id;not null;index;uniqueIndex:idx_field_value_definition_entity,priority:2" json:"entity_id" validate:"required,uuid4"`

	Value json.RawMessage `gorm:"type:text;serializer:json" json:"value"`
}
