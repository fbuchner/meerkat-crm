package vcard4

import (
	"testing"

	"mycorrhizal/internal/rfctest"
)

// Concepts: lang, related, member.
func init() {
	registerImportCoverage("lang", "related", "member")
}

func TestImport_Lang(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("lang.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.PreferredLanguages) != 1 || rec.Card.PreferredLanguages[0].Language != "en" {
		t.Fatalf("PreferredLanguages = %+v", rec.Card.PreferredLanguages)
	}
	if rec.Card.PreferredLanguages[0].Pref == nil || *rec.Card.PreferredLanguages[0].Pref != 1 {
		t.Errorf("Pref = %v, want 1", rec.Card.PreferredLanguages[0].Pref)
	}
}

func TestImport_Related(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("related.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.RelatedTo) != 1 {
		t.Fatalf("RelatedTo = %+v", rec.Card.RelatedTo)
	}
	r := rec.Card.RelatedTo[0]
	if r.Target != "https://example.com/directory/jdoe" {
		t.Errorf("Target = %q", r.Target)
	}
	if len(r.Relations) != 1 || r.Relations[0] != "friend" {
		t.Errorf("Relations = %v, want [friend]", r.Relations)
	}
}

func TestImport_Member(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("member.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Members) != 1 || rec.Card.Members[0] != "urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af" {
		t.Fatalf("Members = %v", rec.Card.Members)
	}
}
