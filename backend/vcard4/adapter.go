// This file (adapter.go, WP-40b) implements the vCard 4.0 <-> neutral-model
// adapter: contactmodel.Importer/Exporter per docs/fork-plan/00-overview.md
// §0.4, built on top of consts.go/components.go (WP-40a) and driven entirely
// by docs/fork-plan/20-correspondence.md (the oracle — see that file's rows
// for the concept_id backing every field touched below; per §0.6 no mapping
// here is invented beyond what a row states).
//
// Low-level line/property parsing is delegated to
// github.com/emersion/go-vcard (vcard.Decoder/vcard.Encoder over
// vcard.Card = map[string][]*vcard.Field); this file only does the
// neutral<->vCard4 field mapping and value transforms named in 20-correspondence.md
// §20.4.
package vcard4

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"meerkat/contactmodel"
	"meerkat/correspondence"

	vcard "github.com/emersion/go-vcard"
)

// Adapter implements contactmodel.Importer and contactmodel.Exporter for the
// vCard 4.0 wire format.
type Adapter struct{}

var _ contactmodel.Importer = Adapter{}
var _ contactmodel.Exporter = Adapter{}

// ---------------------------------------------------------------------------
// Mapped-property set: derived from correspondence.Load() so it can never
// drift from the oracle. Any vCard property name NOT in this set is, per 0.5
// and 20-correspondence.md's "pt.vcard" row, unknown data preserved via
// Record.Passthrough.VCard rather than silently dropped.
// ---------------------------------------------------------------------------

