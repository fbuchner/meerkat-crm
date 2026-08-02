package models

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"strings"
	"testing"

	"mycorrhizal/contactmodel"
)

// testJPEGDataURL returns a "data:image/jpeg;base64,..." URI wrapping a
// minimal, valid 2x2 JPEG — usable wherever a test needs
// Contact.PhotoThumbnail (or any decodable photo payload, e.g. for
// photostore.SaveContactPhoto) without caring about actual pixel content.
func testJPEGDataURL() string {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// fullyPopulatedContact builds a *Contact with every field RecordFromContact
// maps populated, so tests can assert each lands in its correct neutral
// home per docs/fork-plan/20-correspondence.md.
func fullyPopulatedContact() *Contact {
	return &Contact{
		Firstname:   "Jane",
		Lastname:    "Doe",
		MiddleName:  "Quinn",
		Prefix:      "Dr.",
		Suffix:      "Jr.",
		Nickname:    "Janie",
		Gender:      "female",
		Email:       "jane.legacy@example.com", // overridden by Emails[0] below
		Phone:       "+10000000000",            // overridden by Phones[0] below
		Birthday:    "1990-06-15",
		Anniversary: "--09-20",
		Emails: []ContactEmail{
			{Type: "work", Value: "jane@example.com"},
		},
		Phones: []ContactPhone{
			{Type: "cell", Value: "+15551234567"},
		},
		Addresses: []ContactAddress{
			{Type: "home", Street: "1 Main St", City: "Springfield", Region: "IL", Postal: "62704", Country: "US"},
		},
		URLs: []ContactURL{
			{Type: "personal", Value: "https://jane.example.com"},
		},
		IMPPs: []ContactIMPP{
			{Type: "telegram", Value: "xmpp:jane@example.com"},
		},
		Organization:       "Acme Corp",
		Department:         "Engineering",
		JobTitle:           "Staff Engineer",
		Role:               "Tech Lead",
		HowWeMet:           "Conference",
		WorkInformation:    "Remote",
		ContactInformation: "Prefers email",
		Circles:            []string{"friends", "work"},
		CustomFields:       map[string]string{"favorite_color": "blue"},
		VCardUID:           "uid-1234",
		VCardExtra:         `{"properties":{"X-CUSTOM":[{"Value":"keep","Params":{"TYPE":["home"]},"Group":""}]}}`,
		PhotoThumbnail:     testJPEGDataURL(),
	}
}

// TestRecordFromContact_FullyPopulated asserts every mapped Contact field
// lands in the neutral home documented by 20-correspondence.md.
func TestRecordFromContact_FullyPopulated(t *testing.T) {
	c := fullyPopulatedContact()
	record := RecordFromContact(c, "")
	card := record.Card

	// "uid" row: Card.UID <- VCardUID
	if card.UID != "uid-1234" {
		t.Errorf("Card.UID = %q, want %q", card.UID, "uid-1234")
	}
	if record.UID != "uid-1234" {
		t.Errorf("Record.UID = %q, want %q", record.UID, "uid-1234")
	}

	// "name.given"/"name.surname"/"name.given2"/"name.title"/"name.credential" rows
	if card.Name == nil {
		t.Fatal("Card.Name is nil, want populated")
	}
	wantComponents := map[string]string{
		"given":      "Jane",
		"surname":    "Doe",
		"given2":     "Quinn",
		"title":      "Dr.",
		"credential": "Jr.",
	}
	gotComponents := map[string]string{}
	for _, comp := range card.Name.Components {
		gotComponents[comp.Kind] = comp.Value
	}
	for kind, want := range wantComponents {
		if got := gotComponents[kind]; got != want {
			t.Errorf("Card.Name.Components[kind=%s] = %q, want %q", kind, got, want)
		}
	}

	// "nickname" row: Card.Nicknames[].Name
	if len(card.Nicknames) != 1 || card.Nicknames[0].Name != "Janie" {
		t.Errorf("Card.Nicknames = %+v, want single entry Name=Janie", card.Nicknames)
	}

	// "org"/"org.unit" rows: Card.Organizations[].Name / .Units[].Name
	if len(card.Organizations) != 1 {
		t.Fatalf("Card.Organizations = %+v, want 1 entry", card.Organizations)
	}
	if card.Organizations[0].Name != "Acme Corp" {
		t.Errorf("Card.Organizations[0].Name = %q, want Acme Corp", card.Organizations[0].Name)
	}
	if len(card.Organizations[0].Units) != 1 || card.Organizations[0].Units[0].Name != "Engineering" {
		t.Errorf("Card.Organizations[0].Units = %+v, want single unit Engineering", card.Organizations[0].Units)
	}

	// "title"/"role" rows: Card.Titles[kind=title/role].Name
	var gotTitle, gotRole string
	for _, ti := range card.Titles {
		switch ti.Kind {
		case "title":
			gotTitle = ti.Name
		case "role":
			gotRole = ti.Name
		}
	}
	if gotTitle != "Staff Engineer" {
		t.Errorf("Card.Titles[kind=title].Name = %q, want Staff Engineer", gotTitle)
	}
	if gotRole != "Tech Lead" {
		t.Errorf("Card.Titles[kind=role].Name = %q, want Tech Lead", gotRole)
	}

	// "email" row: Card.Emails[].Address (Emails[] takes precedence over the
	// legacy Email scalar, which is only a fallback for scalar-only contacts)
	if len(card.Emails) != 1 || card.Emails[0].Address != "jane@example.com" || card.Emails[0].Label != "work" {
		t.Errorf("Card.Emails = %+v, want [{Address:jane@example.com Label:work}]", card.Emails)
	}

	// "phone" row: Card.Phones[].Number
	if len(card.Phones) != 1 || card.Phones[0].Number != "+15551234567" || card.Phones[0].Label != "cell" {
		t.Errorf("Card.Phones = %+v, want [{Number:+15551234567 Label:cell}]", card.Phones)
	}

	// "impp" row: Card.ImppAddresses[].URI ; legacy Type -> Service, per this
	// WP's explicit instruction (Type holds a service name, not a category)
	if len(card.ImppAddresses) != 1 || card.ImppAddresses[0].Service != "telegram" || card.ImppAddresses[0].URI != "xmpp:jane@example.com" {
		t.Errorf("Card.ImppAddresses = %+v, want [{Service:telegram URI:xmpp:jane@example.com}]", card.ImppAddresses)
	}

	// "adr" row: Card.Addresses[]
	if len(card.Addresses) != 1 {
		t.Fatalf("Card.Addresses = %+v, want 1 entry", card.Addresses)
	}
	addr := card.Addresses[0]
	wantAddrComponents := map[string]string{
		"name":     "1 Main St",
		"locality": "Springfield",
		"region":   "IL",
		"postcode": "62704",
		"country":  "US",
	}
	gotAddrComponents := map[string]string{}
	for _, comp := range addr.Components {
		gotAddrComponents[comp.Kind] = comp.Value
	}
	for kind, want := range wantAddrComponents {
		if got := gotAddrComponents[kind]; got != want {
			t.Errorf("Card.Addresses[0].Components[kind=%s] = %q, want %q", kind, got, want)
		}
	}
	if addr.Full == "" {
		t.Error("Card.Addresses[0].Full is empty, want the formatted address line")
	}
	if len(addr.Contexts) != 1 || addr.Contexts[0] != "home" {
		t.Errorf("Card.Addresses[0].Contexts = %+v, want [home]", addr.Contexts)
	}

	// "link" row: Card.Links[].URI
	if len(card.Links) != 1 || card.Links[0].URI != "https://jane.example.com" || card.Links[0].Label != "personal" {
		t.Errorf("Card.Links = %+v, want [{URI:https://jane.example.com Label:personal}]", card.Links)
	}

	// "anniversary.birth"/"anniversary.wedding" rows: Card.Anniversaries[kind=X].Date
	var birth, wedding *contactmodel.Anniversary
	for i := range card.Anniversaries {
		switch card.Anniversaries[i].Kind {
		case "birth":
			birth = &card.Anniversaries[i]
		case "wedding":
			wedding = &card.Anniversaries[i]
		}
	}
	if birth == nil || birth.Date.Partial == nil {
		t.Fatal("Card.Anniversaries[kind=birth] missing or has no partial date")
	}
	if birth.Date.Partial.Year == nil || *birth.Date.Partial.Year != 1990 ||
		birth.Date.Partial.Month == nil || *birth.Date.Partial.Month != 6 ||
		birth.Date.Partial.Day == nil || *birth.Date.Partial.Day != 15 {
		t.Errorf("Card.Anniversaries[kind=birth].Date.Partial = %+v, want 1990-06-15", birth.Date.Partial)
	}
	if wedding == nil || wedding.Date.Partial == nil {
		t.Fatal("Card.Anniversaries[kind=wedding] missing or has no partial date")
	}
	if wedding.Date.Partial.Year != nil {
		t.Errorf("Card.Anniversaries[kind=wedding].Date.Partial.Year = %v, want nil (year-less --MM-DD)", *wedding.Date.Partial.Year)
	}
	if wedding.Date.Partial.Month == nil || *wedding.Date.Partial.Month != 9 ||
		wedding.Date.Partial.Day == nil || *wedding.Date.Partial.Day != 20 {
		t.Errorf("Card.Anniversaries[kind=wedding].Date.Partial = %+v, want --09-20", wedding.Date.Partial)
	}

	// CRM-only fields: 1:1 direct copy into Record.Envelope
	env := record.Envelope
	if env.HowWeMet != "Conference" ||
		env.WorkInformation != "Remote" || env.ContactInformation != "Prefers email" {
		t.Errorf("Record.Envelope text fields = %+v, want the CustomFields set on the Contact", env)
	}
	if len(env.Circles) != 2 || env.Circles[0] != "friends" || env.Circles[1] != "work" {
		t.Errorf("Record.Envelope.Circles = %+v, want [friends work]", env.Circles)
	}
	if env.CustomFields["favorite_color"] != "blue" {
		t.Errorf("Record.Envelope.CustomFields = %+v, want favorite_color=blue", env.CustomFields)
	}

	// WP-73 photo-bridging prerequisite: Contact.PhotoThumbnail (there is no
	// Contact.Photo/on-disk file in this test, so ReadContactPhoto falls back
	// to the thumbnail) bridges into a single Card.Media{Kind:"photo"} entry,
	// encoded as the same "data:<mediaType>;base64,<data>" URI convention the
	// vcard4/vcard3 adapters already use for an embedded PHOTO value.
	if len(card.Media) != 1 {
		t.Fatalf("Card.Media = %+v, want 1 photo entry bridged from PhotoThumbnail", card.Media)
	}
	photo := card.Media[0]
	if photo.Kind != "photo" {
		t.Errorf("Card.Media[0].Kind = %q, want photo", photo.Kind)
	}
	if photo.MediaType != "image/jpeg" {
		t.Errorf("Card.Media[0].MediaType = %q, want image/jpeg", photo.MediaType)
	}
	if !strings.HasPrefix(photo.URI, "data:image/jpeg;base64,") {
		t.Errorf("Card.Media[0].URI = %q, want a data:image/jpeg;base64,... URI", photo.URI)
	}

	// "pt.vcard" row (best-effort): Passthrough.VCard <- VCardExtra
	if len(record.Passthrough.VCard) != 1 {
		t.Fatalf("Record.Passthrough.VCard = %+v, want 1 entry from VCardExtra", record.Passthrough.VCard)
	}
	if record.Passthrough.VCard[0].Name != "X-CUSTOM" {
		t.Errorf("Record.Passthrough.VCard[0].Name = %q, want X-CUSTOM", record.Passthrough.VCard[0].Name)
	}

	// Deliberate, flagged gap: Gender has no neutral home in this WP (see the
	// long comment in RecordFromContact). Contact.Gender itself must remain
	// completely untouched by RecordFromContact (it's read-only here).
	if c.Gender != "female" {
		t.Errorf("c.Gender was mutated to %q, want it untouched (RecordFromContact must not mutate its input)", c.Gender)
	}
}

// TestRecordFromContact_ZeroValue asserts RecordFromContact never panics on
// an empty Contact or a nil receiver, per this WP's explicit requirement
// (needed for BeforeSave on a brand-new, mostly-empty contact being created).
func TestRecordFromContact_ZeroValue(t *testing.T) {
	record := RecordFromContact(&Contact{}, "")
	if record == nil {
		t.Fatal("RecordFromContact(&Contact{}, \"\") returned nil")
	}
	if record.Card.Name != nil {
		t.Errorf("zero-value Contact produced non-nil Card.Name: %+v", record.Card.Name)
	}
	if len(record.Card.Emails) != 0 || len(record.Card.Phones) != 0 || len(record.Card.Addresses) != 0 {
		t.Errorf("zero-value Contact produced non-empty collections: %+v", record.Card)
	}
	if len(record.Card.Media) != 0 {
		t.Errorf("zero-value Contact produced non-empty Card.Media: %+v", record.Card.Media)
	}

	nilRecord := RecordFromContact(nil, "")
	if nilRecord == nil {
		t.Fatal("RecordFromContact(nil, \"\") returned nil, want a safe empty *Record")
	}
}

// TestRecordFromContact_ScalarOnlyFallback asserts that contacts whose
// Email/Phone/Address were only ever set as legacy scalars (no
// Emails/Phones/Addresses array entries — a real pattern in this codebase's
// own existing tests/fixtures) still get that data mapped into Card, so the
// P1 data migration is lossless for them.
func TestRecordFromContact_ScalarOnlyFallback(t *testing.T) {
	c := &Contact{
		Firstname: "Alice",
		Email:     "alice@example.com",
		Phone:     "+15550001111",
		Address:   "42 Wallaby Way, Sydney",
	}
	record := RecordFromContact(c, "")

	if len(record.Card.Emails) != 1 || record.Card.Emails[0].Address != "alice@example.com" {
		t.Errorf("Card.Emails = %+v, want scalar-fallback entry alice@example.com", record.Card.Emails)
	}
	if len(record.Card.Phones) != 1 || record.Card.Phones[0].Number != "+15550001111" {
		t.Errorf("Card.Phones = %+v, want scalar-fallback entry +15550001111", record.Card.Phones)
	}
	if len(record.Card.Addresses) != 1 || record.Card.Addresses[0].Full != "42 Wallaby Way, Sydney" {
		t.Errorf("Card.Addresses = %+v, want scalar-fallback Full=42 Wallaby Way, Sydney", record.Card.Addresses)
	}

	proj := contactmodel.DeriveProjection(record)
	if proj.PrimaryEmail != "alice@example.com" {
		t.Errorf("DeriveProjection PrimaryEmail = %q, want alice@example.com (round-trip of the scalar-only contact)", proj.PrimaryEmail)
	}
	if proj.PrimaryPhone != "+15550001111" {
		t.Errorf("DeriveProjection PrimaryPhone = %q, want +15550001111", proj.PrimaryPhone)
	}
}

// TestRecordForContact_PrefersPersistedCardOverFreshDerivation is the
// regression test for a real, live bug found while auditing WP-73's work:
// three call sites (CardDAV export, the REST API's detail/write response,
// and VCF/JSContact export) each independently called RecordFromContact a
// second time on a contact whose Card was already persisted, which silently
// discards any data with no flat-field home (SpeakToAs here) — exactly the
// data a nested REST POST/PUT or a CardDAV PUT of a modern vCard would have
// set. RecordForContact must return the persisted Card as-is instead of
// re-deriving it from the (necessarily lossy) flat fields.
func TestRecordForContact_PrefersPersistedCardOverFreshDerivation(t *testing.T) {
	c := &Contact{
		Firstname: "Ada",
		Card: contactmodel.Card{
			Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Ada"}}},
			SpeakToAs: &contactmodel.SpeakToAs{
				Pronouns: []contactmodel.Pronouns{{Pronouns: "she/her"}},
			},
		},
	}

	record := RecordForContact(c, "", nil)

	if record.Card.SpeakToAs == nil || len(record.Card.SpeakToAs.Pronouns) != 1 || record.Card.SpeakToAs.Pronouns[0].Pronouns != "she/her" {
		t.Errorf("RecordForContact's Card.SpeakToAs = %+v, want the persisted she/her preserved (a fresh RecordFromContact call would drop it — no flat field holds pronouns)", record.Card.SpeakToAs)
	}
}

// TestRecordForContact_FallsBackWhenCardIsZeroValue covers the other half of
// RecordForContact: a contact whose Card has never been populated at all
// (e.g. a pre-migration row that predates cmd/backfill-contact-records, or a
// Contact built directly in a test without going through BeforeSave) must
// still fall back to deriving from the flat fields — not return an empty Card.
func TestRecordForContact_FallsBackWhenCardIsZeroValue(t *testing.T) {
	c := &Contact{
		Firstname: "Bob",
		Email:     "bob@example.com",
	}

	record := RecordForContact(c, "", nil)

	if len(record.Card.Emails) != 1 || record.Card.Emails[0].Address != "bob@example.com" {
		t.Errorf("RecordForContact's Card.Emails = %+v, want a fallback-derived entry for bob@example.com (Card was zero-value, should behave like RecordFromContact)", record.Card.Emails)
	}
}

// TestBeforeSave_DerivesProjection asserts BeforeSave populates
// Firstname/Lastname/Email/Phone/Birthday/FN/Org (and the Card/CRM/
// Passthrough columns) via the new DeriveProjection-based path.
func TestBeforeSave_DerivesProjection(t *testing.T) {
	c := fullyPopulatedContact()

	if err := c.BeforeSave(nil); err != nil {
		t.Fatalf("BeforeSave returned error: %v", err)
	}

	if c.Firstname != "Jane" {
		t.Errorf("c.Firstname = %q, want Jane", c.Firstname)
	}
	if c.Lastname != "Doe" {
		t.Errorf("c.Lastname = %q, want Doe", c.Lastname)
	}
	if c.Email != "jane@example.com" {
		t.Errorf("c.Email = %q, want jane@example.com (from Emails[0], via DeriveProjection)", c.Email)
	}
	if c.Phone != "+15551234567" {
		t.Errorf("c.Phone = %q, want +15551234567 (from Phones[0], via DeriveProjection)", c.Phone)
	}
	if c.Birthday != "1990-06-15" {
		t.Errorf("c.Birthday = %q, want 1990-06-15 (round-tripped through Card.Anniversaries)", c.Birthday)
	}
	if c.FN != "Jane Doe" {
		t.Errorf("c.FN = %q, want \"Jane Doe\" (Card.Name.Full is unset, so FN falls back to given+surname)", c.FN)
	}
	if c.Org != "Acme Corp" {
		t.Errorf("c.Org = %q, want Acme Corp", c.Org)
	}

	if c.Card.UID != "uid-1234" {
		t.Errorf("c.Card.UID = %q, want uid-1234 (BeforeSave must populate the Card column too)", c.Card.UID)
	}
	if c.CRM.HowWeMet != "Conference" {
		t.Errorf("c.CRM.HowWeMet = %q, want Conference (BeforeSave must populate the CRM column too)", c.CRM.HowWeMet)
	}
	if len(c.Passthrough.VCard) != 1 {
		t.Errorf("c.Passthrough.VCard = %+v, want 1 entry (BeforeSave must populate the Passthrough column too)", c.Passthrough.VCard)
	}
}

// TestBeforeSave_DoesNotBlankScalarOnlyContact is a regression guard: many
// existing contacts (and existing tests/fixtures elsewhere in this
// codebase) set only the legacy Email/Phone scalars directly, without ever
// populating the Emails/Phones arrays. BeforeSave must not blank those
// scalars out just because DeriveProjection is now in the loop — the
// guarded ("only overwrite when non-empty") assignment plus
// RecordFromContact's scalar fallback together must make this a no-op.
func TestBeforeSave_DoesNotBlankScalarOnlyContact(t *testing.T) {
	c := &Contact{
		Firstname: "Bob",
		Email:     "bob@example.com",
		Phone:     "+15559998888",
	}

	if err := c.BeforeSave(nil); err != nil {
		t.Fatalf("BeforeSave returned error: %v", err)
	}

	if c.Email != "bob@example.com" {
		t.Errorf("c.Email = %q, want unchanged bob@example.com", c.Email)
	}
	if c.Phone != "+15559998888" {
		t.Errorf("c.Phone = %q, want unchanged +15559998888", c.Phone)
	}
}

// TestRoundTrip_ProjectionStable is the round-trip check called for by
// WP-70: for a sample of contacts, re-deriving the projection from a
// contact that has already gone through BeforeSave (simulating an
// already-migrated row) must reproduce an equal projection, i.e.
// RecordFromContact -> DeriveProjection is a stable fixed point, not a
// moving target that would drift on repeated saves/backfill runs.
func TestRoundTrip_ProjectionStable(t *testing.T) {
	samples := []*Contact{
		fullyPopulatedContact(),
		{Firstname: "Alice", Email: "alice@example.com", Phone: "+15550001111", Address: "42 Wallaby Way"},
		{Firstname: "Zero"},
	}

	for _, c := range samples {
		if err := c.BeforeSave(nil); err != nil {
			t.Fatalf("BeforeSave returned error for %q: %v", c.Firstname, err)
		}
		firstProj := contactmodel.DeriveProjection(RecordFromContact(c, ""))

		// Simulate a second migration/save pass over the now-migrated row.
		if err := c.BeforeSave(nil); err != nil {
			t.Fatalf("second BeforeSave returned error for %q: %v", c.Firstname, err)
		}
		secondProj := contactmodel.DeriveProjection(RecordFromContact(c, ""))

		if firstProj != secondProj {
			t.Errorf("projection for %q drifted across a second pass: first=%+v second=%+v", c.Firstname, firstProj, secondProj)
		}
	}
}
