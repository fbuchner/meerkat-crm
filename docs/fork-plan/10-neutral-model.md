# 10 — Neutral internal model (`backend/contactmodel`)  ·  WP-10

Pure data types. **No parsing, no mapping, no gorm, no imports of other new packages.** The neutral
model is a *superset* shaped closely after JSContact (the richest registry) but is our own type set,
so adapters can populate it from any format. Collections are **ordered slices whose elements carry an
optional stable `ID`** — this preserves JSContact map keys and vCard `PROP-ID`s without forcing a map.

Sub-agent: create the files below verbatim in shape; field names are binding (the correspondence table
references them by Go path). Add doc comments citing the registry object. **"Constructor + round-trip
test" (WP-10 acceptance) means:** for each exported type, a test that builds a fully-populated Go
literal, `json.Marshal`s it, `json.Unmarshal`s the result into a fresh value, and asserts
`reflect.DeepEqual` with the original. No `New*()` builder functions are required or expected —
callers (adapters) construct these as plain struct literals.

**Element `ID` fields serialize.** Every collection-element type (`Nickname`, `Organization`, `Title`,
`Email`, `Phone`, `OnlineService`, `Address`, `Anniversary`, `Pronouns`, `GrammaticalGender`,
`PersonalInfo`, `Note`, `Resource`, `LanguagePref`) carries `ID string \`json:"id,omitempty"\`` — **not**
`json:"-"`. This is
required because `Record.Card` is persisted as a JSON DB column (WP-70) and the `PROP-ID`/JSContact-map-key
round-trip invariant (`30-adapters.md` §30.D) depends on the ID surviving a save/reload cycle. Only
`Record.UID` and `Record.ETag` (persistence-layer identity, stored as separate DB columns, not part of
the Card payload) stay `json:"-"`.

## 10.1 `model.go` — the Card payload

```go
package contactmodel

import "encoding/json"

// Card is the neutral superset of a single contact's standardized data
// (union of RFC 9553 JSContact + the full IANA vCard-elements registry).
type Card struct {
    UID      string     `json:"uid,omitempty"`
    Kind     string     `json:"kind,omitempty"`      // individual|group|org|location|application|device
    Language string     `json:"language,omitempty"`  // default language tag for the card
    ProdID   string     `json:"prodId,omitempty"`
    Created  *Timestamp `json:"created,omitempty"`   // RFC 9554 CREATED
    Updated  *Timestamp `json:"updated,omitempty"`   // vCard REV

    Name           *Name           `json:"name,omitempty"`
    Nicknames      []Nickname      `json:"nicknames,omitempty"`
    Organizations  []Organization  `json:"organizations,omitempty"`
    Titles         []Title         `json:"titles,omitempty"`
    Emails         []Email         `json:"emails,omitempty"`
    Phones         []Phone         `json:"phones,omitempty"`
    ImppAddresses       []OnlineService `json:"imppAddresses,omitempty"`       // vCard IMPP only
    SocialProfiles      []OnlineService `json:"socialProfiles,omitempty"`      // vCard SOCIALPROFILE only (9554)
    OtherOnlineServices []OnlineService `json:"otherOnlineServices,omitempty"` // unclassified: GUI-added with no vCard-property preference, or JSContact-imported with no `vCardName` hint (RFC 9555 §2.15.3). NOT exported to vCard (no safe default property to guess) — see 20-correspondence.md §20.7
    Addresses      []Address       `json:"addresses,omitempty"`
    Anniversaries  []Anniversary   `json:"anniversaries,omitempty"`
    SpeakToAs      *SpeakToAs      `json:"speakToAs,omitempty"`
    PersonalInfo   []PersonalInfo  `json:"personalInfo,omitempty"`
    Notes          []Note          `json:"notes,omitempty"`
    Keywords       []string        `json:"keywords,omitempty"` // vCard CATEGORIES

    // Resource-shaped collections (all reuse Resource). Each field below is now
    // SINGLE-PURPOSE — one vCard property per field, never Kind-shared — so that
    // import always lands in an unambiguous, discretely-typed home and never has
    // to guess a value's origin on export (see 60-review-gates.md's post-P0 review
    // notes: this replaced an earlier Kind-discriminated design that silently
    // conflated IMPP/SOCIALPROFILE, URL/CONTACT-URI, and CALURI/FBURL). `Resource.Kind`
    // is still used by `Media` (photo|logo|sound) and `Directories` (directory|entry),
    // which remain genuinely Kind-discriminated per JSContact's own object model.
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
```

