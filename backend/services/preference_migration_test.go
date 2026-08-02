package services

import (
	"path/filepath"
	"testing"

	"mycorrhizal/database"
	"mycorrhizal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupPreferenceMigrationTestDB uses database.InitDB (real migrated schema,
// per CLAUDE.md trap 1) rather than AutoMigrate: the migration reads the raw
// legacy contacts.food_preference column (removed from models.Contact this
// ticket), which only exists in the real 000001 migration SQL — and it
// writes models.Preference against the real 000040 schema, exercising the
// entity_id column mapping GORM would otherwise derive silently.
func setupPreferenceMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "pref-migrate.db"))
	require.NoError(t, err)
	return db
}

func createMigrationTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	user := models.User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

// createLegacyFoodContact creates a contact via the GORM model (which no
// longer carries FoodPreference), then writes the legacy column directly —
// simulating the pre-migration row the backfill exists to read.
func createLegacyFoodContact(t *testing.T, db *gorm.DB, userID uint, firstname, food string) LegacyFoodContact {
	t.Helper()
	contact := models.Contact{UserID: userID, Firstname: firstname}
	require.NoError(t, db.Create(&contact).Error)
	require.NotEmpty(t, contact.VCardUID, "a saved contact must carry a VCardUID (BeforeCreate)")
	if food != "" {
		require.NoError(t, db.Model(&contact).Update("food_preference", food).Error)
	}
	return LegacyFoodContact{ID: contact.ID, UserID: userID, VCardUID: contact.VCardUID, FoodPreference: food}
}

func preferenceCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&models.Preference{}).Count(&count).Error)
	return count
}

func TestMigrateContactFoodPreference_DryRunMakesNoWrites(t *testing.T) {
	db := setupPreferenceMigrationTestDB(t)
	user := createMigrationTestUser(t, db)
	contact := createLegacyFoodContact(t, db, user.ID, "Alice", "Vegetarian")

	outcome, err := MigrateContactFoodPreference(db, contact, false)
	require.NoError(t, err)
	assert.Empty(t, outcome.Skipped)
	assert.Empty(t, outcome.PreferenceID)

	assert.Zero(t, preferenceCount(t, db), "dry run must make zero writes")
}

func TestMigrateContactFoodPreference_WriteCreatesPreference(t *testing.T) {
	db := setupPreferenceMigrationTestDB(t)
	user := createMigrationTestUser(t, db)
	contact := createLegacyFoodContact(t, db, user.ID, "Alice", "Vegetarian")

	outcome, err := MigrateContactFoodPreference(db, contact, true)
	require.NoError(t, err)
	require.NotEmpty(t, outcome.PreferenceID)

	var pref models.Preference
	require.NoError(t, db.First(&pref, "id = ?", outcome.PreferenceID).Error)
	assert.Equal(t, contact.VCardUID, pref.EntityID, "the preference keys to the contact by VCardUID")
	assert.Equal(t, models.PreferenceCategoryFood, pref.Category)
	assert.Equal(t, "Vegetarian", pref.Value)
	assert.Equal(t, models.PreferenceSourceUser, pref.Source)
	assert.Equal(t, models.RelationshipSensitivityNormal, pref.Sensitivity)
	require.NotNil(t, pref.Confidence)
	assert.Equal(t, 1.0, *pref.Confidence)
}

func TestMigrateContactFoodPreference_IdempotentOnRerun(t *testing.T) {
	db := setupPreferenceMigrationTestDB(t)
	user := createMigrationTestUser(t, db)
	contact := createLegacyFoodContact(t, db, user.ID, "Alice", "Vegetarian")

	first, err := MigrateContactFoodPreference(db, contact, true)
	require.NoError(t, err)
	require.NotEmpty(t, first.PreferenceID)

	second, err := MigrateContactFoodPreference(db, contact, true)
	require.NoError(t, err)
	assert.NotEmpty(t, second.Skipped, "a re-run must skip, not create a duplicate")

	assert.EqualValues(t, 1, preferenceCount(t, db))
}

func TestMigrateContactFoodPreference_SkipsSoftDeletedExisting(t *testing.T) {
	db := setupPreferenceMigrationTestDB(t)
	user := createMigrationTestUser(t, db)
	contact := createLegacyFoodContact(t, db, user.ID, "Alice", "Vegetarian")

	// A preference already migrated, then soft-deleted by the user — the
	// migration must treat it as "already migrated" (Unscoped), never
	// resurrect it.
	confidence := 1.0
	existing := models.Preference{
		UserID: user.ID, EntityID: contact.VCardUID, Category: models.PreferenceCategoryFood,
		Value: "Vegetarian", Source: models.PreferenceSourceUser, Confidence: &confidence,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&existing).Error)
	require.NoError(t, db.Delete(&existing).Error)

	outcome, err := MigrateContactFoodPreference(db, contact, true)
	require.NoError(t, err)
	assert.NotEmpty(t, outcome.Skipped, "a soft-deleted migrated preference must still count as migrated")

	var count int64
	require.NoError(t, db.Model(&models.Preference{}).Unscoped().Count(&count).Error)
	assert.EqualValues(t, 1, count, "no new row may be created")
}

func TestMigrateContactFoodPreference_SkipsEmptyFood(t *testing.T) {
	db := setupPreferenceMigrationTestDB(t)
	user := createMigrationTestUser(t, db)
	contact := createLegacyFoodContact(t, db, user.ID, "No Food", "")

	outcome, err := MigrateContactFoodPreference(db, contact, true)
	require.NoError(t, err)
	assert.NotEmpty(t, outcome.Skipped)

	assert.Zero(t, preferenceCount(t, db))
}
