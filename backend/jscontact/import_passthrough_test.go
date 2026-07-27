package jscontact

import (
	"encoding/json"
	"reflect"
	"testing"

	"mycorrhizal/contactmodel"
)

// Concepts: pt.vcard, pt.jscontact.
// Row pt.vcard:     Passthrough.VCard      /vCardProps    passthrough_vcard
// Row pt.jscontact: Passthrough.JSContact  (pointer keys) passthrough_js
func init() {
	registerImportCoverage("pt.vcard", "pt.jscontact")
}

func TestImport_PassthroughVCardProps(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "pt-vcard-example",
		"vCardProps": [
			{ "name": "x-custom-vcard-prop", "type": "text", "value": "hello" }
		]
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Passthrough.VCard) != 1 {
		t.Fatalf("len(Passthrough.VCard) = %d, want 1", len(rec.Passthrough.VCard))
	}
	p := rec.Passthrough.VCard[0]
	if p.Name != "x-custom-vcard-prop" || string(p.Value) != `"hello"` {
		t.Errorf("Passthrough.VCard[0] = %+v", p)
	}
}

// TestImport_PassthroughUnknownTopLevelProperty exercises 0.5's degradation
// policy for genuinely unmappable data: an unrecognized top-level JSContact
// property must not cause an error, and must be preserved (not silently
// dropped) via Record.Passthrough.JSContact.
func TestImport_PassthroughUnknownTopLevelProperty(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "pt-jscontact-example",
		"x-vendor-extension": { "foo": "bar", "n": 42 }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import should not error on an unknown top-level property: %v", err)
	}
	raw2, ok := rec.Passthrough.JSContact["/x-vendor-extension"]
	if !ok {
		t.Fatalf(`Passthrough.JSContact["/x-vendor-extension"] missing; keys: %v`, keysOfJSContactPassthrough(rec))
	}
	// Passthrough.JSContact stores the exact raw bytes (json.RawMessage), so
	// compare structurally rather than assuming canonical (whitespace-free)
	// formatting of the source literal.
	var got map[string]any
	if err := json.Unmarshal(raw2, &got); err != nil {
		t.Fatalf("Passthrough.JSContact[/x-vendor-extension] is not valid JSON: %v", err)
	}
	want := map[string]any{"foo": "bar", "n": float64(42)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Passthrough.JSContact[/x-vendor-extension] = %v, want %v", got, want)
	}
}

func keysOfJSContactPassthrough(r *contactmodel.Record) []string {
	out := make([]string, 0, len(r.Passthrough.JSContact))
	for k := range r.Passthrough.JSContact {
		out = append(out, k)
	}
	return out
}

// TestImport_PassthroughUnknownNestedProperty is the regression test for the
// defect this fix addresses: an unrecognized JSON key nested INSIDE a known
// object (here, one emails{} map entry) must not be silently discarded by
// codec.go's json.Unmarshal-based decoding -- it must be preserved via
// Record.Passthrough.JSContact, keyed by the JSON pointer to where it
// actually lives ("/emails/E1/x-custom"), not the top-level-only pointer an
// earlier (buggy) version of this adapter would have produced (or, since it
// couldn't see the value at all, simply dropped).
func TestImport_PassthroughUnknownNestedProperty(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "pt-nested-example",
		"emails": {
			"E1": { "@type": "EmailAddress", "address": "alice@example.com", "x-custom": "keep-me" }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import should not error on an unknown nested property: %v", err)
	}
	// The mapped sibling field must still have landed normally.
	if len(rec.Card.Emails) != 1 || rec.Card.Emails[0].Address != "alice@example.com" {
		t.Fatalf("Card.Emails = %+v, want one mapped alice@example.com entry", rec.Card.Emails)
	}
	raw2, ok := rec.Passthrough.JSContact["/emails/E1/x-custom"]
	if !ok {
		t.Fatalf(`Passthrough.JSContact["/emails/E1/x-custom"] missing; keys: %v`, keysOfJSContactPassthrough(rec))
	}
	var got string
	if err := json.Unmarshal(raw2, &got); err != nil {
		t.Fatalf("Passthrough.JSContact[/emails/E1/x-custom] is not valid JSON: %v", err)
	}
	if got != "keep-me" {
		t.Errorf("Passthrough.JSContact[/emails/E1/x-custom] = %q, want %q", got, "keep-me")
	}
}

// TestImport_PassthroughUnknownDeeplyNestedProperty exercises a deeper
// nesting: an unrecognized key inside one array element of a nested
// (non-Id-map) component list -- addresses{}.components[] -- to confirm the
// pointer-building walk handles a mixed map/array path
// ("/addresses/{id}/components/{index}/{key}"), not just a single
// id-map-element level.
func TestImport_PassthroughUnknownDeeplyNestedProperty(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "pt-deep-nested-example",
		"addresses": {
			"A1": { "@type": "Address", "components": [
				{ "@type": "AddressComponent", "kind": "locality", "value": "Springfield", "x-phonetic-hint": "SPRING-field" }
			]}
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import should not error on an unknown deeply nested property: %v", err)
	}
	raw2, ok := rec.Passthrough.JSContact["/addresses/A1/components/0/x-phonetic-hint"]
	if !ok {
		t.Fatalf(`Passthrough.JSContact["/addresses/A1/components/0/x-phonetic-hint"] missing; keys: %v`, keysOfJSContactPassthrough(rec))
	}
	var got string
	if err := json.Unmarshal(raw2, &got); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if got != "SPRING-field" {
		t.Errorf("value = %q, want %q", got, "SPRING-field")
	}
}
