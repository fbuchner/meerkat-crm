package correspondence

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"meerkat/contactmodel"
)

// --- (a) no duplicate concept_id -------------------------------------------------

func TestNoDuplicateConceptID(t *testing.T) {
	seen := make(map[string]bool)
	for _, r := range Load() {
		if seen[r.ConceptID] {
			t.Errorf("duplicate concept_id %q", r.ConceptID)
		}
		seen[r.ConceptID] = true
	}

	// ByConcept must also succeed (and implicitly re-checks this invariant by
	// panicking on a duplicate, per its documented contract).
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ByConcept() panicked unexpectedly: %v", r)
			}
		}()
		ByConcept()
	}()
}

// --- (b) every neutral_path resolves against contactmodel.Record ---------------
//
// This implements exactly the resolution algorithm given in
// docs/fork-plan/20-correspondence.md §20.2 — pure reflect, no custom path DSL
// beyond the grammar, no enum-value checking.

// parseSegment splits one path segment into its field name and bracket kind.
// bracket is one of "" (none), "[]", or "[kind=X]".
func parseSegment(segment string) (name string, bracket string) {
	if !strings.HasSuffix(segment, "]") {
		return segment, ""
	}
	idx := strings.Index(segment, "[")
	if idx < 0 {
		// Not actually bracketed (shouldn't happen given the HasSuffix check
		// above unless the segment is malformed); treat as a bare name.
		return segment, ""
	}
	name = segment[:idx]
	inner := segment[idx+1 : len(segment)-1]
	if inner == "" {
		return name, "[]"
	}
	return name, "[kind=X]"
}

// resolveNeutralPath walks path per the §20.2 algorithm, starting from
// contactmodel.Record. It returns an error describing the first failure, or
// nil if the path resolves.
func resolveNeutralPath(path string) error {
	cur := reflect.TypeOf(contactmodel.Record{})
	for _, segment := range strings.Split(path, ".") {
		name, bracket := parseSegment(segment)

		field, ok := cur.FieldByName(name)
		if !ok {
			return fmt.Errorf("unknown field %q on %s", name, cur.String())
		}
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		if bracket != "" {
			if ft.Kind() != reflect.Slice {
				return fmt.Errorf("field %q is not a slice, cannot use [] / [kind=X]", name)
			}
			elem := ft.Elem()
			if bracket == "[kind=X]" {
				if _, ok := elem.FieldByName("Kind"); !ok {
					return fmt.Errorf("predicate key 'kind' has no corresponding field on %s", elem.String())
				}
			}
			cur = elem
		} else {
			cur = ft
		}
	}
	return nil
}

func TestNeutralPathResolves(t *testing.T) {
	for _, r := range Load() {
		r := r
		t.Run(r.ConceptID, func(t *testing.T) {
			if err := resolveNeutralPath(r.NeutralPath); err != nil {
				t.Errorf("neutral_path %q for concept_id %q does not resolve: %v", r.NeutralPath, r.ConceptID, err)
			}
		})
	}
}

// --- (c) every transform name exists in the 20.4 transforms registry -----------

// validTransforms is the hardcoded name list from docs/fork-plan/20-correspondence.md
// §20.4. These are names only, for this mechanical check — the transform
// implementations themselves belong to the adapter WPs (30/40/50).
var validTransforms = map[string]bool{
	"identity":          true,
	"ts_rfc3339":        true,
	"date_partial":      true,
	"n_component":       true,
	"adr_components":    true,
	"org_units":         true,
	"ctx2type":          true,
	"feat2type":         true,
	"pref":              true,
	"enum_lower":        true,
	"csv_join":          true,
	"geo_uri":           true,
	"media_uri":         true,
	"onlineservice":     true,
	"personalinfo":      true,
	"related":           true,
	"place_text":        true,
	"passthrough_vcard": true,
	"passthrough_js":    true,
}

func TestTransformNamesAreRegistered(t *testing.T) {
	for _, r := range Load() {
		r := r
		t.Run(r.ConceptID, func(t *testing.T) {
			if !validTransforms[r.Transform] {
				t.Errorf("concept_id %q: transform %q is not in the 20.4 registry", r.ConceptID, r.Transform)
			}
		})
	}
}

