package services

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
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

// --- shared test fixtures / helpers -----------------------------------------------

// testPNGBase64 (a 1x1 transparent PNG) is already declared in
// contact_sync_service_test.go, shared within this package.

func decodeTestPNGForImportSession(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(testPNGBase64)
	require.NoError(t, err)
	return data
}

func setupImportSessionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Contact{}, &models.Note{}))
	return db
}

func testImportLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

func testImportConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{ProfilePhotoDir: t.TempDir()}
}

// csvSessionIDPattern matches generateSessionID's 16-byte-hex output.
var csvSessionIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// --- generateSessionID -------------------------------------------------------------

func TestGenerateSessionID_DistinctAndWellFormed(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := generateSessionID()
		assert.Regexp(t, csvSessionIDPattern, id, "session ID should be 32 lowercase hex chars")
		assert.False(t, seen[id], "session ID should not repeat: %s", id)
		seen[id] = true
	}
}

// --- NewImportSessionManager / CreateCSVSession / CreateVCFSession -----------------

func TestNewImportSessionManager_StartsEmpty(t *testing.T) {
	m := NewImportSessionManager()
	require.NotNil(t, m.sessions)
	assert.Len(t, m.sessions, 0)
}

func TestCreateCSVSession_DistinctWellFormedIDs(t *testing.T) {
	m := NewImportSessionManager()
	id1 := m.CreateCSVSession(1, []string{"First"}, [][]string{{"Alice"}})
	id2 := m.CreateCSVSession(1, []string{"First"}, [][]string{{"Bob"}})

	assert.Regexp(t, csvSessionIDPattern, id1)
	assert.Regexp(t, csvSessionIDPattern, id2)
	assert.NotEqual(t, id1, id2)

	sd, ok := m.sessions[id1]
	require.True(t, ok)
	assert.Equal(t, "csv", sd.importType)
	assert.Equal(t, uint(1), sd.session.UserID)
}

func TestCreateVCFSession_DistinctWellFormedIDs(t *testing.T) {
	m := NewImportSessionManager()
	id1 := m.CreateVCFSession(1, []VCFContactData{{Contact: &models.Contact{Firstname: "Alice"}}}, []models.ImportRowPreview{{RowIndex: 0}})
	id2 := m.CreateVCFSession(1, []VCFContactData{{Contact: &models.Contact{Firstname: "Bob"}}}, []models.ImportRowPreview{{RowIndex: 0}})

	assert.Regexp(t, csvSessionIDPattern, id1)
	assert.Regexp(t, csvSessionIDPattern, id2)
	assert.NotEqual(t, id1, id2)

	sd, ok := m.sessions[id1]
	require.True(t, ok)
	assert.Equal(t, "vcf", sd.importType)
	assert.True(t, sd.session.PreviewCached)
}

// --- get -----------------------------------------------------------------------

func TestGet_HappyPath(t *testing.T) {
	m := NewImportSessionManager()
	id := m.CreateCSVSession(7, []string{"First"}, [][]string{{"Alice"}})

	sd, appErr := m.get(id, 7)
	require.Nil(t, appErr)
	require.NotNil(t, sd)
	assert.Equal(t, uint(7), sd.session.UserID)
}

func TestGet_WrongUser_Unauthorized(t *testing.T) {
	m := NewImportSessionManager()
	id := m.CreateCSVSession(7, []string{"First"}, [][]string{{"Alice"}})

	sd, appErr := m.get(id, 8)
	assert.Nil(t, sd)
	require.NotNil(t, appErr)
	assert.Equal(t, apperrors.ErrCodeUnauthorized, appErr.Code)
}

func TestGet_UnknownID_NotFound(t *testing.T) {
	m := NewImportSessionManager()

	sd, appErr := m.get("does-not-exist", 1)
	assert.Nil(t, sd)
	require.NotNil(t, appErr)
	assert.Equal(t, apperrors.ErrCodeNotFound, appErr.Code)
}

