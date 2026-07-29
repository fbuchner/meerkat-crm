package services

import (
	"testing"

	"mycorrhizal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupHouseholdServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Contact{}, &models.Household{}, &models.HouseholdMember{}, &models.RelationshipEdge{}))
	return db
}

func createHouseholdTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	user := models.User{Username: "tester", Password: "x", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

// createHouseholdTestContact optionally sets CRM.Kind (pass "" for an
// ordinary human contact). Setting Contact.CRM directly before Create does
// NOT survive BeforeSave — it rebuilds CRM from the flat fields via
// RecordFromContact, discarding any pre-existing nested CRM data unless
// ApplyRecordToContact set cardSetDirectly first (the same fix WP-81's
// TestRecordForContact_MergesWithPassthroughWithoutDuplication needed for
// the identical reason). Confirmed by two tests failing with every "pet"
// silently classified as an adult before this fix.
func createHouseholdTestContact(t *testing.T, db *gorm.DB, userID uint, firstname, kind string) models.Contact {
	t.Helper()
	contact := models.Contact{UserID: userID, Firstname: firstname}
	if kind != "" {
		record := models.RecordForContact(&contact, "", nil)
		record.Envelope.Kind = kind
		models.ApplyRecordToContact(&contact, record, "")
	}
	require.NoError(t, db.Create(&contact).Error)
	return contact
}

func addHouseholdMember(t *testing.T, db *gorm.DB, householdID string, userID uint, contact models.Contact, role string) {
	t.Helper()
	require.NoError(t, db.Create(&models.HouseholdMember{
		HouseholdID:    householdID,
		UserID:         userID,
		MemberVCardUID: contact.VCardUID,
		Role:           role,
	}).Error)
}

// edgeTypeCounts tallies RelationshipEdge.Type across all edges for a user,
// so tests can assert on the shape of what was generated without depending
// on row order.
func edgeTypeCounts(t *testing.T, db *gorm.DB, userID uint) map[string]int {
	t.Helper()
	var edges []models.RelationshipEdge
	require.NoError(t, db.Where("user_id = ?", userID).Find(&edges).Error)
	counts := make(map[string]int)
	for _, e := range edges {
		counts[e.Type]++
	}
	return counts
}

// TestGenerateHouseholdSuggestions_FamilyUnit is the plan's worked example:
// 2 adults + 1 child + 1 pet. Note this asserts 3 owned_by, not 2 — §91.4
// says "every human -> household pet owned_by", not "every ADULT", so the
// child also gets an owned_by suggestion to the pet. (Caught while
// implementing the engine, not just while writing this test: the plan's
// own worked example undercounted this by one.)
func TestGenerateHouseholdSuggestions_FamilyUnit(t *testing.T) {
	db := setupHouseholdServiceTestDB(t)
	user := createHouseholdTestUser(t, db)

	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)

	adult1 := createHouseholdTestContact(t, db, user.ID, "Alice", "")
	adult2 := createHouseholdTestContact(t, db, user.ID, "Bob", "")
	child := createHouseholdTestContact(t, db, user.ID, "Charlie", "")
	pet := createHouseholdTestContact(t, db, user.ID, "Fluffy", "pet")

	addHouseholdMember(t, db, household.ID, user.ID, adult1, models.HouseholdRoleHead)
	addHouseholdMember(t, db, household.ID, user.ID, adult2, models.HouseholdRoleHead)
	addHouseholdMember(t, db, household.ID, user.ID, child, models.HouseholdRoleChild)
	addHouseholdMember(t, db, household.ID, user.ID, pet, models.HouseholdRolePet)

	created, err := GenerateHouseholdSuggestions(db, household)
	require.NoError(t, err)
	assert.Len(t, created, 6, "1 spouse_of + 2 parent_of + 3 owned_by")

	for _, edge := range created {
		assert.Equal(t, models.RelationshipStatusSuggested, edge.Status)
		assert.Equal(t, models.RelationshipSourceHouseholdInferred, edge.Source)
		assert.Equal(t, 0.8, edge.Confidence)
		assert.Equal(t, models.RelationshipSensitivityNormal, edge.Sensitivity)
	}

	counts := edgeTypeCounts(t, db, user.ID)
	assert.Equal(t, 1, counts["spouse_of"])
	assert.Equal(t, 2, counts["parent_of"])
	assert.Equal(t, 3, counts["owned_by"])
	assert.Len(t, counts, 3, "no other edge types should appear")

	// The pet must be the SOURCE of each owned_by edge ("pet owned_by
	// human"), not the target.
	var petEdges []models.RelationshipEdge
	require.NoError(t, db.Where("type = ?", "owned_by").Find(&petEdges).Error)
	for _, e := range petEdges {
		assert.Equal(t, pet.VCardUID, e.SourceID)
	}
}

