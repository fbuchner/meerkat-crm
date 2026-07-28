package vcard4

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage(
		"anniversary.birth", "anniversary.wedding", "anniversary.death",
		"anniversary.place.birth", "anniversary.place.death",
	)
}

func partialDate(y, m, d int) contactmodel.AnniversaryDate {
	return contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Year: intPtr(y), Month: intPtr(m), Day: intPtr(d)}}
}

func TestExport_AnniversaryBirth(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{Kind: "birth", Date: partialDate(1985, 4, 12)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "BDAY", nil, "1985-04-12")
}

func TestExport_AnniversaryWedding(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{Kind: "wedding", Date: partialDate(2009, 8, 16)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "ANNIVERSARY", nil, "2009-08-16")
}

func TestExport_AnniversaryDeath(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{Kind: "death", Date: partialDate(1996, 4, 15)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "DEATHDATE", nil, "1996-04-15")
}

func TestExport_AnniversaryPlaceBirth(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{Kind: "birth", Place: &contactmodel.Address{Full: "Babies'R'Us Hospital"}}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "BIRTHPLACE", nil, "Babies'R'Us Hospital")
}

func TestExport_AnniversaryPlaceDeath(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{Kind: "death", Place: &contactmodel.Address{Coordinates: "geo:41.7325,-49.9469"}}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "DEATHPLACE", map[string]string{"VALUE": "uri"}, "geo:41.7325,-49.9469")
}
