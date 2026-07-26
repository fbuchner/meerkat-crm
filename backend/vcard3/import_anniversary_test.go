package vcard3

import "testing"

// Concept covered (coverage_test.go): anniversary.birth.

const anniversaryBirthImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"BDAY:1996-04-15\n" +
	"END:VCARD\n"

func TestImport_AnniversaryBirth(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(anniversaryBirthImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	var found *int
	for _, a := range rec.Card.Anniversaries {
		if a.Kind == "birth" && a.Date.Partial != nil {
			found = a.Date.Partial.Year
		}
	}
	if found == nil || *found != 1996 {
		t.Errorf("birth anniversary year = %v, want 1996 (anniversaries=%+v)", found, rec.Card.Anniversaries)
	}
}
