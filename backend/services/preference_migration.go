package services

import (
	"errors"
	"fmt"

	"mycorrhizal/models"

	"gorm.io/gorm"
)

// LegacyFoodContact is the one-shot read shape for the retiring
// contacts.food_preference column — see cmd/backfill-preferences' doc
// comment: the Go field is removed from models.Contact by T20a, so the
// backfill reads the raw column through this struct instead.
//
// VCardUID carries an explicit gorm:"column:vcard_uid" tag for the same
// reason models.Contact.VCardUID does: GORM's default namer derives
// v_card_uid from the field name, which would silently fail to match the
// real vcard_uid column during the backfill's raw Scan (CLAUDE.md trap 1).
type LegacyFoodContact struct {
	ID             uint
	UserID         uint
	VCardUID       string `gorm:"column:vcard_uid"`
	FoodPreference string `gorm:"column:food_preference"`
}

// PreferenceMigrationOutcome summarizes what MigrateContactFoodPreference did
// (or, in dry run, would do) for one contact's FoodPreference.
type PreferenceMigrationOutcome struct {
	// Skipped is non-empty when this food preference was not (and, in dry
	// run, would not be) migrated; its value is the human-readable reason.
	Skipped string
	// PreferenceID is populated only when write=true and the preference was
	// actually persisted (empty in dry run).
	PreferenceID string
}

// MigrateContactFoodPreference migrates one contact's FoodPreference into a
// structured food-category Preference (models/preference.go), per
// docs/fork-plan/91-envelope-data-model.md §91.9's "the free-text
// Contact.FoodPreference migrates into a food preference":
//
//	category: food, key: "", value: <the food string>, source: user,
//	confidence: 1.0, sensitivity: normal
//
// Lives in services (not the cmd package) so both the
// cmd/backfill-preferences command and the real-DB migration test exercise
// the exact same function. write=false performs every check but makes zero
// writes. Idempotency is checked against (entity_id, category, value) with
// Unscoped() — an already-migrated row OR a soft-deleted one counts as
// "already migrated", so a re-run never duplicates and a deleted preference
// is never silently resurrected by the backfill.
func MigrateContactFoodPreference(db *gorm.DB, contact LegacyFoodContact, write bool) (PreferenceMigrationOutcome, error) {
	food := contact.FoodPreference
	if food == "" {
		return PreferenceMigrationOutcome{Skipped: "empty food preference"}, nil
	}
	if contact.VCardUID == "" {
		return PreferenceMigrationOutcome{Skipped: "contact has no vcard_uid"}, nil
	}

	var existing models.Preference
	err := db.Unscoped().
		Where("entity_id = ? AND category = ? AND value = ?", contact.VCardUID, models.PreferenceCategoryFood, food).
		First(&existing).Error
	switch {
	case err == nil:
		return PreferenceMigrationOutcome{Skipped: fmt.Sprintf("already migrated to preference %s", existing.ID)}, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		// Not yet migrated -- the common case.
	default:
		return PreferenceMigrationOutcome{}, fmt.Errorf("checking idempotency for entity=%s: %w", contact.VCardUID, err)
	}

	if !write {
		return PreferenceMigrationOutcome{}, nil
	}

	confidence := 1.0
	pref := models.Preference{
		UserID:      contact.UserID,
		EntityID:    contact.VCardUID,
		Category:    models.PreferenceCategoryFood,
		Value:       food,
		Source:      models.PreferenceSourceUser,
		Confidence:  &confidence,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	if err := db.Create(&pref).Error; err != nil {
		return PreferenceMigrationOutcome{}, fmt.Errorf("creating food preference for entity=%s: %w", contact.VCardUID, err)
	}

	return PreferenceMigrationOutcome{PreferenceID: pref.ID}, nil
}
