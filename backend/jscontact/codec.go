package jscontact

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"meerkat/contactmodel"
)

// Marshal renders a Card to its RFC 9553 JSON wire form. Every object gets its
// correct @type (codec rule 1); Card.version is always "1.0" on export
// (codec rule 2); Id-keyed collections (emails, phones, addresses, ...) are
// emitted as JSON maps keyed by each element's ID, generating a synthetic
// "k<index>" key when ID is empty, in slice order (codec rule 3); boolean-set
// fields (contexts, features, keywords, members, relation, ...) are emitted
// as JSON `{"value":true}` maps (codec rule 4).
func Marshal(c *Card) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("jscontact: cannot marshal a nil Card")
	}
	return json.Marshal(c)
}

// Unmarshal parses RFC 9553 JSON wire bytes into a Card, performing the
// inverse of Marshal's conversions. @type is validated if present and
// defaulted if absent (codec rule 1); any Card.version value is accepted
// (codec rule 2).
func Unmarshal(data []byte) (*Card, error) {
	var c Card
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// --- shared helpers -------------------------------------------------------

// checkType validates an object's @type if present, or defaults it to want
// if absent (empty). Returns an error only on an explicit mismatch.
func checkType(got *string, want string) error {
	if *got == "" {
		*got = want
		return nil
	}
	if *got != want {
		return fmt.Errorf("jscontact: expected @type %q, got %q", want, *got)
	}
	return nil
}

// boolSetMarshal converts a neutral []string set into the wire
// `{"value":true,...}` boolean-set-map shape.
func boolSetMarshal(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}

// boolSetUnmarshal converts a wire boolean-set-map back into a deterministic
// (sorted) []string. Entries explicitly set to false are dropped.
func boolSetUnmarshal(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

// sortedKeys returns a map's keys in sorted order, giving a deterministic
// iteration order over JSON object maps (which Go/JSON do not otherwise
// guarantee).
func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// idMapToSlice converts a wire Id-keyed map into an ordered slice, assigning
// each element's key (via setKey) in sorted-key order. Per-element @type
// validation/defaulting has already happened by this point: it runs inside
// each element type's own UnmarshalJSON, invoked automatically while decoding
// the map[string]T field on the enclosing wire struct.
func idMapToSlice[T any](m map[string]T, setKey func(*T, string)) []T {
	if len(m) == 0 {
		return nil
	}
	keys := sortedKeys(m)
	out := make([]T, 0, len(m))
	for _, k := range keys {
		v := m[k]
		setKey(&v, k)
		out = append(out, v)
	}
	return out
}

// idMapFromSlice converts an ordered slice into a wire Id-keyed map, using
// getKey(element) as the map key, or a synthetic "k<index>" when that key is
// empty. Each element's own MarshalJSON (invoked automatically while encoding
// the resulting map[string]T) is responsible for emitting its own @type.
func idMapFromSlice[T any](items []T, getKey func(T) string) map[string]T {
	if len(items) == 0 {
		return nil
	}
	m := make(map[string]T, len(items))
	for i, v := range items {
		k := getKey(v)
		if k == "" {
			k = fmt.Sprintf("k%d", i)
		}
		m[k] = v
	}
	return m
}

// --- nested-unknown-property capture (passthrough) --------------------------
//
// Bug fixed here: every wire type below with a custom UnmarshalJSON used to
// decode via a plain json.Unmarshal(data, &typedStruct)-shaped call. Go's
// default unmarshal behavior silently discards any JSON object key that
// doesn't match a struct field's JSON tag, so an unrecognized key nested
// inside a known object (e.g. an extra key inside one emails{} map entry, or
// inside "name") was lost before adapter.go ever got a chance to preserve it
// -- adapter.go's importUnknownTopLevel/spliceJSContactPassthrough only ever
// covered the Card's own top-level properties. This violates
// docs/fork-plan/00-overview.md §0.5 ("genuinely unmappable/unknown data...
// preserve, don't reject") and docs/fork-plan/30-adapters.md §30.A rule 5
// ("Unknown top-level or object properties on import ->
// Record.Passthrough.JSContact[pointer]" -- "or object" means nested-in-an-
// object too, not just top-level).
//
// Fix: every wire type gets an unexported `extra` field (types.go) capturing
// exactly the JSON keys its own struct doesn't recognize (captureExtra,
// called from UnmarshalJSON below) and re-emitting them on Marshal
// (mergeExtraJSON, called from MarshalJSON below), so an unrecognized nested
// key round-trips through this codec byte-for-byte-equivalent even though no
// Go field names it. adapter.go's importNestedPassthrough/
// collectNestedPassthrough then walk the just-unmarshaled Card, gathering
// every populated `extra` map into Record.Passthrough.JSContact keyed by the
// JSON pointer to where it actually lives (e.g. "/emails/k1/x-custom", not
// just "/x-custom") -- see adapter.go for that walk and for the generalized
// (now arbitrary-depth) export-side splice.
//
// Scope decision (deliberately not applied to every possible nesting):
//   - Card itself does NOT get this mechanism -- its top-level unknowns were
//     already correctly handled by adapter.go's pre-existing
//     importUnknownTopLevel/spliceJSContactPassthrough, which predates this
//     fix and is untouched (beyond generalizing the splice to work at any
//     depth, which also benefits the nested case).
//   - PartialDate and Timestamp do NOT get this mechanism: RFC 9553 §2.8.1's
//     date leaf value types have a tiny, stable, well-known field set
//     (year/month/day/calendarScale; utc) that a real producer is very
//     unlikely to extend, and both are only ever reached through
//     Anniversary.date's already-unusual PartialDate|Timestamp polymorphic
//     union. Adding capture there would add real complexity (a third,
//     ambiguous "which one is it" capture path inside that union) for
//     negligible real-world benefit -- exactly the kind of "simple leaf
//     type" exclusion flagged as plausible in the task that produced this
//     fix. If this assumption ever proves wrong, the same mechanism can be
//     added identically to these two types.
// Every other wire type in this package -- every Id-map collection element
// directly on Card (Nickname, Organization, Title, EmailAddress, Phone,
// OnlineService, Address, Anniversary, PersonalInfo, Note, Media, Calendar,
// SchedulingAddress, CryptoKey, Directory, Link, LanguagePref, Relation),
// nested non-Id-map array elements (NameComponent, AddressComponent,
// OrgUnit), a nested Id-map element (Pronouns), and nested singular objects
// (Name, SpeakToAs, Author) -- gets the mechanism: RFC 9553 permits vendor
// extensions on any JSContact object, and a real producer extending a Card
// is plausibly going to attach that extension to a collection element it
// authored, which is exactly the case this fix's bug report calls out.

// captureExtra re-parses data (the same bytes passed to a type's
// UnmarshalJSON) generically and returns every top-level object key not
// present in known, or nil if there are none. known should be the result of
// knownJSONKeys[T]() for that same type T.
func captureExtra(data []byte, known map[string]bool) map[string]json.RawMessage {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		// The caller's own json.Unmarshal into the typed struct already
		// succeeded (or this is being called from that same UnmarshalJSON
		// after its own decode step succeeded), so re-parsing generically
		// into map[string]json.RawMessage cannot fail for any data that was
		// itself a JSON object; guarded defensively (fail closed: no extras
		// captured) rather than assumed impossible.
		return nil
	}
	var extra map[string]json.RawMessage
	for k, v := range doc {
		if known[k] {
			continue
		}
		if extra == nil {
			extra = map[string]json.RawMessage{}
		}
		extra[k] = v
	}
	return extra
}

// mergeExtraJSON splices extra's entries into the already-marshaled JSON
// object base, skipping any key base already defines (a known field must
// never be shadowed or duplicated by a captured-unknown one -- the same
// de-dup guard adapter.go applies at the top level, 20.5). Returns base
// unchanged if extra is empty (the overwhelmingly common case, so this never
// touches the marshal fast path for objects with no captured unknowns).
func mergeExtraJSON(base []byte, extra map[string]json.RawMessage) ([]byte, error) {
	if len(extra) == 0 {
		return base, nil
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(base, &doc); err != nil {
		return nil, err
	}
	for k, v := range extra {
		if _, exists := doc[k]; exists {
			continue
		}
		doc[k] = v
	}
	return json.Marshal(doc)
}

// knownJSONKeys returns the set of JSON object keys type T's own struct
// fields recognize: every field's JSON tag name (or Go field name, if
// untagged), excluding "-"-tagged and unexported fields (e.g. the `extra`
// field itself). It reflects on T directly -- never on a private wire/alias
// struct a type's Marshal/UnmarshalJSON might additionally use internally --
// because every wire type in this package is written so that any such
// wire/alias struct only ever overrides a field that already carries the
// same JSON tag on the plain type itself (e.g. Address's boolean-set
// "contexts" wire override still marshals/unmarshals under the same
// "contexts" key as the plain Address.Contexts field): confirmed by
// inspection of every MarshalJSON/UnmarshalJSON pair in this file.
func knownJSONKeys[T any]() map[string]bool {
	keys := map[string]bool{}
	t := reflect.TypeFor[T]()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported field (e.g. "extra"): never a JSON key
			continue
		}
		tag, ok := f.Tag.Lookup("json")
		if !ok {
			keys[f.Name] = true
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		keys[name] = true
	}
	return keys
}

// --- Card ------------------------------------------------------------------

// cardWire is Card's wire-JSON shape: Id-keyed collections are maps, boolean
// sets are map[string]bool. codec.go converts to/from Card's ergonomic
// ordered-slice shape (types.go).
type cardWire struct {
	Type     string     `json:"@type"`
	Version  string     `json:"version"`
	UID      string     `json:"uid,omitempty"`
	Kind     string     `json:"kind,omitempty"`
	Language string     `json:"language,omitempty"`
	ProdID   string     `json:"prodId,omitempty"`
	Created  *Timestamp `json:"created,omitempty"`
	Updated  *Timestamp `json:"updated,omitempty"`

	Name *Name `json:"name,omitempty"`

	Nicknames      map[string]Nickname      `json:"nicknames,omitempty"`
	Organizations  map[string]Organization  `json:"organizations,omitempty"`
	Titles         map[string]Title         `json:"titles,omitempty"`
	Emails         map[string]EmailAddress  `json:"emails,omitempty"`
	Phones         map[string]Phone         `json:"phones,omitempty"`
	OnlineServices map[string]OnlineService `json:"onlineServices,omitempty"`
	Addresses      map[string]Address       `json:"addresses,omitempty"`
	Anniversaries  map[string]Anniversary   `json:"anniversaries,omitempty"`
	SpeakToAs      *SpeakToAs               `json:"speakToAs,omitempty"`
	PersonalInfo   map[string]PersonalInfo  `json:"personalInfo,omitempty"`
	Notes          map[string]Note          `json:"notes,omitempty"`
	Keywords       map[string]bool          `json:"keywords,omitempty"`

	Media               map[string]Media             `json:"media,omitempty"`
	Calendars           map[string]Calendar          `json:"calendars,omitempty"`
	SchedulingAddresses map[string]SchedulingAddress `json:"schedulingAddresses,omitempty"`
	CryptoKeys          map[string]CryptoKey         `json:"cryptoKeys,omitempty"`
	Directories         map[string]Directory         `json:"directories,omitempty"`
	Links               map[string]Link              `json:"links,omitempty"`

	PreferredLanguages map[string]LanguagePref `json:"preferredLanguages,omitempty"`
	RelatedTo          map[string]Relation     `json:"relatedTo,omitempty"`
	Members            map[string]bool         `json:"members,omitempty"`

	VCardProps []contactmodel.JCardProp `json:"vCardProps,omitempty"`
}

// MarshalJSON emits Card as its RFC 9553 wire shape (Id-keyed maps rather
// than this package's slices, per cardWire's own doc comment).
func (c Card) MarshalJSON() ([]byte, error) {
	w := cardWire{
		Type:     "Card",
		Version:  "1.0", // codec rule 2: always "1.0" on export
		UID:      c.UID,
		Kind:     c.Kind,
		Language: c.Language,
		ProdID:   c.ProdID,
		Created:  c.Created,
		Updated:  c.Updated,
		Name:     c.Name,

		Nicknames:      idMapFromSlice(c.Nicknames, func(v Nickname) string { return v.ID }),
		Organizations:  idMapFromSlice(c.Organizations, func(v Organization) string { return v.ID }),
		Titles:         idMapFromSlice(c.Titles, func(v Title) string { return v.ID }),
		Emails:         idMapFromSlice(c.Emails, func(v EmailAddress) string { return v.ID }),
		Phones:         idMapFromSlice(c.Phones, func(v Phone) string { return v.ID }),
		OnlineServices: idMapFromSlice(c.OnlineServices, func(v OnlineService) string { return v.ID }),
		Addresses:      idMapFromSlice(c.Addresses, func(v Address) string { return v.ID }),
		Anniversaries:  idMapFromSlice(c.Anniversaries, func(v Anniversary) string { return v.ID }),
		SpeakToAs:      c.SpeakToAs,
		PersonalInfo:   idMapFromSlice(c.PersonalInfo, func(v PersonalInfo) string { return v.ID }),
		Notes:          idMapFromSlice(c.Notes, func(v Note) string { return v.ID }),
		Keywords:       boolSetMarshal(c.Keywords),

		Media:               idMapFromSlice(c.Media, func(v Media) string { return v.ID }),
		Calendars:           idMapFromSlice(c.Calendars, func(v Calendar) string { return v.ID }),
		SchedulingAddresses: idMapFromSlice(c.SchedulingAddresses, func(v SchedulingAddress) string { return v.ID }),
		CryptoKeys:          idMapFromSlice(c.CryptoKeys, func(v CryptoKey) string { return v.ID }),
		Directories:         idMapFromSlice(c.Directories, func(v Directory) string { return v.ID }),
		Links:               idMapFromSlice(c.Links, func(v Link) string { return v.ID }),

		PreferredLanguages: idMapFromSlice(c.PreferredLanguages, func(v LanguagePref) string { return v.ID }),
		RelatedTo:          idMapFromSlice(c.RelatedTo, func(v Relation) string { return v.Target }),
		Members:            boolSetMarshal(c.Members),

		VCardProps: c.VCardProps,
	}
	return json.Marshal(w)
}

// UnmarshalJSON parses the RFC 9553 wire shape (Id-keyed maps) back into
// Card's own slice-based fields.
func (c *Card) UnmarshalJSON(data []byte) error {
	var w cardWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	if err := checkType(&w.Type, "Card"); err != nil {
		return err
	}

	c.Type = w.Type
	c.Version = w.Version // codec rule 2: any value accepted on import
	c.UID = w.UID
	c.Kind = w.Kind
	c.Language = w.Language
	c.ProdID = w.ProdID
	c.Created = w.Created
	c.Updated = w.Updated
	c.Name = w.Name

	c.Nicknames = idMapToSlice(w.Nicknames, func(v *Nickname, k string) { v.ID = k })
	c.Organizations = idMapToSlice(w.Organizations, func(v *Organization, k string) { v.ID = k })
	c.Titles = idMapToSlice(w.Titles, func(v *Title, k string) { v.ID = k })
	c.Emails = idMapToSlice(w.Emails, func(v *EmailAddress, k string) { v.ID = k })
	c.Phones = idMapToSlice(w.Phones, func(v *Phone, k string) { v.ID = k })
	c.OnlineServices = idMapToSlice(w.OnlineServices, func(v *OnlineService, k string) { v.ID = k })
	c.Addresses = idMapToSlice(w.Addresses, func(v *Address, k string) { v.ID = k })
	c.Anniversaries = idMapToSlice(w.Anniversaries, func(v *Anniversary, k string) { v.ID = k })
	c.SpeakToAs = w.SpeakToAs
	c.PersonalInfo = idMapToSlice(w.PersonalInfo, func(v *PersonalInfo, k string) { v.ID = k })
	c.Notes = idMapToSlice(w.Notes, func(v *Note, k string) { v.ID = k })
	c.Keywords = boolSetUnmarshal(w.Keywords)

	c.Media = idMapToSlice(w.Media, func(v *Media, k string) { v.ID = k })
	c.Calendars = idMapToSlice(w.Calendars, func(v *Calendar, k string) { v.ID = k })
	c.SchedulingAddresses = idMapToSlice(w.SchedulingAddresses, func(v *SchedulingAddress, k string) { v.ID = k })
	c.CryptoKeys = idMapToSlice(w.CryptoKeys, func(v *CryptoKey, k string) { v.ID = k })
	c.Directories = idMapToSlice(w.Directories, func(v *Directory, k string) { v.ID = k })
	c.Links = idMapToSlice(w.Links, func(v *Link, k string) { v.ID = k })

	c.PreferredLanguages = idMapToSlice(w.PreferredLanguages, func(v *LanguagePref, k string) { v.ID = k })
	c.RelatedTo = idMapToSlice(w.RelatedTo, func(v *Relation, k string) { v.Target = k })
	c.Members = boolSetUnmarshal(w.Members)

	c.VCardProps = w.VCardProps
	return nil
}

// --- SpeakToAs (nested Id-map: pronouns) -----------------------------------

// MarshalJSON emits SpeakToAs.Pronouns as the RFC 9553 Id-keyed map shape.
func (s SpeakToAs) MarshalJSON() ([]byte, error) {
	type wire struct {
		Type              string              `json:"@type"`
		GrammaticalGender string              `json:"grammaticalGender,omitempty"`
		Pronouns          map[string]Pronouns `json:"pronouns,omitempty"`
	}
	base, err := json.Marshal(wire{
		Type:              "SpeakToAs",
		GrammaticalGender: s.GrammaticalGender,
		Pronouns:          idMapFromSlice(s.Pronouns, func(p Pronouns) string { return p.ID }),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, s.extra)
}

// UnmarshalJSON parses the RFC 9553 Id-keyed pronouns map back into
// SpeakToAs.Pronouns.
func (s *SpeakToAs) UnmarshalJSON(data []byte) error {
	type wire struct {
		Type              string              `json:"@type"`
		GrammaticalGender string              `json:"grammaticalGender,omitempty"`
		Pronouns          map[string]Pronouns `json:"pronouns,omitempty"`
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	if err := checkType(&w.Type, "SpeakToAs"); err != nil {
		return err
	}
	s.Type = w.Type
	s.GrammaticalGender = w.GrammaticalGender
	s.Pronouns = idMapToSlice(w.Pronouns, func(p *Pronouns, k string) { p.ID = k })
	s.extra = captureExtra(data, knownJSONKeys[SpeakToAs]())
	return nil
}

// --- Anniversary (polymorphic date) -----------------------------------------

// MarshalJSON emits Anniversary in its RFC 9553 wire shape.
func (a Anniversary) MarshalJSON() ([]byte, error) {
	type wire struct {
		Type  string          `json:"@type"`
		Kind  string          `json:"kind"`
		Date  AnniversaryDate `json:"date"`
		Place *Address        `json:"place,omitempty"`
	}
	base, err := json.Marshal(wire{Type: "Anniversary", Kind: a.Kind, Date: a.Date, Place: a.Place})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, a.extra)
}

// UnmarshalJSON parses the RFC 9553 wire shape back into Anniversary.
func (a *Anniversary) UnmarshalJSON(data []byte) error {
	type wire struct {
		Type  string          `json:"@type"`
		Kind  string          `json:"kind"`
		Date  AnniversaryDate `json:"date"`
		Place *Address        `json:"place,omitempty"`
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	if err := checkType(&w.Type, "Anniversary"); err != nil {
		return err
	}
	a.Type, a.Kind, a.Date, a.Place = w.Type, w.Kind, w.Date, w.Place
	a.extra = captureExtra(data, knownJSONKeys[Anniversary]())
	return nil
}

// MarshalJSON emits whichever of PartialDate or Timestamp is set (RFC 9553:
// Anniversary.date is PartialDate|Timestamp, defaultType PartialDate).
func (d AnniversaryDate) MarshalJSON() ([]byte, error) {
	if d.Timestamp != nil {
		return json.Marshal(*d.Timestamp)
	}
	if d.PartialDate != nil {
		return json.Marshal(*d.PartialDate)
	}
	return json.Marshal(PartialDate{Type: "PartialDate"})
}

// UnmarshalJSON peeks at @type to decide whether to parse the payload as a
// PartialDate or a Timestamp (see MarshalJSON's doc comment).
func (d *AnniversaryDate) UnmarshalJSON(data []byte) error {
	var peek struct {
		Type string `json:"@type"`
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		return err
	}
	switch peek.Type {
	case "Timestamp":
		var t Timestamp
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
		d.Timestamp = &t
		d.PartialDate = nil
	case "", "PartialDate":
		var p PartialDate
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		d.PartialDate = &p
		d.Timestamp = nil
	default:
		return fmt.Errorf("jscontact: Anniversary.date has unknown @type %q", peek.Type)
	}
	return nil
}

// --- trivial @type-only objects (no boolean-set fields) ---------------------
// Each of these forces/validates its own @type via a plain type-alias cast
// (cheap: no field-shape change needed, so no embed trick required).

// MarshalJSON forces Name's @type to "Name" via a plain type-alias cast
// (see the trivial-@type-only comment above).
func (n Name) MarshalJSON() ([]byte, error) {
	n.Type = "Name"
	type alias Name
	base, err := json.Marshal(alias(n))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, n.extra)
}

func (n *Name) UnmarshalJSON(data []byte) error {
	type alias Name
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "Name"); err != nil {
		return err
	}
	*n = Name(a)
	n.extra = captureExtra(data, knownJSONKeys[Name]())
	return nil
}

func (nc NameComponent) MarshalJSON() ([]byte, error) {
	nc.Type = "NameComponent"
	type alias NameComponent
	base, err := json.Marshal(alias(nc))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, nc.extra)
}

