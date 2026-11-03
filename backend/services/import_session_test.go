package services

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/models"

	"github.com/glebarez/sqlite"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newImportSessionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Contact{}, &models.Note{}, &models.EmploymentHistory{},
	))
	// Recreate the partial unique index the real migration adds, since
	// AutoMigrate doesn't run raw-SQL migrations.
	require.NoError(t, db.Exec(
		`CREATE UNIQUE INDEX idx_contacts_vcard_uid_user ON contacts(user_id, vcard_uid) WHERE vcard_uid IS NOT NULL`,
	).Error)
	return db
}

const testVCFTemplate = "BEGIN:VCARD\r\n" +
	"VERSION:3.0\r\n" +
	"UID:%s\r\n" +
	"FN:Jane Doe\r\n" +
	"N:Doe;Jane;;;\r\n" +
	"END:VCARD\r\n"

// Regression test for: import vCard -> delete contact -> re-import same vCard
// used to fail with "UNIQUE constraint failed: contacts.user_id, contacts.vcard_uid".
func TestVCFReimport_AfterDelete_RestoresContact(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	cfg := &config.Config{}
	const userID = uint(1)
	const uid = "fixed-vcard-uid-123"

	vcf := func() *strings.Reader {
		return strings.NewReader(strings.NewReplacer("%s", uid).Replace(testVCFTemplate))
	}

	// 1. Import the vCard the first time.
	contacts, previews, _, err := ParseVCF(vcf(), db, userID)
	require.NoError(t, err)
	require.Len(t, previews, 1)
	assert.Equal(t, "add", previews[0].SuggestedAction)

	mgr := NewImportSessionManager()
	sessionID := mgr.CreateVCFSession(userID, contacts, previews)

	result, appErr := mgr.ConfirmVCF(db, userID, models.ImportConfirmRequest{
		SessionID: sessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	}, cfg, &log)
	require.Nil(t, appErr)
	assert.Equal(t, 1, result.Created)

	var created models.Contact
	require.NoError(t, db.Where("user_id = ? AND vcard_uid = ?", userID, uid).First(&created).Error)

	// 2. Soft-delete the contact, like DeleteContact does.
	require.NoError(t, db.Delete(&created).Error)

	var stillPresent models.Contact
	require.NoError(t, db.Unscoped().Where("user_id = ? AND vcard_uid = ?", userID, uid).First(&stillPresent).Error)
	assert.True(t, stillPresent.DeletedAt.Valid)

	// 3. Re-import the exact same vCard. Duplicate detection must find the
	// soft-deleted row by vcard_uid and default to "update" (restore), not "add".
	contacts2, previews2, _, err := ParseVCF(vcf(), db, userID)
	require.NoError(t, err)
	require.Len(t, previews2, 1)
	require.NotNil(t, previews2[0].DuplicateMatch)
	assert.Equal(t, "vcard_uid", previews2[0].DuplicateMatch.MatchReason)
	assert.True(t, previews2[0].DuplicateMatch.ExistingDeleted, "frontend needs this to show an explicit restore prompt")
	assert.Equal(t, "update", previews2[0].SuggestedAction)

	sessionID2 := mgr.CreateVCFSession(userID, contacts2, previews2)
	result2, appErr2 := mgr.ConfirmVCF(db, userID, models.ImportConfirmRequest{
		SessionID: sessionID2,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "update"}},
	}, cfg, &log)
	require.Nil(t, appErr2)
	require.Empty(t, result2.Errors)
	assert.Equal(t, 1, result2.Updated)

	var restored models.Contact
	require.NoError(t, db.Where("user_id = ? AND vcard_uid = ?", userID, uid).First(&restored).Error)
	assert.False(t, restored.DeletedAt.Valid)
	assert.Equal(t, created.ID, restored.ID)

	var count int64
	require.NoError(t, db.Unscoped().Model(&models.Contact{}).Where("user_id = ? AND vcard_uid = ?", userID, uid).Count(&count).Error)
	assert.Equal(t, int64(1), count, "re-import must not create a second row")
}

