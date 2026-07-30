package services

import (
	"mycorrhizal/config"
	"mycorrhizal/models"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupWebhookRetryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// :memory: sqlite is per-connection; deliverWebhook and ProcessWebhookRetries
	// dispatch through goroutines, so a second pooled connection would see an
	// empty (table-less) database. Pin to a single connection, matching the
	// convention used by the other services *_test.go DB setups that touch
	// goroutines (e.g. setupCalendarSyncTestDB, setupReminderTestDB).
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Webhook{}, &models.WebhookDelivery{}, &models.JobExecution{}))
	return db
}

// TestProcessWebhookRetriesSkipsWhenLocked is the regression test for Tier 3c
// item 4 (docs/fork-plan/95-backlog-and-priorities.md): ProcessWebhookRetries
// previously had no job lock at all, unlike reminders/calendar sync, so
// multiple instances could double-process the same retry window.
func TestProcessWebhookRetriesSkipsWhenLocked(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	cfg := config.Config{}

	require.NoError(t, db.Create(&models.JobExecution{
		JobName:   models.JobNameWebhookRetries,
		LastRunAt: time.Now(),
	}).Error)

	require.NotPanics(t, func() {
		ProcessWebhookRetries(db, cfg)
	})

	var job models.JobExecution
	require.NoError(t, db.Where("job_name = ?", models.JobNameWebhookRetries).First(&job).Error)
	assert.Nil(t, job.LockedAt, "job must not have been re-locked while rate-limited")
}

// TestProcessWebhookRetriesAcquiresAndReleasesLock covers the opposite path:
// no lock row exists yet, so the job runs and the lock is released afterward
// (LastRunAt bumped, LockedAt cleared) rather than left held.
func TestProcessWebhookRetriesAcquiresAndReleasesLock(t *testing.T) {
	db := setupWebhookRetryTestDB(t)
	cfg := config.Config{}

	require.NotPanics(t, func() {
		ProcessWebhookRetries(db, cfg)
	})

	var job models.JobExecution
	require.NoError(t, db.Where("job_name = ?", models.JobNameWebhookRetries).First(&job).Error)
	assert.WithinDuration(t, time.Now(), job.LastRunAt, 5*time.Second)
	assert.Nil(t, job.LockedAt, "lock should be released after the job completes")
}