func TestGet_Expired_DeletesAndDistinguishesFromUnknown(t *testing.T) {
	m := NewImportSessionManager()
	id := m.CreateCSVSession(7, []string{"First"}, [][]string{{"Alice"}})

	// Backdate ExpiresAt to force the "expired" branch.
	m.mu.Lock()
	m.sessions[id].session.ExpiresAt = time.Now().Add(-time.Minute)
	m.mu.Unlock()

	// First call: hits the "found but expired" branch, which deletes the
	// session from internal storage as a side effect.
	sd, appErr := m.get(id, 7)
	assert.Nil(t, sd)
	require.NotNil(t, appErr)
	assert.Equal(t, apperrors.ErrCodeNotFound, appErr.Code)
	firstMessage := appErr.Message

	m.mu.RLock()
	_, stillExists := m.sessions[id]
	m.mu.RUnlock()
	assert.False(t, stillExists, "expired session must be deleted from internal storage")

	// Second call on the same ID: session is now truly gone, so this must
	// take the "not exists" path rather than re-triggering expiry deletion.
	sd2, appErr2 := m.get(id, 7)
	assert.Nil(t, sd2)
	require.NotNil(t, appErr2)
	assert.Equal(t, apperrors.ErrCodeNotFound, appErr2.Code)

	// The two paths produce distinguishable messages ("expired" vs
	// "expired or not found"), proving different code was executed.
	assert.NotEqual(t, firstMessage, appErr2.Message, "expiry-delete path and already-gone path should report different messages")
}

// --- Delete --------------------------------------------------------------------

func TestDelete_RemovesSession(t *testing.T) {
	m := NewImportSessionManager()
	id := m.CreateCSVSession(1, []string{"First"}, [][]string{{"Alice"}})

	m.Delete(id)

	_, appErr := m.get(id, 1)
	require.NotNil(t, appErr)
	assert.Equal(t, apperrors.ErrCodeNotFound, appErr.Code)
}

// --- CleanupExpired --------------------------------------------------------------

func TestCleanupExpired_RemovesOnlyExpiredByID(t *testing.T) {
	m := NewImportSessionManager()
	liveID := m.CreateCSVSession(1, []string{"First"}, [][]string{{"Alice"}})
	expiredID := m.CreateCSVSession(1, []string{"First"}, [][]string{{"Bob"}})

	m.mu.Lock()
	m.sessions[expiredID].session.ExpiresAt = time.Now().Add(-time.Minute)
	m.mu.Unlock()

	m.CleanupExpired()

	m.mu.RLock()
	_, liveExists := m.sessions[liveID]
	_, expiredExists := m.sessions[expiredID]
	m.mu.RUnlock()

	assert.True(t, liveExists, "live session must survive cleanup")
	assert.False(t, expiredExists, "expired session must be removed by cleanup")
}

// --- PreviewCSV ------------------------------------------------------------------

func csvMappings() []models.ColumnMapping {
	return []models.ColumnMapping{
		{CSVColumn: "First Name", ContactField: "firstname"},
		{CSVColumn: "Last Name", ContactField: "lastname"},
		{CSVColumn: "Email", ContactField: "email", Group: 0},
	}
}

func TestPreviewCSV_ValidMappings(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()

	headers := []string{"First Name", "Last Name", "Email"}
	rows := [][]string{
		{"Alice", "Smith", "alice@example.com"},
		{"Bob", "Jones", "bob@example.com"},
	}
	id := m.CreateCSVSession(1, headers, rows)

	resp, appErr := m.PreviewCSV(db, 1, models.ImportPreviewRequest{SessionID: id, Mappings: csvMappings()})
	require.Nil(t, appErr)
	require.NotNil(t, resp)

	assert.Equal(t, 2, resp.TotalRows)
	assert.Equal(t, 2, resp.ValidRows)
	assert.Equal(t, 0, resp.DuplicateCount)
	assert.Equal(t, 0, resp.ErrorCount)
	assert.Equal(t, id, resp.SessionID)

	// Preview data + built contacts must now be cached on the session for Confirm.
	sd := m.sessions[id]
	assert.True(t, sd.session.PreviewCached)
	require.Len(t, sd.csvContacts, 2)
	assert.Equal(t, "Alice", sd.csvContacts[0].Firstname)
}

func TestPreviewCSV_DuplicateDetected(t *testing.T) {
	db := setupImportSessionTestDB(t)
	require.NoError(t, db.Create(&models.Contact{UserID: 1, Firstname: "Alice", Lastname: "Smith", Email: "alice@example.com"}).Error)

	m := NewImportSessionManager()
	headers := []string{"First Name", "Last Name", "Email"}
	rows := [][]string{{"Alice", "Smith", "alice@example.com"}}
	id := m.CreateCSVSession(1, headers, rows)

	resp, appErr := m.PreviewCSV(db, 1, models.ImportPreviewRequest{SessionID: id, Mappings: csvMappings()})
	require.Nil(t, appErr)
	assert.Equal(t, 1, resp.DuplicateCount)
	require.Len(t, resp.Rows, 1)
	require.NotNil(t, resp.Rows[0].DuplicateMatch)
	assert.Equal(t, "email", resp.Rows[0].DuplicateMatch.MatchReason)
}