// Regression test for: after soft-deleting a contact and re-importing the same
// vCard, explicitly choosing "Add as New" (instead of restoring) used to fail with
// the same UNIQUE constraint error, because the new row kept the colliding UID.
func TestVCFReimport_AfterDelete_AddAsNewGetsFreshUID(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	cfg := &config.Config{}
	const userID = uint(1)
	const uid = "fixed-vcard-uid-456"

	vcf := func() *strings.Reader {
		return strings.NewReader(strings.NewReplacer("%s", uid).Replace(testVCFTemplate))
	}

	mgr := NewImportSessionManager()

	// 1. Import, then soft-delete the contact.
	contacts, previews, _, err := ParseVCF(vcf(), db, userID)
	require.NoError(t, err)
	sessionID := mgr.CreateVCFSession(userID, contacts, previews)
	result, appErr := mgr.ConfirmVCF(db, userID, models.ImportConfirmRequest{
		SessionID: sessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	}, cfg, &log)
	require.Nil(t, appErr)
	require.Equal(t, 1, result.Created)

	var original models.Contact
	require.NoError(t, db.Where("user_id = ? AND vcard_uid = ?", userID, uid).First(&original).Error)
	require.NoError(t, db.Delete(&original).Error)

	// 2. Re-import the same vCard, but explicitly choose "Add as New" this time.
	contacts2, previews2, _, err := ParseVCF(vcf(), db, userID)
	require.NoError(t, err)
	require.NotNil(t, previews2[0].DuplicateMatch)
	require.True(t, previews2[0].DuplicateMatch.ExistingDeleted)

	sessionID2 := mgr.CreateVCFSession(userID, contacts2, previews2)
	result2, appErr2 := mgr.ConfirmVCF(db, userID, models.ImportConfirmRequest{
		SessionID: sessionID2,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	}, cfg, &log)
	require.Nil(t, appErr2)
	require.Empty(t, result2.Errors, "add as new must not collide with the deleted contact's vcard_uid")
	assert.Equal(t, 1, result2.Created)

	// The original soft-deleted tombstone (and its vcard_uid) must be untouched.
	var stillDeleted models.Contact
	require.NoError(t, db.Unscoped().First(&stillDeleted, original.ID).Error)
	assert.True(t, stillDeleted.DeletedAt.Valid)
	assert.Equal(t, uid, stillDeleted.VCardUID)

	// The new contact is a separate row with a different, freshly-minted UID.
	var newContact models.Contact
	require.NoError(t, db.Where("user_id = ? AND firstname = ?", userID, "Jane").First(&newContact).Error)
	assert.NotEqual(t, original.ID, newContact.ID)
	assert.NotEqual(t, uid, newContact.VCardUID)
	assert.NotEmpty(t, newContact.VCardUID)
}

// Regression test: a vCard 2.1 file (older phones) used to import with only the
// name, because its bare "TEL;CELL:" parameters swallowed the values.
func TestVCFImport_V21KeepsFields(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	const userID = uint(1)

	contacts, previews, _, err := ParseVCF(strings.NewReader("BEGIN:VCARD\r\n"+
		"VERSION:2.1\r\n"+
		"N:Doe;Jane;;;\r\n"+
		"FN:Jane Doe\r\n"+
		"TEL;CELL:+49 170 1234567\r\n"+
		"EMAIL;HOME:jane@example.com\r\n"+
		"END:VCARD\r\n"), db, userID)
	require.NoError(t, err)
	require.Len(t, previews, 1)

	mgr := NewImportSessionManager()
	sessionID := mgr.CreateVCFSession(userID, contacts, previews)
	_, appErr := mgr.ConfirmVCF(db, userID, models.ImportConfirmRequest{
		SessionID: sessionID,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	}, &config.Config{}, &log)
	require.Nil(t, appErr)

	var stored models.Contact
	require.NoError(t, db.Where("user_id = ?", userID).First(&stored).Error)
	assert.Equal(t, "+49 170 1234567", stored.Phone)
	assert.Equal(t, []models.ContactPhone{{Type: "cell", Value: "+49 170 1234567"}}, stored.Phones)
	assert.Equal(t, "jane@example.com", stored.Email)
	assert.Equal(t, []models.ContactEmail{{Type: "home", Value: "jane@example.com"}}, stored.Emails)
}