## 10.2 `model.go` — sub-objects (one struct per registry object)

Common optional fields appear explicitly (no embedding) so JSON shapes stay exact.

```go
type Name struct {
    Components       []NameComponent   `json:"components,omitempty"`
    Full             string            `json:"full,omitempty"`      // vCard FN
    SortAs           map[string]string `json:"sortAs,omitempty"`    // component-kind -> sort string
    IsOrdered        *bool             `json:"isOrdered,omitempty"`
    DefaultSeparator string            `json:"defaultSeparator,omitempty"`
    PhoneticSystem   string            `json:"phoneticSystem,omitempty"` // ipa|jyut|piny
    PhoneticScript   string            `json:"phoneticScript,omitempty"`
}
// NameComponent.Kind ∈ title,given,given2,surname,surname2,credential,generation,separator
type NameComponent struct {
    Kind     string `json:"kind"`
    Value    string `json:"value"`
    Phonetic string `json:"phonetic,omitempty"`
}

type Nickname struct {
    ID       string   `json:"id,omitempty"`
    Name     string   `json:"name"`
    Contexts []string `json:"contexts,omitempty"` // private|work
    Pref     *int     `json:"pref,omitempty"`
}

type Organization struct {
    ID     string    `json:"id,omitempty"`
    Name   string    `json:"name,omitempty"`
    Units  []OrgUnit `json:"units,omitempty"`
    SortAs string    `json:"sortAs,omitempty"`
}
type OrgUnit struct {
    Name   string `json:"name"`
    SortAs string `json:"sortAs,omitempty"`
}

// Title.Kind ∈ title|role
type Title struct {
    ID             string `json:"id,omitempty"`
    Name           string `json:"name"`
    Kind           string `json:"kind,omitempty"`
    OrganizationID string `json:"organizationId,omitempty"`
}

type Email struct {
    ID       string   `json:"id,omitempty"`
    Address  string   `json:"address"`
    Contexts []string `json:"contexts,omitempty"` // private|work
    Pref     *int     `json:"pref,omitempty"`
    Label    string   `json:"label,omitempty"`
}

// Phone.Features ∈ voice,fax,cell/mobile,video,pager,text,textphone,main-number
type Phone struct {
    ID       string   `json:"id,omitempty"`
    Number   string   `json:"number"`
    Features []string `json:"features,omitempty"`
    Contexts []string `json:"contexts,omitempty"`
    Pref     *int     `json:"pref,omitempty"`
    Label    string   `json:"label,omitempty"`
}

type OnlineService struct {
    ID       string   `json:"id,omitempty"`
    Service  string   `json:"service,omitempty"` // e.g. "Mastodon"
    URI      string   `json:"uri,omitempty"`
    User     string   `json:"user,omitempty"`
    Contexts []string `json:"contexts,omitempty"`
    Pref     *int     `json:"pref,omitempty"`
    Label    string   `json:"label,omitempty"`
}

type Address struct {
    ID               string             `json:"id,omitempty"`
    Components       []AddressComponent `json:"components,omitempty"`
    CountryCode      string             `json:"countryCode,omitempty"` // vCard CC param
    Coordinates      string             `json:"coordinates,omitempty"` // geo: URI  (vCard GEO)
    TimeZone         string             `json:"timeZone,omitempty"`    // vCard TZ
    Contexts         []string           `json:"contexts,omitempty"`    // billing|delivery|private|work
    Pref             *int               `json:"pref,omitempty"`
    Full             string             `json:"full,omitempty"`        // vCard LABEL
    IsOrdered        *bool              `json:"isOrdered,omitempty"`
    DefaultSeparator string             `json:"defaultSeparator,omitempty"`
    PhoneticSystem   string             `json:"phoneticSystem,omitempty"`
    PhoneticScript   string             `json:"phoneticScript,omitempty"`
}
// AddressComponent.Kind ∈ room,apartment,floor,building,number,name,block,subdistrict,
//   district,locality,region,postcode,country,direction,landmark,postOfficeBox,separator
type AddressComponent struct {
    Kind     string `json:"kind"`
    Value    string `json:"value"`
    Phonetic string `json:"phonetic,omitempty"`
}

// Anniversary.Kind ∈ birth|death|wedding
type Anniversary struct {
    ID    string          `json:"id,omitempty"`
    Kind  string          `json:"kind"`
    Date  AnniversaryDate `json:"date"`
    Place *Address        `json:"place,omitempty"` // vCard BIRTHPLACE/DEATHPLACE
}
type AnniversaryDate struct {
    Partial   *PartialDate `json:"partial,omitempty"`
    Timestamp *string      `json:"timestamp,omitempty"` // RFC3339
}
type PartialDate struct {
    Year          *int   `json:"year,omitempty"`
    Month         *int   `json:"month,omitempty"`
    Day           *int   `json:"day,omitempty"`
    CalendarScale string `json:"calendarScale,omitempty"` // gregorian
}

type SpeakToAs struct {
    // GrammaticalGenders is multi-valued (RFC 9554 §3.2 cardinality "*", one
    // per LANGUAGE — "multiple occurrences ... MUST be distinguished by the
    // LANGUAGE parameter"). Import must store every occurrence losslessly; a
    // scalar field would silently drop data on import from a real vCard4
    // source, which the degradation policy (0.5) forbids. Loss only happens
    // on EXPORT to a format that is inherently single-valued (JSContact's own
    // `speakToAs.grammaticalGender` is a scalar, RFC 9553 §2.2.4) — see 20.7
    // for the export-selection rule.
    GrammaticalGenders []GrammaticalGender `json:"grammaticalGenders,omitempty"`
    Pronouns           []Pronouns          `json:"pronouns,omitempty"` // vCard PRONOUNS (9554)
}
// GrammaticalGender.Value ∈ animate|common|feminine|inanimate|masculine|neuter
type GrammaticalGender struct {
    ID       string `json:"id,omitempty"`
    Value    string `json:"value"`
    Language string `json:"language,omitempty"` // RFC 9554 §3.2 LANGUAGE param; no PREF param exists for GRAMGENDER
}
type Pronouns struct {
    ID       string   `json:"id,omitempty"`
    Pronouns string   `json:"pronouns"` // the text, e.g. "they/them" — confirmed field name, RFC 9553 §2.2.4
    Contexts []string `json:"contexts,omitempty"`
    Pref     *int     `json:"pref,omitempty"`
}

// PersonalInfo.Kind ∈ expertise|hobby|interest ; Level ∈ high|medium|low
type PersonalInfo struct {
    ID     string `json:"id,omitempty"`
    Kind   string `json:"kind"`
    Value  string `json:"value"`
    Level  string `json:"level,omitempty"`
    ListAs *int   `json:"listAs,omitempty"`
    Label  string `json:"label,omitempty"`
}

type Note struct {
    ID      string     `json:"id,omitempty"`
    Note    string     `json:"note"`
    Author  *Author    `json:"author,omitempty"`  // vCard AUTHOR/AUTHOR-NAME params (9554)
    Created *Timestamp `json:"created,omitempty"` // vCard CREATED param (9554)
}
type Author struct {
    Name string `json:"name,omitempty"`
    URI  string `json:"uri,omitempty"`
}

// Resource is the shared shape for media/calendars/schedulingAddresses/cryptoKeys/directories/links.
type Resource struct {
    ID        string   `json:"id,omitempty"`
    Kind      string   `json:"kind,omitempty"` // photo|logo|sound | calendar|freeBusy | directory|entry | contact
    URI       string   `json:"uri"`
    MediaType string   `json:"mediaType,omitempty"`
    Label     string   `json:"label,omitempty"`
    Contexts  []string `json:"contexts,omitempty"`
    Pref      *int     `json:"pref,omitempty"`
    ListAs    *int     `json:"listAs,omitempty"` // directories only
}

type LanguagePref struct {
    ID       string   `json:"id,omitempty"`
    Language string   `json:"language"`
    Contexts []string `json:"contexts,omitempty"`
    Pref     *int     `json:"pref,omitempty"`
}

// Relation.Relations keys ∈ the relatedTo enum (acquaintance,friend,child,parent,…,emergency)
type Relation struct {
    Target    string   `json:"target"` // uri or uid of the related entity
    Relations []string `json:"relations,omitempty"`
}

type Timestamp struct {
    UTC string `json:"utc"` // RFC3339
}
```