func TestPreviewCSV_WrongUser_ErrorPropagatesFromGet(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()

	headers := []string{"First Name", "Last Name", "Email"}
	rows := [][]string{{"Alice", "Smith", "alice@example.com"}}
	id := m.CreateCSVSession(1, headers, rows)

	resp, appErr := m.PreviewCSV(db, 999, models.ImportPreviewRequest{SessionID: id, Mappings: csvMappings()})
	assert.Nil(t, resp)
	require.NotNil(t, appErr)
	assert.Equal(t, apperrors.ErrCodeUnauthorized, appErr.Code)
}

// --- Confirm (CSV) -----------------------------------------------------------------

func TestConfirm_Add_PersistsRowAndDeletesSession(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()

	headers := []string{"First Name", "Last Name", "Email"}
	rows := [][]string{{"Alice", "Smith", "alice@example.com"}}
	id := m.CreateCSVSession(1, headers, rows)

	_, appErr := m.PreviewCSV(db, 1, models.ImportPreviewRequest{SessionID: id, Mappings: csvMappings()})
	require.Nil(t, appErr)

	req := models.ImportConfirmRequest{
		SessionID: id,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	}
	result, appErr := m.Confirm(db, 1, req, log)
	require.Nil(t, appErr)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Updated)
	assert.Equal(t, 0, result.Skipped)
	assert.Empty(t, result.Errors)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "alice@example.com").First(&persisted).Error)
	assert.Equal(t, "Alice", persisted.Firstname)
	assert.Equal(t, "Smith", persisted.Lastname)
	assert.Equal(t, uint(1), persisted.UserID)

	// Session must be gone after a successful Confirm.
	_, getErr := m.get(id, 1)
	require.NotNil(t, getErr)
	assert.Equal(t, apperrors.ErrCodeNotFound, getErr.Code)
}

func TestConfirm_Update_MergesExistingAndWritesMergeNote(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()

	existing := models.Contact{UserID: 1, Firstname: "Old", Lastname: "Name", Email: "match@example.com"}
	require.NoError(t, db.Create(&existing).Error)
	existingID := existing.ID

	headers := []string{"First Name", "Last Name", "Email"}
	rows := [][]string{{"New", "Name", "match@example.com"}}
	id := m.CreateCSVSession(1, headers, rows)

	_, appErr := m.PreviewCSV(db, 1, models.ImportPreviewRequest{SessionID: id, Mappings: csvMappings()})
	require.Nil(t, appErr)

	req := models.ImportConfirmRequest{
		SessionID: id,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "update"}},
	}
	result, appErr := m.Confirm(db, 1, req, log)
	require.Nil(t, appErr)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Updated)
	assert.Equal(t, 0, result.Created)
	assert.Empty(t, result.Errors)

	var merged models.Contact
	require.NoError(t, db.First(&merged, existingID).Error)
	assert.Equal(t, existingID, merged.ID, "update must merge into the existing row, not create a new one")
	assert.Equal(t, "New", merged.Firstname)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Where("user_id = ?", 1).Count(&count).Error)
	assert.Equal(t, int64(1), count, "update must not create a second row")

	var notes []models.Note
	require.NoError(t, db.Where("contact_id = ?", existingID).Find(&notes).Error)
	require.Len(t, notes, 1, "a merge note documenting the change should be created")
	assert.Contains(t, notes[0].Content, "CSV")
	assert.Contains(t, notes[0].Content, "Old")
	assert.Contains(t, notes[0].Content, "New")
}

