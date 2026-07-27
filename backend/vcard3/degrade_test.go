package vcard3

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	vcard "github.com/emersion/go-vcard"

	"mycorrhizal/contactmodel"
)

// degrade_test.go asserts, for every docs/fork-plan/20-correspondence.md
// §20.6 "no-3.0-home" concept, that exporting a Record carrying that concept
// (a) appends a contactmodel.Diagnostic{Severity:"warn", Concept: <id>},
// (b) the concept's value is left untouched on the source Record (obviously
// true here since Export never mutates its input, but asserted explicitly
// for the representative cases below), and (c) the field is correctly
// absent from the vCard 3.0 output — except the two concepts 20.6 itself
// documents as being redirected onto a real 3.0 property instead of purely
// dropped (anniversary.wedding -> X-ANNIVERSARY; related -> AGENT), where the
// warning still fires but the value legitimately appears via that
// escape-hatch property. See docs/fork-plan/60-review-gates.md §60.3.
//
// Beyond the 17 concepts 20.6 names directly, this file also asserts three
// extra cases that are not spelled out in 20.6's prose summary but are
// unambiguous "no-3.0-home" rows in the table itself (V3Prop == "-", and,
// unlike adr.geo/adr.tz, with no alternate v3 property to redirect to):
// anniversary.place.birth, anniversary.place.death (RFC 6474 is 4.0-only),
// and pt.jscontact (whose own row notes literally say "3.0 warn-drop"). This
// is a documented judgment call — see the final WP-50 report.

func hasWarn(diags []contactmodel.Diagnostic, concept string) bool {
	for _, d := range diags {
		if d.Concept == concept && d.Severity == "warn" {
			return true
		}
	}
	return false
}

func decodedProps(t *testing.T, out []byte) vcard.Card {
	t.Helper()
	card, err := vcard.NewDecoder(bytes.NewReader(out)).Decode()
	if err != nil {
		t.Fatalf("decode exported output: %v", err)
	}
	return card
}

func hasProp(t *testing.T, out []byte, prop string) bool {
	t.Helper()
	return len(decodedProps(t, out)[strings.ToUpper(prop)]) > 0
}

func TestDegrade_Kind(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Kind: "group"}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "kind") {
		t.Errorf("diags = %+v, want a warn for concept kind", diags)
	}
	if hasProp(t, out, "KIND") {
		t.Errorf("KIND unexpectedly present in output")
	}
	if rec.Card.Kind != "group" {
		t.Errorf("source Record.Card.Kind mutated: %q", rec.Card.Kind)
	}
}

func TestDegrade_Created(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Created: &contactmodel.Timestamp{UTC: "2020-01-01T00:00:00Z"}}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "created") {
		t.Errorf("diags = %+v, want a warn for concept created", diags)
	}
	if hasProp(t, out, "CREATED") {
		t.Errorf("CREATED unexpectedly present in output")
	}
}

// TestDegrade_Language covers the `language` concept: per
// docs/fork-plan/20-correspondence.md's corrected language row and §20.6
// (now explicitly listing `language`), P0 scope is a plain warn-drop like any
// other no-3.0-home concept — the richer "apply as default LANGUAGE param on
// other properties" behavior (RFC 9555) is deferred post-P0, and never loses
// data since Card.Language remains on the neutral Record regardless of
// serialization.
func TestDegrade_Language(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Language: "de-AT"}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "language") {
		t.Errorf("diags = %+v, want a warn for concept language", diags)
	}
	if hasProp(t, out, "LANGUAGE") {
		t.Errorf("LANGUAGE unexpectedly present as its own property in output")
	}
	if strings.Contains(string(out), "de-AT") {
		t.Errorf("language value unexpectedly leaked into output:\n%s", out)
	}
	if rec.Card.Language != "de-AT" {
		t.Errorf("source Record.Card.Language mutated: %q", rec.Card.Language)
	}
}

func TestDegrade_AnniversaryWedding_RedirectsToXAnniversary(t *testing.T) {
	year, month, day := 2001, 6, 10
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{
			Kind: "wedding",
			Date: contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Year: &year, Month: &month, Day: &day}},
		}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "anniversary.wedding") {
		t.Errorf("diags = %+v, want a warn for concept anniversary.wedding", diags)
	}
	// Documented exception: the value legitimately survives via X-ANNIVERSARY.
	if !hasProp(t, out, PropXAnniversary) {
		t.Errorf("X-ANNIVERSARY missing from output; wedding anniversary should be redirected there")
	}
	if hasProp(t, out, "ANNIVERSARY") {
		t.Errorf("standard ANNIVERSARY (4.0-only) property unexpectedly present")
	}
}

