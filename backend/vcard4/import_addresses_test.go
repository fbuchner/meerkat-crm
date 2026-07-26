package vcard4

import (
	"testing"

	"meerkat/internal/rfctest"
)

// Concepts: adr, adr.geo, adr.tz.
// Golden fixture: adr-expanded.v4.vcf (RFC 9554 §2.1, 18-component ADR + GEO
// param). Per-concept fixture: adr-tz.v4.vcf (7-legacy-field ADR + TZ).
func init() {
	registerImportCoverage("adr", "adr.geo", "adr.tz")
}

func TestImport_AddressExpanded(t *testing.T) {
	raw := rfctest.LoadFixture("adr-expanded.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Addresses) != 1 {
		t.Fatalf("Addresses = %+v", rec.Card.Addresses)
	}
	a := rec.Card.Addresses[0]
	get := func(kind string) string {
		for _, c := range a.Components {
			if c.Kind == kind {
				return c.Value
			}
		}
		return ""
	}
	if get("locality") != "Any Town" {
		t.Errorf("locality = %q", get("locality"))
	}
	if get("region") != "CA" {
		t.Errorf("region = %q", get("region"))
	}
	if get("postcode") != "91921-1234" {
		t.Errorf("postcode = %q", get("postcode"))
	}
	if get("country") != "U.S.A" {
		t.Errorf("country = %q", get("country"))
	}
	if get("number") != "123" {
		t.Errorf("number = %q", get("number"))
	}
	if get("name") != "Main Street" {
		t.Errorf("name (street) = %q, want Main Street (prefers the 9554 StreetName position over the legacy combined Street field)", get("name"))
	}
}

func TestImport_AddressGeo(t *testing.T) {
	raw := rfctest.LoadFixture("adr-expanded.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Addresses) != 1 || rec.Card.Addresses[0].Coordinates != "geo:12.3457,78.910" {
		t.Fatalf("Coordinates = %+v", rec.Card.Addresses)
	}
}

func TestImport_AddressTZ(t *testing.T) {
	raw := rfctest.LoadFixture("adr-tz.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Addresses) != 1 {
		t.Fatalf("Addresses = %+v", rec.Card.Addresses)
	}
	a := rec.Card.Addresses[0]
	if a.TimeZone != "America/New_York" {
		t.Errorf("TimeZone = %q", a.TimeZone)
	}
	var street string
	for _, c := range a.Components {
		if c.Kind == "name" {
			street = c.Value
		}
	}
	if street != "123 Main Street" {
		t.Errorf("street (name) = %q, want the legacy Street fallback since no StreetName/Number are present", street)
	}
}
