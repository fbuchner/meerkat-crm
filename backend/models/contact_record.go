package models

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"mycorrhizal/contactmodel"
	"mycorrhizal/logger"
	"mycorrhizal/photostore"

	"gorm.io/gorm"
)

// RecordForContact returns the authoritative Record for a Contact that is
// about to be READ (served over CardDAV, returned from the REST API, or
// exported as VCF/JSContact) — as opposed to RecordFromContact, which
// derives one FRESH from the flat legacy fields only.
//
// The distinction matters: BeforeSave/ApplyRecordToContact already keep
// c.Card/c.CRM/c.Passthrough in sync as the full-fidelity representation,
// including data with no flat-field home at all (SpeakToAs, PersonalInfo,
// SocialProfiles, extra name components, ...). Calling RecordFromContact
// again on a read path silently discards all of that, reconstructing only
// what the (necessarily lossy) flat fields can represent — this was a real,
// live bug found three times independently: once during WP-73 (CardDAV
// export, fixed as this function's prototype), and twice more here (REST
// API detail/write responses in contact_summary.go, and VCF/JSContact
// export in export_controller.go) while auditing WP-73's work. All three
// call sites now go through this one function instead of guessing at the
// same fix independently — see docs/fork-plan/50-integration-and-rebrand.md
// WP-70's "Shared mapping function" note; the same discipline applies here.
//
// Falls back to a fresh RecordFromContact derivation only when c.Card is
// still the zero value — i.e. a pre-migration row that hasn't been through
// cmd/backfill-contact-records yet, or a Contact that was never saved (so
// BeforeSave never ran). This protects those cases from exporting/serving a
// near-empty Card instead of the best-effort flat-field reconstruction.
// db is used to project confirmed relationship-graph edges (WP-80, docs/
// fork-plan/91-envelope-data-model.md §91.2) into the returned Record's
// Card.RelatedTo — see projectRelationshipEdges below. Passing nil skips
// projection entirely (Card.RelatedTo is returned exactly as stored), which
// existing callers that predate WP-80 and don't have a *gorm.DB handy can
// rely on rather than being forced to thread one through immediately.
func RecordForContact(c *Contact, photoDir string, db *gorm.DB) *contactmodel.Record {
	if c != nil && !reflect.DeepEqual(c.Card, contactmodel.Card{}) {
		card := c.Card
		card.RelatedTo = projectRelationshipEdges(db, c.VCardUID, card.RelatedTo)
		card.Keywords = projectTags(db, c.VCardUID, card.Keywords)
		card.PersonalInfo = projectPreferences(db, c.VCardUID, card.PersonalInfo)
		passthrough := c.Passthrough
		passthrough.VCard = projectCustomFields(db, c.VCardUID, passthrough.VCard)
		return &contactmodel.Record{
			UID:         c.VCardUID,
			ETag:        c.ETag,
			Card:        card,
			Envelope:    c.CRM,
			Passthrough: passthrough,
		}
	}
	record := RecordFromContact(c, photoDir)
	if record != nil {
		record.Card.RelatedTo = projectRelationshipEdges(db, record.UID, record.Card.RelatedTo)
		record.Card.Keywords = projectTags(db, record.UID, record.Card.Keywords)
		record.Card.PersonalInfo = projectPreferences(db, record.UID, record.Card.PersonalInfo)
		record.Passthrough.VCard = projectCustomFields(db, record.UID, record.Passthrough.VCard)
	}
	return record
}

