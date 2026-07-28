package jscontact

import (
	"testing"

	"mycorrhizal/internal/rfctest"
)

// Concepts: name.full, name.surname, name.given, name.given2, name.title,
// name.credential, name.surname2, name.generation, name.phonetic.
// All rows: js_ptr /name/full or /name/components (the components sub-rows
// share one js_ptr, disambiguated by NameComponent.Kind) or
// /name/phoneticScript; transform n_component (components) / identity
// (full, phonetic).
func init() {
	registerImportCoverage(
		"name.full", "name.surname", "name.given", "name.given2",
		"name.title", "name.credential", "name.surname2", "name.generation",
		"name.phonetic",
	)
}

func TestImport_NameGivenSurname(t *testing.T) {
	// johndoe.jscontact.json (RFC 9553 Fig. 6): components = [given:John, surname:Doe].
	raw := rfctest.LoadFixture("johndoe.jscontact.json")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil {
		t.Fatal("Card.Name is nil")
	}
	if len(rec.Card.Name.Components) != 2 {
		t.Fatalf("len(Components) = %d, want 2", len(rec.Card.Name.Components))
	}
	if rec.Card.Name.Components[0].Kind != "given" || rec.Card.Name.Components[0].Value != "John" {
		t.Errorf("Components[0] = %+v", rec.Card.Name.Components[0])
	}
	if rec.Card.Name.Components[1].Kind != "surname" || rec.Card.Name.Components[1].Value != "Doe" {
		t.Errorf("Components[1] = %+v", rec.Card.Name.Components[1])
	}
	if rec.Card.Name.IsOrdered == nil || !*rec.Card.Name.IsOrdered {
		t.Errorf("Name.IsOrdered = %v, want true", rec.Card.Name.IsOrdered)
	}
}

func TestImport_NameFull(t *testing.T) {
	raw := []byte(`{"@type":"Card","version":"1.0","uid":"name-full-example","name":{"@type":"Name","full":"Dr. John Doe Jr."}}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil || rec.Card.Name.Full != "Dr. John Doe Jr." {
		t.Errorf("Card.Name.Full = %+v", rec.Card.Name)
	}
}

func TestImport_NameGiven2TitleCredentialSurname2Generation(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "name-expanded-example",
		"name": {
			"@type": "Name",
			"components": [
				{ "kind": "title", "value": "Dr." },
				{ "kind": "given", "value": "John" },
				{ "kind": "given2", "value": "Philip" },
				{ "kind": "surname", "value": "Doe" },
				{ "kind": "surname2", "value": "Public" },
				{ "kind": "credential", "value": "Jr." },
				{ "kind": "generation", "value": "III" }
			]
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil {
		t.Fatal("Card.Name is nil")
	}
	byKind := map[string]string{}
	for _, c := range rec.Card.Name.Components {
		byKind[c.Kind] = c.Value
	}
	want := map[string]string{
		"title": "Dr.", "given": "John", "given2": "Philip", "surname": "Doe",
		"surname2": "Public", "credential": "Jr.", "generation": "III",
	}
	for k, v := range want {
		if byKind[k] != v {
			t.Errorf("Components[kind=%s] = %q, want %q", k, byKind[k], v)
		}
	}
}

func TestImport_NamePhonetic(t *testing.T) {
	raw := []byte(`{"@type":"Card","version":"1.0","uid":"name-phonetic-example","name":{"@type":"Name","full":"Bocelli","phoneticScript":"Latn","phoneticSystem":"ipa"}}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Name == nil || rec.Card.Name.PhoneticScript != "Latn" {
		t.Errorf("Card.Name.PhoneticScript = %+v", rec.Card.Name)
	}
}
