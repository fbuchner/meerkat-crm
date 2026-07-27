package vcard3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	vcard "github.com/emersion/go-vcard"

	"mycorrhizal/contactmodel"
)

// Adapter implements contactmodel.Importer and contactmodel.Exporter for
// legacy vCard 3.0 (RFC 2426). See docs/fork-plan/30-adapters.md §30.C for
// the binding rules this file follows, and docs/fork-plan/20-correspondence.md
// (§20.3 table, §20.6 degradation summary) for the mapping this file
// implements. This adapter shares no code with backend/vcard4 or
// backend/jscontact and does not import either package.
type Adapter struct{}

var _ contactmodel.Importer = Adapter{}
var _ contactmodel.Exporter = Adapter{}

// ---------------------------------------------------------------------
// Export: neutral Record -> vCard 3.0 bytes
// ---------------------------------------------------------------------

// Export renders r into vCard 3.0 wire bytes. Per the degradation policy
// (docs/fork-plan/00-overview.md §0.5), it never returns an error for
// unmappable/degraded data: every concept named in
// docs/fork-plan/20-correspondence.md §20.6 as having no vCard 3.0 home
// yields a contactmodel.Diagnostic{Severity:"warn"} and is omitted from the
// output, while remaining untouched on the input Record.
func (Adapter) Export(r *contactmodel.Record) ([]byte, []contactmodel.Diagnostic, error) {
	if r == nil {
		r = &contactmodel.Record{}
	}
	var diags []contactmodel.Diagnostic

	card := make(vcard.Card)
	card.SetValue(PropVersion, "3.0")

	groupN := 0
	nextGroup := func() string {
		groupN++
		return fmt.Sprintf("item%d", groupN)
	}

	// --- Identity / meta ---
	if r.Card.UID != "" {
		card.SetValue(PropUID, r.Card.UID)
	}
	if r.Card.Kind != "" {
		warn(&diags, "kind", "vCard 3.0 has no KIND property; dropped")
	}
	if r.Card.ProdID != "" {
		card.SetValue(PropProdid, r.Card.ProdID)
	}
	if r.Card.Updated != nil && r.Card.Updated.UTC != "" {
		card.SetValue(PropRev, tsToV3(r.Card.Updated.UTC))
	}
	if r.Card.Created != nil {
		warn(&diags, "created", "vCard 3.0 has no CREATED property (RFC 9554); dropped")
	}
	if r.Card.Language != "" {
		warn(&diags, "language", "vCard 3.0 has no home for a card-level default language in this P0 scope; dropped (RFC 9555's richer 'apply as default LANGUAGE param' behavior is deferred post-P0 per 20-correspondence.md's language row)")
	}

	// --- Name ---
	exportName(card, r.Card.Name, &diags)

	// --- Nicknames ---
	for _, nn := range r.Card.Nicknames {
		if nn.Name == "" {
			continue
		}
		tokens := ctxToType(nn.Contexts)
		if isPreferred(nn.Pref) {
			tokens = append(tokens, "PREF")
		}
		card.Add(PropNickname, newTypedField(nn.Name, tokens))
	}

	// --- Organizations ---
	for _, org := range r.Card.Organizations {
		if org.Name == "" && len(org.Units) == 0 {
			continue
		}
		parts := []string{org.Name}
		for _, u := range org.Units {
			parts = append(parts, u.Name)
		}
		card.Add(PropOrg, &vcard.Field{Value: joinComponents(parts)})
	}

	// --- Titles / roles ---
	for _, t := range r.Card.Titles {
		if t.Name == "" {
			continue
		}
		switch t.Kind {
		case "title":
			card.Add(PropTitle, &vcard.Field{Value: t.Name})
		case "role":
			card.Add(PropRole, &vcard.Field{Value: t.Name})
		}
	}

	// --- Emails ---
	for _, e := range r.Card.Emails {
		if e.Address == "" {
			continue
		}
		tokens := []string{"INTERNET"}
		tokens = append(tokens, ctxToType(e.Contexts)...)
		if isPreferred(e.Pref) {
			tokens = append(tokens, "PREF")
		}
		card.Add(PropEmail, newTypedField(e.Address, tokens))
	}

	// --- Phones ---
	for _, p := range r.Card.Phones {
		if p.Number == "" {
			continue
		}
		var tokens []string
		for _, f := range p.Features {
			tokens = append(tokens, featToType(f))
		}
		tokens = append(tokens, ctxToType(p.Contexts)...)
		if isPreferred(p.Pref) {
			tokens = append(tokens, "PREF")
		}
		card.Add(PropTel, newTypedField(p.Number, tokens))
	}

	// --- Online services: three-array design (20.7) ---
	// vCard import/export is always unambiguous — IMPP always reads/writes
	// ImppAddresses[], X-SOCIALPROFILE always reads/writes SocialProfiles[].
	// Array membership is the sole decision; no presence-based heuristic.
	for _, os := range r.Card.ImppAddresses {
		if os.URI == "" {
			continue
		}
		tokens := ctxToType(os.Contexts)
		if isPreferred(os.Pref) {
			tokens = append(tokens, "PREF")
		}
		f := newTypedField(os.URI, tokens)
		// RFC 9554 SS4.9/4.10: SERVICE-TYPE/USERNAME "MAY be specified on an
		// IMPP or a SOCIALPROFILE property" — vCard 3.0 has no real custom
		// parameters, so (mirroring the X-SOCIALPROFILE/X-SERVICE-TYPE
		// group-linked companion property convention below) they are emitted
		// as grouped X-SERVICE-TYPE/X-USERNAME companion properties sharing
		// IMPP's GROUP token.
		if os.Service != "" || os.User != "" {
			grp := nextGroup()
			f.Group = grp
			card.Add(PropImpp, f)
			if os.Service != "" {
				card.Add(PropXServiceType, &vcard.Field{Group: grp, Value: os.Service})
			}
			if os.User != "" {
				card.Add(PropXUsername, &vcard.Field{Group: grp, Value: os.User})
			}
		} else {
			card.Add(PropImpp, f)
		}
	}
	for _, os := range r.Card.SocialProfiles {
		if os.URI == "" {
			continue
		}
		tokens := ctxToType(os.Contexts)
		if isPreferred(os.Pref) {
			tokens = append(tokens, "PREF")
		}
		grp := nextGroup()
		f := newTypedField(os.URI, tokens)
		f.Group = grp
		card.Add(PropXSocialProfile, f)
		if os.Service != "" {
			card.Add(PropXServiceType, &vcard.Field{Group: grp, Value: os.Service})
		}
		if os.User != "" {
			card.Add(PropXUsername, &vcard.Field{Group: grp, Value: os.User})
		}
	}
	// OtherOnlineServices has no vCard export at all (20.7): neither IMPP nor
	// SOCIALPROFILE is a safe default guess for genuinely unclassified data.
	for range r.Card.OtherOnlineServices {
		// Concept "onlineservice.other" (not "impp"): OtherOnlineServices has
		// no correspondence-table row at all (20-correspondence.md SS20.7) —
		// it is not the `impp` concept, just an unclassified bucket that
		// happens to share the OnlineService element type.
		warn(&diags, "onlineservice.other", "Card.OtherOnlineServices has no vCard export (neither IMPP nor SOCIALPROFILE is a safe default for unclassified data); dropped")
	}

	// --- Addresses (adr / adr.geo / adr.tz / LABEL) ---
	exportAddresses(card, r.Card.Addresses, &diags)

	// --- Anniversaries ---
	for _, ann := range r.Card.Anniversaries {
		val := formatDatePartial(ann.Date)
		switch ann.Kind {
		case "birth":
			if val != "" {
				card.SetValue(PropBday, val)
			}
			if ann.Place != nil {
				warn(&diags, "anniversary.place.birth", "BIRTHPLACE has no vCard 3.0 home (RFC 6474); dropped")
			}
		case "wedding":
			if val != "" {
				card.Add(PropXAnniversary, &vcard.Field{Value: val})
				warn(&diags, "anniversary.wedding", "vCard 3.0 has no ANNIVERSARY property; emitted as X-ANNIVERSARY instead")
			}
		case "death":
			if val != "" {
				warn(&diags, "anniversary.death", "vCard 3.0 has no DEATHDATE property (RFC 6474); dropped")
			}
			if ann.Place != nil {
				warn(&diags, "anniversary.place.death", "DEATHPLACE has no vCard 3.0 home (RFC 6474); dropped")
			}
		}
	}

	// --- SpeakToAs (9554; no 3.0 home) ---
	if r.Card.SpeakToAs != nil {
		if len(r.Card.SpeakToAs.GrammaticalGenders) > 0 {
			warn(&diags, "gramgender", "GRAMGENDER has no vCard 3.0 home (RFC 9554); dropped")
		}
		if len(r.Card.SpeakToAs.Pronouns) > 0 {
			warn(&diags, "pronouns", "PRONOUNS has no vCard 3.0 home (RFC 9554); dropped")
		}
	}

	// --- Personal info (RFC 6715; no 3.0 home) ---
	for _, pi := range r.Card.PersonalInfo {
		switch pi.Kind {
		case "expertise":
			warn(&diags, "expertise", "EXPERTISE has no vCard 3.0 home (RFC 6715); dropped")
		case "hobby":
			warn(&diags, "hobby", "HOBBY has no vCard 3.0 home (RFC 6715); dropped")
		case "interest":
			warn(&diags, "interest", "INTEREST has no vCard 3.0 home (RFC 6715); dropped")
		}
	}

	// --- Notes ---
	for _, n := range r.Card.Notes {
		if n.Note == "" {
			continue
		}
		card.Add(PropNote, &vcard.Field{Value: n.Note})
		if n.Author != nil || n.Created != nil {
			warn(&diags, "note", "NOTE AUTHOR/AUTHOR-NAME/CREATED params have no vCard 3.0 home (RFC 9554); dropped")
		}
	}

	// --- Keywords ---
	if len(r.Card.Keywords) > 0 {
		card.SetValue(PropCategories, strings.Join(r.Card.Keywords, ","))
	}

	// --- Media (photo/logo/sound) ---
	for _, m := range r.Card.Media {
		var prop string
		switch m.Kind {
		case "photo":
			prop = PropPhoto
		case "logo":
			prop = PropLogo
		case "sound":
			prop = PropSound
		default:
			continue
		}
		if m.URI == "" {
			continue
		}
		card.Add(prop, exportMediaField(m.URI, m.MediaType))
	}

	// --- Calendars / FreeBusyURLs (discrete fields, 20.7-style split) ---
	for _, c := range r.Card.Calendars {
		if c.URI == "" {
			continue
		}
		card.Add(PropCaluri, &vcard.Field{Value: c.URI})
	}
	for _, c := range r.Card.FreeBusyURLs {
		if c.URI == "" {
			continue
		}
		card.Add(PropFburl, &vcard.Field{Value: c.URI})
	}

	// --- Scheduling addresses ---
	for _, s := range r.Card.SchedulingAddresses {
		if s.URI != "" {
			card.Add(PropCaladruri, &vcard.Field{Value: s.URI})
		}
	}

	// --- Crypto keys ---
	for _, k := range r.Card.CryptoKeys {
		if k.URI == "" {
			continue
		}
		card.Add(PropKey, exportMediaField(k.URI, k.MediaType))
	}

	// --- Directories: directory (no 3.0 home) vs entry (SOURCE) ---
	for _, d := range r.Card.Directories {
		if d.URI == "" {
			continue
		}
		switch d.Kind {
		case "directory":
			warn(&diags, "directory", "ORG-DIRECTORY has no vCard 3.0 home (RFC 6715); dropped")
		case "entry":
			card.Add(PropSource, &vcard.Field{Value: d.URI})
		}
	}

	// --- Links (URL only, discrete field) ---
	for _, l := range r.Card.Links {
		if l.URI == "" {
			continue
		}
		tokens := ctxToType(l.Contexts)
		if isPreferred(l.Pref) {
			tokens = append(tokens, "PREF")
		}
		card.Add(PropURL, newTypedField(l.URI, tokens))
	}

	// --- ContactURIs: no vCard 3.0 home (RFC 8605 postdates 3.0) ---
	for range r.Card.ContactURIs {
		warn(&diags, "contacturi", "CONTACT-URI has no vCard 3.0 home (RFC 8605); dropped")
	}

	// --- Preferred languages (no 3.0 home) ---
	if len(r.Card.PreferredLanguages) > 0 {
		warn(&diags, "lang", "LANG has no vCard 3.0 home; dropped")
	}

	// --- Related (redirected onto AGENT, still warned per 20.6) ---
	for _, rel := range r.Card.RelatedTo {
		if rel.Target == "" {
			continue
		}
		f := &vcard.Field{Value: rel.Target}
		if looksLikeURI(rel.Target) {
			f.Params = vcard.Params{ParamValue: {"uri"}}
		}
		card.Add(PropAgent, f)
		warn(&diags, "related", "vCard 3.0 has no RELATED property; emitted as AGENT instead")
	}

	// --- Members (group kind only; no 3.0 home) ---
	if len(r.Card.Members) > 0 {
		warn(&diags, "member", "MEMBER has no vCard 3.0 home; dropped")
	}

	// --- Passthrough ---
	if len(r.Passthrough.JSContact) > 0 {
		warn(&diags, "pt.jscontact", "JSContact-only passthrough properties have no vCard 3.0 home; dropped")
	}
	known := knownPropNames()
	for _, jp := range r.Passthrough.VCard {
		name := strings.ToUpper(jp.Name)
		if known[name] {
			continue // 20.5 de-dup guard: never re-emit a name a mapped row already produced
		}
		card.Add(name, jcardPropToField(jp))
	}

	var buf bytes.Buffer
	if err := vcard.NewEncoder(&buf).Encode(card); err != nil {
		return nil, diags, err
	}
	return buf.Bytes(), diags, nil
}

