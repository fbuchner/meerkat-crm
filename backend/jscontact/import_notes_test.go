package jscontact

import "testing"

// Concepts: note, keywords.
// Row note:     Card.Notes[].Note  /notes/{id}/note  identity (author/created
//
//	params 9554; no dedicated .jscontact.json fixture, values
//	below match the golden note-author.v4.vcf worked example).
//
// Row keywords:  Card.Keywords     /keywords         csv_join (vCard-side
//
//	comma-join only; JSContact already carries []string).
func init() {
	registerImportCoverage("note", "keywords")
}

func TestImport_NoteWithAuthor(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "note-author-example",
		"notes": {
			"N1": {
				"@type": "Note", "note": "This is some note.",
				"author": { "@type": "Author", "name": "John Doe", "uri": "mailto:john@example.com" },
				"created": { "@type": "Timestamp", "utc": "2022-11-22T15:18:23Z" }
			}
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Notes) != 1 {
		t.Fatalf("len(Notes) = %d, want 1", len(rec.Card.Notes))
	}
	n := rec.Card.Notes[0]
	if n.Note != "This is some note." {
		t.Errorf("Note = %q", n.Note)
	}
	if n.Author == nil || n.Author.Name != "John Doe" || n.Author.URI != "mailto:john@example.com" {
		t.Errorf("Author = %+v", n.Author)
	}
	if n.Created == nil || n.Created.UTC != "2022-11-22T15:18:23Z" {
		t.Errorf("Created = %+v", n.Created)
	}
}

func TestImport_Keywords(t *testing.T) {
	raw := []byte(`{"@type":"Card","version":"1.0","uid":"keywords-example","keywords":{"family":true,"work":true}}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	want := map[string]bool{"family": true, "work": true}
	if len(rec.Card.Keywords) != 2 {
		t.Fatalf("Keywords = %v, want 2 entries", rec.Card.Keywords)
	}
	for _, k := range rec.Card.Keywords {
		if !want[k] {
			t.Errorf("unexpected keyword %q", k)
		}
	}
}
