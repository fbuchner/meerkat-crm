package jscontact

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// canonicalize runs raw bytes through one Unmarshal/Marshal pass, returning
// the re-marshaled bytes. Used to compute the "fixed point" byte form that a
// second pass must reproduce exactly.
func canonicalize(t *testing.T, data []byte) []byte {
	t.Helper()
	c, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v\ninput: %s", err, data)
	}
	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	return out
}

// TestFixtureRoundTripFixedPoint asserts, for every *.json fixture under
// testdata/, that Marshal(Unmarshal(x)) is a fixed point: re-running
// Unmarshal/Marshal on that output reproduces the identical bytes. This is
// the WP-30a acceptance criterion — raw byte equality against the original
// input isn't required (whitespace/key-order in the source fixture need not
// match our canonical form), but the codec's own canonical form must be
// stable under repeated round-tripping.
func TestFixtureRoundTripFixedPoint(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no fixtures found under testdata/")
	}
	for _, path := range matches {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			firstPass := canonicalize(t, raw)
			secondPass := canonicalize(t, firstPass)

			if !bytes.Equal(firstPass, secondPass) {
				t.Fatalf("Marshal(Unmarshal(x)) is not a fixed point for %s\nfirst:  %s\nsecond: %s",
					path, firstPass, secondPass)
			}
		})
	}
}

// TestJohnDoeFixtureFields spot-checks the RFC 9553 Fig. 6 golden fixture
// (docs/fork-plan/golden-fixtures/johndoe.jscontact.json) imports the
// expected scalar/structured fields, beyond the generic fixed-point check.
func TestJohnDoeFixtureFields(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "johndoe.jscontact.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	c, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if c.Type != "Card" {
		t.Errorf("Type = %q, want Card", c.Type)
	}
	if c.UID != "22B2C7DF-9120-4969-8460-05956FE6B065" {
		t.Errorf("UID = %q", c.UID)
	}
	if c.Kind != "individual" {
		t.Errorf("Kind = %q, want individual", c.Kind)
	}
	if c.Name == nil {
		t.Fatal("Name is nil")
	}
	if c.Name.Type != "Name" {
		t.Errorf("Name.Type = %q, want Name (defaulted)", c.Name.Type)
	}
	if c.Name.IsOrdered == nil || !*c.Name.IsOrdered {
		t.Errorf("Name.IsOrdered = %v, want true", c.Name.IsOrdered)
	}
	if len(c.Name.Components) != 2 {
		t.Fatalf("len(Name.Components) = %d, want 2", len(c.Name.Components))
	}
	if c.Name.Components[0].Kind != "given" || c.Name.Components[0].Value != "John" {
		t.Errorf("Components[0] = %+v", c.Name.Components[0])
	}
	if c.Name.Components[1].Kind != "surname" || c.Name.Components[1].Value != "Doe" {
		t.Errorf("Components[1] = %+v", c.Name.Components[1])
	}
	if c.Name.Components[0].Type != "NameComponent" {
		t.Errorf("Components[0].Type = %q, want NameComponent (defaulted)", c.Name.Components[0].Type)
	}
}

// TestTitleRoleFixtureFields spot-checks the title/organization Id-map
// golden fixture (docs/fork-plan/golden-fixtures/title-role.jscontact.json).
func TestTitleRoleFixtureFields(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "title-role.jscontact.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	c, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(c.Titles) != 2 {
		t.Fatalf("len(Titles) = %d, want 2", len(c.Titles))
	}
	// Id-map order is sorted-key order: "TITLE-1" < "TITLE-2".
	if c.Titles[0].ID != "TITLE-1" || c.Titles[0].Kind != "title" || c.Titles[0].Name != "Research Scientist" {
		t.Errorf("Titles[0] = %+v", c.Titles[0])
	}
	if c.Titles[1].ID != "TITLE-2" || c.Titles[1].Kind != "role" || c.Titles[1].OrganizationID != "ORG-1" {
		t.Errorf("Titles[1] = %+v", c.Titles[1])
	}
	if len(c.Organizations) != 1 || c.Organizations[0].ID != "ORG-1" || c.Organizations[0].Name != "ABC, Inc." {
		t.Errorf("Organizations = %+v", c.Organizations)
	}
}

