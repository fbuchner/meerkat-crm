package jscontact

import "testing"

// Concepts: anniversary.birth, anniversary.wedding, anniversary.death,
// anniversary.place.birth, anniversary.place.death.
// All share js_ptr /anniversaries/{id}/date or /anniversaries/{id}/place,
// disambiguated by Anniversary.Kind ∈ birth|wedding|death.
func init() {
	registerImportCoverage(
		"anniversary.birth", "anniversary.wedding", "anniversary.death",
		"anniversary.place.birth", "anniversary.place.death",
	)
}

func TestImport_AnniversaryBirth(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "anniversary-birth-example",
		"anniversaries": {
			"A1": { "@type": "Anniversary", "kind": "birth", "date": { "@type": "PartialDate", "year": 1985, "month": 4, "day": 12 } }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 {
		t.Fatalf("len(Anniversaries) = %d, want 1", len(rec.Card.Anniversaries))
	}
	a := rec.Card.Anniversaries[0]
	if a.Kind != "birth" {
		t.Errorf("Kind = %q, want birth", a.Kind)
	}
	if a.Date.Partial == nil || *a.Date.Partial.Year != 1985 || *a.Date.Partial.Month != 4 || *a.Date.Partial.Day != 12 {
		t.Errorf("Date.Partial = %+v", a.Date.Partial)
	}
}

func TestImport_AnniversaryWedding(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "anniversary-wedding-example",
		"anniversaries": {
			"A1": { "@type": "Anniversary", "kind": "wedding", "date": { "@type": "PartialDate", "year": 2009, "month": 6, "day": 20 } }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 || rec.Card.Anniversaries[0].Kind != "wedding" {
		t.Fatalf("Anniversaries = %+v", rec.Card.Anniversaries)
	}
	d := rec.Card.Anniversaries[0].Date.Partial
	if d == nil || *d.Year != 2009 || *d.Month != 6 || *d.Day != 20 {
		t.Errorf("Date.Partial = %+v", d)
	}
}

func TestImport_AnniversaryDeath(t *testing.T) {
	// RFC 6474 §2.3's DEATHDATE:19960415 worked example, expressed in JSContact.
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "anniversary-death-example",
		"anniversaries": {
			"A1": { "@type": "Anniversary", "kind": "death", "date": { "@type": "PartialDate", "year": 1996, "month": 4, "day": 15 } }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 || rec.Card.Anniversaries[0].Kind != "death" {
		t.Fatalf("Anniversaries = %+v", rec.Card.Anniversaries)
	}
	d := rec.Card.Anniversaries[0].Date.Partial
	if d == nil || *d.Year != 1996 || *d.Month != 4 || *d.Day != 15 {
		t.Errorf("Date.Partial = %+v", d)
	}
}

func TestImport_AnniversaryPlaceBirth(t *testing.T) {
	// RFC 6474 §2.1's BIRTHPLACE:Babies'R'Us Hospital worked example.
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "anniversary-place-birth-example",
		"anniversaries": {
			"A1": {
				"@type": "Anniversary", "kind": "birth",
				"date": { "@type": "PartialDate" },
				"place": { "@type": "Address", "full": "Babies'R'Us Hospital" }
			}
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 {
		t.Fatalf("len(Anniversaries) = %d, want 1", len(rec.Card.Anniversaries))
	}
	p := rec.Card.Anniversaries[0].Place
	if p == nil || p.Full != "Babies'R'Us Hospital" {
		t.Errorf("Place = %+v", p)
	}
}

func TestImport_AnniversaryPlaceDeath(t *testing.T) {
	// RFC 6474 §2.2's DEATHPLACE:Aboard the Titanic\, near Newfoundland worked example.
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "anniversary-place-death-example",
		"anniversaries": {
			"A1": {
				"@type": "Anniversary", "kind": "death",
				"date": { "@type": "PartialDate" },
				"place": { "@type": "Address", "full": "Aboard the Titanic, near Newfoundland" }
			}
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Anniversaries) != 1 {
		t.Fatalf("len(Anniversaries) = %d, want 1", len(rec.Card.Anniversaries))
	}
	p := rec.Card.Anniversaries[0].Place
	if p == nil || p.Full != "Aboard the Titanic, near Newfoundland" {
		t.Errorf("Place = %+v", p)
	}
}
