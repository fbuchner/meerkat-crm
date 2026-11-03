package controllers

import (
	"meerkat/config"
	"meerkat/models"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A database outage must surface as a server error: reporting it as "not found" would tell the
// frontend a contact or connection does not exist, and a 200 would hide the failure.
func TestHandlers_DatabaseFailureIsAServerError(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db, "owner", true)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	cfg := &config.Config{ProfilePhotoDir: t.TempDir(), JWTSecretKey: testJWTSecret}
	router := routerAs(db, user.ID, *cfg)
	validated := func(factory func() any) gin.HandlerFunc { return withValidated(factory) }
	reminder := validated(func() any { return &models.Reminder{} })
	calendar := validated(func() any { return &models.CalendarSubscriptionInput{} })
	adminUpdate := validated(func() any { return &models.AdminUserUpdateInput{} })

	router.GET("/users/me", GetCurrentUser)
	router.GET("/admin/users", ListUsers)
	router.GET("/admin/users/:id", GetUser)
	router.PATCH("/admin/users/:id", adminUpdate, UpdateUser)
	router.DELETE("/admin/users/:id", DeleteUser)
	router.POST("/trigger", func(c *gin.Context) { TriggerReminders(c, *cfg) })
	router.GET("/contacts/:id/picture", func(c *gin.Context) { GetProfilePicture(c, cfg) })
	router.POST("/contacts/:id/picture", func(c *gin.Context) { AddPhotoToContact(c, cfg) })
	router.GET("/graph", GetGraph)
	router.GET("/reminders", GetAllReminders)
	router.GET("/reminders/upcoming", GetUpcomingReminders)
	router.GET("/reminders/:id", GetReminder)
	router.PUT("/reminders/:id", reminder, UpdateReminder)
	router.DELETE("/reminders/:id", DeleteReminder)
	router.POST("/reminders/:id/complete", CompleteReminder)
	router.POST("/contacts/:id/reminders", reminder, CreateReminder)
	router.GET("/contacts/:id/reminders", GetRemindersForContact)
	router.GET("/contacts/:id/completions", GetCompletionsForContact)
	router.DELETE("/completions/:id", DeleteCompletion)
	router.GET("/calendars", ListCalendarSubscriptions)
	router.POST("/calendars", calendar, CreateCalendarSubscription)
	router.PUT("/calendars/:id", calendar, UpdateCalendarSubscription)
	router.DELETE("/calendars/:id", DeleteCalendarSubscription)
	router.POST("/calendars/:id/sync", SyncCalendarSubscription)
	router.GET("/carddav", GetCardDAVConnection)
	router.DELETE("/carddav", DeleteCardDAVConnection)
	router.POST("/carddav/sync", SyncCardDAVConnection)
	router.PUT("/carddav", validated(func() any { return &models.CardDAVConnectionInput{} }), SaveCardDAVConnection)

	reminderBody := gin.H{"message": "m", "recurrence": "once", "contact_id": 1, "remind_at": "2030-01-01T00:00:00Z"}
	calendarBody := gin.H{"name": "n", "url": "https://cal.example.com/"}
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/users/me", nil},
		{http.MethodGet, "/admin/users", nil},
		{http.MethodGet, "/admin/users/2", nil},
		{http.MethodPatch, "/admin/users/2", gin.H{}},
		{http.MethodDelete, "/admin/users/2", nil},
		{http.MethodPost, "/trigger", nil},
		{http.MethodGet, "/contacts/1/picture", nil},
		{http.MethodPost, "/contacts/1/picture", nil},
		{http.MethodGet, "/graph", nil},
		{http.MethodGet, "/reminders", nil},
		{http.MethodGet, "/reminders/upcoming", nil},
		{http.MethodGet, "/reminders/1", nil},
		{http.MethodPut, "/reminders/1", reminderBody},
		{http.MethodDelete, "/reminders/1", nil},
		{http.MethodPost, "/reminders/1/complete", nil},
		{http.MethodPost, "/contacts/1/reminders", reminderBody},
		{http.MethodGet, "/contacts/1/reminders", nil},
		{http.MethodGet, "/contacts/1/completions", nil},
		{http.MethodDelete, "/completions/1", nil},
		{http.MethodGet, "/calendars", nil},
		{http.MethodPost, "/calendars", calendarBody},
		{http.MethodPut, "/calendars/1", calendarBody},
		{http.MethodDelete, "/calendars/1", nil},
		{http.MethodPost, "/calendars/1/sync", nil},
		{http.MethodGet, "/carddav", nil},
		{http.MethodDelete, "/carddav", nil},
		{http.MethodPost, "/carddav/sync", nil},
		{http.MethodPut, "/carddav", gin.H{"base_url": "https://dav.example.com/", "address_book_path": "/x/"}},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := doJSON(t, router, tc.method, tc.path, tc.body)
			assert.GreaterOrEqual(t, w.Code, 500, w.Body.String())
			assert.False(t, strings.Contains(w.Body.String(), "database is closed"), "driver errors must not reach the client: %s", w.Body.String())
		})
	}
}
