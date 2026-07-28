package vcard3

import "testing"

// Concept covered: note.
func init() {
	registerImportCoverage("note")
}

const noteImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"NOTE:Met at IETF.\n" +
	"END:VCARD\n"

func TestImport_Note(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(noteImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Notes) != 1 || rec.Card.Notes[0].Note != "Met at IETF." {
		t.Errorf("Notes = %+v, want [{Note: Met at IETF.}]", rec.Card.Notes)
	}
}
