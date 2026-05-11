package services

import (
	"context"
	"encoding/base64"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCalDAVSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Contact{}, &models.Activity{}))

	return db
}

func testCalDAVSyncService(client *http.Client) *CalDAVSyncService {
	return &CalDAVSyncService{
		client: client,
		validateURL: func(rawURL string) (*url.URL, error) {
			return url.Parse(rawURL)
		},
	}
}

func TestCalDAVSyncCreatesActivitiesFromCalendarData(t *testing.T) {
	db := setupCalDAVSyncTestDB(t)
	user := models.User{Username: "tester", Password: "password123", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	contact := models.Contact{UserID: user.ID, Firstname: "Ada", Lastname: "Lovelace"}
	require.NoError(t, db.Create(&contact).Error)

	var sawReport bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawReport = true
		assert.Equal(t, "REPORT", r.Method)
		assert.Equal(t, "1", r.Header.Get("Depth"))
		assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("calendar:secret")), r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:propstat>
      <d:prop>
        <c:calendar-data><![CDATA[BEGIN:VCALENDAR
BEGIN:VEVENT
UID:planning-1
SUMMARY:Planning lunch
DESCRIPTION:Talk through the next project
LOCATION:Cafe Central
DTSTART:20260512T130000Z
END:VEVENT
END:VCALENDAR]]></c:calendar-data>
      </d:prop>
    </d:propstat>
  </d:response>
  <d:response>
    <d:propstat>
      <d:prop>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VEVENT
UID:call-2
SUMMARY:Check-in call
DESCRIPTION:Remote catch up
LOCATION:Video
DTSTART;VALUE=DATE:20260513
END:VEVENT
END:VCALENDAR</c:calendar-data>
      </d:prop>
    </d:propstat>
  </d:response>
</d:multistatus>`))
	}))
	defer server.Close()

	service := testCalDAVSyncService(server.Client())
	result, err := service.Sync(context.Background(), db, user.ID, models.CalDAVSyncInput{
		URL:        server.URL + "/calendars/tester/main/",
		Username:   "calendar",
		Password:   "secret",
		ContactIDs: []uint{contact.ID},
	})
	require.NoError(t, err)
	assert.True(t, sawReport)
	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 0, result.Skipped)

	var activities []models.Activity
	require.NoError(t, db.Preload("Contacts").Order("date asc").Find(&activities).Error)
	require.Len(t, activities, 2)
	assert.Equal(t, "Planning lunch", activities[0].Title)
	assert.Equal(t, "Talk through the next project", activities[0].Description)
	assert.Equal(t, "Cafe Central", activities[0].Location)
	assert.Equal(t, time.Date(2026, 5, 12, 13, 0, 0, 0, time.UTC), activities[0].Date)
	require.Len(t, activities[0].Contacts, 1)
	assert.Equal(t, contact.ID, activities[0].Contacts[0].ID)
	assert.Equal(t, "Check-in call", activities[1].Title)
	assert.Equal(t, time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC), activities[1].Date)

	result, err = service.Sync(context.Background(), db, user.ID, models.CalDAVSyncInput{
		URL:        server.URL + "/calendars/tester/main/",
		Username:   "calendar",
		Password:   "secret",
		ContactIDs: []uint{contact.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 2, result.Skipped)
}

func TestCalDAVSyncRejectsContactsFromOtherUsers(t *testing.T) {
	db := setupCalDAVSyncTestDB(t)
	user := models.User{Username: "tester", Password: "password123", Email: "tester@example.com"}
	otherUser := models.User{Username: "other", Password: "password123", Email: "other@example.com"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&otherUser).Error)
	foreignContact := models.Contact{UserID: otherUser.ID, Firstname: "Grace", Lastname: "Hopper"}
	require.NoError(t, db.Create(&foreignContact).Error)

	service := testCalDAVSyncService(http.DefaultClient)
	_, err := service.Sync(context.Background(), db, user.ID, models.CalDAVSyncInput{
		URL:        "https://calendar.example.com/dav/",
		Username:   "calendar",
		Password:   "secret",
		ContactIDs: []uint{foreignContact.ID},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "one or more contacts")

	var count int64
	require.NoError(t, db.Model(&models.Activity{}).Count(&count).Error)
	assert.EqualValues(t, 0, count)
}

func TestParseICalendarEventsHandlesFoldedLinesAndTimezones(t *testing.T) {
	events, err := parseICalendarEvents(`BEGIN:VCALENDAR
BEGIN:VEVENT
UID:folded
SUMMARY:Long summary
DESCRIPTION:First line\nSecond line with a folded
 continuation
LOCATION:Office\, Room 2
DTSTART;TZID=America/New_York:20260514T090000
END:VEVENT
END:VCALENDAR`)

	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Long summary", events[0].Title)
	assert.Equal(t, "First line\nSecond line with a foldedcontinuation", events[0].Description)
	assert.Equal(t, "Office, Room 2", events[0].Location)
	assert.Equal(t, time.Date(2026, 5, 14, 13, 0, 0, 0, time.UTC), events[0].Date)
}