func mappedV4PropNames() map[string]bool {
	m := make(map[string]bool)
	for _, row := range correspondence.Load() {
		if row.V4Prop == "-" || row.V4Prop == "*verbatim*" {
			continue
		}
		name := row.V4Prop
		if i := strings.IndexByte(name, '['); i >= 0 {
			name = name[:i]
		}
		m[strings.ToUpper(name)] = true
	}
	return m
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

// Import parses raw vCard 4.0 bytes into the neutral model. Per 0.5, it
// returns an error only when raw is not validly framed vCard at all (no
// BEGIN/END, malformed structure); every other gap (unmappable/unknown data)
// is preserved via passthrough and reported as a Diagnostic.
func (Adapter) Import(raw []byte) (*contactmodel.Record, []contactmodel.Diagnostic, error) {
	dec := vcard.NewDecoder(bytes.NewReader(raw))
	card, err := dec.Decode()
	if err != nil {
		return nil, nil, fmt.Errorf("vcard4: malformed vCard: %w", err)
	}

	rec := &contactmodel.Record{}
	var diags []contactmodel.Diagnostic

	importIdentity(card, rec)
	importName(card, rec, &diags)
	importNicknames(card, rec)
	groupToOrgID := importOrganizations(card, rec)
	importTitles(card, rec, groupToOrgID)
	importEmails(card, rec)
	importPhones(card, rec)
	importOnlineServices(card, rec)
	importAddresses(card, rec, &diags)
	importAnniversaries(card, rec, &diags)
	importSpeakToAs(card, rec, &diags)
	importPersonalInfo(card, rec)
	importNotes(card, rec)
	importKeywords(card, rec)
	importResources(card, rec)
	importLangsRelatedMembers(card, rec)
	importPassthrough(card, rec)

	return rec, diags, nil
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// geoSplice records a post-encode string substitution. See exportAddresses's
// comment for why this exists: it works around a confirmed limitation of the
// pinned github.com/emersion/go-vcard version rather than forking it.
type geoSplice struct{ find, replace string }

// Export renders the neutral model as vCard 4.0 bytes. Per 0.5, it never
// errors on a field with no vCard4 home (drop-with-Diagnostic instead); it
// only errors if the underlying encoder itself fails (e.g. VERSION missing,
// which cannot happen here since this function always sets it).
func (Adapter) Export(rec *contactmodel.Record) ([]byte, []contactmodel.Diagnostic, error) {
	if rec == nil {
		return nil, nil, fmt.Errorf("vcard4: nil record")
	}

	card := vcard.Card{}
	card.SetValue(PropVersion, "4.0")

	var diags []contactmodel.Diagnostic
	var splices []geoSplice

	exportIdentity(rec, card)
	exportName(rec, card, &diags)
	exportNicknames(rec, card)
	orgIDToGroup := exportOrganizations(rec, card)
	exportTitles(rec, card, orgIDToGroup)
	exportEmails(rec, card)
	exportPhones(rec, card)
	exportOnlineServices(rec, card, &diags)
	exportAddresses(rec, card, &splices)
	exportAnniversaries(rec, card, &diags)
	exportSpeakToAs(rec, card)
	exportPersonalInfo(rec, card)
	exportNotes(rec, card)
	exportKeywords(rec, card)
	exportResources(rec, card)
	exportLangsRelatedMembers(rec, card)
	exportPassthrough(rec, card, mappedV4PropNames())

	var buf bytes.Buffer
	if err := vcard.NewEncoder(&buf).Encode(card); err != nil {
		return nil, diags, fmt.Errorf("vcard4: encode: %w", err)
	}
	out := buf.Bytes()
	for _, sp := range splices {
		out = bytes.Replace(out, []byte(sp.find), []byte(sp.replace), 1)
	}
	return out, diags, nil
}

// ---------------------------------------------------------------------------
// Shared field/param helpers
// ---------------------------------------------------------------------------

func paramValue(f *vcard.Field, key string) string {
	if f == nil || f.Params == nil {
		return ""
	}
	return f.Params.Get(key)
}

func paramAll(f *vcard.Field, key string) []string {
	if f == nil || f.Params == nil {
		return nil
	}
	return f.Params[key]
}

func paramIsTrue(f *vcard.Field, key string) bool {
	return strings.EqualFold(paramValue(f, key), "TRUE")
}

// paramJoined reassembles a parameter's full value when it contains a
// literal comma. go-vcard's decoder (parseParamValues in decoder.go) always
// splits a parameter value on "," — even when the value is quoted — since
// its `values = strings.Split(vs, ",")` call runs unconditionally after
// either the quoted or unquoted branch. A plain paramValue/Params.Get would
// therefore silently return only the text before the first comma (e.g.
// "geo:12.3457" instead of "geo:12.3457,78.910" for a GEO param). This
// rejoins the split pieces, which is safe here because nothing else in this
// adapter emits a genuinely multi-valued (repeated) instance of these single
// comma-bearing-value parameters (GEO).
func paramJoined(f *vcard.Field, key string) string {
	vals := paramAll(f, key)
	if len(vals) == 0 {
		return ""
	}
	return strings.Join(vals, ",")
}

func setParam(f *vcard.Field, key, val string) {
	if val == "" {
		return
	}
	if f.Params == nil {
		f.Params = vcard.Params{}
	}
	f.Params.Set(key, val)
}

// quotedParam wraps raw in literal DQUOTE characters so that, after passing
// through go-vcard's encoder (which applies blanket TEXT-style escaping of
// "\", "," and "\n" to every parameter value but never adds quoting itself),
// the wire output is an RFC 6350 quoted-string parameter value. This is
// required for any parameter value containing ':' or ';' (which would
// otherwise be misparsed as the end of the parameter section — go-vcard's
// unquoted-parameter scan is a naive IndexAny(s, ";:") with no escaping
// awareness). It is safe to use ONLY when raw contains no comma: go-vcard's
// decoder always splits a parameter's value on "," (decoder.go's
// parseParamValues calls strings.Split unconditionally, even in the quoted
// branch after unquoting), so a comma-bearing value silently comes back as
// multiple param values; if that comma is instead protected by pre-escaping
// it, go-vcard's decoder chokes trying to unescape "\," via
// strconv.UnquoteChar (not a recognized Go escape) and drops the whole line.
// See exportAddresses's geoSplice mechanism for the (mandatory-comma) GEO
// case this helper cannot be used for, and paramJoined for how the
// unconditional-comma-split is worked around on the read side.
func quotedParam(raw string) string { return `"` + raw + `"` }

func prefOf(f *vcard.Field) *int {
	v := paramValue(f, ParamPref)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &n
}

func setPref(f *vcard.Field, pref *int) {
	if pref == nil {
		return
	}
	setParam(f, ParamPref, strconv.Itoa(*pref))
}

// ctx2type (20.4): Contexts private<->home, work<->work; 9554 also adds
// billing/delivery (identity both ways) for addresses.
var typeTokenToContext = map[string]string{
	"home": "private", "work": "work", "billing": "billing", "delivery": "delivery",
}
var contextToTypeToken = map[string]string{
	"private": "home", "work": "work", "billing": "billing", "delivery": "delivery",
}

// splitTypeTokens classifies a field's TYPE tokens into recognized Contexts
// (home/work/billing/delivery) vs. everything else ("rest" — e.g. phone
// feature tokens per feat2type, which the caller inspects separately).
func splitTypeTokens(f *vcard.Field) (contexts []string, rest []string) {
	for _, tok := range paramAll(f, ParamType) {
		lt := strings.ToLower(tok)
		if ctx, ok := typeTokenToContext[lt]; ok {
			contexts = append(contexts, ctx)
		} else {
			rest = append(rest, lt)
		}
	}
	return
}

func contextsToTypeTokens(contexts []string) []string {
	out := make([]string, 0, len(contexts))
	for _, c := range contexts {
		if tok, ok := contextToTypeToken[strings.ToLower(c)]; ok {
			out = append(out, tok)
		} else {
			out = append(out, strings.ToLower(c))
		}
	}
	return out
}

func addTypeTokens(f *vcard.Field, tokens ...string) {
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		if f.Params == nil {
			f.Params = vcard.Params{}
		}
		f.Params.Add(ParamType, tok)
	}
}

// idOrSynthetic returns id, or a short deterministic synthetic id
// (prefix + 1-based index) if id is empty — 30-adapters.md §30.B: "carry
// PROP-ID = element.ID (generate if empty) so identity survives" a round
// trip.
func idOrSynthetic(id, prefix string, idx int) string {
	if id != "" {
		return id
	}
	return fmt.Sprintf("%s%d", prefix, idx+1)
}

// fieldToJCardProp captures an arbitrary vCard field (unknown property, or a
// known property being routed to passthrough for a documented sub-case) as
// a contactmodel.JCardProp (RFC 9555 vCardProps shape: [name, params, type,
// value]).
func fieldToJCardProp(name string, f *vcard.Field) contactmodel.JCardProp {
	var params map[string]any
	if len(f.Params) > 0 {
		params = make(map[string]any, len(f.Params))
		for k, vs := range f.Params {
			params[strings.ToLower(k)] = vs
		}
	}
	if f.Group != "" {
		if params == nil {
			params = map[string]any{}
		}
		params["group"] = f.Group
	}
	valJSON, _ := json.Marshal(f.Value)
	return contactmodel.JCardProp{
		Name:   strings.ToLower(name),
		Params: params,
		Type:   "text",
		Value:  json.RawMessage(valJSON),
	}
}

// jCardPropToField is fieldToJCardProp's inverse, used when re-emitting
// Passthrough.VCard entries on export.
func jCardPropToField(p contactmodel.JCardProp) (name string, f *vcard.Field) {
	f = &vcard.Field{}
	var val string
	if err := json.Unmarshal(p.Value, &val); err == nil {
		f.Value = val
	} else {
		f.Value = string(p.Value)
	}
	for k, v := range p.Params {
		if strings.EqualFold(k, "group") {
			if s, ok := v.(string); ok {
				f.Group = s
			}
			continue
		}
		if f.Params == nil {
			f.Params = vcard.Params{}
		}
		switch vv := v.(type) {
		case string:
			f.Params.Add(strings.ToUpper(k), vv)
		case []string:
			for _, s := range vv {
				f.Params.Add(strings.ToUpper(k), s)
			}
		case []any:
			for _, item := range vv {
				if s, ok := item.(string); ok {
					f.Params.Add(strings.ToUpper(k), s)
				}
			}
		}
	}
	name = strings.ToUpper(p.Name)
	return
}

// ---------------------------------------------------------------------------
// ts_rfc3339 transform (updated/created/note.created)
// ---------------------------------------------------------------------------

const vcardTimestampLayout = "20060102T150405Z"

func rfc3339ToVCardTimestamp(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.UTC().Format(vcardTimestampLayout)
}

func vcardTimestampToRFC3339(v string) string {
	t, err := time.Parse(vcardTimestampLayout, v)
	if err != nil {
		return v
	}
	return t.UTC().Format(time.RFC3339)
}

// ---------------------------------------------------------------------------
// Identity / meta: uid, kind, prodid, updated, created, language.
//
// "language" was corrected in 20-correspondence.md (was wrongly v4_prop
// "-"): RFC 9554 §3.3 defines vCard4 LANGUAGE as a dedicated property —
// "default language of human-readable values" — a direct 1:1 match for
// Card.Language (transform identity). v3 (RFC 2426) predates this property
// entirely, so it stays "-" and warn-drops on 3.0 export only (WP-50's
// concern, not this adapter's).
// ---------------------------------------------------------------------------

func importIdentity(card vcard.Card, rec *contactmodel.Record) {
	if v := card.Value(PropUID); v != "" {
		rec.Card.UID = v
	}
	if v := card.Value(PropKind); v != "" {
		rec.Card.Kind = v
	}
	if v := card.Value(PropProdid); v != "" {
		rec.Card.ProdID = v
	}
	if v := card.Value(PropRev); v != "" {
		rec.Card.Updated = &contactmodel.Timestamp{UTC: vcardTimestampToRFC3339(v)}
	}
	if v := card.Value(PropCreated); v != "" {
		rec.Card.Created = &contactmodel.Timestamp{UTC: vcardTimestampToRFC3339(v)}
	}
	if v := card.Value(PropLanguage); v != "" {
		rec.Card.Language = v
	}
}

func exportIdentity(rec *contactmodel.Record, card vcard.Card) {
	if rec.Card.UID != "" {
		card.SetValue(PropUID, rec.Card.UID)
	}
	if rec.Card.Kind != "" {
		card.SetValue(PropKind, rec.Card.Kind)
	}
	if rec.Card.ProdID != "" {
		card.SetValue(PropProdid, rec.Card.ProdID)
	}
	if rec.Card.Updated != nil {
		card.SetValue(PropRev, rfc3339ToVCardTimestamp(rec.Card.Updated.UTC))
	}
	if rec.Card.Created != nil {
		card.SetValue(PropCreated, rfc3339ToVCardTimestamp(rec.Card.Created.UTC))
	}
	if rec.Card.Language != "" {
		card.SetValue(PropLanguage, rec.Card.Language)
	}
}

// ---------------------------------------------------------------------------
// Name: FN (+DERIVED), N components (name.surname..generation), N phonetic
// variant (name.phonetic).
//
// JUDGMENT CALL (flagged per docs/fork-plan/60-review-gates.md §60.4 — not a
// silent invention): the "name.phonetic" row's neutral_path names only
// Card.Name.PhoneticScript, but its v4_params column also scopes PHONETIC
// and SCRIPT (and ALTID for pairing). Two neutral fields exist for exactly
// this data (Name.PhoneticSystem, Name.PhoneticScript) and per-component
// phonetic text has its own home too (NameComponent.Phonetic) — a pattern
// already established by the "note" row in this same table, whose params
// (AUTHOR/AUTHOR-NAME/CREATED) populate Note's sibling Author/Created fields
// despite neutral_path naming only Note.Note. Following that precedent:
// PHONETIC param -> Name.PhoneticSystem, SCRIPT param -> Name.PhoneticScript,
// and the phonetic N line's own components -> the matching (by kind)
// NameComponent.Phonetic. See fixture phonetic-n.v4.vcf.
// ---------------------------------------------------------------------------

// nComponentKindOrder mirrors N's wire order (RFC 9554 §2.2) to the
// NameComponent.Kind vocabulary (10-neutral-model.md / model.go): Family ->
// surname, Given -> given, Additional -> given2, Prefix -> title, Suffix ->
// credential, Surname2 -> surname2, Generation -> generation.
func importName(card vcard.Card, rec *contactmodel.Record, diags *[]contactmodel.Diagnostic) {
	nFields := card[PropN]
	fn := card.Get(PropFN)
	if len(nFields) == 0 && fn == nil {
		return
	}

	var name contactmodel.Name
	var primary, phonetic *vcard.Field
	for _, f := range nFields {
		if paramValue(f, ParamPhonetic) != "" {
			if phonetic == nil {
				phonetic = f
			}
			continue
		}
		if primary == nil {
			primary = f
		}
	}

	addComp := func(kind, val string) {
		if val != "" {
			name.Components = append(name.Components, contactmodel.NameComponent{Kind: kind, Value: val})
		}
	}
	if primary != nil {
		nc := disassembleN(primary.Value)
		addComp("surname", nc.Family)
		addComp("given", nc.Given)
		addComp("given2", nc.Additional)
		addComp("title", nc.Prefix)
		addComp("credential", nc.Suffix)
		addComp("surname2", nc.Surname2)
		addComp("generation", nc.Generation)
	}

	if phonetic != nil {
		name.PhoneticSystem = paramValue(phonetic, ParamPhonetic)
		name.PhoneticScript = paramValue(phonetic, ParamScript)
		pc := disassembleN(phonetic.Value)
		setPhon := func(kind, val string) {
			if val == "" {
				return
			}
			for i := range name.Components {
				if name.Components[i].Kind == kind {
					name.Components[i].Phonetic = val
					return
				}
			}
		}
		setPhon("surname", pc.Family)
		setPhon("given", pc.Given)
		setPhon("given2", pc.Additional)
		setPhon("title", pc.Prefix)
		setPhon("credential", pc.Suffix)
		setPhon("surname2", pc.Surname2)
		setPhon("generation", pc.Generation)
	}

	if fn != nil {
		if paramIsTrue(fn, ParamDerived) {
			// 30-adapters.md §30.B: never import a DERIVED value as
			// authoritative. It's otherwise-mapped (name.full), so per 0.5
			// it is simply not imported (not passthrough either) — an info
			// Diagnostic notes the drop.
			*diags = append(*diags, contactmodel.Diagnostic{
				Severity: "info", Concept: "name.full",
				Message: "vCard FN was DERIVED=TRUE; not imported as authoritative Name.Full",
			})
		} else {
			name.Full = fn.Value
		}
	}

	if len(name.Components) > 0 || name.Full != "" || name.PhoneticScript != "" || name.PhoneticSystem != "" {
		rec.Card.Name = &name
	}
}

// deriveFullName implements the "derive from components" side of the
// name.full row (RFC 9554 §4.4 / RFC 9555 §3.1 — neither specifies an exact
// join algorithm). This order (Prefix, Given, Additional, Family, Suffix,
// Surname2, Generation; skip-empty; space-joined) is chosen because it
// exactly reproduces the golden fixture derived-fn.v4.vcf's worked example
// (N:;John;Quinlan;Mr.; -> FN;DERIVED=TRUE:Mr. John Quinlan).
func deriveFullName(nc NComponents) string {
	parts := []string{nc.Prefix, nc.Given, nc.Additional, nc.Family, nc.Suffix, nc.Surname2, nc.Generation}
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, " ")
}

