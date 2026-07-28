package jscontact

import "testing"

// Concepts: expertise, hobby, interest.
// Row shape: Card.PersonalInfo[kind=X]  /personalInfo/{id}  personalinfo
// Values match the RFC 6715 worked examples also used by the golden
// expertise.v4.vcf/hobby.v4.vcf/interest.v4.vcf fixtures (EXPERTISE
// LEVEL=expert:chemistry; HOBBY LEVEL=high:reading; INTEREST
// LEVEL=medium:r&b music) — no dedicated .jscontact.json fixture exists for
// these concepts, so the same values are hand-built here as inline JSON.
func init() {
	registerImportCoverage("expertise", "hobby", "interest")
}

func TestImport_Expertise(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "expertise-example",
		"personalInfo": { "PI1": { "@type": "PersonalInfo", "kind": "expertise", "value": "chemistry", "level": "expert", "listAs": 1 } }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.PersonalInfo) != 1 {
		t.Fatalf("len(PersonalInfo) = %d, want 1", len(rec.Card.PersonalInfo))
	}
	pi := rec.Card.PersonalInfo[0]
	if pi.Kind != "expertise" || pi.Value != "chemistry" || pi.Level != "expert" {
		t.Errorf("PersonalInfo[0] = %+v", pi)
	}
}

func TestImport_Hobby(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "hobby-example",
		"personalInfo": { "PI1": { "@type": "PersonalInfo", "kind": "hobby", "value": "reading", "level": "high", "listAs": 1 } }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.PersonalInfo) != 1 || rec.Card.PersonalInfo[0].Kind != "hobby" || rec.Card.PersonalInfo[0].Value != "reading" {
		t.Errorf("PersonalInfo = %+v", rec.Card.PersonalInfo)
	}
}

func TestImport_Interest(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "interest-example",
		"personalInfo": { "PI1": { "@type": "PersonalInfo", "kind": "interest", "value": "r&b music", "level": "medium", "listAs": 1 } }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.PersonalInfo) != 1 || rec.Card.PersonalInfo[0].Kind != "interest" || rec.Card.PersonalInfo[0].Value != "r&b music" {
		t.Errorf("PersonalInfo = %+v", rec.Card.PersonalInfo)
	}
}
