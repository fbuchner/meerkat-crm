package services

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"meerkat/config"
	"meerkat/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDeletedUserJobsDB(t *testing.T) (*gorm.DB, models.User, models.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Contact{}, &models.Activity{}, &models.Note{}, &models.Relationship{}, &models.Reminder{},
		&models.EmploymentHistory{}, &models.CalendarSubscription{}, &models.CalendarEventLink{},
		&models.CardDAVConnection{}, &models.CardDAVContactLink{}, &models.Webhook{}, &models.WebhookDelivery{},
	))

	live := models.User{Username: "live", Password: "password123!A", Email: "live@example.com"}
	deleted := models.User{Username: "deleted", Password: "password123!A", Email: "deleted@example.com"}
	require.NoError(t, db.Create(&live).Error)
	require.NoError(t, db.Create(&deleted).Error)
	return db, live, deleted
}

func unavailableServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	return server, &hits
}

func TestSyncAllCalendarsSkipsDeletedUsers(t *testing.T) {
	db, live, deleted := setupDeletedUserJobsDB(t)
	cfg := config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
	liveServer, liveHits := unavailableServer(t)
	orphanServer, orphanHits := unavailableServer(t)

	liveSub := models.CalendarSubscription{UserID: live.ID, Name: "live", URL: liveServer.URL, SyncEnabled: true}
	orphan := models.CalendarSubscription{UserID: deleted.ID, Name: "orphan", URL: orphanServer.URL, SyncEnabled: true}
	require.NoError(t, db.Create(&liveSub).Error)
	require.NoError(t, db.Create(&orphan).Error)
	require.NoError(t, db.Delete(&deleted).Error)

	SyncAllCalendars(db, cfg)

	assert.NotZero(t, liveHits.Load())
	assert.Zero(t, orphanHits.Load(), "stored credentials of a deleted user must not be used")
	require.NoError(t, db.First(&orphan, orphan.ID).Error)
	assert.Nil(t, orphan.LastSyncedAt)
}

func TestSyncAllContactConnectionsSkipsDeletedUsers(t *testing.T) {
	db, live, deleted := setupDeletedUserJobsDB(t)
	cfg := config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
	liveServer, liveHits := unavailableServer(t)
	orphanServer, orphanHits := unavailableServer(t)

	liveConn := models.CardDAVConnection{UserID: live.ID, BaseURL: liveServer.URL, AddressBookPath: "/ab/", Direction: models.CardDAVDirectionTwoWay, SyncEnabled: true}
	orphan := models.CardDAVConnection{UserID: deleted.ID, BaseURL: orphanServer.URL, AddressBookPath: "/ab/", Direction: models.CardDAVDirectionTwoWay, SyncEnabled: true}
	require.NoError(t, db.Create(&liveConn).Error)
	require.NoError(t, db.Create(&orphan).Error)
	require.NoError(t, db.Delete(&deleted).Error)

	SyncAllContactConnections(db, cfg)

	assert.NotZero(t, liveHits.Load())
	assert.Zero(t, orphanHits.Load(), "stored credentials of a deleted user must not be used")
	require.NoError(t, db.First(&orphan, orphan.ID).Error)
	assert.Nil(t, orphan.LastSyncedAt)
}

func TestProcessWebhookRetriesSkipsDeletedUsers(t *testing.T) {
	db, live, deleted := setupDeletedUserJobsDB(t)
	liveServer, liveHits := unavailableServer(t)
	orphanServer, orphanHits := unavailableServer(t)

	past := time.Now().Add(-time.Minute)
	liveHook := models.Webhook{UserID: live.ID, Name: "live", URL: liveServer.URL, Secret: "s", IsActive: true}
	orphanHook := models.Webhook{UserID: deleted.ID, Name: "orphan", URL: orphanServer.URL, Secret: "s", IsActive: true}
	require.NoError(t, db.Create(&liveHook).Error)
	require.NoError(t, db.Create(&orphanHook).Error)
	liveDelivery := models.WebhookDelivery{WebhookID: liveHook.ID, EventType: "e", Payload: "{}", Attempts: 1, NextRetryAt: &past}
	orphanDelivery := models.WebhookDelivery{WebhookID: orphanHook.ID, EventType: "e", Payload: "{}", Attempts: 1, NextRetryAt: &past}
	require.NoError(t, db.Create(&liveDelivery).Error)
	require.NoError(t, db.Create(&orphanDelivery).Error)
	require.NoError(t, db.Delete(&deleted).Error)

	ProcessWebhookRetries(db, config.Config{})

	// The live retry saves a new delivery when done
	assert.Eventually(t, func() bool {
		var n int64
		db.Model(&models.WebhookDelivery{}).Where("webhook_id = ?", liveHook.ID).Count(&n)
		return n == 2
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, int32(1), liveHits.Load())
	assert.Zero(t, orphanHits.Load())

	var reloaded models.WebhookDelivery
	require.NoError(t, db.First(&reloaded, orphanDelivery.ID).Error)
	assert.Nil(t, reloaded.NextRetryAt, "the orphaned retry must not be picked up again")
}

func TestSendRemindersSkipsDeletedUsers(t *testing.T) {
	db, _, deleted := setupDeletedUserJobsDB(t)
	byMail := true
	contactID := uint(1)
	reminder := models.Reminder{UserID: deleted.ID, Message: "m", ByMail: &byMail, RemindAt: time.Now().Add(-time.Hour), Recurrence: "once", ContactID: &contactID}
	require.NoError(t, db.Create(&reminder).Error)
	require.NoError(t, db.Delete(&deleted).Error)

	var mailed int
	original := sendReminderEmailFn
	sendReminderEmailFn = func(models.User, []models.Reminder, config.Config, *gorm.DB) error {
		mailed++
		return nil
	}
	defer func() { sendReminderEmailFn = original }()

	require.NoError(t, SendReminders(db, config.Config{UseResend: true, ResendAPIKey: "k", ResendFromEmail: "a@example.com", ReminderTime: "12:00"}))
	assert.Zero(t, mailed)
}
