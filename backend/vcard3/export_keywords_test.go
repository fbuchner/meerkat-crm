package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concept covered (coverage_test.go): keywords.

func TestExport_Keywords(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Keywords: []string{"Family", "Friends"}}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropCategories, nil, "Family,Friends")
}