// projectRelationshipEdges is WP-80's "Card.RelatedTo projection wiring": it
// synthesizes graph-derived contactmodel.Relation entries for every
// confirmed, normal-sensitivity RelationshipEdge touching vcardUID, and
// returns them appended to a COPY of existing (imported/passthrough)
// entries. It never mutates existing's backing array and the result is never
// written back to Contact.Card — this is pure read-time synthesis, so
// Card.RelatedTo's passthrough entries stay exactly as imported regardless
// of how many times a Record is read (docs/fork-plan/90-vision-and-
// reconciliation.md D3: the graph is the source of truth, Card.RelatedTo is
// "the export projection... not a second source of truth").
//
// A single edge can project onto BOTH endpoints' cards, because vCard
// RELATED is inherently one-sided (it describes what the *other* entity is
// *to the card holder*): edge (source=A, target=B, type=parent_of) means
// "on B's card, A is my parent" (RelationVCardTypeTag("parent_of")) AND "on
// A's card, B is my child" (RelationVCardTypeTag(InverseRelationType(
// "parent_of"))). Only emitted where the relevant side's vCard TYPE tag is
// non-empty — types with no standard equivalent (co_parent_of, the affinity
// edges, ...) simply never appear here, per §91.2's "deliberately lossy"
// export rule.
//
// Suggested-status edges are never projected (§91.2: "only confirmed edges
// are authoritative"), and neither are edges above normal sensitivity
// (§91.13's default-exclude-from-export rule) — both filtered in the query
// itself rather than after loading, so a query failure can't accidentally
// leak a suggested or sensitive edge into an export.
func projectRelationshipEdges(db *gorm.DB, vcardUID string, existing []contactmodel.Relation) []contactmodel.Relation {
	if db == nil || vcardUID == "" {
		return existing
	}

	var edges []RelationshipEdge
	if err := db.Where(
		"(source_id = ? OR target_id = ?) AND status = ? AND sensitivity = ?",
		vcardUID, vcardUID, RelationshipStatusConfirmed, RelationshipSensitivityNormal,
	).Find(&edges).Error; err != nil {
		// Projection is best-effort enrichment of an already-valid export
		// payload — RecordForContact has no error return (matching
		// RecordFromContact's "never panics" contract), so a query failure
		// degrades to "just the passthrough entries" rather than failing the
		// whole read/export.
		logger.Warn().Err(err).Str("vcard_uid", vcardUID).Msg("projectRelationshipEdges: failed to load confirmed edges")
		return existing
	}
	if len(edges) == 0 {
		return existing
	}

	seen := make(map[[2]string]bool, len(existing))
	for _, rel := range existing {
		for _, tag := range rel.Relations {
			seen[[2]string{rel.Target, tag}] = true
		}
	}

	result := append([]contactmodel.Relation(nil), existing...)
	appendIfNew := func(targetUID, tag string) {
		if tag == "" {
			return
		}
		target := "urn:uuid:" + targetUID
		key := [2]string{target, tag}
		if seen[key] {
			return
		}
		seen[key] = true
		result = append(result, contactmodel.Relation{Target: target, Relations: []string{tag}})
	}

	for _, edge := range edges {
		if edge.SourceID == vcardUID {
			// This contact is the source: describe the target using this
			// edge's own type (e.g. "B is my child" for parent_of A->B).
			appendIfNew(edge.TargetID, RelationVCardTypeTag(InverseRelationType(edge.Type)))
		}
		if edge.TargetID == vcardUID {
			// This contact is the target: describe the source using the
			// edge's type directly (e.g. "A is my parent" for parent_of
			// A->B, read from B's side).
			appendIfNew(edge.SourceID, RelationVCardTypeTag(edge.Type))
		}
	}

	return result
}