func (nc *NameComponent) UnmarshalJSON(data []byte) error {
	type alias NameComponent
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "NameComponent"); err != nil {
		return err
	}
	*nc = NameComponent(a)
	nc.extra = captureExtra(data, knownJSONKeys[NameComponent]())
	return nil
}

func (ac AddressComponent) MarshalJSON() ([]byte, error) {
	ac.Type = "AddressComponent"
	type alias AddressComponent
	base, err := json.Marshal(alias(ac))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, ac.extra)
}

func (ac *AddressComponent) UnmarshalJSON(data []byte) error {
	type alias AddressComponent
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "AddressComponent"); err != nil {
		return err
	}
	*ac = AddressComponent(a)
	ac.extra = captureExtra(data, knownJSONKeys[AddressComponent]())
	return nil
}

func (o Organization) MarshalJSON() ([]byte, error) {
	o.Type = "Organization"
	type alias Organization
	base, err := json.Marshal(alias(o))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, o.extra)
}

func (o *Organization) UnmarshalJSON(data []byte) error {
	type alias Organization
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "Organization"); err != nil {
		return err
	}
	*o = Organization(a)
	o.extra = captureExtra(data, knownJSONKeys[Organization]())
	return nil
}

func (u OrgUnit) MarshalJSON() ([]byte, error) {
	u.Type = "OrgUnit"
	type alias OrgUnit
	base, err := json.Marshal(alias(u))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, u.extra)
}

