package jscontact

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func init() {
	registerExportCoverage("nickname")
}

func TestExport_Nickname(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "nickname-example",
		Nicknames: []contactmodel.Nickname{
			{ID: "N1", Name: "Johnny", Contexts: []string{"private"}, Pref: intPtr(1)},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/nicknames/N1/name", "Johnny")
	rfctest.AssertJSONPointer(t, out, "/nicknames/N1/contexts/private", true)
	rfctest.AssertJSONPointer(t, out, "/nicknames/N1/pref", 1)
}
