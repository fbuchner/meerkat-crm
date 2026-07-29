package models

import (
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupFieldDefinitionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &FieldDefinition{}, &FieldValue{}))
	return db
}

func TestFieldDefinitionBeforeCreateGeneratesUUID(t *testing.T) {
	db := setupFieldDefinitionTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	def := FieldDefinition{
		UserID: user.ID, Label: "Gender Identity", Key: "gender_identity",
		Target: FieldDefinitionTargetContact, Type: FieldTypeString, Projection: "internal-only",
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&def).Error)

	assert.NotEmpty(t, def.ID)
}

func TestFieldDefinitionBeforeCreatePreservesExplicitID(t *testing.T) {
	db := setupFieldDefinitionTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	def := FieldDefinition{
		ID: "explicit-id", UserID: user.ID, Label: "Pronouns", Key: "pronouns",
		Target: FieldDefinitionTargetContact, Type: FieldTypeEnum, Projection: "vcard:X-PRONOUNS",
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&def).Error)

	assert.Equal(t, "explicit-id", def.ID)
}

// Two definitions with the same key for the same user would be ambiguous to
// reference (the key is the stable API field name) -- the unique index on
// (user_id, key) is the real DB-level constraint this depends on.
func TestFieldDefinitionUniqueConstraintRejectsDuplicateKey(t *testing.T) {
	db := setupFieldDefinitionTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	first := FieldDefinition{
		UserID: user.ID, Label: "Pronouns", Key: "pronouns",
		Target: FieldDefinitionTargetContact, Type: FieldTypeString, Projection: "internal-only",
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&first).Error)

	second := FieldDefinition{
		UserID: user.ID, Label: "Pronouns (again)", Key: "pronouns",
		Target: FieldDefinitionTargetContact, Type: FieldTypeString, Projection: "internal-only",
		Sensitivity: RelationshipSensitivityNormal,
	}
	assert.Error(t, db.Create(&second).Error, "the same user must not define the same key twice")
}

// The same key must still be usable by two DIFFERENT users -- uniqueness is
// scoped to (user, key), not the key alone.
func TestFieldDefinitionAllowsSameKeyForDifferentUsers(t *testing.T) {
	db := setupFieldDefinitionTestDB(t)
	userA := User{Username: "a", Password: "x", Email: "a@example.com"}
	userB := User{Username: "b", Password: "x", Email: "b@example.com"}
	require.NoError(t, db.Create(&userA).Error)
	require.NoError(t, db.Create(&userB).Error)

	mk := func(userID uint) FieldDefinition {
		return FieldDefinition{
			UserID: userID, Label: "Pronouns", Key: "pronouns",
			Target: FieldDefinitionTargetContact, Type: FieldTypeString, Projection: "internal-only",
			Sensitivity: RelationshipSensitivityNormal,
		}
	}
	defA := mk(userA.ID)
	defB := mk(userB.ID)
	require.NoError(t, db.Create(&defA).Error)
	assert.NoError(t, db.Create(&defB).Error)
}

// FieldConstraints must round-trip through a real save/reload exactly as
// stored -- not just compile, per the WP-83 lesson that only a real
// AutoMigrate-backed save/reload catches column/serializer mismatches.
func TestFieldDefinitionConstraintsRoundTrip(t *testing.T) {
	db := setupFieldDefinitionTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	min := 0.0
	max := 10.0
	def := FieldDefinition{
		UserID: user.ID, Label: "Rating", Key: "rating",
		Target: FieldDefinitionTargetContact, Type: FieldTypeNumber, Projection: "internal-only",
		Sensitivity: RelationshipSensitivityNormal,
		Constraints: FieldConstraints{Min: &min, Max: &max},
	}
	require.NoError(t, db.Create(&def).Error)

	var reloaded FieldDefinition
	require.NoError(t, db.First(&reloaded, "id = ?", def.ID).Error)

	require.NotNil(t, reloaded.Constraints.Min)
	require.NotNil(t, reloaded.Constraints.Max)
	assert.Equal(t, 0.0, *reloaded.Constraints.Min)
	assert.Equal(t, 10.0, *reloaded.Constraints.Max)
}

// A contact must not have two values for the same field definition -- the
// unique index on (field_definition_id, entity_id) is the real DB-level
// constraint.
func TestFieldValueUniqueConstraintRejectsDuplicate(t *testing.T) {
	db := setupFieldDefinitionTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	def := FieldDefinition{
		UserID: user.ID, Label: "Pronouns", Key: "pronouns",
		Target: FieldDefinitionTargetContact, Type: FieldTypeString, Projection: "internal-only",
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&def).Error)
	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	raw, err := json.Marshal("she/her")
	require.NoError(t, err)

	first := FieldValue{FieldDefinitionID: def.ID, UserID: user.ID, EntityID: contact.VCardUID, Value: raw}
	require.NoError(t, db.Create(&first).Error)

	second := FieldValue{FieldDefinitionID: def.ID, UserID: user.ID, EntityID: contact.VCardUID, Value: raw}
	assert.Error(t, db.Create(&second).Error, "a contact must not have two values for the same field definition")
}