func (u *OrgUnit) UnmarshalJSON(data []byte) error {
	type alias OrgUnit
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "OrgUnit"); err != nil {
		return err
	}
	*u = OrgUnit(a)
	u.extra = captureExtra(data, knownJSONKeys[OrgUnit]())
	return nil
}

func (t Title) MarshalJSON() ([]byte, error) {
	t.Type = "Title"
	type alias Title
	base, err := json.Marshal(alias(t))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, t.extra)
}

func (t *Title) UnmarshalJSON(data []byte) error {
	type alias Title
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "Title"); err != nil {
		return err
	}
	*t = Title(a)
	t.extra = captureExtra(data, knownJSONKeys[Title]())
	return nil
}

func (p PersonalInfo) MarshalJSON() ([]byte, error) {
	p.Type = "PersonalInfo"
	type alias PersonalInfo
	base, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, p.extra)
}

func (p *PersonalInfo) UnmarshalJSON(data []byte) error {
	type alias PersonalInfo
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "PersonalInfo"); err != nil {
		return err
	}
	*p = PersonalInfo(a)
	p.extra = captureExtra(data, knownJSONKeys[PersonalInfo]())
	return nil
}

func (n Note) MarshalJSON() ([]byte, error) {
	n.Type = "Note"
	type alias Note
	base, err := json.Marshal(alias(n))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, n.extra)
}

