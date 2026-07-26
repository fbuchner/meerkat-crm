package jscontact

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("adr", "adr.geo", "adr.tz")
}

func TestExport_Address(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "adr-example",
		Addresses: []contactmodel.Address{
			{
				ID: "ADR-1",
				Components: []contactmodel.AddressComponent{
					{Kind: "locality", Value: "Hometown"},
					{Kind: "country", Value: "US"},
				},
				Contexts: []string{"work"},
			},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/addresses/ADR-1/components/0/kind", "locality")
	rfctest.AssertJSONPointer(t, out, "/addresses/ADR-1/components/0/value", "Hometown")
	rfctest.AssertJSONPointer(t, out, "/addresses/ADR-1/contexts/work", true)
}

func TestExport_AddressGeo(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:       "adr-geo-example",
		Addresses: []contactmodel.Address{{ID: "ADR-1", Coordinates: "geo:46.772673,-71.282945"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/addresses/ADR-1/coordinates", "geo:46.772673,-71.282945")
}

func TestExport_AddressTZ(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:       "adr-tz-example",
		Addresses: []contactmodel.Address{{ID: "ADR-1", TimeZone: "America/New_York"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/addresses/ADR-1/timeZone", "America/New_York")
}
