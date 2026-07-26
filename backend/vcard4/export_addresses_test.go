package vcard4

import (
	"strings"
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("adr", "adr.geo", "adr.tz")
}

func TestExport_Address(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Addresses: []contactmodel.Address{{
			Components: []contactmodel.AddressComponent{
				{Kind: "locality", Value: "Any Town"},
				{Kind: "region", Value: "CA"},
				{Kind: "postcode", Value: "91921-1234"},
				{Kind: "country", Value: "U.S.A"},
				{Kind: "number", Value: "123"},
				{Kind: "name", Value: "Main Street"},
			},
			Contexts: []string{"work"},
		}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "ADR", map[string]string{"TYPE": "work"},
		";;123 Main Street;Any Town;CA;91921-1234;U.S.A;;;;123;Main Street;;;;;;")
}

func TestExport_AddressGeo(t *testing.T) {
	// GEO carries a mandatory literal comma (RFC 5870 geo: URI syntax); see
	// exportAddresses's comment on the go-vcard quoting limitation this
	// works around via a post-encode splice rather than the plain Params
	// path. Assert on the raw bytes (not AssertVCardLine's re-decode path)
	// since that's exactly the case this workaround targets.
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Addresses: []contactmodel.Address{{Coordinates: "geo:12.3457,78.910"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !strings.Contains(string(out), `GEO="geo:12.3457,78.910"`) {
		t.Errorf("output does not contain quoted GEO param:\n%s", out)
	}
}

func TestExport_AddressTZ(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Addresses: []contactmodel.Address{{TimeZone: "America/New_York"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "ADR", map[string]string{"TZ": "America/New_York"}, ";;;;;;;;;;;;;;;;;")
}