func TestDegrade_AnniversaryDeath(t *testing.T) {
	year := 2010
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{
			Kind: "death",
			Date: contactmodel.AnniversaryDate{Partial: &contactmodel.PartialDate{Year: &year}},
		}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "anniversary.death") {
		t.Errorf("diags = %+v, want a warn for concept anniversary.death", diags)
	}
	if hasProp(t, out, "DEATHDATE") {
		t.Errorf("DEATHDATE (4.0-only) unexpectedly present in output")
	}
}

func TestDegrade_AnniversaryPlaceBirth(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{
			Kind:  "birth",
			Date:  contactmodel.AnniversaryDate{},
			Place: &contactmodel.Address{Full: "Somewhere, USA"},
		}},
	}}
	_, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "anniversary.place.birth") {
		t.Errorf("diags = %+v, want a warn for concept anniversary.place.birth", diags)
	}
}

func TestDegrade_AnniversaryPlaceDeath(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Anniversaries: []contactmodel.Anniversary{{
			Kind:  "death",
			Date:  contactmodel.AnniversaryDate{},
			Place: &contactmodel.Address{Full: "Somewhere, USA"},
		}},
	}}
	_, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "anniversary.place.death") {
		t.Errorf("diags = %+v, want a warn for concept anniversary.place.death", diags)
	}
}

func TestDegrade_GramGender(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SpeakToAs: &contactmodel.SpeakToAs{GrammaticalGenders: []contactmodel.GrammaticalGender{{Value: "feminine"}}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "gramgender") {
		t.Errorf("diags = %+v, want a warn for concept gramgender", diags)
	}
	if hasProp(t, out, "GRAMGENDER") {
		t.Errorf("GRAMGENDER (4.0-only) unexpectedly present in output")
	}
}

func TestDegrade_Pronouns(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		SpeakToAs: &contactmodel.SpeakToAs{Pronouns: []contactmodel.Pronouns{{Pronouns: "they/them"}}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "pronouns") {
		t.Errorf("diags = %+v, want a warn for concept pronouns", diags)
	}
	if hasProp(t, out, "PRONOUNS") {
		t.Errorf("PRONOUNS (4.0-only) unexpectedly present in output")
	}
}

func TestDegrade_Expertise(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		PersonalInfo: []contactmodel.PersonalInfo{{Kind: "expertise", Value: "Go programming"}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "expertise") {
		t.Errorf("diags = %+v, want a warn for concept expertise", diags)
	}
	if hasProp(t, out, "EXPERTISE") {
		t.Errorf("EXPERTISE (4.0-only) unexpectedly present in output")
	}
}

func TestDegrade_Hobby(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		PersonalInfo: []contactmodel.PersonalInfo{{Kind: "hobby", Value: "Chess"}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "hobby") {
		t.Errorf("diags = %+v, want a warn for concept hobby", diags)
	}
	if hasProp(t, out, "HOBBY") {
		t.Errorf("HOBBY (4.0-only) unexpectedly present in output")
	}
}

func TestDegrade_Interest(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		PersonalInfo: []contactmodel.PersonalInfo{{Kind: "interest", Value: "Hiking"}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "interest") {
		t.Errorf("diags = %+v, want a warn for concept interest", diags)
	}
	if hasProp(t, out, "INTEREST") {
		t.Errorf("INTEREST (4.0-only) unexpectedly present in output")
	}
}

func TestDegrade_Directory(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Directories: []contactmodel.Resource{{Kind: "directory", URI: "https://dir.example.com/frank"}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "directory") {
		t.Errorf("diags = %+v, want a warn for concept directory", diags)
	}
	if hasProp(t, out, PropSource) {
		t.Errorf("directory kind unexpectedly leaked into SOURCE")
	}
}

func TestDegrade_ContactURI(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		ContactURIs: []contactmodel.Resource{{URI: "https://contact.example.com/frank"}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "contacturi") {
		t.Errorf("diags = %+v, want a warn for concept contacturi", diags)
	}
	if hasProp(t, out, PropURL) {
		t.Errorf("ContactURIs entry unexpectedly leaked into URL")
	}
	if len(rec.Card.ContactURIs) != 1 {
		t.Errorf("source Record.Card.ContactURIs mutated: %+v", rec.Card.ContactURIs)
	}
}

