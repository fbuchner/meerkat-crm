package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupHouseholdTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&User{}, &Contact{}, &Household{}, &HouseholdMember{}))
	return db
}

func TestHouseholdBeforeCreateGeneratesUUID(t *testing.T) {
	db := setupHouseholdTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	household := Household{UserID: user.ID, Name: "Smith Family", Type: HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)

	assert.NotEmpty(t, household.ID)
}

func TestHouseholdBeforeCreatePreservesExplicitID(t *testing.T) {
	db := setupHouseholdTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	household := Household{ID: "explicit-id", UserID: user.ID, Name: "Apt 4B", Type: HouseholdTypeRoommates}
	require.NoError(t, db.Create(&household).Error)

	assert.Equal(t, "explicit-id", household.ID)
}

// A contact must not be added to the same household twice — the unique
// index on (household_id, member_vcard_uid) is what the suggestion engine's
// idempotency relies on being enforceable at all.
func TestHouseholdMemberUniqueConstraintRejectsDuplicate(t *testing.T) {
	db := setupHouseholdTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	household := Household{UserID: user.ID, Name: "Smith Family", Type: HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	first := HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID, Role: HouseholdRoleHead}
	require.NoError(t, db.Create(&first).Error)

	second := HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID, Role: HouseholdRoleRoommate}
	assert.Error(t, db.Create(&second).Error, "the same contact must not be addable to the same household twice")
}

// The same contact must still be addable to two DIFFERENT households — the
// uniqueness is scoped to the (household, member) pair, not the member alone.
func TestHouseholdMemberAllowsSameContactInDifferentHouseholds(t *testing.T) {
	db := setupHouseholdTestDB(t)
	user := User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)

	householdA := Household{UserID: user.ID, Name: "Household A", Type: HouseholdTypeFamilyUnit}
	householdB := Household{UserID: user.ID, Name: "Household B", Type: HouseholdTypeRoommates}
	require.NoError(t, db.Create(&householdA).Error)
	require.NoError(t, db.Create(&householdB).Error)

	contact := Contact{UserID: user.ID, Firstname: "Alice"}
	require.NoError(t, db.Create(&contact).Error)

	require.NoError(t, db.Create(&HouseholdMember{HouseholdID: householdA.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID}).Error)
	assert.NoError(t, db.Create(&HouseholdMember{HouseholdID: householdB.ID, UserID: user.ID, MemberVCardUID: contact.VCardUID}).Error)
}
