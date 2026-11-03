package controllers

import (
	"errors"
	"fmt"
	"meerkat/config"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type calendarFixture struct {
	db       *gorm.DB
	user     models.User
	stranger models.User
	router   *gin.Engine
}

func setupCalendars(t *testing.T) calendarFixture {
	t.Helper()
	return setupCalendarsWith(t, config.Config{JWTSecretKey: testJWTSecret})
}

func setupCalendarsWith(t *testing.T, cfg config.Config) calendarFixture {
	t.Helper()
	db := newTestDB(t)
	user := newTestUser(t, db, "owner", false)
	stranger := newTestUser(t, db, "stranger", false)

	validated := middleware.ValidateJSONMiddleware(&models.CalendarSubscriptionInput{})
	router := routerAs(db, user.ID, cfg)
	router.GET("/calendars", ListCalendarSubscriptions)
	router.POST("/calendars", validated, CreateCalendarSubscription)
	router.PUT("/calendars/:id", validated, UpdateCalendarSubscription)
	router.DELETE("/calendars/:id", DeleteCalendarSubscription)
	router.POST("/calendars/:id/sync", SyncCalendarSubscription)
	return calendarFixture{db: db, user: user, stranger: stranger, router: router}
}

func (f calendarFixture) seed(t *testing.T, userID uint, url, password string) models.CalendarSubscription {
	t.Helper()
	encrypted, err := services.EncryptCredential(testJWTSecret, password)
	require.NoError(t, err)
	subscription := models.CalendarSubscription{
		UserID: userID, Name: "Cal", URL: url, Username: "cal-user", PasswordEncrypted: encrypted,
		SyncEnabled: true, PastDays: 30, FutureDays: 30,
	}
	require.NoError(t, f.db.Create(&subscription).Error)
	return subscription
}

func (f calendarFixture) reload(t *testing.T, id uint) models.CalendarSubscription {
	t.Helper()
	var subscription models.CalendarSubscription
	require.NoError(t, f.db.First(&subscription, id).Error)
	return subscription
}

func TestCreateCalendarSubscription_AppliesDefaultsAndEncrypts(t *testing.T) {
	f := setupCalendars(t)

	w := doJSON(t, f.router, http.MethodPost, "/calendars", gin.H{"name": "Work", "url": "webcal://cal.example.com/feed.ics", "username": "me", "password": "hunter2"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), "hunter2")

	response := decodeJSON[models.CalendarSubscriptionResponse](t, w)
	assert.True(t, response.HasPassword)
	assert.True(t, response.SyncEnabled)
	assert.Equal(t, 5, response.PastDays)
	assert.Equal(t, 10, response.FutureDays)

	stored := f.reload(t, response.ID)
	assert.Equal(t, f.user.ID, stored.UserID)
	assert.NotContains(t, stored.PasswordEncrypted, "hunter2")
	decrypted, err := services.DecryptCredential(testJWTSecret, stored.PasswordEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "hunter2", decrypted)
}

func TestCreateCalendarSubscription_HonoursExplicitOptions(t *testing.T) {
	f := setupCalendars(t)

	w := doJSON(t, f.router, http.MethodPost, "/calendars", gin.H{"name": "Work", "url": "https://cal.example.com/", "past_days": 7, "future_days": 400})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	response := decodeJSON[models.CalendarSubscriptionResponse](t, w)
	assert.False(t, response.HasPassword)
	stored := f.reload(t, response.ID)
	assert.Equal(t, 7, stored.PastDays)
	assert.Equal(t, 400, stored.FutureDays)
}

func TestCreateCalendarSubscription_ZeroValuesAreNotReplacedByColumnDefaults(t *testing.T) {
	skipKnownBug(t, "calendar_controller.go:86 db.Create drops false/0 for columns with a gorm default tag, so sync_enabled=false and past_days=0 are stored as true and 5")
	f := setupCalendars(t)

	w := doJSON(t, f.router, http.MethodPost, "/calendars", gin.H{"name": "Work", "url": "https://cal.example.com/", "sync_enabled": false, "past_days": 0, "future_days": 0})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	stored := f.reload(t, decodeJSON[models.CalendarSubscriptionResponse](t, w).ID)
	assert.False(t, stored.SyncEnabled, "the user opted out of background syncing")
	assert.Zero(t, stored.PastDays)
	assert.Zero(t, stored.FutureDays)
}

func TestCreateCalendarSubscription_Rejections(t *testing.T) {
	f := setupCalendars(t)

	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, "/calendars", gin.H{"name": "x", "url": "ftp://cal.example.com/"}).Code, "scheme")
	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, "/calendars", gin.H{"url": "https://cal.example.com/"}).Code, "name")
	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, "/calendars", gin.H{"name": "x", "url": "https://cal.example.com/", "past_days": -1}).Code, "days")

	var count int64
	require.NoError(t, f.db.Model(&models.CalendarSubscription{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCreateCalendarSubscription_LimitIsPerUser(t *testing.T) {
	f := setupCalendars(t)
	for range maxCalendarSubscriptionsPerUser {
		f.seed(t, f.stranger.ID, "https://theirs.example.com/", "")
	}
	for range maxCalendarSubscriptionsPerUser - 1 {
		f.seed(t, f.user.ID, "https://cal.example.com/", "")
	}
	valid := gin.H{"name": "x", "url": "https://cal.example.com/"}

	assert.Equal(t, http.StatusCreated, doJSON(t, f.router, http.MethodPost, "/calendars", valid).Code, "another user's subscriptions do not count")
	assert.Equal(t, http.StatusConflict, doJSON(t, f.router, http.MethodPost, "/calendars", valid).Code)

	// A deleted subscription frees its slot.
	var first models.CalendarSubscription
	require.NoError(t, f.db.Where("user_id = ?", f.user.ID).First(&first).Error)
	require.Equal(t, http.StatusOK, doRequest(f.router, http.MethodDelete, fmt.Sprintf("/calendars/%d", first.ID), nil, "").Code)
	assert.Equal(t, http.StatusCreated, doJSON(t, f.router, http.MethodPost, "/calendars", valid).Code)
}

func TestListCalendarSubscriptions_OnlyTheUsersAndNoSecrets(t *testing.T) {
	f := setupCalendars(t)
	mine := f.seed(t, f.user.ID, "https://cal.example.com/", "hunter2")
	f.seed(t, f.stranger.ID, "https://theirs.example.com/", "their-secret")

	w := doRequest(f.router, http.MethodGet, "/calendars", nil, "")
	require.Equal(t, http.StatusOK, w.Code)

	assert.NotContains(t, w.Body.String(), "theirs.example.com")
	assert.NotContains(t, w.Body.String(), mine.PasswordEncrypted)
	calendars := decodeJSON[struct {
		Calendars []models.CalendarSubscriptionResponse
	}](t, w).Calendars
	require.Len(t, calendars, 1)
	assert.Equal(t, mine.ID, calendars[0].ID)
	assert.True(t, calendars[0].HasPassword)
}

func TestUpdateCalendarSubscription_FieldsAndPasswordHandling(t *testing.T) {
	f := setupCalendars(t)
	subscription := f.seed(t, f.user.ID, "https://cal.example.com/", "original")
	path := fmt.Sprintf("/calendars/%d", subscription.ID)
	password := func() string {
		decrypted, err := services.DecryptCredential(testJWTSecret, f.reload(t, subscription.ID).PasswordEncrypted)
		require.NoError(t, err)
		return decrypted
	}

	w := doJSON(t, f.router, http.MethodPut, path, gin.H{"name": "Renamed", "url": "https://cal.example.com/new", "username": "u2", "sync_enabled": false, "past_days": 1, "future_days": 2})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	stored := f.reload(t, subscription.ID)
	assert.Equal(t, "Renamed", stored.Name)
	assert.Equal(t, "https://cal.example.com/new", stored.URL)
	assert.Equal(t, "u2", stored.Username)
	assert.False(t, stored.SyncEnabled)
	assert.Equal(t, 1, stored.PastDays)
	assert.Equal(t, 2, stored.FutureDays)
	assert.Equal(t, "original", password(), "an empty password keeps the stored one")

	require.Equal(t, http.StatusOK, doJSON(t, f.router, http.MethodPut, path, gin.H{"name": "n", "url": "https://cal.example.com/", "password": "replaced"}).Code)
	assert.Equal(t, "replaced", password())

	w = doJSON(t, f.router, http.MethodPut, path, gin.H{"name": "n", "url": "https://cal.example.com/", "clear_password": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, decodeJSON[models.CalendarSubscriptionResponse](t, w).HasPassword)
	assert.Empty(t, f.reload(t, subscription.ID).PasswordEncrypted)

	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPut, path, gin.H{"name": "n", "url": "ftp://cal.example.com/"}).Code)
	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPut, "/calendars/abc", gin.H{"name": "n", "url": "https://cal.example.com/"}).Code)
}

