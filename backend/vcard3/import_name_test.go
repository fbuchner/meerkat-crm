package vcard3

import "testing"

// Concepts covered: name.full, name.surname, name.given, name.given2,
// name.title, name.credential.
//
// Fixture value from docs/specs/rfc2426-v3-baseline.md §2's N example:
// N:Public;John;Quinlan;Mr.;Esq.
func init() {
	registerImportCoverage(
		"name.full", "name.surname", "name.given", "name.given2",
		"name.title", "name.credential",
	)
}

const nameImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Mr. John Q. Public\\, Esq.\n" +
	"N:Public;John;Quinlan;Mr.;Esq.\n" +
	"END:VCARD\n"

func TestImport_NameFull(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(nameImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil {
		t.Fatalf("Name = nil")
	}
	want := "Mr. John Q. Public, Esq."
	if rec.Card.Name.Full != want {
		t.Errorf("Name.Full = %q, want %q", rec.Card.Name.Full, want)
	}
}

func TestImport_NameComponents(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(nameImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil {
		t.Fatalf("Name = nil")
	}
	want := map[string]string{
		"surname":    "Public",
		"given":      "John",
		"given2":     "Quinlan",
		"title":      "Mr.",
		"credential": "Esq.",
	}
	got := map[string]string{}
	for _, c := range rec.Card.Name.Components {
		got[c.Kind] = c.Value
	}
	for kind, wantVal := range want {
		if got[kind] != wantVal {
			t.Errorf("component %q = %q, want %q", kind, got[kind], wantVal)
		}
	}
}
