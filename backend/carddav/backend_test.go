package carddav

import (
	"context"
	"meerkat/models"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	webdavcarddav "github.com/emersion/go-webdav/carddav"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newTestBackend builds an in-memory-SQLite-backed Backend for CardDAV
// integration tests, mirroring the setup pattern used by
// controllers/activity_controller_test.go's setupRouter.
func newTestBackend(t *testing.T) (*Backend, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&models.Contact{}); err != nil {
		t.Fatalf("failed to migrate Contact: %v", err)
	}
	return NewBackend(db, t.TempDir()), db
}

// TestPutGetAddressObject_VCard4RoundTrip is the concrete, end-to-end proof
// for docs/fork-plan/50-integration-and-rebrand.md WP-73's mapper-retirement
// step: a vCard 4.0 body carrying PRONOUNS and GRAMGENDER — properties the
// retired legacy carddav.VCardToContact/ContactToVCard mapper never
// understood at all — is PUT through Backend.PutAddressObject (which now
// routes through the vcard4 adapter + models.ApplyRecordToContact instead of
// the legacy mapper) and then read back via Backend.GetAddressObject (which
// routes through models.RecordFromContact + the vcard4 adapter's Export).
// The data surviving that round trip end-to-end (not just "the adapters
// compile") is the actual proof the new CardDAV code path works.
func TestPutGetAddressObject_VCard4RoundTrip(t *testing.T) {
	backend, db := newTestBackend(t)
	ctx := ContextWithUser(context.Background(), 1, "tester", db, backend.photoDir, "")

	const uid = "gramgender-pronouns-roundtrip"
	urlPath := "/carddav/addressbooks/tester/contacts/" + uid + ".vcf"

	raw := "BEGIN:VCARD\r\n" +
		"VERSION:4.0\r\n" +
		"UID:" + uid + "\r\n" +
		"FN:Alex Doe\r\n" +
		"N:Doe;Alex;;;\r\n" +
		"PRONOUNS;PREF=1:they/them\r\n" +
		"GRAMGENDER;LANGUAGE=en:neuter\r\n" +
		"END:VCARD\r\n"

	parsed, err := vcard.NewDecoder(strings.NewReader(raw)).Decode()
	if err != nil {
		t.Fatalf("failed to decode test vCard: %v", err)
	}

	// PUT: exercises PutAddressObject's new vcard4-adapter +
	// ApplyRecordToContact path.
	putObj, err := backend.PutAddressObject(ctx, urlPath, parsed, nil)
	if err != nil {
		t.Fatalf("PutAddressObject returned error: %v", err)
	}
	if putObj == nil {
		t.Fatal("PutAddressObject returned a nil AddressObject")
	}

	// Sanity: the contact was actually persisted with the flat fields
	// ApplyRecordToContact derives (proves this isn't just an in-memory
	// echo of the input).
	var stored models.Contact
	if err := db.Where("user_id = ? AND vcard_uid = ?", uint(1), uid).First(&stored).Error; err != nil {
		t.Fatalf("contact was not persisted: %v", err)
	}
	if stored.Firstname != "Alex" || stored.Lastname != "Doe" {
		t.Errorf("stored contact name = %q/%q, want Alex/Doe", stored.Firstname, stored.Lastname)
	}

	// GET: exercises GetAddressObject's new RecordFromContact + vcard4
	// adapter Export path, independently reloading from the DB rather than
	// reusing PutAddressObject's return value.
	getObj, err := backend.GetAddressObject(ctx, urlPath, nil)
	if err != nil {
		t.Fatalf("GetAddressObject returned error: %v", err)
	}
	if getObj == nil {
		t.Fatal("GetAddressObject returned a nil AddressObject")
	}

	cases := []struct {
		name string
		obj  *webdavcarddav.AddressObject
	}{
		{name: "PutAddressObject response", obj: putObj},
		{name: "GetAddressObject response", obj: getObj},
	}
	for _, tc := range cases {
		card := tc.obj.Card
		if v := card.Value("VERSION"); v != "4.0" {
			t.Errorf("%s: VERSION = %q, want 4.0 (DefaultVCardVersion)", tc.name, v)
		}
		if v := card.Value("PRONOUNS"); v != "they/them" {
			t.Errorf("%s: PRONOUNS = %q, want they/them (never supported by the legacy mapper)", tc.name, v)
		}
		if v := card.Value("GRAMGENDER"); v != "neuter" {
			t.Errorf("%s: GRAMGENDER = %q, want neuter (never supported by the legacy mapper)", tc.name, v)
		}
		if v := card.Value("FN"); v != "Alex Doe" {
			t.Errorf("%s: FN = %q, want Alex Doe", tc.name, v)
		}
	}
}