// --- (d) every v4_prop/v3_prop (when not "-") is a real IANA property name -----
//
// Hardcoded for this check only, from docs/fork-plan/30-adapters.md.

// vcard4Properties is the exact property-name list from 30-adapters.md §30.B's
// consts.go section ("from IANA vcard-elements; all must appear").
var vcard4Properties = map[string]bool{
	"SOURCE": true, "KIND": true, "XML": true, "FN": true, "N": true,
	"NICKNAME": true, "PHOTO": true, "BDAY": true, "ANNIVERSARY": true,
	"GENDER": true, "ADR": true, "TEL": true, "EMAIL": true, "IMPP": true,
	"LANG": true, "TZ": true, "GEO": true, "TITLE": true, "ROLE": true,
	"LOGO": true, "ORG": true, "MEMBER": true, "RELATED": true,
	"CATEGORIES": true, "NOTE": true, "PRODID": true, "REV": true,
	"SOUND": true, "UID": true, "CLIENTPIDMAP": true, "URL": true,
	"VERSION": true, "KEY": true, "FBURL": true, "CALADRURI": true,
	"CALURI": true, "BIRTHPLACE": true, "DEATHPLACE": true,
	"DEATHDATE": true, "EXPERTISE": true, "HOBBY": true, "INTEREST": true,
	"ORG-DIRECTORY": true, "CONTACT-URI": true, "CREATED": true,
	"GRAMGENDER": true, "LANGUAGE": true, "PRONOUNS": true,
	"SOCIALPROFILE": true, "JSPROP": true,
}

// vcard3Properties is the exact property-name list from 30-adapters.md
// §30.C's consts.go section: "Per RFC 2426 §5 ..., plus the X- extension
// properties this table actually maps to in 20-correspondence.md".
var vcard3Properties = map[string]bool{
	"BEGIN": true, "END": true, "SOURCE": true, "NAME": true, "FN": true,
	"N": true, "NICKNAME": true, "PHOTO": true, "BDAY": true, "ADR": true,
	"LABEL": true, "TEL": true, "EMAIL": true, "MAILER": true, "TZ": true,
	"GEO": true, "TITLE": true, "ROLE": true, "LOGO": true, "AGENT": true,
	"ORG": true, "CATEGORIES": true, "NOTE": true, "PRODID": true,
	"REV": true, "SORT-STRING": true, "SOUND": true, "UID": true,
	"URL": true, "VERSION": true, "CLASS": true, "KEY": true, "IMPP": true,
	"CALURI": true, "FBURL": true, "CALADRURI": true,
	"X-SOCIALPROFILE": true, "X-SERVICE-TYPE": true, "X-ANNIVERSARY": true,
}

// baseProp strips a structured-value index suffix like "N[0]" -> "N" and
// recognizes the "-" (none) and "*verbatim*" (escape-hatch passthrough,
// 20.3 pt.vcard row) sentinel values, which are not checked against the IANA
// list.
func baseProp(prop string) (name string, skip bool) {
	if prop == "-" || prop == "*verbatim*" {
		return "", true
	}
	if idx := strings.Index(prop, "["); idx >= 0 {
		return prop[:idx], false
	}
	return prop, false
}

func TestPropertyNamesAreIANA(t *testing.T) {
	for _, r := range Load() {
		r := r
		t.Run(r.ConceptID+"/v4", func(t *testing.T) {
			name, skip := baseProp(r.V4Prop)
			if skip {
				return
			}
			if !vcard4Properties[name] {
				t.Errorf("concept_id %q: v4_prop %q (base %q) not in vCard4 IANA property list", r.ConceptID, r.V4Prop, name)
			}
		})
		t.Run(r.ConceptID+"/v3", func(t *testing.T) {
			name, skip := baseProp(r.V3Prop)
			if skip {
				return
			}
			if !vcard3Properties[name] {
				t.Errorf("concept_id %q: v3_prop %q (base %q) not in vCard3 property list", r.ConceptID, r.V3Prop, name)
			}
		})
	}
}