// projectTags is WP-84's Tag -> Card.Keywords projection (§91.5): tags are
// "an attribute a set of people share" and have a clean standards home
// (vCard CATEGORIES / JSContact keywords), unlike Circles which stay
// internal-only. Structurally identical to projectRelationshipEdges above —
// nil-safe, best-effort (a query failure degrades to "just the existing
// keywords" rather than failing the whole read), and returns a new slice
// rather than mutating existing.
func projectTags(db *gorm.DB, vcardUID string, existing []string) []string {
	if db == nil || vcardUID == "" {
		return existing
	}

	var taggings []ContactTag
	if err := db.Where("contact_vcard_uid = ?", vcardUID).Find(&taggings).Error; err != nil {
		logger.Warn().Err(err).Str("vcard_uid", vcardUID).Msg("projectTags: failed to load taggings")
		return existing
	}
	if len(taggings) == 0 {
		return existing
	}

	tagIDs := make([]string, len(taggings))
	for i, tagging := range taggings {
		tagIDs[i] = tagging.TagID
	}
	var tags []Tag
	if err := db.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
		logger.Warn().Err(err).Str("vcard_uid", vcardUID).Msg("projectTags: failed to load tags")
		return existing
	}

	seen := make(map[string]bool, len(existing))
	result := append([]string(nil), existing...)
	for _, kw := range existing {
		seen[kw] = true
	}
	for _, tag := range tags {
		if tag.Name == "" || seen[tag.Name] {
			continue
		}
		seen[tag.Name] = true
		result = append(result, tag.Name)
	}

	return result
}

// projectPreferences is T20a's Preference -> Card.PersonalInfo projection
// (docs/fork-plan/91-envelope-data-model.md §91.9): "a Preference of category
// hobby may project to Card.PersonalInfo, but most preference categories have
// no standard home and stay internal." So hobby-category preferences become
// PersonalInfo{Kind:"hobby", Value:<value>} entries appended to a COPY of
// existing (imported/passthrough) entries, and every other category never
// appears here. Structurally identical to projectTags/projectRelationshipEdges
// above — nil-safe, best-effort (a query failure degrades to "just the
// existing personalInfo entries" rather than failing the whole read), and
// §91.13 sensitivity is filtered in the query itself (only
// sensitivity='normal' preferences project), never in the caller.
func projectPreferences(db *gorm.DB, vcardUID string, existing []contactmodel.PersonalInfo) []contactmodel.PersonalInfo {
	if db == nil || vcardUID == "" {
		return existing
	}

	var prefs []Preference
	if err := db.Where(
		"entity_id = ? AND category = ? AND sensitivity = ?",
		vcardUID, PreferenceCategoryHobby, RelationshipSensitivityNormal,
	).Find(&prefs).Error; err != nil {
		logger.Warn().Err(err).Str("vcard_uid", vcardUID).Msg("projectPreferences: failed to load hobby preferences")
		return existing
	}
	if len(prefs) == 0 {
		return existing
	}

	seen := make(map[string]bool, len(existing))
	result := append([]contactmodel.PersonalInfo(nil), existing...)
	for _, p := range existing {
		if p.Value != "" {
			seen[p.Value] = true
		}
	}
	for _, pref := range prefs {
		value := pref.Value
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, contactmodel.PersonalInfo{Kind: "hobby", Value: value})
	}

	return result
}

// projectCustomFields is WP-84b's FieldValue -> Passthrough.VCard projection
// (docs/fork-plan/94-custom-fields.md §94.5): a definition whose Projection
// is "vcard:X-<NAME>" rides the existing JCardProp/passthrough machinery,
// exactly like an imported unknown vCard property, except the CRM itself is
// the source rather than an external file. "internal-only" definitions (the
// default) never appear here. Structurally identical to projectTags/
// projectRelationshipEdges above — nil-safe, best-effort, sensitivity
// filtered in the query itself (only sensitivity='normal' definitions
// project, matching projectRelationshipEdges' own §91.13 discipline), and
// returns a new slice rather than mutating existing.
//
// Note the target is Passthrough.VCard, a Record-level field (sibling of
// Card), not something nested under Card — unlike projectTags/
// projectRelationshipEdges above, which both write into card.* fields.
func projectCustomFields(db *gorm.DB, vcardUID string, existing []contactmodel.JCardProp) []contactmodel.JCardProp {
	if db == nil || vcardUID == "" {
		return existing
	}

	type projectedField struct {
		FieldValue
		Projection string
	}
	var fields []projectedField
	err := db.Table("field_values").
		Select("field_values.*, field_definitions.projection").
		Joins("JOIN field_definitions ON field_definitions.id = field_values.field_definition_id").
		Where("field_values.entity_id = ? AND field_definitions.sensitivity = ? AND field_definitions.projection LIKE ?",
			vcardUID, RelationshipSensitivityNormal, "vcard:%").
		Find(&fields).Error
	if err != nil {
		logger.Warn().Err(err).Str("vcard_uid", vcardUID).Msg("projectCustomFields: failed to load field values")
		return existing
	}
	if len(fields) == 0 {
		return existing
	}

	seen := make(map[string]bool, len(existing))
	for _, p := range existing {
		seen[p.Name] = true
	}

	result := append([]contactmodel.JCardProp(nil), existing...)
	for _, f := range fields {
		name := strings.TrimPrefix(f.Projection, "vcard:")
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, contactmodel.JCardProp{Name: name, Type: "text", Value: f.Value})
	}

	return result
}

