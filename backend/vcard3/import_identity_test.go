package vcard3

import "testing"

// Concepts covered (coverage_test.go): uid, prodid, updated.

const identityImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"UID:frank-dawson-uid-1\n" +
	"PRODID:-//Meerkat//WP-50 Test//EN\n" +
	"REV:2023-01-02T03:04:05Z\n" +
	"END:VCARD\n"

func TestImport_UID(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(identityImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.UID != "frank-dawson-uid-1" {
		t.Errorf("UID = %q, want %q", rec.Card.UID, "frank-dawson-uid-1")
	}
}

func TestImport_ProdID(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(identityImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.ProdID != "-//Meerkat//WP-50 Test//EN" {
		t.Errorf("ProdID = %q, want %q", rec.Card.ProdID, "-//Meerkat//WP-50 Test//EN")
	}
}

func TestImport_Updated(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(identityImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Updated == nil {
		t.Fatalf("Updated = nil, want non-nil")
	}
	want := "2023-01-02T03:04:05Z"
	if rec.Card.Updated.UTC != want {
		t.Errorf("Updated.UTC = %q, want %q", rec.Card.Updated.UTC, want)
	}
}

// Sanity: no diagnostics fire for plain, fully-mapped identity fields.
func TestImport_Identity_NoDiagnostics(t *testing.T) {
	_, diags, err := (Adapter{}).Import([]byte(identityImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(diags) != 0 {
		t.Errorf("diags = %+v, want none", diags)
	}
}