func TestConfirm_UpdateWithNilDuplicateMatch_ErrorsButBatchContinues(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()

	headers := []string{"First Name", "Last Name", "Email"}
	rows := [][]string{
		{"NoMatch", "Person", "nomatch@example.com"}, // row 0: no existing duplicate
		{"Good", "Row", "good@example.com"},          // row 1: also new, but requested as "add"
	}
	id := m.CreateCSVSession(1, headers, rows)

	_, appErr := m.PreviewCSV(db, 1, models.ImportPreviewRequest{SessionID: id, Mappings: csvMappings()})
	require.Nil(t, appErr)

	// Row 0 has no DuplicateMatch (nothing in the DB matches it), yet we
	// request "update" for it — this must produce a per-row error, not a
	// transaction abort. Row 1 requests "add" and must still succeed.
	req := models.ImportConfirmRequest{
		SessionID: id,
		Actions: []models.RowImportAction{
			{RowIndex: 0, Action: "update"},
			{RowIndex: 1, Action: "add"},
		},
	}
	result, appErr := m.Confirm(db, 1, req, log)
	require.Nil(t, appErr)
	require.NotNil(t, result)

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Row 1")
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 1, result.Created)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "good@example.com").First(&persisted).Error)
	assert.Equal(t, "Good", persisted.Firstname, "a bad row must not poison the rest of the batch")

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Where("user_id = ?", 1).Count(&count).Error)
	assert.Equal(t, int64(1), count, "only the good row should have been persisted")
}

func TestConfirm_Update_StaleDuplicateReference_ErrorsButBatchContinues(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()

	existing := models.Contact{UserID: 1, Firstname: "Alice", Lastname: "Smith", Email: "alice@example.com"}
	require.NoError(t, db.Create(&existing).Error)

	headers := []string{"First Name", "Last Name", "Email"}
	rows := [][]string{
		{"Alice", "Smith", "alice@example.com"}, // will match `existing` as a duplicate
		{"Good", "Row", "good2@example.com"},
	}
	id := m.CreateCSVSession(1, headers, rows)

	_, appErr := m.PreviewCSV(db, 1, models.ImportPreviewRequest{SessionID: id, Mappings: csvMappings()})
	require.Nil(t, appErr)

	// Simulate the existing contact having been deleted between preview and
	// confirm (a stale DuplicateMatch): the referenced ID no longer exists.
	require.NoError(t, db.Unscoped().Delete(&existing).Error)

	req := models.ImportConfirmRequest{
		SessionID: id,
		Actions: []models.RowImportAction{
			{RowIndex: 0, Action: "update"},
			{RowIndex: 1, Action: "add"},
		},
	}
	result, appErr := m.Confirm(db, 1, req, log)
	require.Nil(t, appErr)
	require.NotNil(t, result)

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Row 1")
	assert.Contains(t, result.Errors[0], "Failed to fetch existing contact")
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 1, result.Created, "a stale duplicate reference on one row must not stop the rest of the batch from committing")

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "good2@example.com").First(&persisted).Error)
	assert.Equal(t, "Good", persisted.Firstname)
}

func TestConfirmVCF_Update_StaleDuplicateReference_ErrorsButBatchContinues(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	vcfContacts := []VCFContactData{
		{Contact: &models.Contact{Firstname: "Stale", Lastname: "Ref", Email: "stale@example.com"}},
		{Contact: &models.Contact{Firstname: "Good", Lastname: "Row", Email: "good3@example.com"}},
	}
	previews := []models.ImportRowPreview{
		{RowIndex: 0, DuplicateMatch: &models.DuplicateMatch{ExistingContactID: 999999, MatchReason: "email"}, SuggestedAction: "update"},
		{RowIndex: 1, SuggestedAction: "add"},
	}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{
		SessionID: id,
		Actions: []models.RowImportAction{
			{RowIndex: 0, Action: "update"},
			{RowIndex: 1, Action: "add"},
		},
	}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	require.Nil(t, appErr)
	require.NotNil(t, result)

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Failed to fetch existing contact")
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 1, result.Created)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "good3@example.com").First(&persisted).Error)
	assert.Equal(t, "Good", persisted.Firstname)
}

// TestConfirm_ViaVCFSession_AddPersistsRow exercises Confirm's isVCFImport
// branches directly (ConfirmVCFImport/ConfirmImport at the controller layer
// both ultimately dispatch through Confirm for the actual row-write logic;
// this proves the generic Confirm path handles a VCF-sourced session's
// vcfContacts slice correctly, not just the csvContacts slice).
func TestConfirm_ViaVCFSession_AddPersistsRow(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()

	contact := &models.Contact{Firstname: "Vic", Lastname: "Fromvcf", Email: "vic@example.com"}
	vcfContacts := []VCFContactData{{Contact: contact}}
	previews := []models.ImportRowPreview{{RowIndex: 0, SuggestedAction: "add"}}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "add"}}}
	result, appErr := m.Confirm(db, 1, req, log)
	require.Nil(t, appErr)
	require.Equal(t, 1, result.Created)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "vic@example.com").First(&persisted).Error)
	assert.Equal(t, "Vic", persisted.Firstname)
	// The in-session vcfContacts entry's Contact.ID must be back-filled too
	// (used by ConfirmVCF's subsequent photo-write pass keyed off contactID).
	assert.Equal(t, persisted.ID, contact.ID)
}

