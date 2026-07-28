package vcard4

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("gramgender", "pronouns")
}

func TestExport_GrammaticalGender(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SpeakToAs: &contactmodel.SpeakToAs{GrammaticalGenders: []contactmodel.GrammaticalGender{{Value: "feminine"}}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "GRAMGENDER", nil, "feminine")
}

// TestExport_GrammaticalGenderMultiple covers full-fidelity vCard4-to-vCard4
// round trip: every stored GrammaticalGenders[] entry is re-emitted as its
// own GRAMGENDER field with its own LANGUAGE param (RFC 9554 §3.2).
func TestExport_GrammaticalGenderMultiple(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SpeakToAs: &contactmodel.SpeakToAs{GrammaticalGenders: []contactmodel.GrammaticalGender{
			{Value: "feminine", Language: "de"},
			{Value: "masculine", Language: "fr"},
		}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "GRAMGENDER", map[string]string{"LANGUAGE": "de"}, "feminine")
	rfctest.AssertVCardLine(t, out, "GRAMGENDER", map[string]string{"LANGUAGE": "fr"}, "masculine")
}

func TestExport_Pronouns(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SpeakToAs: &contactmodel.SpeakToAs{Pronouns: []contactmodel.Pronouns{{Pronouns: "xe/xir", Pref: intPtr(1)}}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "PRONOUNS", map[string]string{"PREF": "1"}, "xe/xir")
}

// TestExport_PronounsContexts covers the bug fix: Pronouns.Contexts (mirrors
// JSContact Pronouns.contexts) must round-trip out as vCard4's PRONOUNS TYPE
// parameter, not be silently dropped.
func TestExport_PronounsContexts(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SpeakToAs: &contactmodel.SpeakToAs{Pronouns: []contactmodel.Pronouns{{Pronouns: "xe/xir", Contexts: []string{"private"}}}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "PRONOUNS", map[string]string{"TYPE": "home"}, "xe/xir")
}
