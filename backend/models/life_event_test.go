package models

import (
	"testing"

	"mycorrhizal/contactmodel"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupLifeEventTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &LifeEvent{}))
	return db
}

func TestLifeEventBeforeCreateGeneratesUUID(t *testing.T) {
	db := setupLifeEventTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	event := LifeEvent{UserID: user.ID, EntityID: contact.VCardUID, Type: LifeEventTypeGraduated}
	require.NoError(t, db.Create(&event).Error)

	assert.NotEmpty(t, event.ID)
}

func TestLifeEventBeforeCreatePreservesExplicitID(t *testing.T) {
	db := setupLifeEventTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	event := LifeEvent{ID: "explicit-id", UserID: user.ID, EntityID: contact.VCardUID, Type: LifeEventTypeMoved}
	require.NoError(t, db.Create(&event).Error)

	assert.Equal(t, "explicit-id", event.ID)
}

// A year-only PartialDate ("known only to a year", per §91.6) and a
// multi-entry RelatedEntityIDs list must both round-trip through a real
// save/reload exactly as stored -- not just compile, per the WP-83 lesson
// that only a real AutoMigrate-backed save/reload catches column/serializer
// mismatches.
func TestLifeEventPartialDateAndRelatedEntityIDsRoundTrip(t *testing.T) {
	db := setupLifeEventTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	subject := Contact{UserID: user.ID, Firstname: "Alice"}
	spouse := Contact{UserID: user.ID, Firstname: "Bob"}
	require.NoError(t, db.Create(&subject).Error)
	require.NoError(t, db.Create(&spouse).Error)

	year := 2024
	event := LifeEvent{
		UserID:           user.ID,
		EntityID:         subject.VCardUID,
		Type:             LifeEventTypeMarried,
		Date:             &contactmodel.PartialDate{Year: &year},
		Source:           LifeEventSourceUser,
		RelatedEntityIDs: []string{spouse.VCardUID},
	}
	require.NoError(t, db.Create(&event).Error)

	var reloaded LifeEvent
	require.NoError(t, db.First(&reloaded, "id = ?", event.ID).Error)

	require.NotNil(t, reloaded.Date)
	require.NotNil(t, reloaded.Date.Year)
	assert.Equal(t, 2024, *reloaded.Date.Year)
	assert.Nil(t, reloaded.Date.Month)
	assert.Equal(t, []string{spouse.VCardUID}, reloaded.RelatedEntityIDs)
}