## 10.3 `envelope.go` — CRM-only sibling (never touched by adapters)

```go
package contactmodel

// CRMEnvelope holds Meerkat-specific data that is NOT part of any contact-exchange
// standard. Format adapters MUST ignore it entirely.
type CRMEnvelope struct {
    Circles            []string          `json:"circles,omitempty"`
    HowWeMet           string            `json:"how_we_met,omitempty"`
    FoodPreference     string            `json:"food_preference,omitempty"`
    WorkInformation    string            `json:"work_information,omitempty"`
    ContactInformation string            `json:"contact_information,omitempty"`
    CustomFields       map[string]string `json:"custom_fields,omitempty"`
    // Reminders/Activities/Relationships remain separate GORM tables keyed by contact ID;
    // they are NOT embedded here.
}
```

## 10.4 `passthrough.go` — spec-blessed escape hatches (RFC 9555)

```go
package contactmodel

import "encoding/json"

// Passthrough preserves data with no neutral/target home so nothing is silently lost.
type Passthrough struct {
    // Unknown vCard properties captured on vCard import (RFC 9555 "vCardProps" shape).
    VCard []JCardProp `json:"vCardProps,omitempty"`
    // Unknown JSContact properties captured on JSContact import, keyed by JSON pointer.
    JSContact map[string]json.RawMessage `json:"jsContactProps,omitempty"`
}

// JCardProp is one jCard property array: [name, params, valuetype, value...].
type JCardProp struct {
    Name   string         `json:"name"`
    Params map[string]any `json:"params,omitempty"`
    Type   string         `json:"type"` // jCard value type, e.g. "text"
    Value  json.RawMessage `json:"value"`
}
```

