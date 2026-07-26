package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concepts covered: title, role.
func init() {
	registerExportCoverage("title", "role")
}

func TestExport_Title(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Titles: []contactmodel.Title{{Kind: "title", Name: "Vice President"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropTitle, nil, "Vice President")
}

func TestExport_Role(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Titles: []contactmodel.Title{{Kind: "role", Name: "Executive"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropRole, nil, "Executive")
}
