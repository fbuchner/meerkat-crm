package vcard4

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concepts: name.full, name.surname, name.given, name.given2, name.title,
// name.credential, name.surname2, name.generation, name.phonetic.
// Golden fixtures: rfc6350-baseline.v4.vcf (plain FN/N), n-expanded.v4.vcf
// (7-component N, RFC 9554 §2.2), derived-fn.v4.vcf (FN DERIVED=TRUE, RFC
// 9554 §4.4), phonetic-n.v4.vcf (N PHONETIC/SCRIPT/ALTID, RFC 9554 §4.6).
func init() {
	registerImportCoverage(
		"name.full", "name.surname", "name.given", "name.given2",
		"name.title", "name.credential", "name.surname2", "name.generation",
		"name.phonetic",
	)
}

func nameComponent(t *testing.T, comps []contactmodel.NameComponent, kind string) string {
	t.Helper()
	for _, c := range comps {
		if c.Kind == kind {
			return c.Value
		}
	}
	return ""
}

func TestImport_NameFull_Baseline(t *testing.T) {
	raw := rfctest.LoadFixture("rfc6350-baseline.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil || rec.Card.Name.Full != "J. Doe" {
		t.Fatalf("Name.Full = %+v, want %q", rec.Card.Name, "J. Doe")
	}
}

func TestImport_NameComponents_Expanded(t *testing.T) {
	raw := rfctest.LoadFixture("n-expanded.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil {
		t.Fatalf("Name = nil")
	}
	comps := rec.Card.Name.Components
	cases := map[string]string{
		"surname":    "Stevenson",
		"given":      "John",
		"given2":     "Philip,Paul",
		"title":      "Dr.",
		"credential": "Jr.,M.D.,A.C.P.",
		"generation": "Jr.",
	}
	for kind, want := range cases {
		if got := nameComponent(t, comps, kind); got != want {
			t.Errorf("component[%s] = %q, want %q", kind, got, want)
		}
	}
	if rec.Card.Name.Full != "John Philip Paul Stevenson Jr." {
		t.Errorf("Name.Full = %q", rec.Card.Name.Full)
	}
}

func TestImport_NameSurname2(t *testing.T) {
	// name.surname2 has no dedicated golden fixture; n-expanded.v4.vcf's N
	// value has an empty 6th (surname2) component (Stevenson;John;Philip,Paul;
	// Dr.;Jr.,M.D.,A.C.P.;;Jr.), so exercise the concept directly against a
	// minimal hand-built card whose value/positions are RFC-9554-conformant
	// (7-component N, position 6 = SecondarySurname).
	raw := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\nUID:surname2-example\r\nFN:Test\r\nN:Garcia;Maria;;;;Torres;\r\nEND:VCARD\r\n")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil {
		t.Fatalf("Name = nil")
	}
	var got string
	for _, c := range rec.Card.Name.Components {
		if c.Kind == "surname2" {
			got = c.Value
		}
	}
	if got != "Torres" {
		t.Errorf("surname2 = %q, want %q", got, "Torres")
	}
}

func TestImport_NameDerivedFN_NotAuthoritative(t *testing.T) {
	raw := rfctest.LoadFixture("derived-fn.v4.vcf")
	rec, diags, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil {
		t.Fatalf("Name = nil")
	}
	if rec.Card.Name.Full != "" {
		t.Errorf("Name.Full = %q, want empty (DERIVED=TRUE must not be imported as authoritative)", rec.Card.Name.Full)
	}
	found := false
	for _, d := range diags {
		if d.Concept == "name.full" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a name.full Diagnostic noting the dropped DERIVED FN, got %+v", diags)
	}
	var given, additional, prefix string
	for _, c := range rec.Card.Name.Components {
		switch c.Kind {
		case "given":
			given = c.Value
		case "given2":
			additional = c.Value
		case "title":
			prefix = c.Value
		}
	}
	if given != "John" || additional != "Quinlan" || prefix != "Mr." {
		t.Errorf("components = given=%q given2=%q title=%q, want John/Quinlan/Mr.", given, additional, prefix)
	}
}

func TestImport_NamePhonetic(t *testing.T) {
	raw := rfctest.LoadFixture("phonetic-n.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil {
		t.Fatalf("Name = nil")
	}
	if rec.Card.Name.PhoneticSystem != "jyut" {
		t.Errorf("PhoneticSystem = %q, want jyut", rec.Card.Name.PhoneticSystem)
	}
	if rec.Card.Name.PhoneticScript != "Latn" {
		t.Errorf("PhoneticScript = %q, want Latn", rec.Card.Name.PhoneticScript)
	}
	var surnamePhon, givenPhon string
	for _, c := range rec.Card.Name.Components {
		switch c.Kind {
		case "surname":
			surnamePhon = c.Phonetic
		case "given":
			givenPhon = c.Phonetic
		}
	}
	if surnamePhon != "syun1" {
		t.Errorf("surname.Phonetic = %q, want syun1", surnamePhon)
	}
	if givenPhon != "zung1saan1" {
		t.Errorf("given.Phonetic = %q, want zung1saan1", givenPhon)
	}
}