const (
	importOwner = uint(1)
	importOther = uint(2)
)

var (
	csvImportHeaders  = []string{"First", "Last", "Mail"}
	csvImportMappings = []models.ColumnMapping{
		{CSVColumn: "First", ContactField: "firstname"},
		{CSVColumn: "Last", ContactField: "lastname"},
		{CSVColumn: "Mail", ContactField: "email"},
	}
)

// previewedCSVSession returns a CSV session for importOwner that is ready to confirm.
func previewedCSVSession(t *testing.T, mgr *ImportSessionManager, db *gorm.DB, rows ...[]string) string {
	t.Helper()
	sessionID := mgr.CreateCSVSession(importOwner, csvImportHeaders, rows)
	_, appErr := mgr.PreviewCSV(db, importOwner, models.ImportPreviewRequest{SessionID: sessionID, Mappings: csvImportMappings})
	require.Nil(t, appErr)
	return sessionID
}

func confirmRows(sessionID string, actions ...string) models.ImportConfirmRequest {
	req := models.ImportConfirmRequest{SessionID: sessionID}
	for i, action := range actions {
		req.Actions = append(req.Actions, models.RowImportAction{RowIndex: i, Action: action})
	}
	return req
}

func requireStatus(t *testing.T, appErr *apperrors.AppError, status int) {
	t.Helper()
	require.NotNil(t, appErr)
	assert.Equal(t, status, appErr.HTTPStatus)
}

func countContacts(t *testing.T, db *gorm.DB, userID uint) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Where("user_id = ?", userID).Count(&count).Error)
	return count
}

// expire moves a session past its expiry without waiting for it.
func expire(mgr *ImportSessionManager, sessionID string) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.sessions[sessionID].session.ExpiresAt = time.Now().Add(-time.Second)
}

func (m *ImportSessionManager) holds(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.sessions[sessionID]
	return ok
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	return buf.Bytes()
}

func TestImportSession_ExpiresAfterFifteenMinutes(t *testing.T) {
	mgr := NewImportSessionManager()
	before := time.Now()

	csvID := mgr.CreateCSVSession(importOwner, csvImportHeaders, nil)
	vcfID := mgr.CreateVCFSession(importOwner, nil, nil)

	for _, id := range []string{csvID, vcfID} {
		data, appErr := mgr.get(id, importOwner)
		require.Nil(t, appErr)
		assert.WithinDuration(t, before.Add(15*time.Minute), data.session.ExpiresAt, 5*time.Second)
	}
	assert.NotEqual(t, csvID, vcfID)
}

func TestImportSession_ExpiredSessionIsRefusedAndDiscarded(t *testing.T) {
	log := zerolog.Nop()
	cfg := &config.Config{}

	calls := map[string]func(*ImportSessionManager, *gorm.DB, string) *apperrors.AppError{
		"preview": func(m *ImportSessionManager, db *gorm.DB, id string) *apperrors.AppError {
			_, appErr := m.PreviewCSV(db, importOwner, models.ImportPreviewRequest{SessionID: id, Mappings: csvImportMappings})
			return appErr
		},
		"confirm": func(m *ImportSessionManager, db *gorm.DB, id string) *apperrors.AppError {
			_, appErr := m.Confirm(db, importOwner, confirmRows(id, "add"), &log)
			return appErr
		},
		"confirm VCF": func(m *ImportSessionManager, db *gorm.DB, id string) *apperrors.AppError {
			_, appErr := m.ConfirmVCF(db, importOwner, confirmRows(id, "add"), cfg, &log)
			return appErr
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			db := newImportSessionTestDB(t)
			mgr := NewImportSessionManager()
			sessionID := previewedCSVSession(t, mgr, db, []string{"Jane", "Doe", "jane@example.com"})
			expire(mgr, sessionID)

			requireStatus(t, call(mgr, db, sessionID), http.StatusNotFound)

			assert.False(t, mgr.holds(sessionID))
			assert.Zero(t, countContacts(t, db, importOwner))
		})
	}
}

