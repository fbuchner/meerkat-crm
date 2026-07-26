package vcard3

import "testing"

// Concept covered: phone.
// Fixture value from docs/specs/rfc2426-v3-baseline.md §1 (RFC 2426 §7 example).
func init() {
	registerImportCoverage("phone")
}

const phoneImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"TEL;TYPE=VOICE,MSG,WORK:+1-919-676-9515\n" +
	"TEL;TYPE=FAX,WORK:+1-919-676-9564\n" +
	"END:VCARD\n"

func TestImport_Phone(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(phoneImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Phones) != 2 {
		t.Fatalf("Phones = %+v, want 2 entries", rec.Card.Phones)
	}
	first := rec.Card.Phones[0]
	if first.Number != "+1-919-676-9515" {
		t.Errorf("Phones[0].Number = %q, want %q", first.Number, "+1-919-676-9515")
	}
	if len(first.Features) != 1 || first.Features[0] != "voice" {
		t.Errorf("Phones[0].Features = %v, want [voice]", first.Features)
	}
	if len(first.Contexts) != 1 || first.Contexts[0] != "work" {
		t.Errorf("Phones[0].Contexts = %v, want [work]", first.Contexts)
	}
	second := rec.Card.Phones[1]
	if len(second.Features) != 1 || second.Features[0] != "fax" {
		t.Errorf("Phones[1].Features = %v, want [fax]", second.Features)
	}
}
