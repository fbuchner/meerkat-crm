package vcard3

import "testing"

// Concept covered: adr.
// Fixture value from docs/specs/rfc2426-v3-baseline.md §1 (RFC 2426 §7 example),
// folded to a single unfolded line (folding itself is go-vcard's concern, not ours).
func init() {
	registerImportCoverage("adr")
}

const adrImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"ADR;TYPE=WORK,POSTAL,PARCEL:;;6544 Battleford Drive;Raleigh;NC;27613-3502;U.S.A.\n" +
	"END:VCARD\n"

func TestImport_Adr(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(adrImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Addresses) != 1 {
		t.Fatalf("Addresses = %+v, want 1 entry", rec.Card.Addresses)
	}
	a := rec.Card.Addresses[0]
	want := map[string]string{
		"name":     "6544 Battleford Drive",
		"locality": "Raleigh",
		"region":   "NC",
		"postcode": "27613-3502",
		"country":  "U.S.A.",
	}
	got := map[string]string{}
	for _, c := range a.Components {
		got[c.Kind] = c.Value
	}
	for kind, wantVal := range want {
		if got[kind] != wantVal {
			t.Errorf("component %q = %q, want %q", kind, got[kind], wantVal)
		}
	}
	if len(a.Contexts) != 1 || a.Contexts[0] != "work" {
		t.Errorf("Contexts = %v, want [work]", a.Contexts)
	}
}

const adrLabelImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:John Q Public\n" +
	"ADR;TYPE=HOME:;;123 Main Street;Any Town;CA;91921-1234;U.S.A.\n" +
	"LABEL;TYPE=HOME:Mr. John Q. Public\\, Esq.\\n123 Main Street\\nAny Town\\, CA  91921-1234\\nU.S.A.\n" +
	"END:VCARD\n"

func TestImport_AdrLabel(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(adrLabelImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Addresses) != 1 {
		t.Fatalf("Addresses = %+v, want 1 entry", rec.Card.Addresses)
	}
	if rec.Card.Addresses[0].Full == "" {
		t.Errorf("Full = empty, want the LABEL text paired by TYPE")
	}
}