func (n *Note) UnmarshalJSON(data []byte) error {
	type alias Note
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "Note"); err != nil {
		return err
	}
	*n = Note(a)
	n.extra = captureExtra(data, knownJSONKeys[Note]())
	return nil
}

func (au Author) MarshalJSON() ([]byte, error) {
	au.Type = "Author"
	type alias Author
	base, err := json.Marshal(alias(au))
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, au.extra)
}

func (au *Author) UnmarshalJSON(data []byte) error {
	type alias Author
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "Author"); err != nil {
		return err
	}
	*au = Author(a)
	au.extra = captureExtra(data, knownJSONKeys[Author]())
	return nil
}

func (pd PartialDate) MarshalJSON() ([]byte, error) {
	pd.Type = "PartialDate"
	type alias PartialDate
	return json.Marshal(alias(pd))
}

func (pd *PartialDate) UnmarshalJSON(data []byte) error {
	type alias PartialDate
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "PartialDate"); err != nil {
		return err
	}
	*pd = PartialDate(a)
	return nil
}

func (ts Timestamp) MarshalJSON() ([]byte, error) {
	ts.Type = "Timestamp"
	type alias Timestamp
	return json.Marshal(alias(ts))
}

func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	type alias Timestamp
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if err := checkType(&a.Type, "Timestamp"); err != nil {
		return err
	}
	*ts = Timestamp(a)
	return nil
}