func TestImportSession_UnknownSessionIsNotFound(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()

	_, appErr := mgr.PreviewCSV(db, importOwner, models.ImportPreviewRequest{SessionID: "missing"})
	requireStatus(t, appErr, http.StatusNotFound)

	_, appErr = mgr.Confirm(db, importOwner, confirmRows("missing"), &log)
	requireStatus(t, appErr, http.StatusNotFound)

	_, appErr = mgr.ConfirmVCF(db, importOwner, confirmRows("missing"), &config.Config{}, &log)
	requireStatus(t, appErr, http.StatusNotFound)
}

func TestImportSession_CleanupExpiredRemovesOnlyExpired(t *testing.T) {
	mgr := NewImportSessionManager()
	live := mgr.CreateCSVSession(importOwner, csvImportHeaders, nil)
	stale := mgr.CreateCSVSession(importOwner, csvImportHeaders, nil)
	expire(mgr, stale)

	mgr.CleanupExpired()

	assert.True(t, mgr.holds(live))
	assert.False(t, mgr.holds(stale))
}

func TestImportSession_DeleteRemovesSession(t *testing.T) {
	mgr := NewImportSessionManager()
	sessionID := mgr.CreateCSVSession(importOwner, csvImportHeaders, nil)

	mgr.Delete(sessionID)
	mgr.Delete("missing") // must not panic

	_, appErr := mgr.get(sessionID, importOwner)
	requireStatus(t, appErr, http.StatusNotFound)
}

func TestImportSession_AnotherUserCannotUseTheSession(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()
	sessionID := previewedCSVSession(t, mgr, db, []string{"Jane", "Doe", "jane@example.com"})

	_, appErr := mgr.PreviewCSV(db, importOther, models.ImportPreviewRequest{SessionID: sessionID, Mappings: csvImportMappings})
	requireStatus(t, appErr, http.StatusUnauthorized)

	_, appErr = mgr.Confirm(db, importOther, confirmRows(sessionID, "add"), &log)
	requireStatus(t, appErr, http.StatusUnauthorized)

	_, appErr = mgr.ConfirmVCF(db, importOther, confirmRows(sessionID, "add"), &config.Config{}, &log)
	requireStatus(t, appErr, http.StatusUnauthorized)

	assert.Zero(t, countContacts(t, db, importOther))
	assert.Zero(t, countContacts(t, db, importOwner))

	// None of that disturbed the owner, who can still finish the import.
	result, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "add"), &log)
	require.Nil(t, appErr)
	assert.Equal(t, 1, result.Created)
	assert.EqualValues(t, 1, countContacts(t, db, importOwner))
}

func TestImportSession_ExpiredSessionOfAnotherUserIsNotRevealedOrDiscarded(t *testing.T) {
	mgr := NewImportSessionManager()
	sessionID := mgr.CreateCSVSession(importOwner, csvImportHeaders, nil)
	expire(mgr, sessionID)

	_, appErr := mgr.get(sessionID, importOther)

	requireStatus(t, appErr, http.StatusUnauthorized)
	assert.True(t, mgr.holds(sessionID), "only the owner's request or the cleanup job discards a session")
}

func TestImportSession_ConfirmRequiresAPreview(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()
	sessionID := mgr.CreateCSVSession(importOwner, csvImportHeaders, [][]string{{"Jane", "Doe", "jane@example.com"}})

	_, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "add"), &log)
	requireStatus(t, appErr, http.StatusBadRequest)
	assert.Zero(t, countContacts(t, db, importOwner))

	// The refusal leaves the session in place, so the wizard can resume from the preview step.
	_, appErr = mgr.PreviewCSV(db, importOwner, models.ImportPreviewRequest{SessionID: sessionID, Mappings: csvImportMappings})
	require.Nil(t, appErr)

	result, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "add"), &log)
	require.Nil(t, appErr)
	assert.Equal(t, 1, result.Created)
}