// RecordFromContact is the single, shared mapping from a *Contact's current,
// legacy flat/array fields into the neutral contactmodel.Record shape. Both
// BeforeSave (contact.go, runs per-save) and the one-shot
// cmd/backfill-contact-records tool (runs per-row at migration time) call
// this same function — per
// docs/fork-plan/50-integration-and-rebrand.md WP-70's "Shared mapping
// function" note, there must be exactly one Contact->Record mapping, not two
// competing implementations.
//
// Field-by-field mapping decisions cite the corresponding
// docs/fork-plan/20-correspondence.md concept_id/row. RecordFromContact
// never panics, including on a nil receiver or a zero-value *Contact.
//
// photoDir is the configured profile-photo directory (config.Config.
// ProfilePhotoDir), needed to bridge Contact.Photo (a filename on disk,
// relative to photoDir) into a self-contained Card.Media{Kind:"photo"} entry
// — see buildMedia below and docs/fork-plan/50-integration-and-rebrand.md
// WP-73's photo-bridging prerequisite. Pass "" when the disk-backed photo is
// genuinely unavailable/irrelevant to the caller (e.g. a context with no
// filesystem access): buildMedia still falls back to the base64
// PhotoThumbnail in that case, so photo data is only fully lost if neither
// Photo nor PhotoThumbnail is set.
func RecordFromContact(c *Contact, photoDir string) *contactmodel.Record {
	record := &contactmodel.Record{}
	if c == nil {
		return record
	}

	record.UID = c.VCardUID
	record.ETag = c.ETag

	record.Card = contactmodel.Card{
		UID:           c.VCardUID, // 20-correspondence.md "uid" row: Card.UID <- VCardUID
		Name:          buildName(c),
		Nicknames:     buildNicknames(c),
		Organizations: buildOrganizations(c),
		Titles:        buildTitles(c),
		Emails:        buildEmails(c),
		Phones:        buildPhones(c),
		ImppAddresses: buildImpp(c),
		Addresses:     buildAddresses(c),
		Anniversaries: buildAnniversaries(c),
		Links:         buildLinks(c),
		Media:         buildMedia(c, photoDir),
	}

	record.Envelope = contactmodel.CRMEnvelope{
		Circles:            c.Circles,
		HowWeMet:           c.HowWeMet,
		WorkInformation:    c.WorkInformation,
		ContactInformation: c.ContactInformation,
		CustomFields:       c.CustomFields,
	}

	// Gender (c.Gender) is deliberately NOT mapped into Record. This is a
	// flagged, deliberate gap, not an oversight:
	//
	// vCard's legacy GENDER property (vCard 4.0 only, RFC 6350 §6.2.7) and
	// JSContact's speakToAs/GRAMGENDER/PRONOUNS (RFC 9554 §3) are NOT the
	// same concept as this CRM's free-text Gender field — confirmed in
	// docs/specs/rfc6350-baseline.md's closing note ("Meerkat's CRM Gender
	// field ... is deliberately not the same concept as vCard GENDER or
	// JSContact speakToAs. There is no correspondence row for it — it never
	// round-trips through the standardized Card, by design."). So Gender has
	// no home in Card, by design, and 20-correspondence.md has no row for it.
	//
	// That same spec note describes Gender as living in CRMEnvelope, but
	// contactmodel.CRMEnvelope (backend/contactmodel/envelope.go) has no
	// Gender field today — adding one is a contactmodel change, and
	// contactmodel is a P0-locked package outside this WP's file scope
	// (backend/models/, backend/database/migrations/,
	// backend/cmd/backfill-contact-records/ only).
	//
	// Routing Gender through Record.Passthrough instead was considered and
	// rejected: Passthrough (contactmodel/passthrough.go) is specifically for
	// spec-blessed vCard/JSContact escape hatches (RFC 9555 "vCardProps" /
	// unknown-JSContact-property preservation) — Gender is neither, so
	// stuffing it there would misrepresent free-text CRM data as
	// standards-originated. Silently dropping it without saying so would
	// also violate this WP's own losslessness goal.
	//
	// Net effect: Gender does not currently round-trip through Record at
	// all. It remains fully intact on Contact.Gender (untouched, still
	// populated exactly as before) — only the neutral Card/Envelope
	// representation doesn't carry it yet. Revisit once contactmodel gains a
	// CRMEnvelope.Gender field (a separate, contactmodel-scoped change).

	record.Passthrough = buildPassthrough(c)

	return record
}