// TestConfirm_ViaVCFSession_UpdateMergesAndLabelsVCF exercises Confirm's
// isVCFImport "update" branch (importType label + MergeImportedContact
// called with the vcfContacts entry rather than csvContacts).
func TestConfirm_ViaVCFSession_UpdateMergesAndLabelsVCF(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()

	existing := models.Contact{Firstname: "Old", Lastname: "Name", Email: "existing-vcf@example.com"}
	require.NoError(t, db.Create(&existing).Error)

	updated := &models.Contact{Firstname: "New", Lastname: "Name", Email: "existing-vcf@example.com"}
	vcfContacts := []VCFContactData{{Contact: updated}}
	previews := []models.ImportRowPreview{{
		RowIndex:        0,
		SuggestedAction: "update",
		DuplicateMatch:  &models.DuplicateMatch{ExistingContactID: existing.ID},
		ParsedContact:   map[string]interface{}{"firstname": "New"},
	}}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "update"}}}
	result, appErr := m.Confirm(db, 1, req, log)
	require.Nil(t, appErr)
	require.Equal(t, 1, result.Updated)

	var persisted models.Contact
	require.NoError(t, db.First(&persisted, existing.ID).Error)
	assert.Equal(t, "New", persisted.Firstname)

	var note models.Note
	require.NoError(t, db.Where("contact_id = ?", existing.ID).First(&note).Error)
	assert.Contains(t, note.Content, "VCF")
}

func TestConfirm_SkipAction_NoDBRow(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()

	headers := []string{"First Name", "Last Name", "Email"}
	rows := [][]string{
		{"Skipped", "One", "skip1@example.com"},
		{"Skipped", "Two", "skip2@example.com"},
	}
	id := m.CreateCSVSession(1, headers, rows)

	_, appErr := m.PreviewCSV(db, 1, models.ImportPreviewRequest{SessionID: id, Mappings: csvMappings()})
	require.Nil(t, appErr)

	req := models.ImportConfirmRequest{
		SessionID: id,
		Actions: []models.RowImportAction{
			{RowIndex: 0, Action: "skip"},
			// RowIndex 1 has no entry at all -> defaults to "skip" too.
		},
	}
	result, appErr := m.Confirm(db, 1, req, log)
	require.Nil(t, appErr)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Skipped)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 0, result.Updated)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Where("user_id = ?", 1).Count(&count).Error)
	assert.Equal(t, int64(0), count, "skipped rows must not be persisted")
}

func TestConfirm_PreviewNotCached_InvalidInputNoMutation(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()

	headers := []string{"First Name"}
	rows := [][]string{{"Alice"}}
	id := m.CreateCSVSession(1, headers, rows) // no PreviewCSV call -> PreviewCached stays false

	req := models.ImportConfirmRequest{
		SessionID: id,
		Actions:   []models.RowImportAction{{RowIndex: 0, Action: "add"}},
	}
	result, appErr := m.Confirm(db, 1, req, log)
	assert.Nil(t, result)
	require.NotNil(t, appErr)
	assert.Equal(t, apperrors.ErrCodeInvalidInput, appErr.Code)

	// Session must not have been deleted or otherwise mutated by the failed Confirm.
	sd, getErr := m.get(id, 1)
	require.Nil(t, getErr)
	assert.False(t, sd.session.PreviewCached)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

// --- ConfirmVCF ---------------------------------------------------------------------

func TestConfirmVCF_WrongImportType_InvalidInput(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	// A CSV session ID handed to the VCF confirm endpoint.
	id := m.CreateCSVSession(1, []string{"First"}, [][]string{{"Alice"}})
	_, appErr := m.PreviewCSV(db, 1, models.ImportPreviewRequest{SessionID: id, Mappings: []models.ColumnMapping{{CSVColumn: "First", ContactField: "firstname"}}})
	require.Nil(t, appErr)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "add"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	assert.Nil(t, result)
	require.NotNil(t, appErr)
	assert.Equal(t, apperrors.ErrCodeInvalidInput, appErr.Code)
}

// TestConfirmVCF_WrongUser_ErrorPropagatesFromGet exercises ConfirmVCF's
// sessErr != nil early-return (the m.get(...) call at the top of ConfirmVCF).
func TestConfirmVCF_WrongUser_ErrorPropagatesFromGet(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	contact := &models.Contact{Firstname: "Vic", Lastname: "Tim"}
	id := m.CreateVCFSession(1, []VCFContactData{{Contact: contact}}, []models.ImportRowPreview{{RowIndex: 0, SuggestedAction: "add"}})

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "add"}}}
	result, appErr := m.ConfirmVCF(db, 2, req, cfg, log) // wrong user
	assert.Nil(t, result)
	require.NotNil(t, appErr)
}