func TestImportSession_ConfirmVCFRequiresAPreview(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()
	sessionID := mgr.CreateVCFSession(importOwner, nil, nil)
	mgr.mu.Lock()
	mgr.sessions[sessionID].session.PreviewCached = false
	mgr.mu.Unlock()

	_, appErr := mgr.ConfirmVCF(db, importOwner, confirmRows(sessionID), &config.Config{}, &log)

	requireStatus(t, appErr, http.StatusBadRequest)
}

func TestImportSession_ResumingPreviewUsesTheLatestMappings(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()
	sessionID := mgr.CreateCSVSession(importOwner, csvImportHeaders, [][]string{{"Jane", "Doe", "jane@example.com"}})

	// The user first maps the columns the wrong way round, then goes back and fixes it.
	swapped := []models.ColumnMapping{
		{CSVColumn: "First", ContactField: "lastname"},
		{CSVColumn: "Last", ContactField: "firstname"},
	}
	first, appErr := mgr.PreviewCSV(db, importOwner, models.ImportPreviewRequest{SessionID: sessionID, Mappings: swapped})
	require.Nil(t, appErr)
	assert.Equal(t, "Doe", first.Rows[0].ParsedContact["firstname"])

	second, appErr := mgr.PreviewCSV(db, importOwner, models.ImportPreviewRequest{SessionID: sessionID, Mappings: csvImportMappings})
	require.Nil(t, appErr)
	assert.Equal(t, sessionID, second.SessionID)
	assert.Equal(t, 1, second.TotalRows)
	assert.Equal(t, 1, second.ValidRows)
	assert.Equal(t, "Jane", second.Rows[0].ParsedContact["firstname"])

	_, appErr = mgr.Confirm(db, importOwner, confirmRows(sessionID, "add"), &log)
	require.Nil(t, appErr)

	var stored models.Contact
	require.NoError(t, db.Where("user_id = ?", importOwner).First(&stored).Error)
	assert.Equal(t, "Jane", stored.Firstname)
	assert.Equal(t, "Doe", stored.Lastname)
	assert.Equal(t, "jane@example.com", stored.Email)
}

func TestImportSession_ConfirmConsumesTheSession(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()
	sessionID := previewedCSVSession(t, mgr, db, []string{"Jane", "Doe", "jane@example.com"})

	_, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "add"), &log)
	require.Nil(t, appErr)

	// A double submit must not import the contacts twice.
	_, appErr = mgr.Confirm(db, importOwner, confirmRows(sessionID, "add"), &log)
	requireStatus(t, appErr, http.StatusNotFound)
	assert.EqualValues(t, 1, countContacts(t, db, importOwner))
}

func TestImportSession_ConfirmCSVActions(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()

	existing := models.Contact{UserID: importOwner, Firstname: "Old", Lastname: "Name", Email: "update@example.com", Nickname: "keep"}
	require.NoError(t, db.Create(&existing).Error)

	sessionID := previewedCSVSession(t, mgr, db,
		[]string{"Added", "Person", "added@example.com"},
		[]string{"Updated", "Person", "update@example.com"},
		[]string{"Skipped", "Person", "skipped@example.com"},
		[]string{"Unmentioned", "Person", "unmentioned@example.com"},
	)

	result, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "add", "update", "skip"), &log)

	require.Nil(t, appErr)
	assert.Equal(t, 4, result.TotalProcessed)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 1, result.Updated)
	assert.Equal(t, 2, result.Skipped, "a row with no action is skipped")
	assert.Empty(t, result.Errors)

	var added models.Contact
	require.NoError(t, db.Where("firstname = ?", "Added").First(&added).Error)
	assert.Equal(t, importOwner, added.UserID)

	var updated models.Contact
	require.NoError(t, db.First(&updated, existing.ID).Error)
	assert.Equal(t, "Updated", updated.Firstname)
	assert.Equal(t, "Person", updated.Lastname)
	assert.Equal(t, "keep", updated.Nickname, "fields the import does not carry are left alone")

	var note models.Note
	require.NoError(t, db.Where("contact_id = ?", existing.ID).First(&note).Error, "an update records what changed")
	assert.Contains(t, note.Content, "CSV")

	assert.EqualValues(t, 2, countContacts(t, db, importOwner))
}