// buildName maps Firstname/Lastname/MiddleName/Prefix/Suffix into
// Card.Name.Components, per 20-correspondence.md's Name section
// (name.given, name.surname, name.given2, name.title, name.credential rows;
// vCard N order Family;Given;Additional;Prefix;Suffix). Card.Name.Full (vCard
// FN) is deliberately left unset: the legacy Contact has no separate FN
// field to source it from, so DeriveProjection's FN fallback (join of
// given+surname) takes over, matching prior behavior where there was no
// separate FN concept either.
func buildName(c *Contact) *contactmodel.Name {
	if c.Firstname == "" && c.Lastname == "" && c.MiddleName == "" && c.Prefix == "" && c.Suffix == "" {
		return nil
	}
	var components []contactmodel.NameComponent
	add := func(kind, value string) {
		if value != "" {
			components = append(components, contactmodel.NameComponent{Kind: kind, Value: value})
		}
	}
	add("title", c.Prefix)      // name.title: N[3], honorific prefix
	add("given", c.Firstname)   // name.given: N[1]
	add("given2", c.MiddleName) // name.given2: N[2], additional/middle
	add("surname", c.Lastname)  // name.surname: N[0], family
	add("credential", c.Suffix) // name.credential: N[4], honorific suffix
	return &contactmodel.Name{Components: components}
}

// buildNicknames maps the legacy singular Nickname string onto the
// "nickname" row (Card.Nicknames[].Name).
func buildNicknames(c *Contact) []contactmodel.Nickname {
	if c.Nickname == "" {
		return nil
	}
	return []contactmodel.Nickname{{Name: c.Nickname}}
}

// buildOrganizations maps Organization/Department onto the "org"/"org.unit"
// rows (Card.Organizations[].Name / .Units[].Name).
func buildOrganizations(c *Contact) []contactmodel.Organization {
	if c.Organization == "" && c.Department == "" {
		return nil
	}
	org := contactmodel.Organization{Name: c.Organization}
	if c.Department != "" {
		org.Units = []contactmodel.OrgUnit{{Name: c.Department}}
	}
	return []contactmodel.Organization{org}
}

// buildTitles maps JobTitle/Role onto the "title"/"role" rows
// (Card.Titles[kind=title].Name / Card.Titles[kind=role].Name).
func buildTitles(c *Contact) []contactmodel.Title {
	var titles []contactmodel.Title
	if c.JobTitle != "" {
		titles = append(titles, contactmodel.Title{Name: c.JobTitle, Kind: "title"})
	}
	if c.Role != "" {
		titles = append(titles, contactmodel.Title{Name: c.Role, Kind: "role"})
	}
	return titles
}

