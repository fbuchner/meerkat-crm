package vcard4

import (
	"testing"

	"mycorrhizal/internal/rfctest"
)

// Concepts: photo, logo, sound, calendar, freebusy, caladruri, key,
// directory, source, link, contacturi. All per-concept fixtures (see
// docs/fork-plan/... / backend/internal/rfctest/fixtures/SOURCES.md);
// directory/contacturi are RFC 6715/8605 verbatim examples.
func init() {
	registerImportCoverage(
		"photo", "logo", "sound", "calendar", "freebusy", "caladruri",
		"key", "directory", "source", "link", "contacturi",
	)
}

func TestImport_Photo(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("photo.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Media) != 1 || rec.Card.Media[0].Kind != "photo" || rec.Card.Media[0].URI != "https://example.com/photo.jpg" {
		t.Fatalf("Media = %+v", rec.Card.Media)
	}
}

func TestImport_Logo(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("logo.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Media) != 1 || rec.Card.Media[0].Kind != "logo" || rec.Card.Media[0].URI != "https://example.com/logo.png" {
		t.Fatalf("Media = %+v", rec.Card.Media)
	}
}

func TestImport_Sound(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("sound.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Media) != 1 || rec.Card.Media[0].Kind != "sound" || rec.Card.Media[0].URI != "https://example.com/sound.wav" {
		t.Fatalf("Media = %+v", rec.Card.Media)
	}
}

func TestImport_Calendar(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("calendar.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Calendars) != 1 || rec.Card.Calendars[0].Kind != "" || rec.Card.Calendars[0].URI != "https://example.com/calendar" {
		t.Fatalf("Calendars = %+v", rec.Card.Calendars)
	}
}

func TestImport_FreeBusy(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("freebusy.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.FreeBusyURLs) != 1 || rec.Card.FreeBusyURLs[0].Kind != "" || rec.Card.FreeBusyURLs[0].URI != "https://example.com/freebusy" {
		t.Fatalf("FreeBusyURLs = %+v", rec.Card.FreeBusyURLs)
	}
}

func TestImport_Caladruri(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("caladruri.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.SchedulingAddresses) != 1 || rec.Card.SchedulingAddresses[0].URI != "mailto:janedoe@example.com" {
		t.Fatalf("SchedulingAddresses = %+v", rec.Card.SchedulingAddresses)
	}
}

func TestImport_Key(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("key.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.CryptoKeys) != 1 || rec.Card.CryptoKeys[0].URI != "https://example.com/key.pem" {
		t.Fatalf("CryptoKeys = %+v", rec.Card.CryptoKeys)
	}
}

func TestImport_Directory(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("directory.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Directories) != 1 {
		t.Fatalf("Directories = %+v", rec.Card.Directories)
	}
	d := rec.Card.Directories[0]
	if d.Kind != "directory" || d.URI != "ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering" {
		t.Errorf("Directories[0] = %+v", d)
	}
	if d.Pref == nil || *d.Pref != 1 {
		t.Errorf("Pref = %v, want 1", d.Pref)
	}
}

func TestImport_Source(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("source.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Directories) != 1 || rec.Card.Directories[0].Kind != "entry" || rec.Card.Directories[0].URI != "https://example.com/directory/jdoe.vcf" {
		t.Fatalf("Directories = %+v", rec.Card.Directories)
	}
}

func TestImport_Link(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("link.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Links) != 1 || rec.Card.Links[0].Kind != "" || rec.Card.Links[0].URI != "https://example.com/jdoe" {
		t.Fatalf("Links = %+v", rec.Card.Links)
	}
}

func TestImport_ContactURI(t *testing.T) {
	rec, _, err := Adapter{}.Import(rfctest.LoadFixture("contacturi.v4.vcf"))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.ContactURIs) != 1 || rec.Card.ContactURIs[0].Kind != "" || rec.Card.ContactURIs[0].URI != "mailto:contact@example.com" {
		t.Fatalf("ContactURIs = %+v", rec.Card.ContactURIs)
	}
}
