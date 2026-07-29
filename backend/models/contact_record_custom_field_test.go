package models

import (
	"encoding/json"
	"testing"

	"mycorrhizal/contactmodel"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCustomFieldProjectionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &FieldDefinition{}, &FieldValue{}, &RelationshipEdge{}, &Tag{}, &ContactTag{}))
	return db
}

func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// WP-84b's FieldValue -> Passthrough.VCard projection (§94.5): a
// "vcard:X-<NAME>"-projected, normal-sensitivity field appears as a JCardProp
// in RecordForContact's output.
func TestRecordForContact_ProjectsVCardCustomField(t *testing.T) {
	db := setupCustomFieldProjectionTestDB(t)
	user := User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	def := FieldDefinition{
		UserID: user.ID, Label: "Pronouns", Key: "pronouns",
		Target: FieldDefinitionTargetContact, Type: FieldTypeEnum, Projection: "vcard:X-PRONOUNS",
		Sensitivity: RelationshipSensitivityNormal,
		Constraints: FieldConstraints{Values: []string{"she/her", "he/him", "they/them"}, Multi: true},
	}
	require.NoError(t, db.Create(&def).Error)

	value := FieldValue{
		FieldDefinitionID: def.ID, UserID: user.ID, EntityID: contact.VCardUID,
		Value: mustMarshal(t, []string{"she/her"}),
	}
	require.NoError(t, db.Create(&value).Error)

	record := RecordForContact(&contact, "", db)
	require.Len(t, record.Passthrough.VCard, 1)
	assert.Equal(t, "X-PRONOUNS", record.Passthrough.VCard[0].Name)
	assert.JSONEq(t, `["she/her"]`, string(record.Passthrough.VCard[0].Value))
}

// An internal-only definition must never appear in the projection, even
// though it has a value.
func TestRecordForContact_DoesNotProjectInternalOnlyCustomField(t *testing.T) {
	db := setupCustomFieldProjectionTestDB(t)
	user := User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	def := FieldDefinition{
		UserID: user.ID, Label: "Internal Note", Key: "internal_note",
		Target: FieldDefinitionTargetContact, Type: FieldTypeString, Projection: "internal-only",
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&def).Error)
	require.NoError(t, db.Create(&FieldValue{
		FieldDefinitionID: def.ID, UserID: user.ID, EntityID: contact.VCardUID,
		Value: mustMarshal(t, "some internal detail"),
	}).Error)

	record := RecordForContact(&contact, "", db)
	assert.Empty(t, record.Passthrough.VCard)
}

// A sensitive field's value must not project even though it has a vcard:
// mapping — §91.13's default-exclude-from-export rule, same discipline
// projectRelationshipEdges/projectTags already enforce.
func TestRecordForContact_DoesNotProjectSensitiveCustomField(t *testing.T) {
	db := setupCustomFieldProjectionTestDB(t)
	user := User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	def := FieldDefinition{
		UserID: user.ID, Label: "HIV Status", Key: "hiv_status",
		Target: FieldDefinitionTargetContact, Type: FieldTypeString, Projection: "vcard:X-HIV-STATUS",
		Sensitivity: RelationshipSensitivitySecret,
	}
	require.NoError(t, db.Create(&def).Error)
	require.NoError(t, db.Create(&FieldValue{
		FieldDefinitionID: def.ID, UserID: user.ID, EntityID: contact.VCardUID,
		Value: mustMarshal(t, "some sensitive value"),
	}).Error)

	record := RecordForContact(&contact, "", db)
	assert.Empty(t, record.Passthrough.VCard)
}

// A custom field's projected name must not clobber an already-imported
// passthrough entry of the same name.
func TestRecordForContact_ProjectsCustomFieldDedupesAgainstExistingPassthrough(t *testing.T) {
	db := setupCustomFieldProjectionTestDB(t)
	user := User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	record := RecordForContact(&contact, "", nil)
	record.Passthrough.VCard = []contactmodel.JCardProp{
		{Name: "X-PRONOUNS", Type: "text", Value: mustMarshal(t, "imported-value")},
	}
	ApplyRecordToContact(&contact, record, "")
	require.NoError(t, db.Create(&contact).Error)

	def := FieldDefinition{
		UserID: user.ID, Label: "Pronouns", Key: "pronouns",
		Target: FieldDefinitionTargetContact, Type: FieldTypeString, Projection: "vcard:X-PRONOUNS",
		Sensitivity: RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&def).Error)
	require.NoError(t, db.Create(&FieldValue{
		FieldDefinitionID: def.ID, UserID: user.ID, EntityID: contact.VCardUID,
		Value: mustMarshal(t, "she/her"),
	}).Error)

	reloaded := RecordForContact(&contact, "", db)
	require.Len(t, reloaded.Passthrough.VCard, 1, "the existing passthrough entry must win, not be duplicated")
	assert.Equal(t, "X-PRONOUNS", reloaded.Passthrough.VCard[0].Name)
	assert.JSONEq(t, `"imported-value"`, string(reloaded.Passthrough.VCard[0].Value))
}

func TestRecordForContact_NilDBSkipsCustomFieldProjection(t *testing.T) {
	contact := Contact{Firstname: "Alice", VCardUID: "some-uid"}
	record := RecordForContact(&contact, "", nil)
	assert.Empty(t, record.Passthrough.VCard)
}