// TestDegrade_OtherOnlineServices covers Card.OtherOnlineServices (20.7): it
// has no vCard export at all, in either 3.0 or 4.0, because neither IMPP nor
// SOCIALPROFILE is a safe default guess for genuinely unclassified data.
func TestDegrade_OtherOnlineServices(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		OtherOnlineServices: []contactmodel.OnlineService{{URI: "https://example.com/@frank", Service: "Unknown"}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "onlineservice.other") {
		t.Errorf("diags = %+v, want a warn for concept onlineservice.other", diags)
	}
	if hasProp(t, out, PropImpp) {
		t.Errorf("IMPP unexpectedly present in output")
	}
	if hasProp(t, out, PropXSocialProfile) {
		t.Errorf("X-SOCIALPROFILE unexpectedly present in output")
	}
	if strings.Contains(string(out), "example.com/@frank") {
		t.Errorf("OtherOnlineServices value unexpectedly leaked into output:\n%s", out)
	}
	if len(rec.Card.OtherOnlineServices) != 1 {
		t.Errorf("source Record.Card.OtherOnlineServices mutated: %+v", rec.Card.OtherOnlineServices)
	}
}

func TestDegrade_Lang(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		PreferredLanguages: []contactmodel.LanguagePref{{Language: "en"}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "lang") {
		t.Errorf("diags = %+v, want a warn for concept lang", diags)
	}
	if hasProp(t, out, "LANG") {
		t.Errorf("LANG (4.0-only) unexpectedly present in output")
	}
}

func TestDegrade_Related_RedirectsToAgent(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		RelatedTo: []contactmodel.Relation{{Target: "urn:uuid:1234", Relations: []string{"friend"}}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "related") {
		t.Errorf("diags = %+v, want a warn for concept related", diags)
	}
	// Documented exception: the target legitimately survives via AGENT.
	if !hasProp(t, out, PropAgent) {
		t.Errorf("AGENT missing from output; related target should be redirected there")
	}
	if hasProp(t, out, "RELATED") {
		t.Errorf("standard RELATED (4.0-only) property unexpectedly present")
	}
}

func TestDegrade_Member(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{Members: []string{"urn:uuid:5678"}}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "member") {
		t.Errorf("diags = %+v, want a warn for concept member", diags)
	}
	if hasProp(t, out, "MEMBER") {
		t.Errorf("MEMBER (4.0-only) unexpectedly present in output")
	}
}

func TestDegrade_NameSurname2(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "surname2", Value: "Garcia"}}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "name.surname2") {
		t.Errorf("diags = %+v, want a warn for concept name.surname2", diags)
	}
	if strings.Contains(string(out), "Garcia") {
		t.Errorf("surname2 value unexpectedly leaked into output:\n%s", out)
	}
}

func TestDegrade_NameGeneration(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "generation", Value: "III"}}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "name.generation") {
		t.Errorf("diags = %+v, want a warn for concept name.generation", diags)
	}
	if strings.Contains(string(out), "III") {
		t.Errorf("generation value unexpectedly leaked into output:\n%s", out)
	}
}

func TestDegrade_NamePhonetic(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Name: &contactmodel.Name{Full: "x", PhoneticScript: "jat6sin1"},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "name.phonetic") {
		t.Errorf("diags = %+v, want a warn for concept name.phonetic", diags)
	}
	if strings.Contains(string(out), "jat6sin1") {
		t.Errorf("phonetic script value unexpectedly leaked into output:\n%s", out)
	}
}

func TestDegrade_AdrExtraComponentKind(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Addresses: []contactmodel.Address{{
			Components: []contactmodel.AddressComponent{
				{Kind: "name", Value: "Baker Street"},
				{Kind: "room", Value: "221B"},
			},
		}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "adr") {
		t.Errorf("diags = %+v, want a warn for concept adr", diags)
	}
	if strings.Contains(string(out), "221B") {
		t.Errorf("extra ADR component kind value unexpectedly leaked into output:\n%s", out)
	}
}

func TestDegrade_NoteParams(t *testing.T) {
	rec := &contactmodel.Record{Card: contactmodel.Card{
		Notes: []contactmodel.Note{{Note: "hi", Author: &contactmodel.Author{Name: "Bob"}}},
	}}
	out, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "note") {
		t.Errorf("diags = %+v, want a warn for concept note", diags)
	}
	rfc := decodedProps(t, out)
	fields := rfc[PropNote]
	if len(fields) != 1 || fields[0].Value != "hi" {
		t.Errorf("NOTE fields = %+v, want a single NOTE:hi", fields)
	}
	if len(fields) == 1 && len(fields[0].Params) != 0 {
		t.Errorf("NOTE params = %+v, want none (AUTHOR/CREATED have no 3.0 home)", fields[0].Params)
	}
}

func TestDegrade_PassthroughJSContact(t *testing.T) {
	rec := &contactmodel.Record{Passthrough: contactmodel.Passthrough{
		JSContact: map[string]json.RawMessage{"/foo": json.RawMessage(`"bar"`)},
	}}
	_, diags, err := (Adapter{}).Export(rec)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !hasWarn(diags, "pt.jscontact") {
		t.Errorf("diags = %+v, want a warn for concept pt.jscontact", diags)
	}
}
