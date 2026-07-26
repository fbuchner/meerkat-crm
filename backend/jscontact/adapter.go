// This file (WP-30b) implements the JSContact <-> neutral-model adapter:
// contactmodel.Importer/Exporter for RFC 9553 JSContact. Mapping decisions
// here are governed entirely by docs/fork-plan/20-correspondence.md (the
// oracle); see that file's rows for the concept_id backing every field
// touched below. Because JSContact is, per docs/fork-plan/30-adapters.md
// §30.A, "the near-identity spoke" (contactmodel's Card/Name/Address/... were
// shaped directly after it), nearly every mapping here is a straight field
// copy between the two (structurally near-identical, separately owned) type
// sets rather than a format-specific transform.
package jscontact

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"meerkat/contactmodel"
)

// Adapter implements contactmodel.Importer and contactmodel.Exporter for the
// JSContact wire format.
type Adapter struct{}

var _ contactmodel.Importer = Adapter{}
var _ contactmodel.Exporter = Adapter{}

// knownCardTopLevelKeys is the set of top-level JSON keys this adapter
// actively maps to/from the neutral model. Anything else found at the top
// level of an incoming document is, per 20.3's "pt.jscontact" row, unmapped
// JSContact data that must be preserved via Record.Passthrough.JSContact
// rather than silently dropped (0.5's degradation policy).
//
// This set has to be maintained by hand here (rather than derived via
// reflection over cardWire, which is unexported in codec.go) because Card
// itself does NOT use codec.go's `extra`-field capture mechanism (see that
// file's "nested-unknown-property capture" section for why: Card's
// top-level unknowns are handled here instead, by this hand-maintained set,
// which predates and remains separate from that mechanism). See the
// adapter-level Import/Export passthrough helpers below for how this file
// recovers top-level unknowns, and importNestedPassthrough/
// collectNestedPassthrough further below for how *nested* unknowns (e.g. an
// extra key inside one emails{} entry) are recovered via the `extra`-field
// mechanism every other wire type in this package does carry.
var knownCardTopLevelKeys = map[string]bool{
	"@type":               true,
	"version":             true,
	"uid":                 true,
	"kind":                true,
	"language":            true,
	"prodId":              true,
	"created":             true,
	"updated":             true,
	"name":                true,
	"nicknames":           true,
	"organizations":       true,
	"titles":              true,
	"emails":              true,
	"phones":              true,
	"onlineServices":      true,
	"addresses":           true,
	"anniversaries":       true,
	"speakToAs":           true,
	"personalInfo":        true,
	"notes":               true,
	"keywords":            true,
	"media":               true,
	"calendars":           true,
	"schedulingAddresses": true,
	"cryptoKeys":          true,
	"directories":         true,
	"links":               true,
	"preferredLanguages":  true,
	"relatedTo":           true,
	"members":             true,
	"vCardProps":          true,
}

// Import parses raw RFC 9553 JSON bytes into the neutral model. It never
// returns an error for unmappable/unknown data (0.5) — only when raw is not
// valid JSContact JSON at all (Unmarshal itself fails).
func (Adapter) Import(raw []byte) (*contactmodel.Record, []contactmodel.Diagnostic, error) {
	card, err := Unmarshal(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("jscontact: import: %w", err)
	}

	var diags []contactmodel.Diagnostic
	rec := &contactmodel.Record{}

	importIdentity(card, rec)
	importName(card, rec)
	importNicknames(card, rec)
	importOrganizationsTitles(card, rec)
	importEmailsPhonesOnline(card, rec)
	importAddresses(card, rec)
	importAnniversaries(card, rec)
	importSpeakToAs(card, rec)
	importPersonalInfo(card, rec)
	importNotesKeywords(card, rec)
	importResources(card, rec)
	importLangsRelatedMembers(card, rec)
	importPassthroughVCard(card, rec)

	if err := importUnknownTopLevel(raw, rec); err != nil {
		// raw already parsed successfully by Unmarshal above, so a generic
		// re-parse into map[string]json.RawMessage cannot fail; guarded
		// defensively rather than assumed.
		return nil, nil, fmt.Errorf("jscontact: import: %w", err)
	}
	importNestedPassthrough(card, rec)

	return rec, diags, nil
}

// Export renders the neutral model into RFC 9553 JSON bytes. Same rule as
// Import: never a hard error for a neutral field with no JSContact home
// (0.5) — for this adapter that case essentially does not arise, since
// JSContact is the near-identity superset-adjacent spoke.
func (Adapter) Export(r *contactmodel.Record) ([]byte, []contactmodel.Diagnostic, error) {
	if r == nil {
		r = &contactmodel.Record{}
	}
	var diags []contactmodel.Diagnostic
	card := &Card{}

	exportIdentity(r, card)
	exportName(r, card)
	exportNicknames(r, card)
	exportOrganizationsTitles(r, card)
	exportEmailsPhonesOnline(r, card)
	exportAddresses(r, card)
	exportAnniversaries(r, card)
	exportSpeakToAs(r, card)
	exportPersonalInfo(r, card)
	exportNotesKeywords(r, card)
	exportResources(r, card)
	exportLangsRelatedMembers(r, card)
	exportPassthroughVCard(r, card)

	raw, err := Marshal(card)
	if err != nil {
		return nil, diags, fmt.Errorf("jscontact: export: %w", err)
	}

	raw, err = spliceJSContactPassthrough(raw, r.Passthrough.JSContact)
	if err != nil {
		return nil, diags, fmt.Errorf("jscontact: export: %w", err)
	}

	return raw, diags, nil
}