// exportName handles the Card.Name -> FN/N mapping, including the
// name.surname2/name.generation/name.phonetic degradations (20.6).
func exportName(card vcard.Card, name *contactmodel.Name, diags *[]contactmodel.Diagnostic) {
	var full string
	var comps NComponents
	if name != nil {
		full = name.Full
		for _, c := range name.Components {
			switch c.Kind {
			case "surname":
				comps.Family = c.Value
			case "given":
				comps.Given = c.Value
			case "given2":
				comps.Additional = c.Value
			case "title":
				comps.Prefix = c.Value
			case "credential":
				comps.Suffix = c.Value
			case "surname2":
				if c.Value != "" {
					warn(diags, "name.surname2", "vCard 3.0 N has only 5 components; SecondarySurname (RFC 9554) dropped")
				}
			case "generation":
				if c.Value != "" {
					warn(diags, "name.generation", "vCard 3.0 N has only 5 components; Generation (RFC 9554) dropped")
				}
			}
		}
		if name.PhoneticScript != "" {
			warn(diags, "name.phonetic", "N PHONETIC/SCRIPT/ALTID has no vCard 3.0 home (RFC 9554); dropped")
		}
	}
	if full == "" {
		full = deriveFN(comps)
	}
	card.SetValue(PropFN, full)
	if name != nil && len(name.Components) > 0 {
		card.SetValue(PropN, assembleN(comps))
	}
}

