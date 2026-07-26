package vcard4

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("note", "keywords")
}

func TestExport_NoteAuthor(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Notes: []contactmodel.Note{{
			Note:    "This is some note.",
			Author:  &contactmodel.Author{Name: "John Doe", URI: "mailto:john@example.com"},
			Created: &contactmodel.Timestamp{UTC: "2022-11-22T15:18:23Z"},
		}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "NOTE", map[string]string{
		"AUTHOR": "mailto:john@example.com", "AUTHOR-NAME": "John Doe", "CREATED": "20221122T151823Z",
	}, "This is some note.")
}

func TestExport_Keywords(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Keywords: []string{"family", "work"}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "CATEGORIES", nil, "family,work")
}