// --- objects with a boolean-set field (contexts/features/relation) ---------
// Each uses the "shallow field wins" embedding trick: an anonymous outer
// field with the wire (map[string]bool) shape at depth 0 shadows the
// same-JSON-name promoted field (the neutral []string shape) from the
// embedded type-aliased struct at depth 1, so only the fields listed
// explicitly need restating.

func (e Address) MarshalJSON() ([]byte, error) {
	type Alias Address
	e.Type = "Address"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(e.Contexts),
		Alias:    Alias(e),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, e.extra)
}

func (e *Address) UnmarshalJSON(data []byte) error {
	type Alias Address
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&e.Type, "Address"); err != nil {
		return err
	}
	e.Contexts = boolSetUnmarshal(aux.Contexts)
	e.extra = captureExtra(data, knownJSONKeys[Address]())
	return nil
}

func (p Phone) MarshalJSON() ([]byte, error) {
	type Alias Phone
	p.Type = "Phone"
	base, err := json.Marshal(struct {
		Features map[string]bool `json:"features,omitempty"`
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Features: boolSetMarshal(p.Features),
		Contexts: boolSetMarshal(p.Contexts),
		Alias:    Alias(p),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, p.extra)
}

func (p *Phone) UnmarshalJSON(data []byte) error {
	type Alias Phone
	aux := struct {
		Features map[string]bool `json:"features,omitempty"`
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(p)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&p.Type, "Phone"); err != nil {
		return err
	}
	p.Features = boolSetUnmarshal(aux.Features)
	p.Contexts = boolSetUnmarshal(aux.Contexts)
	p.extra = captureExtra(data, knownJSONKeys[Phone]())
	return nil
}

func (e EmailAddress) MarshalJSON() ([]byte, error) {
	type Alias EmailAddress
	e.Type = "EmailAddress"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(e.Contexts),
		Alias:    Alias(e),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, e.extra)
}

