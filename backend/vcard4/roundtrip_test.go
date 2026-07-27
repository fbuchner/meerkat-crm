package vcard4

import (
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/internal/rfctest"
)

// Spot-check round trips (40-testing.md §40.1 point 4): a handful of
// P -> neutral -> P and neutral -> P -> neutral checks on whole fixtures,
// confidence only (not exhaustive field coverage — that's the
// import_*/export_*/coverage_test.go job).

func TestRoundtrip_RFC6350Baseline(t *testing.T) {
	raw := rfctest.LoadFixture("rfc6350-baseline.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rec2, _, err := Adapter{}.Import(out)
	if err != nil {
		t.Fatalf("re-Import: %v", err)
	}
	if rec2.Card.UID != rec.Card.UID {
		t.Errorf("UID: got %q, want %q", rec2.Card.UID, rec.Card.UID)
	}
	if rec2.Card.Name == nil || rec.Card.Name == nil || rec2.Card.Name.Full != rec.Card.Name.Full {
		t.Errorf("Name.Full: got %+v, want %+v", rec2.Card.Name, rec.Card.Name)
	}
	if len(rec2.Card.Emails) != 1 || rec2.Card.Emails[0].Address != "jdoe@example.com" {
		t.Errorf("Emails = %+v", rec2.Card.Emails)
	}
	// CLIENTPIDMAP has no correspondence row; it must survive the round trip
	// via passthrough (20.5) as a TRUE inverse — not just present after
	// re-import, but re-emitted on export with its original value intact.
	rfctest.AssertVCardLine(t, out, "CLIENTPIDMAP", nil, "1;urn:uuid:53e374d9-337e-4727-8803-a1e9c14e0556")
	if len(rec2.Passthrough.VCard) != len(rec.Passthrough.VCard) {
		t.Errorf("Passthrough.VCard: got %+v, want same length as %+v", rec2.Passthrough.VCard, rec.Passthrough.VCard)
	}
}

func TestRoundtrip_NExpanded(t *testing.T) {
	raw := rfctest.LoadFixture("n-expanded.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "N", nil, "Stevenson;John;Philip,Paul;Dr.;Jr.,M.D.,A.C.P.;;Jr.")
	rfctest.AssertVCardLine(t, out, "FN", nil, "John Philip Paul Stevenson Jr.")
}

func TestRoundtrip_AdrExpanded(t *testing.T) {
	raw := rfctest.LoadFixture("adr-expanded.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rec2, _, err := Adapter{}.Import(out)
	if err != nil {
		t.Fatalf("re-Import: %v", err)
	}
	if len(rec2.Card.Addresses) != 1 {
		t.Fatalf("Addresses = %+v", rec2.Card.Addresses)
	}
	if rec2.Card.Addresses[0].Coordinates != "geo:12.3457,78.910" {
		t.Errorf("Coordinates = %q, want geo:12.3457,78.910", rec2.Card.Addresses[0].Coordinates)
	}
	var gotName, gotNumber, gotLocality string
	for _, c := range rec2.Card.Addresses[0].Components {
		switch c.Kind {
		case "name":
			gotName = c.Value
		case "number":
			gotNumber = c.Value
		case "locality":
			gotLocality = c.Value
		}
	}
	if gotName != "Main Street" || gotNumber != "123" || gotLocality != "Any Town" {
		t.Errorf("components: name=%q number=%q locality=%q", gotName, gotNumber, gotLocality)
	}
}

func TestRoundtrip_Pronouns(t *testing.T) {
	raw := rfctest.LoadFixture("pronouns.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rec2, _, err := Adapter{}.Import(out)
	if err != nil {
		t.Fatalf("re-Import: %v", err)
	}
	if rec2.Card.SpeakToAs == nil || len(rec2.Card.SpeakToAs.Pronouns) != 2 {
		t.Fatalf("SpeakToAs = %+v", rec2.Card.SpeakToAs)
	}
	if rec2.Card.SpeakToAs.Pronouns[0].Pronouns != "xe/xir" || rec2.Card.SpeakToAs.Pronouns[1].Pronouns != "they/them" {
		t.Errorf("Pronouns = %+v", rec2.Card.SpeakToAs.Pronouns)
	}
}

func TestRoundtrip_TitleRole(t *testing.T) {
	raw := rfctest.LoadFixture("title-role.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rec2, _, err := Adapter{}.Import(out)
	if err != nil {
		t.Fatalf("re-Import: %v", err)
	}
	if len(rec2.Card.Organizations) != 1 || rec2.Card.Organizations[0].Name != "ABC, Inc." {
		t.Fatalf("Organizations = %+v", rec2.Card.Organizations)
	}
	orgID := rec2.Card.Organizations[0].ID
	var roleLinked bool
	for _, ti := range rec2.Card.Titles {
		if ti.Kind == "role" {
			roleLinked = ti.OrganizationID == orgID && ti.OrganizationID != ""
		}
	}
	if !roleLinked {
		t.Errorf("Titles = %+v, want role.OrganizationID == %q (GROUP link survives round trip)", rec2.Card.Titles, orgID)
	}
}

// TestRoundtrip_PropIDIdentity is a dedicated check for 60-review-gates.md
// §60.3's "PROP-ID/ID round-trips" bullet: an element's ID, once exported as
// PROP-ID, must come back as the same ID on re-import (30-adapters.md §30.B).
func TestRoundtrip_PropIDIdentity(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Name: &contactmodel.Name{Full: "Test"},
		Emails: []contactmodel.Email{
			{ID: "custom-email-id", Address: "a@example.com"},
			{Address: "b@example.com"}, // no ID: must get a synthesized one on export.
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "EMAIL", map[string]string{"PROP-ID": "custom-email-id"}, "a@example.com")

	rec2, _, err := Adapter{}.Import(out)
	if err != nil {
		t.Fatalf("re-Import: %v", err)
	}
	if len(rec2.Card.Emails) != 2 {
		t.Fatalf("Emails = %+v", rec2.Card.Emails)
	}
	var gotCustom, gotSynthesized bool
	for _, e := range rec2.Card.Emails {
		if e.Address == "a@example.com" && e.ID == "custom-email-id" {
			gotCustom = true
		}
		if e.Address == "b@example.com" && e.ID != "" {
			gotSynthesized = true
		}
	}
	if !gotCustom {
		t.Errorf("Emails = %+v, want a@example.com to keep ID custom-email-id across the round trip", rec2.Card.Emails)
	}
	if !gotSynthesized {
		t.Errorf("Emails = %+v, want b@example.com to have a non-empty synthesized ID recovered via PROP-ID", rec2.Card.Emails)
	}
}
