package contactmodel

import (
	"encoding/json"
	"reflect"
	"testing"
)

// ptr returns a pointer to a copy of v — used to populate the *int/*bool
// optional-pointer fields throughout the neutral model with non-zero values.
func ptr[T any](v T) *T {
	return &v
}

// assertRoundTrip marshals original to JSON, unmarshals into a fresh zero
// value of the same type, and asserts the result is deeply equal to
// original. This is the "constructor + round-trip test" WP-10 acceptance
// criterion for every exported struct type.
func assertRoundTrip[T any](t *testing.T, original T) {
	t.Helper()

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var got T
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v\njson: %s", err, data)
	}

	if !reflect.DeepEqual(original, got) {
		t.Errorf("round-trip mismatch\n original: %#v\n got:      %#v\n json:     %s", original, got, data)
	}
}

// ---- fully-populated builder functions, one per sub-object type ----
// Every field (including nested slices/pointers) is set to a non-zero value.

func fullNameComponent() NameComponent {
	return NameComponent{
		Kind:     "given",
		Value:    "Ada",
		Phonetic: "EY-da",
	}
}

func fullName() *Name {
	return &Name{
		Components:       []NameComponent{fullNameComponent(), {Kind: "surname", Value: "Lovelace", Phonetic: "LUV-lace"}},
		Full:             "Ada Lovelace",
		SortAs:           map[string]string{"surname": "Lovelace"},
		IsOrdered:        ptr(true),
		DefaultSeparator: " ",
		PhoneticSystem:   "ipa",
		PhoneticScript:   "Latn",
	}
}

func fullNickname() Nickname {
	return Nickname{
		ID:       "nick-1",
		Name:     "Ada",
		Contexts: []string{"private", "work"},
		Pref:     ptr(1),
	}
}

func fullOrgUnit() OrgUnit {
	return OrgUnit{
		Name:   "R&D",
		SortAs: "RD",
	}
}

func fullOrganization() Organization {
	return Organization{
		ID:     "org-1",
		Name:   "Analytical Engines Ltd",
		Units:  []OrgUnit{fullOrgUnit()},
		SortAs: "Analytical",
	}
}

func fullTitle() Title {
	return Title{
		ID:             "title-1",
		Name:           "Chief Mathematician",
		Kind:           "title",
		OrganizationID: "org-1",
	}
}

func fullEmail() Email {
	return Email{
		ID:       "email-1",
		Address:  "ada@example.com",
		Contexts: []string{"work"},
		Pref:     ptr(1),
		Label:    "Work Email",
	}
}

func fullPhone() Phone {
	return Phone{
		ID:       "phone-1",
		Number:   "+15551234567",
		Features: []string{"voice", "cell"},
		Contexts: []string{"private"},
		Pref:     ptr(1),
		Label:    "Mobile",
	}
}

func fullOnlineService() OnlineService {
	return OnlineService{
		ID:       "svc-1",
		Service:  "Mastodon",
		URI:      "https://example.social/@ada",
		User:     "@ada",
		Contexts: []string{"work"},
		Pref:     ptr(1),
		Label:    "Mastodon",
	}
}

func fullAddressComponent() AddressComponent {
	return AddressComponent{
		Kind:     "locality",
		Value:    "London",
		Phonetic: "LUN-dun",
	}
}

func fullAddress() *Address {
	return &Address{
		ID:               "addr-1",
		Components:       []AddressComponent{fullAddressComponent(), {Kind: "country", Value: "UK"}},
		CountryCode:      "GB",
		Coordinates:      "geo:51.5074,-0.1278",
		TimeZone:         "Europe/London",
		Contexts:         []string{"work"},
		Pref:             ptr(1),
		Full:             "1 Analytical Way, London, UK",
		IsOrdered:        ptr(true),
		DefaultSeparator: ",",
		PhoneticSystem:   "ipa",
		PhoneticScript:   "Latn",
	}
}

func fullPartialDate() *PartialDate {
	return &PartialDate{
		Year:          ptr(1815),
		Month:         ptr(12),
		Day:           ptr(10),
		CalendarScale: "gregorian",
	}
}

func fullAnniversaryDate() AnniversaryDate {
	return AnniversaryDate{
		Partial:   fullPartialDate(),
		Timestamp: ptr("1815-12-10T00:00:00Z"),
	}
}

func fullAnniversary() Anniversary {
	return Anniversary{
		ID:    "anniv-1",
		Kind:  "birth",
		Date:  fullAnniversaryDate(),
		Place: fullAddress(),
	}
}

func fullPronouns() Pronouns {
	return Pronouns{
		ID:       "pron-1",
		Pronouns: "she/her",
		Contexts: []string{"private"},
		Pref:     ptr(1),
	}
}

func fullGrammaticalGender() GrammaticalGender {
	return GrammaticalGender{
		ID:       "gg-1",
		Value:    "feminine",
		Language: "en",
	}
}

func fullSpeakToAs() *SpeakToAs {
	return &SpeakToAs{
		GrammaticalGenders: []GrammaticalGender{fullGrammaticalGender()},
		Pronouns:           []Pronouns{fullPronouns()},
	}
}

func fullPersonalInfo() PersonalInfo {
	return PersonalInfo{
		ID:     "pi-1",
		Kind:   "hobby",
		Value:  "chess",
		Level:  "high",
		ListAs: ptr(1),
		Label:  "Hobby",
	}
}