// mustParseVCard decodes raw vCard text into a vcard.Card, the type
// PutAddressObject/AddressObject.Card carry (go-webdav parses the wire body
// before calling Backend.PutAddressObject, and always re-encodes
// AddressObject.Card itself — see contactToAddressObject's doc comment).
func mustParseVCard(t *testing.T, raw string) vcard.Card {
	t.Helper()
	parsed, err := vcard.NewDecoder(strings.NewReader(raw)).Decode()
	if err != nil {
		t.Fatalf("failed to decode test vCard: %v", err)
	}
	return parsed
}

// simpleVCard4 is a minimal, valid vCard 4.0 body for tests that don't care
// about the card's content beyond having a distinguishable FN/UID.
func simpleVCard4(uid, fn string) string {
	return "BEGIN:VCARD\r\nVERSION:4.0\r\nUID:" + uid + "\r\nFN:" + fn + "\r\nN:" + fn + ";;;;\r\nEND:VCARD\r\n"
}

// TestPutGetAddressObject_VCard3RoundTrip is TestPutGetAddressObject_VCard4RoundTrip's
// counterpart for the vCard 3.0 path: backend_test.go previously only proved
// the 4.0 adapter route worked end to end, leaving PutAddressObject's
// importerForVCard 3.0-sniffing branch (and GetAddressObject's 3.0 export
// branch, via requestedVCardVersion) completely unexercised.
//
// The PUT body deliberately uses genuine RFC 2426 3.0-only idiom (comma-
// joined TYPE token lists, PREF as a TYPE token rather than 4.0's dedicated
// parameter — see docs/specs/rfc2426-v3-baseline.md's canonical example) and
// omits PRONOUNS/GRAMGENDER, which docs/specs/rfc2426-v3-baseline.md #5
// confirms don't exist at all in RFC 2426 — so this isn't just "a vCard the
// 4.0 adapter would also happily accept".
func TestPutGetAddressObject_VCard3RoundTrip(t *testing.T) {
	backend, db := newTestBackend(t)
	putCtx := ContextWithUser(context.Background(), 1, "tester", db, backend.photoDir, "")

	const uid = "vcard3-roundtrip-uid"
	urlPath := "/carddav/addressbooks/tester/contacts/" + uid + ".vcf"

	raw := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		"UID:" + uid + "\r\n" +
		"FN:Frank Dawson\r\n" +
		"N:Dawson;Frank;;;\r\n" +
		"ORG:Lotus Development Corporation\r\n" +
		"ADR;TYPE=WORK,POSTAL,PARCEL:;;6544 Battleford Drive;Raleigh;NC;27613-3502;U.S.A.\r\n" +
		"TEL;TYPE=VOICE,WORK:+1-919-676-9515\r\n" +
		"EMAIL;TYPE=INTERNET,PREF:Frank_Dawson@Lotus.com\r\n" +
		"END:VCARD\r\n"

	// PUT: exercises PutAddressObject's importerForVCard sniffing "3" off the
	// VERSION field and routing to vcard3.Adapter (rather than the 4.0
	// default).
	putObj, err := backend.PutAddressObject(putCtx, urlPath, mustParseVCard(t, raw), nil)
	if err != nil {
		t.Fatalf("PutAddressObject returned error: %v", err)
	}
	if putObj == nil {
		t.Fatal("PutAddressObject returned a nil AddressObject")
	}

	// Sanity: the contact was actually persisted with the flat fields
	// vcard3.Adapter.Import + ApplyRecordToContact derive.
	var stored models.Contact
	if err := db.Where("user_id = ? AND vcard_uid = ?", uint(1), uid).First(&stored).Error; err != nil {
		t.Fatalf("contact was not persisted: %v", err)
	}
	if stored.Firstname != "Frank" || stored.Lastname != "Dawson" {
		t.Errorf("stored contact name = %q/%q, want Frank/Dawson", stored.Firstname, stored.Lastname)
	}
	if stored.Organization != "Lotus Development Corporation" {
		t.Errorf("stored organization = %q, want Lotus Development Corporation", stored.Organization)
	}

	// GET, explicitly requesting 3.0 back via the Accept-header negotiation
	// mechanism (ContextWithUser's acceptHeaderValue -> requestedVCardVersion's
	// "version=3" match). Without this, GetAddressObject would serve
	// DefaultVCardVersion (4.0 unless overridden), which would not prove the
	// 3.0 export path was actually taken.
	getCtx := ContextWithUser(context.Background(), 1, "tester", db, backend.photoDir, "text/vcard;version=3.0")
	getObj, err := backend.GetAddressObject(getCtx, urlPath, nil)
	if err != nil {
		t.Fatalf("GetAddressObject returned error: %v", err)
	}
	if getObj == nil {
		t.Fatal("GetAddressObject returned a nil AddressObject")
	}

	if v := getObj.Card.Value("VERSION"); v != "3.0" {
		t.Errorf("VERSION = %q, want 3.0 (explicit Accept version=3.0 request)", v)
	}
	if v := getObj.Card.Value("FN"); v != "Frank Dawson" {
		t.Errorf("FN = %q, want Frank Dawson", v)
	}
	if v := getObj.Card.Value("ORG"); v != "Lotus Development Corporation" {
		t.Errorf("ORG = %q, want Lotus Development Corporation", v)
	}
	if v := getObj.Card.Value("TEL"); v != "+1-919-676-9515" {
		t.Errorf("TEL = %q, want +1-919-676-9515", v)
	}
	if v := getObj.Card.Value("EMAIL"); v != "Frank_Dawson@Lotus.com" {
		t.Errorf("EMAIL = %q, want Frank_Dawson@Lotus.com", v)
	}
	addrs := getObj.Card.Addresses()
	if len(addrs) != 1 {
		t.Fatalf("expected 1 address, got %d", len(addrs))
	}
	a := addrs[0]
	if a.StreetAddress != "6544 Battleford Drive" || a.Locality != "Raleigh" || a.Region != "NC" ||
		a.PostalCode != "27613-3502" || a.Country != "U.S.A." {
		t.Errorf("address structure lost/corrupted: %+v", a)
	}
}

