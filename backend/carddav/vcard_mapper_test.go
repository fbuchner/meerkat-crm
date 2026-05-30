package carddav

import (
	"meerkat/models"
	"testing"

	"github.com/emersion/go-vcard"
)

// TestVCardRoundTrip verifies that the multi-valued and structured vCard fields
// survive a Contact -> vCard -> Contact round trip without data loss.
func TestVCardRoundTrip(t *testing.T) {
	original := &models.Contact{
		Firstname:  "Ada",
		Lastname:   "Lovelace",
		MiddleName: "Augusta",
		Prefix:     "Dr.",
		Suffix:     "PhD",
		Nickname:   "Ada",
		Emails: []models.ContactEmail{
			{Type: "home", Value: "ada@home.example"},
			{Type: "work", Value: "ada@work.example"},
		},
		Phones: []models.ContactPhone{
			{Type: "cell", Value: "+15551234567"},
			{Type: "work", Value: "+15557654321"},
		},
		Addresses: []models.ContactAddress{
			{Type: "home", Street: "1 Analytical Way", City: "London", Region: "ENG", Postal: "EC1", Country: "UK"},
		},
		URLs:         []models.ContactURL{{Type: "home", Value: "https://example.com"}},
		IMPPs:        []models.ContactIMPP{{Type: "telegram", Value: "@ada"}},
		Organization: "Analytical Engines Ltd",
		Department:   "R&D",
		JobTitle:     "Mathematician",
		Role:         "Pioneer",
		Birthday:     "1815-12-10",
		Anniversary:  "1835-07-08",
		Circles:      []string{"friends", "history"},
	}

	card := ContactToVCard(original, "")
	got, _, _, _ := VCardToContact(card, nil)

	if got.Firstname != original.Firstname || got.Lastname != original.Lastname {
		t.Errorf("name mismatch: got %q %q", got.Firstname, got.Lastname)
	}
	if got.MiddleName != "Augusta" || got.Prefix != "Dr." || got.Suffix != "PhD" {
		t.Errorf("structured name parts lost: %+v", got)
	}
	if len(got.Emails) != 2 {
		t.Fatalf("expected 2 emails, got %d: %+v", len(got.Emails), got.Emails)
	}
	if got.Emails[0].Value != "ada@home.example" || got.Emails[0].Type != "home" {
		t.Errorf("email[0] mismatch: %+v", got.Emails[0])
	}
	if got.Emails[1].Type != "work" {
		t.Errorf("email[1] type lost: %+v", got.Emails[1])
	}
	if got.Email != "ada@home.example" {
		t.Errorf("primary email scalar not set: %q", got.Email)
	}
	if len(got.Phones) != 2 || got.Phones[0].Type != "cell" {
		t.Errorf("phones mismatch: %+v", got.Phones)
	}
	if len(got.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(got.Addresses))
	}
	a := got.Addresses[0]
	if a.Street != "1 Analytical Way" || a.City != "London" || a.Region != "ENG" || a.Postal != "EC1" || a.Country != "UK" {
		t.Errorf("address structure lost: %+v", a)
	}
	if len(got.URLs) != 1 || got.URLs[0].Value != "https://example.com" {
		t.Errorf("url lost: %+v", got.URLs)
	}
	if len(got.IMPPs) != 1 || got.IMPPs[0].Value != "@ada" || got.IMPPs[0].Type != "telegram" {
		t.Errorf("impp lost: %+v", got.IMPPs)
	}
	if got.Organization != "Analytical Engines Ltd" || got.Department != "R&D" {
		t.Errorf("org/department lost: org=%q dept=%q", got.Organization, got.Department)
	}
	if got.JobTitle != "Mathematician" || got.Role != "Pioneer" {
		t.Errorf("title/role lost: title=%q role=%q", got.JobTitle, got.Role)
	}
	if got.Anniversary != "1835-07-08" {
		t.Errorf("anniversary lost: %q", got.Anniversary)
	}
}