// exportAddresses handles the Card.Addresses -> ADR/LABEL/GEO/TZ mapping,
// including the extra-component-kind degradation (part of 20.6's "extra ADR
// component kinds" bullet, folded into the `adr` concept).
func exportAddresses(card vcard.Card, addresses []contactmodel.Address, diags *[]contactmodel.Diagnostic) {
	for _, a := range addresses {
		var comps AdrComponents
		extraKind := false
		for _, c := range a.Components {
			switch c.Kind {
			case "postOfficeBox":
				comps.POBox = c.Value
			case "name":
				comps.Street = c.Value
			case "locality":
				comps.Locality = c.Value
			case "region":
				comps.Region = c.Value
			case "postcode":
				comps.Postal = c.Value
			case "country":
				comps.Country = c.Value
			default:
				if c.Value != "" {
					extraKind = true
				}
			}
		}
		if len(a.Components) == 0 && a.Full == "" && a.Coordinates == "" && a.TimeZone == "" {
			continue
		}
		tokens := ctxToType(a.Contexts)
		if isPreferred(a.Pref) {
			tokens = append(tokens, "PREF")
		}
		card.Add(PropAdr, newTypedField(assembleAdr(comps), tokens))
		if extraKind {
			warn(diags, "adr", "extra RFC 9553/9554 address component kind(s) have no vCard 3.0 ADR position; dropped")
		}
		if a.Full != "" {
			card.Add(PropLabel, newTypedField(a.Full, tokens))
		}
		if a.Coordinates != "" {
			if geo, ok := geoURIToLatLon(a.Coordinates); ok {
				card.Add(PropGeo, &vcard.Field{Value: geo})
			}
		}
		if a.TimeZone != "" {
			card.Add(PropTz, &vcard.Field{Value: a.TimeZone})
		}
	}
}