func fullAuthor() *Author {
	return &Author{
		Name: "Charles Babbage",
		URI:  "https://example.com/babbage",
	}
}

func fullTimestamp() *Timestamp {
	return &Timestamp{UTC: "1843-01-01T00:00:00Z"}
}

func fullNote() Note {
	return Note{
		ID:      "note-1",
		Note:    "Met at the Analytical Society",
		Author:  fullAuthor(),
		Created: fullTimestamp(),
	}
}

func fullResource() Resource {
	return Resource{
		ID:        "res-1",
		Kind:      "photo",
		URI:       "https://example.com/photo.jpg",
		MediaType: "image/jpeg",
		Label:     "Portrait",
		Contexts:  []string{"work"},
		Pref:      ptr(1),
		ListAs:    ptr(1),
	}
}

func fullLanguagePref() LanguagePref {
	return LanguagePref{
		ID:       "lang-1",
		Language: "en-GB",
		Contexts: []string{"work"},
		Pref:     ptr(1),
	}
}

func fullRelation() Relation {
	return Relation{
		Target:    "urn:uuid:1234",
		Relations: []string{"friend", "colleague"},
	}
}

func fullCard() Card {
	return Card{
		UID:      "card-uid-1",
		Kind:     "individual",
		Language: "en",
		ProdID:   "-//Mycorrhizal//RFC9553//EN",
		Created:  fullTimestamp(),
		Updated:  fullTimestamp(),

		Name:                fullName(),
		Nicknames:           []Nickname{fullNickname()},
		Organizations:       []Organization{fullOrganization()},
		Titles:              []Title{fullTitle()},
		Emails:              []Email{fullEmail()},
		Phones:              []Phone{fullPhone()},
		ImppAddresses:       []OnlineService{fullOnlineService()},
		SocialProfiles:      []OnlineService{{ID: "svc-2", Service: "Mastodon", URI: "https://example.social/@ada2", User: "@ada2", Contexts: []string{"private"}, Pref: ptr(2), Label: "Mastodon (social)"}},
		OtherOnlineServices: []OnlineService{{ID: "svc-3", Service: "Unknown", URI: "https://example.com/@ada3", User: "@ada3", Contexts: []string{"work"}, Pref: ptr(3), Label: "Unclassified"}},
		Addresses:           []Address{*fullAddress()},
		Anniversaries:       []Anniversary{fullAnniversary()},
		SpeakToAs:           fullSpeakToAs(),
		PersonalInfo:        []PersonalInfo{fullPersonalInfo()},
		Notes:               []Note{fullNote()},
		Keywords:            []string{"friend", "colleague"},

		Media:               []Resource{fullResource()},
		Calendars:           []Resource{{ID: "cal-1", URI: "https://example.com/cal", Pref: ptr(1)}},
		FreeBusyURLs:        []Resource{{ID: "fb-1", URI: "https://example.com/freebusy", Pref: ptr(1)}},
		SchedulingAddresses: []Resource{{ID: "sched-1", URI: "mailto:ada@example.com"}},
		CryptoKeys:          []Resource{{ID: "key-1", URI: "https://example.com/key.pem"}},
		Directories:         []Resource{{ID: "dir-1", Kind: "directory", URI: "https://example.com/dir"}},
		Links:               []Resource{{ID: "link-1", URI: "https://example.com/ada"}},
		ContactURIs:         []Resource{{ID: "contact-uri-1", URI: "mailto:ada@example.com"}},

		PreferredLanguages: []LanguagePref{fullLanguagePref()},
		RelatedTo:          []Relation{fullRelation()},
		Members:            []string{"urn:uuid:member-1"},

		Localizations: map[string]json.RawMessage{"fr": json.RawMessage(`{"name":{"full":"Ada Lovelace"}}`)},
	}
}

func fullCRMEnvelope() CRMEnvelope {
	return CRMEnvelope{
		Circles:            []string{"friends", "family"},
		HowWeMet:           "University",
		WorkInformation:    "Mathematician",
		ContactInformation: "Prefers email",
		CustomFields:       map[string]string{"favorite_color": "blue"},
	}
}

func fullJCardProp() JCardProp {
	return JCardProp{
		Name:   "x-custom",
		Params: map[string]any{"type": "home"},
		Type:   "text",
		Value:  json.RawMessage(`"custom value"`),
	}
}

func fullPassthrough() Passthrough {
	return Passthrough{
		VCard:     []JCardProp{fullJCardProp()},
		JSContact: map[string]json.RawMessage{"/custom": json.RawMessage(`{"foo":"bar"}`)},
	}
}

func fullRecord() Record {
	return Record{
		Card:        fullCard(),
		Envelope:    fullCRMEnvelope(),
		Passthrough: fullPassthrough(),
		UID:         "record-uid-1",
		ETag:        "etag-1",
	}
}

func fullDiagnostic() Diagnostic {
	return Diagnostic{
		Severity: "warn",
		Concept:  "email.address",
		Message:  "unmappable value",
	}
}

func fullProjection() Projection {
	return Projection{
		Firstname:    "Ada",
		Lastname:     "Lovelace",
		FN:           "Ada Lovelace",
		PrimaryEmail: "ada@example.com",
		PrimaryPhone: "+15551234567",
		Birthday:     "1815-12-10",
		Org:          "Analytical Engines Ltd",
	}
}
