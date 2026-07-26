package vcard3

import "testing"

// Concept covered (coverage_test.go): nickname.

const nicknameImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Johnny\n" +
	"NICKNAME;TYPE=WORK:Johnny\n" +
	"END:VCARD\n"

func TestImport_Nickname(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(nicknameImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Nicknames) != 1 {
		t.Fatalf("Nicknames = %+v, want 1 entry", rec.Card.Nicknames)
	}
	nn := rec.Card.Nicknames[0]
	if nn.Name != "Johnny" {
		t.Errorf("Name = %q, want %q", nn.Name, "Johnny")
	}
	if len(nn.Contexts) != 1 || nn.Contexts[0] != "work" {
		t.Errorf("Contexts = %v, want [work]", nn.Contexts)
	}
}
