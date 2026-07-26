package vcard3

import "testing"

// Concept covered (coverage_test.go): keywords.

const keywordsImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"CATEGORIES:Family,Friends\n" +
	"END:VCARD\n"

func TestImport_Keywords(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(keywordsImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	want := []string{"Family", "Friends"}
	if len(rec.Card.Keywords) != len(want) {
		t.Fatalf("Keywords = %v, want %v", rec.Card.Keywords, want)
	}
	for i, w := range want {
		if rec.Card.Keywords[i] != w {
			t.Errorf("Keywords[%d] = %q, want %q", i, rec.Card.Keywords[i], w)
		}
	}
}