## 10.5 `model.go` — the top-level Record

```go
package contactmodel

// Record is one contact end to end: standardized Card + CRM sibling + preserved unknowns.
type Record struct {
    Card        Card        `json:"card"`
    Envelope    CRMEnvelope `json:"crm"`
    Passthrough Passthrough `json:"passthrough,omitempty"`
    // Identity/sync (populated by the persistence layer in P1, unused by adapters):
    UID  string `json:"-"`
    ETag string `json:"-"`
}
```

## 10.6 `projection.go` — derived columns (used later by P1 persistence)

Pure function; no DB. Extends the intent of today's `BeforeSave` denormalization.

```go
package contactmodel

type Projection struct {
    Firstname, Lastname, FN string
    PrimaryEmail, PrimaryPhone string
    Birthday string // ISO or --MM-DD
    Org string
}

func DeriveProjection(r *Record) Projection // spec:
// Firstname = first NameComponent kind=given .Value; Lastname = kind=surname .Value
// FN = Card.Name.Full (fallback: join given+surname)
// PrimaryEmail = Emails sorted by (Pref asc, index) [0].Address
// PrimaryPhone = Phones sorted by (Pref asc, index) [0].Number
// Birthday = Anniversaries where Kind=="birth" -> partial/timestamp formatted ISO
// Org = Organizations[0].Name
```

## 10.7 Field inventory & registry cross-reference (coverage checklist)

The neutral model has a home for **every** JSContact property and **every** vCard-elements property.
This table is the coverage contract; WP-20's test verifies each `neutral_path` resolves.

