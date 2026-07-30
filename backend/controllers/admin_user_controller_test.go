package controllers

import (
	"encoding/json"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteUser_CleansUpAllOwnedRows is the regression test for Tier 3c
// item 1 (docs/fork-plan/95-backlog-and-priorities.md): DeleteUser must not
// leave orphaned rows in any of the 14 tables that reference a user, not just
// the handful it originally covered.
func TestDeleteUser_CleansUpAllOwnedRows(t *testing.T) {
	db, router := setupRouter()

	// setupRouter seeds the "tester" user and puts their ID in context as the
	// acting admin — create a second, deletable target user.
	target := models.User{Username: "target", Email: "target@example.com", Password: "password123"}
	require.NoError(t, db.Create(&target).Error)

	contact := models.Contact{UserID: target.ID, Firstname: "Orphan", Lastname: "Check"}
	require.NoError(t, db.Create(&contact).Error)

	require.NoError(t, db.Create(&models.Reminder{UserID: target.ID, ContactID: &contact.ID, Message: "m"}).Error)
	require.NoError(t, db.Create(&models.Note{UserID: target.ID, ContactID: &contact.ID, Content: "n"}).Error)
	require.NoError(t, db.Create(&models.Relationship{UserID: target.ID, ContactID: contact.ID, Name: "r", Type: "friend"}).Error)

	webhook := models.Webhook{UserID: target.ID, URL: "https://example.com/hook", Events: []string{"contact.created"}}
	require.NoError(t, db.Create(&webhook).Error)
	require.NoError(t, db.Create(&models.WebhookDelivery{WebhookID: webhook.ID, EventType: "contact.created", Payload: "{}"}).Error)

	sub := models.ContactSubscription{UserID: target.ID, Name: "sub", URL: "https://example.com/dav"}
	require.NoError(t, db.Create(&sub).Error)
	require.NoError(t, db.Create(&models.ContactSyncLink{SubscriptionID: sub.ID, UserID: target.ID, ContactID: contact.ID, Href: "/dav/1.vcf"}).Error)

	household := models.Household{UserID: target.ID, Name: "h", Type: "other"}
	require.NoError(t, db.Create(&household).Error)
	require.NoError(t, db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: target.ID, MemberVCardUID: contact.VCardUID, Role: "adult"}).Error)

	circle := models.Circle{UserID: target.ID, Name: "c"}
	require.NoError(t, db.Create(&circle).Error)
	require.NoError(t, db.Create(&models.CircleMember{CircleID: circle.ID, UserID: target.ID, MemberVCardUID: contact.VCardUID}).Error)

	tag := models.Tag{UserID: target.ID, Name: "t"}
	require.NoError(t, db.Create(&tag).Error)
	require.NoError(t, db.Create(&models.ContactTag{TagID: tag.ID, UserID: target.ID, ContactVCardUID: contact.VCardUID}).Error)

	fieldDef := models.FieldDefinition{UserID: target.ID, Label: "f", Key: "f", Target: "contact", Type: "text"}
	require.NoError(t, db.Create(&fieldDef).Error)
	require.NoError(t, db.Create(&models.FieldValue{FieldDefinitionID: fieldDef.ID, UserID: target.ID, EntityID: contact.VCardUID, Value: json.RawMessage(`"v"`)}).Error)

	require.NoError(t, db.Create(&models.RelationshipEdge{UserID: target.ID, SourceID: contact.VCardUID, TargetID: contact.VCardUID, Type: "related_to"}).Error)
	require.NoError(t, db.Create(&models.LifeEvent{UserID: target.ID, EntityID: contact.VCardUID, Type: "custom"}).Error)

	require.NoError(t, db.Create(&models.CardDAVSync{UserID: target.ID, SyncToken: "tok", LastModified: time.Now()}).Error)
	require.NoError(t, db.Create(&models.ApiToken{UserID: target.ID, Name: "token", TokenHash: "hash"}).Error)
	require.NoError(t, db.Create(&models.ReminderCompletion{UserID: target.ID, ContactID: contact.ID, Message: "done", CompletedAt: time.Now()}).Error)

	calSub := models.CalendarSubscription{UserID: target.ID, Name: "cal", URL: "https://example.com/cal.ics"}
	require.NoError(t, db.Create(&calSub).Error)
	activity := models.Activity{UserID: target.ID, Title: "call", Type: "call", Date: time.Now()}
	require.NoError(t, db.Create(&activity).Error)
	require.NoError(t, db.Create(&models.CalendarEventLink{SubscriptionID: calSub.ID, UserID: target.ID, UID: "evt-1", ActivityID: activity.ID, ContentHash: "h"}).Error)

	router.DELETE("/users/:id", DeleteUser)

	req, _ := http.NewRequest("DELETE", "/users/"+strconv.Itoa(int(target.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assertGone := func(name string, model any, where string, args ...any) {
		t.Helper()
		var count int64
		require.NoError(t, db.Model(model).Where(where, args...).Count(&count).Error)
		assert.Zero(t, count, "%s rows should be gone after DeleteUser", name)
	}

	assertGone("Reminder", &models.Reminder{}, "user_id = ?", target.ID)
	assertGone("Note", &models.Note{}, "user_id = ?", target.ID)
	assertGone("Relationship", &models.Relationship{}, "user_id = ?", target.ID)
	assertGone("Webhook", &models.Webhook{}, "user_id = ?", target.ID)
	assertGone("WebhookDelivery", &models.WebhookDelivery{}, "webhook_id = ?", webhook.ID)
	assertGone("ContactSubscription", &models.ContactSubscription{}, "user_id = ?", target.ID)
	assertGone("ContactSyncLink", &models.ContactSyncLink{}, "user_id = ?", target.ID)
	assertGone("Household", &models.Household{}, "user_id = ?", target.ID)
	assertGone("HouseholdMember", &models.HouseholdMember{}, "user_id = ?", target.ID)
	assertGone("Circle", &models.Circle{}, "user_id = ?", target.ID)
	assertGone("CircleMember", &models.CircleMember{}, "user_id = ?", target.ID)
	assertGone("Tag", &models.Tag{}, "user_id = ?", target.ID)
	assertGone("ContactTag", &models.ContactTag{}, "user_id = ?", target.ID)
	assertGone("FieldDefinition", &models.FieldDefinition{}, "user_id = ?", target.ID)
	assertGone("FieldValue", &models.FieldValue{}, "user_id = ?", target.ID)
	assertGone("RelationshipEdge", &models.RelationshipEdge{}, "user_id = ?", target.ID)
	assertGone("LifeEvent", &models.LifeEvent{}, "user_id = ?", target.ID)
	assertGone("CardDAVSync", &models.CardDAVSync{}, "user_id = ?", target.ID)
	assertGone("ApiToken", &models.ApiToken{}, "user_id = ?", target.ID)
	assertGone("ReminderCompletion", &models.ReminderCompletion{}, "user_id = ?", target.ID)
	assertGone("CalendarEventLink", &models.CalendarEventLink{}, "user_id = ?", target.ID)
	assertGone("CalendarSubscription", &models.CalendarSubscription{}, "user_id = ?", target.ID)
	assertGone("Contact", &models.Contact{}, "user_id = ?", target.ID)

	var remainingUser models.User
	err := db.First(&remainingUser, target.ID).Error
	assert.Error(t, err, "target user should be deleted")
}
