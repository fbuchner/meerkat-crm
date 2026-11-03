package controllers

import (
	"fmt"
	"meerkat/config"
	"meerkat/database"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
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

func adminRouter(db *gorm.DB, callerID uint, setup ...gin.HandlerFunc) *gin.Engine {
	router := routerForUser(db, callerID)
	router.Use(setup...)
	validated := middleware.ValidateJSONMiddleware(&models.AdminUserUpdateInput{})
	group := router.Group("/admin", middleware.AdminMiddleware())
	group.GET("/users", ListUsers)
	group.GET("/users/:id", GetUser)
	group.PATCH("/users/:id", validated, UpdateUser)
	group.DELETE("/users/:id", DeleteUser)
	return router
}

func TestAdminEndpoints_RefuseNonAdmins(t *testing.T) {
	db := newTestDB(t)
	caller := newTestUser(t, db, "caller", false)
	target := newTestUser(t, db, "target", false)
	router := adminRouter(db, caller.ID)
	targetPath := fmt.Sprintf("/admin/users/%d", target.ID)

	assert.Equal(t, http.StatusForbidden, doRequest(router, http.MethodGet, "/admin/users", nil, "").Code)
	assert.Equal(t, http.StatusForbidden, doRequest(router, http.MethodGet, targetPath, nil, "").Code)
	assert.Equal(t, http.StatusForbidden, doRequest(router, http.MethodDelete, targetPath, nil, "").Code)
	assert.Equal(t, http.StatusForbidden, doJSON(t, router, http.MethodPatch, fmt.Sprintf("/admin/users/%d", caller.ID), gin.H{"is_admin": true}).Code)

	var survivor, reloaded models.User
	require.NoError(t, db.First(&survivor, target.ID).Error, "the target must survive")
	require.NoError(t, db.First(&reloaded, caller.ID).Error)
	assert.False(t, reloaded.IsAdmin, "a non-admin must not be able to promote themselves")
}

func TestAdminEndpoints_RefuseAPITokensEvenForAdmins(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	router := adminRouter(db, admin.ID, func(c *gin.Context) { c.Set("isAPIToken", true); c.Next() })

	assert.Equal(t, http.StatusForbidden, doRequest(router, http.MethodGet, "/admin/users", nil, "").Code)
}

func TestListUsers_PaginatesAndOmitsSecrets(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	for i := range 4 {
		newTestUser(t, db, fmt.Sprintf("user%d", i), false)
	}
	router := adminRouter(db, admin.ID)

	w := doRequest(router, http.MethodGet, "/admin/users?page=2&limit=2", nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, strings.ToLower(w.Body.String()), "password")

	body := decodeJSON[models.AdminUsersListResponse](t, w)
	assert.EqualValues(t, 5, body.Total)
	assert.Equal(t, 3, body.TotalPages)
	assert.Equal(t, 2, body.Page)
	require.Len(t, body.Users, 2)
	assert.Equal(t, "user1", body.Users[0].Username)
}

func TestGetUser(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	router := adminRouter(db, admin.ID)

	w := doRequest(router, http.MethodGet, fmt.Sprintf("/admin/users/%d", admin.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "admin", decodeJSON[models.AdminUserResponse](t, w).Username)
	assert.NotContains(t, strings.ToLower(w.Body.String()), "password")

	assert.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodGet, "/admin/users/abc", nil, "").Code)
	assert.Equal(t, http.StatusNotFound, doRequest(router, http.MethodGet, "/admin/users/9999", nil, "").Code)
}

func TestGetCurrentUser(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db, "me", false)
	router := routerForUser(db, user.ID)
	router.GET("/users/me", GetCurrentUser)

	w := doRequest(router, http.MethodGet, "/users/me", nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "me", decodeJSON[models.CurrentUserResponse](t, w).Username)
	assert.NotContains(t, strings.ToLower(w.Body.String()), "password")

	gone := routerForUser(db, 9999)
	gone.GET("/users/me", GetCurrentUser)
	assert.Equal(t, http.StatusNotFound, doRequest(gone, http.MethodGet, "/users/me", nil, "").Code)
}

func TestUpdateUser_AppliesFields(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	target := newTestUser(t, db, "target", false)
	router := adminRouter(db, admin.ID)

	w := doJSON(t, router, http.MethodPatch, fmt.Sprintf("/admin/users/%d", target.ID), gin.H{
		"username": "NewName", "email": "New@Example.com", "password": "Sup3r-Secret-Pass!", "is_admin": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), "Sup3r-Secret-Pass!")

	var reloaded models.User
	require.NoError(t, db.First(&reloaded, target.ID).Error)
	assert.Equal(t, "newname", reloaded.Username)
	assert.Equal(t, "new@example.com", reloaded.Email)
	assert.True(t, reloaded.IsAdmin)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(reloaded.Password), []byte("Sup3r-Secret-Pass!")), "the password must be stored hashed")
	assert.NotNil(t, reloaded.PasswordChangedAt, "existing sessions must stop being renewed")
}