func (e *EmailAddress) UnmarshalJSON(data []byte) error {
	type Alias EmailAddress
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&e.Type, "EmailAddress"); err != nil {
		return err
	}
	e.Contexts = boolSetUnmarshal(aux.Contexts)
	e.extra = captureExtra(data, knownJSONKeys[EmailAddress]())
	return nil
}

func (o OnlineService) MarshalJSON() ([]byte, error) {
	type Alias OnlineService
	o.Type = "OnlineService"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(o.Contexts),
		Alias:    Alias(o),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, o.extra)
}

func (o *OnlineService) UnmarshalJSON(data []byte) error {
	type Alias OnlineService
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(o)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&o.Type, "OnlineService"); err != nil {
		return err
	}
	o.Contexts = boolSetUnmarshal(aux.Contexts)
	o.extra = captureExtra(data, knownJSONKeys[OnlineService]())
	return nil
}

func (m Calendar) MarshalJSON() ([]byte, error) {
	type Alias Calendar
	m.Type = "Calendar"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(m.Contexts),
		Alias:    Alias(m),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, m.extra)
}

func (m *Calendar) UnmarshalJSON(data []byte) error {
	type Alias Calendar
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(m)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&m.Type, "Calendar"); err != nil {
		return err
	}
	m.Contexts = boolSetUnmarshal(aux.Contexts)
	m.extra = captureExtra(data, knownJSONKeys[Calendar]())
	return nil
}

