package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concept covered (coverage_test.go): phone.

func TestExport_Phone(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Phones: []contactmodel.Phone{{
			Number: "+1-919-676-9515", Features: []string{"voice"}, Contexts: []string{"work"},
		}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropTel, map[string]string{"TYPE": "VOICE"}, "+1-919-676-9515")
	rfctest.AssertVCardLine(t, out, PropTel, map[string]string{"TYPE": "WORK"}, "+1-919-676-9515")
}
