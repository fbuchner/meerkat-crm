package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/database"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContactMerge_RealMigratedSchema is the real-DB check for ticket N1
// (docs/fork-plan/tickets/01-N1-contact-merge.md): every other test in this
// package uses AutoMigrate against :memory: sqlite, which cannot catch a
// GORM column-tag mismatch against the real migration SQL. This test seeds a
// keeper (Alice) and loser (Bob) plus a third contact (Carol) covering every
// merge case in one pass, then exercises preview -> commit against a
// database.InitDB-migrated real file database, and finally the missing-
// resolution rejection path.
func TestContactMerge_RealMigratedSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "contact-merge-real.db")
	db, err := database.InitDB(dbPath)
	require.NoError(t, err)

	user := models.User{Username: "mergetester", Password: "password123!A", Email: "merge@example.com"}
	require.NoError(t, db.Create(&user).Error)

	alice := models.Contact{
		UserID: user.ID, Firstname: "Alice",
		Emails: []models.ContactEmail{{Type: "home", Value: "alice@home.example"}},
	}
	bob := models.Contact{
		UserID: user.ID, Firstname: "Robert", Lastname: "Smith",
		Emails: []models.ContactEmail{
			{Type: "home", Value: "alice@home.example"}, // duplicate of Alice's -- must dedup
			{Type: "work", Value: "bob@work.example"},   // unique -- must survive the union
		},
	}
	carol := models.Contact{UserID: user.ID, Firstname: "Carol"}
	require.NoError(t, db.Create(&alice).Error)
	require.NoError(t, db.Create(&bob).Error)
	require.NoError(t, db.Create(&carol).Error)

	// Notes / reminders / reminder completions on Bob.
	note := models.Note{UserID: user.ID, ContactID: &bob.ID, Content: "a note about bob"}
	require.NoError(t, db.Create(&note).Error)
	reminder := models.Reminder{UserID: user.ID, ContactID: &bob.ID, Message: "follow up", Recurrence: "once", RemindAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, db.Create(&reminder).Error)
	completion := models.ReminderCompletion{UserID: user.ID, ContactID: bob.ID, ReminderID: &reminder.ID, Message: "done", CompletedAt: time.Now()}
	require.NoError(t, db.Create(&completion).Error)

	// Activities: one shared (dedup case), one Bob-only (plain repoint).
	sharedActivity := models.Activity{UserID: user.ID, Title: "Dinner", Contacts: []models.Contact{alice, bob}}
	require.NoError(t, db.Create(&sharedActivity).Error)
	bobOnlyActivity := models.Activity{UserID: user.ID, Title: "Coffee", Contacts: []models.Contact{bob}}
	require.NoError(t, db.Create(&bobOnlyActivity).Error)

	// Relationship edges: a direct Alice<->Bob edge (becomes a self-loop),
	// and an inverse pair through Carol (exactly one must survive).
	directEdge := models.RelationshipEdge{
		UserID: user.ID, SourceID: alice.VCardUID, TargetID: bob.VCardUID, Type: "friend_of",
		Source: models.RelationshipSourceUserConfirmed, Confidence: 1.0, Status: models.RelationshipStatusConfirmed,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&directEdge).Error)
	bobParentEdge := models.RelationshipEdge{
		UserID: user.ID, SourceID: bob.VCardUID, TargetID: carol.VCardUID, Type: "parent_of",
		Source: models.RelationshipSourceUserConfirmed, Confidence: 1.0, Status: models.RelationshipStatusConfirmed,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&bobParentEdge).Error)
	// The same fact as bobParentEdge ("bob parent_of carol", which becomes
	// "alice parent_of carol" after repoint), recorded from carol's side
	// instead: "carol child_of alice" -- source/target swapped, type
	// inverted. This is the genuine inverse-pair collision the dedup logic
	// must catch, distinct from two edges that merely disagree.
	carolChildEdge := models.RelationshipEdge{
		UserID: user.ID, SourceID: carol.VCardUID, TargetID: alice.VCardUID, Type: "child_of",
		Source: models.RelationshipSourceHouseholdInferred, Confidence: 0.5, Status: models.RelationshipStatusSuggested,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&carolChildEdge).Error)

	// Household: Alice and Bob both already members (duplicate case).
	household := models.Household{UserID: user.ID, Name: "The House", Type: models.HouseholdTypeRoommates}
	require.NoError(t, db.Create(&household).Error)
	require.NoError(t, db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: alice.VCardUID}).Error)
	require.NoError(t, db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: bob.VCardUID}).Error)

	// Circle / Tag: Bob-only (plain repoint).
	circle := models.Circle{UserID: user.ID, Name: "College Friends"}
	require.NoError(t, db.Create(&circle).Error)
	require.NoError(t, db.Create(&models.CircleMember{CircleID: circle.ID, UserID: user.ID, MemberVCardUID: bob.VCardUID}).Error)
	tag := models.Tag{UserID: user.ID, Name: "vip"}
	require.NoError(t, db.Create(&tag).Error)
	require.NoError(t, db.Create(&models.ContactTag{TagID: tag.ID, UserID: user.ID, ContactVCardUID: bob.VCardUID}).Error)

	// Custom field values: one Bob-only (plain repoint), one on both
	// (collision -- must appear as a conflict).
	bobOnlyDef := models.FieldDefinition{UserID: user.ID, Label: "Favorite Color", Key: "favorite_color", Target: "contact", Type: "string", Projection: "internal-only", Sensitivity: "normal"}
	require.NoError(t, db.Create(&bobOnlyDef).Error)
	require.NoError(t, db.Create(&models.FieldValue{FieldDefinitionID: bobOnlyDef.ID, UserID: user.ID, EntityID: bob.VCardUID, Value: json.RawMessage(`"blue"`)}).Error)

	sharedDef := models.FieldDefinition{UserID: user.ID, Label: "T-Shirt Size", Key: "tshirt_size", Target: "contact", Type: "string", Projection: "internal-only", Sensitivity: "normal"}
	require.NoError(t, db.Create(&sharedDef).Error)
	require.NoError(t, db.Create(&models.FieldValue{FieldDefinitionID: sharedDef.ID, UserID: user.ID, EntityID: alice.VCardUID, Value: json.RawMessage(`"M"`)}).Error)
	require.NoError(t, db.Create(&models.FieldValue{FieldDefinitionID: sharedDef.ID, UserID: user.ID, EntityID: bob.VCardUID, Value: json.RawMessage(`"L"`)}).Error)

	// Life events: Bob's own (repoint), and Carol's referencing Bob as a
	// secondary participant (the RelatedEntityIDs JSON-array rewrite case).
	bobLifeEvent := models.LifeEvent{UserID: user.ID, EntityID: bob.VCardUID, Type: "moved"}
	require.NoError(t, db.Create(&bobLifeEvent).Error)
	carolLifeEvent := models.LifeEvent{UserID: user.ID, EntityID: carol.VCardUID, Type: "married", RelatedEntityIDs: []string{bob.VCardUID}}
	require.NoError(t, db.Create(&carolLifeEvent).Error)

	// ContactSyncLink on Bob (must be discarded, not repointed).
	subscription := models.ContactSubscription{UserID: user.ID, Name: "Test CardDAV", URL: "https://example.com/dav"}
	require.NoError(t, db.Create(&subscription).Error)
	syncLink := models.ContactSyncLink{SubscriptionID: subscription.ID, UserID: user.ID, Href: "/dav/bob.vcf", ContactID: bob.ID, ContentHash: "abc123"}
	require.NoError(t, db.Create(&syncLink).Error)

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", user.ID)
		c.Next()
	})
	router.POST("/contacts/merge/preview", withValidated(func() any { return &models.ContactMergeRequest{} }), PreviewContactMerge)
	router.POST("/contacts/merge", withValidated(func() any { return &models.ContactMergeRequest{} }), CommitContactMerge)

	doJSON := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			require.NoError(t, json.NewEncoder(&buf).Encode(body))
		}
		req, _ := http.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	// --- Preview ---
	previewResp := doJSON("POST", "/contacts/merge/preview", models.ContactMergeRequest{KeepID: alice.ID, MergeID: bob.ID})
	require.Equal(t, http.StatusOK, previewResp.Code, previewResp.Body.String())
	var preview models.ContactMergePreviewResponse
	require.NoError(t, json.Unmarshal(previewResp.Body.Bytes(), &preview))

	require.Len(t, preview.Resolution.Conflicts, 1, "firstname must be the only scalar conflict")
	assert.Equal(t, "firstname", preview.Resolution.Conflicts[0].Field)
	assert.Equal(t, "Alice", preview.Resolution.Conflicts[0].KeeperValue)
	assert.Equal(t, "Robert", preview.Resolution.Conflicts[0].LoserValue)
	assert.Equal(t, "Smith", preview.Resolution.ResolvedScalars["lastname"], "lastname only set on the loser must be auto-resolved")

	require.Len(t, preview.Resolution.Emails, 2, "duplicate email deduped, unique email unioned")

	require.Len(t, preview.Resolution.FieldValueConflicts, 1, "the shared custom field must be a conflict")
	assert.Equal(t, sharedDef.ID, preview.Resolution.FieldValueConflicts[0].Field)
	assert.Equal(t, `"M"`, preview.Resolution.FieldValueConflicts[0].KeeperValue)
	assert.Equal(t, `"L"`, preview.Resolution.FieldValueConflicts[0].LoserValue)

	counts := preview.AssociationCounts
	assert.EqualValues(t, 1, counts.Notes)
	assert.EqualValues(t, 2, counts.Activities)
	assert.EqualValues(t, 1, counts.Reminders)
	assert.EqualValues(t, 1, counts.ReminderCompletions)
	assert.EqualValues(t, 2, counts.RelationshipEdges)
	assert.EqualValues(t, 1, counts.HouseholdMemberships)
	assert.EqualValues(t, 1, counts.CircleMemberships)
	assert.EqualValues(t, 1, counts.Tags)
	assert.EqualValues(t, 1, counts.LifeEvents)
	assert.EqualValues(t, 1, counts.LifeEventReferences)
	assert.EqualValues(t, 2, counts.FieldValues)
	assert.EqualValues(t, 1, counts.ContactSyncLinks)

	// --- Commit without a resolution for the known conflicts: rejected, no partial merge ---
	rejectResp := doJSON("POST", "/contacts/merge", models.ContactMergeRequest{KeepID: alice.ID, MergeID: bob.ID})
	require.Equal(t, http.StatusBadRequest, rejectResp.Code, rejectResp.Body.String())
	assert.Contains(t, rejectResp.Body.String(), "firstname")

	var untouchedAlice models.Contact
	require.NoError(t, db.First(&untouchedAlice, alice.ID).Error)
	assert.Equal(t, "Alice", untouchedAlice.Firstname, "a rejected commit must not have modified the keeper")
	var untouchedBobCount int64
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", bob.ID).Count(&untouchedBobCount).Error)
	assert.EqualValues(t, 1, untouchedBobCount, "a rejected commit must not have soft-deleted the loser")

	// --- Commit with both conflicts resolved ---
	commitResp := doJSON("POST", "/contacts/merge", models.ContactMergeRequest{
		KeepID: alice.ID, MergeID: bob.ID,
		Resolutions: map[string]string{"firstname": "Alice", sharedDef.ID: `"L"`},
	})
	require.Equal(t, http.StatusOK, commitResp.Code, commitResp.Body.String())

	// Keeper's fields.
	var keeper models.Contact
	require.NoError(t, db.First(&keeper, alice.ID).Error)
	assert.Equal(t, "Alice", keeper.Firstname)
	assert.Equal(t, "Smith", keeper.Lastname)
	assert.Len(t, keeper.Emails, 2)

	// Loser soft-deleted, not hard-deleted.
	var scopedCount int64
	require.NoError(t, db.Model(&models.Contact{}).Where("id = ?", bob.ID).Count(&scopedCount).Error)
	assert.EqualValues(t, 0, scopedCount, "plain query must not see the soft-deleted loser")
	var unscopedCount int64
	require.NoError(t, db.Unscoped().Model(&models.Contact{}).Where("id = ?", bob.ID).Count(&unscopedCount).Error)
	assert.EqualValues(t, 1, unscopedCount, "Unscoped query must still find the soft-deleted loser")

	// Notes / reminders / reminder completions re-pointed, zero orphans.
	assertZero := func(model any, cond string, args ...any) {
		t.Helper()
		var count int64
		require.NoError(t, db.Model(model).Where(cond, args...).Count(&count).Error)
		assert.EqualValues(t, 0, count, "orphan rows still reference the loser: %v", cond)
	}
	assertZero(&models.Note{}, "contact_id = ?", bob.ID)
	assertZero(&models.Reminder{}, "contact_id = ?", bob.ID)
	assertZero(&models.ReminderCompletion{}, "contact_id = ?", bob.ID)
	assertZero(&models.HouseholdMember{}, "member_vcard_uid = ?", bob.VCardUID)
	assertZero(&models.CircleMember{}, "member_vcard_uid = ?", bob.VCardUID)
	assertZero(&models.ContactTag{}, "contact_vcard_uid = ?", bob.VCardUID)
	assertZero(&models.FieldValue{}, "entity_id = ?", bob.VCardUID)
	assertZero(&models.LifeEvent{}, "entity_id = ?", bob.VCardUID)
	assertZero(&models.ContactSyncLink{}, "contact_id = ?", bob.ID)
	assertZero(&models.RelationshipEdge{}, "source_id = ? OR target_id = ?", bob.VCardUID, bob.VCardUID)

	var notesForKeeper, remindersForKeeper int64
	require.NoError(t, db.Model(&models.Note{}).Where("contact_id = ?", alice.ID).Count(&notesForKeeper).Error)
	require.NoError(t, db.Model(&models.Reminder{}).Where("contact_id = ?", alice.ID).Count(&remindersForKeeper).Error)
	assert.EqualValues(t, 2, notesForKeeper, "Bob's repointed note plus the merge audit note")
	assert.EqualValues(t, 1, remindersForKeeper)

	// activity_contacts: shared activity has exactly one row for Alice (no
	// PK violation, no duplicate); Bob-only activity now points to Alice.
	var sharedActivityRows, bobOnlyActivityRows int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM activity_contacts WHERE activity_id = ? AND contact_id = ?", sharedActivity.ID, alice.ID).Scan(&sharedActivityRows).Error)
	assert.EqualValues(t, 1, sharedActivityRows)
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM activity_contacts WHERE activity_id = ? AND contact_id = ?", bobOnlyActivity.ID, alice.ID).Scan(&bobOnlyActivityRows).Error)
	assert.EqualValues(t, 1, bobOnlyActivityRows)
	var totalActivityRows int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM activity_contacts WHERE activity_id = ? AND contact_id = ?", sharedActivity.ID, bob.ID).Scan(&totalActivityRows).Error)
	assert.EqualValues(t, 0, totalActivityRows)

	// Household membership deduped: exactly one row for Alice, not two.
	var householdRows int64
	require.NoError(t, db.Model(&models.HouseholdMember{}).Where("household_id = ? AND member_vcard_uid = ?", household.ID, alice.VCardUID).Count(&householdRows).Error)
	assert.EqualValues(t, 1, householdRows)

	// Circle/Tag re-pointed to Alice.
	var circleRows, tagRows int64
	require.NoError(t, db.Model(&models.CircleMember{}).Where("circle_id = ? AND member_vcard_uid = ?", circle.ID, alice.VCardUID).Count(&circleRows).Error)
	assert.EqualValues(t, 1, circleRows)
	require.NoError(t, db.Model(&models.ContactTag{}).Where("tag_id = ? AND contact_vcard_uid = ?", tag.ID, alice.VCardUID).Count(&tagRows).Error)
	assert.EqualValues(t, 1, tagRows)

	// FieldValue: Bob-only value re-pointed; the colliding one resolved to
	// the chosen value ("L"), not silently kept as Alice's original ("M").
	var bobOnlyValue, sharedValue models.FieldValue
	require.NoError(t, db.Where("field_definition_id = ? AND entity_id = ?", bobOnlyDef.ID, alice.VCardUID).First(&bobOnlyValue).Error)
	assert.JSONEq(t, `"blue"`, string(bobOnlyValue.Value))
	require.NoError(t, db.Where("field_definition_id = ? AND entity_id = ?", sharedDef.ID, alice.VCardUID).First(&sharedValue).Error)
	assert.JSONEq(t, `"L"`, string(sharedValue.Value), "the resolved conflict value must win, not the keeper's original")
	var sharedValueCount int64
	require.NoError(t, db.Model(&models.FieldValue{}).Where("field_definition_id = ?", sharedDef.ID).Count(&sharedValueCount).Error)
	assert.EqualValues(t, 1, sharedValueCount, "the collision must leave exactly one row, not two")

	// LifeEvent: Bob's own subject re-pointed; Carol's RelatedEntityIDs
	// rewritten from Bob's VCardUID to Alice's.
	var repointedLifeEvent models.LifeEvent
	require.NoError(t, db.Where("id = ?", bobLifeEvent.ID).First(&repointedLifeEvent).Error)
	assert.Equal(t, alice.VCardUID, repointedLifeEvent.EntityID)

	var reloadedCarolEvent models.LifeEvent
	require.NoError(t, db.Where("id = ?", carolLifeEvent.ID).First(&reloadedCarolEvent).Error)
	assert.Equal(t, []string{alice.VCardUID}, reloadedCarolEvent.RelatedEntityIDs)

	// RelationshipEdge: self-loop dropped entirely; of the inverse pair,
	// exactly one survives -- and per the tie-break (higher Confidence /
	// confirmed status) it must be the one that started as Bob's parent_of.
	var selfLoopCount int64
	require.NoError(t, db.Model(&models.RelationshipEdge{}).
		Where("user_id = ? AND source_id = ? AND target_id = ?", user.ID, alice.VCardUID, alice.VCardUID).
		Count(&selfLoopCount).Error)
	assert.EqualValues(t, 0, selfLoopCount, "the direct Alice<->Bob edge must become a self-loop and be dropped")

	var aliceCarolEdges []models.RelationshipEdge
	require.NoError(t, db.Where(
		"user_id = ? AND ((source_id = ? AND target_id = ?) OR (source_id = ? AND target_id = ?))",
		user.ID, alice.VCardUID, carol.VCardUID, carol.VCardUID, alice.VCardUID,
	).Find(&aliceCarolEdges).Error)
	require.Len(t, aliceCarolEdges, 1, "exactly one of the inverse pair must survive")
	assert.Equal(t, "parent_of", aliceCarolEdges[0].Type, "the higher-confidence, confirmed edge must be the survivor")
	assert.Equal(t, alice.VCardUID, aliceCarolEdges[0].SourceID)
	assert.Equal(t, models.RelationshipStatusConfirmed, aliceCarolEdges[0].Status)

	// Audit note exists on the keeper and names the merge.
	var mergeNote models.Note
	require.NoError(t, db.Where("contact_id = ? AND content LIKE ?", alice.ID, "%Merged contact%").First(&mergeNote).Error)
	assert.Contains(t, mergeNote.Content, "Robert")
}

// TestContactMerge_KeepEqualsMerge_Rejected is a lightweight guard against
// the trivial self-merge case, on AutoMigrate (no real-DB dependency needed
// for this check).
func TestContactMerge_KeepEqualsMerge_Rejected(t *testing.T) {
	db, router := setupRouter()
	var user models.User
	require.NoError(t, db.First(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Solo"}
	require.NoError(t, db.Create(&contact).Error)

	router.POST("/contacts/merge/preview", withValidated(func() any { return &models.ContactMergeRequest{} }), PreviewContactMerge)

	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(models.ContactMergeRequest{KeepID: contact.ID, MergeID: contact.ID}))
	req, _ := http.NewRequest("POST", "/contacts/merge/preview", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
