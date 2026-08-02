package models

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mycorrhizal/contactmodel"
)

// TestApplyRecordToContact_RoundTrip exercises WP-71 Gap 1: RecordFromContact
// a fully-populated Contact into a Record, ApplyRecordToContact that Record
// onto a fresh Contact, and assert the result matches the original closely
// enough that nothing was silently lost in a way the doc doesn't already
// call out as an accepted, documented lossy case.
//
// Also exercises WP-73's photo-bridging prerequisite end-to-end: original
// (fullyPopulatedContact) carries a PhotoThumbnail, which RecordFromContact
// bridges into record.Card.Media, which ApplyRecordToContact (given a real
// photoDir) then decodes and persists back to disk — proving the photo
// round-trips through Card.Media in both directions, not just one.
func TestApplyRecordToContact_RoundTrip(t *testing.T) {
	original := fullyPopulatedContact()
	photoDir := t.TempDir()
	record := RecordFromContact(original, photoDir)

	got := &Contact{}
	ApplyRecordToContact(got, record, photoDir)

	// Card/CRM/Passthrough are the authoritative full-fidelity copy: exact,
	// byte-for-byte (deep-equal) match required.
	if !reflect.DeepEqual(got.Card, record.Card) {
		t.Errorf("got.Card = %+v, want exact match with record.Card = %+v", got.Card, record.Card)
	}
	if !reflect.DeepEqual(got.CRM, record.Envelope) {
		t.Errorf("got.CRM = %+v, want exact match with record.Envelope = %+v", got.CRM, record.Envelope)
	}
	if !reflect.DeepEqual(got.Passthrough, record.Passthrough) {
		t.Errorf("got.Passthrough = %+v, want exact match with record.Passthrough = %+v", got.Passthrough, record.Passthrough)
	}

	// Flat legacy fields: exact match expected for everything that has an
	// unambiguous neutral home.
	if got.Firstname != original.Firstname {
		t.Errorf("Firstname = %q, want %q", got.Firstname, original.Firstname)
	}
	if got.Lastname != original.Lastname {
		t.Errorf("Lastname = %q, want %q", got.Lastname, original.Lastname)
	}
	if got.MiddleName != original.MiddleName {
		t.Errorf("MiddleName = %q, want %q", got.MiddleName, original.MiddleName)
	}
	if got.Prefix != original.Prefix {
		t.Errorf("Prefix = %q, want %q", got.Prefix, original.Prefix)
	}
	if got.Suffix != original.Suffix {
		t.Errorf("Suffix = %q, want %q", got.Suffix, original.Suffix)
	}
	if got.Nickname != original.Nickname {
		t.Errorf("Nickname = %q, want %q", got.Nickname, original.Nickname)
	}
	if got.Organization != original.Organization {
		t.Errorf("Organization = %q, want %q", got.Organization, original.Organization)
	}
	if got.Department != original.Department {
		t.Errorf("Department = %q, want %q", got.Department, original.Department)
	}
	if got.JobTitle != original.JobTitle {
		t.Errorf("JobTitle = %q, want %q", got.JobTitle, original.JobTitle)
	}
	if got.Role != original.Role {
		t.Errorf("Role = %q, want %q", got.Role, original.Role)
	}
	if got.Birthday != original.Birthday {
		t.Errorf("Birthday = %q, want %q", got.Birthday, original.Birthday)
	}
	if got.Anniversary != original.Anniversary {
		t.Errorf("Anniversary = %q, want %q", got.Anniversary, original.Anniversary)
	}
	if got.HowWeMet != original.HowWeMet {
		t.Errorf("HowWeMet = %q, want %q", got.HowWeMet, original.HowWeMet)
	}
	if got.WorkInformation != original.WorkInformation {
		t.Errorf("WorkInformation = %q, want %q", got.WorkInformation, original.WorkInformation)
	}
	if got.ContactInformation != original.ContactInformation {
		t.Errorf("ContactInformation = %q, want %q", got.ContactInformation, original.ContactInformation)
	}
	if !reflect.DeepEqual(got.Circles, original.Circles) {
		t.Errorf("Circles = %+v, want %+v", got.Circles, original.Circles)
	}
	if !reflect.DeepEqual(got.CustomFields, original.CustomFields) {
		t.Errorf("CustomFields = %+v, want %+v", got.CustomFields, original.CustomFields)
	}
	if got.VCardUID != original.VCardUID {
		t.Errorf("VCardUID = %q, want %q", got.VCardUID, original.VCardUID)
	}

	// Emails/Phones/URLs/IMPPs: exact match (Label<->Type is a lossless 1:1
	// round trip for these).
	if !reflect.DeepEqual(got.Emails, original.Emails) {
		t.Errorf("Emails = %+v, want %+v", got.Emails, original.Emails)
	}
	if !reflect.DeepEqual(got.Phones, original.Phones) {
		t.Errorf("Phones = %+v, want %+v", got.Phones, original.Phones)
	}
	if !reflect.DeepEqual(got.URLs, original.URLs) {
		t.Errorf("URLs = %+v, want %+v", got.URLs, original.URLs)
	}
	if !reflect.DeepEqual(got.IMPPs, original.IMPPs) {
		t.Errorf("IMPPs = %+v, want %+v", got.IMPPs, original.IMPPs)
	}

	// Addresses: exact match expected here too, since fullyPopulatedContact's
	// one address has real structured components (not the Full-only fallback
	// case documented as lossy in applyAddresses).
	if !reflect.DeepEqual(got.Addresses, original.Addresses) {
		t.Errorf("Addresses = %+v, want %+v", got.Addresses, original.Addresses)
	}

	// Known, documented, ACCEPTED gaps (not asserted equal):
	//   - Gender: RecordFromContact never maps it into Record at all (no
	//     neutral home exists for it yet — see that function's own doc
	//     comment), so ApplyRecordToContact has nothing to restore it from.
	//     A fresh Contact{} naturally comes out with Gender == "", not
	//     original.Gender == "female". This is not a round-trip bug: Gender
	//     simply doesn't participate in this round trip, by design, on both
	//     sides.
	if got.Gender != "" {
		t.Errorf("got.Gender = %q, want \"\" (Gender has no neutral home; ApplyRecordToContact must not invent one)", got.Gender)
	}
	//   - VCardExtra: RecordFromContact's buildPassthrough is explicitly
	//     best-effort/lossy converting VCardExtra -> Passthrough.VCard (see
	//     that function's doc comment), and ApplyRecordToContact deliberately
	//     does not attempt to re-serialize Passthrough.VCard back into
	//     VCardExtra's bespoke JSON shape (see ApplyRecordToContact's own doc
	//     comment) — the data lives on, losslessly, in c.Passthrough (already
	//     asserted equal to record.Passthrough above), just not mirrored back
	//     into the legacy VCardExtra column.
	if got.VCardExtra != "" {
		t.Errorf("got.VCardExtra = %q, want \"\" (Passthrough is not re-serialized back into the legacy VCardExtra column)", got.VCardExtra)
	}

	// WP-73 photo-bridging prerequisite: the photo bridged into
	// record.Card.Media by RecordFromContact (from original.PhotoThumbnail,
	// since original.Photo has no on-disk file) round-trips through
	// ApplyRecordToContact back onto disk and onto got.Photo/PhotoThumbnail.
	if got.Photo == "" {
		t.Error("got.Photo is empty, want a saved photo filename (the Card.Media photo entry should have been persisted to disk)")
	}
	if !strings.HasPrefix(got.PhotoThumbnail, "data:image/jpeg;base64,") {
		t.Errorf("got.PhotoThumbnail = %q, want a data:image/jpeg;base64,... thumbnail", got.PhotoThumbnail)
	}
	if _, err := os.Stat(filepath.Join(photoDir, got.Photo)); err != nil {
		t.Errorf("saved photo file not found on disk at %q: %v", filepath.Join(photoDir, got.Photo), err)
	}

	// The cardSetDirectly guard must be set so a subsequent BeforeSave
	// doesn't clobber the Card/CRM/Passthrough we just asserted above.
	if !got.cardSetDirectly {
		t.Error("got.cardSetDirectly = false, want true after ApplyRecordToContact")
	}
}

