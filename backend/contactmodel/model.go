// Package contactmodel defines the neutral internal contact model: a superset
// type set shaped closely after RFC 9553 (JSContact) that also covers the full
// IANA vCard-elements registry. It contains pure data types only — no parsing,
// no mapping, no persistence. Format adapters (jscontact, vcard4, vcard3)
// populate and read these types; this package must not import any of them.
package contactmodel

import "encoding/json"

// Card is the neutral superset of a single contact's standardized data
// (union of RFC 9553 JSContact + the full IANA vCard-elements registry).
type Card struct {
	UID      string     `json:"uid,omitempty"`
	Kind     string     `json:"kind,omitempty"`     // individual|group|org|location|application|device
	Language string     `json:"language,omitempty"` // default language tag for the card
	ProdID   string     `json:"prodId,omitempty"`
	Created  *Timestamp `json:"created,omitempty"` // RFC 9554 CREATED
	Updated  *Timestamp `json:"updated,omitempty"` // vCard REV

	Name                *Name           `json:"name,omitempty"`
	Nicknames           []Nickname      `json:"nicknames,omitempty"`
	Organizations       []Organization  `json:"organizations,omitempty"`
	Titles              []Title         `json:"titles,omitempty"`
	Emails              []Email         `json:"emails,omitempty"`
	Phones              []Phone         `json:"phones,omitempty"`
	ImppAddresses       []OnlineService `json:"imppAddresses,omitempty"`       // vCard IMPP only
	SocialProfiles      []OnlineService `json:"socialProfiles,omitempty"`      // vCard SOCIALPROFILE only (9554)
	OtherOnlineServices []OnlineService `json:"otherOnlineServices,omitempty"` // unclassified: GUI-added with no vCard-property preference, or JSContact-imported with no vCardName hint (RFC 9555 §2.15.3). NOT exported to vCard (no safe default property to guess) — see 20-correspondence.md §20.7
	Addresses           []Address       `json:"addresses,omitempty"`
	Anniversaries       []Anniversary   `json:"anniversaries,omitempty"`
	SpeakToAs           *SpeakToAs      `json:"speakToAs,omitempty"`
	PersonalInfo        []PersonalInfo  `json:"personalInfo,omitempty"`
	Notes               []Note          `json:"notes,omitempty"`
	Keywords            []string        `json:"keywords,omitempty"` // vCard CATEGORIES

	// Resource-shaped collections (all reuse Resource). Each field below is now
	// SINGLE-PURPOSE — one vCard property per field, never Kind-shared — so that
	// import always lands in an unambiguous, discretely-typed home and never has
	// to guess a value's origin on export. This replaced an earlier
	// Kind-discriminated design that silently conflated IMPP/SOCIALPROFILE (see
	// ImppAddresses/SocialProfiles above), URL/CONTACT-URI (Links/ContactURIs
	// below), and CALURI/FBURL (Calendars/FreeBusyURLs below): a bare-URI value
	// in either half of each pair was structurally identical on import, so which
	// vCard property produced it was permanently lost. `Resource.Kind` is still
	// used by `Media` (photo|logo|sound) and `Directories` (directory|entry),
	// which remain genuinely Kind-discriminated per JSContact's own object
	// model — those are legitimately one concept with sub-kinds, not two
	// distinct IANA properties sharing a field.
	Media               []Resource `json:"media,omitempty"`               // kind: photo|logo|sound
	Calendars           []Resource `json:"calendars,omitempty"`           // vCard CALURI only
	FreeBusyURLs        []Resource `json:"freeBusyUrls,omitempty"`        // vCard FBURL only
	SchedulingAddresses []Resource `json:"schedulingAddresses,omitempty"` // vCard CALADRURI
	CryptoKeys          []Resource `json:"cryptoKeys,omitempty"`          // vCard KEY
	Directories         []Resource `json:"directories,omitempty"`         // kind: directory|entry
	Links               []Resource `json:"links,omitempty"`               // vCard URL only
	ContactURIs         []Resource `json:"contactUris,omitempty"`         // vCard CONTACT-URI only (8605)

	PreferredLanguages []LanguagePref `json:"preferredLanguages,omitempty"` // vCard LANG
	RelatedTo          []Relation     `json:"relatedTo,omitempty"`          // vCard RELATED
	Members            []string       `json:"members,omitempty"`            // vCard MEMBER (group kind)

	// Localizations preserved opaquely for now (advanced; see 20 §Localizations).
	Localizations map[string]json.RawMessage `json:"localizations,omitempty"`
}

// Name is the neutral home for JSContact `name` and vCard FN/N.
type Name struct {
	Components       []NameComponent   `json:"components,omitempty"`
	Full             string            `json:"full,omitempty"`   // vCard FN
	SortAs           map[string]string `json:"sortAs,omitempty"` // component-kind -> sort string
	IsOrdered        *bool             `json:"isOrdered,omitempty"`
	DefaultSeparator string            `json:"defaultSeparator,omitempty"`
	PhoneticSystem   string            `json:"phoneticSystem,omitempty"` // ipa|jyut|piny
	PhoneticScript   string            `json:"phoneticScript,omitempty"`
}

