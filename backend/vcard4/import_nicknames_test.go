package vcard4

import (
	"testing"

	"mycorrhizal/internal/rfctest"
)

// Concept: nickname. Row: Card.Nicknames[].Name / NICKNAME / PREF;TYPE / identity.
func init() {
	registerImportCoverage("nickname")
}

func TestImport_Nickname(t *testing.T) {
	raw := rfctest.LoadFixture("nickname.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Nicknames) != 1 || rec.Card.Nicknames[0].Name != "Johnny" {
		t.Fatalf("Nicknames = %+v, want [Johnny]", rec.Card.Nicknames)
	}
}

func TestImport_NicknamePrefType(t *testing.T) {
	raw := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\nUID:nick-pt\r\nFN:Test\r\nNICKNAME;PREF=1;TYPE=work:Johnny\r\nEND:VCARD\r\n")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Nicknames) != 1 {
		t.Fatalf("Nicknames = %+v", rec.Card.Nicknames)
	}
	n := rec.Card.Nicknames[0]
	if n.Pref == nil || *n.Pref != 1 {
		t.Errorf("Pref = %v, want 1", n.Pref)
	}
	if len(n.Contexts) != 1 || n.Contexts[0] != "work" {
		t.Errorf("Contexts = %v, want [work]", n.Contexts)
	}
}
