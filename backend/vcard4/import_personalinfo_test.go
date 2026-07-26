package vcard4

import (
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: expertise, hobby, interest.
// Golden fixtures: expertise.v4.vcf / hobby.v4.vcf / interest.v4.vcf, each a
// verbatim RFC 6715 example.
func init() {
	registerImportCoverage("expertise", "hobby", "interest")
}

func TestImport_Expertise(t *testing.T) {
	raw := rfctest.LoadFixture("expertise.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.PersonalInfo) != 1 {
		t.Fatalf("PersonalInfo = %+v", rec.Card.PersonalInfo)
	}
	pi := rec.Card.PersonalInfo[0]
	if pi.Kind != "expertise" || pi.Value != "chemistry" {
		t.Errorf("PersonalInfo[0] = %+v", pi)
	}
	// vCard EXPERTISE LEVEL=expert <-> neutral level "high" (20.4 personalinfo transform).
	if pi.Level != "high" {
		t.Errorf("Level = %q, want high (expert->high)", pi.Level)
	}
	if pi.ListAs == nil || *pi.ListAs != 1 {
		t.Errorf("ListAs = %v, want 1", pi.ListAs)
	}
}

func TestImport_Hobby(t *testing.T) {
	raw := rfctest.LoadFixture("hobby.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.PersonalInfo) != 1 {
		t.Fatalf("PersonalInfo = %+v", rec.Card.PersonalInfo)
	}
	pi := rec.Card.PersonalInfo[0]
	if pi.Kind != "hobby" || pi.Value != "reading" || pi.Level != "high" {
		t.Errorf("PersonalInfo[0] = %+v", pi)
	}
}

func TestImport_Interest(t *testing.T) {
	raw := rfctest.LoadFixture("interest.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.PersonalInfo) != 1 {
		t.Fatalf("PersonalInfo = %+v", rec.Card.PersonalInfo)
	}
	pi := rec.Card.PersonalInfo[0]
	if pi.Kind != "interest" || pi.Value != "r&b music" || pi.Level != "medium" {
		t.Errorf("PersonalInfo[0] = %+v", pi)
	}
}
