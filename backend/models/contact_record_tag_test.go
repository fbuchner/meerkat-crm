package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTagProjectionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &Tag{}, &ContactTag{}, &RelationshipEdge{}))
	return db
}

// WP-84's Tag -> Card.Keywords projection (§91.5): a contact tagged with two
// Tags gets both tag names merged into RecordForContact's Card.Keywords,
// alongside any pre-existing passthrough keyword, with no duplication.
func TestRecordForContact_ProjectsTagsOntoKeywords(t *testing.T) {
	db := setupTagProjectionTestDB(t)
	user := User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	poly := Tag{UserID: user.ID, Name: "poly"}
	swer := Tag{UserID: user.ID, Name: "SWer"}
	require.NoError(t, db.Create(&poly).Error)
	require.NoError(t, db.Create(&swer).Error)

	require.NoError(t, db.Create(&ContactTag{TagID: poly.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID}).Error)
	require.NoError(t, db.Create(&ContactTag{TagID: swer.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID}).Error)

	record := RecordForContact(&contact, "", db)
	assert.ElementsMatch(t, []string{"poly", "SWer"}, record.Card.Keywords)
}

// A tag name that coincides with an existing passthrough keyword must not be
// duplicated in the merged result.
//
// Setting Contact.Card directly before Save does NOT survive BeforeSave — it
// rebuilds Card from the flat fields via RecordFromContact, discarding any
// pre-existing nested Card data unless ApplyRecordToContact set
// cardSetDirectly first (the same WP-81/WP-83 pitfall documented in
// services/household_service_test.go's createHouseholdTestContact).
func TestRecordForContact_ProjectsTagsDedupesAgainstExistingKeywords(t *testing.T) {
	db := setupTagProjectionTestDB(t)
	user := User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	record := RecordForContact(&contact, "", nil)
	record.Card.Keywords = []string{"vegan"}
	ApplyRecordToContact(&contact, record, "")
	require.NoError(t, db.Create(&contact).Error)

	tag := Tag{UserID: user.ID, Name: "vegan"}
	require.NoError(t, db.Create(&tag).Error)
	require.NoError(t, db.Create(&ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID}).Error)

	reloaded := RecordForContact(&contact, "", db)
	assert.Equal(t, []string{"vegan"}, reloaded.Card.Keywords)
}

func TestRecordForContact_NilDBSkipsTagProjection(t *testing.T) {
	contact := Contact{Firstname: "Alice", VCardUID: "some-uid"}
	record := RecordForContact(&contact, "", nil)
	assert.Empty(t, record.Card.Keywords)
}
