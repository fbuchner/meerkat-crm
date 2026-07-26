package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concepts covered: photo, logo, sound, calendar, freebusy, caladruri, key,
// source, link.
func init() {
	registerExportCoverage(
		"photo", "logo", "sound", "calendar", "freebusy", "caladruri",
		"key", "source", "link",
	)
}

func TestExport_Photo(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Media: []contactmodel.Resource{{Kind: "photo", URI: "data:image/jpeg;base64,/9j/4AAQ"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropPhoto, map[string]string{"ENCODING": "b", "TYPE": "JPEG"}, "/9j/4AAQ")
}

func TestExport_Photo_BareURI(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Media: []contactmodel.Resource{{Kind: "photo", URI: "https://example.com/frank.jpg"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropPhoto, nil, "https://example.com/frank.jpg")
}

func TestExport_Logo(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Media: []contactmodel.Resource{{Kind: "logo", URI: "data:image/png;base64,iVBORw0K"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropLogo, map[string]string{"ENCODING": "b", "TYPE": "PNG"}, "iVBORw0K")
}

func TestExport_Sound(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Media: []contactmodel.Resource{{Kind: "sound", URI: "data:audio/basic;base64,AAAA"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropSound, map[string]string{"ENCODING": "b", "TYPE": "BASIC"}, "AAAA")
}

func TestExport_Calendar(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Calendars: []contactmodel.Resource{{URI: "http://cal.example.com/frank"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropCaluri, nil, "http://cal.example.com/frank")
}

func TestExport_FreeBusy(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		FreeBusyURLs: []contactmodel.Resource{{URI: "http://fb.example.com/frank"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropFburl, nil, "http://fb.example.com/frank")
}

func TestExport_Caladruri(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SchedulingAddresses: []contactmodel.Resource{{URI: "mailto:frank@example.com"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropCaladruri, nil, "mailto:frank@example.com")
}

func TestExport_Key(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		CryptoKeys: []contactmodel.Resource{{URI: "data:application/x509;base64,MIIBIjANBgkqhkiG"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropKey, map[string]string{"ENCODING": "b", "TYPE": "X509"}, "MIIBIjANBgkqhkiG")
}

func TestExport_Source(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Directories: []contactmodel.Resource{{Kind: "entry", URI: "ldap://ldap.example.com/cn=Frank"}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropSource, nil, "ldap://ldap.example.com/cn=Frank")
}

func TestExport_Link(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Links: []contactmodel.Resource{{URI: "http://home.earthlink.net/~fdawson", Contexts: []string{"work"}}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropURL, map[string]string{"TYPE": "WORK"}, "http://home.earthlink.net/~fdawson")
}