// buildEmails maps the Emails[] array onto the "email" row
// (Card.Emails[].Address); the legacy free-text Type is preserved verbatim
// in .Label rather than forced into the private/work Contexts enum it
// doesn't necessarily conform to.
//
// Fallback: if Emails is empty but the legacy scalar Email is set (contacts
// created/edited before the Emails array existed, or via any path that only
// ever touched the scalar), synthesize a single entry so the scalar's data
// isn't invisible to the neutral side and the P1 data migration stays
// lossless. This also keeps DeriveProjection's PrimaryEmail equal to the
// existing Email scalar in that case, so BeforeSave's guarded re-assignment
// (see contact.go) is a true no-op rather than a silent blank-out.
func buildEmails(c *Contact) []contactmodel.Email {
	var out []contactmodel.Email
	for _, e := range c.Emails {
		out = append(out, contactmodel.Email{Address: e.Value, Label: e.Type})
	}
	if len(out) == 0 && c.Email != "" {
		out = append(out, contactmodel.Email{Address: c.Email})
	}
	return out
}

// buildPhones mirrors buildEmails for the "phone" row (Card.Phones[].Number),
// including the same legacy-scalar-only fallback.
func buildPhones(c *Contact) []contactmodel.Phone {
	var out []contactmodel.Phone
	for _, p := range c.Phones {
		out = append(out, contactmodel.Phone{Number: p.Value, Label: p.Type})
	}
	if len(out) == 0 && c.Phone != "" {
		out = append(out, contactmodel.Phone{Number: c.Phone})
	}
	return out
}

// buildImpp maps IMPPs[] onto the "impp" row (Card.ImppAddresses[].URI). The
// legacy ContactIMPP.Type holds a service name (e.g. "telegram", "signal"),
// which maps onto OnlineService.Service per this WP's explicit instruction
// (not Label/Contexts, since it identifies the service rather than a
// home/work-style category). There is no legacy scalar analog, so no
// fallback is needed here (unlike Email/Phone/Address).
func buildImpp(c *Contact) []contactmodel.OnlineService {
	var out []contactmodel.OnlineService
	for _, i := range c.IMPPs {
		out = append(out, contactmodel.OnlineService{Service: i.Type, URI: i.Value})
	}
	return out
}

// buildLinks maps URLs[] onto the "link" row (Card.Links[].URI).
func buildLinks(c *Contact) []contactmodel.Resource {
	var out []contactmodel.Resource
	for _, u := range c.URLs {
		out = append(out, contactmodel.Resource{URI: u.Value, Label: u.Type})
	}
	return out
}

// buildMedia bridges the disk-based Contact.Photo/PhotoThumbnail into a
// single Card.Media{Kind:"photo"} entry, per
// docs/fork-plan/50-integration-and-rebrand.md WP-73's photo-bridging
// prerequisite: without this, retiring carddav's legacy mapper for the live
// CardDAV server (and, before that, WP-71's VCF/JSContact export) would
// silently stop serving contact photos, since neither RecordFromContact nor
// ApplyRecordToContact touched Photo/PhotoThumbnail <-> Card.Media at all
// until now.
//
// Encodes the photo as a "data:<mediaType>;base64,<data>" URI — the same
// convention the vcard4/vcard3 adapters already use for an embedded PHOTO
// value (see vcard3/adapter.go's parseDataURI/exportMediaField and
// vcard4/adapter.go's resource import/export) — so this Card.Media entry
// round-trips through either adapter unchanged. photostore.ReadContactPhoto
// prefers the full-resolution file on disk (photoDir/c.Photo) and falls back
// to the base64 PhotoThumbnail, so a photo still round-trips even when
// photoDir is "" (disk access unavailable) as long as a thumbnail exists.
func buildMedia(c *Contact, photoDir string) []contactmodel.Resource {
	uri, mediaType := photostore.ReadContactPhotoDataURI(c.Photo, c.PhotoThumbnail, photoDir)
	if uri == "" {
		return nil
	}
	return []contactmodel.Resource{{Kind: "photo", URI: uri, MediaType: mediaType}}
}

