package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concepts covered (coverage_test.go): name.full, name.surname, name.given,
// name.given2, name.title, name.credential.

func nameRecord() *contactmodel.Record {
	return &contactmodel.Record{Card: contactmodel.Card{
		Name: &contactmodel.Name{
			Full: "Mr. John Q. Public, Esq.",
			Components: []contactmodel.NameComponent{
				{Kind: "surname", Value: "Public"},
				{Kind: "given", Value: "John"},
				{Kind: "given2", Value: "Quinlan"},
				{Kind: "title", Value: "Mr."},
				{Kind: "credential", Value: "Esq."},
			},
		},
	}}
}

func TestExport_NameFull(t *testing.T) {
	out, _, err := (Adapter{}).Export(nameRecord())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropFN, nil, "Mr. John Q. Public, Esq.")
}

func TestExport_NameComponents(t *testing.T) {
	out, _, err := (Adapter{}).Export(nameRecord())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropN, nil, `Public;John;Quinlan;Mr.;Esq.`)
}

func TestExport_NameFull_DerivedWhenEmpty(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Name: &contactmodel.Name{
			Components: []contactmodel.NameComponent{
				{Kind: "given", Value: "John"},
				{Kind: "surname", Value: "Public"},
			},
		},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropFN, nil, "John Public")
}