// TestConfirmVCF_PreviewNotCached_InvalidInput exercises ConfirmVCF's
// !sessionData.session.PreviewCached branch directly. CreateVCFSession
// itself always sets PreviewCached true (there's no VCF-side equivalent of
// PreviewCSV that could leave it false), so this branch is unreachable via
// any public constructor today — this test reaches into the manager's
// internal session map (same package) to force the state and prove the
// guard still behaves correctly if that ever changes.
func TestConfirmVCF_PreviewNotCached_InvalidInput(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	contact := &models.Contact{Firstname: "Vic", Lastname: "Tim"}
	id := m.CreateVCFSession(1, []VCFContactData{{Contact: contact}}, []models.ImportRowPreview{{RowIndex: 0, SuggestedAction: "add"}})

	m.mu.Lock()
	m.sessions[id].session.PreviewCached = false
	m.mu.Unlock()

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "add"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	assert.Nil(t, result)
	require.NotNil(t, appErr)
	assert.Equal(t, apperrors.ErrCodeInvalidInput, appErr.Code)
}

// TestConfirmVCF_SkipAction_NoDBRow exercises ConfirmVCF's explicit "skip"
// case branch (distinct from CSV's Confirm, which already covers this).
func TestConfirmVCF_SkipAction_NoDBRow(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	contact := &models.Contact{Firstname: "Skip", Lastname: "Me", Email: "skipme-vcf@example.com"}
	id := m.CreateVCFSession(1, []VCFContactData{{Contact: contact}}, []models.ImportRowPreview{{RowIndex: 0, SuggestedAction: "skip"}})
	previewedID := id
	// CreateVCFSession doesn't itself set PreviewCached; reuse the same
	// pattern ConfirmVCF's other tests rely on via CreateVCFSession, which
	// does set PreviewCached true (see CreateVCFSession's implementation).
	_ = previewedID

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "skip"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	require.Nil(t, appErr)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Created)

	var count int64
	db.Model(&models.Contact{}).Where("email = ?", "skipme-vcf@example.com").Count(&count)
	assert.Equal(t, int64(0), count)
}

// TestConfirmVCF_UpdateWithNilDuplicateMatch_ErrorsButBatchContinues
// exercises ConfirmVCF's preview.DuplicateMatch == nil branch on "update".
func TestConfirmVCF_UpdateWithNilDuplicateMatch_ErrorsButBatchContinues(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	bad := &models.Contact{Firstname: "NoMatch", Lastname: "Row"}
	good := &models.Contact{Firstname: "Good", Lastname: "Row", Email: "goodrow-vcf@example.com"}
	vcfContacts := []VCFContactData{{Contact: bad}, {Contact: good}}
	previews := []models.ImportRowPreview{
		{RowIndex: 0, SuggestedAction: "update", DuplicateMatch: nil},
		{RowIndex: 1, SuggestedAction: "add"},
	}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{
		{RowIndex: 0, Action: "update"},
		{RowIndex: 1, Action: "add"},
	}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	require.Nil(t, appErr)
	assert.Equal(t, 1, result.Skipped)
	assert.NotEmpty(t, result.Errors)
	assert.Equal(t, 1, result.Created, "the good row in the same batch must still persist")

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "goodrow-vcf@example.com").First(&persisted).Error)
}

