package controllers

import (
	"path/filepath"
	"testing"

	"mycorrhizal/database"
	"mycorrhizal/models"
	"mycorrhizal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPreferences_RealMigratedSchema is the real-DB check for T20a
// (docs/fork-plan/tickets/10-T20a-preferences.md): every other controller
// test in this package uses AutoMigrate against :memory: sqlite, which
// derives its schema from the same Go struct tags the application code uses —
// it cannot catch a GORM column-tag mismatch against the real migration SQL
// (this fork's own recurring bug class, e.g. ContactSyncLink.ETag, and the
// ticket's trap about Preference.EntityID). This test runs against a
// database.InitDB-migrated real file database and proves the ticket's three
// real-DB assertions end to end:
//
//  1. a seeded legacy FoodPreference migrates to a structured Preference row;
//  2. a hobby preference projects into RecordForContact(...).Card.PersonalInfo;
//  3. a non-normal-sensitivity preference does NOT project (§91.13 filter).
func TestPreferences_RealMigratedSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "preferences-real.db")
	db, err := database.InitDB(dbPath)
	require.NoError(t, err)

	user := models.User{Username: "realdbtester", Password: "password123!A", Email: "realdb@example.com"}
	require.NoError(t, db.Create(&user).Error)

	contact := models.Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)
	require.NotEmpty(t, contact.VCardUID)

	// Seed the legacy free-text column directly (the Go field is retired;
	// only the migration reads it now), then run the real migration function.
	require.NoError(t, db.Model(&contact).Update("food_preference", "Vegetarian").Error)
	outcome, err := services.MigrateContactFoodPreference(db, services.LegacyFoodContact{
		ID: contact.ID, UserID: user.ID, VCardUID: contact.VCardUID, FoodPreference: "Vegetarian",
	}, true)
	require.NoError(t, err)
	require.NotEmpty(t, outcome.PreferenceID)

	var migrated models.Preference
	require.NoError(t, db.First(&migrated, "id = ?", outcome.PreferenceID).Error)
	assert.Equal(t, contact.VCardUID, migrated.EntityID, "assertion 1: food preference keyed to the contact by VCardUID")
	assert.Equal(t, models.PreferenceCategoryFood, migrated.Category)
	assert.Equal(t, "Vegetarian", migrated.Value)

	// Re-run is idempotent.
	outcome2, err := services.MigrateContactFoodPreference(db, services.LegacyFoodContact{
		ID: contact.ID, UserID: user.ID, VCardUID: contact.VCardUID, FoodPreference: "Vegetarian",
	}, true)
	require.NoError(t, err)
	assert.NotEmpty(t, outcome2.Skipped, "re-running the migration must not duplicate")

	// Seed a normal-sensitivity hobby (must project) and a private-sensitivity
	// hobby with the same value (must NOT change the projection — the filter
	// is per-row in the query).
	for _, s := range []struct {
		value, sensitivity string
	}{
		{"Photography", models.RelationshipSensitivityNormal},
		{"Secret Hobby", models.RelationshipSensitivitySecret},
	} {
		require.NoError(t, db.Create(&models.Preference{
			UserID: user.ID, EntityID: contact.VCardUID, Category: models.PreferenceCategoryHobby,
			Value: s.value, Source: models.PreferenceSourceUser, Sensitivity: s.sensitivity,
		}).Error)
	}

	record := models.RecordForContact(&contact, "", db)
	require.NotNil(t, record)
	projected := record.Card.PersonalInfo
	assert.Len(t, projected, 1, "assertion 2: exactly the normal-sensitivity hobby projects")
	assert.Equal(t, "Photography", projected[0].Value, "assertion 2: the hobby preference appears in Card.PersonalInfo")
	assert.Equal(t, "hobby", projected[0].Kind)
	for _, p := range projected {
		assert.NotEqual(t, "Secret Hobby", p.Value, "BROKEN assertion 3")
	}
}
