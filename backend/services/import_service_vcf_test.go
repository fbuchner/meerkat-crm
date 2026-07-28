package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"mycorrhizal/contactmodel"
	"mycorrhizal/jscontact"
	"mycorrhizal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseVCF_BasicImport_VCard4 exercises WP-71 Gap 4: ParseVCF now routes
// through the vcard4 adapter (sniffed from VERSION:4.0) instead of the
// legacy carddav.VCardToContact mapper, and the resulting Record is turned
// into a flat *models.Contact via models.ApplyRecordToContact.
func TestParseVCF_BasicImport_VCard4(t *testing.T) {
	db, _ := setupRouter()

	var user models.User
	db.First(&user)

	raw := "BEGIN:VCARD\r\n" +
		"VERSION:4.0\r\n" +
		"FN:John Doe\r\n" +
		"N:Doe;John;;;\r\n" +
		"EMAIL:john@example.com\r\n" +
		"TEL:+15551234567\r\n" +
		"END:VCARD\r\n"

	contacts, previews, stats, err := ParseVCF(strings.NewReader(raw), db, user.ID)
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	require.Len(t, previews, 1)

	assert.Equal(t, 1, stats.ValidCount)
	assert.Equal(t, 0, stats.ErrorCount)
	assert.Equal(t, "add", previews[0].SuggestedAction)

	contact := contacts[0].Contact
	assert.Equal(t, "John", contact.Firstname)
	assert.Equal(t, "Doe", contact.Lastname)
	assert.Equal(t, "john@example.com", contact.Email)
	assert.Equal(t, "+15551234567", contact.Phone)
	assert.NotEmpty(t, contact.VCardUID, "a UID should be generated when the vCard has none")

	// The neutral Card must also be populated directly (ApplyRecordToContact
	// sets it as the authoritative copy), not just the flat fields.
	assert.Len(t, contact.Card.Emails, 1)
	assert.Equal(t, "john@example.com", contact.Card.Emails[0].Address)
}

// TestParseVCF_VCard3Sniffing asserts a VERSION:3.0 block is routed through
// the vcard3 adapter and still produces a usable Contact.
func TestParseVCF_VCard3Sniffing(t *testing.T) {
	db, _ := setupRouter()

	var user models.User
	db.First(&user)

	raw := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		"FN:Jane Roe\r\n" +
		"N:Roe;Jane;;;\r\n" +
		"EMAIL:jane@example.com\r\n" +
		"END:VCARD\r\n"

	contacts, _, stats, err := ParseVCF(strings.NewReader(raw), db, user.ID)
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	assert.Equal(t, 1, stats.ValidCount)

	contact := contacts[0].Contact
	assert.Equal(t, "Jane", contact.Firstname)
	assert.Equal(t, "Roe", contact.Lastname)
	assert.Equal(t, "jane@example.com", contact.Email)
}

// TestParseVCF_MultipleCardsSplit asserts a .vcf file containing several
// concatenated vCard blocks (the normal shape of a real export) is split
// into one contact per block, not just the first one.
func TestParseVCF_MultipleCardsSplit(t *testing.T) {
	db, _ := setupRouter()

	var user models.User
	db.First(&user)

	raw := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Alice A\r\nN:A;Alice;;;\r\nEND:VCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Bob B\r\nN:B;Bob;;;\r\nEND:VCARD\r\n"

	contacts, previews, stats, err := ParseVCF(strings.NewReader(raw), db, user.ID)
	require.NoError(t, err)
	require.Len(t, contacts, 2)
	require.Len(t, previews, 2)
	assert.Equal(t, 2, stats.ValidCount)
	assert.Equal(t, "Alice", contacts[0].Contact.Firstname)
	assert.Equal(t, "Bob", contacts[1].Contact.Firstname)
}