// TestGetAddressObject_VersionNegotiation pins requestedVCardVersion's
// content-negotiation behavior (backend.go): an explicit version=3/version=4
// hint in the Accept header served through ContextWithUser's acceptHeaderValue
// must actually change which adapter GetAddressObject exports through, and
// the absence of any hint must fall back to DefaultVCardVersion. Before this
// test, requestedVCardVersion had no direct test coverage of its own — only
// the single hard-coded 4.0 default path was incidentally exercised by
// TestPutGetAddressObject_VCard4RoundTrip.
func TestGetAddressObject_VersionNegotiation(t *testing.T) {
	backend, db := newTestBackend(t)
	putCtx := ContextWithUser(context.Background(), 1, "tester", db, backend.photoDir, "")

	const uid = "version-negotiation-uid"
	urlPath := "/carddav/addressbooks/tester/contacts/" + uid + ".vcf"
	if _, err := backend.PutAddressObject(putCtx, urlPath, mustParseVCard(t, simpleVCard4(uid, "Nego Tiator")), nil); err != nil {
		t.Fatalf("PutAddressObject returned error: %v", err)
	}

	cases := []struct {
		name         string
		acceptHeader string
		wantVersion  string
	}{
		{name: "explicit version=3.0", acceptHeader: "text/vcard;version=3.0", wantVersion: "3.0"},
		{name: "explicit version=4.0", acceptHeader: "text/vcard;version=4.0", wantVersion: "4.0"},
		{name: "no Accept hint uses DefaultVCardVersion", acceptHeader: "", wantVersion: DefaultVCardVersion},
		{name: "generic text/vcard without version uses DefaultVCardVersion", acceptHeader: "text/vcard", wantVersion: DefaultVCardVersion},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getCtx := ContextWithUser(context.Background(), 1, "tester", db, backend.photoDir, tc.acceptHeader)
			obj, err := backend.GetAddressObject(getCtx, urlPath, nil)
			if err != nil {
				t.Fatalf("GetAddressObject returned error: %v", err)
			}
			if v := obj.Card.Value("VERSION"); v != tc.wantVersion {
				t.Errorf("Accept=%q: VERSION = %q, want %q", tc.acceptHeader, v, tc.wantVersion)
			}
		})
	}
}

