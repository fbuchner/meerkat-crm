package controllers

import (
	"fmt"
	"meerkat/database"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedUserData creates at least one row in every table holding user data.
func seedUserData(t *testing.T, db *gorm.DB, user models.User) {
	t.Helper()
	contact := models.Contact{UserID: user.ID, Firstname: user.Username}
	require.NoError(t, db.Create(&contact).Error)
	activity := models.Activity{UserID: user.ID, Title: "a", Date: time.Now()}
	require.NoError(t, db.Create(&activity).Error)
	require.NoError(t, db.Exec("INSERT INTO activity_contacts (activity_id, contact_id) VALUES (?, ?)", activity.ID, contact.ID).Error)
	reminder := models.Reminder{UserID: user.ID, Message: "m", RemindAt: time.Now(), Recurrence: "once", ContactID: &contact.ID}
	webhook := models.Webhook{UserID: user.ID, Name: "w", URL: "http://example.com", Secret: "secret", Events: []string{"e"}}
	softDeletedWebhook := models.Webhook{UserID: user.ID, Name: "gone", URL: "http://example.com", Secret: "old-secret", Events: []string{"e"}}
	calendar := models.CalendarSubscription{UserID: user.ID, Name: "c", URL: "http://example.com", PasswordEncrypted: "enc"}
	connection := models.CardDAVConnection{UserID: user.ID, BaseURL: "http://example.com", AddressBookPath: "/", PasswordEncrypted: "enc"}

	for _, row := range []any{
		&reminder, &webhook, &softDeletedWebhook, &calendar, &connection,
		&models.Note{UserID: user.ID, Content: "n", Date: time.Now(), ContactID: &contact.ID},
		&models.EmploymentHistory{UserID: user.ID, ContactID: contact.ID, Organization: "o"},
		&models.Relationship{UserID: user.ID, Name: "r", Type: "Friend", ContactID: contact.ID},
		&models.ApiToken{UserID: user.ID, Name: "t", TokenHash: "hash-" + user.Username},
		&models.CardDAVSync{UserID: user.ID, SyncToken: "s", LastModified: time.Now()},
		&models.ContactDuplicateDismissal{UserID: user.ID, ContactAID: contact.ID, ContactBID: contact.ID + 1},
		&models.ReminderCompletion{UserID: user.ID, ContactID: contact.ID, Message: "m", CompletedAt: time.Now()},
	} {
		require.NoError(t, db.Create(row).Error)
	}
	require.NoError(t, db.Create(&models.WebhookDelivery{WebhookID: webhook.ID, EventType: "e", Payload: "p"}).Error)
	require.NoError(t, db.Create(&models.WebhookDelivery{WebhookID: softDeletedWebhook.ID, EventType: "e", Payload: "p"}).Error)
	require.NoError(t, db.Delete(&softDeletedWebhook).Error)
	require.NoError(t, db.Create(&models.CalendarEventLink{SubscriptionID: calendar.ID, UserID: user.ID, UID: "u", ActivityID: activity.ID, ContentHash: "h"}).Error)
	require.NoError(t, db.Create(&models.CardDAVContactLink{ConnectionID: connection.ID, UserID: user.ID, ContactID: contact.ID, RemoteUID: "u", RemotePath: "/p"}).Error)
}

// userOwnedTables lists the tables with a user_id column in the migrated schema.
func userOwnedTables(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var tables []string
	require.NoError(t, db.Raw(`SELECT m.name FROM sqlite_master m, pragma_table_info(m.name) c
		WHERE m.type = 'table' AND c.name = 'user_id' ORDER BY m.name`).Scan(&tables).Error)
	require.NotEmpty(t, tables)
	return tables
}

func countRows(t *testing.T, db *gorm.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(query, args...).Scan(&n).Error)
	return n
}

// Runs against the real migrations so that a new table with a user_id column fails
// here until DeleteUser handles it and seedUserData covers it.
func TestDeleteUser_RemovesEverythingTheUserOwns(t *testing.T) {
	db, err := database.InitDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)

	admin := models.User{Username: "admin", Email: "admin@example.com", Password: "x", IsAdmin: true}
	target := models.User{Username: "target", Email: "target@example.com", Password: "x"}
	other := models.User{Username: "other", Email: "other@example.com", Password: "x"}
	for _, u := range []*models.User{&admin, &target, &other} {
		require.NoError(t, db.Create(u).Error)
	}
	seedUserData(t, db, target)
	seedUserData(t, db, other)

	tables := userOwnedTables(t, db)
	userRows := func(table string, id uint) int64 {
		return countRows(t, db, fmt.Sprintf("SELECT COUNT(*) FROM %q WHERE user_id = ?", table), id)
	}
	otherRowsBefore := map[string]int64{}
	for _, table := range tables {
		require.NotZero(t, userRows(table, target.ID), "seedUserData must cover %s, and DeleteUser must remove it", table)
		otherRowsBefore[table] = userRows(table, other.ID)
	}
	otherDeliveries := countRows(t, db, "SELECT COUNT(*) FROM webhook_deliveries WHERE webhook_id IN (SELECT id FROM webhooks WHERE user_id = ?)", other.ID)
	require.NotZero(t, otherDeliveries)

	router := routerForUser(db, admin.ID)
	router.DELETE("/admin/users/:id", DeleteUser)
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/admin/users/%d", target.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	for _, table := range tables {
		live := userRows(table, target.ID)
		if countRows(t, db, "SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'deleted_at'", table) > 0 {
			live = countRows(t, db, fmt.Sprintf("SELECT COUNT(*) FROM %q WHERE user_id = ? AND deleted_at IS NULL", table), target.ID)
		}
		assert.Zero(t, live, "%s still has live rows of the deleted user", table)
		assert.Equal(t, otherRowsBefore[table], userRows(table, other.ID), "%s lost rows of another user", table)
	}

	// Credentials and outgoing payloads must be gone, not just hidden
	for _, table := range []string{"webhooks", "calendar_subscriptions", "calendar_event_links", "carddav_connections", "carddav_contact_links", "carddav_sync"} {
		assert.Zero(t, userRows(table, target.ID), table)
	}
	assert.Equal(t, otherDeliveries, countRows(t, db, "SELECT COUNT(*) FROM webhook_deliveries"),
		"only the other user's deliveries may remain, not those of the deleted user's soft-deleted webhooks")
	assert.Equal(t, int64(1), countRows(t, db, "SELECT COUNT(*) FROM activity_contacts"))
}