// --- small copy helpers (avoid aliasing slices/pointers across the two type sets) ---

func copyStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func copyIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func copyBoolPtr(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// --- identity / meta: uid, kind, prodid, created, updated, language --------
//
// NOTE on "created": docs/fork-plan/20-correspondence.md's "created" row was
// corrected by a later audit (it was wrongly "-"; RFC 9553 §2.1.3 defines
// Card.created as a real, optional Card-level UTCDateTime) and now reads
// js_ptr "/created". This adapter maps Card.Created <-> contactmodel's
// Card.Created exactly like Updated, immediately below.

func importIdentity(c *Card, r *contactmodel.Record) {
	r.Card.UID = c.UID
	r.Card.Kind = c.Kind
	r.Card.ProdID = c.ProdID
	r.Card.Language = c.Language
	if c.Created != nil {
		r.Card.Created = &contactmodel.Timestamp{UTC: c.Created.UTC}
	}
	if c.Updated != nil {
		r.Card.Updated = &contactmodel.Timestamp{UTC: c.Updated.UTC}
	}
}

func exportIdentity(r *contactmodel.Record, c *Card) {
	c.UID = r.Card.UID
	c.Kind = r.Card.Kind
	c.ProdID = r.Card.ProdID
	c.Language = r.Card.Language
	if r.Card.Created != nil {
		c.Created = &Timestamp{UTC: r.Card.Created.UTC}
	}
	if r.Card.Updated != nil {
		c.Updated = &Timestamp{UTC: r.Card.Updated.UTC}
	}
}

// --- name --------------------------------------------------------------

func importName(c *Card, r *contactmodel.Record) {
	if c.Name == nil {
		return
	}
	n := &contactmodel.Name{
		Full:             c.Name.Full,
		IsOrdered:        copyBoolPtr(c.Name.IsOrdered),
		DefaultSeparator: c.Name.DefaultSeparator,
		PhoneticSystem:   c.Name.PhoneticSystem,
		PhoneticScript:   c.Name.PhoneticScript,
	}
	if len(c.Name.SortAs) > 0 {
		n.SortAs = make(map[string]string, len(c.Name.SortAs))
		for k, v := range c.Name.SortAs {
			n.SortAs[k] = v
		}
	}
	for _, comp := range c.Name.Components {
		n.Components = append(n.Components, contactmodel.NameComponent{
			Kind: comp.Kind, Value: comp.Value, Phonetic: comp.Phonetic,
		})
	}
	r.Card.Name = n
}

func exportName(r *contactmodel.Record, c *Card) {
	n := r.Card.Name
	if n == nil {
		return
	}
	jn := &Name{
		Full:             n.Full,
		IsOrdered:        copyBoolPtr(n.IsOrdered),
		DefaultSeparator: n.DefaultSeparator,
		PhoneticSystem:   n.PhoneticSystem,
		PhoneticScript:   n.PhoneticScript,
	}
	if len(n.SortAs) > 0 {
		jn.SortAs = make(map[string]string, len(n.SortAs))
		for k, v := range n.SortAs {
			jn.SortAs[k] = v
		}
	}
	for _, comp := range n.Components {
		jn.Components = append(jn.Components, NameComponent{
			Kind: comp.Kind, Value: comp.Value, Phonetic: comp.Phonetic,
		})
	}
	c.Name = jn
}

// --- nicknames -----------------------------------------------------------

func importNicknames(c *Card, r *contactmodel.Record) {
	for _, n := range c.Nicknames {
		r.Card.Nicknames = append(r.Card.Nicknames, contactmodel.Nickname{
			ID: n.ID, Name: n.Name, Contexts: copyStrings(n.Contexts), Pref: copyIntPtr(n.Pref),
		})
	}
}

func exportNicknames(r *contactmodel.Record, c *Card) {
	for _, n := range r.Card.Nicknames {
		c.Nicknames = append(c.Nicknames, Nickname{
			ID: n.ID, Name: n.Name, Contexts: copyStrings(n.Contexts), Pref: copyIntPtr(n.Pref),
		})
	}
}

// --- organizations / titles ------------------------------------------------

func importOrganizationsTitles(c *Card, r *contactmodel.Record) {
	for _, o := range c.Organizations {
		no := contactmodel.Organization{ID: o.ID, Name: o.Name, SortAs: o.SortAs}
		for _, u := range o.Units {
			no.Units = append(no.Units, contactmodel.OrgUnit{Name: u.Name, SortAs: u.SortAs})
		}
		r.Card.Organizations = append(r.Card.Organizations, no)
	}
	for _, t := range c.Titles {
		r.Card.Titles = append(r.Card.Titles, contactmodel.Title{
			ID: t.ID, Name: t.Name, Kind: t.Kind, OrganizationID: t.OrganizationID,
		})
	}
}

func exportOrganizationsTitles(r *contactmodel.Record, c *Card) {
	for _, o := range r.Card.Organizations {
		jo := Organization{ID: o.ID, Name: o.Name, SortAs: o.SortAs}
		for _, u := range o.Units {
			jo.Units = append(jo.Units, OrgUnit{Name: u.Name, SortAs: u.SortAs})
		}
		c.Organizations = append(c.Organizations, jo)
	}
	for _, t := range r.Card.Titles {
		c.Titles = append(c.Titles, Title{
			ID: t.ID, Name: t.Name, Kind: t.Kind, OrganizationID: t.OrganizationID,
		})
	}
}

// --- emails / phones / online services --------------------------------------
//
// onlineServices three-way split (20-correspondence.md §20.7): JSContact's
// onlineServices has no kind-style discriminator of its own, so the RFC 9555
// §2.15.3 `vCardName` escape hatch is the only signal available at import
// time. A "IMPP" hint (case-insensitively, per the RFC's own comparison rule
// — see types.go's OnlineService.VCardName doc) routes into
// Card.ImppAddresses; "SOCIALPROFILE" routes into Card.SocialProfiles;
// anything else (including an absent hint) routes into
// Card.OtherOnlineServices — never a presence-based heuristic (e.g.
// Service/User being set) substituting for the actual hint. On export the
// three neutral arrays merge back into one wire onlineServices collection,
// re-tagging ImppAddresses/SocialProfiles entries with the RFC's own
// lowercase spelling ("impp"/"socialprofile", per Figure 17/Figure 47) and
// leaving OtherOnlineServices entries untagged (their origin is genuinely
// unknown; fabricating a tag would assert something we don't know).

const (
	vCardNameIMPP          = "impp"
	vCardNameSocialProfile = "socialprofile"
)

func importEmailsPhonesOnline(c *Card, r *contactmodel.Record) {
	for _, e := range c.Emails {
		r.Card.Emails = append(r.Card.Emails, contactmodel.Email{
			ID: e.ID, Address: e.Address, Contexts: copyStrings(e.Contexts), Pref: copyIntPtr(e.Pref), Label: e.Label,
		})
	}
	for _, p := range c.Phones {
		r.Card.Phones = append(r.Card.Phones, contactmodel.Phone{
			ID: p.ID, Number: p.Number, Features: copyStrings(p.Features), Contexts: copyStrings(p.Contexts), Pref: copyIntPtr(p.Pref), Label: p.Label,
		})
	}
	for _, o := range c.OnlineServices {
		no := contactmodel.OnlineService{
			ID: o.ID, Service: o.Service, URI: o.URI, User: o.User, Contexts: copyStrings(o.Contexts), Pref: copyIntPtr(o.Pref), Label: o.Label,
		}
		switch strings.ToLower(o.VCardName) {
		case vCardNameIMPP:
			r.Card.ImppAddresses = append(r.Card.ImppAddresses, no)
		case vCardNameSocialProfile:
			r.Card.SocialProfiles = append(r.Card.SocialProfiles, no)
		default:
			r.Card.OtherOnlineServices = append(r.Card.OtherOnlineServices, no)
		}
	}
}

func exportEmailsPhonesOnline(r *contactmodel.Record, c *Card) {
	for _, e := range r.Card.Emails {
		c.Emails = append(c.Emails, EmailAddress{
			ID: e.ID, Address: e.Address, Contexts: copyStrings(e.Contexts), Pref: copyIntPtr(e.Pref), Label: e.Label,
		})
	}
	for _, p := range r.Card.Phones {
		c.Phones = append(c.Phones, Phone{
			ID: p.ID, Number: p.Number, Features: copyStrings(p.Features), Contexts: copyStrings(p.Contexts), Pref: copyIntPtr(p.Pref), Label: p.Label,
		})
	}
	for _, o := range r.Card.ImppAddresses {
		c.OnlineServices = append(c.OnlineServices, OnlineService{
			ID: o.ID, Service: o.Service, URI: o.URI, User: o.User, Contexts: copyStrings(o.Contexts), Pref: copyIntPtr(o.Pref), Label: o.Label,
			VCardName: vCardNameIMPP,
		})
	}
	for _, o := range r.Card.SocialProfiles {
		c.OnlineServices = append(c.OnlineServices, OnlineService{
			ID: o.ID, Service: o.Service, URI: o.URI, User: o.User, Contexts: copyStrings(o.Contexts), Pref: copyIntPtr(o.Pref), Label: o.Label,
			VCardName: vCardNameSocialProfile,
		})
	}
	for _, o := range r.Card.OtherOnlineServices {
		c.OnlineServices = append(c.OnlineServices, OnlineService{
			ID: o.ID, Service: o.Service, URI: o.URI, User: o.User, Contexts: copyStrings(o.Contexts), Pref: copyIntPtr(o.Pref), Label: o.Label,
		})
	}
}

// --- addresses -----------------------------------------------------------
//
// The "adr" row's neutral_path (Card.Addresses[]) and js_ptr (/addresses/{id})
// both name the *whole* Address element, not a single scalar (unlike most
// other rows) — the same "anchor the whole element" convention the table
// itself uses for "org" (Name + Units[]) and "social" (Service + User). So
// this maps the complete Address object (all its structural sibling fields:
// IsOrdered/DefaultSeparator/PhoneticScript/PhoneticSystem included, none of
// which have their own concept_id) in one pass; adr.geo/adr.tz are called out
// by the table only because vCard4/vCard3 must split Coordinates/TimeZone
// into separate GEO/TZ constructs — JSContact already carries them as plain
// Address fields, so there is nothing extra to do for those two rows here
// beyond what this whole-element copy already does.

func addressToNeutral(a Address) contactmodel.Address {
	na := contactmodel.Address{
		ID:               a.ID,
		CountryCode:      a.CountryCode,
		Coordinates:      a.Coordinates,
		TimeZone:         a.TimeZone,
		Contexts:         copyStrings(a.Contexts),
		Pref:             copyIntPtr(a.Pref),
		Full:             a.Full,
		IsOrdered:        copyBoolPtr(a.IsOrdered),
		DefaultSeparator: a.DefaultSeparator,
		PhoneticSystem:   a.PhoneticSystem,
		PhoneticScript:   a.PhoneticScript,
	}
	for _, comp := range a.Components {
		na.Components = append(na.Components, contactmodel.AddressComponent{
			Kind: comp.Kind, Value: comp.Value, Phonetic: comp.Phonetic,
		})
	}
	return na
}

func addressFromNeutral(a contactmodel.Address) Address {
	ja := Address{
		ID:               a.ID,
		CountryCode:      a.CountryCode,
		Coordinates:      a.Coordinates,
		TimeZone:         a.TimeZone,
		Contexts:         copyStrings(a.Contexts),
		Pref:             copyIntPtr(a.Pref),
		Full:             a.Full,
		IsOrdered:        copyBoolPtr(a.IsOrdered),
		DefaultSeparator: a.DefaultSeparator,
		PhoneticSystem:   a.PhoneticSystem,
		PhoneticScript:   a.PhoneticScript,
	}
	for _, comp := range a.Components {
		ja.Components = append(ja.Components, AddressComponent{
			Kind: comp.Kind, Value: comp.Value, Phonetic: comp.Phonetic,
		})
	}
	return ja
}

func importAddresses(c *Card, r *contactmodel.Record) {
	for _, a := range c.Addresses {
		r.Card.Addresses = append(r.Card.Addresses, addressToNeutral(a))
	}
}

func exportAddresses(r *contactmodel.Record, c *Card) {
	for _, a := range r.Card.Addresses {
		c.Addresses = append(c.Addresses, addressFromNeutral(a))
	}
}

// --- anniversaries -----------------------------------------------------

func timestampToNeutral(t *Timestamp) *contactmodel.Timestamp {
	if t == nil {
		return nil
	}
	return &contactmodel.Timestamp{UTC: t.UTC}
}

func timestampFromNeutral(t *contactmodel.Timestamp) *Timestamp {
	if t == nil {
		return nil
	}
	return &Timestamp{UTC: t.UTC}
}

func anniversaryDateToNeutral(d AnniversaryDate) contactmodel.AnniversaryDate {
	var out contactmodel.AnniversaryDate
	if d.Timestamp != nil {
		utc := d.Timestamp.UTC
		out.Timestamp = &utc
	}
	if d.PartialDate != nil {
		out.Partial = &contactmodel.PartialDate{
			Year:          copyIntPtr(d.PartialDate.Year),
			Month:         copyIntPtr(d.PartialDate.Month),
			Day:           copyIntPtr(d.PartialDate.Day),
			CalendarScale: d.PartialDate.CalendarScale,
		}
	}
	return out
}

func anniversaryDateFromNeutral(d contactmodel.AnniversaryDate) AnniversaryDate {
	var out AnniversaryDate
	if d.Timestamp != nil {
		out.Timestamp = &Timestamp{UTC: *d.Timestamp}
	}
	if d.Partial != nil {
		out.PartialDate = &PartialDate{
			Year:          copyIntPtr(d.Partial.Year),
			Month:         copyIntPtr(d.Partial.Month),
			Day:           copyIntPtr(d.Partial.Day),
			CalendarScale: d.Partial.CalendarScale,
		}
	}
	return out
}

func importAnniversaries(c *Card, r *contactmodel.Record) {
	for _, a := range c.Anniversaries {
		na := contactmodel.Anniversary{ID: a.ID, Kind: a.Kind, Date: anniversaryDateToNeutral(a.Date)}
		if a.Place != nil {
			p := addressToNeutral(*a.Place)
			na.Place = &p
		}
		r.Card.Anniversaries = append(r.Card.Anniversaries, na)
	}
}

func exportAnniversaries(r *contactmodel.Record, c *Card) {
	for _, a := range r.Card.Anniversaries {
		ja := Anniversary{ID: a.ID, Kind: a.Kind, Date: anniversaryDateFromNeutral(a.Date)}
		if a.Place != nil {
			p := addressFromNeutral(*a.Place)
			ja.Place = &p
		}
		c.Anniversaries = append(c.Anniversaries, ja)
	}
}

// --- speakToAs -------------------------------------------------------------
//
// gramgender (20-correspondence.md's "gramgender" row): JSContact's own
// speakToAs.grammaticalGender is a genuine RFC 9553 §2.2.4 scalar (this wire
// shape does not change), but the neutral model's SpeakToAs.GrammaticalGenders
// is a slice (RFC 9554 §3.2 cardinality "*", one per LANGUAGE). JSContact
// import never has more than one value and never a language tag for it, so
// it becomes a single-element (or nil) slice. JSContact export can only hold
// one value, so it picks: the entry whose Language matches Card.Language, if
// set and present; else the first entry; else nothing. This collapse is an
// expected, RFC-inherent lossy export (JSContact structurally cannot hold
// more than one) — not a defect, no diagnostic needed.

func importSpeakToAs(c *Card, r *contactmodel.Record) {
	if c.SpeakToAs == nil {
		return
	}
	s := &contactmodel.SpeakToAs{}
	if c.SpeakToAs.GrammaticalGender != "" {
		s.GrammaticalGenders = []contactmodel.GrammaticalGender{
			{Value: c.SpeakToAs.GrammaticalGender},
		}
	}
	for _, p := range c.SpeakToAs.Pronouns {
		s.Pronouns = append(s.Pronouns, contactmodel.Pronouns{
			ID: p.ID, Pronouns: p.Pronouns, Contexts: copyStrings(p.Contexts), Pref: copyIntPtr(p.Pref),
		})
	}
	r.Card.SpeakToAs = s
}

func exportSpeakToAs(r *contactmodel.Record, c *Card) {
	s := r.Card.SpeakToAs
	if s == nil {
		return
	}
	js := &SpeakToAs{GrammaticalGender: selectGrammaticalGender(r.Card.Language, s.GrammaticalGenders)}
	for _, p := range s.Pronouns {
		js.Pronouns = append(js.Pronouns, Pronouns{
			ID: p.ID, Pronouns: p.Pronouns, Contexts: copyStrings(p.Contexts), Pref: copyIntPtr(p.Pref),
		})
	}
	c.SpeakToAs = js
}

// selectGrammaticalGender implements the export-selection rule from
// 20-correspondence.md's "gramgender" row: if cardLanguage is set and some
// entry's Language matches it, use that entry's Value; otherwise use the
// first entry's Value, if any; otherwise "".
func selectGrammaticalGender(cardLanguage string, genders []contactmodel.GrammaticalGender) string {
	if len(genders) == 0 {
		return ""
	}
	if cardLanguage != "" {
		for _, g := range genders {
			if g.Language == cardLanguage {
				return g.Value
			}
		}
	}
	return genders[0].Value
}

// --- personal info -----------------------------------------------------

func importPersonalInfo(c *Card, r *contactmodel.Record) {
	for _, p := range c.PersonalInfo {
		r.Card.PersonalInfo = append(r.Card.PersonalInfo, contactmodel.PersonalInfo{
			ID: p.ID, Kind: p.Kind, Value: p.Value, Level: p.Level, ListAs: copyIntPtr(p.ListAs), Label: p.Label,
		})
	}
}

func exportPersonalInfo(r *contactmodel.Record, c *Card) {
	for _, p := range r.Card.PersonalInfo {
		c.PersonalInfo = append(c.PersonalInfo, PersonalInfo{
			ID: p.ID, Kind: p.Kind, Value: p.Value, Level: p.Level, ListAs: copyIntPtr(p.ListAs), Label: p.Label,
		})
	}
}

// --- notes / keywords -----------------------------------------------------

func importNotesKeywords(c *Card, r *contactmodel.Record) {
	for _, n := range c.Notes {
		nn := contactmodel.Note{ID: n.ID, Note: n.Note, Created: timestampToNeutral(n.Created)}
		if n.Author != nil {
			nn.Author = &contactmodel.Author{Name: n.Author.Name, URI: n.Author.URI}
		}
		r.Card.Notes = append(r.Card.Notes, nn)
	}
	r.Card.Keywords = copyStrings(c.Keywords)
}

func exportNotesKeywords(r *contactmodel.Record, c *Card) {
	for _, n := range r.Card.Notes {
		jn := Note{ID: n.ID, Note: n.Note, Created: timestampFromNeutral(n.Created)}
		if n.Author != nil {
			jn.Author = &Author{Name: n.Author.Name, URI: n.Author.URI}
		}
		c.Notes = append(c.Notes, jn)
	}
	c.Keywords = copyStrings(r.Card.Keywords)
}

// --- resources: media/calendars/schedulingAddresses/cryptoKeys/directories/links ---
//
// photo/logo/sound share Card.Media (disambiguated by .Kind) and
// directory/source share Card.Directories — those two stay Kind-shared per
// JSContact's own object model, unchanged.
//
// calendars and links, however, now split into discrete neutral fields
// (20-correspondence.md's "calendar"/"freebusy" and "link"/"contacturi"
// rows): unlike onlineServices, the wire Calendar/Link types already carry
// their own `Kind` field, so routing is unambiguous straight off that field
// — no escape-hatch hint needed. Calendar.Kind=="freeBusy" -> FreeBusyURLs,
// anything else (including absent/"calendar") -> Calendars. Link.Kind=="contact"
// -> ContactURIs, anything else (including absent) -> Links. On export the
// two split neutral fields merge back into the one wire calendars/links
// collection, setting Kind accordingly.

func importResources(c *Card, r *contactmodel.Record) {
	for _, m := range c.Media {
		r.Card.Media = append(r.Card.Media, contactmodel.Resource{
			ID: m.ID, Kind: m.Kind, URI: m.URI, MediaType: m.MediaType, Label: m.Label,
			Contexts: copyStrings(m.Contexts), Pref: copyIntPtr(m.Pref),
		})
	}
	for _, cal := range c.Calendars {
		res := contactmodel.Resource{
			ID: cal.ID, URI: cal.URI, MediaType: cal.MediaType, Label: cal.Label,
			Contexts: copyStrings(cal.Contexts), Pref: copyIntPtr(cal.Pref),
		}
		if cal.Kind == "freeBusy" {
			r.Card.FreeBusyURLs = append(r.Card.FreeBusyURLs, res)
		} else {
			r.Card.Calendars = append(r.Card.Calendars, res)
		}
	}
	for _, s := range c.SchedulingAddresses {
		r.Card.SchedulingAddresses = append(r.Card.SchedulingAddresses, contactmodel.Resource{
			ID: s.ID, URI: s.URI, Label: s.Label, Contexts: copyStrings(s.Contexts), Pref: copyIntPtr(s.Pref),
		})
	}
	for _, k := range c.CryptoKeys {
		r.Card.CryptoKeys = append(r.Card.CryptoKeys, contactmodel.Resource{
			ID: k.ID, URI: k.URI, MediaType: k.MediaType, Label: k.Label,
			Contexts: copyStrings(k.Contexts), Pref: copyIntPtr(k.Pref),
		})
	}
	for _, d := range c.Directories {
		r.Card.Directories = append(r.Card.Directories, contactmodel.Resource{
			ID: d.ID, Kind: d.Kind, URI: d.URI, MediaType: d.MediaType, Label: d.Label,
			Contexts: copyStrings(d.Contexts), Pref: copyIntPtr(d.Pref), ListAs: copyIntPtr(d.ListAs),
		})
	}
	for _, l := range c.Links {
		res := contactmodel.Resource{
			ID: l.ID, URI: l.URI, MediaType: l.MediaType, Label: l.Label,
			Contexts: copyStrings(l.Contexts), Pref: copyIntPtr(l.Pref),
		}
		if l.Kind == "contact" {
			r.Card.ContactURIs = append(r.Card.ContactURIs, res)
		} else {
			r.Card.Links = append(r.Card.Links, res)
		}
	}
}

func exportResources(r *contactmodel.Record, c *Card) {
	for _, m := range r.Card.Media {
		c.Media = append(c.Media, Media{
			ID: m.ID, Kind: m.Kind, URI: m.URI, MediaType: m.MediaType, Label: m.Label,
			Contexts: copyStrings(m.Contexts), Pref: copyIntPtr(m.Pref),
		})
	}
	for _, cal := range r.Card.Calendars {
		c.Calendars = append(c.Calendars, Calendar{
			ID: cal.ID, URI: cal.URI, MediaType: cal.MediaType, Label: cal.Label,
			Contexts: copyStrings(cal.Contexts), Pref: copyIntPtr(cal.Pref),
		})
	}
	for _, cal := range r.Card.FreeBusyURLs {
		c.Calendars = append(c.Calendars, Calendar{
			ID: cal.ID, Kind: "freeBusy", URI: cal.URI, MediaType: cal.MediaType, Label: cal.Label,
			Contexts: copyStrings(cal.Contexts), Pref: copyIntPtr(cal.Pref),
		})
	}
	for _, s := range r.Card.SchedulingAddresses {
		c.SchedulingAddresses = append(c.SchedulingAddresses, SchedulingAddress{
			ID: s.ID, URI: s.URI, Label: s.Label, Contexts: copyStrings(s.Contexts), Pref: copyIntPtr(s.Pref),
		})
	}
	for _, k := range r.Card.CryptoKeys {
		c.CryptoKeys = append(c.CryptoKeys, CryptoKey{
			ID: k.ID, URI: k.URI, MediaType: k.MediaType, Label: k.Label,
			Contexts: copyStrings(k.Contexts), Pref: copyIntPtr(k.Pref),
		})
	}
	for _, d := range r.Card.Directories {
		c.Directories = append(c.Directories, Directory{
			ID: d.ID, Kind: d.Kind, URI: d.URI, MediaType: d.MediaType, Label: d.Label,
			Contexts: copyStrings(d.Contexts), Pref: copyIntPtr(d.Pref), ListAs: copyIntPtr(d.ListAs),
		})
	}
	for _, l := range r.Card.Links {
		c.Links = append(c.Links, Link{
			ID: l.ID, URI: l.URI, MediaType: l.MediaType, Label: l.Label,
			Contexts: copyStrings(l.Contexts), Pref: copyIntPtr(l.Pref),
		})
	}
	for _, l := range r.Card.ContactURIs {
		c.Links = append(c.Links, Link{
			ID: l.ID, Kind: "contact", URI: l.URI, MediaType: l.MediaType, Label: l.Label,
			Contexts: copyStrings(l.Contexts), Pref: copyIntPtr(l.Pref),
		})
	}
}

// --- languages / related / members ----------------------------------------

func importLangsRelatedMembers(c *Card, r *contactmodel.Record) {
	for _, l := range c.PreferredLanguages {
		r.Card.PreferredLanguages = append(r.Card.PreferredLanguages, contactmodel.LanguagePref{
			ID: l.ID, Language: l.Language, Contexts: copyStrings(l.Contexts), Pref: copyIntPtr(l.Pref),
		})
	}
	for _, rel := range c.RelatedTo {
		r.Card.RelatedTo = append(r.Card.RelatedTo, contactmodel.Relation{
			Target: rel.Target, Relations: copyStrings(rel.Relation),
		})
	}
	r.Card.Members = copyStrings(c.Members)
}

func exportLangsRelatedMembers(r *contactmodel.Record, c *Card) {
	for _, l := range r.Card.PreferredLanguages {
		c.PreferredLanguages = append(c.PreferredLanguages, LanguagePref{
			ID: l.ID, Language: l.Language, Contexts: copyStrings(l.Contexts), Pref: copyIntPtr(l.Pref),
		})
	}
	for _, rel := range r.Card.RelatedTo {
		c.RelatedTo = append(c.RelatedTo, Relation{
			Target: rel.Target, Relation: copyStrings(rel.Relations),
		})
	}
	c.Members = copyStrings(r.Card.Members)
}

// --- passthrough ---------------------------------------------------------
//
// pt.vcard: Card.VCardProps is already typed as []contactmodel.JCardProp
// (types.go: "one type, one owner"), so this is a plain slice copy in both
// directions — no conversion needed.
//
// pt.jscontact: unknown JSContact properties, at any nesting depth.
// Top-level unknowns are recovered by re-parsing raw generically and diffing
// against knownCardTopLevelKeys (importUnknownTopLevel, below). Unknowns
// nested inside a known object (e.g. an extra key inside one emails{} map
// entry) are a separate concern fixed alongside this comment: codec.go's
// UnmarshalJSON methods now capture those into each wire type's unexported
// `extra` field (codec.go's "nested-unknown-property capture" section), and
// importNestedPassthrough/collectNestedPassthrough (below) walk the
// just-unmarshaled Card gathering every populated `extra` map into
// Record.Passthrough.JSContact keyed by the JSON pointer to where it
// actually lives (e.g. "/emails/k1/x-custom", not just "/x-custom" as an
// earlier, buggy version of this adapter would have keyed it, had it
// recovered the value at all). On export, spliceJSContactPassthrough
// (below) has been generalized to splice a passthrough entry back in at any
// pointer depth, not just the Card's own top level, so both cases are
// handled by the same function.

func importPassthroughVCard(c *Card, r *contactmodel.Record) {
	if len(c.VCardProps) == 0 {
		return
	}
	r.Passthrough.VCard = append([]contactmodel.JCardProp(nil), c.VCardProps...)
}

func exportPassthroughVCard(r *contactmodel.Record, c *Card) {
	if len(r.Passthrough.VCard) == 0 {
		return
	}
	c.VCardProps = append([]contactmodel.JCardProp(nil), r.Passthrough.VCard...)
}

// importUnknownTopLevel re-parses raw generically and stashes any top-level
// key not in knownCardTopLevelKeys into Record.Passthrough.JSContact, keyed
// by a top-level JSON pointer ("/"+key).
func importUnknownTopLevel(raw []byte, r *contactmodel.Record) error {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	for key, val := range doc {
		if knownCardTopLevelKeys[key] {
			continue
		}
		if r.Passthrough.JSContact == nil {
			r.Passthrough.JSContact = map[string]json.RawMessage{}
		}
		r.Passthrough.JSContact["/"+key] = val
	}
	return nil
}

// spliceJSContactPassthrough re-parses marshaled Card JSON generically and
// splices each Passthrough.JSContact entry back in at its recorded JSON
// pointer (RFC 6901), at any nesting depth — e.g. both a top-level pointer
// ("/x-vendor-extension", from an unknown top-level Card property) and a
// nested pointer ("/emails/k1/x-custom", from an unknown property inside one
// emails{} entry — see collectNestedPassthrough) are handled the same way,
// by walking the pointer's segments through the doc's existing map/array
// structure and setting the final segment as a new key. This generalizes an
// earlier version of this function that only supported single-segment
// top-level pointers; the nested case is exactly the gap this fix closes.
// A pointer that would require descending into a container that doesn't
// exist in the exported document (e.g. the referenced collection element's
// ID is no longer present on this Record) is left un-spliced rather than
// fabricating one — same fail-safe philosophy as before. The de-dup guard
// (20.5) is preserved: a pointer whose final segment already exists in the
// document (i.e. collides with a mapped/known property) is skipped, so a
// mapped property can never be shadowed/duplicated by a passthrough entry.
func spliceJSContactPassthrough(raw []byte, pt map[string]json.RawMessage) ([]byte, error) {
	if len(pt) == 0 {
		return raw, nil
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	for ptr, val := range pt {
		if ptr == "" || !strings.HasPrefix(ptr, "/") {
			continue
		}
		splicePointer(doc, strings.Split(ptr[1:], "/"), val)
	}
	return json.Marshal(doc)
}

// splicePointer descends node through segs (RFC 6901 pointer tokens, still
// escaped) and, once the final segment is reached, sets val as a new key —
// provided every intermediate segment already resolves to an existing
// container (map or array) and the final segment does not already exist
// there (the de-dup guard). A pointer that cannot be resolved this way is
// silently left un-spliced (fail-safe: never fabricates structure, never
// panics on an unexpected shape).
func splicePointer(node any, segs []string, val json.RawMessage) {
	if len(segs) == 0 {
		return
	}
	seg := jsonPointerUnescape(segs[0])
	switch n := node.(type) {
	case map[string]any:
		if len(segs) == 1 {
			if _, exists := n[seg]; exists {
				return // de-dup guard: never shadow/duplicate a mapped property
			}
			var v any
			if err := json.Unmarshal(val, &v); err != nil {
				return
			}
			n[seg] = v
			return
		}
		child, ok := n[seg]
		if !ok {
			return
		}
		splicePointer(child, segs[1:], val)
	case []any:
		idx, err := strconv.Atoi(seg)
		if err != nil || idx < 0 || idx >= len(n) {
			return
		}
		if len(segs) == 1 {
			// This package never targets a bare array index as a pointer's
			// final segment — every generated pointer's last segment is an
			// object property name (e.g. ".../components/2/x-custom", never
			// ".../components/2"). Nothing to do.
			return
		}
		splicePointer(n[idx], segs[1:], val)
	default:
		// Scalar or nil: cannot descend further; leave un-spliced.
	}
}

// jsonPointerEscape/jsonPointerUnescape implement RFC 6901 §3's token
// escaping ("~" -> "~0", "/" -> "~1", and the reverse), used when a
// collection element's ID or a relatedTo Relation's Target (either of which
// may itself contain "/" or "~", e.g. a URI) is embedded as a pointer
// segment.
func jsonPointerEscape(tok string) string {
	tok = strings.ReplaceAll(tok, "~", "~0")
	tok = strings.ReplaceAll(tok, "/", "~1")
	return tok
}

func jsonPointerUnescape(tok string) string {
	tok = strings.ReplaceAll(tok, "~1", "/")
	tok = strings.ReplaceAll(tok, "~0", "~")
	return tok
}

// importNestedPassthrough walks the just-unmarshaled Card (whose per-object
// `extra` fields, populated by codec.go's UnmarshalJSON methods, hold every
// JSON key nested inside a known object that this package's Go structs don't
// have a field for) and stashes each one into Record.Passthrough.JSContact,
// keyed by the JSON pointer to where it actually lives — see
// collectNestedPassthrough for the pointer-building walk itself.
func importNestedPassthrough(c *Card, r *contactmodel.Record) {
	nested := collectNestedPassthrough(c)
	if len(nested) == 0 {
		return
	}
	if r.Passthrough.JSContact == nil {
		r.Passthrough.JSContact = map[string]json.RawMessage{}
	}
	for ptr, val := range nested {
		r.Passthrough.JSContact[ptr] = val
	}
}

// collectNestedPassthrough walks every object in c that supports the
// nested-unknown-property capture mechanism (codec.go's `extra` field; see
// its "nested-unknown-property capture" section for exactly which types and
// why) and returns a flat map of JSON-pointer -> raw value for every
// captured-but-unrecognized key found, at whatever depth it actually lives.
// Returns nil if none were found (the overwhelmingly common case).
//
// This mirrors the Card's own field-by-field shape rather than being a
// generic reflective tree walk: Card's structure is small and fixed (it's
// this package's own root wire type), and an explicit walk is easier to
// audit against the "which types get the mechanism" scope decision than a
// reflection-based one would be.
func collectNestedPassthrough(c *Card) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	add := func(prefix string, extra map[string]json.RawMessage) {
		for k, v := range extra {
			out[prefix+"/"+jsonPointerEscape(k)] = v
		}
	}

	if c.Name != nil {
		add("/name", c.Name.extra)
		for i, comp := range c.Name.Components {
			add(fmt.Sprintf("/name/components/%d", i), comp.extra)
		}
	}
	for _, n := range c.Nicknames {
		add("/nicknames/"+jsonPointerEscape(n.ID), n.extra)
	}
	for _, o := range c.Organizations {
		p := "/organizations/" + jsonPointerEscape(o.ID)
		add(p, o.extra)
		for i, u := range o.Units {
			add(fmt.Sprintf("%s/units/%d", p, i), u.extra)
		}
	}
	for _, t := range c.Titles {
		add("/titles/"+jsonPointerEscape(t.ID), t.extra)
	}
	for _, e := range c.Emails {
		add("/emails/"+jsonPointerEscape(e.ID), e.extra)
	}
	for _, ph := range c.Phones {
		add("/phones/"+jsonPointerEscape(ph.ID), ph.extra)
	}
	for _, o := range c.OnlineServices {
		add("/onlineServices/"+jsonPointerEscape(o.ID), o.extra)
	}
	for _, a := range c.Addresses {
		p := "/addresses/" + jsonPointerEscape(a.ID)
		add(p, a.extra)
		for i, comp := range a.Components {
			add(fmt.Sprintf("%s/components/%d", p, i), comp.extra)
		}
	}
	for _, an := range c.Anniversaries {
		p := "/anniversaries/" + jsonPointerEscape(an.ID)
		add(p, an.extra)
		if an.Place != nil {
			pp := p + "/place"
			add(pp, an.Place.extra)
			for i, comp := range an.Place.Components {
				add(fmt.Sprintf("%s/components/%d", pp, i), comp.extra)
			}
		}
	}
	if c.SpeakToAs != nil {
		add("/speakToAs", c.SpeakToAs.extra)
		for _, pr := range c.SpeakToAs.Pronouns {
			add("/speakToAs/pronouns/"+jsonPointerEscape(pr.ID), pr.extra)
		}
	}
	for _, p := range c.PersonalInfo {
		add("/personalInfo/"+jsonPointerEscape(p.ID), p.extra)
	}
	for _, n := range c.Notes {
		p := "/notes/" + jsonPointerEscape(n.ID)
		add(p, n.extra)
		if n.Author != nil {
			add(p+"/author", n.Author.extra)
		}
	}
	for _, m := range c.Media {
		add("/media/"+jsonPointerEscape(m.ID), m.extra)
	}
	for _, cal := range c.Calendars {
		add("/calendars/"+jsonPointerEscape(cal.ID), cal.extra)
	}
	for _, s := range c.SchedulingAddresses {
		add("/schedulingAddresses/"+jsonPointerEscape(s.ID), s.extra)
	}
	for _, k := range c.CryptoKeys {
		add("/cryptoKeys/"+jsonPointerEscape(k.ID), k.extra)
	}
	for _, d := range c.Directories {
		add("/directories/"+jsonPointerEscape(d.ID), d.extra)
	}
	for _, l := range c.Links {
		add("/links/"+jsonPointerEscape(l.ID), l.extra)
	}
	for _, l := range c.PreferredLanguages {
		add("/preferredLanguages/"+jsonPointerEscape(l.ID), l.extra)
	}
	for _, rel := range c.RelatedTo {
		add("/relatedTo/"+jsonPointerEscape(rel.Target), rel.extra)
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
