package jscontact

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage(
		"name.full", "name.surname", "name.given", "name.given2",
		"name.title", "name.credential", "name.surname2", "name.generation",
		"name.phonetic",
	)
}

func TestExport_NameGivenSurname(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "22B2C7DF-9120-4969-8460-05956FE6B065",
		Name: &contactmodel.Name{
			Components: []contactmodel.NameComponent{
				{Kind: "given", Value: "John"},
				{Kind: "surname", Value: "Doe"},
			},
			IsOrdered: boolPtr(true),
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/name/components/0/kind", "given")
	rfctest.AssertJSONPointer(t, out, "/name/components/0/value", "John")
	rfctest.AssertJSONPointer(t, out, "/name/components/1/kind", "surname")
	rfctest.AssertJSONPointer(t, out, "/name/components/1/value", "Doe")
}

func TestExport_NameFull(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:  "name-full-example",
		Name: &contactmodel.Name{Full: "Dr. John Doe Jr."},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/name/full", "Dr. John Doe Jr.")
}

func TestExport_NameGiven2TitleCredentialSurname2Generation(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "name-expanded-example",
		Name: &contactmodel.Name{
			Components: []contactmodel.NameComponent{
				{Kind: "title", Value: "Dr."},
				{Kind: "given", Value: "John"},
				{Kind: "given2", Value: "Philip"},
				{Kind: "surname", Value: "Doe"},
				{Kind: "surname2", Value: "Public"},
				{Kind: "credential", Value: "Jr."},
				{Kind: "generation", Value: "III"},
			},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	wantByPos := []struct {
		idx  int
		kind string
		val  string
	}{
		{0, "title", "Dr."}, {1, "given", "John"}, {2, "given2", "Philip"},
		{3, "surname", "Doe"}, {4, "surname2", "Public"}, {5, "credential", "Jr."},
		{6, "generation", "III"},
	}
	for _, w := range wantByPos {
		rfctest.AssertJSONPointer(t, out, sprintfPointer("/name/components/%d/kind", w.idx), w.kind)
		rfctest.AssertJSONPointer(t, out, sprintfPointer("/name/components/%d/value", w.idx), w.val)
	}
}

func TestExport_NamePhonetic(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:  "name-phonetic-example",
		Name: &contactmodel.Name{Full: "Bocelli", PhoneticScript: "Latn", PhoneticSystem: "ipa"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/name/phoneticScript", "Latn")
}