// TestExplicitEmailIDsPreserved exercises the Id-map conversion: two emails
// with explicit map keys must survive Unmarshal -> Marshal with their IDs
// (and hence their map keys) intact and in sorted order.
func TestExplicitEmailIDsPreserved(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "emails-explicit-ids.jscontact.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	c, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(c.Emails) != 2 {
		t.Fatalf("len(Emails) = %d, want 2", len(c.Emails))
	}
	if c.Emails[0].ID != "E1" || c.Emails[0].Address != "alice@example.com" {
		t.Errorf("Emails[0] = %+v", c.Emails[0])
	}
	if len(c.Emails[0].Contexts) != 1 || c.Emails[0].Contexts[0] != "work" {
		t.Errorf("Emails[0].Contexts = %v, want [work]", c.Emails[0].Contexts)
	}
	if c.Emails[1].ID != "E2" || c.Emails[1].Address != "alice@home.example.com" {
		t.Errorf("Emails[1] = %+v", c.Emails[1])
	}

	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw2 map[string]any
	if err := json.Unmarshal(out, &raw2); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	emails, ok := raw2["emails"].(map[string]any)
	if !ok {
		t.Fatalf("emails is not a map in re-marshaled JSON: %v", raw2["emails"])
	}
	if _, ok := emails["E1"]; !ok {
		t.Error(`marshaled JSON missing "E1" key under emails`)
	}
	if _, ok := emails["E2"]; !ok {
		t.Error(`marshaled JSON missing "E2" key under emails`)
	}
}

// TestPhoneSyntheticKeyGeneration exercises codec rule 3's synthetic-key
// path via the fixture whose phone entry has an empty-string map key: on
// import ID becomes "" (the literal key), and on export the codec must
// generate "k0" (the slice's 0th, and only, element) since ID is empty.
func TestPhoneSyntheticKeyGeneration(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "phone-synthetic-key.jscontact.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	c, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(c.Phones) != 1 {
		t.Fatalf("len(Phones) = %d, want 1", len(c.Phones))
	}
	if c.Phones[0].ID != "" {
		t.Errorf("Phones[0].ID = %q, want empty (literal map key)", c.Phones[0].ID)
	}
	if c.Phones[0].Number != "+1-555-0100" {
		t.Errorf("Phones[0].Number = %q", c.Phones[0].Number)
	}

	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	phones, ok := decoded["phones"].(map[string]any)
	if !ok {
		t.Fatalf("phones is not a map: %v", decoded["phones"])
	}
	if _, ok := phones["k0"]; !ok {
		t.Errorf(`marshaled JSON missing synthetic "k0" key under phones; got keys %v`, keysOf(phones))
	}
}

// TestSyntheticKeyGenerationFromGoValue constructs a Card directly in Go
// (bypassing JSON entirely) with a collection element whose ID is empty, and
// checks Marshal assigns the synthetic "k<index>" key — the same rule, but
// exercised without going through Unmarshal first.
func TestSyntheticKeyGenerationFromGoValue(t *testing.T) {
	c := &Card{
		UID: "synthetic-from-go",
		Emails: []EmailAddress{
			{Address: "first@example.com"}, // ID empty -> "k0"
			{ID: "e2", Address: "second@example.com"},
			{Address: "third@example.com"}, // ID empty -> "k2"
		},
	}
	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	emails, ok := decoded["emails"].(map[string]any)
	if !ok {
		t.Fatalf("emails is not a map: %v", decoded["emails"])
	}
	for _, want := range []string{"k0", "e2", "k2"} {
		if _, ok := emails[want]; !ok {
			t.Errorf("missing expected key %q; got keys %v", want, keysOf(emails))
		}
	}
}

