package jscontact

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("lang", "related", "member")
}

func TestExport_PreferredLanguage(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:                "lang-example",
		PreferredLanguages: []contactmodel.LanguagePref{{ID: "L1", Language: "en", Pref: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/preferredLanguages/L1/language", "en")
}

func TestExport_Related(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "related-example",
		RelatedTo: []contactmodel.Relation{
			{Target: "https://example.com/directory/jdoe", Relations: []string{"friend"}},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/relatedTo/https:~1~1example.com~1directory~1jdoe/relation/friend", true)
}

func TestExport_Member(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:     "member-example",
		Kind:    "group",
		Members: []string{"urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af"},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/members/urn:uuid:03a0e51f-d1aa-4385-8a53-e29025acd8af", true)
}