// ---------------------------------------------------------------------
// Import: vCard 3.0 bytes -> neutral Record
// ---------------------------------------------------------------------

// Import parses raw as a single vCard 3.0 card into the neutral model. It
// returns an error only when raw is not a syntactically valid vCard at all
// (0.5) — unmappable/unrecognized properties are preserved in
// Record.Passthrough.VCard and never cause a hard failure.
func (Adapter) Import(raw []byte) (*contactmodel.Record, []contactmodel.Diagnostic, error) {
	dec := vcard.NewDecoder(bytes.NewReader(raw))
	card, err := dec.Decode()
	if err != nil {
		return nil, nil, fmt.Errorf("vcard3: invalid vCard: %w", err)
	}

	var diags []contactmodel.Diagnostic
	rec := &contactmodel.Record{}

	rec.Card.UID = card.Value(PropUID)
	rec.Card.ProdID = card.Value(PropProdid)
	if rev := card.Value(PropRev); rev != "" {
		if ts, ok := v3ToTs(rev); ok {
			rec.Card.Updated = &contactmodel.Timestamp{UTC: ts}
		}
	}

	importName(card, rec)
	importNicknames(card, rec)
	importOrganizations(card, rec)
	importTitlesRoles(card, rec)
	importEmails(card, rec)
	importPhones(card, rec)
	importOnlineServices(card, rec)
	importAddresses(card, rec)
	importAnniversaries(card, rec)
	importNotes(card, rec)
	importKeywords(card, rec)
	importMediaResources(card, rec)
	importCalendars(card, rec)
	importSchedulingAddresses(card, rec)
	importCryptoKeys(card, rec)
	importDirectorySource(card, rec)
	importLinks(card, rec)
	importAgentRelated(card, rec)

	// Unknown properties -> passthrough (0.5).
	known := knownPropNames()
	for name, fields := range card {
		if known[name] {
			continue
		}
		for _, f := range fields {
			rec.Passthrough.VCard = append(rec.Passthrough.VCard, fieldToJCardProp(name, f))
		}
	}

	return rec, diags, nil
}

func importName(card vcard.Card, rec *contactmodel.Record) {
	fn := card.Value(PropFN)
	nVal := card.Value(PropN)
	if fn == "" && nVal == "" {
		return
	}
	name := &contactmodel.Name{Full: fn}
	if nVal != "" {
		c := disassembleN(nVal)
		add := func(kind, v string) {
			if v != "" {
				name.Components = append(name.Components, contactmodel.NameComponent{Kind: kind, Value: v})
			}
		}
		add("surname", c.Family)
		add("given", c.Given)
		add("given2", c.Additional)
		add("title", c.Prefix)
		add("credential", c.Suffix)
	}
	rec.Card.Name = name
}

func importNicknames(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropNickname] {
		if f.Value == "" {
			continue
		}
		ctx, pref := contextsAndPrefFromTokens(typeTokens(f))
		rec.Card.Nicknames = append(rec.Card.Nicknames, contactmodel.Nickname{
			Name: f.Value, Contexts: ctx, Pref: pref,
		})
	}
}

func importOrganizations(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropOrg] {
		if f.Value == "" {
			continue
		}
		comps := splitComponents(f.Value)
		org := contactmodel.Organization{Name: component(comps, 0)}
		for _, u := range comps[minInt(1, len(comps)):] {
			if u != "" {
				org.Units = append(org.Units, contactmodel.OrgUnit{Name: u})
			}
		}
		rec.Card.Organizations = append(rec.Card.Organizations, org)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func importTitlesRoles(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropTitle] {
		if f.Value != "" {
			rec.Card.Titles = append(rec.Card.Titles, contactmodel.Title{Name: f.Value, Kind: "title"})
		}
	}
	for _, f := range card[PropRole] {
		if f.Value != "" {
			rec.Card.Titles = append(rec.Card.Titles, contactmodel.Title{Name: f.Value, Kind: "role"})
		}
	}
}

func importEmails(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropEmail] {
		if f.Value == "" {
			continue
		}
		ctx, pref := contextsAndPrefFromTokens(typeTokens(f))
		rec.Card.Emails = append(rec.Card.Emails, contactmodel.Email{Address: f.Value, Contexts: ctx, Pref: pref})
	}
}

func importPhones(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropTel] {
		if f.Value == "" {
			continue
		}
		toks := typeTokens(f)
		ctx, pref := contextsAndPrefFromTokens(toks)
		rec.Card.Phones = append(rec.Card.Phones, contactmodel.Phone{
			Number: f.Value, Features: featuresFromTokens(toks), Contexts: ctx, Pref: pref,
		})
	}
}

