// Package jscontact implements the RFC 9553 JSContact wire format: Go structs
// mirroring the JSContact registry objects, plus a JSON codec (see codec.go)
// that converts between the wire JSON shape (Id-keyed maps, boolean-set maps)
// and ergonomic Go slices. This package intentionally defines its OWN type
// set — separate from backend/contactmodel's neutral model — except for
// contactmodel.JCardProp, which is reused directly for the vCardProps
// passthrough field, per docs/fork-plan/30-adapters.md §30.A.
//
// This file (WP-30a) contains types only. Neutral-model conversion
// (ToNeutral/FromNeutral) is WP-30b's adapter.go, not this file.
package jscontact

import (
	"encoding/json"

	"mycorrhizal/contactmodel"
)

// extra (added to most wire object types below) captures JSON object keys
// nested inside that object which this package's Go struct has no field for
// (RFC 9553 permits vendor/future properties on essentially any object, not
// just at the Card's top level). It is unexported and json:"-"-equivalent by
// construction (encoding/json never touches unexported fields either way):
// codec.go's UnmarshalJSON for each such type populates it via captureExtra,
// and its MarshalJSON splices it back in via mergeExtraJSON, so an
// unrecognized nested key round-trips instead of being silently discarded by
// Go's default json.Unmarshal behavior (the defect this fix addresses; see
// codec.go's "nested-unknown-property capture" section and
// docs/fork-plan/30-adapters.md §30.A rule 5). adapter.go reads these
// directly (same package) to populate Record.Passthrough.JSContact at the
// correct nested JSON pointer.
//
// Deliberately NOT added to: Card (its top-level unknowns are already
// handled by adapter.go's importUnknownTopLevel/spliceJSContactPassthrough,
// predating this fix), and PartialDate/Timestamp (RFC 9553 §2.8.1 leaf date
// value types with a tiny, stable field set reached only through
// Anniversary.date's polymorphic union — see codec.go for the full scope
// rationale).

// Card is the root JSContact object (RFC 9553 §2).
type Card struct {
	Type     string     `json:"@type"`
	Version  string     `json:"version"`
	UID      string     `json:"uid,omitempty"`
	Kind     string     `json:"kind,omitempty"`
	Language string     `json:"language,omitempty"`
	ProdID   string     `json:"prodId,omitempty"`
	Created  *Timestamp `json:"created,omitempty"`
	Updated  *Timestamp `json:"updated,omitempty"`

	Name *Name `json:"name,omitempty"`

	Nicknames      []Nickname      `json:"nicknames,omitempty"`      // Id-map on the wire
	Organizations  []Organization  `json:"organizations,omitempty"`  // Id-map on the wire
	Titles         []Title         `json:"titles,omitempty"`         // Id-map on the wire
	Emails         []EmailAddress  `json:"emails,omitempty"`         // Id-map on the wire
	Phones         []Phone         `json:"phones,omitempty"`         // Id-map on the wire
	OnlineServices []OnlineService `json:"onlineServices,omitempty"` // Id-map on the wire
	Addresses      []Address       `json:"addresses,omitempty"`      // Id-map on the wire
	Anniversaries  []Anniversary   `json:"anniversaries,omitempty"`  // Id-map on the wire
	SpeakToAs      *SpeakToAs      `json:"speakToAs,omitempty"`
	PersonalInfo   []PersonalInfo  `json:"personalInfo,omitempty"` // Id-map on the wire
	Notes          []Note          `json:"notes,omitempty"`        // Id-map on the wire
	Keywords       []string        `json:"keywords,omitempty"`     // boolean-set on the wire

	Media               []Media             `json:"media,omitempty"`               // Id-map on the wire
	Calendars           []Calendar          `json:"calendars,omitempty"`           // Id-map on the wire
	SchedulingAddresses []SchedulingAddress `json:"schedulingAddresses,omitempty"` // Id-map on the wire
	CryptoKeys          []CryptoKey         `json:"cryptoKeys,omitempty"`          // Id-map on the wire
	Directories         []Directory         `json:"directories,omitempty"`         // Id-map on the wire
	Links               []Link              `json:"links,omitempty"`               // Id-map on the wire

	PreferredLanguages []LanguagePref `json:"preferredLanguages,omitempty"` // Id-map on the wire
	RelatedTo          []Relation     `json:"relatedTo,omitempty"`          // keyed-by-value map on the wire (key = Relation.Target)
	Members            []string       `json:"members,omitempty"`            // boolean-set on the wire

	// VCardProps reuses the neutral passthrough type directly (one type, one owner).
	VCardProps []contactmodel.JCardProp `json:"vCardProps,omitempty"`
}