// TestContextsAndKeywordsBooleanSets exercises codec rule 4: Card.keywords
// (a Card-level boolean-set) and Address.contexts (a nested boolean-set)
// both convert cleanly between the wire {"x":true} map shape and Go []string.
func TestContextsAndKeywordsBooleanSets(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "contexts-and-keywords.jscontact.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	c, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	wantKeywords := []string{"family", "vip"} // sorted
	sortedKeywords := append([]string(nil), c.Keywords...)
	sort.Strings(sortedKeywords)
	if !equalStrings(sortedKeywords, wantKeywords) {
		t.Errorf("Keywords = %v, want (sorted) %v", c.Keywords, wantKeywords)
	}

	if len(c.Addresses) != 1 {
		t.Fatalf("len(Addresses) = %d, want 1", len(c.Addresses))
	}
	addr := c.Addresses[0]
	if addr.ID != "ADR-1" {
		t.Errorf("Addresses[0].ID = %q, want ADR-1", addr.ID)
	}
	sortedContexts := append([]string(nil), addr.Contexts...)
	sort.Strings(sortedContexts)
	if !equalStrings(sortedContexts, []string{"private", "work"}) {
		t.Errorf("Addresses[0].Contexts = %v, want (sorted) [private work]", addr.Contexts)
	}
	if len(addr.Components) != 1 || addr.Components[0].Kind != "locality" || addr.Components[0].Value != "Springfield" {
		t.Errorf("Addresses[0].Components = %+v", addr.Components)
	}
}

// TestRelatedToKeyedByValueMap exercises Card.relatedTo: a keyed-by-value map
// (RFC 9553 §3: "relatedTo = String[Relation]") where the map key is the
// related entity's URI/text (Relation.Target), not a synthetic id, plus
// Relation's own "relation" boolean-set field.
func TestRelatedToKeyedByValueMap(t *testing.T) {
	c := &Card{
		UID: "related-to-example",
		RelatedTo: []Relation{
			{Target: "urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af", Relation: []string{"friend", "colleague"}},
		},
	}
	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	relatedTo, ok := decoded["relatedTo"].(map[string]any)
	if !ok {
		t.Fatalf("relatedTo is not a map: %v", decoded["relatedTo"])
	}
	rel, ok := relatedTo["urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af"].(map[string]any)
	if !ok {
		t.Fatalf("relatedTo missing expected target key; got %v", keysOf(relatedTo))
	}
	relSet, ok := rel["relation"].(map[string]any)
	if !ok {
		t.Fatalf("relation is not a map: %v", rel["relation"])
	}
	if relSet["friend"] != true || relSet["colleague"] != true {
		t.Errorf("relation set = %v, want friend+colleague both true", relSet)
	}

	// Round trip back through Unmarshal and confirm Target survives as the
	// element's Go field (not marshaled as its own JSON property).
	back, err := Unmarshal(out)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(back.RelatedTo) != 1 || back.RelatedTo[0].Target != "urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af" {
		t.Errorf("RelatedTo = %+v", back.RelatedTo)
	}
}

// TestTypeDefaultedAndValidated exercises codec rule 1: @type is defaulted
// when absent, and validated (error) when present but wrong.
func TestTypeDefaultedAndValidated(t *testing.T) {
	t.Run("defaulted when absent", func(t *testing.T) {
		raw := []byte(`{"uid":"no-type-example","titles":{"T1":{"name":"Engineer"}}}`)
		c, err := Unmarshal(raw)
		if err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if c.Type != "Card" {
			t.Errorf("Card.Type = %q, want defaulted Card", c.Type)
		}
		if len(c.Titles) != 1 || c.Titles[0].Type != "Title" {
			t.Errorf("Titles = %+v, want Type defaulted to Title", c.Titles)
		}
	})

	t.Run("validated when present and wrong", func(t *testing.T) {
		raw := []byte(`{"@type":"NotACard","uid":"bad-type-example"}`)
		if _, err := Unmarshal(raw); err == nil {
			t.Fatal("Unmarshal succeeded with a mismatched top-level @type, want error")
		}
	})

	t.Run("validated when present and wrong inside an Id-map element", func(t *testing.T) {
		raw := []byte(`{"@type":"Card","titles":{"T1":{"@type":"Organization","name":"Engineer"}}}`)
		if _, err := Unmarshal(raw); err == nil {
			t.Fatal("Unmarshal succeeded with a mismatched nested @type, want error")
		}
	})
}