// §91.4: roommates households suggest ONLY roommate_of, regardless of role
// or kind — explicitly never parent/owner/spouse. A pet in a roommates
// household still just gets roommate_of, not owned_by.
func TestGenerateHouseholdSuggestions_Roommates(t *testing.T) {
	db := setupHouseholdServiceTestDB(t)
	user := createHouseholdTestUser(t, db)

	household := models.Household{UserID: user.ID, Name: "Apt 4B", Type: models.HouseholdTypeRoommates}
	require.NoError(t, db.Create(&household).Error)

	a := createHouseholdTestContact(t, db, user.ID, "Dana", "")
	b := createHouseholdTestContact(t, db, user.ID, "Erin", "")
	petC := createHouseholdTestContact(t, db, user.ID, "Whiskers", "pet")

	addHouseholdMember(t, db, household.ID, user.ID, a, models.HouseholdRoleRoommate)
	addHouseholdMember(t, db, household.ID, user.ID, b, models.HouseholdRoleRoommate)
	addHouseholdMember(t, db, household.ID, user.ID, petC, models.HouseholdRolePet)

	created, err := GenerateHouseholdSuggestions(db, household)
	require.NoError(t, err)
	assert.Len(t, created, 3, "every pair, including the pet, gets exactly roommate_of")

	counts := edgeTypeCounts(t, db, user.ID)
	assert.Equal(t, 3, counts["roommate_of"])
	assert.Len(t, counts, 1, "no owned_by/spouse_of/parent_of must appear for a roommates household")

	for _, edge := range created {
		assert.Equal(t, 0.4, edge.Confidence)
	}
}

func TestGenerateHouseholdSuggestions_OtherProducesNothing(t *testing.T) {
	db := setupHouseholdServiceTestDB(t)
	user := createHouseholdTestUser(t, db)

	household := models.Household{UserID: user.ID, Name: "Mixed group", Type: models.HouseholdTypeOther}
	require.NoError(t, db.Create(&household).Error)

	a := createHouseholdTestContact(t, db, user.ID, "Foo", "")
	b := createHouseholdTestContact(t, db, user.ID, "Bar", "")
	addHouseholdMember(t, db, household.ID, user.ID, a, "")
	addHouseholdMember(t, db, household.ID, user.ID, b, "")

	created, err := GenerateHouseholdSuggestions(db, household)
	require.NoError(t, err)
	assert.Empty(t, created)

	var count int64
	db.Model(&models.RelationshipEdge{}).Count(&count)
	assert.Zero(t, count)
}

