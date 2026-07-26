package vcard4

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage(
		"photo", "logo", "sound", "calendar", "freebusy", "caladruri",
		"key", "directory", "source", "link", "contacturi",
	)
}

func TestExport_Photo(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Media: []contactmodel.Resource{{Kind: "photo", URI: "https://example.com/photo.jpg", MediaType: "image/jpeg"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "PHOTO", map[string]string{"MEDIATYPE": "image/jpeg"}, "https://example.com/photo.jpg")
}

func TestExport_Logo(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Media: []contactmodel.Resource{{Kind: "logo", URI: "https://example.com/logo.png"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "LOGO", nil, "https://example.com/logo.png")
}

func TestExport_Sound(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Media: []contactmodel.Resource{{Kind: "sound", URI: "https://example.com/sound.wav"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "SOUND", nil, "https://example.com/sound.wav")
}

func TestExport_Calendar(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Calendars: []contactmodel.Resource{{URI: "https://example.com/calendar", Pref: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "CALURI", map[string]string{"PREF": "1"}, "https://example.com/calendar")
}

func TestExport_FreeBusy(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		FreeBusyURLs: []contactmodel.Resource{{URI: "https://example.com/freebusy"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "FBURL", nil, "https://example.com/freebusy")
}

func TestExport_Caladruri(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SchedulingAddresses: []contactmodel.Resource{{URI: "mailto:janedoe@example.com"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "CALADRURI", nil, "mailto:janedoe@example.com")
}

func TestExport_Key(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		CryptoKeys: []contactmodel.Resource{{URI: "https://example.com/key.pem"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "KEY", nil, "https://example.com/key.pem")
}

func TestExport_Directory(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Directories: []contactmodel.Resource{{Kind: "directory", URI: "ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering", Pref: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "ORG-DIRECTORY", map[string]string{"PREF": "1"}, "ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering")
}

func TestExport_Source(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Directories: []contactmodel.Resource{{Kind: "entry", URI: "https://example.com/directory/jdoe.vcf"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "SOURCE", nil, "https://example.com/directory/jdoe.vcf")
}

func TestExport_Link(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Links: []contactmodel.Resource{{URI: "https://example.com/jdoe"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "URL", nil, "https://example.com/jdoe")
}

func TestExport_ContactURI(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		ContactURIs: []contactmodel.Resource{{URI: "mailto:contact@example.com", Pref: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "CONTACT-URI", map[string]string{"PREF": "1"}, "mailto:contact@example.com")
}