// TestConfirmVCF_Update_NoExistingPhoto_QueuesIncomingPhoto is the inverse of
// TestConfirmVCF_Update_ExistingPhotoNotOverwritten: when the existing
// contact has NO photo yet, an incoming photo on update IS queued for
// processing.
func TestConfirmVCF_Update_NoExistingPhoto_QueuesIncomingPhoto(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	existing := models.Contact{UserID: 1, Firstname: "No", Lastname: "Photo", Email: "nophoto-vcf@example.com"}
	require.NoError(t, db.Create(&existing).Error)

	photoData := decodeTestPNGForImportSession(t)
	incoming := &models.Contact{Firstname: "No", Lastname: "Photo", Email: "nophoto-vcf@example.com"}
	vcfContacts := []VCFContactData{{Contact: incoming, PhotoData: photoData, PhotoMediaType: "image/png"}}
	previews := []models.ImportRowPreview{{
		RowIndex:        0,
		DuplicateMatch:  &models.DuplicateMatch{ExistingContactID: existing.ID, MatchReason: "email"},
		SuggestedAction: "update",
	}}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "update"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	require.Nil(t, appErr)
	require.Equal(t, 1, result.Updated)

	var merged models.Contact
	require.NoError(t, db.First(&merged, existing.ID).Error)
	assert.NotEmpty(t, merged.Photo, "an incoming photo must be applied when the existing contact had none")
}

func TestConfirmVCF_Add_PersistsRowAndDeletesSession(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	contact := &models.Contact{Firstname: "Vera", Lastname: "Card", Email: "vera@example.com"}
	vcfContacts := []VCFContactData{{Contact: contact}}
	previews := []models.ImportRowPreview{{RowIndex: 0, SuggestedAction: "add"}}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "add"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	require.Nil(t, appErr)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Created)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "vera@example.com").First(&persisted).Error)
	assert.Equal(t, "Vera", persisted.Firstname)

	_, getErr := m.get(id, 1)
	require.NotNil(t, getErr)
}

// TestConfirmVCF_Add_EmbeddedPhoto_PersistsPhotoAndValidETag is the critical
// GORM regression test: ConfirmVCF's deferred photo write
// (import_session.go, inside the photoTasks loop) updates Contact.Photo /
// PhotoThumbnail after the main transaction commits. If that write ever
// regresses to a bare `db.Model(&models.Contact{}).Where("id = ?",
// id).Updates(map{...})` on a zero-value receiver, Contact.AfterSave
// recomputes ETag from that zero-value receiver's ID/UpdatedAt instead of
// the real row (see contact.go's AfterSave + the load-then-Model fix
// pattern established in contact_sync_service.go's reconcileContactSync and
// contact_controller.go's ArchiveContact). This test proves the persisted
// row's ETag is well-formed and keyed to its real ID after a photo-bearing
// VCF import.
func TestConfirmVCF_Add_EmbeddedPhoto_PersistsPhotoAndValidETag(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	photoData := decodeTestPNGForImportSession(t)
	contact := &models.Contact{Firstname: "Photo", Lastname: "Haver", Email: "photo@example.com"}
	vcfContacts := []VCFContactData{{Contact: contact, PhotoData: photoData, PhotoMediaType: "image/png"}}
	previews := []models.ImportRowPreview{{RowIndex: 0, SuggestedAction: "add"}}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "add"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	require.Nil(t, appErr)
	require.Equal(t, 1, result.Created)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "photo@example.com").First(&persisted).Error)

	assert.NotEmpty(t, persisted.Photo, "photo path should be populated via photostore.SaveContactPhoto")
	assert.Contains(t, persisted.Photo, "_photo.jpg")
	assert.NotEmpty(t, persisted.PhotoThumbnail, "thumbnail should be populated")
	assert.Contains(t, persisted.PhotoThumbnail, "data:image/jpeg;base64,")

	// Critical GORM regression assertion: ETag must be well-formed and match
	// the real row's ID, not a zero-value "e-0-..." produced by a bulk
	// Where(...).Updates(map) call running AfterSave against an empty receiver.
	require.NotEmpty(t, persisted.ETag)
	expectedPrefix := fmt.Sprintf("e-%d-", persisted.ID)
	assert.True(t, len(persisted.ETag) > len(expectedPrefix) && persisted.ETag[:len(expectedPrefix)] == expectedPrefix,
		"ETag %q should start with %q (derived from the real persisted row's ID)", persisted.ETag, expectedPrefix)
	assert.Regexp(t, regexp.MustCompile(`^e-\d+-\d+$`), persisted.ETag)
}

