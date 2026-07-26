package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concept covered: email.
func init() {
	registerExportCoverage("email")
}

func TestExport_Email(t *testing.T) {
	pref := 1
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Emails: []contactmodel.Email{{Address: "Frank_Dawson@Lotus.com", Pref: &pref, Contexts: []string{"work"}}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropEmail, map[string]string{"TYPE": "INTERNET"}, "Frank_Dawson@Lotus.com")
	rfctest.AssertVCardLine(t, out, PropEmail, map[string]string{"TYPE": "WORK"}, "Frank_Dawson@Lotus.com")
	rfctest.AssertVCardLine(t, out, PropEmail, map[string]string{"TYPE": "PREF"}, "Frank_Dawson@Lotus.com")
}
