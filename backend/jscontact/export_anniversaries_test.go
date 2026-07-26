package jscontact

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage(
		"anniversary.birth", "anniversary.wedding", "anniversary.death",
		"anniversary.place.birth", "anniversary.place.death",
	)
}

func TestExport_AnniversaryBirth(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "anniversary-birth-example",
		Anniversaries: []contactmodel.Anniversary{
			{ID: "A1", Kind: "birth", Date: contactmodel.AnniversaryDate{
				Partial: &contactmodel.PartialDate{Year: intPtr(1985), Month: intPtr(4), Day: intPtr(12)},
			}},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/kind", "birth")
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/date/year", 1985)
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/date/month", 4)
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/date/day", 12)
}

func TestExport_AnniversaryWedding(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "anniversary-wedding-example",
		Anniversaries: []contactmodel.Anniversary{
			{ID: "A1", Kind: "wedding", Date: contactmodel.AnniversaryDate{
				Partial: &contactmodel.PartialDate{Year: intPtr(2009), Month: intPtr(6), Day: intPtr(20)},
			}},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/kind", "wedding")
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/date/year", 2009)
}

func TestExport_AnniversaryDeath(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "anniversary-death-example",
		Anniversaries: []contactmodel.Anniversary{
			{ID: "A1", Kind: "death", Date: contactmodel.AnniversaryDate{
				Partial: &contactmodel.PartialDate{Year: intPtr(1996), Month: intPtr(4), Day: intPtr(15)},
			}},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/kind", "death")
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/date/year", 1996)
}

func TestExport_AnniversaryPlaceBirth(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "anniversary-place-birth-example",
		Anniversaries: []contactmodel.Anniversary{
			{ID: "A1", Kind: "birth", Place: &contactmodel.Address{Full: "Babies'R'Us Hospital"}},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/place/full", "Babies'R'Us Hospital")
}

func TestExport_AnniversaryPlaceDeath(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "anniversary-place-death-example",
		Anniversaries: []contactmodel.Anniversary{
			{ID: "A1", Kind: "death", Place: &contactmodel.Address{Full: "Aboard the Titanic, near Newfoundland"}},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/anniversaries/A1/place/full", "Aboard the Titanic, near Newfoundland")
}