// TestApplyRecordToContact_AddressFullOnlyFallback documents the one known
// lossy case called out in applyAddresses's doc comment: an Address with no
// structured Components, only Full, reconstructs to an all-blank
// ContactAddress (nothing structured to recover it from) — but the data is
// not gone, it remains verbatim in c.Card.Addresses[].Full.
func TestApplyRecordToContact_AddressFullOnlyFallback(t *testing.T) {
	record := &contactmodel.Record{
		Card: contactmodel.Card{
			Addresses: []contactmodel.Address{{Full: "42 Wallaby Way, Sydney"}},
		},
	}

	got := &Contact{}
	ApplyRecordToContact(got, record, "")

	if len(got.Addresses) != 1 {
		t.Fatalf("Addresses = %+v, want 1 (blank-but-present) entry", got.Addresses)
	}
	if (got.Addresses[0] != ContactAddress{}) {
		t.Errorf("Addresses[0] = %+v, want all-blank (no structured components to recover)", got.Addresses[0])
	}
	// The raw string is not lost: it is still on c.Card.
	if got.Card.Addresses[0].Full != "42 Wallaby Way, Sydney" {
		t.Errorf("Card.Addresses[0].Full = %q, want the original Full string preserved", got.Card.Addresses[0].Full)
	}
}

