package vcard3

import "testing"

// Concepts covered: title, role.
func init() {
	registerImportCoverage("title", "role")
}

const titlesImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"TITLE:Vice President\n" +
	"ROLE:Executive\n" +
	"END:VCARD\n"

func TestImport_Title(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(titlesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	found := false
	for _, ti := range rec.Card.Titles {
		if ti.Kind == "title" && ti.Name == "Vice President" {
			found = true
		}
	}
	if !found {
		t.Errorf("Titles = %+v, want a title=%q entry", rec.Card.Titles, "Vice President")
	}
}

func TestImport_Role(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(titlesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	found := false
	for _, ti := range rec.Card.Titles {
		if ti.Kind == "role" && ti.Name == "Executive" {
			found = true
		}
	}
	if !found {
		t.Errorf("Titles = %+v, want a role=%q entry", rec.Card.Titles, "Executive")
	}
}
