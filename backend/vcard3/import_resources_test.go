package vcard3

import "testing"

// Concepts covered: photo, logo, sound, calendar, freebusy, caladruri, key,
// source, link.
func init() {
	registerImportCoverage(
		"photo", "logo", "sound", "calendar", "freebusy", "caladruri",
		"key", "source", "link",
	)
}

const resourcesImportVCF = "BEGIN:VCARD\n" +
	"VERSION:3.0\n" +
	"FN:Frank Dawson\n" +
	"PHOTO;ENCODING=b;TYPE=JPEG:/9j/4AAQ\n" +
	"LOGO;ENCODING=b;TYPE=PNG:iVBORw0K\n" +
	"SOUND;ENCODING=b;TYPE=BASIC:AAAA\n" +
	"CALURI:http://cal.example.com/frank\n" +
	"FBURL:http://fb.example.com/frank\n" +
	"CALADRURI:mailto:frank@example.com\n" +
	"KEY;ENCODING=b;TYPE=X509:MIIBIjANBgkqhkiG\n" +
	"SOURCE:ldap://ldap.example.com/cn=Frank\n" +
	"URL;TYPE=WORK:http://home.earthlink.net/~fdawson\n" +
	"END:VCARD\n"

func TestImport_Photo(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	want := "data:image/jpeg;base64,/9j/4AAQ"
	for _, m := range rec.Card.Media {
		if m.Kind == "photo" {
			if m.URI != want {
				t.Errorf("photo URI = %q, want %q", m.URI, want)
			}
			return
		}
	}
	t.Errorf("no photo media entry found in %+v", rec.Card.Media)
}

func TestImport_Logo(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	want := "data:image/png;base64,iVBORw0K"
	for _, m := range rec.Card.Media {
		if m.Kind == "logo" {
			if m.URI != want {
				t.Errorf("logo URI = %q, want %q", m.URI, want)
			}
			return
		}
	}
	t.Errorf("no logo media entry found in %+v", rec.Card.Media)
}

func TestImport_Sound(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	want := "data:audio/basic;base64,AAAA"
	for _, m := range rec.Card.Media {
		if m.Kind == "sound" {
			if m.URI != want {
				t.Errorf("sound URI = %q, want %q", m.URI, want)
			}
			return
		}
	}
	t.Errorf("no sound media entry found in %+v", rec.Card.Media)
}

func TestImport_Calendar(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	for _, c := range rec.Card.Calendars {
		if c.URI == "http://cal.example.com/frank" {
			return
		}
	}
	t.Errorf("no calendar entry found in %+v", rec.Card.Calendars)
}

func TestImport_FreeBusy(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	for _, c := range rec.Card.FreeBusyURLs {
		if c.URI == "http://fb.example.com/frank" {
			return
		}
	}
	t.Errorf("no freeBusy entry found in %+v", rec.Card.FreeBusyURLs)
}

func TestImport_Caladruri(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.SchedulingAddresses) != 1 || rec.Card.SchedulingAddresses[0].URI != "mailto:frank@example.com" {
		t.Errorf("SchedulingAddresses = %+v, want [{URI: mailto:frank@example.com}]", rec.Card.SchedulingAddresses)
	}
}

func TestImport_Key(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	want := "data:application/x509;base64,MIIBIjANBgkqhkiG"
	if len(rec.Card.CryptoKeys) != 1 || rec.Card.CryptoKeys[0].URI != want {
		t.Errorf("CryptoKeys = %+v, want [{URI: %s}]", rec.Card.CryptoKeys, want)
	}
}

func TestImport_Source(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	for _, d := range rec.Card.Directories {
		if d.Kind == "entry" && d.URI == "ldap://ldap.example.com/cn=Frank" {
			return
		}
	}
	t.Errorf("no source/entry directory found in %+v", rec.Card.Directories)
}

func TestImport_Link(t *testing.T) {
	rec, _, err := (Adapter{}).Import([]byte(resourcesImportVCF))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Links) != 1 || rec.Card.Links[0].URI != "http://home.earthlink.net/~fdawson" {
		t.Errorf("Links = %+v, want [{URI: http://home.earthlink.net/~fdawson}]", rec.Card.Links)
	}
	if len(rec.Card.Links) == 1 && (len(rec.Card.Links[0].Contexts) != 1 || rec.Card.Links[0].Contexts[0] != "work") {
		t.Errorf("Links[0].Contexts = %v, want [work]", rec.Card.Links[0].Contexts)
	}
}