func (s SchedulingAddress) MarshalJSON() ([]byte, error) {
	type Alias SchedulingAddress
	s.Type = "SchedulingAddress"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(s.Contexts),
		Alias:    Alias(s),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, s.extra)
}

func (s *SchedulingAddress) UnmarshalJSON(data []byte) error {
	type Alias SchedulingAddress
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(s)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&s.Type, "SchedulingAddress"); err != nil {
		return err
	}
	s.Contexts = boolSetUnmarshal(aux.Contexts)
	s.extra = captureExtra(data, knownJSONKeys[SchedulingAddress]())
	return nil
}

func (k CryptoKey) MarshalJSON() ([]byte, error) {
	type Alias CryptoKey
	k.Type = "CryptoKey"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(k.Contexts),
		Alias:    Alias(k),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, k.extra)
}

func (k *CryptoKey) UnmarshalJSON(data []byte) error {
	type Alias CryptoKey
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(k)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&k.Type, "CryptoKey"); err != nil {
		return err
	}
	k.Contexts = boolSetUnmarshal(aux.Contexts)
	k.extra = captureExtra(data, knownJSONKeys[CryptoKey]())
	return nil
}

func (d Directory) MarshalJSON() ([]byte, error) {
	type Alias Directory
	d.Type = "Directory"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(d.Contexts),
		Alias:    Alias(d),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, d.extra)
}