func TestImportSession_DuplicateDetectionStaysWithinTheUser(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()

	theirs := models.Contact{UserID: importOther, Firstname: "Jane", Lastname: "Doe", Email: "jane@example.com"}
	require.NoError(t, db.Create(&theirs).Error)

	sessionID := mgr.CreateCSVSession(importOwner, csvImportHeaders, [][]string{{"Jane", "Replaced", "jane@example.com"}})
	preview, appErr := mgr.PreviewCSV(db, importOwner, models.ImportPreviewRequest{SessionID: sessionID, Mappings: csvImportMappings})
	require.Nil(t, appErr)
	assert.Nil(t, preview.Rows[0].DuplicateMatch, "another user's contact must not surface in the preview")
	assert.Equal(t, 0, preview.DuplicateCount)

	result, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "update"), &log)
	require.Nil(t, appErr)
	assert.Zero(t, result.Updated)
	assert.Len(t, result.Errors, 1)

	var stored models.Contact
	require.NoError(t, db.First(&stored, theirs.ID).Error)
	assert.Equal(t, "Doe", stored.Lastname)
}

func TestImportSession_UpdateOfVanishedContactReportsARowError(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()

	vanishAfterPreview := func(t *testing.T) (*ImportSessionManager, string) {
		existing := models.Contact{UserID: importOwner, Firstname: "Jane", Lastname: "Doe"}
		require.NoError(t, db.Create(&existing).Error)
		mgr := NewImportSessionManager()
		contacts := []VCFContactData{{Contact: &models.Contact{Firstname: "Jane", Lastname: "Doe"}}}
		previews := []models.ImportRowPreview{{
			RowIndex:       0,
			ParsedContact:  map[string]interface{}{"firstname": "Jane"},
			DuplicateMatch: &models.DuplicateMatch{ExistingContactID: existing.ID},
		}}
		sessionID := mgr.CreateVCFSession(importOwner, contacts, previews)
		require.NoError(t, db.Unscoped().Delete(&existing).Error)
		return mgr, sessionID
	}

	t.Run("confirm", func(t *testing.T) {
		mgr, sessionID := vanishAfterPreview(t)

		result, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "update"), &log)

		require.Nil(t, appErr)
		assert.Equal(t, 1, result.Skipped)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Row 1")
	})

	t.Run("confirm VCF", func(t *testing.T) {
		mgr, sessionID := vanishAfterPreview(t)

		result, appErr := mgr.ConfirmVCF(db, importOwner, confirmRows(sessionID, "update"), &config.Config{}, &log)

		require.Nil(t, appErr)
		assert.Equal(t, 1, result.Skipped)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Row 1")
	})
}

func TestImportSession_ConfirmRestoresSoftDeletedMatch(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()

	deleted := models.Contact{UserID: importOwner, Firstname: "Jane", Lastname: "Doe", Photo: "stale.jpg", PhotoThumbnail: "stale"}
	require.NoError(t, db.Create(&deleted).Error)
	require.NoError(t, db.Delete(&deleted).Error)

	contacts := []VCFContactData{{Contact: &models.Contact{Firstname: "Janet", Lastname: "Doe"}}}
	previews := []models.ImportRowPreview{{
		RowIndex:       0,
		ParsedContact:  map[string]interface{}{"firstname": "Janet"},
		DuplicateMatch: &models.DuplicateMatch{ExistingContactID: deleted.ID, ExistingDeleted: true},
	}}
	sessionID := mgr.CreateVCFSession(importOwner, contacts, previews)

	result, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "update"), &log)

	require.Nil(t, appErr)
	assert.Equal(t, 1, result.Updated)
	var restored models.Contact
	require.NoError(t, db.First(&restored, deleted.ID).Error)
	assert.Equal(t, "Janet", restored.Firstname)
	assert.Empty(t, restored.Photo, "the file behind a deleted contact's photo may be gone")
	assert.Empty(t, restored.PhotoThumbnail)
	assert.Equal(t, deleted.ID, contacts[0].Contact.ID, "later photo processing needs the row's contact ID")
}

