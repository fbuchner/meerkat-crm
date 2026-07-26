package vcard4

import (
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: gramgender, pronouns.
// Golden fixtures: gramgender.v4.vcf (RFC 9554 §3.2), pronouns.v4.vcf
// (RFC 9554 §3.4).
func init() {
	registerImportCoverage("gramgender", "pronouns")
}

func TestImport_GrammaticalGender(t *testing.T) {
	raw := rfctest.LoadFixture("gramgender.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.SpeakToAs == nil || len(rec.Card.SpeakToAs.GrammaticalGenders) != 1 || rec.Card.SpeakToAs.GrammaticalGenders[0].Value != "feminine" {
		t.Fatalf("SpeakToAs = %+v", rec.Card.SpeakToAs)
	}
}

func TestImport_Pronouns(t *testing.T) {
	raw := rfctest.LoadFixture("pronouns.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.SpeakToAs == nil || len(rec.Card.SpeakToAs.Pronouns) != 2 {
		t.Fatalf("SpeakToAs = %+v", rec.Card.SpeakToAs)
	}
	p0, p1 := rec.Card.SpeakToAs.Pronouns[0], rec.Card.SpeakToAs.Pronouns[1]
	if p0.Pronouns != "xe/xir" || p0.Pref == nil || *p0.Pref != 1 {
		t.Errorf("Pronouns[0] = %+v, want xe/xir pref=1", p0)
	}
	if p1.Pronouns != "they/them" || p1.Pref == nil || *p1.Pref != 2 {
		t.Errorf("Pronouns[1] = %+v, want they/them pref=2", p1)
	}
}

// TestImport_PronounsType covers RFC 9554 §3.4's TYPE parameter (bug fix:
// PRONOUNS TYPE was listed as a valid parameter alongside LANGUAGE/PREF/ALTID
// but was never read into Pronouns.Contexts).
func TestImport_PronounsType(t *testing.T) {
	raw := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\nUID:pronouns-type\r\nFN:Test\r\n" +
		"PRONOUNS;TYPE=home:xe/xir\r\nEND:VCARD\r\n")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.SpeakToAs == nil || len(rec.Card.SpeakToAs.Pronouns) != 1 {
		t.Fatalf("SpeakToAs = %+v", rec.Card.SpeakToAs)
	}
	p := rec.Card.SpeakToAs.Pronouns[0]
	if p.Pronouns != "xe/xir" {
		t.Errorf("Pronouns = %q, want xe/xir", p.Pronouns)
	}
	if len(p.Contexts) != 1 || p.Contexts[0] != "private" {
		t.Errorf("Contexts = %v, want [private]", p.Contexts)
	}
}

// TestImport_GrammaticalGenderMultiple covers RFC 9554 §3.2's actual
// cardinality ("*", multiple occurrences distinguished by LANGUAGE):
// SpeakToAs.GrammaticalGenders is a slice, so every occurrence is stored
// losslessly on import — no diagnostic, no dropped data.
func TestImport_GrammaticalGenderMultiple(t *testing.T) {
	raw := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\nUID:gramgender-multi\r\nFN:Test\r\n" +
		"GRAMGENDER;LANGUAGE=de:feminine\r\nGRAMGENDER;LANGUAGE=fr:masculine\r\nEND:VCARD\r\n")
	rec, diags, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.SpeakToAs == nil || len(rec.Card.SpeakToAs.GrammaticalGenders) != 2 {
		t.Fatalf("SpeakToAs = %+v, want 2 GrammaticalGenders entries", rec.Card.SpeakToAs)
	}
	g0, g1 := rec.Card.SpeakToAs.GrammaticalGenders[0], rec.Card.SpeakToAs.GrammaticalGenders[1]
	if g0.Value != "feminine" || g0.Language != "de" {
		t.Errorf("GrammaticalGenders[0] = %+v, want feminine/de", g0)
	}
	if g1.Value != "masculine" || g1.Language != "fr" {
		t.Errorf("GrammaticalGenders[1] = %+v, want masculine/fr", g1)
	}
	for _, d := range diags {
		if d.Concept == "gramgender" {
			t.Errorf("unexpected gramgender Diagnostic (no data loss expected): %+v", d)
		}
	}
}
