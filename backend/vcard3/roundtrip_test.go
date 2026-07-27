package vcard3

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

func neutralRoundtripFixture(pref int) *contactmodel.Record {
	return &contactmodel.Record{Card: contactmodel.Card{
		UID:  "roundtrip-uid-1",
		Name: &contactmodel.Name{Full: "Jane Roundtrip"},
		Emails: []contactmodel.Email{
			{Address: "jane@example.com", Pref: &pref},
		},
		Phones: []contactmodel.Phone{
			{Number: "+1-555-0100", Features: []string{"voice"}},
		},
		Addresses: []contactmodel.Address{{
			Components: []contactmodel.AddressComponent{
				{Kind: "name", Value: "1 Test Way"},
				{Kind: "locality", Value: "Testville"},
			},
		}},
	}}
}

// roundtrip_test.go: a handful of P -> neutral -> P and neutral -> P ->
// neutral spot checks (40-testing.md §40.1 point 4). Confidence only, not
// exhaustive — the per-concept import_*/export_*_test.go files own field
// correctness.

// TestRoundtrip_RFC2426Baseline is the authoritative golden-fixture check
// (60-review-gates.md §60.1): Frank Dawson's example vCard from RFC 2426 §7,
// copied verbatim into backend/internal/rfctest/fixtures/rfc2426-baseline.v3.vcf.
// It is imported and checked against the RFC-documented values directly,
// independent of any correspondence-row bookkeeping, so a shared
// misconception in this package's own tests cannot hide behind it.
func TestRoundtrip_RFC2426Baseline(t *testing.T) {
	raw := rfctest.LoadFixture("rfc2426-baseline.v3.vcf")
	rec, diags, err := (Adapter{}).Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(diags) != 0 {
		t.Errorf("diags = %+v, want none for this fully-mappable fixture", diags)
	}

	if rec.Card.Name == nil || rec.Card.Name.Full != "Frank Dawson" {
		t.Errorf("Name.Full = %+v, want %q", rec.Card.Name, "Frank Dawson")
	}
	if len(rec.Card.Organizations) != 1 || rec.Card.Organizations[0].Name != "Lotus Development Corporation" {
		t.Errorf("Organizations = %+v, want [{Name: Lotus Development Corporation}]", rec.Card.Organizations)
	}
	if len(rec.Card.Addresses) != 1 {
		t.Fatalf("Addresses = %+v, want 1 entry", rec.Card.Addresses)
	}
	addr := rec.Card.Addresses[0]
	wantComps := map[string]string{"name": "6544 Battleford Drive", "locality": "Raleigh", "region": "NC", "postcode": "27613-3502", "country": "U.S.A."}
	gotComps := map[string]string{}
	for _, c := range addr.Components {
		gotComps[c.Kind] = c.Value
	}
	for k, want := range wantComps {
		if gotComps[k] != want {
			t.Errorf("address component %q = %q, want %q", k, gotComps[k], want)
		}
	}
	if len(rec.Card.Phones) != 2 {
		t.Fatalf("Phones = %+v, want 2 entries", rec.Card.Phones)
	}
	if rec.Card.Phones[0].Number != "+1-919-676-9515" {
		t.Errorf("Phones[0].Number = %q, want %q", rec.Card.Phones[0].Number, "+1-919-676-9515")
	}
	if rec.Card.Phones[1].Number != "+1-919-676-9564" {
		t.Errorf("Phones[1].Number = %q, want %q", rec.Card.Phones[1].Number, "+1-919-676-9564")
	}
	if len(rec.Card.Emails) != 2 {
		t.Fatalf("Emails = %+v, want 2 entries", rec.Card.Emails)
	}
	if rec.Card.Emails[0].Address != "Frank_Dawson@Lotus.com" || rec.Card.Emails[0].Pref == nil {
		t.Errorf("Emails[0] = %+v, want Frank_Dawson@Lotus.com with Pref set", rec.Card.Emails[0])
	}
	if rec.Card.Emails[1].Address != "fdawson@earthlink.net" {
		t.Errorf("Emails[1].Address = %q, want %q", rec.Card.Emails[1].Address, "fdawson@earthlink.net")
	}
	if len(rec.Card.Links) != 1 || rec.Card.Links[0].URI != "http://home.earthlink.net/~fdawson" {
		t.Errorf("Links = %+v, want [{URI: http://home.earthlink.net/~fdawson}]", rec.Card.Links)
	}
}

// TestRoundtrip_RFC2426Baseline_ReExport re-exports the neutral Record
// produced above and spot-checks a few properties survive the
// import -> neutral -> export leg (P -> neutral -> P).
func TestRoundtrip_RFC2426Baseline_ReExport(t *testing.T) {
	raw := rfctest.LoadFixture("rfc2426-baseline.v3.vcf")
	rec, _, err := (Adapter{}).Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropFN, nil, "Frank Dawson")
	rfctest.AssertVCardLine(t, out, PropOrg, nil, "Lotus Development Corporation")
	rfctest.AssertVCardLine(t, out, PropEmail, map[string]string{"TYPE": "INTERNET"}, "Frank_Dawson@Lotus.com")
	rfctest.AssertVCardLine(t, out, PropURL, nil, "http://home.earthlink.net/~fdawson")
}

// TestRoundtrip_NeutralExportImport builds a Record directly (neutral -> P ->
// neutral) and checks the round trip preserves a representative set of
// fully-mapped fields.
func TestRoundtrip_NeutralExportImport(t *testing.T) {
	pref := 1
	rec := neutralRoundtripFixture(pref)

	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	got, _, err := (Adapter{}).Import(out)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if got.Card.UID != rec.Card.UID {
		t.Errorf("UID = %q, want %q", got.Card.UID, rec.Card.UID)
	}
	if got.Card.Name == nil || got.Card.Name.Full != rec.Card.Name.Full {
		t.Errorf("Name.Full = %+v, want %q", got.Card.Name, rec.Card.Name.Full)
	}
	if len(got.Card.Emails) != 1 || got.Card.Emails[0].Address != rec.Card.Emails[0].Address {
		t.Errorf("Emails = %+v, want [{Address: %s}]", got.Card.Emails, rec.Card.Emails[0].Address)
	}
	if len(got.Card.Phones) != 1 || got.Card.Phones[0].Number != rec.Card.Phones[0].Number {
		t.Errorf("Phones = %+v, want [{Number: %s}]", got.Card.Phones, rec.Card.Phones[0].Number)
	}
	if len(got.Card.Addresses) != 1 {
		t.Fatalf("Addresses = %+v, want 1 entry", got.Card.Addresses)
	}
}
