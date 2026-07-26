package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concepts covered: uid, prodid, updated.
func init() {
	registerExportCoverage("uid", "prodid", "updated")
}

func TestExport_UID(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{UID: "frank-dawson-uid-1"}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropUID, nil, "frank-dawson-uid-1")
}

func TestExport_ProdID(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{ProdID: "-//Meerkat//WP-50 Test//EN"}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropProdid, nil, "-//Meerkat//WP-50 Test//EN")
}

func TestExport_Updated(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Updated: &contactmodel.Timestamp{UTC: "2023-01-02T03:04:05Z"},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropRev, nil, "2023-01-02T03:04:05Z")
}

func TestExport_AlwaysHasVersion3(t *testing.T) {
	out, _, err := (Adapter{}).Export(&contactmodel.Record{})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropVersion, nil, "3.0")
}