// TestParseVCF_DuplicateDetectionAndMerge exercises the Gap 4 promise most
// directly: duplicate-detection/merge UX (DetectDuplicate/
// MergeImportedContact, unchanged) must keep working against the new
// adapter-produced flat Contact fields.
func TestParseVCF_DuplicateDetectionAndMerge(t *testing.T) {
	db, _ := setupRouter()

	var user models.User
	db.First(&user)

	existing := models.Contact{
		UserID:    user.ID,
		Firstname: "John",
		Lastname:  "Doe",
		Email:     "john@example.com",
		Phone:     "+10000000000",
	}
	require.NoError(t, db.Create(&existing).Error)

	raw := "BEGIN:VCARD\r\n" +
		"VERSION:4.0\r\n" +
		"FN:John Doe\r\n" +
		"N:Doe;John;;;\r\n" +
		"EMAIL:john@example.com\r\n" +
		"TEL:+15559998888\r\n" +
		"END:VCARD\r\n"

	contacts, previews, stats, err := ParseVCF(strings.NewReader(raw), db, user.ID)
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	require.Len(t, previews, 1)

	assert.Equal(t, 1, stats.DuplicateCount)
	assert.Equal(t, "update", previews[0].SuggestedAction)
	require.NotNil(t, previews[0].DuplicateMatch)
	assert.Equal(t, existing.ID, previews[0].DuplicateMatch.ExistingContactID)

	// MergeImportedContact (unchanged) must still work against the
	// adapter-produced incoming contact.
	MergeImportedContact(&existing, contacts[0].Contact)
	assert.Equal(t, "+15559998888", existing.Phone)
}

