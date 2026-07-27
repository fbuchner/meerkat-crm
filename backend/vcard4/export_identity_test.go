package vcard4

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("uid", "kind", "prodid", "updated", "created", "language")
}

func TestExport_UID(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{UID: "urn:uuid:test-uid", Name: &contactmodel.Name{Full: "Test"}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "UID", nil, "urn:uuid:test-uid")
}

func TestExport_Kind(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Kind: "group", Name: &contactmodel.Name{Full: "Group"}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "KIND", nil, "group")
}

func TestExport_ProdID(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{ProdID: "-//Mycorrhizal//Test 1.0//EN", Name: &contactmodel.Name{Full: "Test"}}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "PRODID", nil, "-//Mycorrhizal//Test 1.0//EN")
}

func TestExport_Updated(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Updated: &contactmodel.Timestamp{UTC: "1996-10-22T14:00:00Z"},
		Name:    &contactmodel.Name{Full: "Test"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "REV", nil, "19961022T140000Z")
}

func TestExport_Created(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Created: &contactmodel.Timestamp{UTC: "2022-07-05T09:34:12Z"},
		Name:    &contactmodel.Name{Full: "Test"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "CREATED", nil, "20220705T093412Z")
}

func TestExport_Language(t *testing.T) {
	// Matches the verbatim RFC 9554 §3.3 example: LANGUAGE:de-AT.
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Language: "de-AT",
		Name:     &contactmodel.Name{Full: "Test"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "LANGUAGE", nil, "de-AT")
}
