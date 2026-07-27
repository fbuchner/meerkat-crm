package jscontact

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("note", "keywords")
}

func TestExport_NoteWithAuthor(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "note-author-example",
		Notes: []contactmodel.Note{
			{
				ID:      "N1",
				Note:    "This is some note.",
				Author:  &contactmodel.Author{Name: "John Doe", URI: "mailto:john@example.com"},
				Created: &contactmodel.Timestamp{UTC: "2022-11-22T15:18:23Z"},
			},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/notes/N1/note", "This is some note.")
	rfctest.AssertJSONPointer(t, out, "/notes/N1/author/name", "John Doe")
	rfctest.AssertJSONPointer(t, out, "/notes/N1/created/utc", "2022-11-22T15:18:23Z")
}

func TestExport_Keywords(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:      "keywords-example",
		Keywords: []string{"family", "work"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/keywords/family", true)
	rfctest.AssertJSONPointer(t, out, "/keywords/work", true)
}
