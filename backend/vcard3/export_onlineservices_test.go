package vcard3

import (
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

// Concepts covered: impp, social.
func init() {
	registerExportCoverage("impp", "social")
}

func TestExport_Impp(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		ImppAddresses: []contactmodel.OnlineService{{URI: "xmpp:frank@example.com", Contexts: []string{"work"}}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropImpp, map[string]string{"TYPE": "WORK"}, "xmpp:frank@example.com")
}

// RFC 9554 SS4.9/4.10: SERVICE-TYPE/USERNAME "MAY be specified on an IMPP or
// a SOCIALPROFILE property". This adapter's vCard-3.0 emulation of those
// params emits grouped X-SERVICE-TYPE/X-USERNAME companion properties sharing
// IMPP's GROUP token (mirroring the existing X-SOCIALPROFILE/X-SERVICE-TYPE
// convention below).
func TestExport_ImppWithServiceAndUsername(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		ImppAddresses: []contactmodel.OnlineService{{
			URI: "xmpp:frank@example.com", Service: "Jabber", User: "frank94", Contexts: []string{"work"},
		}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropImpp, map[string]string{"TYPE": "WORK"}, "xmpp:frank@example.com")
	rfctest.AssertVCardLine(t, out, PropXServiceType, nil, "Jabber")
	rfctest.AssertVCardLine(t, out, PropXUsername, nil, "frank94")
}

func TestExport_Social(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SocialProfiles: []contactmodel.OnlineService{{
			URI: "https://example.com/@frank", Service: "Mastodon", Contexts: []string{"work"},
		}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropXSocialProfile, map[string]string{"TYPE": "WORK"}, "https://example.com/@frank")
	rfctest.AssertVCardLine(t, out, PropXServiceType, nil, "Mastodon")
}

// Mirrors TestExport_ImppWithServiceAndUsername: SOCIALPROFILE's User field
// must round-trip through a grouped X-USERNAME companion property too
// (previously never emitted at all for SOCIALPROFILE).
func TestExport_SocialWithUsername(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SocialProfiles: []contactmodel.OnlineService{{
			URI: "https://example.com/@frank", Service: "Mastodon", User: "frank94", Contexts: []string{"work"},
		}},
	}}
	out, _, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertVCardLine(t, out, PropXSocialProfile, map[string]string{"TYPE": "WORK"}, "https://example.com/@frank")
	rfctest.AssertVCardLine(t, out, PropXServiceType, nil, "Mastodon")
	rfctest.AssertVCardLine(t, out, PropXUsername, nil, "frank94")
}
