package vcard4

import (
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: note, keywords.
// Golden fixture: note-author.v4.vcf (RFC 9554 §4.1-4.3, AUTHOR/AUTHOR-NAME/CREATED).
func init() {
	registerImportCoverage("note", "keywords")
}

func TestImport_NoteAuthor(t *testing.T) {
	raw := rfctest.LoadFixture("note-author.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Notes) != 1 {
		t.Fatalf("Notes = %+v", rec.Card.Notes)
	}
	n := rec.Card.Notes[0]
	if n.Note != "This is some note." {
		t.Errorf("Note = %q", n.Note)
	}
	if n.Author == nil || n.Author.URI != "mailto:john@example.com" || n.Author.Name != "John Doe" {
		t.Errorf("Author = %+v", n.Author)
	}
	if n.Created == nil || n.Created.UTC != "2022-11-22T15:18:23Z" {
		t.Errorf("Created = %+v", n.Created)
	}
}

func TestImport_Keywords(t *testing.T) {
	raw := rfctest.LoadFixture("keywords.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Keywords) != 2 || rec.Card.Keywords[0] != "family" || rec.Card.Keywords[1] != "work" {
		t.Fatalf("Keywords = %v, want [family work]", rec.Card.Keywords)
	}
}