// TestParseVCF_MalformedBlockSkipped asserts a malformed block is skipped
// (reported as an error preview row) without aborting the whole file.
func TestParseVCF_ReadError(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	r := &errReader{err: fmt.Errorf("simulated read failure")}
	_, _, _, err := ParseVCF(r, db, user.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read VCF file")
}

// TestParseVCF_TooManyContacts asserts the MaxVCFContacts guard applies to
// VCF imports.
func TestParseVCF_TooManyContacts(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	var buf strings.Builder
	for i := 0; i <= MaxVCFContacts; i++ {
		buf.WriteString("BEGIN:VCARD\r\nVERSION:4.0\r\nFN:X\r\nEND:VCARD\r\n")
	}

	_, _, _, err := ParseVCF(strings.NewReader(buf.String()), db, user.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many contacts")
}

func TestParseVCF_MalformedBlockSkipped(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	// A BEGIN/END block with no colon-separated properties at all inside is
	// still syntactically a vCard to go-vcard (empty card), so use raw bytes
	// that don't match the BEGIN/END framing at all to force zero blocks.
	_, _, _, err := ParseVCF(strings.NewReader("this is not a vcard at all"), db, user.ID)
	assert.Error(t, err)
}

// TestParseJSContact_BasicImport exercises the new JSContact import path
// (Gap 4 extension). It builds a JSContact document via the jscontact
// Adapter's own Export (rather than hand-authoring wire JSON, which risks
// drifting from the real shape), then feeds it back through
// services.ParseJSContact and asserts the round trip lands the same data.
func TestParseJSContact_BasicImport(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	record := &contactmodel.Record{
		Card: contactmodel.Card{
			Name: &contactmodel.Name{Components: []contactmodel.NameComponent{
				{Kind: "given", Value: "Ada"},
				{Kind: "surname", Value: "Lovelace"},
			}},
			Emails: []contactmodel.Email{{Address: "ada@example.com"}},
		},
	}
	raw, _, err := jscontact.Adapter{}.Export(record)
	require.NoError(t, err)

	contacts, previews, stats, err := ParseJSContact(strings.NewReader(string(raw)), db, user.ID)
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	require.Len(t, previews, 1)
	assert.Equal(t, 1, stats.ValidCount)

	contact := contacts[0].Contact
	assert.Equal(t, "Ada", contact.Firstname)
	assert.Equal(t, "Lovelace", contact.Lastname)
	assert.Equal(t, "ada@example.com", contact.Email)
}

// TestParseJSContact_ArrayForm asserts the "Card set" (JSON array) shape
// ExportContactsAsJSContact produces is also accepted on import.
func TestParseJSContact_ArrayForm(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	rec1 := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Alice"}}}}}
	rec2 := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Bob"}}}}}
	raw1, _, err := jscontact.Adapter{}.Export(rec1)
	require.NoError(t, err)
	raw2, _, err := jscontact.Adapter{}.Export(rec2)
	require.NoError(t, err)

	arr, err := json.Marshal([]json.RawMessage{raw1, raw2})
	require.NoError(t, err)

	contacts, _, stats, err := ParseJSContact(strings.NewReader(string(arr)), db, user.ID)
	require.NoError(t, err)
	require.Len(t, contacts, 2)
	assert.Equal(t, 2, stats.ValidCount)
	assert.Equal(t, "Alice", contacts[0].Contact.Firstname)
	assert.Equal(t, "Bob", contacts[1].Contact.Firstname)
}

func TestParseJSContact_ReadError(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	r := &errReader{err: fmt.Errorf("simulated read failure")}
	_, _, _, err := ParseJSContact(r, db, user.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read JSContact file")
}

func TestParseJSContact_EmptyArray_Errors(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	_, _, _, err := ParseJSContact(strings.NewReader("[]"), db, user.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no cards")
}

// TestParseJSContact_MalformedCardInArray asserts one bad card in an array
// doesn't abort the whole import — it's recorded as a per-row error preview
// (SuggestedAction "skip") and the rest of the batch still parses.
func TestParseJSContact_MalformedCardInArray(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	good := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Carol"}}}}}
	goodRaw, _, err := jscontact.Adapter{}.Export(good)
	require.NoError(t, err)

	badRaw := json.RawMessage(`{"@type":"Card","version":"1.0","name":{"full":123}}`)

	arr, err := json.Marshal([]json.RawMessage{badRaw, goodRaw})
	require.NoError(t, err)

	contacts, previews, stats, err := ParseJSContact(strings.NewReader(string(arr)), db, user.ID)
	require.NoError(t, err)
	require.Len(t, contacts, 1, "the malformed card should be skipped, not abort the batch")
	require.Len(t, previews, 2)
	assert.Equal(t, 1, stats.ErrorCount)
	assert.Equal(t, "skip", previews[0].SuggestedAction)
	assert.NotEmpty(t, previews[0].ValidationErrors)
	assert.Equal(t, "Carol", contacts[0].Contact.Firstname)
}

// TestParseJSContact_AllCardsFail_ReturnsNoValidContactsError asserts the
// final "JSContact file contains no valid contacts" branch when every card
// in the file fails to parse.
func TestParseJSContact_AllCardsFail_ReturnsNoValidContactsError(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	badRaw := json.RawMessage(`{"@type":"Card","version":"1.0","name":{"full":123}}`)
	arr, err := json.Marshal([]json.RawMessage{badRaw})
	require.NoError(t, err)

	_, _, _, err = ParseJSContact(strings.NewReader(string(arr)), db, user.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid contacts")
}

// TestParseJSContact_DuplicateDetected asserts the duplicate-detection
// branch (mirrors ParseVCF's own duplicate handling) fires for JSContact
// import too: an existing contact matching by email is flagged as an
// "update" suggestion rather than "add".
func TestParseJSContact_DuplicateDetected(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	existing := models.Contact{UserID: user.ID, Firstname: "Dana", Email: "dana@example.com"}
	require.NoError(t, db.Create(&existing).Error)

	rec := &contactmodel.Record{Card: contactmodel.Card{
		Name:   &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "Dana"}}},
		Emails: []contactmodel.Email{{Address: "dana@example.com"}},
	}}
	raw, _, err := jscontact.Adapter{}.Export(rec)
	require.NoError(t, err)

	_, previews, stats, err := ParseJSContact(strings.NewReader(string(raw)), db, user.ID)
	require.NoError(t, err)
	require.Len(t, previews, 1)
	assert.Equal(t, 1, stats.DuplicateCount)
	assert.Equal(t, "update", previews[0].SuggestedAction)
	require.NotNil(t, previews[0].DuplicateMatch)
}

// TestParseJSContact_TooManyContacts asserts the MaxVCFContacts guard (shared
// with ParseVCF) applies to JSContact imports too.
func TestParseJSContact_TooManyContacts(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	raws := make([]json.RawMessage, 0, MaxVCFContacts+1)
	for i := 0; i <= MaxVCFContacts; i++ {
		rec := &contactmodel.Record{Card: contactmodel.Card{Name: &contactmodel.Name{Components: []contactmodel.NameComponent{{Kind: "given", Value: "X"}}}}}
		raw, _, err := jscontact.Adapter{}.Export(rec)
		require.NoError(t, err)
		raws = append(raws, raw)
	}
	arr, err := json.Marshal(raws)
	require.NoError(t, err)

	_, _, _, err = ParseJSContact(strings.NewReader(string(arr)), db, user.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many contacts")
}