func TestUpdateCalendarSubscription_ChangingHostDropsTheStoredPassword(t *testing.T) {
	skipKnownBug(t, "calendar_controller.go:118-139 keeps the stored password when url moves to another host, so the next sync sends it there")
	f := setupCalendars(t)
	subscription := f.seed(t, f.user.ID, "https://cal.example.com/", "original")

	w := doJSON(t, f.router, http.MethodPut, fmt.Sprintf("/calendars/%d", subscription.ID), gin.H{"name": "n", "url": "https://evil.example.org/"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.Empty(t, f.reload(t, subscription.ID).PasswordEncrypted, "credentials stored for one host must not follow the subscription to another")
}

func TestDeleteCalendarSubscription_RemovesLinksButKeepsActivities(t *testing.T) {
	f := setupCalendars(t)
	subscription := f.seed(t, f.user.ID, "https://cal.example.com/", "secret")
	activity := models.Activity{UserID: f.user.ID, Title: "Imported", Date: time.Now()}
	require.NoError(t, f.db.Create(&activity).Error)
	require.NoError(t, f.db.Create(&models.CalendarEventLink{SubscriptionID: subscription.ID, UserID: f.user.ID, UID: "u", ActivityID: activity.ID, ContentHash: "h"}).Error)

	w := doRequest(f.router, http.MethodDelete, fmt.Sprintf("/calendars/%d", subscription.ID), nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var gone models.CalendarSubscription
	assert.ErrorIs(t, f.db.First(&gone, subscription.ID).Error, gorm.ErrRecordNotFound)
	var links int64
	require.NoError(t, f.db.Model(&models.CalendarEventLink{}).Count(&links).Error)
	assert.Zero(t, links)
	var kept models.Activity
	assert.NoError(t, f.db.First(&kept, activity.ID).Error)

	assert.Equal(t, http.StatusBadRequest, doRequest(f.router, http.MethodDelete, "/calendars/abc", nil, "").Code)
	assert.Equal(t, http.StatusNotFound, doRequest(f.router, http.MethodDelete, fmt.Sprintf("/calendars/%d", subscription.ID), nil, "").Code)
}

func TestCalendarSubscriptions_AnotherUsersIsNotFound(t *testing.T) {
	f := setupCalendars(t)
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer server.Close()
	theirs := f.seed(t, f.stranger.ID, server.URL, "their-secret")
	path := fmt.Sprintf("/calendars/%d", theirs.ID)

	assert.Equal(t, http.StatusNotFound, doJSON(t, f.router, http.MethodPut, path, gin.H{"name": "Hacked", "url": "https://evil.example.org/"}).Code)
	assert.Equal(t, http.StatusNotFound, doRequest(f.router, http.MethodPost, path+"/sync", nil, "").Code)
	assert.Equal(t, http.StatusNotFound, doRequest(f.router, http.MethodDelete, path, nil, "").Code)

	stored := f.reload(t, theirs.ID)
	assert.Equal(t, "Cal", stored.Name)
	assert.Equal(t, server.URL, stored.URL)
	assert.Zero(t, hits.Load(), "another user's calendar must not be fetched with their credentials")
}

func calendarFeed(t *testing.T, events string) (*httptest.Server, *atomic.Value) {
	t.Helper()
	var auth atomic.Value
	auth.Store("")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, password, ok := r.BasicAuth(); ok {
			auth.Store(user + ":" + password)
		}
		if r.Method == "REPORT" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\n"+events+"END:VCALENDAR\n")
	}))
	t.Cleanup(server.Close)
	return server, &auth
}

