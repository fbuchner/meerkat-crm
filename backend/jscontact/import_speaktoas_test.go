package jscontact

import "testing"

// Concepts: gramgender, pronouns.
// Row gramgender: Card.SpeakToAs.GrammaticalGenders[].Value /speakToAs/grammaticalGender  enum_lower
// Row pronouns:   Card.SpeakToAs.Pronouns[].Pronouns /speakToAs/pronouns/{id}/pronouns  identity
// Values below match the RFC 9554 §3.2/§3.4 worked examples also used by the
// golden gramgender.v4.vcf / pronouns.v4.vcf fixtures (GRAMGENDER:feminine;
// PRONOUNS PREF=1:xe/xir, PREF=2:they/them) — no dedicated .jscontact.json
// fixture exists for these concepts (per 40-testing.md §40.4's scope note),
// so the same values are hand-built here as inline JSON.
//
// JSContact's own speakToAs.grammaticalGender is a scalar (RFC 9553 §2.2.4)
// with no language tag, so import always produces a single-element (or nil)
// GrammaticalGenders slice with Language left empty — see
// 20-correspondence.md's "gramgender" row.
func init() {
	registerImportCoverage("gramgender", "pronouns")
}

func TestImport_GrammaticalGender(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "gramgender-example",
		"speakToAs": { "@type": "SpeakToAs", "grammaticalGender": "feminine" }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.SpeakToAs == nil || len(rec.Card.SpeakToAs.GrammaticalGenders) != 1 {
		t.Fatalf("SpeakToAs = %+v", rec.Card.SpeakToAs)
	}
	g := rec.Card.SpeakToAs.GrammaticalGenders[0]
	if g.Value != "feminine" || g.Language != "" {
		t.Errorf("GrammaticalGenders[0] = %+v, want {Value: feminine, Language: \"\"}", g)
	}
}

func TestImport_Pronouns(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "pronouns-example",
		"speakToAs": {
			"@type": "SpeakToAs",
			"pronouns": {
				"P1": { "@type": "Pronouns", "pronouns": "xe/xir", "pref": 1 },
				"P2": { "@type": "Pronouns", "pronouns": "they/them", "pref": 2 }
			}
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if rec.Card.SpeakToAs == nil || len(rec.Card.SpeakToAs.Pronouns) != 2 {
		t.Fatalf("SpeakToAs = %+v", rec.Card.SpeakToAs)
	}
	p := rec.Card.SpeakToAs.Pronouns
	if p[0].ID != "P1" || p[0].Pronouns != "xe/xir" || p[0].Pref == nil || *p[0].Pref != 1 {
		t.Errorf("Pronouns[0] = %+v", p[0])
	}
	if p[1].ID != "P2" || p[1].Pronouns != "they/them" || p[1].Pref == nil || *p[1].Pref != 2 {
		t.Errorf("Pronouns[1] = %+v", p[1])
	}
}