// TestGetAddressObject_MultiUserIsolation proves the CardDAV server backend
// scopes every read to the authenticated user: two users each sync in their
// own contact, and neither ListAddressObjects nor GetAddressObject for one
// user may return or be affected by the other user's data — including the
// case where user A guesses/constructs a path containing user B's UID
// (GetAddressObject only trusts the userID out of the context, not the
// username portion of the path, so this specifically proves that isn't a
// cross-user data leak).
func TestGetAddressObject_MultiUserIsolation(t *testing.T) {
	backend, db := newTestBackend(t)

	const userA, userB = uint(1), uint(2)
	ctxA := ContextWithUser(context.Background(), userA, "alice", db, backend.photoDir, "")
	ctxB := ContextWithUser(context.Background(), userB, "bob", db, backend.photoDir, "")

	pathFor := func(username, uid string) string {
		return "/carddav/addressbooks/" + username + "/contacts/" + uid + ".vcf"
	}

	if _, err := backend.PutAddressObject(ctxA, pathFor("alice", "alice-uid"), mustParseVCard(t, simpleVCard4("alice-uid", "Alice A")), nil); err != nil {
		t.Fatalf("PutAddressObject(alice) returned error: %v", err)
	}
	if _, err := backend.PutAddressObject(ctxB, pathFor("bob", "bob-uid"), mustParseVCard(t, simpleVCard4("bob-uid", "Bob B")), nil); err != nil {
		t.Fatalf("PutAddressObject(bob) returned error: %v", err)
	}

	listA, err := backend.ListAddressObjects(ctxA, pathFor("alice", ""), nil)
	if err != nil {
		t.Fatalf("ListAddressObjects(alice) returned error: %v", err)
	}
	if len(listA) != 1 || listA[0].Card.Value("FN") != "Alice A" {
		t.Fatalf("ListAddressObjects(alice) = %+v, want exactly Alice's contact", listA)
	}

	listB, err := backend.ListAddressObjects(ctxB, pathFor("bob", ""), nil)
	if err != nil {
		t.Fatalf("ListAddressObjects(bob) returned error: %v", err)
	}
	if len(listB) != 1 || listB[0].Card.Value("FN") != "Bob B" {
		t.Fatalf("ListAddressObjects(bob) = %+v, want exactly Bob's contact", listB)
	}

	// Cross-user GetAddressObject must 404/error, not return the other
	// user's contact.
	if obj, err := backend.GetAddressObject(ctxA, pathFor("alice", "bob-uid"), nil); err == nil {
		t.Errorf("expected error when user A requests user B's UID, got object: %+v", obj)
	}
	if obj, err := backend.GetAddressObject(ctxB, pathFor("bob", "alice-uid"), nil); err == nil {
		t.Errorf("expected error when user B requests user A's UID, got object: %+v", obj)
	}
}

