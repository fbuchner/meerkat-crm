package vcard4

import (
	"fmt"
	"strings"

	"github.com/emersion/go-vcard"
)

// This file (components.go, WP-40a) is a low-level, pure structured-value
// codec: escaping/splitting/joining of vCard TEXT structured values, the N
// and ADR component assemblers, and the group-based custom-label helpers. It
// has no dependency on contactmodel or correspondence — the neutral<->vcard4
// field mapping is WP-40b's adapter.go, which does not exist yet.
//
// The escape/split logic is salvaged and generalized from
// backend/carddav/vcard_mapper.go's escapeComponent/splitComponents; the
// label-handling helpers generalize that file's group + X-ABLabel idiom
// (see addTypedField/labelFromGroup/normalizeLabel there). See
// docs/specs/rfc6350-baseline.md §2 for the TEXT escaping rules this
// implements.

// escapeComponent escapes a single value before it is joined into a
// structured vCard value (e.g. N or ADR) with ";" separators.
//
// go-vcard's own line encoder (formatValue in encoder.go) already escapes
// "\", "\n" and "," when it serializes a Field.Value to wire text — but it
// does NOT escape the structured ";" separator, since it has no notion of
// structured values. So this codec only escapes "\" and ";" itself; the
// comma/newline escaping is left to go-vcard's encoder further down the
// pipeline. unescapeComponent/splitComponents reverse only this "\"/";"
// escaping. "\" must be escaped first so the backslash added for ";" is not
// itself re-escaped.
func escapeComponent(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	return s
}

// unescapeComponent reverses escapeComponent for a single already-split
// component: "\;" -> ";", "\\" -> "\", and (defensively) any other
// backslash-escaped byte is unescaped to the byte itself.
func unescapeComponent(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			b.WriteByte(s[i+1])
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// splitComponents splits a semicolon-delimited structured vCard value (e.g.
// an N or ADR value) into its components, honoring backslash-escaped
// semicolons (an escaped "\;" is not treated as a separator) and unescaping
// each resulting component per unescapeComponent. It always returns at least
// one element (an empty value yields [""]).
func splitComponents(value string) []string {
	var parts []string
	var cur strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			cur.WriteByte(value[i+1])
			i++
			continue
		}
		if value[i] == ';' {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(value[i])
	}
	return append(parts, cur.String())
}

// joinComponents is the inverse of splitComponents: it escapes each part
// (per escapeComponent) and joins them with ";" into a single structured
// vCard value.
func joinComponents(parts []string) string {
	escaped := make([]string, len(parts))
	for i, p := range parts {
		escaped[i] = escapeComponent(p)
	}
	return strings.Join(escaped, ";")
}

// component returns the i-th element of a split structured value, or "" if
// the value has fewer than i+1 components (e.g. a legacy/short-form N or ADR
// value missing trailing components).
func component(comps []string, i int) string {
	if i < len(comps) {
		return comps[i]
	}
	return ""
}

// --- N: structured name (RFC 9554 §2.2 expands RFC 6350's 5 components to 7) ---

// NComponents holds the seven structured components of a vCard N value, in
// wire order (docs/specs/rfc9554-vcard-extensions.md §3, "N (§2.2) — 7
// components in order: Family; Given; Additional; Prefix; Suffix;
// SecondarySurname; Generation"):
//
//  1. Family      (family name / surname)
//  2. Given       (given name)
//  3. Additional  (additional/middle names)
//  4. Prefix      (honorific prefix)
//  5. Suffix      (honorific suffix)
//  6. Surname2    (secondary surname, RFC 9554 new component)
//  7. Generation  (generation marker, e.g. "Jr.", RFC 9554 new component)
type NComponents struct {
	Family     string
	Given      string
	Additional string
	Prefix     string
	Suffix     string
	Surname2   string
	Generation string
}

// assembleN renders n as a 7-component vCard N value string.
func assembleN(n NComponents) string {
	return joinComponents([]string{
		n.Family, n.Given, n.Additional, n.Prefix, n.Suffix, n.Surname2, n.Generation,
	})
}

// disassembleN parses a vCard N value string into its 7 components. Legacy
// 5-component (or shorter) N values are accepted: missing trailing
// components are treated as empty.
func disassembleN(value string) NComponents {
	c := splitComponents(value)
	return NComponents{
		Family:     component(c, 0),
		Given:      component(c, 1),
		Additional: component(c, 2),
		Prefix:     component(c, 3),
		Suffix:     component(c, 4),
		Surname2:   component(c, 5),
		Generation: component(c, 6),
	}
}

// --- ADR: structured address (RFC 9554 §2.1 expands RFC 6350's 7 components to 18) ---