// TestApplyRecordToContact_NilSafety mirrors RecordFromContact's own
// nil-safety test: ApplyRecordToContact must not panic on a nil Contact or a
// nil Record.
func TestApplyRecordToContact_NilSafety(t *testing.T) {
	ApplyRecordToContact(nil, &contactmodel.Record{}, "")
	ApplyRecordToContact(&Contact{}, nil, "")
}

// TestApplyRecordToContact_PreservesUnmappedCardData asserts the central
// claim of Gap 1's resolution: Card-only data with no flat-field home
// (SpeakToAs here) is preserved on c.Card even though nothing on the flat
// side reflects it, and — critically — survives a subsequent BeforeSave
// untouched (this is what the cardSetDirectly guard exists to protect;
// without it, BeforeSave's flat->Card derivation would silently wipe
// SpeakToAs out on the very next save).
func TestApplyRecordToContact_PreservesUnmappedCardData(t *testing.T) {
	record := &contactmodel.Record{
		Card: contactmodel.Card{
			Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Ada"}}},
			SpeakToAs: &contactmodel.SpeakToAs{
				Pronouns: []contactmodel.Pronouns{{Pronouns: "she/her"}},
			},
		},
	}

	c := &Contact{}
	ApplyRecordToContact(c, record, "")

	if c.Card.SpeakToAs == nil || len(c.Card.SpeakToAs.Pronouns) != 1 || c.Card.SpeakToAs.Pronouns[0].Pronouns != "she/her" {
		t.Fatalf("c.Card.SpeakToAs = %+v, want the SpeakToAs from the input Record preserved", c.Card.SpeakToAs)
	}

	if err := c.BeforeSave(nil); err != nil {
		t.Fatalf("BeforeSave returned error: %v", err)
	}

	if c.Card.SpeakToAs == nil || len(c.Card.SpeakToAs.Pronouns) != 1 || c.Card.SpeakToAs.Pronouns[0].Pronouns != "she/her" {
		t.Errorf("c.Card.SpeakToAs after BeforeSave = %+v, want it still preserved (BeforeSave must not re-derive Card when cardSetDirectly was set)", c.Card.SpeakToAs)
	}
}

// TestApplyRecordToContact_ClearsPhoneScalarWhenPhonesRemoved is the
// regression guard for a real bug found by Tier 3c item 11a's audit
// (docs/fork-plan/95-backlog-and-priorities.md): applyPhones cleared
// c.Phones but left the c.Phone scalar untouched when the incoming Record
// had no phones at all, unlike its sibling applyEmails (which always
// resets c.Email, falling back to proj.PrimaryEmail). A contact whose last
// phone number was removed via REST PUT, CardDAV sync, or VCF import kept a
// stale c.Phone value even though c.Phones correctly went empty.
func TestApplyRecordToContact_ClearsPhoneScalarWhenPhonesRemoved(t *testing.T) {
	c := &Contact{Phone: "555-0100", Phones: []ContactPhone{{Type: "home", Value: "555-0100"}}}

	ApplyRecordToContact(c, &contactmodel.Record{Card: contactmodel.Card{
		Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Ada"}}},
	}}, "")

	if len(c.Phones) != 0 {
		t.Errorf("c.Phones = %+v, want empty (incoming Record has no phones)", c.Phones)
	}
	if c.Phone != "" {
		t.Errorf("c.Phone = %q, want empty — must not retain a stale value once Phones is cleared", c.Phone)
	}
}
