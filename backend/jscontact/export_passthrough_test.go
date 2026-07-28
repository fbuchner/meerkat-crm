package jscontact

import (
	"encoding/json"
	"fmt"
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("pt.vcard", "pt.jscontact")
}

func TestExport_PassthroughVCardProps(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "pt-vcard-example",
	}}
	rec.Passthrough.VCard = []contactmodel.JCardProp{
		{Name: "x-custom-vcard-prop", Type: "text", Value: json.RawMessage(`"hello"`)},
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/vCardProps/0/name", "x-custom-vcard-prop")
	rfctest.AssertJSONPointer(t, out, "/vCardProps/0/value", "hello")
}

// TestExport_PassthroughUnknownTopLevelPropertyIsTrueInverse exercises the
// review-gate requirement that passthrough be a true inverse: importing a
// fixture with an unknown top-level property and re-exporting it must
// re-emit that property unchanged (60-review-gates.md §60.3).
func TestExport_PassthroughUnknownTopLevelPropertyIsTrueInverse(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "pt-jscontact-roundtrip-example",
		"x-vendor-extension": { "foo": "bar", "n": 42 }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/x-vendor-extension/foo", "bar")
	rfctest.AssertJSONPointer(t, out, "/x-vendor-extension/n", 42)
	// The mapped uid property must still be present and correct alongside it.
	rfctest.AssertJSONPointer(t, out, "/uid", "pt-jscontact-roundtrip-example")
}

// TestExport_PassthroughDeDupGuard exercises 20.5's de-dup guard: a
// passthrough entry recorded under a pointer that collides with a property
// this adapter actually maps (e.g. "/uid") must never shadow or duplicate
// the mapped value.
func TestExport_PassthroughDeDupGuard(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{UID: "mapped-uid-wins"}}
	rec.Passthrough.JSContact = map[string]json.RawMessage{
		"/uid": json.RawMessage(`"should-never-appear"`),
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/uid", "mapped-uid-wins")
}

// TestExport_PassthroughUnknownNestedPropertyIsTrueInverse is the regression
// test for the defect this fix addresses, on the export side: importing a
// fixture with an unknown property nested INSIDE one emails{} entry and
// re-exporting it must re-emit that property at the same nested location
// ("/emails/E1/x-custom"), alongside the mapped sibling field, not drop it
// or (as an earlier, buggy version of this adapter would have done, had it
// even recovered the value) mis-splice it at the Card's top level.
func TestExport_PassthroughUnknownNestedPropertyIsTrueInverse(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "pt-nested-roundtrip-example",
		"emails": {
			"E1": { "@type": "EmailAddress", "address": "alice@example.com", "x-custom": "keep-me" }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/emails/E1/address", "alice@example.com")
	rfctest.AssertJSONPointer(t, out, "/emails/E1/x-custom", "keep-me")
	// It must NOT have been mis-spliced at the top level.
	if _, err := jsonPointerLookup(out, "/x-custom"); err == nil {
		t.Errorf("nested passthrough entry was spliced at the Card top level instead of /emails/E1/x-custom")
	}
}

// TestExport_PassthroughUnknownDeeplyNestedPropertyIsTrueInverse mirrors
// TestImport_PassthroughUnknownDeeplyNestedProperty on the export side: a
// property nested inside an array element of a non-Id-map component list
// (addresses{}.components[]) must round-trip at its full mixed map/array
// pointer.
func TestExport_PassthroughUnknownDeeplyNestedPropertyIsTrueInverse(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "pt-deep-nested-roundtrip-example",
		"addresses": {
			"A1": { "@type": "Address", "components": [
				{ "@type": "AddressComponent", "kind": "locality", "value": "Springfield", "x-phonetic-hint": "SPRING-field" }
			]}
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/addresses/A1/components/0/value", "Springfield")
	rfctest.AssertJSONPointer(t, out, "/addresses/A1/components/0/x-phonetic-hint", "SPRING-field")
}

// jsonPointerLookup is a minimal single-segment top-level lookup helper used
// only to assert a key's ABSENCE at the Card's top level (rfctest.AssertJSONPointer
// is built to assert presence/equality, not absence, so a small local helper
// is used here instead of stretching that one to a different job).
func jsonPointerLookup(raw []byte, pointer string) (json.RawMessage, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	key := pointer[1:]
	v, ok := doc[key]
	if !ok {
		return nil, fmt.Errorf("no key %q", key)
	}
	return v, nil
}