| Registry element(s) | Neutral home (`contactmodel` path) |
|---|---|
| JSContact `name`, vCard `FN`,`N` | `Card.Name` (+ `.Components`, `.Full`, `.SortAs`, phonetics) |
| `nicknames`, `NICKNAME` | `Card.Nicknames[]` |
| `organizations`, `ORG`,`ORG-DIRECTORY` | `Card.Organizations[]` (+ directory as `Card.Directories`) |
| `titles`, `TITLE`,`ROLE` | `Card.Titles[]` (`Kind` role/title) |
| `emails`, `EMAIL` | `Card.Emails[]` |
| `phones`, `TEL` | `Card.Phones[]` |
| `onlineServices` (`vCardName`=IMPP), `IMPP` | `Card.ImppAddresses[]` |
| `onlineServices` (`vCardName`=SOCIALPROFILE), `SOCIALPROFILE` | `Card.SocialProfiles[]` (9554) |
| `onlineServices` (no `vCardName` hint) | `Card.OtherOnlineServices[]` — not exported to vCard (warn-drop, no safe default) |
| `addresses`, `ADR`,`GEO`,`TZ` | `Card.Addresses[]` |
| `anniversaries`, `BDAY`,`ANNIVERSARY`,`DEATHDATE`,`BIRTHPLACE`,`DEATHPLACE` | `Card.Anniversaries[]` (+ `.Place`) |
| `speakToAs`, `GRAMGENDER`,`PRONOUNS` | `Card.SpeakToAs` |
| `personalInfo`, `EXPERTISE`,`HOBBY`,`INTEREST` | `Card.PersonalInfo[]` |
| `notes`, `NOTE` | `Card.Notes[]` |
| `keywords`, `CATEGORIES` | `Card.Keywords[]` |
| `media`, `PHOTO`,`LOGO`,`SOUND` | `Card.Media[]` (`Kind`) |
| `calendars` (kind=calendar), `CALURI` | `Card.Calendars[]` |
| `calendars` (kind=freeBusy), `FBURL` | `Card.FreeBusyURLs[]` |
| `schedulingAddresses`, `CALADRURI` | `Card.SchedulingAddresses[]` |
| `cryptoKeys`, `KEY` | `Card.CryptoKeys[]` |
| `directories`, `SOURCE`,`ORG-DIRECTORY` | `Card.Directories[]` |
| `links` (no kind), `URL` | `Card.Links[]` |
| `links` (kind=contact), `CONTACT-URI` | `Card.ContactURIs[]` (8605) |
| `preferredLanguages`, `LANG` | `Card.PreferredLanguages[]` |
| `relatedTo`, `RELATED` | `Card.RelatedTo[]` |
| `members`, `MEMBER` | `Card.Members[]` |
| `uid`,`kind`,`language`,`prodId`,`created`,`updated`; `UID`,`KIND`,`PRODID`,`REV`,`CREATED` | `Card.{UID,Kind,Language,ProdID,Created,Updated}` |
| `vCardProps`/`JSPROP` and any unlisted/x-name property | `Record.Passthrough` |
| Meerkat-only (circles, how_we_met, …) | `Record.Envelope` |

**Previously-open points — now RESOLVED against the RFCs (see `docs/specs/`), no guessing required:**
- `Pronouns.Pronouns` — **confirmed**: RFC 9553 §2.2.4 defines Pronouns object property `pronouns`
  (String, mandatory) as the pronoun text. Field name stands. Pronouns also has `contexts`, `pref`.
- `preferredLanguages` — **confirmed** `Id[LanguagePref]` (RFC 9553 §2.3.4). Model as slice + `ID`
  (the codec maps the JSContact map key ↔ `LanguagePref.ID`), consistent with emails/phones.
- `localizations` — `String[PatchObject]` keyed by language tag (RFC 9553 §2.7.1). **P0 decision:
  keep opaque** (`map[string]json.RawMessage`); do not synthesize patches. See
  `docs/specs/rfc9555-correspondence.md` §3 (LANGUAGE) for how tagged alternates arrive.

No `// VERIFY:` comments remain for WP-10.
