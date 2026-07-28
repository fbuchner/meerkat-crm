package vcard4

import (
	"testing"

	"mycorrhizal/internal/rfctest"
)

// Concepts: email, phone, impp, social.
// Golden fixtures: rfc6350-baseline.v4.vcf (EMAIL), socialprofile.v4.vcf
// (RFC 9554 §3.5, SOCIALPROFILE). impp.v4.vcf is a per-concept fixture (no
// RFC-worked example needed for a baseline URI property). phone has no v4
// fixture of its own (phone.jscontact.json illustrates the concept on the
// JSContact side; rfc2426-baseline.v3.vcf's TEL lines are vcard3's concern),
// so it is exercised via a minimal hand-built card here.
func init() {
	registerImportCoverage("email", "phone", "impp", "social")
}

func TestImport_Email(t *testing.T) {
	raw := rfctest.LoadFixture("rfc6350-baseline.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Emails) != 1 || rec.Card.Emails[0].Address != "jdoe@example.com" {
		t.Fatalf("Emails = %+v", rec.Card.Emails)
	}
}

func TestImport_Phone(t *testing.T) {
	raw := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\nUID:phone-example\r\nFN:Test\r\nTEL;PREF=1;TYPE=home,voice:+1-555-555-0100\r\nEND:VCARD\r\n")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Phones) != 1 {
		t.Fatalf("Phones = %+v", rec.Card.Phones)
	}
	p := rec.Card.Phones[0]
	if p.Number != "+1-555-555-0100" {
		t.Errorf("Number = %q", p.Number)
	}
	if p.Pref == nil || *p.Pref != 1 {
		t.Errorf("Pref = %v, want 1", p.Pref)
	}
	if len(p.Contexts) != 1 || p.Contexts[0] != "private" {
		t.Errorf("Contexts = %v, want [private]", p.Contexts)
	}
	if len(p.Features) != 1 || p.Features[0] != "voice" {
		t.Errorf("Features = %v, want [voice]", p.Features)
	}
}

func TestImport_Impp(t *testing.T) {
	raw := rfctest.LoadFixture("impp.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.ImppAddresses) != 1 || rec.Card.ImppAddresses[0].URI != "xmpp:alice@example.com" {
		t.Fatalf("ImppAddresses = %+v", rec.Card.ImppAddresses)
	}
}

// TestImport_ImppServiceTypeUsername covers the bug fix: RFC 9554 §4.9/§4.10
// state SERVICE-TYPE/USERNAME "MAY be specified on an IMPP or a
// SOCIALPROFILE property" — importOnlineServices previously only read them
// on the SOCIALPROFILE branch, silently losing both params when they
// appeared on IMPP.
func TestImport_ImppServiceTypeUsername(t *testing.T) {
	raw := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\nUID:impp-svc\r\nFN:Test\r\n" +
		"IMPP;SERVICE-TYPE=xmpp;USERNAME=alice@example.com:xmpp:alice@example.com\r\nEND:VCARD\r\n")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.ImppAddresses) != 1 {
		t.Fatalf("ImppAddresses = %+v", rec.Card.ImppAddresses)
	}
	os := rec.Card.ImppAddresses[0]
	if os.URI != "xmpp:alice@example.com" {
		t.Errorf("URI = %q, want xmpp:alice@example.com", os.URI)
	}
	if os.Service != "xmpp" {
		t.Errorf("Service = %q, want xmpp", os.Service)
	}
	if os.User != "alice@example.com" {
		t.Errorf("User = %q, want alice@example.com", os.User)
	}
}

func TestImport_SocialProfile(t *testing.T) {
	raw := rfctest.LoadFixture("socialprofile.v4.vcf")
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.SocialProfiles) != 4 {
		t.Fatalf("SocialProfiles = %+v, want 4 entries", rec.Card.SocialProfiles)
	}
	// SOCIALPROFILE;SERVICE-TYPE=Mastodon:https://example.com/@foo
	if got := rec.Card.SocialProfiles[0]; got.Service != "Mastodon" || got.URI != "https://example.com/@foo" {
		t.Errorf("[0] = %+v", got)
	}
	// SOCIALPROFILE:https://example.com/ietf
	if got := rec.Card.SocialProfiles[1]; got.URI != "https://example.com/ietf" || got.Service != "" {
		t.Errorf("[1] = %+v", got)
	}
	// SOCIALPROFILE;SERVICE-TYPE=SomeSite;VALUE=text:peter94
	if got := rec.Card.SocialProfiles[2]; got.Service != "SomeSite" || got.User != "peter94" || got.URI != "" {
		t.Errorf("[2] = %+v", got)
	}
	// SOCIALPROFILE;USERNAME="The Foo":https://example.com/@foo
	if got := rec.Card.SocialProfiles[3]; got.User != "The Foo" || got.URI != "https://example.com/@foo" {
		t.Errorf("[3] = %+v", got)
	}
}
