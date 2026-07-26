package vcard3

import "testing"

// Concepts covered (coverage_test.go): org, org.unit.

const orgImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"ORG:Lotus Development Corporation;Sales;East Region\n" +
	"END:VCARD\n"

func TestImport_Org(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(orgImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Organizations) != 1 {
		t.Fatalf("Organizations = %+v, want 1 entry", rec.Card.Organizations)
	}
	if rec.Card.Organizations[0].Name != "Lotus Development Corporation" {
		t.Errorf("Name = %q, want %q", rec.Card.Organizations[0].Name, "Lotus Development Corporation")
	}
}

func TestImport_OrgUnit(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(orgImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	units := rec.Card.Organizations[0].Units
	if len(units) != 2 || units[0].Name != "Sales" || units[1].Name != "East Region" {
		t.Errorf("Units = %+v, want [Sales East Region]", units)
	}
}
