package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concept covered: note.
func init() {
	registerExportCoverage("note")
}

func TestExport_Note(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Notes: []contactmodel.Note{{Note: "Met at IETF."}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropNote, nil, "Met at IETF.")
}