// buildAddresses maps Addresses[] onto the "adr" row (Card.Addresses[]),
// splitting the flat ContactAddress fields into AddressComponent entries.
// AddressComponent.Kind "name" is used for Street per JSContact's
// component-kind registry (10-neutral-model.md), which represents the
// street name (there is no distinct "street" kind — "name" plus "number"
// together cover it, and the legacy ContactAddress has no separate
// street-number field to populate "number" from).
//
// Full is populated via the existing FormatAddress helper (the same
// human-readable line already used to keep the legacy Address scalar in
// sync), matching the vCard LABEL correspondence intent ("adr" row notes).
//
// Fallback: if Addresses is empty but the legacy scalar Address is set,
// synthesize a single entry carrying only Full (no structured components are
// recoverable from an already-flattened string) — same
// losslessness/no-blank-out rationale as buildEmails/buildPhones.
func buildAddresses(c *Contact) []contactmodel.Address {
	var out []contactmodel.Address
	for _, a := range c.Addresses {
		out = append(out, addressFromContactAddress(a))
	}
	if len(out) == 0 && c.Address != "" {
		out = append(out, contactmodel.Address{Full: c.Address})
	}
	return out
}

func addressFromContactAddress(a ContactAddress) contactmodel.Address {
	var components []contactmodel.AddressComponent
	addComp := func(kind, value string) {
		if value != "" {
			components = append(components, contactmodel.AddressComponent{Kind: kind, Value: value})
		}
	}
	addComp("name", a.Street) // JSContact "name" component = street name
	addComp("locality", a.City)
	addComp("region", a.Region)
	addComp("postcode", a.Postal)
	addComp("country", a.Country)

	addr := contactmodel.Address{
		Components: components,
		Full:       FormatAddress(a),
	}
	// Address has no Label field (unlike Email/Phone/Link), so the legacy
	// free-text Type — which isn't guaranteed to be one of the
	// private/work Contexts enum values ctx2type expects — goes into
	// Contexts as the best available home rather than being dropped.
	if a.Type != "" {
		addr.Contexts = []string{a.Type}
	}
	return addr
}

// buildAnniversaries maps Birthday/Anniversary onto the
// "anniversary.birth"/"anniversary.wedding" rows
// (Card.Anniversaries[kind=birth/wedding].Date).
func buildAnniversaries(c *Contact) []contactmodel.Anniversary {
	var out []contactmodel.Anniversary
	if d := parsePartialDateString(c.Birthday); d != nil {
		out = append(out, contactmodel.Anniversary{Kind: "birth", Date: contactmodel.AnniversaryDate{Partial: d}})
	}
	if d := parsePartialDateString(c.Anniversary); d != nil {
		out = append(out, contactmodel.Anniversary{Kind: "wedding", Date: contactmodel.AnniversaryDate{Partial: d}})
	}
	return out
}

// birthdayFormatRE mirrors middleware/validation.go's validateBirthday regex
// exactly (^(--|\d{4}-)\d{2}-\d{2}$, "YYYY-MM-DD" or year-less "--MM-DD").
// models cannot import middleware (middleware already imports models —
// importing back would be an import cycle), so this is intentionally kept
// in sync by hand rather than shared.
//
// Checking this BEFORE attempting to split-and-parse matters: without it,
// any legacy/free-form Birthday string that merely happens to contain two
// dashes (e.g. "15-03-1985", a real DD-MM-YYYY value found in this
// codebase's own controller tests, which the "birthday" validator would
// reject but which existing rows may still contain) would otherwise be
// silently misparsed as YYYY-MM-DD and corrupted (e.g. "0015-03-1985") on
// round-trip. Only strings already in the validator's exact accepted shape
// are parsed; anything else is left as an unparsed passthrough (Birthday
// stays on the legacy scalar, untouched, simply absent from
// Card.Anniversaries for now).
var birthdayFormatRE = regexp.MustCompile(`^(--|\d{4}-)\d{2}-\d{2}$`)

