package jscontact

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("uid", "kind", "prodid", "created", "updated", "language")
}

func TestExport_UIDAndKind(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:  "22B2C7DF-9120-4969-8460-05956FE6B065",
		Kind: "individual",
	}}
	out, diags, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(diags) != 0 {
		t.Errorf("diags = %v, want none", diags)
	}
	rfctest.AssertJSONPointer(t, out, "/uid", "22B2C7DF-9120-4969-8460-05956FE6B065")
	rfctest.AssertJSONPointer(t, out, "/kind", "individual")
}

func TestExport_ProdID(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:    "prodid-example",
		ProdID: "-//ACME//AddressBook//EN",
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/prodId", "-//ACME//AddressBook//EN")
}

// TestExport_Created is the export-direction counterpart of
// TestImport_Created (import_identity_test.go): Record.Card.Created maps to
// the wire Card's top-level "created" property, exactly like Updated.
func TestExport_Created(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:     "created-example",
		Created: &contactmodel.Timestamp{UTC: "2021-08-24T18:30:00Z"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/created/utc", "2021-08-24T18:30:00Z")
}

func TestExport_Updated(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:     "updated-example",
		Updated: &contactmodel.Timestamp{UTC: "2021-08-24T18:30:00Z"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/updated/utc", "2021-08-24T18:30:00Z")
}

func TestExport_Language(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:      "language-example",
		Language: "de-AT",
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/language", "de-AT")
}
