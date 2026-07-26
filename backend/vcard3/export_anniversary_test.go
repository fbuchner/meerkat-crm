package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concept covered: anniversary.birth.
func init() {
	registerExportCoverage("anniversary.birth")
}

func TestExport_AnniversaryBirth(t *testing.T) {
	year, month, day := 1996, 4, 15
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{
			Kind: "birth",
			Date: contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Year: &year, Month: &month, Day: &day}},
		}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropBday, nil, "1996-04-15")
}
