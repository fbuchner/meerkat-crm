package jscontact

import "testing"

// Concepts: adr, adr.geo, adr.tz.
// Row "adr": Card.Addresses[]  /addresses/{id}  adr_components — anchors the
// *whole* Address element (see adapter.go's comment on addressToNeutral).
// adr.geo: Coordinates  /addresses/{id}/coordinates  geo_uri
// adr.tz:  TimeZone     /addresses/{id}/timeZone     identity
func init() {
	registerImportCoverage("adr", "adr.geo", "adr.tz")
}

func TestImport_Address(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "adr-example",
		"addresses": {
			"ADR-1": {
				"@type": "Address",
				"components": [
					{ "kind": "name", "value": "54321" },
					{ "kind": "locality", "value": "Hometown" },
					{ "kind": "region", "value": "PA" },
					{ "kind": "country", "value": "US" }
				],
				"contexts": { "work": true },
				"coordinates": "geo:46.772673,-71.282945",
				"timeZone": "America/New_York"
			}
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Addresses) != 1 {
		t.Fatalf("len(Addresses) = %d, want 1", len(rec.Card.Addresses))
	}
	a := rec.Card.Addresses[0]
	if a.ID != "ADR-1" {
		t.Errorf("Addresses[0].ID = %q, want ADR-1", a.ID)
	}
	if len(a.Components) != 4 || a.Components[1].Kind != "locality" || a.Components[1].Value != "Hometown" {
		t.Errorf("Addresses[0].Components = %+v", a.Components)
	}
	if len(a.Contexts) != 1 || a.Contexts[0] != "work" {
		t.Errorf("Addresses[0].Contexts = %v, want [work]", a.Contexts)
	}
}

func TestImport_AddressGeo(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "adr-geo-example",
		"addresses": { "ADR-1": { "@type": "Address", "coordinates": "geo:46.772673,-71.282945" } }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Addresses) != 1 || rec.Card.Addresses[0].Coordinates != "geo:46.772673,-71.282945" {
		t.Errorf("Addresses = %+v", rec.Card.Addresses)
	}
}

func TestImport_AddressTZ(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "adr-tz-example",
		"addresses": { "ADR-1": { "@type": "Address", "timeZone": "America/New_York" } }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Addresses) != 1 || rec.Card.Addresses[0].TimeZone != "America/New_York" {
		t.Errorf("Addresses = %+v", rec.Card.Addresses)
	}
}
