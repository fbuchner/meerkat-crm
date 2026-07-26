package jscontact

import (
	"encoding/json"
	"testing"

	"meerkat/contactmodel"
	"meerkat/internal/rfctest"
)

func init() {
	registerExportCoverage("email", "phone", "impp", "social")
}

func TestExport_Email(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:    "email-example",
		Emails: []contactmodel.Email{{ID: "k1", Address: "alice@example.com"}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/emails/k1/address", "alice@example.com")
}

func TestExport_Phone(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID:    "phone-example",
		Phones: []contactmodel.Phone{{ID: "k1", Number: "+15551234567", Features: []string{"voice"}}},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/phones/k1/number", "+15551234567")
}

func TestExport_IMPP(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "impp-example",
		ImppAddresses: []contactmodel.OnlineService{
			{ID: "OS-1", URI: "xmpp:alice@example.com", Pref: intPtr(1)},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/onlineServices/OS-1/uri", "xmpp:alice@example.com")
	// re-tagged with vCardName="impp" (RFC 9555 §2.7.2/§2.15.3) so a later
	// re-import is fully faithful, not just heuristic.
	rfctest.AssertJSONPointer(t, out, "/onlineServices/OS-1/vCardName", "impp")
}

func TestExport_SocialProfile(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "social-example",
		SocialProfiles: []contactmodel.OnlineService{
			{ID: "OS-1", Service: "Mastodon", User: "alice", URI: "https://example.social/@alice"},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/onlineServices/OS-1/service", "Mastodon")
	rfctest.AssertJSONPointer(t, out, "/onlineServices/OS-1/user", "alice")
	rfctest.AssertJSONPointer(t, out, "/onlineServices/OS-1/vCardName", "socialprofile")
}

func TestExport_OtherOnlineServiceNoVCardNameTag(t *testing.T) {
	// OtherOnlineServices entries have genuinely unknown origin, so export
	// must NOT fabricate a vCardName tag (20-correspondence.md §20.7).
	rec := &contactmodel.Record{Card: contactmodel.Card{
		UID: "unclassified-example",
		OtherOnlineServices: []contactmodel.OnlineService{
			{ID: "OS-1", Service: "Mastodon", User: "alice", URI: "https://example.social/@alice"},
		},
	}}
	out, _, err := Adapter{}.Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	rfctest.AssertJSONPointer(t, out, "/onlineServices/OS-1/service", "Mastodon")

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	os1 := doc["onlineServices"].(map[string]any)["OS-1"].(map[string]any)
	if _, present := os1["vCardName"]; present {
		t.Errorf("expected no vCardName key for an OtherOnlineServices entry, got %+v", os1)
	}
}
