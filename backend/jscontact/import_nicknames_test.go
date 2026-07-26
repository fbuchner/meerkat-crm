package jscontact

import "testing"

// Concept: nickname. Row: Card.Nicknames[].Name  /nicknames/{id}/name  identity
// (ctx→TYPE via ctx2type / pref→PREF are vCard-side concerns of the same row;
// JSContact carries Contexts/Pref directly, no transform needed there).
func init() {
	registerImportCoverage("nickname")
}

func TestImport_Nickname(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "nickname-example",
		"nicknames": {
			"N1": { "@type": "Nickname", "name": "Johnny", "contexts": { "private": true }, "pref": 1 }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Nicknames) != 1 {
		t.Fatalf("len(Nicknames) = %d, want 1", len(rec.Card.Nicknames))
	}
	n := rec.Card.Nicknames[0]
	if n.ID != "N1" || n.Name != "Johnny" {
		t.Errorf("Nicknames[0] = %+v", n)
	}
	if len(n.Contexts) != 1 || n.Contexts[0] != "private" {
		t.Errorf("Nicknames[0].Contexts = %v, want [private]", n.Contexts)
	}
	if n.Pref == nil || *n.Pref != 1 {
		t.Errorf("Nicknames[0].Pref = %v, want 1", n.Pref)
	}
}