// importOnlineServices implements the three-array design (20.7): IMPP always
// reads into ImppAddresses[], X-SOCIALPROFILE always reads into
// SocialProfiles[] — the source property is the sole, unambiguous
// discriminator; no per-element tag or heuristic is needed on the vCard side.
func importOnlineServices(card vcard.Card, rec *contactmodel.Record) {
	// serviceTypeFields/usernameFields are the grouped X-SERVICE-TYPE/
	// X-USERNAME companion properties (this package's vCard-3.0 emulation of
	// vCard 4.0's SERVICE-TYPE/USERNAME parameters, RFC 9554 SS4.9/4.10, which
	// "MAY be specified on an IMPP or a SOCIALPROFILE property" — read here
	// for both).
	serviceTypeFields := card[PropXServiceType]
	usernameFields := card[PropXUsername]
	for _, f := range card[PropImpp] {
		if f.Value == "" {
			continue
		}
		ctx, pref := contextsAndPrefFromTokens(typeTokens(f))
		service := labelFromGroup(serviceTypeFields, f.Group)
		user := labelFromGroup(usernameFields, f.Group)
		rec.Card.ImppAddresses = append(rec.Card.ImppAddresses, contactmodel.OnlineService{
			URI: f.Value, Service: service, User: user, Contexts: ctx, Pref: pref,
		})
	}
	for _, f := range card[PropXSocialProfile] {
		if f.Value == "" {
			continue
		}
		ctx, pref := contextsAndPrefFromTokens(typeTokens(f))
		service := labelFromGroup(serviceTypeFields, f.Group)
		user := labelFromGroup(usernameFields, f.Group)
		rec.Card.SocialProfiles = append(rec.Card.SocialProfiles, contactmodel.OnlineService{
			URI: f.Value, Service: service, User: user, Contexts: ctx, Pref: pref,
		})
	}
}

func importAddresses(card vcard.Card, rec *contactmodel.Record) {
	adrFields := card[PropAdr]
	labelFields := card[PropLabel]
	geoFields := card[PropGeo]
	tzFields := card[PropTz]
	for i, f := range adrFields {
		c := disassembleAdr(f.Value)
		var comps []contactmodel.AddressComponent
		add := func(kind, v string) {
			if v != "" {
				comps = append(comps, contactmodel.AddressComponent{Kind: kind, Value: v})
			}
		}
		add("postOfficeBox", c.POBox)
		add("name", c.Street)
		add("locality", c.Locality)
		add("region", c.Region)
		add("postcode", c.Postal)
		add("country", c.Country)

		ctx, pref := contextsAndPrefFromTokens(typeTokens(f))
		addr := contactmodel.Address{Components: comps, Contexts: ctx, Pref: pref}
		if full, ok := matchLabelByType(labelFields, typeTokens(f)); ok {
			addr.Full = full
		}
		if i < len(geoFields) {
			if uri, ok := latLonToGeoURI(geoFields[i].Value); ok {
				addr.Coordinates = uri
			}
		}
		if i < len(tzFields) {
			addr.TimeZone = tzFields[i].Value
		}
		rec.Card.Addresses = append(rec.Card.Addresses, addr)
	}
}

func importAnniversaries(card vcard.Card, rec *contactmodel.Record) {
	if v := card.Value(PropBday); v != "" {
		rec.Card.Anniversaries = append(rec.Card.Anniversaries, contactmodel.Anniversary{
			Kind: "birth", Date: parseDatePartial(v),
		})
	}
	for _, f := range card[PropXAnniversary] {
		if f.Value == "" {
			continue
		}
		rec.Card.Anniversaries = append(rec.Card.Anniversaries, contactmodel.Anniversary{
			Kind: "wedding", Date: parseDatePartial(f.Value),
		})
	}
}

func importNotes(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropNote] {
		if f.Value != "" {
			rec.Card.Notes = append(rec.Card.Notes, contactmodel.Note{Note: f.Value})
		}
	}
}

func importKeywords(card vcard.Card, rec *contactmodel.Record) {
	v := card.Value(PropCategories)
	if v == "" {
		return
	}
	for _, k := range strings.Split(v, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			rec.Card.Keywords = append(rec.Card.Keywords, k)
		}
	}
}

func importMediaResources(card vcard.Card, rec *contactmodel.Record) {
	importOne := func(prop, kind string) {
		for _, f := range card[prop] {
			if f.Value == "" {
				continue
			}
			uri := importMediaURI(f, kind)
			if uri == "" {
				continue
			}
			rec.Card.Media = append(rec.Card.Media, contactmodel.Resource{Kind: kind, URI: uri})
		}
	}
	importOne(PropPhoto, "photo")
	importOne(PropLogo, "logo")
	importOne(PropSound, "sound")
}

// importCalendars implements the discrete Calendars/FreeBusyURLs split
// (20.7-style): CALURI always lands in Calendars[], FBURL always lands in
// FreeBusyURLs[] — Kind is no longer used to discriminate these.
func importCalendars(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropCaluri] {
		if f.Value != "" {
			rec.Card.Calendars = append(rec.Card.Calendars, contactmodel.Resource{URI: f.Value})
		}
	}
	for _, f := range card[PropFburl] {
		if f.Value != "" {
			rec.Card.FreeBusyURLs = append(rec.Card.FreeBusyURLs, contactmodel.Resource{URI: f.Value})
		}
	}
}

func importSchedulingAddresses(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropCaladruri] {
		if f.Value != "" {
			rec.Card.SchedulingAddresses = append(rec.Card.SchedulingAddresses, contactmodel.Resource{URI: f.Value})
		}
	}
}

func importCryptoKeys(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropKey] {
		if f.Value == "" {
			continue
		}
		rec.Card.CryptoKeys = append(rec.Card.CryptoKeys, contactmodel.Resource{URI: importMediaURI(f, "key")})
	}
}

func importDirectorySource(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropSource] {
		if f.Value != "" {
			rec.Card.Directories = append(rec.Card.Directories, contactmodel.Resource{Kind: "entry", URI: f.Value})
		}
	}
}

