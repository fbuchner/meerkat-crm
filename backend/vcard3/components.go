package vcard3

import "strings"

// This file (components.go, WP-50) is a low-level, pure structured-value
// codec: escaping/splitting/joining of vCard TEXT structured values, and the
// N and ADR component assemblers, trimmed to vCard 3.0's legacy-only field
// counts (5-field N, 7-field ADR — no 9554 expansion, no PROP-ID/component
// params). It has no dependency on contactmodel or correspondence.
//
// Per docs/fork-plan/30-adapters.md §30.C's correction: this is a **fresh,
// duplicated** copy of the same kind of logic backend/vcard4/components.go
// has (itself salvaged from backend/carddav/vcard_mapper.go), not an import
// of vcard4 — vcard4's functions are unexported and no adapter package may
// import another adapter package (docs/fork-plan/00-overview.md §0.6). The
// escaping rules are confirmed against docs/specs/rfc2426-v3-baseline.md
// (same backslash-escape family as 4.0: backslash, semicolon as the
// structured-value separator, plus comma/newline handled by go-vcard's own
// value-level encoder — see the same division of labor documented in
// vcard4/components.go).

// escapeComponent escapes a single value before it is joined into a
// structured vCard 3.0 value (N or ADR) with ";" separators. go-vcard's own
// line encoder escapes "\", "\n" and "," when serializing a Field.Value to
// wire text, but has no notion of structured values, so it does not escape
// the structured ";" separator. This codec therefore only escapes "\" and
// ";" itself ("\" first, so the backslash added for ";" is not re-escaped);
// comma/newline escaping is left to go-vcard's encoder further down the
// pipeline.
func escapeComponent(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	return s
}

// splitComponents splits a semicolon-delimited structured vCard 3.0 value
// (N or ADR) into its components, honoring backslash-escaped semicolons (an
// escaped "\;" is not treated as a separator) and unescaping each resulting
// component. It always returns at least one element (an empty value yields
// [""]).
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

// joinComponents is the inverse of splitComponents: it escapes each part (per
// escapeComponent) and joins them with ";" into a single structured vCard 3.0
// value.
func joinComponents(parts []string) string {
	escaped := make([]string, len(parts))
	for i, p := range parts {
		escaped[i] = escapeComponent(p)
	}
	return strings.Join(escaped, ";")
}

// component returns the i-th element of a split structured value, or "" if
// the value has fewer than i+1 components (e.g. a short-form N or ADR value
// missing trailing components).
func component(comps []string, i int) string {
	if i < len(comps) {
		return comps[i]
	}
	return ""
}

// --- N: structured name (RFC 2426 §3.1.2 — exactly 5 components) ---

// NComponents holds the five structured components of a vCard 3.0 N value,
// in wire order (docs/specs/rfc2426-v3-baseline.md §2):
// Family;Given;Additional;Prefix;Suffix. Unlike 4.0/RFC 9554, there is no
// SecondarySurname or Generation component — those concepts have no 3.0 home
// (correspondence rows name.surname2/name.generation warn-drop on export).
type NComponents struct {
	Family     string
	Given      string
	Additional string
	Prefix     string
	Suffix     string
}

// assembleN renders n as a 5-component vCard 3.0 N value string.
func assembleN(n NComponents) string {
	return joinComponents([]string{n.Family, n.Given, n.Additional, n.Prefix, n.Suffix})
}

// disassembleN parses a vCard 3.0 N value string into its 5 components.
// Shorter (or empty) N values are accepted: missing trailing components are
// treated as empty. Any components beyond the 5th (e.g. a 4.0-authored file
// with SecondarySurname/Generation) are ignored here; the adapter is
// responsible for warning about that loss, not this pure codec.
func disassembleN(value string) NComponents {
	c := splitComponents(value)
	return NComponents{
		Family:     component(c, 0),
		Given:      component(c, 1),
		Additional: component(c, 2),
		Prefix:     component(c, 3),
		Suffix:     component(c, 4),
	}
}

// --- ADR: structured address (RFC 2426 §3.2.1 — exactly 7 components) ---

// AdrComponents holds the seven structured components of a vCard 3.0 ADR
// value, in wire order (docs/specs/rfc2426-v3-baseline.md §2):
// POBox;Ext;Street;Locality;Region;PostalCode;Country. No room/floor/
// building/etc — those are RFC 9554/4.0-only extensions (extra 9553
// AddressComponent kinds warn-drop on v3 export).
type AdrComponents struct {
	POBox    string
	Ext      string
	Street   string
	Locality string
	Region   string
	Postal   string
	Country  string
}

// assembleAdr renders a as a 7-component vCard 3.0 ADR value string.
func assembleAdr(a AdrComponents) string {
	return joinComponents([]string{a.POBox, a.Ext, a.Street, a.Locality, a.Region, a.Postal, a.Country})
}

// disassembleAdr parses a vCard 3.0 ADR value string into its 7 components.
// Shorter (or empty) ADR values are accepted: missing trailing components are
// treated as empty. Any components beyond the 7th (e.g. a 4.0-authored file
// with the 9554 18-component expansion) are ignored here; the adapter is
// responsible for warning about that loss, not this pure codec.
func disassembleAdr(value string) AdrComponents {
	c := splitComponents(value)
	return AdrComponents{
		POBox:    component(c, 0),
		Ext:      component(c, 1),
		Street:   component(c, 2),
		Locality: component(c, 3),
		Region:   component(c, 4),
		Postal:   component(c, 5),
		Country:  component(c, 6),
	}
}
