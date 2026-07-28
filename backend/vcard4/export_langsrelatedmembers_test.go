package vcard4

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("lang", "related", "member")
}

func TestExport_Lang(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		PreferredLanguages: []contactmodel.LanguagePref{{Language: "en", Pref: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "LANG", map[string]string{"PREF": "1"}, "en")
}

func TestExport_Related(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		RelatedTo: []contactmodel.Relation{{Target: "https://example.com/directory/jdoe", Relations: []string{"friend"}}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "RELATED", map[string]string{"TYPE": "friend"}, "https://example.com/directory/jdoe")
}

func TestExport_Member(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Kind:    "group",
		Members: []string{"urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "MEMBER", nil, "urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af")
}
