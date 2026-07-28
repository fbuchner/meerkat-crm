package vcard3

import "testing"

// Concepts covered: impp, social.
func init() {
	registerImportCoverage("impp", "social")
}

const imppImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"IMPP;TYPE=WORK:xmpp:frank@example.com\n" +
	"END:VCARD\n"

func TestImport_Impp(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(imppImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.ImppAddresses) != 1 {
		t.Fatalf("ImppAddresses = %+v, want 1 entry", rec.Card.ImppAddresses)
	}
	os := rec.Card.ImppAddresses[0]
	if os.URI != "xmpp:frank@example.com" {
		t.Errorf("URI = %q, want %q", os.URI, "xmpp:frank@example.com")
	}
	if os.Service != "" {
		t.Errorf("Service = %q, want empty", os.Service)
	}
	if len(rec.Card.SocialProfiles) != 0 {
		t.Errorf("SocialProfiles = %+v, want none", rec.Card.SocialProfiles)
	}
}

// RFC 9554 SS4.9/4.10: SERVICE-TYPE/USERNAME "MAY be specified on an IMPP or
// a SOCIALPROFILE property" — vCard 3.0 has no real custom parameters, so
// this adapter emulates them as grouped X-SERVICE-TYPE/X-USERNAME companion
// properties sharing IMPP's GROUP token (the same convention already used for
// X-SOCIALPROFILE/X-SERVICE-TYPE). A service-type/username-tagged IMPP field
// MUST populate Service/User on import — it must not be silently dropped.
const imppImportWithServiceAndUsernameVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"item1.IMPP;TYPE=WORK:xmpp:frank@example.com\n" +
	"item1.X-SERVICE-TYPE:Jabber\n" +
	"item1.X-USERNAME:frank94\n" +
	"END:VCARD\n"

func TestImport_ImppWithServiceAndUsername(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(imppImportWithServiceAndUsernameVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.ImppAddresses) != 1 {
		t.Fatalf("ImppAddresses = %+v, want 1 entry", rec.Card.ImppAddresses)
	}
	os := rec.Card.ImppAddresses[0]
	if os.URI != "xmpp:frank@example.com" {
		t.Errorf("URI = %q, want %q", os.URI, "xmpp:frank@example.com")
	}
	if os.Service != "Jabber" {
		t.Errorf("Service = %q, want %q", os.Service, "Jabber")
	}
	if os.User != "frank94" {
		t.Errorf("User = %q, want %q", os.User, "frank94")
	}
}

const socialImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"item1.X-SOCIALPROFILE;TYPE=WORK:https://example.com/@frank\n" +
	"item1.X-SERVICE-TYPE:Mastodon\n" +
	"END:VCARD\n"

func TestImport_Social(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(socialImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.SocialProfiles) != 1 {
		t.Fatalf("SocialProfiles = %+v, want 1 entry", rec.Card.SocialProfiles)
	}
	os := rec.Card.SocialProfiles[0]
	if os.URI != "https://example.com/@frank" {
		t.Errorf("URI = %q, want %q", os.URI, "https://example.com/@frank")
	}
	if os.Service != "Mastodon" {
		t.Errorf("Service = %q, want %q", os.Service, "Mastodon")
	}
	if len(rec.Card.ImppAddresses) != 0 {
		t.Errorf("ImppAddresses = %+v, want none", rec.Card.ImppAddresses)
	}
}

// Mirrors TestImport_ImppWithServiceAndUsername: RFC 9554 SS4.9/4.10 also
// allows SERVICE-TYPE/USERNAME on SOCIALPROFILE, so a grouped X-USERNAME
// companion field alongside X-SOCIALPROFILE/X-SERVICE-TYPE must populate User
// (previously never read at all for SOCIALPROFILE).
const socialImportWithUsernameVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"item1.X-SOCIALPROFILE;TYPE=WORK:https://example.com/@frank\n" +
	"item1.X-SERVICE-TYPE:Mastodon\n" +
	"item1.X-USERNAME:frank94\n" +
	"END:VCARD\n"

func TestImport_SocialWithUsername(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(socialImportWithUsernameVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.SocialProfiles) != 1 {
		t.Fatalf("SocialProfiles = %+v, want 1 entry", rec.Card.SocialProfiles)
	}
	os := rec.Card.SocialProfiles[0]
	if os.URI != "https://example.com/@frank" {
		t.Errorf("URI = %q, want %q", os.URI, "https://example.com/@frank")
	}
	if os.Service != "Mastodon" {
		t.Errorf("Service = %q, want %q", os.Service, "Mastodon")
	}
	if os.User != "frank94" {
		t.Errorf("User = %q, want %q", os.User, "frank94")
	}
}
