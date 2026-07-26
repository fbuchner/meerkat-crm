package vcard4

import (
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: uid, kind, prodid, updated, created, language.
// Rows: Card.UID/UID/identity; Card.Kind/KIND/identity; Card.ProdID/PRODID/identity;
// Card.Updated/REV/ts_rfc3339; Card.Created/CREATED/ts_rfc3339 (9554 CREATED,
// golden fixture created.v4.vcf); Card.Language/LANGUAGE/identity (RFC 9554
// §3.3 — corrected in 20-correspondence.md, was wrongly v4_prop "-").
func init() {
	registerImportCoverage("uid", "kind", "prodid", "updated", "created", "language")
}

func TestImport_UID(t *testing.T) {
	raw := rfctest.LoadFixture("rfc6350-baseline.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if want := "urn:uuid:4fbe8971-0bc3-424c-9c26-36c3e1eff6b1"; rec.Card.UID != want {
		t.Errorf("UID = %q, want %q", rec.Card.UID, want)
	}
}

func TestImport_Kind(t *testing.T) {
	raw := rfctest.LoadFixture("member.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Kind != "group" {
		t.Errorf("Kind = %q, want %q", rec.Card.Kind, "group")
	}
}

func TestImport_ProdID(t *testing.T) {
	raw := rfctest.LoadFixture("prodid.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if want := "-//Meerkat//RFCTest 1.0//EN"; rec.Card.ProdID != want {
		t.Errorf("ProdID = %q, want %q", rec.Card.ProdID, want)
	}
}

func TestImport_Updated(t *testing.T) {
	raw := rfctest.LoadFixture("updated.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Updated == nil {
		t.Fatalf("Updated = nil")
	}
	if want := "1996-10-22T14:00:00Z"; rec.Card.Updated.UTC != want {
		t.Errorf("Updated.UTC = %q, want %q", rec.Card.Updated.UTC, want)
	}
}

func TestImport_Created(t *testing.T) {
	raw := rfctest.LoadFixture("created.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.Created == nil {
		t.Fatalf("Created = nil")
	}
	if want := "2022-07-05T09:34:12Z"; rec.Card.Created.UTC != want {
		t.Errorf("Created.UTC = %q, want %q", rec.Card.Created.UTC, want)
	}
}

func TestImport_Language(t *testing.T) {
	// language.v4.vcf: verbatim RFC 9554 §3.3 example, LANGUAGE:de-AT.
	raw := rfctest.LoadFixture("language.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if want := "de-AT"; rec.Card.Language != want {
		t.Errorf("Language = %q, want %q", rec.Card.Language, want)
	}
}