// TestVCardUnmappedPreserved verifies an unknown property is kept in VCardExtra.
func TestVCardUnmappedPreserved(t *testing.T) {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldFormattedName, "Test Person")
	card.SetValue(vcard.FieldName, "Person;Test;;;")
	card.Add("X-CUSTOM-PROP", &vcard.Field{Value: "keep-me"})

	got, _, _, _ := VCardToContact(card, nil)
	if got.VCardExtra == "" {
		t.Fatal("expected VCardExtra to capture unmapped X-CUSTOM-PROP")
	}

	// Re-export and confirm the unmapped property is restored.
	out := ContactToVCard(got, "")
	if v := out.Value("X-CUSTOM-PROP"); v != "keep-me" {
		t.Errorf("unmapped property not restored on export: %q", v)
	}
}

// TestNoDuplicateFromStaleExtra verifies that a property which now maps to a column
// is not emitted twice when a stale copy still lingers in vcard_extra (the situation
// migration 000021 cleans up, with this export guard as the safety net).
func TestNoDuplicateFromStaleExtra(t *testing.T) {
	c := &models.Contact{
		Firstname: "Stale",
		Lastname:  "Extra",
		// New column has the website…
		URLs: []models.ContactURL{{Type: "home", Value: "https://example.com"}},
		// …and a leftover pre-upgrade copy still sits in vcard_extra.
		VCardExtra: `{"properties":{"URL":[{"Value":"https://example.com","Params":{},"Group":""}],"X-CUSTOM":[{"Value":"keep","Params":{},"Group":""}]}}`,
	}

	card := ContactToVCard(c, "")

	if got := len(card[vcard.FieldURL]); got != 1 {
		t.Errorf("expected exactly 1 URL on export, got %d: %v", got, card[vcard.FieldURL])
	}
	// Genuinely unmapped properties must still be restored.
	if v := card.Value("X-CUSTOM"); v != "keep" {
		t.Errorf("unmapped X-CUSTOM should still be restored, got %q", v)
	}
}

// TestLegacyScalarFallback verifies a contact with only the legacy scalar fields
// still exports valid EMAIL/TEL/ADR entries.
func TestLegacyScalarFallback(t *testing.T) {
	c := &models.Contact{
		Firstname: "Legacy",
		Lastname:  "User",
		Email:     "legacy@example.com",
		Phone:     "+15550000000",
		Address:   "10 Old Street",
	}
	card := ContactToVCard(c, "")
	if v := card.Value(vcard.FieldEmail); v != "legacy@example.com" {
		t.Errorf("legacy email not exported: %q", v)
	}
	if v := card.Value(vcard.FieldTelephone); v != "+15550000000" {
		t.Errorf("legacy phone not exported: %q", v)
	}
	if len(card.Addresses()) == 0 {
		t.Error("legacy address not exported")
	}
}

// TestStructuredSemicolonRoundTrip verifies that a literal ";" inside ORG/ADR
// components survives a round trip instead of leaking into the next component.
func TestStructuredSemicolonRoundTrip(t *testing.T) {
	original := &models.Contact{
		Firstname:    "Semi",
		Lastname:     "Colon",
		Organization: "Smith; Jones & Co",
		Department:   "R&D; Labs",
		Addresses: []models.ContactAddress{
			{Type: "work", Street: "1 Main St; Suite 2", City: "Town; ville", Region: "RE", Postal: "12345", Country: "UK"},
		},
	}

	card := ContactToVCard(original, "")
	got, _, _, _ := VCardToContact(card, nil)

	if got.Organization != original.Organization {
		t.Errorf("organization corrupted: got %q want %q", got.Organization, original.Organization)
	}
	if got.Department != original.Department {
		t.Errorf("department corrupted: got %q want %q", got.Department, original.Department)
	}
	if len(got.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(got.Addresses))
	}
	a := got.Addresses[0]
	if a.Street != "1 Main St; Suite 2" || a.City != "Town; ville" {
		t.Errorf("address component with ';' corrupted: %+v", a)
	}
}

// TestComponentBackslashRoundTrip verifies a literal backslash in a structured
// component is not mangled by our escaping interacting with go-vcard's own.
func TestComponentBackslashRoundTrip(t *testing.T) {
	original := &models.Contact{
		Firstname:    "Back",
		Lastname:     "Slash",
		Organization: `Path\To\Co`,
	}
	got, _, _, _ := VCardToContact(ContactToVCard(original, ""), nil)
	if got.Organization != `Path\To\Co` {
		t.Errorf("backslash corrupted: got %q", got.Organization)
	}
}