// parsePartialDateString parses the two formats the "birthday" custom
// validator accepts: full "YYYY-MM-DD" or year-less "--MM-DD". Returns nil,
// without error, for anything else (empty string, or legacy/malformed data
// that doesn't match that exact shape) — best effort, never panics. This is
// the inverse of contactmodel.formatAnniversaryDate, so a well-formed input
// round-trips through RecordFromContact -> DeriveProjection back to the
// same string.
func parsePartialDateString(s string) *contactmodel.PartialDate {
	s = strings.TrimSpace(s)
	if s == "" || !birthdayFormatRE.MatchString(s) {
		return nil
	}
	if strings.HasPrefix(s, "--") {
		parts := strings.Split(strings.TrimPrefix(s, "--"), "-")
		if len(parts) != 2 {
			return nil
		}
		month, err1 := strconv.Atoi(parts[0])
		day, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil
		}
		return &contactmodel.PartialDate{Month: &month, Day: &day}
	}
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return nil
	}
	year, err1 := strconv.Atoi(parts[0])
	month, err2 := strconv.Atoi(parts[1])
	day, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return nil
	}
	return &contactmodel.PartialDate{Year: &year, Month: &month, Day: &day}
}

// legacyVCardExtra / legacyVCardField mirror the on-disk JSON shape of
// carddav.VCardExtra (backend/carddav/vcard_mapper.go, read read-only as
// reference for this WP) closely enough for best-effort parsing here.
// backend/models cannot import backend/carddav (carddav already imports
// models — importing back would be an import cycle), so this is an
// independent, hand-kept-in-sync shape rather than a shared type.
//
// Conversion (buildPassthrough below) is deliberately best-effort, not
// perfect-fidelity: VCardExtra is a legacy free-form jCard-ish blob, and per
// this WP's own scope note, forcing it into flawless round-trip fidelity is
// not the point. Specifically:
//   - Params values are flattened into JCardProp.Params as []string (their
//     original shape).
//   - Group (vCard property grouping, e.g. Apple's item1./item2. prefixes
//     used to pair X-ABLabel) is folded into a synthetic "group" param key,
//     since JCardProp has no dedicated field for it.
//   - Type is always reported as "text" — an approximation, since the
//     legacy blob does not record jCard value types.
type legacyVCardExtra struct {
	Properties map[string][]legacyVCardField `json:"properties,omitempty"`
}

type legacyVCardField struct {
	Value  string              `json:"Value"`
	Params map[string][]string `json:"Params,omitempty"`
	Group  string              `json:"Group,omitempty"`
}

// buildPassthrough maps VCardExtra onto Record.Passthrough.VCard
// (the "pt.vcard" row), on a best-effort basis (see type doc comment above).
// If VCardExtra is empty, or doesn't parse as the expected shape, it is left
// out of Passthrough entirely (flagged in this WP's report) rather than
// guessed at.
func buildPassthrough(c *Contact) contactmodel.Passthrough {
	var pt contactmodel.Passthrough
	if c.VCardExtra == "" {
		return pt
	}
	var extra legacyVCardExtra
	if err := json.Unmarshal([]byte(c.VCardExtra), &extra); err != nil {
		return pt
	}
	for name, fields := range extra.Properties {
		for _, f := range fields {
			valueJSON, err := json.Marshal(f.Value)
			if err != nil {
				continue
			}
			params := map[string]any{}
			for k, v := range f.Params {
				params[k] = v
			}
			if f.Group != "" {
				params["group"] = f.Group
			}
			pt.VCard = append(pt.VCard, contactmodel.JCardProp{
				Name:   name,
				Params: params,
				Type:   "text",
				Value:  valueJSON,
			})
		}
	}
	return pt
}
