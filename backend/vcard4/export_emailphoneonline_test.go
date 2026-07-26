package vcard4

import (
	"strings"
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("email", "phone", "impp", "social")
}

func TestExport_Email(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Emails: []contactmodel.Email{{Address: "ada@example.com", Contexts: []string{"work"}, Pref: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "EMAIL", map[string]string{"PREF": "1", "TYPE": "work"}, "ada@example.com")
}

func TestExport_Phone(t *testing.T) {
	// 40-testing.md §40.3's worked example: feat2type/pref transforms.
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Phones: []contactmodel.Phone{{Number: "+15551234567", Features: []string{"cell"}, Pref: intPtr(1)}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "TEL", map[string]string{"PREF": "1", "TYPE": "cell"}, "+15551234567")
}

func TestExport_Impp(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		ImppAddresses: []contactmodel.OnlineService{{URI: "xmpp:alice@example.com"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "IMPP", nil, "xmpp:alice@example.com")
}

// TestExport_OnlineServiceServiceTypeUsername covers the bug fix's export
// side: SERVICE-TYPE/USERNAME must not be lost when an OnlineService that
// came from IMPP carries them. Per 20-correspondence.md §20.7's three-array
// design, which array an entry lives in IS the provenance decision — an
// ImppAddresses entry always emits as IMPP (never SOCIALPROFILE), carrying
// its SERVICE-TYPE/USERNAME params (RFC 9554 §4.9/§4.10 allows both on
// either property).
func TestExport_OnlineServiceServiceTypeUsername(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		ImppAddresses: []contactmodel.OnlineService{{
			URI: "xmpp:alice@example.com", Service: "xmpp", User: "alice@example.com",
		}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "IMPP",
		map[string]string{"SERVICE-TYPE": "xmpp", "USERNAME": "alice@example.com"}, "xmpp:alice@example.com")
}

func TestExport_SocialProfile(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SocialProfiles: []contactmodel.OnlineService{{Service: "Mastodon", URI: "https://example.com/@foo", User: "foo"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "SOCIALPROFILE",
		map[string]string{"SERVICE-TYPE": "Mastodon", "USERNAME": "foo"}, "https://example.com/@foo")
}

func TestExport_SocialProfileTextValue(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SocialProfiles: []contactmodel.OnlineService{{Service: "SomeSite", User: "peter94"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, "SOCIALPROFILE",
		map[string]string{"SERVICE-TYPE": "SomeSite", "VALUE": "text"}, "peter94")
}

// TestExport_OtherOnlineServicesWarnDrop covers Card.OtherOnlineServices
// (unclassified online services, e.g. GUI-added or JSContact-imported with
// no vCardName hint): per 20-correspondence.md §20.7, neither IMPP nor
// SOCIALPROFILE is a safe default guess, so these entries are dropped from
// vCard export entirely and reported via a warn Diagnostic.
func TestExport_OtherOnlineServicesWarnDrop(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		OtherOnlineServices: []contactmodel.OnlineService{{URI: "https://example.com/unclassified"}},
	}}
	out, diags, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if strings.Contains(string(out), "example.com/unclassified") {
		t.Errorf("OtherOnlineServices entry was emitted into vCard output, want dropped: %s", out)
	}
	var found bool
	for _, d := range diags {
		if d.Concept == "impp" && d.Severity == "warn" {
			found = true
		}
	}
	if !found {
		t.Errorf("no warn Diagnostic emitted for dropped OtherOnlineServices entry; diags = %+v", diags)
	}
}
