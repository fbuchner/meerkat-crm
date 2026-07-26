package vcard3

import "testing"

// Concept covered (coverage_test.go): email.
// Fixture value from docs/specs/rfc2426-v3-baseline.md §1 (RFC 2426 §7 example).

const emailImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"EMAIL;TYPE=INTERNET,PREF:Frank_Dawson@Lotus.com\n" +
	"EMAIL;TYPE=INTERNET:fdawson@earthlink.net\n" +
	"END:VCARD\n"

func TestImport_Email(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(emailImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Emails) != 2 {
		t.Fatalf("Emails = %+v, want 2 entries", rec.Card.Emails)
	}
	first := rec.Card.Emails[0]
	if first.Address != "Frank_Dawson@Lotus.com" {
		t.Errorf("Emails[0].Address = %q, want %q", first.Address, "Frank_Dawson@Lotus.com")
	}
	if first.Pref == nil || *first.Pref != 1 {
		t.Errorf("Emails[0].Pref = %v, want ptr(1)", first.Pref)
	}
	second := rec.Card.Emails[1]
	if second.Address != "fdawson@earthlink.net" {
		t.Errorf("Emails[1].Address = %q, want %q", second.Address, "fdawson@earthlink.net")
	}
}
