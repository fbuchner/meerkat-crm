package jscontact

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("gramgender", "pronouns")
}

func TestExport_GrammaticalGender(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "gramgender-example",
		SpeakToAs: &contactmodel.SpeakToAs{
			GrammaticalGenders: []contactmodel.GrammaticalGender{{Value: "feminine"}},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/speakToAs/grammaticalGender", "feminine")
}

func TestExport_GrammaticalGender_MultipleCollapseByLanguage(t *testing.T) {
	// JSContact's speakToAs.grammaticalGender is a scalar (RFC 9553 §2.2.4),
	// so multiple neutral GrammaticalGenders entries must collapse to one on
	// export. 20-correspondence.md's "gramgender" row: prefer the entry whose
	// Language matches Card.Language; otherwise use the first entry.
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:      "gramgender-multi-example",
		Language: "fr",
		SpeakToAs: &contactmodel.SpeakToAs{
			GrammaticalGenders: []contactmodel.GrammaticalGender{
				{Value: "masculine", Language: "en"},
				{Value: "feminine", Language: "fr"},
			},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/speakToAs/grammaticalGender", "feminine")
}

func TestExport_GrammaticalGender_MultipleNoLanguageMatchUsesFirst(t *testing.T) {
	// No Card.Language set (and/or no entry matches it): fall back to the
	// first stored entry, per the same "gramgender" row rule.
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "gramgender-multi-fallback-example",
		SpeakToAs: &contactmodel.SpeakToAs{
			GrammaticalGenders: []contactmodel.GrammaticalGender{
				{Value: "masculine", Language: "en"},
				{Value: "feminine", Language: "fr"},
			},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/speakToAs/grammaticalGender", "masculine")
}

func TestExport_Pronouns(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "pronouns-example",
		SpeakToAs: &contactmodel.SpeakToAs{
			Pronouns: []contactmodel.Pronouns{
				{ID: "P1", Pronouns: "xe/xir", Pref: intPtr(1)},
				{ID: "P2", Pronouns: "they/them", Pref: intPtr(2)},
			},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/speakToAs/pronouns/P1/pronouns", "xe/xir")
	rfctest.AssertJSONPointer(t, out, "/speakToAs/pronouns/P1/pref", 1)
	rfctest.AssertJSONPointer(t, out, "/speakToAs/pronouns/P2/pronouns", "they/them")
}