// NameComponent is one labeled piece of a Name (e.g. "given" -> "Ada").
// Kind ∈ title,given,given2,surname,surname2,credential,generation,separator
type NameComponent struct {
	Kind     string `json:"kind"`
	Value    string `json:"value"`
	Phonetic string `json:"phonetic,omitempty"`
}

// Nickname is a single nickname entry (JSContact nicknames / vCard NICKNAME).
type Nickname struct {
	ID       string   `json:"id,omitempty"`
	Name     string   `json:"name"`
	Contexts []string `json:"contexts,omitempty"` // private|work
	Pref     *int     `json:"pref,omitempty"`
}

// Organization is one employer/org affiliation (JSContact organizations /
// vCard ORG), optionally split into OrgUnit sub-units.
type Organization struct {
	ID     string    `json:"id,omitempty"`
	Name   string    `json:"name,omitempty"`
	Units  []OrgUnit `json:"units,omitempty"`
	SortAs string    `json:"sortAs,omitempty"`
}

// OrgUnit is one named sub-unit within an Organization's Units.
type OrgUnit struct {
	Name   string `json:"name"`
	SortAs string `json:"sortAs,omitempty"`
}

// Title is a job title or role (JSContact titles / vCard TITLE, ROLE).
// Kind ∈ title|role
type Title struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name"`
	Kind           string `json:"kind,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
}

// Email is one email address entry (JSContact emails / vCard EMAIL).
type Email struct {
	ID       string   `json:"id,omitempty"`
	Address  string   `json:"address"`
	Contexts []string `json:"contexts,omitempty"` // private|work
	Pref     *int     `json:"pref,omitempty"`
	Label    string   `json:"label,omitempty"`
}

// Phone is one phone number entry (JSContact phones / vCard TEL).
// Features ∈ voice,fax,cell/mobile,video,pager,text,textphone,main-number
type Phone struct {
	ID       string   `json:"id,omitempty"`
	Number   string   `json:"number"`
	Features []string `json:"features,omitempty"`
	Contexts []string `json:"contexts,omitempty"`
	Pref     *int     `json:"pref,omitempty"`
	Label    string   `json:"label,omitempty"`
}

// OnlineService is one online-presence entry: an IMPP address, an
// RFC 9554 SOCIALPROFILE, or an unclassified other-online-service (see
// Card.ImppAddresses/SocialProfiles/OtherOnlineServices for which field
// each belongs to).
type OnlineService struct {
	ID       string   `json:"id,omitempty"`
	Service  string   `json:"service,omitempty"` // e.g. "Mastodon"
	URI      string   `json:"uri,omitempty"`
	User     string   `json:"user,omitempty"`
	Contexts []string `json:"contexts,omitempty"`
	Pref     *int     `json:"pref,omitempty"`
	Label    string   `json:"label,omitempty"`
}

// Address is one postal address entry (JSContact addresses / vCard ADR).
type Address struct {
	ID               string             `json:"id,omitempty"`
	Components       []AddressComponent `json:"components,omitempty"`
	CountryCode      string             `json:"countryCode,omitempty"` // vCard CC param
	Coordinates      string             `json:"coordinates,omitempty"` // geo: URI  (vCard GEO)
	TimeZone         string             `json:"timeZone,omitempty"`    // vCard TZ
	Contexts         []string           `json:"contexts,omitempty"`    // billing|delivery|private|work
	Pref             *int               `json:"pref,omitempty"`
	Full             string             `json:"full,omitempty"` // vCard LABEL
	IsOrdered        *bool              `json:"isOrdered,omitempty"`
	DefaultSeparator string             `json:"defaultSeparator,omitempty"`
	PhoneticSystem   string             `json:"phoneticSystem,omitempty"`
	PhoneticScript   string             `json:"phoneticScript,omitempty"`
}

// AddressComponent is one labeled piece of an Address (e.g. "locality" ->
// "Springfield"). Kind ∈ room,apartment,floor,building,number,name,block,
// subdistrict,district,locality,region,postcode,country,direction,landmark,
// postOfficeBox,separator
type AddressComponent struct {
	Kind     string `json:"kind"`
	Value    string `json:"value"`
	Phonetic string `json:"phonetic,omitempty"`
}

// Anniversary is one dated life event (JSContact anniversaries / vCard
// BDAY, ANNIVERSARY, DEATHDATE). Kind ∈ birth|death|wedding
type Anniversary struct {
	ID    string          `json:"id,omitempty"`
	Kind  string          `json:"kind"`
	Date  AnniversaryDate `json:"date"`
	Place *Address        `json:"place,omitempty"` // vCard BIRTHPLACE/DEATHPLACE
}

// AnniversaryDate holds an Anniversary's date, either a full RFC3339
// Timestamp or a partial/year-unknown PartialDate.
type AnniversaryDate struct {
	Partial   *PartialDate `json:"partial,omitempty"`
	Timestamp *string      `json:"timestamp,omitempty"` // RFC3339
}

// PartialDate is a date with one or more components (typically the year)
// deliberately unknown, e.g. a birthday recorded without a year.
type PartialDate struct {
	Year          *int   `json:"year,omitempty"`
	Month         *int   `json:"month,omitempty"`
	Day           *int   `json:"day,omitempty"`
	CalendarScale string `json:"calendarScale,omitempty"` // gregorian
}

// SpeakToAs holds gendered/pronoun-related addressing preferences.
type SpeakToAs struct {
	// GrammaticalGenders is multi-valued (RFC 9554 §3.2 cardinality "*", one
	// per LANGUAGE — "multiple occurrences ... MUST be distinguished by the
	// LANGUAGE parameter"). Import must store every occurrence losslessly; a
	// scalar field would silently drop data on import from a real vCard4
	// source, which the degradation policy forbids. Loss only happens on
	// EXPORT to a format that is inherently single-valued (JSContact's own
	// `speakToAs.grammaticalGender` is a scalar, RFC 9553 §2.2.4) — that
	// narrowing belongs to the export adapter, not this model.
	GrammaticalGenders []GrammaticalGender `json:"grammaticalGenders,omitempty"`
	Pronouns           []Pronouns          `json:"pronouns,omitempty"` // vCard PRONOUNS (9554)
}

// GrammaticalGender is one language-scoped grammatical-gender preference
// (RFC 9554 GRAMGENDER). Value ∈ animate|common|feminine|inanimate|masculine|neuter
type GrammaticalGender struct {
	ID       string `json:"id,omitempty"`
	Value    string `json:"value"`
	Language string `json:"language,omitempty"` // RFC 9554 §3.2 LANGUAGE param; no PREF param exists for GRAMGENDER
}

// Pronouns is one set of personal pronouns (RFC 9554 PRONOUNS).
type Pronouns struct {
	ID       string   `json:"id,omitempty"`
	Pronouns string   `json:"pronouns"` // the text, e.g. "they/them" — confirmed field name, RFC 9553 §2.2.4
	Contexts []string `json:"contexts,omitempty"`
	Pref     *int     `json:"pref,omitempty"`
}

// PersonalInfo is one hobby/interest/expertise entry (JSContact personalInfo
// / vCard EXPERTISE, HOBBY, INTEREST). Kind ∈ expertise|hobby|interest ;
// Level ∈ high|medium|low
type PersonalInfo struct {
	ID     string `json:"id,omitempty"`
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Level  string `json:"level,omitempty"`
	ListAs *int   `json:"listAs,omitempty"`
	Label  string `json:"label,omitempty"`
}

// Note is one free-text note (JSContact notes / vCard NOTE).
type Note struct {
	ID      string     `json:"id,omitempty"`
	Note    string     `json:"note"`
	Author  *Author    `json:"author,omitempty"`  // vCard AUTHOR/AUTHOR-NAME params (9554)
	Created *Timestamp `json:"created,omitempty"` // vCard CREATED param (9554)
}

// Author identifies who wrote a Note (RFC 9554 AUTHOR/AUTHOR-NAME params).
type Author struct {
	Name string `json:"name,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// Resource is the shared shape for media/calendars/freeBusyURLs/schedulingAddresses/