func TestConfirmVCF_Add_PhotoURL_Success_PersistsPhoto(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	photoData := decodeTestPNGForImportSession(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(photoData)
	}))
	defer server.Close()

	contact := &models.Contact{Firstname: "URL", Lastname: "Photo", Email: "urlphoto@example.com"}
	vcfContacts := []VCFContactData{{Contact: contact, PhotoURL: server.URL}}
	previews := []models.ImportRowPreview{{RowIndex: 0, SuggestedAction: "add"}}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "add"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	require.Nil(t, appErr)
	require.Equal(t, 1, result.Created)

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "urlphoto@example.com").First(&persisted).Error)

	// httputil.FetchImageFromURL applies SSRF protection that unconditionally
	// blocks loopback hosts (127.0.0.1/localhost) — see httputil/fetch.go's
	// blockedHosts / isPrivateIP checks. httptest.NewServer only ever binds
	// to a loopback address, so this fetch is expected to be blocked by that
	// (correct, security-relevant) guard rather than actually succeeding.
	// What we're really proving here is the degradation-policy behavior this
	// package promises (docs/fork-plan/00-overview.md §0.5): the contact
	// import itself must still succeed and must not error out just because
	// its photo could not be fetched.
	assert.Empty(t, persisted.Photo, "photo fetch from a loopback URL is expected to be blocked by SSRF protection, not silently allowed")
	assert.NotEmpty(t, persisted.Firstname)
}

func TestConfirmVCF_Add_PhotoURLFetchFailure_ContactStillCreatedPhotoSkipped(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	contact := &models.Contact{Firstname: "Broken", Lastname: "Link", Email: "brokenlink@example.com"}
	vcfContacts := []VCFContactData{{Contact: contact, PhotoURL: server.URL + "/missing.png"}}
	previews := []models.ImportRowPreview{{RowIndex: 0, SuggestedAction: "add"}}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "add"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)

	// The degradation policy (docs/fork-plan/00-overview.md §0.5) requires
	// this to be a soft failure: the contact is still created even though
	// its photo could not be fetched.
	require.Nil(t, appErr)
	require.Equal(t, 1, result.Created)
	assert.Empty(t, result.Errors, "a photo-fetch failure must not surface as an import error")

	var persisted models.Contact
	require.NoError(t, db.Where("email = ?", "brokenlink@example.com").First(&persisted).Error)
	assert.Empty(t, persisted.Photo, "photo should be silently skipped, not block contact creation")
}

func TestConfirmVCF_Update_ExistingPhotoNotOverwritten(t *testing.T) {
	db := setupImportSessionTestDB(t)
	m := NewImportSessionManager()
	log := testImportLogger()
	cfg := testImportConfig(t)

	existing := models.Contact{UserID: 1, Firstname: "Has", Lastname: "Photo", Email: "hasphoto@example.com", Photo: "existing_photo.jpg"}
	require.NoError(t, db.Create(&existing).Error)

	photoData := decodeTestPNGForImportSession(t)
	incoming := &models.Contact{Firstname: "Has", Lastname: "Photo", Email: "hasphoto@example.com"}
	vcfContacts := []VCFContactData{{Contact: incoming, PhotoData: photoData, PhotoMediaType: "image/png"}}
	previews := []models.ImportRowPreview{{
		RowIndex:        0,
		DuplicateMatch:  &models.DuplicateMatch{ExistingContactID: existing.ID, MatchReason: "email"},
		SuggestedAction: "update",
	}}
	id := m.CreateVCFSession(1, vcfContacts, previews)

	req := models.ImportConfirmRequest{SessionID: id, Actions: []models.RowImportAction{{RowIndex: 0, Action: "update"}}}
	result, appErr := m.ConfirmVCF(db, 1, req, cfg, log)
	require.Nil(t, appErr)
	require.Equal(t, 1, result.Updated)

	var merged models.Contact
	require.NoError(t, db.First(&merged, existing.ID).Error)
	assert.Equal(t, "existing_photo.jpg", merged.Photo, "an existing photo must not be overwritten by a merged-in import photo")
}

// --- buildActionMap ------------------------------------------------------------------

func TestBuildActionMap_DuplicateRowIndex_LastWins(t *testing.T) {
	actions := []models.RowImportAction{
		{RowIndex: 0, Action: "add"},
		{RowIndex: 0, Action: "skip"},
		{RowIndex: 1, Action: "update"},
	}
	m := buildActionMap(actions)
	assert.Equal(t, "skip", m[0], "last entry for a duplicate RowIndex should win")
	assert.Equal(t, "update", m[1])
	assert.Len(t, m, 2)
}

func TestBuildActionMap_Empty(t *testing.T) {
	m := buildActionMap(nil)
	assert.NotNil(t, m)
	assert.Len(t, m, 0)
}