// AdrComponents holds the eighteen structured components of a vCard ADR
// value, in wire order (docs/specs/rfc9554-vcard-extensions.md §3, "ADR
// (§2.1) — 18 components in order: POBox; Ext; Street; Locality; Region;
// Code; Country; Room; Apartment; Floor; StreetNumber; StreetName; Building;
// Block; Subdistrict; District; Landmark; Direction"):
//
//  1. POBox        7.  Country      13. Building
//  2. Ext           8.  Room         14. Block
//  3. Street        9.  Apartment    15. Subdistrict
//  4. Locality     10.  Floor        16. District
//  5. Region       11.  Number       17. Landmark
//  6. Code         12.  StreetName   18. Direction
type AdrComponents struct {
	POBox       string
	Ext         string
	Street      string
	Locality    string
	Region      string
	Code        string
	Country     string
	Room        string
	Apartment   string
	Floor       string
	Number      string // RFC 9554 "StreetNumber"
	StreetName  string // RFC 9554 "StreetName"
	Building    string
	Block       string
	Subdistrict string
	District    string
	Landmark    string
	Direction   string
}

// assembleAdr renders a as an 18-component vCard ADR value string.
func assembleAdr(a AdrComponents) string {
	return joinComponents([]string{
		a.POBox, a.Ext, a.Street, a.Locality, a.Region, a.Code, a.Country,
		a.Room, a.Apartment, a.Floor, a.Number, a.StreetName, a.Building,
		a.Block, a.Subdistrict, a.District, a.Landmark, a.Direction,
	})
}

// disassembleAdr parses a vCard ADR value string into its 18 components.
// Legacy 7-component (or shorter) ADR values are accepted: missing trailing
// components are treated as empty.
func disassembleAdr(value string) AdrComponents {
	c := splitComponents(value)
	return AdrComponents{
		POBox:       component(c, 0),
		Ext:         component(c, 1),
		Street:      component(c, 2),
		Locality:    component(c, 3),
		Region:      component(c, 4),
		Code:        component(c, 5),
		Country:     component(c, 6),
		Room:        component(c, 7),
		Apartment:   component(c, 8),
		Floor:       component(c, 9),
		Number:      component(c, 10),
		StreetName:  component(c, 11),
		Building:    component(c, 12),
		Block:       component(c, 13),
		Subdistrict: component(c, 14),
		District:    component(c, 15),
		Landmark:    component(c, 16),
		Direction:   component(c, 17),
	}
}

// --- PROP-ID / group / X-ABLabel label handling ---
//
// Generalizes backend/carddav/vcard_mapper.go's grouped custom-label idiom:
// a value field that carries a user-defined (non-standard) label is given a
// property group (e.g. "item1"), and a companion X-ABLabel field in the same
// group carries the human-readable label text. This is the de facto
// (Apple-originated, non-IANA) convention most CardDAV clients use; RFC 9554
// separately adds the standardized PROP-ID parameter for stable per-value
// identifiers, handled by the small propID/setPropID helpers below.

// xabLabelProp is the (non-standard but ubiquitous) property Apple Contacts
// and most CardDAV clients use to carry a human-readable custom label for a
// value, linked to that value via a shared property group.
const xabLabelProp = "X-ABLabel"

// groupForIndex returns the property-group name used to link a value field
// to its companion X-ABLabel field, e.g. groupForIndex(1) == "item1".
func groupForIndex(n int) string {
	return fmt.Sprintf("item%d", n)
}

// isXABLabelField reports whether propName is the non-standard X-ABLabel
// property used to carry a custom label for a grouped field.
func isXABLabelField(propName string) bool {
	return strings.EqualFold(propName, xabLabelProp)
}

// labelFromXABLabel returns the trimmed value of whichever field in
// labelFields (typically card[xabLabelProp]) has a Group matching group
// (case-insensitively). It returns "" if group is empty or no field matches.
func labelFromXABLabel(group string, labelFields []*vcard.Field) string {
	if group == "" {
		return ""
	}
	for _, f := range labelFields {
		if f != nil && strings.EqualFold(f.Group, group) {
			return strings.TrimSpace(f.Value)
		}
	}
	return ""
}

// normalizeAppleLabel maps Apple's pseudo-label tokens (e.g. "_$!<Home>!$_",
// as emitted by Apple Contacts for its own built-in labels even when using
// the X-ABLabel mechanism) to a plain lowercase label; any other label is
// returned unchanged.
func normalizeAppleLabel(label string) string {
	const prefix, suffix = "_$!<", ">!$_"
	if strings.HasPrefix(label, prefix) && strings.HasSuffix(label, suffix) {
		inner := strings.ToLower(label[len(prefix) : len(label)-len(suffix)])
		if inner == "mobile" {
			return "cell"
		}
		return inner
	}
	return label
}

// propID returns the RFC 9554 PROP-ID parameter value set on f, or "" if f
// (or the parameter) is unset.
func propID(f *vcard.Field) string {
	if f == nil || f.Params == nil {
		return ""
	}
	return f.Params.Get(ParamPropID)
}

// setPropID sets the RFC 9554 PROP-ID parameter on f to id.
func setPropID(f *vcard.Field, id string) {
	if f.Params == nil {
		f.Params = vcard.Params{}
	}
	f.Params.Set(ParamPropID, id)
}