// Name (RFC 9553 §2.2.1.1).
type Name struct {
	Type             string            `json:"@type"`
	Components       []NameComponent   `json:"components,omitempty"`
	IsOrdered        *bool             `json:"isOrdered,omitempty"`
	DefaultSeparator string            `json:"defaultSeparator,omitempty"`
	Full             string            `json:"full,omitempty"`
	SortAs           map[string]string `json:"sortAs,omitempty"`
	PhoneticScript   string            `json:"phoneticScript,omitempty"`
	PhoneticSystem   string            `json:"phoneticSystem,omitempty"`
	extra            map[string]json.RawMessage
}

// NameComponent (RFC 9553 §2.2.1.2). Kind ∈ title,given,given2,surname,surname2,
// credential,generation,separator.
type NameComponent struct {
	Type     string `json:"@type"`
	Value    string `json:"value"`
	Kind     string `json:"kind"`
	Phonetic string `json:"phonetic,omitempty"`
	extra    map[string]json.RawMessage
}

// Address (RFC 9553 §2.5.1.1). ID is the wire Id-map key; excluded from the
// object's own JSON (json:"-") and handled by codec.go's map<->slice conversion.
type Address struct {
	Type             string             `json:"@type"`
	ID               string             `json:"-"`
	Components       []AddressComponent `json:"components,omitempty"`
	IsOrdered        *bool              `json:"isOrdered,omitempty"`
	CountryCode      string             `json:"countryCode,omitempty"`
	Coordinates      string             `json:"coordinates,omitempty"`
	TimeZone         string             `json:"timeZone,omitempty"`
	Contexts         []string           `json:"contexts,omitempty"` // boolean-set
	Full             string             `json:"full,omitempty"`
	DefaultSeparator string             `json:"defaultSeparator,omitempty"`
	PhoneticScript   string             `json:"phoneticScript,omitempty"`
	PhoneticSystem   string             `json:"phoneticSystem,omitempty"`
	Pref             *int               `json:"pref,omitempty"`
	extra            map[string]json.RawMessage
}

// AddressComponent (RFC 9553 §2.5.1.2). Kind ∈ room,apartment,floor,building,
// number,name,block,subdistrict,district,locality,region,postcode,country,
// direction,landmark,postOfficeBox,separator.
type AddressComponent struct {
	Type     string `json:"@type"`
	Value    string `json:"value"`
	Kind     string `json:"kind"`
	Phonetic string `json:"phonetic,omitempty"`
	extra    map[string]json.RawMessage
}

// Organization's field shape follows backend/contactmodel.Organization (RFC
// 9553's Organization/OrgUnit objects are not enumerated in
// docs/specs/rfc9553-model.md §1; see this package's codec_test.go / the WP-30a
// report for the flagged gap).
type Organization struct {
	Type   string    `json:"@type"`
	ID     string    `json:"-"`
	Name   string    `json:"name,omitempty"`
	Units  []OrgUnit `json:"units,omitempty"`
	SortAs string    `json:"sortAs,omitempty"`
	extra  map[string]json.RawMessage
}

// OrgUnit is a plain (non-Id-map) array element of Organization.Units.
type OrgUnit struct {
	Type   string `json:"@type"`
	Name   string `json:"name"`
	SortAs string `json:"sortAs,omitempty"`
	extra  map[string]json.RawMessage
}