func importLinks(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropURL] {
		if f.Value == "" {
			continue
		}
		ctx, pref := contextsAndPrefFromTokens(typeTokens(f))
		rec.Card.Links = append(rec.Card.Links, contactmodel.Resource{URI: f.Value, Contexts: ctx, Pref: pref})
	}
}

// importAgentRelated reverses the "related" concept's export-side AGENT
// redirect (20.6: "3.0: no RELATED → AGENT/warn") so a vcard3 round-trip
// preserves the relation's target. This isn't separately called out by
// 30-adapters.md's import rule (which only names X-ANNIVERSARY/
// X-SOCIALPROFILE explicitly), but AGENT is otherwise one of our "known"
// properties with no import destination at all, which would silently drop
// data on a round trip rather than degrade gracefully — so it is treated
// symmetrically with the other redirected concepts. See the final WP-50
// report for this judgment call.
func importAgentRelated(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropAgent] {
		if f.Value != "" {
			rec.Card.RelatedTo = append(rec.Card.RelatedTo, contactmodel.Relation{Target: f.Value})
		}
	}
}

// ---------------------------------------------------------------------
// Shared helpers (transforms, 20.4)
// ---------------------------------------------------------------------

func warn(diags *[]contactmodel.Diagnostic, concept, msg string) {
	*diags = append(*diags, contactmodel.Diagnostic{Severity: "warn", Concept: concept, Message: msg})
}

// ctxToType implements the `ctx2type` transform (20.4): private->home,
// work->work, rendered as vCard 3.0 TYPE tokens.
func ctxToType(contexts []string) []string {
	var out []string
	for _, c := range contexts {
		switch c {
		case "private":
			out = append(out, "HOME")
		case "work":
			out = append(out, "WORK")
		default:
			if c != "" {
				out = append(out, strings.ToUpper(c))
			}
		}
	}
	return out
}

// contextsAndPrefFromTokens is the reverse of ctxToType plus PREF-token
// recognition (RFC 2426 §3.3.2: PREF is itself a TYPE token in 3.0, unlike
// 4.0's dedicated PREF parameter — docs/specs/rfc2426-v3-baseline.md §3).
func contextsAndPrefFromTokens(tokens []string) ([]string, *int) {
	var ctx []string
	var pref *int
	for _, raw := range tokens {
		switch strings.ToUpper(strings.TrimSpace(raw)) {
		case "PREF":
			one := 1
			pref = &one
		case "HOME":
			ctx = append(ctx, "private")
		case "WORK":
			ctx = append(ctx, "work")
		}
	}
	return ctx, pref
}

// isPreferred reports whether pref designates the (most) preferred entry —
// the only case that gets a PREF TYPE token on vCard 3.0 export.
func isPreferred(pref *int) bool {
	return pref != nil && *pref == 1
}

// featToType implements (the v3 half of) the `feat2type` transform (20.4).
func featToType(f string) string {
	if strings.EqualFold(f, "mobile") {
		return "CELL"
	}
	return strings.ToUpper(f)
}

// telFeatureTokens are the RFC 2426 TEL TYPE tokens this adapter recognizes
// as neutral Phone.Features (voice,fax,cell,video,pager) plus the 4.0-only
// tokens text/textphone/main-number, which RFC 2426 doesn't define but which
// this adapter still emits/parses permissively (graceful degradation, not a
// hard error) since the `phone`/TEL row itself is fully mapped (not a 20.6
// concept) — see the final WP-50 report.
var telFeatureTokens = map[string]string{
	"VOICE":       "voice",
	"FAX":         "fax",
	"CELL":        "cell",
	"VIDEO":       "video",
	"PAGER":       "pager",
	"TEXT":        "text",
	"TEXTPHONE":   "textphone",
	"MAIN-NUMBER": "main-number",
}

func featuresFromTokens(tokens []string) []string {
	var out []string
	for _, raw := range tokens {
		if f, ok := telFeatureTokens[strings.ToUpper(strings.TrimSpace(raw))]; ok {
			out = append(out, f)
		}
	}
	return out
}

// newTypedField builds a *vcard.Field whose TYPE parameter carries typeTokens
// as repeated Params values — matching the existing, salvaged
// backend/carddav/vcard_mapper.go idiom (addTypedField/typeTokens) for how
// this codebase represents a vCard 3.0 multi-token TYPE list. See this
// package's final report for the (documented, test-invisible) wire-format
// nuance this implies.
func newTypedField(value string, typeTokens []string) *vcard.Field {
	f := &vcard.Field{Value: value}
	if len(typeTokens) > 0 {
		f.Params = vcard.Params{}
		for _, t := range typeTokens {
			f.Params.Add(ParamType, t)
		}
	}
	return f
}

// typeTokens returns field f's TYPE parameter values (already
// comma/occurrence-flattened by go-vcard's decoder).
func typeTokens(f *vcard.Field) []string {
	if f == nil || f.Params == nil {
		return nil
	}
	return f.Params[ParamType]
}

