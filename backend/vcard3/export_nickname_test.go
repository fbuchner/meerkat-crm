package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concept covered (coverage_test.go): nickname.

func TestExport_Nickname(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Nicknames: []contactmodel.Nickname{{Name: "Johnny", Contexts: []string{"work"}}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropNickname, map[string]string{"TYPE": "WORK"}, "Johnny")
}