// TestDeleteAddressObject pins DeleteAddressObject's actual current behavior
// (backend.go's doc comment says "soft delete"): the underlying
// models.Contact row is archived via GORM's soft delete (DeletedAt set, row
// retained) rather than hard-removed, and a subsequent GetAddressObject for
// the same path must fail since gorm.Model-backed queries exclude
// soft-deleted rows by default.
func TestDeleteAddressObject(t *testing.T) {
	backend, db := newTestBackend(t)
	ctx := ContextWithUser(context.Background(), 1, "tester", db, backend.photoDir, "")

	const uid = "delete-me-uid"
	urlPath := "/carddav/addressbooks/tester/contacts/" + uid + ".vcf"
	if _, err := backend.PutAddressObject(ctx, urlPath, mustParseVCard(t, simpleVCard4(uid, "Delete Me")), nil); err != nil {
		t.Fatalf("PutAddressObject returned error: %v", err)
	}

	var before models.Contact
	if err := db.Where("user_id = ? AND vcard_uid = ?", uint(1), uid).First(&before).Error; err != nil {
		t.Fatalf("contact was not persisted before delete: %v", err)
	}

	if err := backend.DeleteAddressObject(ctx, urlPath); err != nil {
		t.Fatalf("DeleteAddressObject returned error: %v", err)
	}

	if obj, err := backend.GetAddressObject(ctx, urlPath, nil); err == nil {
		t.Errorf("expected GetAddressObject to fail for a deleted contact, got object: %+v", obj)
	}

	var afterUnscoped models.Contact
	if err := db.Unscoped().Where("id = ?", before.ID).First(&afterUnscoped).Error; err != nil {
		t.Fatalf("contact row should still exist after a soft delete, got error: %v", err)
	}
	if !afterUnscoped.DeletedAt.Valid {
		t.Error("expected DeletedAt to be set after DeleteAddressObject (soft delete)")
	}

	// Deleting an already-deleted contact must error, not panic or silently
	// no-op.
	if err := backend.DeleteAddressObject(ctx, urlPath); err == nil {
		t.Error("expected deleting an already-deleted contact to return an error")
	}
}

// TestPutAddressObject_MalformedCardRejected proves the malformed-input
// boundary check backend.go's PutAddressObject actually has: since
// PutAddressObject's signature takes an already-decoded vcard.Card (go-webdav
// parses the wire body before calling the Backend at all — see
// ContextWithUser's doc comment on why this layer never sees raw bytes
// directly), the reachable equivalent of "genuinely unparseable input" is a
// vcard.Card missing the VERSION field go-vcard's own Encoder requires to
// serialize anything back to wire format at all (confirmed via `go doc
// github.com/emersion/go-vcard Encoder.Encode`: "The card must have a
// FieldVersion field.") — PutAddressObject's own re-encode step
// (`vcard.NewEncoder(&buf).Encode(card)`) is exactly where that surfaces.
// This must return a clear error and must not create a Contact row.
func TestPutAddressObject_MalformedCardRejected(t *testing.T) {
	backend, db := newTestBackend(t)
	ctx := ContextWithUser(context.Background(), 1, "tester", db, backend.photoDir, "")

	card := make(vcard.Card)
	card.SetValue(vcard.FieldFormattedName, "Ghost")
	// Deliberately no VERSION field.

	urlPath := "/carddav/addressbooks/tester/contacts/malformed-uid.vcf"
	obj, err := backend.PutAddressObject(ctx, urlPath, card, nil)
	if err == nil {
		t.Fatal("expected PutAddressObject to reject a vCard with no VERSION field, got nil error")
	}
	if obj != nil {
		t.Errorf("expected a nil AddressObject on error, got %+v", obj)
	}

	var count int64
	if err := db.Model(&models.Contact{}).Where("user_id = ?", uint(1)).Count(&count).Error; err != nil {
		t.Fatalf("failed to count contacts: %v", err)
	}
	if count != 0 {
		t.Errorf("expected no Contact row to be created for a malformed PUT, got %d", count)
	}
}