// deriveFN computes a fallback FN (required by both 3.0 and 4.0) from N
// components when Name.Full is empty, joining non-empty parts in display
// order: Prefix Given Additional Family Suffix.
func deriveFN(c NComponents) string {
	var parts []string
	for _, p := range []string{c.Prefix, c.Given, c.Additional, c.Family, c.Suffix} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// looksLikeURI is a permissive heuristic used only to decide whether a
// RelatedTo.Target gets AGENT's VALUE=uri parameter on export.
func looksLikeURI(s string) bool {
	return strings.Contains(s, "://") || strings.HasPrefix(s, "urn:") || strings.HasPrefix(s, "mailto:")
}

// typeKey normalizes a TYPE token list into an order-independent comparison
// key, used to pair an ADR field with its LABEL property (RFC 2426 has no
// ADR LABEL parameter — LABEL is its own property, paired to ADR by matching
// TYPE; docs/specs/rfc2426-v3-baseline.md §6).
func typeKey(tokens []string) string {
	up := make([]string, len(tokens))
	for i, t := range tokens {
		up[i] = strings.ToUpper(strings.TrimSpace(t))
	}
	sort.Strings(up)
	return strings.Join(up, ",")
}

func matchLabelByType(labelFields []*vcard.Field, wantTokens []string) (string, bool) {
	want := typeKey(wantTokens)
	for _, f := range labelFields {
		if typeKey(typeTokens(f)) == want {
			return f.Value, true
		}
	}
	return "", false
}

func labelFromGroup(fields []*vcard.Field, group string) string {
	if group == "" {
		return ""
	}
	for _, f := range fields {
		if f != nil && strings.EqualFold(f.Group, group) {
			return f.Value
		}
	}
	return ""
}

// formatDatePartial implements the `date_partial` transform's export
// direction: neutral AnniversaryDate -> vCard DATE-AND-OR-TIME text.
func formatDatePartial(d contactmodel.AnniversaryDate) string {
	if d.Timestamp != nil && *d.Timestamp != "" {
		ts := *d.Timestamp
		if len(ts) >= 10 {
			return ts[:10]
		}
		return ts
	}
	if d.Partial != nil {
		p := d.Partial
		switch {
		case p.Year != nil && p.Month != nil && p.Day != nil:
			return fmt.Sprintf("%04d-%02d-%02d", *p.Year, *p.Month, *p.Day)
		case p.Year == nil && p.Month != nil && p.Day != nil:
			return fmt.Sprintf("--%02d-%02d", *p.Month, *p.Day)
		case p.Year != nil && p.Month != nil && p.Day == nil:
			return fmt.Sprintf("%04d-%02d", *p.Year, *p.Month)
		case p.Year != nil && p.Month == nil && p.Day == nil:
			return fmt.Sprintf("%04d", *p.Year)
		case p.Year == nil && p.Month != nil && p.Day == nil:
			return fmt.Sprintf("--%02d", *p.Month)
		}
	}
	return ""
}

// parseDatePartial implements the `date_partial` transform's import
// direction: vCard DATE-AND-OR-TIME text -> neutral AnniversaryDate. It
// accepts YYYY-MM-DD, --MM-DD, YYYY, and the legacy compact YYYYMMDD/--MMDD
// forms (docs/carddav's normalizeBirthday handled the same compact forms).
func parseDatePartial(v string) contactmodel.AnniversaryDate {
	v = strings.TrimSpace(v)
	switch {
	case len(v) == 10 && v[4] == '-' && v[7] == '-':
		y, _ := strconv.Atoi(v[0:4])
		m, _ := strconv.Atoi(v[5:7])
		d, _ := strconv.Atoi(v[8:10])
		return contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Year: &y, Month: &m, Day: &d}}
	case len(v) == 8 && v[0] == '-' && v[1] == '-' && v[4] == '-':
		m, _ := strconv.Atoi(v[2:4])
		d, _ := strconv.Atoi(v[5:7])
		return contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Month: &m, Day: &d}}
	case len(v) == 8 && isAllDigits(v):
		y, _ := strconv.Atoi(v[0:4])
		m, _ := strconv.Atoi(v[4:6])
		d, _ := strconv.Atoi(v[6:8])
		return contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Year: &y, Month: &m, Day: &d}}
	case len(v) == 6 && v[0] == '-' && v[1] == '-':
		m, _ := strconv.Atoi(v[2:4])
		d, _ := strconv.Atoi(v[4:6])
		return contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Month: &m, Day: &d}}
	case len(v) == 4 && isAllDigits(v):
		y, _ := strconv.Atoi(v)
		return contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Year: &y}}
	default:
		return contactmodel.AnniversaryDate{Timestamp: &v}
	}
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// tsToV3 implements the `ts_rfc3339` transform's export direction. Unlike
// vCard 4.0's compact TIMESTAMP form, RFC 2426 §3.6.4 REV uses the same
// extended (dashed) ISO 8601 form as the neutral model's RFC3339
// representation, so this is close to identity; it just normalizes to UTC.
func tsToV3(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// v3ToTs implements the `ts_rfc3339` transform's import direction, also
// tolerating the vCard 4.0-style compact TIMESTAMP form in case a
// 4.0-authored REV value ends up in a 3.0 file.
func v3ToTs(v string) (string, bool) {
	layouts := []string{time.RFC3339, "20060102T150405Z", "2006-01-02T15:04:05"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t.UTC().Format(time.RFC3339), true
		}
	}
	return "", false
}

// geoURIToLatLon implements the `geo_uri` transform's export direction:
// neutral "geo:LAT,LON" -> vCard 3.0 GEO property value "LAT;LON".
func geoURIToLatLon(uri string) (string, bool) {
	if !strings.HasPrefix(strings.ToLower(uri), "geo:") {
		return "", false
	}
	rest := uri[len("geo:"):]
	if i := strings.Index(rest, ";"); i >= 0 {
		rest = rest[:i]
	}
	parts := strings.SplitN(rest, ",", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0] + ";" + parts[1], true
}

