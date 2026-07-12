package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"meerkat/config"
	"meerkat/i18n"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCalendarSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Contact{}, &models.Activity{}, &models.JobExecution{},
		&models.CalendarSubscription{}, &models.CalendarEventLink{},
	))
	return db
}

func calendarTestConfig() config.Config {
	return config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
}

func createCalendarTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	user := models.User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func newTestSubscription(t *testing.T, db *gorm.DB, cfg config.Config, userID uint, url, username, password string) *models.CalendarSubscription {
	t.Helper()
	encrypted, err := EncryptCredential(cfg.JWTSecretKey, password)
	require.NoError(t, err)
	sub := models.CalendarSubscription{
		UserID:            userID,
		Name:              "Test calendar",
		URL:               url,
		Username:          username,
		PasswordEncrypted: encrypted,
		SyncEnabled:       true,
		PastDays:          30,
		FutureDays:        90,
	}
	require.NoError(t, db.Create(&sub).Error)
	return &sub
}

// icalDate renders a UTC datetime a given number of days from now, so test
// events always fall inside (or outside) the sync window.
func icalDate(daysFromNow int) string {
	return time.Now().UTC().AddDate(0, 0, daysFromNow).Format("20060102T150405Z")
}

func multistatusResponse(calendars ...string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	sb.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">` + "\n")
	for i, cal := range calendars {
		escaped := strings.ReplaceAll(cal, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		sb.WriteString(fmt.Sprintf(`<d:response><d:href>/calendars/test/event%d.ics</d:href>
<d:propstat><d:prop><c:calendar-data>%s</c:calendar-data></d:prop>
<d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, i, escaped))
	}
	sb.WriteString(`</d:multistatus>`)
	return sb.String()
}

func TestCalendarSyncCreatesUpdatesAndSkips(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	contact := models.Contact{UserID: user.ID, Firstname: "Ada", Lastname: "Lovelace",
		Emails: []models.ContactEmail{{Type: "work", Value: "Ada@Example.com"}}}
	require.NoError(t, db.Create(&contact).Error)

	summary := "Lunch <with> Ada & the \"team\""
	eventStart := icalDate(2)
	var sawReport, sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "REPORT" {
			sawReport = true
			assert.Equal(t, "1", r.Header.Get("Depth"))
			body, _ := io.ReadAll(r.Body)
			assert.Contains(t, string(body), "time-range", "REPORT should request server-side date filtering")
		}
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("calendar:secret"))
		sawAuth = r.Header.Get("Authorization") == expectedAuth
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		fmt.Fprint(w, multistatusResponse(
			"BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\nBEGIN:VEVENT\nUID:event-1\nSUMMARY:"+
				strings.ReplaceAll(summary, ",", "\\,")+
				"\nDESCRIPTION:Quarterly catch-up\nLOCATION:Cafe Central\nATTENDEE:mailto:ada@example.com\nDTSTART:"+eventStart+"\nEND:VEVENT\nEND:VCALENDAR",
			"BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\nBEGIN:VEVENT\nUID:event-2\nSUMMARY:All-day planning\nDTSTART;VALUE=DATE:"+
				time.Now().UTC().AddDate(0, 0, 5).Format("20060102")+
				"\nEND:VEVENT\nEND:VCALENDAR",
		))
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/calendars/test/", "calendar", "secret")
	service := NewCalendarSyncService(false)

	stats, err := service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.True(t, sawReport, "should use CalDAV REPORT")
	assert.True(t, sawAuth, "should send basic auth credentials")
	assert.Equal(t, CalendarSyncStats{Created: 2}, stats)
	assert.Equal(t, models.CalendarSyncStatusSuccess, sub.LastSyncStatus)
	require.NotNil(t, sub.LastSyncedAt)

	var activities []models.Activity
	require.NoError(t, db.Preload("Contacts").Order("date asc").Find(&activities).Error)
	require.Len(t, activities, 2)
	assert.Equal(t, summary, activities[0].Title, "special characters must survive XML and iCal escaping")
	assert.Equal(t, "Quarterly catch-up", activities[0].Description)
	assert.Equal(t, "Cafe Central", activities[0].Location)
	require.Len(t, activities[0].Contacts, 1, "attendee email should match the contact")
	assert.Equal(t, contact.ID, activities[0].Contacts[0].ID)
	assert.Empty(t, activities[1].Contacts)

	// Second sync with identical data: everything is skipped.
	stats, err = service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Skipped: 2}, stats)

	// Changing an event upstream updates the linked activity instead of duplicating it.
	summary = "Dinner instead"
	stats, err = service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Updated: 1, Skipped: 1}, stats)

	require.NoError(t, db.Order("date asc").Find(&activities).Error)
	require.Len(t, activities, 2)
	assert.Equal(t, "Dinner instead", activities[0].Title)
}

func TestCalendarSyncEmptyCalendarSucceeds(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		fmt.Fprint(w, multistatusResponse())
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/calendars/test/", "", "")
	stats, err := NewCalendarSyncService(false).SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err, "an empty calendar is not an error")
	assert.Equal(t, CalendarSyncStats{}, stats)
	assert.Equal(t, models.CalendarSyncStatusSuccess, sub.LastSyncStatus)
}

func TestCalendarSyncWithoutCredentialsSendsNoAuthHeader(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"), "no credentials configured, no auth header expected")
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\nBEGIN:VEVENT\nUID:public-1\nSUMMARY:Public event\nDTSTART:"+icalDate(1)+"\nEND:VEVENT\nEND:VCALENDAR\n")
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/public.ics", "", "")
	stats, err := NewCalendarSyncService(false).SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Created: 1}, stats)
}

func TestCalendarSyncUnauthorizedReturnsSentinel(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/calendars/test/", "user", "wrong")
	_, err := NewCalendarSyncService(false).SyncSubscription(context.Background(), db, cfg, sub)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCalendarUnauthorized)
	assert.Equal(t, models.CalendarSyncStatusError, sub.LastSyncStatus)
	assert.NotEmpty(t, sub.LastSyncError)
}

func TestCalendarSyncFallsBackToPlainICS(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	var sawGet bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "REPORT" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		sawGet = true
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "text/calendar")
		// One event inside the window, one far in the past, one too far ahead.
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\n"+
			"BEGIN:VEVENT\nUID:in-window\nSUMMARY:Recent catch-up\nDTSTART:"+icalDate(-3)+"\nEND:VEVENT\n"+
			"BEGIN:VEVENT\nUID:ancient\nSUMMARY:Ancient event\nDTSTART:20150101T100000Z\nEND:VEVENT\n"+
			"BEGIN:VEVENT\nUID:far-future\nSUMMARY:Too far ahead\nDTSTART:"+icalDate(400)+"\nEND:VEVENT\n"+
			"END:VCALENDAR\n")
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/export", "", "")
	stats, err := NewCalendarSyncService(false).SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.True(t, sawGet, "should fall back to GET when REPORT is not allowed")
	assert.Equal(t, CalendarSyncStats{Created: 1}, stats, "past/future events outside the window are filtered out")

	var activities []models.Activity
	require.NoError(t, db.Find(&activities).Error)
	require.Len(t, activities, 1)
	assert.Equal(t, "Recent catch-up", activities[0].Title)
}

func TestCalendarSyncRespectsUserDeletedActivities(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	summary := "Meeting v1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\nBEGIN:VEVENT\nUID:meeting-1\nSUMMARY:"+summary+"\nDTSTART:"+icalDate(1)+"\nEND:VEVENT\nEND:VCALENDAR\n")
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/cal.ics", "", "")
	service := NewCalendarSyncService(false)

	stats, err := service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Created: 1}, stats)

	// User deletes the imported activity.
	require.NoError(t, db.Where("user_id = ?", user.ID).Delete(&models.Activity{}).Error)

	// Upstream event changes; the deleted activity must not come back.
	summary = "Meeting v2"
	stats, err = service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Skipped: 1}, stats)

	var count int64
	require.NoError(t, db.Model(&models.Activity{}).Count(&count).Error)
	assert.EqualValues(t, 0, count, "user-deleted activities are not re-imported")
}

func TestCalendarSyncMatchesByContentWhenUIDsChange(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	// Some feeds (e.g. generated waste-collection calendars) mint fresh UIDs
	// on every request; unchanged events must still be recognized.
	requests := 0
	eventStart := icalDate(3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprintf(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\n"+
			"BEGIN:VEVENT\nUID:generated-%d-a\nSUMMARY:Bioabfall\nDTSTART:%s\nEND:VEVENT\n"+
			"BEGIN:VEVENT\nUID:generated-%d-b\nSUMMARY:Restabfall\nDTSTART:%s\nEND:VEVENT\n"+
			"END:VCALENDAR\n", requests, eventStart, requests, icalDate(5))
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/cal.ics", "", "")
	service := NewCalendarSyncService(false)

	stats, err := service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Created: 2}, stats)

	// Second sync: all UIDs changed, but the content did not.
	stats, err = service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Skipped: 2}, stats, "unchanged events with new UIDs must not duplicate")

	var count int64
	require.NoError(t, db.Model(&models.Activity{}).Count(&count).Error)
	assert.EqualValues(t, 2, count)

	// The links now carry the latest UIDs so future syncs keep matching.
	var links []models.CalendarEventLink
	require.NoError(t, db.Find(&links).Error)
	require.Len(t, links, 2)
	for _, link := range links {
		assert.Contains(t, link.UID, fmt.Sprintf("generated-%d-", requests))
	}

	// A changed event under yet another UID cannot be recognized; it becomes a
	// new activity (the duplication case the settings dialog warns about).
	eventStart = icalDate(4)
	stats, err = service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Created: 1, Skipped: 1}, stats)
}

func TestCalendarSyncLocalizesUntitledEvents(t *testing.T) {
	require.NoError(t, i18n.Init())
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)
	require.NoError(t, db.Model(&user).Update("language", "de").Error)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\nBEGIN:VEVENT\nUID:untitled-1\nDTSTART:"+icalDate(1)+"\nEND:VEVENT\nEND:VCALENDAR\n")
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/cal.ics", "", "")
	stats, err := NewCalendarSyncService(false).SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Created: 1}, stats)

	var activities []models.Activity
	require.NoError(t, db.Find(&activities).Error)
	require.Len(t, activities, 1)
	assert.Equal(t, "Unbenannter Termin", activities[0].Title, "untitled events get a fallback title in the user's language")
}

func TestCalendarSyncSkipsCancelledAndImportsOverrides(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\n"+
			"BEGIN:VEVENT\nUID:keeper\nSUMMARY:Keeper\nDTSTART:"+icalDate(1)+"\nEND:VEVENT\n"+
			"BEGIN:VEVENT\nUID:cancelled\nSUMMARY:Cancelled\nSTATUS:CANCELLED\nDTSTART:"+icalDate(2)+"\nEND:VEVENT\n"+
			"BEGIN:VEVENT\nUID:keeper\nRECURRENCE-ID:"+icalDate(3)+"\nSUMMARY:Override instance\nDTSTART:"+icalDate(3)+"\nEND:VEVENT\n"+
			"END:VCALENDAR\n")
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/cal.ics", "", "")
	stats, err := NewCalendarSyncService(false).SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Created: 2}, stats, "the base event and the override instance are imported, the cancelled one is not")

	var activities []models.Activity
	require.NoError(t, db.Order("date asc").Find(&activities).Error)
	require.Len(t, activities, 2)
	assert.Equal(t, "Keeper", activities[0].Title)
	assert.Equal(t, "Override instance", activities[1].Title)
}

func TestCalendarSyncExpandsRecurringEvents(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	// Fix the base time once so DTSTART, EXDATE and RECURRENCE-ID line up exactly.
	base := time.Now().UTC()
	at := func(daysFromNow int) string {
		return base.AddDate(0, 0, daysFromNow).Format("20060102T150405Z")
	}

	// Weekly event that started over a year ago. With the window PastDays=30 /
	// FutureDays=90 its occurrences fall on days -370+7k, i.e. -27, -20, …, +85:
	// 17 occurrences. One is removed via EXDATE (-20) and one is moved via a
	// RECURRENCE-ID override (-13 → -12).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\n"+
			"BEGIN:VEVENT\nUID:weekly\nSUMMARY:Team meeting\nDTSTART:"+at(-370)+
			"\nRRULE:FREQ=WEEKLY\nEXDATE:"+at(-20)+"\nEND:VEVENT\n"+
			"BEGIN:VEVENT\nUID:weekly\nRECURRENCE-ID:"+at(-13)+"\nSUMMARY:Moved team meeting\nDTSTART:"+at(-12)+"\nEND:VEVENT\n"+
			"END:VCALENDAR\n")
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/cal.ics", "", "")
	service := NewCalendarSyncService(false)

	stats, err := service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Created: 16}, stats,
		"17 occurrences in the window, minus one EXDATE and one overridden occurrence, plus the override instance")

	var moved []models.Activity
	require.NoError(t, db.Where("title = ?", "Moved team meeting").Find(&moved).Error)
	require.Len(t, moved, 1)
	assert.Equal(t, base.AddDate(0, 0, -12).Truncate(time.Second).UTC(), moved[0].Date.UTC())

	var excluded int64
	require.NoError(t, db.Model(&models.Activity{}).Where("date = ?", base.AddDate(0, 0, -20).Truncate(time.Second)).Count(&excluded).Error)
	assert.EqualValues(t, 0, excluded, "EXDATE occurrences are not imported")

	// Re-syncing must not duplicate occurrences.
	stats, err = service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Skipped: 16}, stats)
}

func TestCalendarSyncRelinksContactsWhenAttendeesChange(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	contact := models.Contact{UserID: user.ID, Firstname: "Grace", Lastname: "Hopper", Email: "grace@example.com"}
	require.NoError(t, db.Create(&contact).Error)

	attendees := ""
	eventStart := icalDate(2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\nBEGIN:VEVENT\nUID:standup\nSUMMARY:Standup\nDTSTART:"+eventStart+attendees+"\nEND:VEVENT\nEND:VCALENDAR\n")
	}))
	defer server.Close()

	sub := newTestSubscription(t, db, cfg, user.ID, server.URL+"/cal.ics", "", "")
	service := NewCalendarSyncService(false)

	stats, err := service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Created: 1}, stats)

	// The only upstream change is a newly invited attendee.
	attendees = "\nATTENDEE:mailto:grace@example.com"
	stats, err = service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, CalendarSyncStats{Updated: 1}, stats, "an attendee change must be detected as an update")

	var activities []models.Activity
	require.NoError(t, db.Preload("Contacts").Find(&activities).Error)
	require.Len(t, activities, 1)
	require.Len(t, activities[0].Contacts, 1, "the new attendee should be linked as a contact")
	assert.Equal(t, contact.ID, activities[0].Contacts[0].ID)
}

func TestRedactURLPassword(t *testing.T) {
	err := fmt.Errorf("Get \"https://bob:hunter2@example.com/cal.ics\": connection refused")
	redacted := redactURLPassword(err, "https://bob:hunter2@example.com/cal.ics")
	assert.NotContains(t, redacted.Error(), "hunter2")
	assert.Contains(t, redacted.Error(), "xxxxx")

	// Sentinel matching must survive redaction.
	wrapped := redactURLPassword(fmt.Errorf("%w: hunter2", ErrCalendarUnreachable), "https://bob:hunter2@example.com/cal.ics")
	assert.ErrorIs(t, wrapped, ErrCalendarUnreachable)

	// URLs without an embedded password leave errors untouched.
	assert.Same(t, err, redactURLPassword(err, "https://example.com/cal.ics"))
	assert.Nil(t, redactURLPassword(nil, "https://example.com/cal.ics"))
}

func TestNormalizeCalendarURL(t *testing.T) {
	parsed, err := NormalizeCalendarURL("webcal://example.com/cal.ics")
	require.NoError(t, err)
	assert.Equal(t, "https", parsed.Scheme, "webcal URLs are converted to https")

	_, err = NormalizeCalendarURL("ftp://example.com/cal.ics")
	assert.ErrorIs(t, err, ErrCalendarInvalidURL)

	_, err = NormalizeCalendarURL("not a url at all\x00")
	assert.ErrorIs(t, err, ErrCalendarInvalidURL)

	_, err = NormalizeCalendarURL("https:///missing-host.ics")
	assert.ErrorIs(t, err, ErrCalendarInvalidURL)
}

func TestCredentialCryptoRoundTrip(t *testing.T) {
	secret := "test-jwt-secret-key-with-32-chars!!"

	encrypted, err := EncryptCredential(secret, "hunter2-🔒")
	require.NoError(t, err)
	assert.NotContains(t, encrypted, "hunter2", "ciphertext must not contain the plaintext")

	decrypted, err := DecryptCredential(secret, encrypted)
	require.NoError(t, err)
	assert.Equal(t, "hunter2-🔒", decrypted)

	// Empty passwords stay empty.
	encrypted, err = EncryptCredential(secret, "")
	require.NoError(t, err)
	assert.Empty(t, encrypted)
	decrypted, err = DecryptCredential(secret, "")
	require.NoError(t, err)
	assert.Empty(t, decrypted)

	// A different key cannot decrypt.
	_, err = DecryptCredential("another-secret-key-that-is-32-chars", mustEncrypt(t, secret, "value"))
	assert.Error(t, err)

	// Corrupted ciphertext fails cleanly.
	_, err = DecryptCredential(secret, "not-base64!!")
	assert.Error(t, err)
}

func mustEncrypt(t *testing.T, secret, value string) string {
	t.Helper()
	encrypted, err := EncryptCredential(secret, value)
	require.NoError(t, err)
	return encrypted
}

func TestSyncAllCalendarsRecordsErrorsPerSubscription(t *testing.T) {
	db := setupCalendarSyncTestDB(t)
	cfg := calendarTestConfig()
	user := createCalendarTestUser(t, db)

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		fmt.Fprint(w, "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//EN\nBEGIN:VEVENT\nUID:ok-1\nSUMMARY:Works\nDTSTART:"+icalDate(1)+"\nEND:VEVENT\nEND:VCALENDAR\n")
	}))
	defer okServer.Close()

	brokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer brokenServer.Close()

	okSub := newTestSubscription(t, db, cfg, user.ID, okServer.URL+"/cal.ics", "", "")
	brokenSub := newTestSubscription(t, db, cfg, user.ID, brokenServer.URL+"/cal.ics", "", "")
	disabledSub := newTestSubscription(t, db, cfg, user.ID, brokenServer.URL+"/cal.ics", "", "")
	require.NoError(t, db.Model(disabledSub).Update("sync_enabled", false).Error)

	SyncAllCalendars(db, cfg)

	var refreshedOk, refreshedBroken, refreshedDisabled models.CalendarSubscription
	require.NoError(t, db.First(&refreshedOk, okSub.ID).Error)
	require.NoError(t, db.First(&refreshedBroken, brokenSub.ID).Error)
	require.NoError(t, db.First(&refreshedDisabled, disabledSub.ID).Error)

	assert.Equal(t, models.CalendarSyncStatusSuccess, refreshedOk.LastSyncStatus)
	assert.Equal(t, models.CalendarSyncStatusError, refreshedBroken.LastSyncStatus)
	assert.NotEmpty(t, refreshedBroken.LastSyncError)
	assert.Empty(t, refreshedDisabled.LastSyncStatus, "disabled subscriptions are not synced")

	var count int64
	require.NoError(t, db.Model(&models.Activity{}).Count(&count).Error)
	assert.EqualValues(t, 1, count, "one working calendar imported one event despite the broken one")

	// Errors are wiped again after a successful sync.
	require.NoError(t, db.Model(&models.CalendarSubscription{}).Where("id = ?", brokenSub.ID).Update("url", okServer.URL+"/cal.ics").Error)
	SyncAllCalendars(db, cfg)
	require.NoError(t, db.First(&refreshedBroken, brokenSub.ID).Error)
	assert.Equal(t, models.CalendarSyncStatusSuccess, refreshedBroken.LastSyncStatus)
	assert.Empty(t, refreshedBroken.LastSyncError)
}