// Title's Kind ∈ title|role. Field shape follows backend/contactmodel.Title;
// confirmed against the title-role golden fixture (docs/fork-plan/golden-fixtures/title-role.jscontact.json).
type Title struct {
	Type           string `json:"@type"`
	ID             string `json:"-"`
	Name           string `json:"name"`
	Kind           string `json:"kind,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
	extra          map[string]json.RawMessage
}

// Phone (RFC 9553 §2.3.3).
type Phone struct {
	Type     string   `json:"@type"`
	ID       string   `json:"-"`
	Number   string   `json:"number"`
	Features []string `json:"features,omitempty"` // boolean-set
	Contexts []string `json:"contexts,omitempty"` // boolean-set
	Pref     *int     `json:"pref,omitempty"`
	Label    string   `json:"label,omitempty"`
	extra    map[string]json.RawMessage
}

// EmailAddress (RFC 9553 §2.3.1). Object type name is "EmailAddress"; the
// Card property is "emails".
type EmailAddress struct {
	Type     string   `json:"@type"`
	ID       string   `json:"-"`
	Address  string   `json:"address"`
	Contexts []string `json:"contexts,omitempty"` // boolean-set
	Pref     *int     `json:"pref,omitempty"`
	Label    string   `json:"label,omitempty"`
	extra    map[string]json.RawMessage
}

// OnlineService (RFC 9553 §2.3.2).
type OnlineService struct {
	Type     string   `json:"@type"`
	ID       string   `json:"-"`
	Service  string   `json:"service,omitempty"`
	URI      string   `json:"uri,omitempty"`
	User     string   `json:"user,omitempty"`
	Contexts []string `json:"contexts,omitempty"` // boolean-set
	Pref     *int     `json:"pref,omitempty"`
	Label    string   `json:"label,omitempty"`
	// VCardName (RFC 9555 §2.15.3) preserves which vCard property name
	// (IMPP or SOCIALPROFILE) produced this object, since both convert to
	// the same JSContact OnlineService type. The RFC's own worked examples
	// (§2.7.2 Figure 17, §2.15.3 Figure 47) show the value written in
	// lowercase ("impp"); §2.15.3 also states the value's comparison is
	// case-insensitive ("The case-insensitive value MUST be valid according
	// to the 'name' ABNF ... [RFC6350]"). This package emits the RFC's own
	// lowercase spelling on export and compares case-insensitively on import
	// (see adapter.go's importEmailsPhonesOnline).
	VCardName string `json:"vCardName,omitempty"`
	extra     map[string]json.RawMessage
}

// Calendar's Kind ∈ calendar|freeBusy. Resource-shaped (see backend/contactmodel.Resource).
type Calendar struct {
	Type      string   `json:"@type"`
	ID        string   `json:"-"`
	Kind      string   `json:"kind,omitempty"`
	URI       string   `json:"uri"`
	MediaType string   `json:"mediaType,omitempty"`
	Label     string   `json:"label,omitempty"`
	Contexts  []string `json:"contexts,omitempty"` // boolean-set
	Pref      *int     `json:"pref,omitempty"`
	extra     map[string]json.RawMessage
}

// SchedulingAddress is Resource-shaped, with no Kind enum (vCard CALADRURI).
type SchedulingAddress struct {
	Type     string   `json:"@type"`
	ID       string   `json:"-"`
	URI      string   `json:"uri"`
	Label    string   `json:"label,omitempty"`
	Contexts []string `json:"contexts,omitempty"` // boolean-set
	Pref     *int     `json:"pref,omitempty"`
	extra    map[string]json.RawMessage
}

// CryptoKey is Resource-shaped, with no Kind enum (vCard KEY).
type CryptoKey struct {
	Type      string   `json:"@type"`
	ID        string   `json:"-"`
	URI       string   `json:"uri"`
	MediaType string   `json:"mediaType,omitempty"`
	Label     string   `json:"label,omitempty"`
	Contexts  []string `json:"contexts,omitempty"` // boolean-set
	Pref      *int     `json:"pref,omitempty"`
	extra     map[string]json.RawMessage
}

// Directory's Kind ∈ directory|entry; ListAs only meaningful for kind=entry.
type Directory struct {
	Type      string   `json:"@type"`
	ID        string   `json:"-"`
	Kind      string   `json:"kind,omitempty"`
	URI       string   `json:"uri"`
	MediaType string   `json:"mediaType,omitempty"`
	Label     string   `json:"label,omitempty"`
	Contexts  []string `json:"contexts,omitempty"` // boolean-set
	Pref      *int     `json:"pref,omitempty"`
	ListAs    *int     `json:"listAs,omitempty"`
	extra     map[string]json.RawMessage
}

// Link's Kind ∈ contact (optional).
type Link struct {
	Type      string   `json:"@type"`
	ID        string   `json:"-"`
	Kind      string   `json:"kind,omitempty"`
	URI       string   `json:"uri"`
	MediaType string   `json:"mediaType,omitempty"`
	Label     string   `json:"label,omitempty"`
	Contexts  []string `json:"contexts,omitempty"` // boolean-set
	Pref      *int     `json:"pref,omitempty"`
	extra     map[string]json.RawMessage
}

// Media's Kind ∈ photo|logo|sound.
type Media struct {
	Type      string   `json:"@type"`
	ID        string   `json:"-"`
	Kind      string   `json:"kind,omitempty"`
	URI       string   `json:"uri"`
	MediaType string   `json:"mediaType,omitempty"`
	Label     string   `json:"label,omitempty"`
	Contexts  []string `json:"contexts,omitempty"` // boolean-set
	Pref      *int     `json:"pref,omitempty"`
	extra     map[string]json.RawMessage
}

// Nickname (RFC 9553 §2.2.2).
type Nickname struct {
	Type     string   `json:"@type"`
	ID       string   `json:"-"`
	Name     string   `json:"name"`
	Contexts []string `json:"contexts,omitempty"` // boolean-set
	Pref     *int     `json:"pref,omitempty"`
	extra    map[string]json.RawMessage
}

// LanguagePref (RFC 9553 §2.3.4).
type LanguagePref struct {
	Type     string   `json:"@type"`
	ID       string   `json:"-"`
	Language string   `json:"language"`
	Contexts []string `json:"contexts,omitempty"` // boolean-set
	Pref     *int     `json:"pref,omitempty"`
	extra    map[string]json.RawMessage
}

// Pronouns (RFC 9553 §2.2.4). Nested Id-map under SpeakToAs.Pronouns.
type Pronouns struct {
	Type     string   `json:"@type"`
	ID       string   `json:"-"`
	Pronouns string   `json:"pronouns"`
	Contexts []string `json:"contexts,omitempty"` // boolean-set
	Pref     *int     `json:"pref,omitempty"`
	extra    map[string]json.RawMessage
}

// SpeakToAs (RFC 9553 §2.2.4).
type SpeakToAs struct {
	Type              string     `json:"@type"`
	GrammaticalGender string     `json:"grammaticalGender,omitempty"`
	Pronouns          []Pronouns `json:"pronouns,omitempty"` // Id-map on the wire
	extra             map[string]json.RawMessage
}

// Anniversary (RFC 9553 §2.8.1).
type Anniversary struct {
	Type  string          `json:"@type"`
	ID    string          `json:"-"`
	Kind  string          `json:"kind"`
	Date  AnniversaryDate `json:"date"`
	Place *Address        `json:"place,omitempty"`
	extra map[string]json.RawMessage
}

// AnniversaryDate is the polymorphic Anniversary.date value: either a
// PartialDate (the default type when @type is absent) or a Timestamp.
// Exactly one of PartialDate/Timestamp should be set.
type AnniversaryDate struct {
	PartialDate *PartialDate
	Timestamp   *Timestamp
}

// PartialDate (RFC 9553 §2.8.1).
type PartialDate struct {
	Type          string `json:"@type"`
	Year          *int   `json:"year,omitempty"`
	Month         *int   `json:"month,omitempty"`
	Day           *int   `json:"day,omitempty"`
	CalendarScale string `json:"calendarScale,omitempty"`
}

// Timestamp (RFC 9553 §2.8.1).
type Timestamp struct {
	Type string `json:"@type"`
	UTC  string `json:"utc"`
}

// PersonalInfo (RFC 9553 §2.8.4). Kind ∈ expertise|hobby|interest; Level ∈ high|medium|low.
type PersonalInfo struct {
	Type   string `json:"@type"`
	ID     string `json:"-"`
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Level  string `json:"level,omitempty"`
	ListAs *int   `json:"listAs,omitempty"`
	Label  string `json:"label,omitempty"`
	extra  map[string]json.RawMessage
}

// Note's field shape follows backend/contactmodel.Note (RFC 9554 AUTHOR/AUTHOR-NAME/CREATED params).
type Note struct {
	Type    string     `json:"@type"`
	ID      string     `json:"-"`
	Note    string     `json:"note"`
	Author  *Author    `json:"author,omitempty"`
	Created *Timestamp `json:"created,omitempty"`
	extra   map[string]json.RawMessage
}

// Author is Note's author sub-object (name and/or URI of the note's author).
type Author struct {
	Type  string `json:"@type"`
	Name  string `json:"name,omitempty"`
	URI   string `json:"uri,omitempty"`
	extra map[string]json.RawMessage
}

// Relation (RFC 9553 §2.1.8). Card.relatedTo is a keyed-by-value map
// (String[Relation]); the map key is the related entity's URI/text, held here
// as Target (excluded from the object's own JSON — it is the map key, not an
// object property). Relation is the boolean-set of relation-type strings.
type Relation struct {
	Type     string   `json:"@type"`
	Target   string   `json:"-"`
	Relation []string `json:"relation,omitempty"` // boolean-set
	extra    map[string]json.RawMessage
}
