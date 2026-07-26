package jscontact

import (
	"encoding/json"
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

func TestExport_MediaResources(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "media-resources-example",
		Media: []contactmodel.Resource{
			{ID: "M1", Kind: "photo", URI: "https://example.com/photo.jpg"},
			{ID: "M2", Kind: "logo", URI: "https://example.com/logo.png"},
			{ID: "M3", Kind: "sound", URI: "https://example.com/sound.wav"},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/media/M1/uri", "https://example.com/photo.jpg")
	rfctest.AssertJSONPointer(t, out, "/media/M2/uri", "https://example.com/logo.png")
	rfctest.AssertJSONPointer(t, out, "/media/M3/uri", "https://example.com/sound.wav")
}

func TestExport_CalendarAndFreeBusy(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:          "calendar-resources-example",
		Calendars:    []contactmodel.Resource{{ID: "C1", URI: "https://example.com/calendar"}},
		FreeBusyURLs: []contactmodel.Resource{{ID: "C2", URI: "https://example.com/freebusy"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/calendars/C1/uri", "https://example.com/calendar")
	rfctest.AssertJSONPointer(t, out, "/calendars/C2/uri", "https://example.com/freebusy")
	rfctest.AssertJSONPointer(t, out, "/calendars/C2/kind", "freeBusy")

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	c1 := doc["calendars"].(map[string]any)["C1"].(map[string]any)
	if _, present := c1["kind"]; present {
		t.Errorf("expected no kind key for a plain Calendars entry, got %+v", c1)
	}
}

func TestExport_CaladrURIAndKey(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:                 "caladruri-key-example",
		SchedulingAddresses: []contactmodel.Resource{{ID: "S1", URI: "mailto:janedoe@example.com"}},
		CryptoKeys:          []contactmodel.Resource{{ID: "K1", URI: "https://example.com/key.pem"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/schedulingAddresses/S1/uri", "mailto:janedoe@example.com")
	rfctest.AssertJSONPointer(t, out, "/cryptoKeys/K1/uri", "https://example.com/key.pem")
}

func TestExport_DirectoryAndSource(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "directory-source-example",
		Directories: []contactmodel.Resource{
			{ID: "D1", Kind: "directory", URI: "ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering", ListAs: intPtr(1)},
			{ID: "D2", Kind: "entry", URI: "https://example.com/directory/jdoe.vcf"},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/directories/D1/uri", "ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering")
	rfctest.AssertJSONPointer(t, out, "/directories/D2/uri", "https://example.com/directory/jdoe.vcf")
}

func TestExport_LinkAndContactURI(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:         "link-contacturi-example",
		Links:       []contactmodel.Resource{{ID: "L1", URI: "https://example.com/jdoe"}},
		ContactURIs: []contactmodel.Resource{{ID: "L2", URI: "mailto:contact@example.com"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/links/L1/uri", "https://example.com/jdoe")
	rfctest.AssertJSONPointer(t, out, "/links/L2/uri", "mailto:contact@example.com")
	rfctest.AssertJSONPointer(t, out, "/links/L2/kind", "contact")

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	l1 := doc["links"].(map[string]any)["L1"].(map[string]any)
	if _, present := l1["kind"]; present {
		t.Errorf("expected no kind key for a plain Links entry, got %+v", l1)
	}
}
