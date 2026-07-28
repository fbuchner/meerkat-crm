package jscontact

import "testing"

// Concepts: photo, logo, sound, calendar, freebusy, caladruri, key, directory,
// source, link, contacturi. All Resource-shaped (URI + optional MediaType/
// Label/Contexts/Pref/ListAs). photo/logo/sound share Card.Media and
// directory/source share Card.Directories, both still disambiguated by
// .Kind. calendar/freebusy and link/contacturi, however, now route by the
// wire Calendar/Link object's own .Kind into two DISCRETE neutral fields
// each (Card.Calendars vs Card.FreeBusyURLs; Card.Links vs Card.ContactURIs)
// — see 20-correspondence.md's "calendar"/"freebusy" and "link"/"contacturi"
// rows. Values below match the corresponding golden *.v4.vcf fixtures' URIs
// (no dedicated .jscontact.json fixture exists per-concept).
func init() {
	registerImportCoverage(
		"photo", "logo", "sound", "calendar", "freebusy", "caladruri",
		"key", "directory", "source", "link", "contacturi",
	)
}

func TestImport_MediaResources(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "media-resources-example",
		"media": {
			"M1": { "@type": "Media", "kind": "photo", "uri": "https://example.com/photo.jpg" },
			"M2": { "@type": "Media", "kind": "logo", "uri": "https://example.com/logo.png" },
			"M3": { "@type": "Media", "kind": "sound", "uri": "https://example.com/sound.wav" }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Media) != 3 {
		t.Fatalf("len(Media) = %d, want 3", len(rec.Card.Media))
	}
	byKind := map[string]string{}
	for _, m := range rec.Card.Media {
		byKind[m.Kind] = m.URI
	}
	if byKind["photo"] != "https://example.com/photo.jpg" {
		t.Errorf("photo URI = %q", byKind["photo"])
	}
	if byKind["logo"] != "https://example.com/logo.png" {
		t.Errorf("logo URI = %q", byKind["logo"])
	}
	if byKind["sound"] != "https://example.com/sound.wav" {
		t.Errorf("sound URI = %q", byKind["sound"])
	}
}

func TestImport_CalendarAndFreeBusy(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "calendar-resources-example",
		"calendars": {
			"C1": { "@type": "Calendar", "kind": "calendar", "uri": "https://example.com/calendar" },
			"C2": { "@type": "Calendar", "kind": "freeBusy", "uri": "https://example.com/freebusy" }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Calendars) != 1 || rec.Card.Calendars[0].URI != "https://example.com/calendar" {
		t.Errorf("Calendars = %+v", rec.Card.Calendars)
	}
	if len(rec.Card.FreeBusyURLs) != 1 || rec.Card.FreeBusyURLs[0].URI != "https://example.com/freebusy" {
		t.Errorf("FreeBusyURLs = %+v", rec.Card.FreeBusyURLs)
	}
}

func TestImport_CaladrURIAndKey(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "caladruri-key-example",
		"schedulingAddresses": { "S1": { "@type": "SchedulingAddress", "uri": "mailto:janedoe@example.com" } },
		"cryptoKeys": { "K1": { "@type": "CryptoKey", "uri": "https://example.com/key.pem" } }
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.SchedulingAddresses) != 1 || rec.Card.SchedulingAddresses[0].URI != "mailto:janedoe@example.com" {
		t.Errorf("SchedulingAddresses = %+v", rec.Card.SchedulingAddresses)
	}
	if len(rec.Card.CryptoKeys) != 1 || rec.Card.CryptoKeys[0].URI != "https://example.com/key.pem" {
		t.Errorf("CryptoKeys = %+v", rec.Card.CryptoKeys)
	}
}

func TestImport_DirectoryAndSource(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "directory-source-example",
		"directories": {
			"D1": { "@type": "Directory", "kind": "directory", "uri": "ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering", "listAs": 1 },
			"D2": { "@type": "Directory", "kind": "entry", "uri": "https://example.com/directory/jdoe.vcf" }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	var directoryURI, sourceURI string
	for _, d := range rec.Card.Directories {
		switch d.Kind {
		case "directory":
			directoryURI = d.URI
		case "entry":
			sourceURI = d.URI
		}
	}
	if directoryURI != "ldap://ldap.tech.example/o=Example%20Tech,ou=Engineering" {
		t.Errorf("directory URI = %q", directoryURI)
	}
	if sourceURI != "https://example.com/directory/jdoe.vcf" {
		t.Errorf("source (entry) URI = %q", sourceURI)
	}
}

func TestImport_LinkAndContactURI(t *testing.T) {
	raw := []byte(`{
		"@type": "Card", "version": "1.0", "uid": "link-contacturi-example",
		"links": {
			"L1": { "@type": "Link", "uri": "https://example.com/jdoe" },
			"L2": { "@type": "Link", "kind": "contact", "uri": "mailto:contact@example.com" }
		}
	}`)
	rec, _, err := Adapter{}.Import(raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(rec.Card.Links) != 1 || rec.Card.Links[0].URI != "https://example.com/jdoe" {
		t.Errorf("Links = %+v", rec.Card.Links)
	}
	if len(rec.Card.ContactURIs) != 1 || rec.Card.ContactURIs[0].URI != "mailto:contact@example.com" {
		t.Errorf("ContactURIs = %+v", rec.Card.ContactURIs)
	}
}
