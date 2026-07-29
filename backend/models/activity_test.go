package models

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupActivityTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &Activity{}))
	return db
}

func TestActivityBeforeCreateGeneratesUUID(t *testing.T) {
	db := setupActivityTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	activity := Activity{UserID: user.ID, Title: "Coffee", Date: time.Now(), Type: InteractionTypeMeal}
	require.NoError(t, db.Create(&activity).Error)

	assert.NotEmpty(t, activity.UUID)
}

func TestActivityBeforeCreatePreservesExplicitUUID(t *testing.T) {
	db := setupActivityTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	activity := Activity{UserID: user.ID, Title: "Coffee", Date: time.Now(), UUID: "explicit-uuid"}
	require.NoError(t, db.Create(&activity).Error)

	assert.Equal(t, "explicit-uuid", activity.UUID)
}

func TestActivityQualifying(t *testing.T) {
	visit := Activity{Type: InteractionTypeVisit}
	assert.True(t, visit.Qualifying(), "a visit is a real qualifying interaction")

	photo := Activity{Type: InteractionTypePhoto}
	assert.False(t, photo.Qualifying(), "a passive/social-media-like photo share does not qualify")

	unset := Activity{}
	assert.True(t, unset.Qualifying(), "an unrecognized/unset type defaults to qualifying")
}