func TestSyncCalendarSubscription_ImportsEventsForTheOwner(t *testing.T) {
	f := setupCalendars(t)
	start := time.Now().UTC().AddDate(0, 0, 2).Format("20060102T150405Z")
	server, auth := calendarFeed(t, "BEGIN:VEVENT\nUID:event-1\nSUMMARY:Lunch\nDTSTART:"+start+"\nEND:VEVENT\n")
	subscription := f.seed(t, f.user.ID, server.URL+"/feed.ics", "stored-secret")
	path := fmt.Sprintf("/calendars/%d/sync", subscription.ID)

	w := doRequest(f.router, http.MethodPost, path, nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := decodeJSON[map[string]any](t, w)
	assert.EqualValues(t, 1, body["created"])
	assert.Equal(t, "cal-user:stored-secret", auth.Load())

	var activity models.Activity
	require.NoError(t, f.db.First(&activity).Error)
	assert.Equal(t, "Lunch", activity.Title)
	assert.Equal(t, f.user.ID, activity.UserID)

	stored := f.reload(t, subscription.ID)
	assert.Equal(t, models.CalendarSyncStatusSuccess, stored.LastSyncStatus)
	assert.NotNil(t, stored.LastSyncedAt)

	// A second run is idempotent.
	w = doRequest(f.router, http.MethodPost, path, nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	body = decodeJSON[map[string]any](t, w)
	assert.EqualValues(t, 0, body["created"])
	assert.EqualValues(t, 1, body["skipped"])
}

func TestSyncCalendarSubscription_FailureIsReportedAndRecorded(t *testing.T) {
	f := setupCalendars(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	subscription := f.seed(t, f.user.ID, server.URL, "wrong")

	w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/calendars/%d/sync", subscription.ID), nil, "")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "authentication failed")
	stored := f.reload(t, subscription.ID)
	assert.Equal(t, models.CalendarSyncStatusError, stored.LastSyncStatus)
	assert.NotEmpty(t, stored.LastSyncError)
}

func TestSyncCalendarSubscription_BlocksPrivateAddressesWhenConfigured(t *testing.T) {
	f := setupCalendarsWith(t, config.Config{JWTSecretKey: testJWTSecret, CalDAVBlockPrivateURLs: true})
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer server.Close()
	subscription := f.seed(t, f.user.ID, server.URL, "secret")

	w := doRequest(f.router, http.MethodPost, fmt.Sprintf("/calendars/%d/sync", subscription.ID), nil, "")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "private address")
	assert.Zero(t, hits.Load())
}

func TestCalendarSyncError_MapsSentinels(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{services.ErrCalendarInvalidURL, http.StatusBadRequest},
		{services.ErrCalendarUnauthorized, http.StatusServiceUnavailable},
		{services.ErrCalendarNotFound, http.StatusServiceUnavailable},
		{services.ErrCalendarPrivateAddress, http.StatusServiceUnavailable},
		{services.ErrCalendarTooLarge, http.StatusServiceUnavailable},
		{services.ErrCalendarInvalidData, http.StatusServiceUnavailable},
		{services.ErrCalendarUnreachable, http.StatusServiceUnavailable},
	} {
		assert.Equal(t, tc.status, calendarSyncError(fmt.Errorf("wrapped: %w", tc.err)).HTTPStatus, tc.err.Error())
	}
	assert.GreaterOrEqual(t, calendarSyncError(errors.New("unexpected")).HTTPStatus, 400)
}

func TestCalendarSyncError_DoesNotExposeInternalDetail(t *testing.T) {
	skipKnownBug(t, "calendar_controller.go:222 puts err.Error() of unexpected errors into the API response, which contactSyncError deliberately does not")
	appErr := calendarSyncError(errors.New("db at 10.0.0.5 exploded"))
	assert.NotContains(t, fmt.Sprintf("%s %v", appErr.Message, appErr.Details), "10.0.0.5")
}
