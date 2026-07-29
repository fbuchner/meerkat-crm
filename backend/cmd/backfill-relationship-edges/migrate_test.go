package main

import (
	"testing"

	"mycorrhizal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Contact{}, &models.Relationship{}, &models.RelationshipEdge{}))
	return db
}

func createTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	user := models.User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func createTestContact(t *testing.T, db *gorm.DB, userID uint, firstname string) models.Contact {
	t.Helper()
	contact := models.Contact{UserID: userID, Firstname: firstname}
	require.NoError(t, db.Create(&contact).Error)
	return contact
}

// The core happy path: a relationship linked to a real, existing contact.
// No thin Contact should be created; the direction and matched type must be
// exactly as worked out in the design (source=related party, target=owner).
func TestMigrateOneRelationship_LinkedContact(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")
	alice := createTestContact(t, db, user.ID, "Alice")

	legacy := models.Relationship{
		UserID:           user.ID,
		Name:             "Alice",
		Type:             "Mother",
		ContactID:        bob.ID,
		RelatedContactID: &alice.ID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	outcome, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	assert.Empty(t, outcome.Skipped)
	assert.False(t, outcome.CreatedThinContact)
	assert.Equal(t, alice.VCardUID, outcome.SourceID)
	assert.Equal(t, bob.VCardUID, outcome.TargetID)
	assert.Equal(t, "parent_of", outcome.MatchedType)
	assert.False(t, outcome.UsedFallback)
	require.NotEmpty(t, outcome.EdgeID)

	var edge models.RelationshipEdge
	require.NoError(t, db.First(&edge, "id = ?", outcome.EdgeID).Error)
	assert.Equal(t, alice.VCardUID, edge.SourceID)
	assert.Equal(t, bob.VCardUID, edge.TargetID)
	assert.Equal(t, "parent_of", edge.Type)
	assert.True(t, edge.Directional) // parent_of is asymmetric
	assert.Equal(t, models.RelationshipSourceImported, edge.Source)
	assert.Equal(t, 1.0, edge.Confidence)
	assert.Equal(t, models.RelationshipStatusConfirmed, edge.Status)
	assert.Equal(t, models.RelationshipSensitivityNormal, edge.Sensitivity)
	require.NotNil(t, edge.LegacyRelationshipID)
	assert.Equal(t, legacy.ID, *edge.LegacyRelationshipID)
	assert.Empty(t, edge.Metadata, "a matched type must not carry a legacy_type fallback note")

	// The pre-existing Alice contact must not have been touched.
	var reloadedAlice models.Contact
	require.NoError(t, db.First(&reloadedAlice, alice.ID).Error)
	assert.Equal(t, alice.VCardUID, reloadedAlice.VCardUID)
}

// D3's explicit migration consequence: a name-only relationship gets a new
// thin Contact, with Gender/Birthday transferred since RelationshipEdge has
// no field for either.
func TestMigrateOneRelationship_NameOnlyPromotesThinContact(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")

	legacy := models.Relationship{
		UserID:    user.ID,
		Name:      "Grandma Sue",
		Type:      "Grandmother",
		Gender:    "female",
		Birthday:  "1945-03-12",
		ContactID: bob.ID,
		// RelatedContactID left nil: the name-only case.
	}
	require.NoError(t, db.Create(&legacy).Error)

	var contactCountBefore int64
	db.Model(&models.Contact{}).Count(&contactCountBefore)

	outcome, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	assert.Empty(t, outcome.Skipped)
	assert.True(t, outcome.CreatedThinContact)
	assert.Equal(t, "Grandma Sue", outcome.ThinContactName)
	require.NotEmpty(t, outcome.SourceID)

	var contactCountAfter int64
	db.Model(&models.Contact{}).Count(&contactCountAfter)
	assert.Equal(t, contactCountBefore+1, contactCountAfter, "exactly one new thin contact must be created")

	var thin models.Contact
	require.NoError(t, db.Where("vcard_uid = ?", outcome.SourceID).First(&thin).Error)
	assert.Equal(t, "Grandma Sue", thin.Firstname, "the whole legacy Name goes into Firstname verbatim, not split")
	assert.Equal(t, "female", thin.Gender)
	assert.Equal(t, "1945-03-12", thin.Birthday)
	assert.False(t, thin.Archived, "a promoted thin contact must be an ordinary, visible contact")

	var edge models.RelationshipEdge
	require.NoError(t, db.First(&edge, "id = ?", outcome.EdgeID).Error)
	assert.Equal(t, thin.VCardUID, edge.SourceID)
	assert.Equal(t, bob.VCardUID, edge.TargetID)
}

// "Grandmother" matches nothing in the registry directly, but "grandmother"
// isn't a synonym either — falls to related_to. This also exercises the
// Metadata/Confidence side of the fallback path the case above didn't.
func TestMigrateOneRelationship_UnmatchedTypeFallsBackToRelatedTo(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")
	alice := createTestContact(t, db, user.ID, "Alice")

	legacy := models.Relationship{
		UserID:           user.ID,
		Name:             "Alice",
		Type:             "Family", // one of the two real unmapped values found in this repo's own test fixtures
		ContactID:        bob.ID,
		RelatedContactID: &alice.ID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	outcome, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	assert.Equal(t, "related_to", outcome.MatchedType)
	assert.True(t, outcome.UsedFallback)

	var edge models.RelationshipEdge
	require.NoError(t, db.First(&edge, "id = ?", outcome.EdgeID).Error)
	assert.Equal(t, "related_to", edge.Type)
	assert.Equal(t, 0.5, edge.Confidence, "fallback type-categorization is uncertain, but the relationship's existence is not")
	assert.Equal(t, models.RelationshipStatusConfirmed, edge.Status, "the user did assert this relationship; only its category is uncertain")
	require.NotNil(t, edge.Metadata)
	assert.Equal(t, "Family", edge.Metadata["legacy_type"], "the original free text must survive for later manual reclassification")
	assert.False(t, edge.Directional, "related_to is symmetric")
}

func TestMigrateOneRelationship_SkipsSelfReference(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")

	legacy := models.Relationship{
		UserID:           user.ID,
		Name:             "Bob",
		Type:             "Self",
		ContactID:        bob.ID,
		RelatedContactID: &bob.ID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	outcome, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	assert.Contains(t, outcome.Skipped, "self-referencing")

	var count int64
	db.Model(&models.RelationshipEdge{}).Count(&count)
	assert.Zero(t, count)
}

func TestMigrateOneRelationship_SkipsEmptyName(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")

	legacy := models.Relationship{
		UserID:    user.ID,
		Name:      "   ", // whitespace-only
		Type:      "Friend",
		ContactID: bob.ID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	outcome, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	assert.Contains(t, outcome.Skipped, "empty name")

	var count int64
	db.Model(&models.Contact{}).Count(&count)
	assert.Equal(t, int64(1), count, "only Bob — no thin contact created for an empty name")
}

// A known, pre-existing gap: deleting a contact only cleans up Relationship
// rows where it's the owner, not rows where it's the target — so a dangling
// related_contact_id is real data this migration must tolerate, not a
// hypothetical.
func TestMigrateOneRelationship_SkipsDanglingRelatedContact(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")

	danglingID := bob.ID + 999
	legacy := models.Relationship{
		UserID:           user.ID,
		Name:             "Ghost",
		Type:             "Friend",
		ContactID:        bob.ID,
		RelatedContactID: &danglingID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	outcome, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	assert.Contains(t, outcome.Skipped, "dangling pointer")
}

// Running the migration twice must not create a second edge for the same
// legacy row — this is what makes fail-fast-without-a-transaction safe.
func TestMigrateOneRelationship_IdempotentAcrossTwoRuns(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")
	alice := createTestContact(t, db, user.ID, "Alice")

	legacy := models.Relationship{
		UserID:           user.ID,
		Name:             "Alice",
		Type:             "Sibling",
		ContactID:        bob.ID,
		RelatedContactID: &alice.ID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	first, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	require.NotEmpty(t, first.EdgeID)

	second, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	assert.Contains(t, second.Skipped, "already migrated")

	var count int64
	db.Model(&models.RelationshipEdge{}).Where("legacy_relationship_id = ?", legacy.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

// -force must reprocess an already-migrated row rather than skipping it.
func TestMigrateOneRelationship_ForceReprocesses(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")
	alice := createTestContact(t, db, user.ID, "Alice")

	legacy := models.Relationship{
		UserID:           user.ID,
		Name:             "Alice",
		Type:             "Sibling",
		ContactID:        bob.ID,
		RelatedContactID: &alice.ID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	first, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	var firstEdge models.RelationshipEdge
	require.NoError(t, db.First(&firstEdge, "id = ?", first.EdgeID).Error)

	outcome, err := migrateOneRelationship(db, legacy, true, true)
	require.NoError(t, err)
	assert.Empty(t, outcome.Skipped)
	assert.Equal(t, first.EdgeID, outcome.EdgeID, "forcing must update the existing edge in place, not create a second one")

	var count int64
	db.Model(&models.RelationshipEdge{}).Where("legacy_relationship_id = ?", legacy.ID).Count(&count)
	assert.Equal(t, int64(1), count, "exactly one edge must exist after a forced reprocess")

	var reprocessed models.RelationshipEdge
	require.NoError(t, db.First(&reprocessed, "id = ?", outcome.EdgeID).Error)
	assert.Equal(t, firstEdge.CreatedAt.Unix(), reprocessed.CreatedAt.Unix(), "CreatedAt must survive an update, not reset to zero")
	assert.NotEmpty(t, outcome.EdgeID)
}

// Regression test: forcing a reprocess of a name-only relationship used to
// create a SECOND, orphaned thin Contact on every -force run rather than
// reusing the one already created — found by running the real CLI twice
// with -force against a seeded scratch database and watching the contact
// count climb on every run. Must reuse the same Contact (by the prior
// edge's SourceID) and refresh its fields in place.
func TestMigrateOneRelationship_ForceReprocessesNameOnlyWithoutDuplicatingContact(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")

	legacy := models.Relationship{
		UserID:    user.ID,
		Name:      "Grandma Sue",
		Type:      "Grandmother",
		Gender:    "female",
		Birthday:  "1945-03-12",
		ContactID: bob.ID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	first, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	require.True(t, first.CreatedThinContact)
	firstThinUID := first.SourceID

	var contactCountAfterFirst int64
	db.Model(&models.Contact{}).Count(&contactCountAfterFirst)

	// Simulate the legacy row being edited before a forced reprocess.
	legacy.Gender = "other"
	require.NoError(t, db.Save(&legacy).Error)

	second, err := migrateOneRelationship(db, legacy, true, true)
	require.NoError(t, err)
	assert.False(t, second.CreatedThinContact, "the second pass must reuse the existing thin contact, not create another")
	assert.Equal(t, firstThinUID, second.SourceID, "the edge must keep pointing at the same thin contact")
	assert.Equal(t, first.EdgeID, second.EdgeID)

	var contactCountAfterSecond int64
	db.Model(&models.Contact{}).Count(&contactCountAfterSecond)
	assert.Equal(t, contactCountAfterFirst, contactCountAfterSecond, "forcing must not create a second thin contact")

	var thin models.Contact
	require.NoError(t, db.Where("vcard_uid = ?", firstThinUID).First(&thin).Error)
	assert.Equal(t, "other", thin.Gender, "the reused contact's fields must refresh from the current legacy row")
}

// Dry run must perform full resolution (so the report line is accurate) but
// make zero database writes — no thin contact, no edge, for either the
// linked or name-only case.
func TestMigrateOneRelationship_DryRunMakesNoWrites(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")
	alice := createTestContact(t, db, user.ID, "Alice")

	linked := models.Relationship{
		UserID: user.ID, Name: "Alice", Type: "Sibling",
		ContactID: bob.ID, RelatedContactID: &alice.ID,
	}
	require.NoError(t, db.Create(&linked).Error)

	nameOnly := models.Relationship{
		UserID: user.ID, Name: "Uncle Rob", Type: "Uncle",
		ContactID: bob.ID,
	}
	require.NoError(t, db.Create(&nameOnly).Error)

	var contactCountBefore, edgeCountBefore int64
	db.Model(&models.Contact{}).Count(&contactCountBefore)
	db.Model(&models.RelationshipEdge{}).Count(&edgeCountBefore)

	linkedOutcome, err := migrateOneRelationship(db, linked, false, false)
	require.NoError(t, err)
	assert.Empty(t, linkedOutcome.Skipped)
	assert.Equal(t, alice.VCardUID, linkedOutcome.SourceID, "resolution still runs in dry run")
	assert.Empty(t, linkedOutcome.EdgeID)

	nameOnlyOutcome, err := migrateOneRelationship(db, nameOnly, false, false)
	require.NoError(t, err)
	assert.Empty(t, nameOnlyOutcome.Skipped)
	assert.True(t, nameOnlyOutcome.CreatedThinContact)
	assert.Empty(t, nameOnlyOutcome.SourceID, "no contact was actually created, so there is no real VCardUID to report")
	assert.Empty(t, nameOnlyOutcome.EdgeID)

	var contactCountAfter, edgeCountAfter int64
	db.Model(&models.Contact{}).Count(&contactCountAfter)
	db.Model(&models.RelationshipEdge{}).Count(&edgeCountAfter)
	assert.Equal(t, contactCountBefore, contactCountAfter, "dry run must not create the thin contact")
	assert.Equal(t, edgeCountBefore, edgeCountAfter, "dry run must not create any edge")
}

// Symmetric matched types must produce Directional=false, not just the
// asymmetric parent_of/child_of case already covered above.
func TestMigrateOneRelationship_SymmetricTypeIsNotDirectional(t *testing.T) {
	db := setupMigrationTestDB(t)
	user := createTestUser(t, db)
	bob := createTestContact(t, db, user.ID, "Bob")
	alice := createTestContact(t, db, user.ID, "Alice")

	legacy := models.Relationship{
		UserID: user.ID, Name: "Alice", Type: "Friendship",
		ContactID: bob.ID, RelatedContactID: &alice.ID,
	}
	require.NoError(t, db.Create(&legacy).Error)

	outcome, err := migrateOneRelationship(db, legacy, true, false)
	require.NoError(t, err)
	assert.Equal(t, "friend_of", outcome.MatchedType)

	var edge models.RelationshipEdge
	require.NoError(t, db.First(&edge, "id = ?", outcome.EdgeID).Error)
	assert.False(t, edge.Directional)
}