// TestCardVersionRules exercises codec rule 2: version is forced to "1.0" on
// export regardless of what's on the Go value, and any value is accepted on
// import.
func TestCardVersionRules(t *testing.T) {
	c := &Card{UID: "version-example", Version: "bogus"}
	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if decoded["version"] != "1.0" {
		t.Errorf(`version = %v, want "1.0" forced on export`, decoded["version"])
	}

	imported, err := Unmarshal([]byte(`{"@type":"Card","version":"99.9"}`))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if imported.Version != "99.9" {
		t.Errorf("imported Version = %q, want 99.9 accepted as-is", imported.Version)
	}
}

// TestMarshalNilCard checks Marshal's nil guard.
func TestMarshalNilCard(t *testing.T) {
	if _, err := Marshal(nil); err == nil {
		t.Fatal("Marshal(nil) succeeded, want error")
	}
}

// TestUnmarshalUnknownFieldsIgnored checks codec rule 5: unrecognized
// top-level/object properties must not cause Unmarshal to error.
func TestUnmarshalUnknownFieldsIgnored(t *testing.T) {
	raw := []byte(`{
		"@type": "Card",
		"uid": "unknown-fields-example",
		"totallyUnknownTopLevelProp": {"foo": "bar"},
		"emails": {
			"E1": {"@type": "EmailAddress", "address": "a@example.com", "somethingUnknown": 42}
		}
	}`)
	c, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal should not error on unknown fields: %v", err)
	}
	if len(c.Emails) != 1 || c.Emails[0].Address != "a@example.com" {
		t.Errorf("Emails = %+v", c.Emails)
	}
}

// TestUnmarshalUnknownNestedFieldsPreserved is the codec-level regression
// test for the defect this fix addresses: an unrecognized JSON key nested
// inside a known object (here, one emails{} map entry) must survive
// Unmarshal -> Marshal, not be silently discarded the way Go's default
// json.Unmarshal behavior would discard it. This complements
// TestUnmarshalUnknownFieldsIgnored (which only asserts "does not error");
// this test asserts the stronger "is preserved".
func TestUnmarshalUnknownNestedFieldsPreserved(t *testing.T) {
	raw := []byte(`{
		"@type": "Card",
		"uid": "unknown-nested-fields-example",
		"emails": {
			"E1": {"@type": "EmailAddress", "address": "a@example.com", "somethingUnknown": 42}
		}
	}`)
	c, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal should not error on unknown nested fields: %v", err)
	}
	if len(c.Emails) != 1 || c.Emails[0].Address != "a@example.com" {
		t.Fatalf("Emails = %+v", c.Emails)
	}

	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	emails, ok := decoded["emails"].(map[string]any)
	if !ok {
		t.Fatalf("emails is not a map: %v", decoded["emails"])
	}
	e1, ok := emails["E1"].(map[string]any)
	if !ok {
		t.Fatalf("emails.E1 is not an object: %v", emails["E1"])
	}
	if got, want := e1["somethingUnknown"], float64(42); got != want {
		t.Errorf(`re-marshaled emails.E1["somethingUnknown"] = %v, want %v (unrecognized nested key must round-trip)`, got, want)
	}
	if e1["address"] != "a@example.com" {
		t.Errorf(`re-marshaled emails.E1["address"] = %v, want "a@example.com" (mapped field must be unaffected)`, e1["address"])
	}

	// Fixed-point check: a second Unmarshal/Marshal pass must reproduce the
	// same bytes (the WP-30a acceptance criterion), confirming the captured
	// "somethingUnknown" key isn't just preserved once but stably so.
	c2, err := Unmarshal(out)
	if err != nil {
		t.Fatalf("second Unmarshal: %v", err)
	}
	out2, err := Marshal(c2)
	if err != nil {
		t.Fatalf("second Marshal: %v", err)
	}
	if !bytes.Equal(out, out2) {
		t.Fatalf("Marshal(Unmarshal(x)) is not a fixed point with a nested unknown key present\nfirst:  %s\nsecond: %s", out, out2)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