func exportName(rec *contactmodel.Record, card vcard.Card, diags *[]contactmodel.Diagnostic) {
	var n contactmodel.Name
	if rec.Card.Name != nil {
		n = *rec.Card.Name
	}

	compVal := func(kind string) string {
		for _, c := range n.Components {
			if c.Kind == kind {
				return c.Value
			}
		}
		return ""
	}
	nc := NComponents{
		Family: compVal("surname"), Given: compVal("given"), Additional: compVal("given2"),
		Prefix: compVal("title"), Suffix: compVal("credential"),
		Surname2: compVal("surname2"), Generation: compVal("generation"),
	}
	hasComponents := len(n.Components) > 0
	if hasComponents {
		card.Add(PropN, &vcard.Field{Value: assembleN(nc)})
	}

	full := n.Full
	derived := false
	if full == "" && hasComponents {
		full = deriveFullName(nc)
		derived = full != ""
	}
	if full == "" {
		*diags = append(*diags, contactmodel.Diagnostic{
			Severity: "warn", Concept: "name.full",
			Message: "no Name.Full and no components to derive FN from; emitting empty FN (vCard 4.0 requires FN)",
		})
	}
	card.Add(PropFN, &vcard.Field{Value: full})
	if derived {
		setParam(card.Get(PropFN), ParamDerived, "TRUE")
	}

	if n.PhoneticScript != "" || n.PhoneticSystem != "" {
		var pc NComponents
		var hasAnyPhon bool
		setIfPhon := func(kind string, dst *string) {
			for _, c := range n.Components {
				if c.Kind == kind && c.Phonetic != "" {
					*dst = c.Phonetic
					hasAnyPhon = true
				}
			}
		}
		setIfPhon("surname", &pc.Family)
		setIfPhon("given", &pc.Given)
		setIfPhon("given2", &pc.Additional)
		setIfPhon("title", &pc.Prefix)
		setIfPhon("credential", &pc.Suffix)
		setIfPhon("surname2", &pc.Surname2)
		setIfPhon("generation", &pc.Generation)
		if hasAnyPhon {
			pf := &vcard.Field{Value: assembleN(pc)}
			setParam(pf, ParamAltid, "1")
			setParam(pf, ParamPhonetic, n.PhoneticSystem)
			setParam(pf, ParamScript, n.PhoneticScript)
			card.Add(PropN, pf)
			if hasComponents {
				setParam(card[PropN][0], ParamAltid, "1")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Nicknames
// ---------------------------------------------------------------------------

func importNicknames(card vcard.Card, rec *contactmodel.Record) {
	idx := 0
	for _, f := range card[PropNickname] {
		ctx, _ := splitTypeTokens(f)
		pref := prefOf(f)
		for _, nm := range strings.Split(f.Value, ",") {
			nm = strings.TrimSpace(nm)
			if nm == "" {
				continue
			}
			rec.Card.Nicknames = append(rec.Card.Nicknames, contactmodel.Nickname{
				ID: idOrSynthetic(propID(f), "nick", idx), Name: nm, Contexts: ctx, Pref: pref,
			})
			idx++
		}
	}
}

func exportNicknames(rec *contactmodel.Record, card vcard.Card) {
	for i, n := range rec.Card.Nicknames {
		f := &vcard.Field{Value: n.Name}
		addTypeTokens(f, contextsToTypeTokens(n.Contexts)...)
		setPref(f, n.Pref)
		setPropID(f, idOrSynthetic(n.ID, "nick", i))
		card.Add(PropNickname, f)
	}
}

// ---------------------------------------------------------------------------
// Organizations / Titles (org, org.unit, title, role) — including the
// organizationId <-> shared GROUP derivation (20.7, fixture title-role).
// ---------------------------------------------------------------------------

func importOrganizations(card vcard.Card, rec *contactmodel.Record) map[string]string {
	groupToOrgID := make(map[string]string)
	for i, f := range card[PropOrg] {
		parts := splitComponents(f.Value)
		var units []contactmodel.OrgUnit
		for _, u := range parts[1:] {
			if u != "" {
				units = append(units, contactmodel.OrgUnit{Name: u})
			}
		}
		id := idOrSynthetic(propID(f), "org", i)
		rec.Card.Organizations = append(rec.Card.Organizations, contactmodel.Organization{
			ID: id, Name: component(parts, 0), Units: units,
		})
		if f.Group != "" {
			groupToOrgID[strings.ToLower(f.Group)] = id
		}
	}
	return groupToOrgID
}

func importTitles(card vcard.Card, rec *contactmodel.Record, groupToOrgID map[string]string) {
	idx := 0
	add := func(kind string, f *vcard.Field) {
		orgID := ""
		if f.Group != "" {
			orgID = groupToOrgID[strings.ToLower(f.Group)]
		}
		rec.Card.Titles = append(rec.Card.Titles, contactmodel.Title{
			ID: idOrSynthetic(propID(f), "title", idx), Name: f.Value, Kind: kind, OrganizationID: orgID,
		})
		idx++
	}
	for _, f := range card[PropTitle] {
		add("title", f)
	}
	for _, f := range card[PropRole] {
		add("role", f)
	}
}

func unitNames(units []contactmodel.OrgUnit) []string {
	out := make([]string, len(units))
	for i, u := range units {
		out[i] = u.Name
	}
	return out
}

// exportOrganizations returns a map from Organization.ID to the synthetic
// GROUP token assigned to it, for every organization referenced by at least
// one Title.OrganizationID (20.7: "emit a shared groupN. prefix on the
// TITLE/ROLE and its ORG").
func exportOrganizations(rec *contactmodel.Record, card vcard.Card) map[string]string {
	referenced := make(map[string]bool)
	for _, t := range rec.Card.Titles {
		if t.OrganizationID != "" {
			referenced[t.OrganizationID] = true
		}
	}

	orgIDToGroup := make(map[string]string)
	groupN := 0
	for i, o := range rec.Card.Organizations {
		id := idOrSynthetic(o.ID, "org", i)
		group := ""
		if referenced[o.ID] || referenced[id] {
			groupN++
			group = fmt.Sprintf("group%d", groupN)
			orgIDToGroup[o.ID] = group
			orgIDToGroup[id] = group
		}
		parts := append([]string{o.Name}, unitNames(o.Units)...)
		f := &vcard.Field{Value: joinComponents(parts), Group: group}
		setPropID(f, id)
		card.Add(PropOrg, f)
	}
	return orgIDToGroup
}

func exportTitles(rec *contactmodel.Record, card vcard.Card, orgIDToGroup map[string]string) {
	for i, t := range rec.Card.Titles {
		prop := PropTitle
		if t.Kind == "role" {
			prop = PropRole
		}
		f := &vcard.Field{Value: t.Name}
		if g, ok := orgIDToGroup[t.OrganizationID]; ok && t.OrganizationID != "" {
			f.Group = g
		}
		setPropID(f, idOrSynthetic(t.ID, "title", i))
		card.Add(prop, f)
	}
}

// ---------------------------------------------------------------------------
// Emails / Phones
// ---------------------------------------------------------------------------

func importEmails(card vcard.Card, rec *contactmodel.Record) {
	for i, f := range card[PropEmail] {
		ctx, _ := splitTypeTokens(f)
		rec.Card.Emails = append(rec.Card.Emails, contactmodel.Email{
			ID: idOrSynthetic(propID(f), "email", i), Address: f.Value, Contexts: ctx, Pref: prefOf(f),
		})
	}
}

func exportEmails(rec *contactmodel.Record, card vcard.Card) {
	for i, e := range rec.Card.Emails {
		f := &vcard.Field{Value: e.Address}
		addTypeTokens(f, contextsToTypeTokens(e.Contexts)...)
		setPref(f, e.Pref)
		setPropID(f, idOrSynthetic(e.ID, "email", i))
		card.Add(PropEmail, f)
	}
}

// feat2type (20.4): Phone.Features <-> TEL TYPE tokens; vCard4/neutral both
// use "cell" (the JSContact "mobile" translation is a jscontact-adapter
// concern, not this one), so this direction is identity — any TYPE token not
// recognized as a context (home/work) is treated as a feature verbatim.
func importPhones(card vcard.Card, rec *contactmodel.Record) {
	for i, f := range card[PropTel] {
		ctx, feat := splitTypeTokens(f)
		rec.Card.Phones = append(rec.Card.Phones, contactmodel.Phone{
			ID: idOrSynthetic(propID(f), "phone", i), Number: f.Value, Features: feat, Contexts: ctx, Pref: prefOf(f),
		})
	}
}

func exportPhones(rec *contactmodel.Record, card vcard.Card) {
	for i, p := range rec.Card.Phones {
		f := &vcard.Field{Value: p.Number}
		addTypeTokens(f, contextsToTypeTokens(p.Contexts)...)
		addTypeTokens(f, p.Features...)
		setPref(f, p.Pref)
		setPropID(f, idOrSynthetic(p.ID, "phone", i))
		card.Add(PropTel, f)
	}
}

// ---------------------------------------------------------------------------
// Online services (impp, social) — 20.7: choose SOCIALPROFILE vs IMPP on
// export by presence of Service/User.
// ---------------------------------------------------------------------------

// serviceTypeAndUsername reads the SERVICE-TYPE/USERNAME parameters into the
// neutral OnlineService.Service/.User fields. RFC 9554 §4.9/§4.10 state both
// params "MAY be specified on an IMPP or a SOCIALPROFILE property" — they
// land on the same neutral fields regardless of which property carried them,
// so both importOnlineServices branches below share this one reader instead
// of each re-deriving it (SERVICE-TYPE/USERNAME were previously only read on
// the SOCIALPROFILE branch, silently losing both params on IMPP).
func serviceTypeAndUsername(f *vcard.Field) (service, user string) {
	return paramValue(f, ParamServiceType), paramValue(f, ParamUsername)
}

// importOnlineServices routes IMPP into Card.ImppAddresses and SOCIALPROFILE
// into Card.SocialProfiles. vCard import is always unambiguous about
// provenance (20-correspondence.md §20.7's "three-array design") — which
// property produced a value IS which array it belongs in, so no per-element
// tag is needed here. Card.OtherOnlineServices is never populated by vCard
// import (only JSContact import, with no vCardName hint, produces it).
func importOnlineServices(card vcard.Card, rec *contactmodel.Record) {
	idx := 0
	for _, f := range card[PropImpp] {
		ctx, _ := splitTypeTokens(f)
		service, user := serviceTypeAndUsername(f)
		rec.Card.ImppAddresses = append(rec.Card.ImppAddresses, contactmodel.OnlineService{
			ID: idOrSynthetic(propID(f), "svc", idx), URI: f.Value,
			Service: service, User: user, Contexts: ctx, Pref: prefOf(f),
		})
		idx++
	}
	for _, f := range card[PropSocialprofile] {
		service, user := serviceTypeAndUsername(f)
		os := contactmodel.OnlineService{
			ID:      idOrSynthetic(propID(f), "svc", idx),
			Service: service,
		}
		if strings.EqualFold(paramValue(f, ParamValue), "text") {
			os.User = f.Value
		} else {
			os.URI = f.Value
			os.User = user
		}
		ctx, _ := splitTypeTokens(f)
		os.Contexts = ctx
		os.Pref = prefOf(f)
		rec.Card.SocialProfiles = append(rec.Card.SocialProfiles, os)
		idx++
	}
}

// exportOnlineServices emits Card.ImppAddresses as IMPP and
// Card.SocialProfiles as SOCIALPROFILE — always, unconditionally, since which
// neutral array an entry lives in already IS the provenance decision (20.7).
// Card.OtherOnlineServices has no vCard export at all (neither IMPP nor
// SOCIALPROFILE is a safe default guess for genuinely unclassified data): each
// entry present is reported via a warn Diagnostic and not emitted.
func exportOnlineServices(rec *contactmodel.Record, card vcard.Card, diags *[]contactmodel.Diagnostic) {
	for i, os := range rec.Card.ImppAddresses {
		f := &vcard.Field{Value: os.URI}
		if os.User != "" {
			setParam(f, ParamUsername, os.User)
		}
		if os.Service != "" {
			setParam(f, ParamServiceType, os.Service)
		}
		addTypeTokens(f, contextsToTypeTokens(os.Contexts)...)
		setPref(f, os.Pref)
		setPropID(f, idOrSynthetic(os.ID, "svc", i))
		card.Add(PropImpp, f)
	}
	for i, os := range rec.Card.SocialProfiles {
		f := &vcard.Field{}
		if os.URI != "" {
			f.Value = os.URI
			if os.User != "" {
				setParam(f, ParamUsername, os.User)
			}
		} else {
			f.Value = os.User
			setParam(f, ParamValue, "text")
		}
		if os.Service != "" {
			setParam(f, ParamServiceType, os.Service)
		}
		addTypeTokens(f, contextsToTypeTokens(os.Contexts)...)
		setPref(f, os.Pref)
		setPropID(f, idOrSynthetic(os.ID, "svc", i))
		card.Add(PropSocialprofile, f)
	}
	for _, os := range rec.Card.OtherOnlineServices {
		msg := "unclassified online service (Card.OtherOnlineServices) has no safe default vCard property (neither IMPP nor SOCIALPROFILE) and was not exported"
		if os.URI != "" {
			msg += " (uri: " + os.URI + ")"
		}
		*diags = append(*diags, contactmodel.Diagnostic{
			Severity: "warn", Concept: "impp", Message: msg,
		})
	}
}

// ---------------------------------------------------------------------------
// Addresses (adr, adr.geo, adr.tz)
// ---------------------------------------------------------------------------

// adrComponentsToKindValues maps the 18 positional AdrComponents (RFC 9554
// §2.1 / docs/specs/rfc9554-vcard-extensions.md §3) onto
// []contactmodel.AddressComponent by kind. The legacy component 2 ("Ext",
// extended address) has no corresponding neutral kind in the registry (only
// position 3 "Street" and position 12 "StreetName" do, both -> kind "name")
// — a non-empty Ext is therefore reported via Diagnostic and dropped, per
// 0.5 (rather than inventing a new kind value, which 60.4 explicitly flags
// as a stop-worthy move).
//
// JUDGMENT CALL: when both legacy Street (pos 3) and modern StreetName
// (pos 12) are present (as in fixture adr-expanded.v4.vcf, where they
// represent the same street redundantly — Street is the combined
// "Number Name" form), the fine-grained modern field is preferred and the
// legacy one is not separately re-added as a duplicate "name" component; the
// legacy Street field is instead reconstructed on export (see
// kindValuesToAdrComponents) by combining Number+StreetName for backward
// compatibility with legacy-only parsers. This is not a documented
// transform detail (20.4's adr_components entry does not spell out a merge
// algorithm for the two schemes coexisting); it is grounded directly in the
// one worked example we have and is round-trip lossless for that fixture.
func adrComponentsToKindValues(ac AdrComponents, diags *[]contactmodel.Diagnostic) []contactmodel.AddressComponent {
	var out []contactmodel.AddressComponent
	add := func(kind, val string) {
		if val != "" {
			out = append(out, contactmodel.AddressComponent{Kind: kind, Value: val})
		}
	}
	add("postOfficeBox", ac.POBox)
	if ac.Ext != "" {
		*diags = append(*diags, contactmodel.Diagnostic{
			Severity: "warn", Concept: "adr",
			Message: "ADR legacy extended-address component (position 2) has no neutral kind; dropped",
		})
	}
	if ac.StreetName != "" {
		add("name", ac.StreetName)
	} else {
		add("name", ac.Street)
	}
	add("locality", ac.Locality)
	add("region", ac.Region)
	add("postcode", ac.Code)
	add("country", ac.Country)
	add("room", ac.Room)
	add("apartment", ac.Apartment)
	add("floor", ac.Floor)
	add("number", ac.Number)
	add("building", ac.Building)
	add("block", ac.Block)
	add("subdistrict", ac.Subdistrict)
	add("district", ac.District)
	add("landmark", ac.Landmark)
	add("direction", ac.Direction)
	return out
}

func kindValuesToAdrComponents(comps []contactmodel.AddressComponent) AdrComponents {
	get := func(kind string) string {
		for _, c := range comps {
			if c.Kind == kind {
				return c.Value
			}
		}
		return ""
	}
	a := AdrComponents{
		POBox:       get("postOfficeBox"),
		Locality:    get("locality"),
		Region:      get("region"),
		Code:        get("postcode"),
		Country:     get("country"),
		Room:        get("room"),
		Apartment:   get("apartment"),
		Floor:       get("floor"),
		Number:      get("number"),
		StreetName:  get("name"),
		Building:    get("building"),
		Block:       get("block"),
		Subdistrict: get("subdistrict"),
		District:    get("district"),
		Landmark:    get("landmark"),
		Direction:   get("direction"),
	}
	if a.StreetName != "" {
		if a.Number != "" {
			a.Street = a.Number + " " + a.StreetName
		} else {
			a.Street = a.StreetName
		}
	}
	return a
}

func importAddresses(card vcard.Card, rec *contactmodel.Record, diags *[]contactmodel.Diagnostic) {
	for i, f := range card[PropAdr] {
		ac := disassembleAdr(f.Value)
		ctx, _ := splitTypeTokens(f)
		addr := contactmodel.Address{
			ID:          idOrSynthetic(propID(f), "addr", i),
			Components:  adrComponentsToKindValues(ac, diags),
			CountryCode: paramValue(f, ParamCC),
			Coordinates: paramJoined(f, ParamGeo),
			TimeZone:    paramValue(f, ParamTz),
			Contexts:    ctx,
			Pref:        prefOf(f),
			Full:        paramValue(f, ParamLabel),
		}
		rec.Card.Addresses = append(rec.Card.Addresses, addr)
	}
}

// exportAddresses emits ADR fields. The GEO param (when Coordinates is set)
// cannot be produced directly via go-vcard's Params/Encoder mechanism, for
// two independent, confirmed reasons (a geo: URI, RFC 5870, always contains
// a comma — "geo:lat,lon" — and always contains a colon):
//
//  1. Comma: decoder.go's parseParamValues calls `strings.Split(vs, ",")`
//     UNCONDITIONALLY on a parameter's value — even in the quoted branch,
//     after the quotes have already been stripped. So a correctly quoted
//     `GEO="geo:12.3457,78.910"` still decodes into TWO param values,
//     ["geo:12.3457", "78.910"], and Params.Get returns only the first.
//     (importAddresses works around this on the read side with
//     paramJoined, which rejoins the split pieces — but that is only
//     possible because we control the read side; it doesn't help emitting
//     a value in the first place.)
//  2. Colon: an UNQUOTED value containing ':' is misparsed by the
//     unquoted-value scanner (a naive IndexAny(s, ";:") with no escaping
//     awareness) as ending the parameter list early — so quoting is
//     mandatory to even get the ':' through.
//  3. But quoting doesn't fix (1), and if we instead try to make the comma
//     "safe" by pre-escaping it before go-vcard's own encoder additionally
//     backslash-escapes it (encoder.go's formatValue, designed for TEXT
//     property values, is applied to parameter values with no distinction),
//     the result is a quoted value containing "\," — which decoder.go's
//     parseQuoted (via strconv.UnquoteChar) does not recognize as a valid
//     escape, erroring and silently dropping the whole ADR line on re-decode.
//
// There is no input string that survives go-vcard's Encode+Decode round
// trip for a comma-bearing parameter value using its public Params API
// alone — confirmed by tracing decoder.go/encoder.go, not assumed.
//
// This is a confirmed limitation of the pinned go-vcard version, not
// something fixable from this package without forking it (disallowed by
// 30-adapters.md §30.B). The workaround below does not fork go-vcard: it
// substitutes a comma/colon-free sentinel token for GEO's value so the
// stock Encoder produces valid, parseable output, then splices the
// correctly quoted real value into the final byte stream afterward — a
// post-processing correction confined entirely to this adapter.
func exportAddresses(rec *contactmodel.Record, card vcard.Card, splices *[]geoSplice) {
	for i, addr := range rec.Card.Addresses {
		ac := kindValuesToAdrComponents(addr.Components)
		f := &vcard.Field{Value: assembleAdr(ac)}
		if addr.CountryCode != "" {
			setParam(f, ParamCC, addr.CountryCode)
		}
		if addr.TimeZone != "" {
			setParam(f, ParamTz, addr.TimeZone)
		}
		if addr.Full != "" {
			setParam(f, ParamLabel, addr.Full)
		}
		addTypeTokens(f, contextsToTypeTokens(addr.Contexts)...)
		setPref(f, addr.Pref)
		setPropID(f, idOrSynthetic(addr.ID, "addr", i))
		if addr.Coordinates != "" {
			sentinel := fmt.Sprintf("VCARD4GEOSENTINEL%d", i)
			setParam(f, ParamGeo, sentinel)
			*splices = append(*splices, geoSplice{
				find:    ParamGeo + "=" + sentinel,
				replace: ParamGeo + "=" + quotedParam(addr.Coordinates),
			})
		}
		card.Add(PropAdr, f)
	}
}

// ---------------------------------------------------------------------------
// Anniversaries (birth/wedding/death) + place (birthplace/deathplace)
// ---------------------------------------------------------------------------

// parsePartialDate parses a vCard DATE-AND-OR-TIME value into a
// contactmodel.PartialDate, accepting both the RFC 6350 basic form
// (YYYYMMDD / --MMDD / YYYY, as in the golden fixtures) and the extended
// dashed form (YYYY-MM-DD / --MM-DD) that 20-correspondence.md §20.4's
// date_partial transform documents as the canonical shape — both are valid
// ISO 8601 date forms and RFC 6350 §4.3 permits either.
func parsePartialDate(v string) *contactmodel.PartialDate {
	if v == "" {
		return nil
	}
	if strings.HasPrefix(v, "--") {
		core := strings.ReplaceAll(strings.TrimPrefix(v, "--"), "-", "")
		var pd contactmodel.PartialDate
		if len(core) >= 2 {
			if m, err := strconv.Atoi(core[0:2]); err == nil {
				pd.Month = &m
			}
		}
		if len(core) >= 4 {
			if d, err := strconv.Atoi(core[2:4]); err == nil {
				pd.Day = &d
			}
		}
		return &pd
	}
	digits := strings.ReplaceAll(v, "-", "")
	var pd contactmodel.PartialDate
	switch len(digits) {
	case 4:
		if y, err := strconv.Atoi(digits); err == nil {
			pd.Year = &y
		}
	case 6:
		if y, err := strconv.Atoi(digits[0:4]); err == nil {
			pd.Year = &y
		}
		if m, err := strconv.Atoi(digits[4:6]); err == nil {
			pd.Month = &m
		}
	case 8:
		if y, err := strconv.Atoi(digits[0:4]); err == nil {
			pd.Year = &y
		}
		if m, err := strconv.Atoi(digits[4:6]); err == nil {
			pd.Month = &m
		}
		if d, err := strconv.Atoi(digits[6:8]); err == nil {
			pd.Day = &d
		}
	default:
		return nil
	}
	return &pd
}

func formatPartialDate(p *contactmodel.PartialDate) string {
	if p == nil {
		return ""
	}
	switch {
	case p.Year != nil && p.Month != nil && p.Day != nil:
		return fmt.Sprintf("%04d-%02d-%02d", *p.Year, *p.Month, *p.Day)
	case p.Year != nil && p.Month != nil:
		return fmt.Sprintf("%04d-%02d", *p.Year, *p.Month)
	case p.Year != nil:
		return fmt.Sprintf("%04d", *p.Year)
	case p.Month != nil && p.Day != nil:
		return fmt.Sprintf("--%02d-%02d", *p.Month, *p.Day)
	case p.Month != nil:
		return fmt.Sprintf("--%02d", *p.Month)
	case p.Day != nil:
		return fmt.Sprintf("---%02d", *p.Day)
	}
	return ""
}

func parseVCardDateAndOrTime(v string) contactmodel.AnniversaryDate {
	if strings.Contains(v, "T") {
		return contactmodel.AnniversaryDate{Timestamp: &v}
	}
	return contactmodel.AnniversaryDate{Partial: parsePartialDate(v)}
}

func formatAnniversaryDate(d contactmodel.AnniversaryDate) string {
	if d.Partial != nil {
		return formatPartialDate(d.Partial)
	}
	if d.Timestamp != nil {
		return *d.Timestamp
	}
	return ""
}

func isZeroAnniversaryDate(d contactmodel.AnniversaryDate) bool {
	return d.Partial == nil && d.Timestamp == nil
}

// placeTextToAddress implements the place_text transform's import direction
// (anniversary.place.birth / anniversary.place.death): TEXT -> Address.Full;
// VALUE=uri geo: -> Address.Coordinates; other URI schemes have no neutral
// Address field to hold a bare URI, so (per the row's own note) they are
// preserved via passthrough instead of silently dropped.
func placeTextToAddress(propName string, f *vcard.Field, diags *[]contactmodel.Diagnostic, pt *[]contactmodel.JCardProp) *contactmodel.Address {
	if strings.EqualFold(paramValue(f, ParamValue), "uri") {
		if strings.HasPrefix(f.Value, "geo:") {
			return &contactmodel.Address{Coordinates: f.Value}
		}
		*pt = append(*pt, fieldToJCardProp(propName, f))
		*diags = append(*diags, contactmodel.Diagnostic{
			Severity: "info", Concept: "anniversary.place",
			Message: propName + " VALUE=uri with a non-geo: scheme has no neutral Address field; preserved via passthrough",
		})
		return nil
	}
	return &contactmodel.Address{Full: f.Value}
}

// addressToPlaceValue is place_text's export direction: Address.Full ->
// TEXT; else Address.Coordinates -> VALUE=uri; else join components (lossy
// — the caller emits a warn Diagnostic for this branch).
func addressToPlaceValue(a *contactmodel.Address) (value string, valueParam string) {
	if a == nil {
		return "", ""
	}
	if a.Full != "" {
		return a.Full, ""
	}
	if a.Coordinates != "" {
		return a.Coordinates, "uri"
	}
	var parts []string
	for _, c := range a.Components {
		if c.Value != "" {
			parts = append(parts, c.Value)
		}
	}
	return strings.Join(parts, ", "), ""
}

func importAnniversaries(card vcard.Card, rec *contactmodel.Record, diags *[]contactmodel.Diagnostic) {
	idx := 0
	addDate := func(kind, propName string, f *vcard.Field) {
		date := parseVCardDateAndOrTime(f.Value)
		if cs := paramValue(f, ParamCalscale); cs != "" && date.Partial != nil {
			date.Partial.CalendarScale = cs
		}
		rec.Card.Anniversaries = append(rec.Card.Anniversaries, contactmodel.Anniversary{
			ID: idOrSynthetic(propID(f), "anniv", idx), Kind: kind, Date: date,
		})
		idx++
	}
	for _, f := range card[PropBday] {
		addDate("birth", PropBday, f)
	}
	for _, f := range card[PropAnniversary] {
		addDate("wedding", PropAnniversary, f)
	}
	for _, f := range card[PropDeathdate] {
		if strings.EqualFold(paramValue(f, ParamValue), "text") {
			// RFC 6474 §2.3 free-text approximate-date variant ("circa
			// 1800"): neutral AnniversaryDate has no free-text case (per
			// this row's note), so it is preserved via passthrough rather
			// than forced into a PartialDate it cannot represent.
			rec.Passthrough.VCard = append(rec.Passthrough.VCard, fieldToJCardProp(PropDeathdate, f))
			*diags = append(*diags, contactmodel.Diagnostic{
				Severity: "info", Concept: "anniversary.death",
				Message: "DEATHDATE VALUE=text free-text form has no neutral field; preserved via passthrough",
			})
			continue
		}
		addDate("death", PropDeathdate, f)
	}

	attachPlace := func(kind, propName string) {
		for _, f := range card[propName] {
			addr := placeTextToAddress(propName, f, diags, &rec.Passthrough.VCard)
			if addr == nil {
				continue
			}
			for i := range rec.Card.Anniversaries {
				if rec.Card.Anniversaries[i].Kind == kind && rec.Card.Anniversaries[i].Place == nil {
					rec.Card.Anniversaries[i].Place = addr
					addr = nil
					break
				}
			}
			if addr != nil {
				rec.Card.Anniversaries = append(rec.Card.Anniversaries, contactmodel.Anniversary{
					ID: idOrSynthetic("", "anniv", idx), Kind: kind, Place: addr,
				})
				idx++
			}
		}
	}
	attachPlace("birth", PropBirthplace)
	attachPlace("death", PropDeathplace)
}

func exportAnniversaries(rec *contactmodel.Record, card vcard.Card, diags *[]contactmodel.Diagnostic) {
	for _, a := range rec.Card.Anniversaries {
		if !isZeroAnniversaryDate(a.Date) {
			var propName string
			switch a.Kind {
			case "birth":
				propName = PropBday
			case "wedding":
				propName = PropAnniversary
			case "death":
				propName = PropDeathdate
			}
			if propName != "" {
				val := formatAnniversaryDate(a.Date)
				f := &vcard.Field{Value: val}
				if a.Date.Partial != nil && a.Date.Partial.CalendarScale != "" && a.Date.Partial.CalendarScale != "gregorian" {
					setParam(f, ParamCalscale, a.Date.Partial.CalendarScale)
				}
				card.Add(propName, f)
			}
		}
		if a.Place != nil {
			var propName string
			switch a.Kind {
			case "birth":
				propName = PropBirthplace
			case "death":
				propName = PropDeathplace
			}
			if propName == "" {
				continue
			}
			val, valueParam := addressToPlaceValue(a.Place)
			if val == "" {
				continue
			}
			if a.Place.Full == "" && a.Place.Coordinates == "" && len(a.Place.Components) > 0 {
				*diags = append(*diags, contactmodel.Diagnostic{
					Severity: "warn", Concept: "anniversary.place." + a.Kind,
					Message: "Address has only structured components (no Full/Coordinates); joined lossily into " + propName,
				})
			}
			f := &vcard.Field{Value: val}
			if valueParam != "" {
				setParam(f, ParamValue, valueParam)
			}
			card.Add(propName, f)
		}
	}
}

// ---------------------------------------------------------------------------
// SpeakToAs (gramgender, pronouns)
// ---------------------------------------------------------------------------

func importSpeakToAs(card vcard.Card, rec *contactmodel.Record, diags *[]contactmodel.Diagnostic) {
	var s contactmodel.SpeakToAs
	var any bool
	// RFC 9554 §3.2 gives GRAMGENDER cardinality "*" (multiple occurrences
	// distinguished by LANGUAGE), and SpeakToAs.GrammaticalGenders is now a
	// slice matching that cardinality exactly — every occurrence is stored
	// losslessly, no diagnostic needed (see 10-neutral-model.md's SpeakToAs
	// doc comment and 20-correspondence.md's gramgender row).
	for i, f := range card[PropGramgender] {
		s.GrammaticalGenders = append(s.GrammaticalGenders, contactmodel.GrammaticalGender{
			ID:       idOrSynthetic(propID(f), "gg", i),
			Value:    strings.ToLower(f.Value),
			Language: paramValue(f, ParamLanguage),
		})
		any = true
	}
	for i, f := range card[PropPronouns] {
		ctx, _ := splitTypeTokens(f)
		s.Pronouns = append(s.Pronouns, contactmodel.Pronouns{
			ID: idOrSynthetic(propID(f), "pron", i), Pronouns: f.Value, Contexts: ctx, Pref: prefOf(f),
		})
		any = true
		// LANGUAGE param has no field on contactmodel.Pronouns (P0 keeps
		// localizations opaque per 20.7); note the drop rather than silently
		// discarding it.
		if paramValue(f, ParamLanguage) != "" {
			*diags = append(*diags, contactmodel.Diagnostic{
				Severity: "info", Concept: "pronouns",
				Message: "PRONOUNS LANGUAGE param has no neutral field (20.7 keeps localizations opaque in P0); not preserved",
			})
		}
	}
	if any {
		rec.Card.SpeakToAs = &s
	}
}

func exportSpeakToAs(rec *contactmodel.Record, card vcard.Card) {
	s := rec.Card.SpeakToAs
	if s == nil {
		return
	}
	for i, g := range s.GrammaticalGenders {
		f := &vcard.Field{Value: strings.ToLower(g.Value)}
		if g.Language != "" {
			setParam(f, ParamLanguage, g.Language)
		}
		setPropID(f, idOrSynthetic(g.ID, "gg", i))
		card.Add(PropGramgender, f)
	}
	for i, p := range s.Pronouns {
		f := &vcard.Field{Value: p.Pronouns}
		addTypeTokens(f, contextsToTypeTokens(p.Contexts)...)
		setPref(f, p.Pref)
		setPropID(f, idOrSynthetic(p.ID, "pron", i))
		card.Add(PropPronouns, f)
	}
}

// ---------------------------------------------------------------------------
// Personal info (expertise, hobby, interest)
// ---------------------------------------------------------------------------

func identityLevel(s string) string { return strings.ToLower(s) }

// levelToExpertise / expertiseToLevel (20.4 personalinfo transform, per the
// expertise row's note): neutral level high/medium/low <-> RFC 6715 EXPERTISE
// LEVEL vocabulary expert/average/beginner. HOBBY/INTEREST use the same
// vocabulary on both sides (identityLevel).
func levelToExpertise(level string) string {
	switch strings.ToLower(level) {
	case "high":
		return "expert"
	case "medium":
		return "average"
	case "low":
		return "beginner"
	}
	return strings.ToLower(level)
}

func expertiseToLevel(vcardLevel string) string {
	switch strings.ToLower(vcardLevel) {
	case "expert":
		return "high"
	case "average":
		return "medium"
	case "beginner":
		return "low"
	}
	return strings.ToLower(vcardLevel)
}

func importPersonalInfo(card vcard.Card, rec *contactmodel.Record) {
	idx := 0
	add := func(kind, propName string, levelIn func(string) string) {
		for _, f := range card[propName] {
			pi := contactmodel.PersonalInfo{ID: idOrSynthetic(propID(f), "pi", idx), Kind: kind, Value: f.Value}
			if lv := paramValue(f, ParamLevel); lv != "" {
				pi.Level = levelIn(lv)
			}
			if ix := paramValue(f, ParamIndex); ix != "" {
				if n, err := strconv.Atoi(ix); err == nil {
					pi.ListAs = &n
				}
			}
			rec.Card.PersonalInfo = append(rec.Card.PersonalInfo, pi)
			idx++
		}
	}
	add("expertise", PropExpertise, expertiseToLevel)
	add("hobby", PropHobby, identityLevel)
	add("interest", PropInterest, identityLevel)
}

func exportPersonalInfo(rec *contactmodel.Record, card vcard.Card) {
	for i, pi := range rec.Card.PersonalInfo {
		var propName string
		var levelOut func(string) string
		switch pi.Kind {
		case "expertise":
			propName, levelOut = PropExpertise, levelToExpertise
		case "hobby":
			propName, levelOut = PropHobby, identityLevel
		case "interest":
			propName, levelOut = PropInterest, identityLevel
		default:
			continue
		}
		f := &vcard.Field{Value: pi.Value}
		if pi.Level != "" {
			setParam(f, ParamLevel, levelOut(pi.Level))
		}
		if pi.ListAs != nil {
			setParam(f, ParamIndex, strconv.Itoa(*pi.ListAs))
		}
		setPropID(f, idOrSynthetic(pi.ID, "pi", i))
		card.Add(propName, f)
	}
}

// ---------------------------------------------------------------------------
// Notes / keywords
// ---------------------------------------------------------------------------

func importNotes(card vcard.Card, rec *contactmodel.Record) {
	for i, f := range card[PropNote] {
		n := contactmodel.Note{ID: idOrSynthetic(propID(f), "note", i), Note: f.Value}
		authorURI := paramValue(f, ParamAuthor)
		authorName := paramValue(f, ParamAuthorName)
		if authorURI != "" || authorName != "" {
			n.Author = &contactmodel.Author{Name: authorName, URI: authorURI}
		}
		if c := paramValue(f, ParamCreated); c != "" {
			n.Created = &contactmodel.Timestamp{UTC: vcardTimestampToRFC3339(c)}
		}
		rec.Card.Notes = append(rec.Card.Notes, n)
	}
}

func exportNotes(rec *contactmodel.Record, card vcard.Card) {
	for i, n := range rec.Card.Notes {
		f := &vcard.Field{Value: n.Note}
		if n.Author != nil {
			if n.Author.URI != "" {
				// AUTHOR is a quoted URI param (needs quoting due to ':';
				// no comma is expected in a mailto:/URI author value, so the
				// simple quotedParam helper — not the GEO splice workaround
				// — is sufficient here).
				setParam(f, ParamAuthor, quotedParam(n.Author.URI))
			}
			if n.Author.Name != "" {
				setParam(f, ParamAuthorName, n.Author.Name)
			}
		}
		if n.Created != nil {
			setParam(f, ParamCreated, rfc3339ToVCardTimestamp(n.Created.UTC))
		}
		setPropID(f, idOrSynthetic(n.ID, "note", i))
		card.Add(PropNote, f)
	}
}

func importKeywords(card vcard.Card, rec *contactmodel.Record) {
	for _, f := range card[PropCategories] {
		for _, k := range strings.Split(f.Value, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				rec.Card.Keywords = append(rec.Card.Keywords, k)
			}
		}
	}
}

func exportKeywords(rec *contactmodel.Record, card vcard.Card) {
	if len(rec.Card.Keywords) == 0 {
		return
	}
	card.SetValue(PropCategories, strings.Join(rec.Card.Keywords, ","))
}

// ---------------------------------------------------------------------------
// Resources: photo, logo, sound, calendar, freebusy, caladruri, key,
// directory, source, link, contacturi.
// ---------------------------------------------------------------------------

func importResourceInto(card vcard.Card, propName, kind string, dst *[]contactmodel.Resource, withMediaType, withIndex bool) {
	for i, f := range card[propName] {
		r := contactmodel.Resource{ID: idOrSynthetic(propID(f), "res", i), Kind: kind, URI: f.Value, Pref: prefOf(f)}
		if withMediaType {
			r.MediaType = paramValue(f, ParamMediatype)
		}
		if withIndex {
			if ix := paramValue(f, ParamIndex); ix != "" {
				if n, err := strconv.Atoi(ix); err == nil {
					r.ListAs = &n
				}
			}
		}
		*dst = append(*dst, r)
	}
}

func exportResourceFrom(card vcard.Card, propName string, src []contactmodel.Resource, kind string, withMediaType, withIndex bool) {
	idx := 0
	for _, r := range src {
		if r.Kind != kind {
			continue
		}
		f := &vcard.Field{Value: r.URI}
		setPref(f, r.Pref)
		if withMediaType && r.MediaType != "" {
			setParam(f, ParamMediatype, r.MediaType)
		}
		if withIndex && r.ListAs != nil {
			setParam(f, ParamIndex, strconv.Itoa(*r.ListAs))
		}
		setPropID(f, idOrSynthetic(r.ID, "res", idx))
		card.Add(propName, f)
		idx++
	}
}

func importResources(card vcard.Card, rec *contactmodel.Record) {
	importResourceInto(card, PropPhoto, "photo", &rec.Card.Media, true, false)
	importResourceInto(card, PropLogo, "logo", &rec.Card.Media, true, false)
	importResourceInto(card, PropSound, "sound", &rec.Card.Media, true, false)
	importResourceInto(card, PropCaluri, "", &rec.Card.Calendars, false, false)
	importResourceInto(card, PropFburl, "", &rec.Card.FreeBusyURLs, false, false)
	importResourceInto(card, PropCaladruri, "", &rec.Card.SchedulingAddresses, false, false)
	importResourceInto(card, PropKey, "", &rec.Card.CryptoKeys, true, false)
	importResourceInto(card, PropOrgDirectory, "directory", &rec.Card.Directories, false, true)
	importResourceInto(card, PropSource, "entry", &rec.Card.Directories, false, false)
	importResourceInto(card, PropURL, "", &rec.Card.Links, false, false)
	importResourceInto(card, PropContactURI, "", &rec.Card.ContactURIs, false, false)
}

func exportResources(rec *contactmodel.Record, card vcard.Card) {
	exportResourceFrom(card, PropPhoto, rec.Card.Media, "photo", true, false)
	exportResourceFrom(card, PropLogo, rec.Card.Media, "logo", true, false)
	exportResourceFrom(card, PropSound, rec.Card.Media, "sound", true, false)
	exportResourceFrom(card, PropCaluri, rec.Card.Calendars, "", false, false)
	exportResourceFrom(card, PropFburl, rec.Card.FreeBusyURLs, "", false, false)
	exportResourceFrom(card, PropCaladruri, rec.Card.SchedulingAddresses, "", false, false)
	exportResourceFrom(card, PropKey, rec.Card.CryptoKeys, "", true, false)
	exportResourceFrom(card, PropOrgDirectory, rec.Card.Directories, "directory", false, true)
	exportResourceFrom(card, PropSource, rec.Card.Directories, "entry", false, false)
	exportResourceFrom(card, PropURL, rec.Card.Links, "", false, false)
	exportResourceFrom(card, PropContactURI, rec.Card.ContactURIs, "", false, false)
}

// ---------------------------------------------------------------------------
// Langs / related / members
// ---------------------------------------------------------------------------

func importLangsRelatedMembers(card vcard.Card, rec *contactmodel.Record) {
	for i, f := range card[PropLang] {
		ctx, _ := splitTypeTokens(f)
		rec.Card.PreferredLanguages = append(rec.Card.PreferredLanguages, contactmodel.LanguagePref{
			ID: idOrSynthetic(propID(f), "lang", i), Language: f.Value, Contexts: ctx, Pref: prefOf(f),
		})
	}
	for _, f := range card[PropRelated] {
		rel := contactmodel.Relation{Target: f.Value}
		for _, tok := range paramAll(f, ParamType) {
			rel.Relations = append(rel.Relations, strings.ToLower(tok))
		}
		rec.Card.RelatedTo = append(rec.Card.RelatedTo, rel)
	}
	for _, f := range card[PropMember] {
		rec.Card.Members = append(rec.Card.Members, f.Value)
	}
}

func exportLangsRelatedMembers(rec *contactmodel.Record, card vcard.Card) {
	for i, l := range rec.Card.PreferredLanguages {
		f := &vcard.Field{Value: l.Language}
		addTypeTokens(f, contextsToTypeTokens(l.Contexts)...)
		setPref(f, l.Pref)
		setPropID(f, idOrSynthetic(l.ID, "lang", i))
		card.Add(PropLang, f)
	}
	for _, r := range rec.Card.RelatedTo {
		f := &vcard.Field{Value: r.Target}
		addTypeTokens(f, r.Relations...)
		card.Add(PropRelated, f)
	}
	for _, m := range rec.Card.Members {
		card.Add(PropMember, &vcard.Field{Value: m})
	}
}

// ---------------------------------------------------------------------------
// Passthrough (pt.vcard, pt.jscontact) — 20.5 escape-hatch precedence.
// ---------------------------------------------------------------------------

func importPassthrough(card vcard.Card, rec *contactmodel.Record) {
	mapped := mappedV4PropNames()

	var names []string
	for name := range card {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if strings.EqualFold(name, PropVersion) {
			continue
		}
		if strings.EqualFold(name, PropJSProp) {
			for _, f := range card[name] {
				ptr := paramValue(f, ParamJSPtr)
				if ptr == "" {
					continue
				}
				var raw json.RawMessage
				if err := json.Unmarshal([]byte(f.Value), &raw); err != nil {
					// Not valid JSON text (or the JSPROP producer emitted a
					// bare string) — fall back to treating the raw text as
					// a JSON string literal so nothing is lost.
					if b, merr := json.Marshal(f.Value); merr == nil {
						raw = json.RawMessage(b)
					} else {
						continue
					}
				}
				if rec.Passthrough.JSContact == nil {
					rec.Passthrough.JSContact = map[string]json.RawMessage{}
				}
				rec.Passthrough.JSContact[ptr] = raw
			}
			continue
		}
		if mapped[strings.ToUpper(name)] {
			continue
		}
		for _, f := range card[name] {
			rec.Passthrough.VCard = append(rec.Passthrough.VCard, fieldToJCardProp(name, f))
		}
	}
}

func exportPassthrough(rec *contactmodel.Record, card vcard.Card, mapped map[string]bool) {
	for _, p := range rec.Passthrough.VCard {
		if mapped[strings.ToUpper(p.Name)] {
			// de-dup guard (20.5): a mapped property name must not also be
			// re-emitted verbatim from passthrough.
			continue
		}
		name, f := jCardPropToField(p)
		card.Add(name, f)
	}

	var ptrs []string
	for ptr := range rec.Passthrough.JSContact {
		ptrs = append(ptrs, ptr)
	}
	sort.Strings(ptrs)
	for _, ptr := range ptrs {
		raw := rec.Passthrough.JSContact[ptr]
		f := &vcard.Field{Value: compactJSON(raw)}
		setParam(f, ParamJSPtr, ptr)
		card.Add(PropJSProp, f)
	}
}

func compactJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return string(raw)
	}
	return buf.String()
}