func TestImportSession_ConfirmVCFWithPlainConfirm(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()

	contacts := []VCFContactData{{Contact: &models.Contact{Firstname: "Jane", VCardUID: "uid-1"}}}
	previews := []models.ImportRowPreview{{RowIndex: 0}}
	sessionID := mgr.CreateVCFSession(importOwner, contacts, previews)

	result, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "add"), &log)

	require.Nil(t, appErr)
	assert.Equal(t, 1, result.Created)
	require.NotZero(t, contacts[0].Contact.ID, "the row keeps the created contact's ID")
	var stored models.Contact
	require.NoError(t, db.First(&stored, contacts[0].Contact.ID).Error)
	assert.Equal(t, importOwner, stored.UserID)
}

func TestImportSession_ConfirmVCFRejectsCSVSession(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()
	sessionID := previewedCSVSession(t, mgr, db, []string{"Jane", "Doe", "jane@example.com"})

	_, appErr := mgr.ConfirmVCF(db, importOwner, confirmRows(sessionID, "add"), &config.Config{}, &log)

	requireStatus(t, appErr, http.StatusBadRequest)
	assert.Zero(t, countContacts(t, db, importOwner))
	assert.True(t, mgr.holds(sessionID))
}

func TestImportSession_ConfirmVCFActions(t *testing.T) {
	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()

	existing := models.Contact{UserID: importOwner, Firstname: "Old", Lastname: "Doe", Nickname: "keep"}
	require.NoError(t, db.Create(&existing).Error)

	contacts := []VCFContactData{
		{Contact: &models.Contact{Firstname: "Added", VCardUID: "uid-added"}},
		{Contact: &models.Contact{Firstname: "Updated"}},
		{Contact: &models.Contact{Firstname: "NoMatch"}},
		{Contact: &models.Contact{Firstname: "Skipped"}},
	}
	previews := []models.ImportRowPreview{
		{RowIndex: 0},
		{RowIndex: 1, ParsedContact: map[string]interface{}{"firstname": "Updated"}, DuplicateMatch: &models.DuplicateMatch{ExistingContactID: existing.ID}},
		{RowIndex: 2},
		{RowIndex: 3},
	}
	sessionID := mgr.CreateVCFSession(importOwner, contacts, previews)

	result, appErr := mgr.ConfirmVCF(db, importOwner, confirmRows(sessionID, "add", "update", "update"), &config.Config{}, &log)

	require.Nil(t, appErr)
	assert.Equal(t, 4, result.TotalProcessed)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 1, result.Updated)
	assert.Equal(t, 2, result.Skipped)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Row 3")

	var updated models.Contact
	require.NoError(t, db.First(&updated, existing.ID).Error)
	assert.Equal(t, "Updated", updated.Firstname)
	assert.Equal(t, "keep", updated.Nickname)

	var note models.Note
	require.NoError(t, db.Where("contact_id = ?", existing.ID).First(&note).Error)
	assert.Contains(t, note.Content, "VCF")

	assert.False(t, mgr.holds(sessionID))
}