// latLonToGeoURI implements the `geo_uri` transform's import direction.
func latLonToGeoURI(v string) (string, bool) {
	parts := strings.SplitN(v, ";", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return "geo:" + parts[0] + "," + parts[1], true
}

// mtypeToV3Type derives a vCard 3.0 TYPE token (e.g. "JPEG") from a MIME
// media type (e.g. "image/jpeg"), per the `media_uri` transform.
func mtypeToV3Type(mt string) string {
	mt = strings.TrimSpace(mt)
	if i := strings.Index(mt, "/"); i >= 0 {
		return strings.ToUpper(mt[i+1:])
	}
	return strings.ToUpper(mt)
}

// mtypeFromV3 is the reverse of mtypeToV3Type, reconstructing a MIME media
// type from a 3.0 TYPE token given the resource kind.
func mtypeFromV3(kind, typ string) string {
	typ = strings.ToLower(typ)
	switch kind {
	case "photo", "logo":
		if typ == "" {
			return "image/jpeg"
		}
		return "image/" + typ
	case "sound":
		if typ == "" {
			return "audio/basic"
		}
		return "audio/" + typ
	default: // key, etc.
		if typ == "" {
			return "application/octet-stream"
		}
		return "application/" + typ
	}
}

// parseDataURI splits a "data:<mediatype>;base64,<data>" URI into its
// mediatype and base64 payload.
func parseDataURI(uri string) (data, mediatype string, ok bool) {
	if !strings.HasPrefix(uri, "data:") {
		return "", "", false
	}
	rest := uri[len("data:"):]
	comma := strings.Index(rest, ",")
	if comma < 0 {
		return "", "", false
	}
	meta := strings.TrimSuffix(rest[:comma], ";base64")
	return rest[comma+1:], meta, true
}

// exportMediaField implements the `media_uri` transform's export direction:
// a data: URI becomes inline ENCODING=b/TYPE=... (RFC 2426 §3.1.4); any other
// URI (the source is already a URI) is emitted as a bare value, per
// docs/specs/rfc2426-v3-baseline.md §4 and 30-adapters.md §30.C.
func exportMediaField(uri, mediaType string) *vcard.Field {
	if data, mt, ok := parseDataURI(uri); ok {
		if mt == "" {
			mt = mediaType
		}
		f := &vcard.Field{Value: data, Params: vcard.Params{}}
		f.Params.Set(ParamEncoding, "b")
		if t := mtypeToV3Type(mt); t != "" {
			f.Params.Set(ParamType, t)
		}
		return f
	}
	return &vcard.Field{Value: uri}
}

// importMediaURI implements the `media_uri` transform's import direction.
func importMediaURI(f *vcard.Field, kind string) string {
	enc := strings.ToLower(f.Params.Get(ParamEncoding))
	if enc == "b" || enc == "base64" {
		return "data:" + mtypeFromV3(kind, f.Params.Get(ParamType)) + ";base64," + f.Value
	}
	return f.Value
}

// knownPropNames lists every vCard 3.0 property name this adapter actively
// interprets on import (structurally mapped, or consumed as a companion of a
// mapped property like X-ABLabel/X-SERVICE-TYPE/GEO/TZ/LABEL). Anything else
// is preserved verbatim in Passthrough.VCard (0.5) and, on export, re-emitted
// unless its name is in this set (the 20.5 de-dup guard).
func knownPropNames() map[string]bool {
	names := []string{
		PropBegin, PropEnd, PropVersion, PropUID, PropProdid, PropRev,
		PropFN, PropN, PropNickname, PropOrg, PropTitle, PropRole,
		PropEmail, PropTel, PropImpp, PropAdr, PropGeo, PropTz, PropLabel,
		PropBday, PropNote, PropCategories, PropPhoto, PropLogo, PropSound,
		PropCaluri, PropFburl, PropCaladruri, PropKey, PropSource, PropURL,
		PropAgent, PropXAnniversary, PropXSocialProfile, PropXServiceType,
		propXABLabel,
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// fieldToJCardProp implements the `passthrough_vcard` transform's import
// direction, converting an unrecognized vCard field into the neutral
// JCardProp shape (RFC 9555 "vCardProps"). Since JCardProp has no dedicated
// Group field, a non-empty property group is preserved as a synthetic
// "group" param entry (a documented judgment call — see the final WP-50
// report).
func fieldToJCardProp(name string, f *vcard.Field) contactmodel.JCardProp {
	params := map[string]any{}
	for k, vs := range f.Params {
		switch len(vs) {
		case 0:
		case 1:
			params[strings.ToLower(k)] = vs[0]
		default:
			params[strings.ToLower(k)] = append([]string(nil), vs...)
		}
	}
	if f.Group != "" {
		params["group"] = f.Group
	}
	valBytes, _ := json.Marshal(f.Value)
	return contactmodel.JCardProp{
		Name:   strings.ToLower(name),
		Params: params,
		Type:   "text",
		Value:  valBytes,
	}
}

// jcardPropToField is the reverse of fieldToJCardProp, used to re-emit
// stored passthrough properties on export (`passthrough_vcard`, export
// direction).
func jcardPropToField(jp contactmodel.JCardProp) *vcard.Field {
	f := &vcard.Field{}
	var s string
	if err := json.Unmarshal(jp.Value, &s); err == nil {
		f.Value = s
	} else {
		f.Value = string(jp.Value)
	}
	if len(jp.Params) > 0 {
		f.Params = vcard.Params{}
		for k, v := range jp.Params {
			if strings.EqualFold(k, "group") {
				if s2, ok := v.(string); ok {
					f.Group = s2
				}
				continue
			}
			switch vv := v.(type) {
			case string:
				f.Params.Add(strings.ToUpper(k), vv)
			case []string:
				for _, item := range vv {
					f.Params.Add(strings.ToUpper(k), item)
				}
			case []any:
				for _, item := range vv {
					if s2, ok := item.(string); ok {
						f.Params.Add(strings.ToUpper(k), s2)
					}
				}
			}
		}
	}
	return f
}
