package vcard4

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("nickname")
}

func TestExport_Nickname(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Nicknames: []contactmodel.Nickname{{Name: "Johnny", Contexts: []string{"work"}, Pref: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "NICKNAME", map[string]string{"PREF": "1", "TYPE": "work"}, "Johnny")
}