// cryptoKeys/directories/links/contactURIs. Kind is only meaningful for Media
// (photo|logo|sound) and Directories (directory|entry), which are genuinely
// Kind-discriminated in JSContact's own object model; the other Resource-typed
// fields are single-purpose (one vCard property each) and leave Kind unset.
type Resource struct {
	ID        string   `json:"id,omitempty"`
	Kind      string   `json:"kind,omitempty"` // photo|logo|sound (Media) | directory|entry (Directories)
	URI       string   `json:"uri"`
	MediaType string   `json:"mediaType,omitempty"`
	Label     string   `json:"label,omitempty"`
	Contexts  []string `json:"contexts,omitempty"`
	Pref      *int     `json:"pref,omitempty"`
	ListAs    *int     `json:"listAs,omitempty"` // directories only
}

// LanguagePref is one preferred-language entry (JSContact preferredLanguages
// / vCard LANG).
type LanguagePref struct {
	ID       string   `json:"id,omitempty"`
	Language string   `json:"language"`
	Contexts []string `json:"contexts,omitempty"`
	Pref     *int     `json:"pref,omitempty"`
}

// Relation is one link to another entity (JSContact relatedTo / vCard
// RELATED). Relations keys ∈ the relatedTo enum (acquaintance,friend,child,
// parent,…,emergency)
type Relation struct {
	Target    string   `json:"target"` // uri or uid of the related entity
	Relations []string `json:"relations,omitempty"`
}

// Timestamp is an RFC3339 instant, used wherever the neutral model needs a
// full date-time rather than a PartialDate.
type Timestamp struct {
	UTC string `json:"utc"` // RFC3339
}

// Record is one contact end to end: standardized Card + CRM sibling + preserved unknowns.
type Record struct {
	Card        Card        `json:"card"`
	Envelope    CRMEnvelope `json:"crm"`
	Passthrough Passthrough `json:"passthrough,omitempty"`
	// Identity/sync (populated by the persistence layer in P1, unused by adapters):
	UID  string `json:"-"`
	ETag string `json:"-"`
}
