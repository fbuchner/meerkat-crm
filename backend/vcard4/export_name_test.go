package vcard4

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

func TestExport_NameFull(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{Full: "J. Doe"}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "FN", nil, "J. Doe")
}

func TestExport_NameComponents(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{
		Full: "Dr. John Philip Paul Stevenson Jr.",
		Components: []contactmodel.NameComponent{
			{Kind: "surname", Value: "Stevenson"},
			{Kind: "given", Value: "John"},
			{Kind: "given2", Value: "Philip"},
			{Kind: "title", Value: "Dr."},
			{Kind: "credential", Value: "Jr."},
			{Kind: "surname2", Value: "Torres"},
			{Kind: "generation", Value: "Jr."},
		},
	}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	// n_component (20.4): N order = Family;Given;Additional;Prefix;Suffix;Surname2;Generation.
	rfctest.AssertVCardLine(t, out, "N", nil, "Stevenson;John;Philip;Dr.;Jr.;Torres;Jr.")
}

func TestExport_NameSurname2AndGeneration(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{
		Components: []contactmodel.NameComponent{
			{Kind: "surname", Value: "Garcia"},
			{Kind: "surname2", Value: "Torres"},
			{Kind: "generation", Value: "II"},
		},
	}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "N", nil, "Garcia;;;;;Torres;II")
}

func TestExport_NameDerivedFN(t *testing.T) {
	// Reproduces the golden fixture derived-fn.v4.vcf: N:;John;Quinlan;Mr.;
	// -> FN;DERIVED=TRUE:Mr. John Quinlan (RFC 9554 §4.4).
	rec := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{
		Components: []contactmodel.NameComponent{
			{Kind: "given", Value: "John"},
			{Kind: "given2", Value: "Quinlan"},
			{Kind: "title", Value: "Mr."},
		},
	}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "FN", map[string]string{"DERIVED": "TRUE"}, "Mr. John Quinlan")
}

func TestExport_NamePhonetic(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{
		PhoneticSystem: "jyut",
		PhoneticScript: "Latn",
		Components: []contactmodel.NameComponent{
			{Kind: "surname", Value: "孫", Phonetic: "syun1"},
			{Kind: "given", Value: "中山", Phonetic: "zung1saan1"},
		},
	}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "N",
		map[string]string{"PHONETIC": "jyut", "SCRIPT": "Latn"},
		"syun1;zung1saan1;;;;;")
}