func (d *Directory) UnmarshalJSON(data []byte) error {
	type Alias Directory
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(d)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&d.Type, "Directory"); err != nil {
		return err
	}
	d.Contexts = boolSetUnmarshal(aux.Contexts)
	d.extra = captureExtra(data, knownJSONKeys[Directory]())
	return nil
}

func (l Link) MarshalJSON() ([]byte, error) {
	type Alias Link
	l.Type = "Link"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(l.Contexts),
		Alias:    Alias(l),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, l.extra)
}

func (l *Link) UnmarshalJSON(data []byte) error {
	type Alias Link
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(l)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&l.Type, "Link"); err != nil {
		return err
	}
	l.Contexts = boolSetUnmarshal(aux.Contexts)
	l.extra = captureExtra(data, knownJSONKeys[Link]())
	return nil
}

func (m Media) MarshalJSON() ([]byte, error) {
	type Alias Media
	m.Type = "Media"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(m.Contexts),
		Alias:    Alias(m),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, m.extra)
}

func (m *Media) UnmarshalJSON(data []byte) error {
	type Alias Media
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(m)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&m.Type, "Media"); err != nil {
		return err
	}
	m.Contexts = boolSetUnmarshal(aux.Contexts)
	m.extra = captureExtra(data, knownJSONKeys[Media]())
	return nil
}

func (n Nickname) MarshalJSON() ([]byte, error) {
	type Alias Nickname
	n.Type = "Nickname"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(n.Contexts),
		Alias:    Alias(n),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, n.extra)
}

func (n *Nickname) UnmarshalJSON(data []byte) error {
	type Alias Nickname
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(n)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&n.Type, "Nickname"); err != nil {
		return err
	}
	n.Contexts = boolSetUnmarshal(aux.Contexts)
	n.extra = captureExtra(data, knownJSONKeys[Nickname]())
	return nil
}

func (l LanguagePref) MarshalJSON() ([]byte, error) {
	type Alias LanguagePref
	l.Type = "LanguagePref"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(l.Contexts),
		Alias:    Alias(l),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, l.extra)
}

func (l *LanguagePref) UnmarshalJSON(data []byte) error {
	type Alias LanguagePref
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(l)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&l.Type, "LanguagePref"); err != nil {
		return err
	}
	l.Contexts = boolSetUnmarshal(aux.Contexts)
	l.extra = captureExtra(data, knownJSONKeys[LanguagePref]())
	return nil
}

func (p Pronouns) MarshalJSON() ([]byte, error) {
	type Alias Pronouns
	p.Type = "Pronouns"
	base, err := json.Marshal(struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		Alias
	}{
		Contexts: boolSetMarshal(p.Contexts),
		Alias:    Alias(p),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, p.extra)
}

func (p *Pronouns) UnmarshalJSON(data []byte) error {
	type Alias Pronouns
	aux := struct {
		Contexts map[string]bool `json:"contexts,omitempty"`
		*Alias
	}{Alias: (*Alias)(p)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&p.Type, "Pronouns"); err != nil {
		return err
	}
	p.Contexts = boolSetUnmarshal(aux.Contexts)
	p.extra = captureExtra(data, knownJSONKeys[Pronouns]())
	return nil
}

func (r Relation) MarshalJSON() ([]byte, error) {
	type Alias Relation
	r.Type = "Relation"
	base, err := json.Marshal(struct {
		RelationSet map[string]bool `json:"relation,omitempty"`
		Alias
	}{
		RelationSet: boolSetMarshal(r.Relation),
		Alias:       Alias(r),
	})
	if err != nil {
		return nil, err
	}
	return mergeExtraJSON(base, r.extra)
}

func (r *Relation) UnmarshalJSON(data []byte) error {
	type Alias Relation
	aux := struct {
		RelationSet map[string]bool `json:"relation,omitempty"`
		*Alias
	}{Alias: (*Alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if err := checkType(&r.Type, "Relation"); err != nil {
		return err
	}
	r.Relation = boolSetUnmarshal(aux.RelationSet)
	r.extra = captureExtra(data, knownJSONKeys[Relation]())
	return nil
}
