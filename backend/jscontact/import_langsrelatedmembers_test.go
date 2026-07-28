package jscontact

import "testing"

// Concepts: lang, related, member.
// Row lang:    Card.PreferredLanguages[].Language  /preferredLanguages/{id}/language  identity
// Row related: Card.RelatedTo[]                    /relatedTo/{target}                related
// Row member:  Card.Members                        /members                           identity
// Values below match the golden lang.v4.vcf/related.v4.vcf/member.v4.vcf fixtures.
func init() {
	registerImportCoverage("lang", "related", "member")
}

func TestImport_PreferredLanguage(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "lang-example",
		"preferredLanguages": { "L1": { "@type": "LanguagePref", "language": "en", "pref": 1 } }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.PreferredLanguages) != 1 || rec.Card.PreferredLanguages[0].Language != "en" {
		t.Errorf("PreferredLanguages = %+v", rec.Card.PreferredLanguages)
	}
}

func TestImport_Related(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "related-example",
		"relatedTo": { "https://example.com/directory/jdoe": { "@type": "Relation", "relation": { "friend": true } } }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.RelatedTo) != 1 {
		t.Fatalf("len(RelatedTo) = %d, want 1", len(rec.Card.RelatedTo))
	}
	rel := rec.Card.RelatedTo[0]
	if rel.Target != "https://example.com/directory/jdoe" {
		t.Errorf("Target = %q", rel.Target)
	}
	if len(rel.Relations) != 1 || rel.Relations[0] != "friend" {
		t.Errorf("Relations = %v, want [friend]", rel.Relations)
	}
}

func TestImport_Member(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "member-example", "kind": "group",
		"members": { "urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af": true }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Members) != 1 || rec.Card.Members[0] != "urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af" {
		t.Errorf("Members = %v", rec.Card.Members)
	}
}