func TestUpdateUser_Errors(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	target := newTestUser(t, db, "target", false)
	router := adminRouter(db, admin.ID)
	targetPath := fmt.Sprintf("/admin/users/%d", target.ID)

	assert.Equal(t, http.StatusBadRequest, doJSON(t, router, http.MethodPatch, "/admin/users/abc", gin.H{}).Code)
	assert.Equal(t, http.StatusNotFound, doJSON(t, router, http.MethodPatch, "/admin/users/9999", gin.H{}).Code)
	assert.Equal(t, http.StatusConflict, doJSON(t, router, http.MethodPatch, targetPath, gin.H{"username": "admin"}).Code, "duplicate username")

	var reloaded models.User
	require.NoError(t, db.First(&reloaded, target.ID).Error)
	assert.Equal(t, "target", reloaded.Username)
}

func TestUpdateUser_CannotRemoveOwnAdminStatus(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	newTestUser(t, db, "second", true)
	router := adminRouter(db, admin.ID)

	w := doJSON(t, router, http.MethodPatch, fmt.Sprintf("/admin/users/%d", admin.ID), gin.H{"is_admin": false})

	assert.Equal(t, http.StatusForbidden, w.Code)
	var reloaded models.User
	require.NoError(t, db.First(&reloaded, admin.ID).Error)
	assert.True(t, reloaded.IsAdmin)
}

func TestUpdateUser_AnotherAdminCanBeDemotedButNotTheLastOne(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	second := newTestUser(t, db, "second", true)
	router := adminRouter(db, admin.ID)

	w := doJSON(t, router, http.MethodPatch, fmt.Sprintf("/admin/users/%d", second.ID), gin.H{"is_admin": false})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var reloaded models.User
	require.NoError(t, db.First(&reloaded, second.ID).Error)
	assert.False(t, reloaded.IsAdmin)

	// The route can only be reached by an admin, who cannot demote themselves, so this guard
	// only matters if the caller lost their admin flag mid-request. Call the handler directly.
	direct := routerForUser(db, second.ID)
	direct.PATCH("/users/:id", middleware.ValidateJSONMiddleware(&models.AdminUserUpdateInput{}), UpdateUser)
	w = doJSON(t, direct, http.MethodPatch, fmt.Sprintf("/users/%d", admin.ID), gin.H{"is_admin": false})
	assert.Equal(t, http.StatusForbidden, w.Code)
	var lastAdmin models.User
	require.NoError(t, db.First(&lastAdmin, admin.ID).Error)
	assert.True(t, lastAdmin.IsAdmin, "the last admin must stay an admin")
}

func TestDeleteUser_Guards(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	router := adminRouter(db, admin.ID)

	assert.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodDelete, "/admin/users/abc", nil, "").Code)
	assert.Equal(t, http.StatusNotFound, doRequest(router, http.MethodDelete, "/admin/users/9999", nil, "").Code)
	assert.Equal(t, http.StatusForbidden, doRequest(router, http.MethodDelete, fmt.Sprintf("/admin/users/%d", admin.ID), nil, "").Code, "cannot delete yourself")

	// Same mid-request case as for demotion: only reachable when the caller is no longer an admin.
	formerAdmin := newTestUser(t, db, "former", false)
	direct := routerForUser(db, formerAdmin.ID)
	direct.DELETE("/users/:id", DeleteUser)
	assert.Equal(t, http.StatusForbidden, doRequest(direct, http.MethodDelete, fmt.Sprintf("/users/%d", admin.ID), nil, "").Code, "cannot delete the last admin")

	var reloaded models.User
	assert.NoError(t, db.First(&reloaded, admin.ID).Error)
}

func TestDeleteUser_AnotherAdminCanBeDeleted(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	second := newTestUser(t, db, "second", true)
	router := adminRouter(db, admin.ID)

	w := doRequest(router, http.MethodDelete, fmt.Sprintf("/admin/users/%d", second.ID), nil, "")

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var reloaded models.User
	assert.ErrorIs(t, db.First(&reloaded, second.ID).Error, gorm.ErrRecordNotFound)
}

func TestDeleteUser_FreesTheUsernameAndEmailForReuse(t *testing.T) {
	db := newTestDB(t)
	admin := newTestUser(t, db, "admin", true)
	target := newTestUser(t, db, "target", false)
	router := adminRouter(db, admin.ID)
	require.Equal(t, http.StatusOK, doRequest(router, http.MethodDelete, fmt.Sprintf("/admin/users/%d", target.ID), nil, "").Code)

	again := models.User{Username: "target", Email: "target@example.com", Password: "x"}
	assert.NoError(t, db.Create(&again).Error)
}

func TestTriggerReminders(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db, "admin", true)
	contact := newTestContact(t, db, user.ID, "Due")
	byMail := true
	reminder := models.Reminder{UserID: user.ID, ContactID: &contact.ID, Message: "m", ByMail: &byMail, Recurrence: "once", RemindAt: time.Now().Add(-time.Hour)}
	require.NoError(t, db.Create(&reminder).Error)

	cfg := config.Config{ReminderTimezone: "UTC"}
	router := routerForUser(db, user.ID)
	router.POST("/trigger", func(c *gin.Context) { TriggerReminders(c, cfg) })

	w := doRequest(router, http.MethodPost, "/trigger", nil, "")

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var reloaded models.Reminder
	require.NoError(t, db.First(&reloaded, reminder.ID).Error)
	assert.False(t, reloaded.EmailSent, "with no email channel configured the reminder must stay pending")
}