// Running the engine twice on an unchanged household must not double the
// suggestions — this is what makes it safe to call after every membership
// change without the caller tracking deltas.
func TestGenerateHouseholdSuggestions_IdempotentRerun(t *testing.T) {
	db := setupHouseholdServiceTestDB(t)
	user := createHouseholdTestUser(t, db)

	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)

	a := createHouseholdTestContact(t, db, user.ID, "Alice", "")
	b := createHouseholdTestContact(t, db, user.ID, "Bob", "")
	addHouseholdMember(t, db, household.ID, user.ID, a, models.HouseholdRoleHead)
	addHouseholdMember(t, db, household.ID, user.ID, b, models.HouseholdRoleHead)

	first, err := GenerateHouseholdSuggestions(db, household)
	require.NoError(t, err)
	assert.Len(t, first, 1)

	second, err := GenerateHouseholdSuggestions(db, household)
	require.NoError(t, err)
	assert.Empty(t, second, "the second run must create nothing new")

	var count int64
	db.Model(&models.RelationshipEdge{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

// A relationship already recorded — at either status, in either direction —
// must not be re-suggested. Covers the case that matters most: a pair the
// user already explicitly confirmed should never get a redundant
// "suggested" duplicate sitting alongside it.
func TestGenerateHouseholdSuggestions_DoesNotDuplicateExistingConfirmedEdge(t *testing.T) {
	db := setupHouseholdServiceTestDB(t)
	user := createHouseholdTestUser(t, db)

	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)

	adult := createHouseholdTestContact(t, db, user.ID, "Alice", "")
	child := createHouseholdTestContact(t, db, user.ID, "Charlie", "")
	addHouseholdMember(t, db, household.ID, user.ID, adult, models.HouseholdRoleHead)
	addHouseholdMember(t, db, household.ID, user.ID, child, models.HouseholdRoleChild)

	// Pre-existing CONFIRMED edge, stored in the INVERSE direction/token —
	// exactly what a user-confirmed edge might look like from the child's
	// side (child_of, not parent_of).
	existing := models.RelationshipEdge{
		UserID:      user.ID,
		SourceID:    child.VCardUID,
		TargetID:    adult.VCardUID,
		Type:        "child_of",
		Source:      models.RelationshipSourceUserConfirmed,
		Confidence:  1.0,
		Status:      models.RelationshipStatusConfirmed,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&existing).Error)

	created, err := GenerateHouseholdSuggestions(db, household)
	require.NoError(t, err)
	assert.Empty(t, created, "an existing child_of edge must be recognized as covering the parent_of suggestion")

	var count int64
	db.Model(&models.RelationshipEdge{}).Count(&count)
	assert.Equal(t, int64(1), count, "only the original confirmed edge should exist")
}

// Pet classification must come from Contact.CRM.Kind, not Role — a member
// whose Role happens to be unset (or anything other than "pet") but whose
// Contact.CRM.Kind is "pet" must still be classified as a pet.
func TestGenerateHouseholdSuggestions_ClassifiesByKindNotRole(t *testing.T) {
	db := setupHouseholdServiceTestDB(t)
	user := createHouseholdTestUser(t, db)

	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)

	adult := createHouseholdTestContact(t, db, user.ID, "Alice", "")
	pet := createHouseholdTestContact(t, db, user.ID, "Fluffy", "animal")

	addHouseholdMember(t, db, household.ID, user.ID, adult, models.HouseholdRoleHead)
	// Role deliberately left empty, not "pet" — Kind alone must drive it.
	addHouseholdMember(t, db, household.ID, user.ID, pet, "")

	created, err := GenerateHouseholdSuggestions(db, household)
	require.NoError(t, err)
	require.Len(t, created, 1)
	assert.Equal(t, "owned_by", created[0].Type)
	assert.Equal(t, pet.VCardUID, created[0].SourceID)
	assert.Equal(t, adult.VCardUID, created[0].TargetID)
}

// A household with fewer than 2 members has no pairs to suggest anything
// for, and must not error.
func TestGenerateHouseholdSuggestions_SingleMemberProducesNothing(t *testing.T) {
	db := setupHouseholdServiceTestDB(t)
	user := createHouseholdTestUser(t, db)

	household := models.Household{UserID: user.ID, Name: "Solo", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)

	a := createHouseholdTestContact(t, db, user.ID, "Alice", "")
	addHouseholdMember(t, db, household.ID, user.ID, a, models.HouseholdRoleHead)

	created, err := GenerateHouseholdSuggestions(db, household)
	require.NoError(t, err)
	assert.Empty(t, created)
}
