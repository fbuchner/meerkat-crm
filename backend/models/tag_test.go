package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTagTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &Tag{}, &ContactTag{}))
	return db
}

func TestTagBeforeCreateGeneratesUUID(t *testing.T) {
	db := setupTagTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	tag := Tag{UserID: user.ID, Name: "SWer"}
	require.NoError(t, db.Create(&tag).Error)

	assert.NotEmpty(t, tag.ID)
}

func TestTagBeforeCreatePreservesExplicitID(t *testing.T) {
	db := setupTagTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	tag := Tag{ID: "explicit-id", UserID: user.ID, Name: "poly"}
	require.NoError(t, db.Create(&tag).Error)

	assert.Equal(t, "explicit-id", tag.ID)
}

// A contact must not be tagged with the same tag twice — the unique index on
// (tag_id, contact_vcard_uid) is the real DB-level constraint.
func TestContactTagUniqueConstraintRejectsDuplicate(t *testing.T) {
	db := setupTagTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	tag := Tag{UserID: user.ID, Name: "vegan"}
	require.NoError(t, db.Create(&tag).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	first := ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID}
	require.NoError(t, db.Create(&first).Error)

	second := ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID}
	assert.Error(t, db.Create(&second).Error, "the same contact must not be taggable with the same tag twice")
}

// The same contact must still be taggable with two DIFFERENT tags.
func TestContactTagAllowsSameContactWithDifferentTags(t *testing.T) {
	db := setupTagTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	tagA := Tag{UserID: user.ID, Name: "poly"}
	tagB := Tag{UserID: user.ID, Name: "vegan"}
	require.NoError(t, db.Create(&tagA).Error)
	require.NoError(t, db.Create(&tagB).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	require.NoError(t, db.Create(&ContactTag{TagID: tagA.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID}).Error)
	assert.NoError(t, db.Create(&ContactTag{TagID: tagB.ID, UserID: user.ID, ContactVCardUID: contact.VCardUID}).Error)
}