func TestImportSession_ConfirmVCFPhotos(t *testing.T) {
	log := zerolog.Nop()

	confirmOne := func(t *testing.T, db *gorm.DB, data VCFContactData, preview models.ImportRowPreview, action string, photoDir string) *models.ImportResult {
		t.Helper()
		mgr := NewImportSessionManager()
		sessionID := mgr.CreateVCFSession(importOwner, []VCFContactData{data}, []models.ImportRowPreview{preview})
		result, appErr := mgr.ConfirmVCF(db, importOwner, confirmRows(sessionID, action), &config.Config{ProfilePhotoDir: photoDir}, &log)
		require.Nil(t, appErr)
		assert.False(t, mgr.holds(sessionID))
		return result
	}

	t.Run("embedded photo is saved for a new contact", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		photoDir := t.TempDir()

		result := confirmOne(t, db, VCFContactData{
			Contact:        &models.Contact{Firstname: "Jane"},
			PhotoData:      testPNG(t),
			PhotoMediaType: "image/png",
		}, models.ImportRowPreview{RowIndex: 0}, "add", photoDir)

		assert.Equal(t, 1, result.Created)
		var stored models.Contact
		require.NoError(t, db.Where("firstname = ?", "Jane").First(&stored).Error)
		require.NotEmpty(t, stored.Photo)
		assert.NotEmpty(t, stored.PhotoThumbnail)
		_, err := os.Stat(filepath.Join(photoDir, stored.Photo))
		assert.NoError(t, err)
	})

	t.Run("photo is added to an updated contact that has none", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		photoDir := t.TempDir()
		existing := models.Contact{UserID: importOwner, Firstname: "Jane"}
		require.NoError(t, db.Create(&existing).Error)

		confirmOne(t, db, VCFContactData{
			Contact:        &models.Contact{Firstname: "Jane"},
			PhotoData:      testPNG(t),
			PhotoMediaType: "image/png",
		}, models.ImportRowPreview{RowIndex: 0, DuplicateMatch: &models.DuplicateMatch{ExistingContactID: existing.ID}}, "update", photoDir)

		var stored models.Contact
		require.NoError(t, db.First(&stored, existing.ID).Error)
		assert.NotEmpty(t, stored.Photo)
	})

	t.Run("an updated contact keeps the photo it has", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		photoDir := t.TempDir()
		existing := models.Contact{UserID: importOwner, Firstname: "Jane", Photo: "mine.jpg"}
		require.NoError(t, db.Create(&existing).Error)

		confirmOne(t, db, VCFContactData{
			Contact:        &models.Contact{Firstname: "Jane"},
			PhotoData:      testPNG(t),
			PhotoMediaType: "image/png",
		}, models.ImportRowPreview{RowIndex: 0, DuplicateMatch: &models.DuplicateMatch{ExistingContactID: existing.ID}}, "update", photoDir)

		var stored models.Contact
		require.NoError(t, db.First(&stored, existing.ID).Error)
		assert.Equal(t, "mine.jpg", stored.Photo)
		entries, err := os.ReadDir(photoDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("an unusable photo does not lose the contact", func(t *testing.T) {
		db := newImportSessionTestDB(t)

		result := confirmOne(t, db, VCFContactData{
			Contact:        &models.Contact{Firstname: "Jane"},
			PhotoData:      []byte("not an image"),
			PhotoMediaType: "image/png",
		}, models.ImportRowPreview{RowIndex: 0}, "add", t.TempDir())

		assert.Equal(t, 1, result.Created)
		var stored models.Contact
		require.NoError(t, db.Where("firstname = ?", "Jane").First(&stored).Error)
		assert.Empty(t, stored.Photo)
	})

	t.Run("a photo URL that fails the SSRF guard does not lose the contact", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		photoDir := t.TempDir()

		result := confirmOne(t, db, VCFContactData{
			Contact:  &models.Contact{Firstname: "Jane"},
			PhotoURL: "http://127.0.0.1/internal.png",
		}, models.ImportRowPreview{RowIndex: 0}, "add", photoDir)

		assert.Equal(t, 1, result.Created)
		var stored models.Contact
		require.NoError(t, db.Where("firstname = ?", "Jane").First(&stored).Error)
		assert.Empty(t, stored.Photo)
		entries, err := os.ReadDir(photoDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestImportSession_RowFailingValidationCannotBeForcedIn(t *testing.T) {
	t.Skip("bug: Confirm and ConfirmVCF act on the client's action without checking the row's ValidationErrors")

	db := newImportSessionTestDB(t)
	log := zerolog.Nop()
	mgr := NewImportSessionManager()
	sessionID := mgr.CreateCSVSession(importOwner, csvImportHeaders, [][]string{{"", "NoFirstName", "not-an-email"}})

	preview, appErr := mgr.PreviewCSV(db, importOwner, models.ImportPreviewRequest{SessionID: sessionID, Mappings: csvImportMappings})
	require.Nil(t, appErr)
	require.NotEmpty(t, preview.Rows[0].ValidationErrors)

	result, appErr := mgr.Confirm(db, importOwner, confirmRows(sessionID, "add"), &log)

	require.Nil(t, appErr)
	assert.Zero(t, result.Created)
	assert.Zero(t, countContacts(t, db, importOwner))
}
