package vcard4

import (
	"strings"
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: anniversary.birth, anniversary.wedding, anniversary.death,
// anniversary.place.birth, anniversary.place.death. All five have
// RFC-verbatim per-concept fixtures (RFC 6350 §4.3 DATE grammar / RFC 6474
// §2.1-2.3), no RFC 9553/9554-introduced-concept golden fixture exists for
// this group specifically (BDAY/ANNIVERSARY/DEATHDATE/BIRTHPLACE/DEATHPLACE
// are all pre-9554 or RFC-6474 properties).
func init() {
	registerImportCoverage(
		"anniversary.birth", "anniversary.wedding", "anniversary.death",
		"anniversary.place.birth", "anniversary.place.death",
	)
}

func TestImport_AnniversaryBirth(t *testing.T) {
	raw := rfctest.LoadFixture("anniversary-birth.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 {
		t.Fatalf("Anniversaries = %+v", rec.Card.Anniversaries)
	}
	a := rec.Card.Anniversaries[0]
	if a.Kind != "birth" {
		t.Errorf("Kind = %q", a.Kind)
	}
	pd := a.Date.Partial
	if pd == nil || pd.Year == nil || *pd.Year != 1985 || pd.Month == nil || *pd.Month != 4 || pd.Day == nil || *pd.Day != 12 {
		t.Errorf("Date.Partial = %+v, want 1985-04-12", pd)
	}
}

func TestImport_AnniversaryWedding(t *testing.T) {
	raw := rfctest.LoadFixture("anniversary-wedding.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 || rec.Card.Anniversaries[0].Kind != "wedding" {
		t.Fatalf("Anniversaries = %+v", rec.Card.Anniversaries)
	}
	pd := rec.Card.Anniversaries[0].Date.Partial
	if pd == nil || pd.Year == nil || *pd.Year != 2009 || pd.Month == nil || *pd.Month != 8 || pd.Day == nil || *pd.Day != 16 {
		t.Errorf("Date.Partial = %+v, want 2009-08-16", pd)
	}
}

func TestImport_AnniversaryDeath(t *testing.T) {
	raw := rfctest.LoadFixture("anniversary-death.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 || rec.Card.Anniversaries[0].Kind != "death" {
		t.Fatalf("Anniversaries = %+v", rec.Card.Anniversaries)
	}
	pd := rec.Card.Anniversaries[0].Date.Partial
	if pd == nil || pd.Year == nil || *pd.Year != 1996 || pd.Month == nil || *pd.Month != 4 || pd.Day == nil || *pd.Day != 15 {
		t.Errorf("Date.Partial = %+v, want 1996-04-15", pd)
	}
}

func TestImport_AnniversaryDeathValueText(t *testing.T) {
	// RFC 6474 §2.3's VALUE=text free-text variant has no neutral
	// AnniversaryDate representation (per the row's note); it must be
	// preserved via passthrough, not forced into a bogus PartialDate.
	raw := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\nUID:death-text\r\nFN:Test\r\nDEATHDATE;VALUE=text:circa 1800\r\nEND:VCARD\r\n")
	rec, diags, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 0 {
		t.Errorf("Anniversaries = %+v, want none (free-text DEATHDATE has no neutral home)", rec.Card.Anniversaries)
	}
	found := false
	for _, p := range rec.Passthrough.VCard {
		if strings.EqualFold(p.Name, "DEATHDATE") {
			found = true
		}
	}
	if !found {
		t.Errorf("Passthrough.VCard = %+v, want a DEATHDATE entry", rec.Passthrough.VCard)
	}
	if len(diags) == 0 {
		t.Errorf("expected a Diagnostic for the free-text DEATHDATE")
	}
}

func TestImport_AnniversaryPlaceBirth(t *testing.T) {
	raw := rfctest.LoadFixture("anniversary-place-birth.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 || rec.Card.Anniversaries[0].Place == nil {
		t.Fatalf("Anniversaries = %+v", rec.Card.Anniversaries)
	}
	if got := rec.Card.Anniversaries[0].Place.Full; got != "Babies'R'Us Hospital" {
		t.Errorf("Place.Full = %q", got)
	}
	if rec.Card.Anniversaries[0].Kind != "birth" {
		t.Errorf("Kind = %q", rec.Card.Anniversaries[0].Kind)
	}
}

func TestImport_AnniversaryPlaceDeath(t *testing.T) {
	raw := rfctest.LoadFixture("anniversary-place-death.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 || rec.Card.Anniversaries[0].Place == nil {
		t.Fatalf("Anniversaries = %+v", rec.Card.Anniversaries)
	}
	if got := rec.Card.Anniversaries[0].Place.Full; got != "Aboard the Titanic, near Newfoundland" {
		t.Errorf("Place.Full = %q", got)
	}
}
