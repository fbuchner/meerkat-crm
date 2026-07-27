package jscontact

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("expertise", "hobby", "interest")
}

func TestExport_Expertise(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "expertise-example",
		PersonalInfo: []contactmodel.PersonalInfo{
			{ID: "PI1", Kind: "expertise", Value: "chemistry", Level: "expert", ListAs: intPtr(1)},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/personalInfo/PI1/kind", "expertise")
	rfctest.AssertJSONPointer(t, out, "/personalInfo/PI1/value", "chemistry")
	rfctest.AssertJSONPointer(t, out, "/personalInfo/PI1/level", "expert")
}

func TestExport_Hobby(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "hobby-example",
		PersonalInfo: []contactmodel.PersonalInfo{
			{ID: "PI1", Kind: "hobby", Value: "reading", Level: "high", ListAs: intPtr(1)},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/personalInfo/PI1/kind", "hobby")
	rfctest.AssertJSONPointer(t, out, "/personalInfo/PI1/value", "reading")
}

func TestExport_Interest(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "interest-example",
		PersonalInfo: []contactmodel.PersonalInfo{
			{ID: "PI1", Kind: "interest", Value: "r&b music", Level: "medium", ListAs: intPtr(1)},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/personalInfo/PI1/kind", "interest")
	rfctest.AssertJSONPointer(t, out, "/personalInfo/PI1/value", "r&b music")
}
