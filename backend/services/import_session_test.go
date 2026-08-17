package services

import (
	"strings"
	"testing"

	"meerkat/config"
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
		&models.User{}, &models.Contact{}, &models.Note{},
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
