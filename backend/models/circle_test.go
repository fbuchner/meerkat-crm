package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCircleTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &Circle{}, &CircleMember{}))
	return db
}

func TestCircleBeforeCreateGeneratesUUID(t *testing.T) {
	db := setupCircleTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	circle := Circle{UserID: user.ID, Name: "Cam Girl friends"}
	require.NoError(t, db.Create(&circle).Error)

	assert.NotEmpty(t, circle.ID)
}

func TestCircleBeforeCreatePreservesExplicitID(t *testing.T) {
	db := setupCircleTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	circle := Circle{ID: "explicit-id", UserID: user.ID, Name: "College friends"}
	require.NoError(t, db.Create(&circle).Error)

	assert.Equal(t, "explicit-id", circle.ID)
}

// A contact must not be added to the same circle twice — the unique index on
// (circle_id, member_vcard_uid) is the real DB-level constraint the model
// depends on, not just an application-level check.
func TestCircleMemberUniqueConstraintRejectsDuplicate(t *testing.T) {
	db := setupCircleTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	circle := Circle{UserID: user.ID, Name: "Studio Porn friends"}
	require.NoError(t, db.Create(&circle).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	first := CircleMember{CircleID: circle.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID}
	require.NoError(t, db.Create(&first).Error)

	second := CircleMember{CircleID: circle.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID}
	assert.Error(t, db.Create(&second).Error, "the same contact must not be addable to the same circle twice")
}

// The same contact must still be addable to two DIFFERENT circles — the
// uniqueness is scoped to the (circle, member) pair, not the member alone.
func TestCircleMemberAllowsSameContactInDifferentCircles(t *testing.T) {
	db := setupCircleTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	circleA := Circle{UserID: user.ID, Name: "Circle A"}
	circleB := Circle{UserID: user.ID, Name: "Circle B"}
	require.NoError(t, db.Create(&circleA).Error)
	require.NoError(t, db.Create(&circleB).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	require.NoError(t, db.Create(&CircleMember{CircleID: circleA.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID}).Error)
	assert.NoError(t, db.Create(&CircleMember{CircleID: circleB.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID}).Error)
}
