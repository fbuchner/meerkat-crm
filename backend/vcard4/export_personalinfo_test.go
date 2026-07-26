package vcard4

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("expertise", "hobby", "interest")
}

func TestExport_Expertise(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		PersonalInfo: []contactmodel.PersonalInfo{{Kind: "expertise", Value: "chemistry", Level: "high", ListAs: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "EXPERTISE", map[string]string{"LEVEL": "expert", "INDEX": "1"}, "chemistry")
}

func TestExport_Hobby(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		PersonalInfo: []contactmodel.PersonalInfo{{Kind: "hobby", Value: "reading", Level: "high", ListAs: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "HOBBY", map[string]string{"LEVEL": "high", "INDEX": "1"}, "reading")
}

func TestExport_Interest(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		PersonalInfo: []contactmodel.PersonalInfo{{Kind: "interest", Value: "r&b music", Level: "medium", ListAs: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "INTEREST", map[string]string{"LEVEL": "medium", "INDEX": "1"}, "r&b music")
}
