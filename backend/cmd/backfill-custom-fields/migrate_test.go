package main

import (
	"encoding/json"
	"testing"

	"mycorrhizal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Contact{}, &models.FieldDefinition{}, &models.FieldValue{}))
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, names ...string) models.User {
	t.Helper()
	user := models.User{Username: "tester", Password: "password123!A", Email: "tester@example.com", CustomFieldNames: names}
	require.NoError(t, db.Create(&user).Error)
	return user
}

// -- pass 1: migrateUserFieldDefinitions --

func TestMigrateUserFieldDefinitions_DryRunMakesNoWrites(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)

	outcome, err := migrateUserFieldDefinitions(db, user.ID, "Pronouns", false)
	require.NoError(t, err)
	assert.Empty(t, outcome.Skipped)
	assert.Empty(t, outcome.DefinitionID)

	var count int64
	db.Model(&models.FieldDefinition{}).Count(&count)
	assert.Zero(t, count, "dry run must make zero writes")
}

func TestMigrateUserFieldDefinitions_WriteCreatesDefinition(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)

	outcome, err := migrateUserFieldDefinitions(db, user.ID, "Pronouns", true)
	require.NoError(t, err)
	require.NotEmpty(t, outcome.DefinitionID)

	var def models.FieldDefinition
	require.NoError(t, db.First(&def, "id = ?", outcome.DefinitionID).Error)
	assert.Equal(t, "Pronouns", def.Label)
	assert.Equal(t, "Pronouns", def.Key, "Key must equal the v1 name verbatim, not a derived slug")
	assert.Equal(t, models.FieldTypeString, def.Type)
	assert.Equal(t, "internal-only", def.Projection)
	assert.Equal(t, models.RelationshipSensitivityNormal, def.Sensitivity)
}

func TestMigrateUserFieldDefinitions_IdempotentOnRerun(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)

	first, err := migrateUserFieldDefinitions(db, user.ID, "Pronouns", true)
	require.NoError(t, err)
	require.NotEmpty(t, first.DefinitionID)

	second, err := migrateUserFieldDefinitions(db, user.ID, "Pronouns", true)
	require.NoError(t, err)
	assert.NotEmpty(t, second.Skipped, "a re-run must skip, not create a duplicate")

	var count int64
	db.Model(&models.FieldDefinition{}).Count(&count)
	assert.EqualValues(t, 1, count)
}

func TestMigrateUserFieldDefinitions_EmptyNameSkipped(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)

	outcome, err := migrateUserFieldDefinitions(db, user.ID, "", true)
	require.NoError(t, err)
	assert.NotEmpty(t, outcome.Skipped)
}

// -- pass 2: migrateContactFieldValue --

func TestMigrateContactFieldValue_WriteCreatesValue(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db, "Pronouns")
	_, err := migrateUserFieldDefinitions(db, user.ID, "Pronouns", true)
	require.NoError(t, err)

	contact := models.Contact{UserID: user.ID, Firstname: "Alice", CustomFields: map[string]string{"Pronouns": "she/her"}}
	require.NoError(t, db.Create(&contact).Error)

	outcome, err := migrateContactFieldValue(db, contact, "Pronouns", "she/her", true, false)
	require.NoError(t, err)
	require.NotZero(t, outcome.ValueID)

	var fv models.FieldValue
	require.NoError(t, db.First(&fv, outcome.ValueID).Error)
	var stored string
	require.NoError(t, json.Unmarshal(fv.Value, &stored))
	assert.Equal(t, "she/her", stored)
	assert.Equal(t, contact.VCardUID, fv.EntityID)
}

func TestMigrateContactFieldValue_DryRunMakesNoWrites(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db, "Pronouns")
	_, err := migrateUserFieldDefinitions(db, user.ID, "Pronouns", true)
	require.NoError(t, err)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	outcome, err := migrateContactFieldValue(db, contact, "Pronouns", "she/her", false, false)
	require.NoError(t, err)
	assert.Zero(t, outcome.ValueID)

	var count int64
	db.Model(&models.FieldValue{}).Count(&count)
	assert.Zero(t, count)
}

// Running the values pass before the definitions pass has actually written
// anything must skip cleanly, not fail the whole run -- matching WP-81's
// "individually unmigratable rows are skipped and reported, not fatal" rule.
func TestMigrateContactFieldValue_SkipsWhenNoDefinitionExists(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db, "Pronouns")
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	outcome, err := migrateContactFieldValue(db, contact, "Pronouns", "she/her", true, false)
	require.NoError(t, err)
	assert.NotEmpty(t, outcome.Skipped)
}

func TestMigrateContactFieldValue_IdempotentOnRerun(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db, "Pronouns")
	_, err := migrateUserFieldDefinitions(db, user.ID, "Pronouns", true)
	require.NoError(t, err)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	first, err := migrateContactFieldValue(db, contact, "Pronouns", "she/her", true, false)
	require.NoError(t, err)
	require.NotZero(t, first.ValueID)

	second, err := migrateContactFieldValue(db, contact, "Pronouns", "she/her", true, false)
	require.NoError(t, err)
	assert.NotEmpty(t, second.Skipped)

	var count int64
	db.Model(&models.FieldValue{}).Count(&count)
	assert.EqualValues(t, 1, count)
}

// -force re-syncs a value that changed in v1 since the first migration —
// unlike definitions, which never drift and so have no -force override.
func TestMigrateContactFieldValue_ForceUpdatesExistingValue(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db, "Pronouns")
	_, err := migrateUserFieldDefinitions(db, user.ID, "Pronouns", true)
	require.NoError(t, err)
	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	first, err := migrateContactFieldValue(db, contact, "Pronouns", "she/her", true, false)
	require.NoError(t, err)

	second, err := migrateContactFieldValue(db, contact, "Pronouns", "they/them", true, true)
	require.NoError(t, err)
	require.Equal(t, first.ValueID, second.ValueID, "force must update the existing row, not create a second one")

	var fv models.FieldValue
	require.NoError(t, db.First(&fv, second.ValueID).Error)
	var stored string
	require.NoError(t, json.Unmarshal(fv.Value, &stored))
	assert.Equal(t, "they/them", stored)
}
