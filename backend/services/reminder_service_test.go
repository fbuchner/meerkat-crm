package services

import (
	"mycorrhizal/config"
	"mycorrhizal/models"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRouter() (*gorm.DB, *gin.Engine) {
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	db.AutoMigrate(&models.Contact{}, &models.Activity{}, &models.Note{}, models.Reminder{}, models.User{}, models.JobExecution{}, models.Webhook{}, models.WebhookDelivery{})

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	return db, router
}

// TestAddMonths tests the month addition helper for edge cases
func TestAddMonths(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		months   int
		expected time.Time
	}{
		{
			name:     "Jan 31 + 1 month = Feb 28 (non-leap year)",
			start:    time.Date(2023, 1, 31, 12, 0, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, 2, 28, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Jan 31 + 1 month = Feb 29 (leap year)",
			start:    time.Date(2024, 1, 31, 12, 0, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Jan 30 + 1 month = Feb 28 (non-leap year)",
			start:    time.Date(2023, 1, 30, 12, 0, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, 2, 28, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Jan 29 + 1 month = Feb 28 (non-leap year)",
			start:    time.Date(2023, 1, 29, 12, 0, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, 2, 28, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Jan 28 + 1 month = Feb 28 (no clamping needed)",
			start:    time.Date(2023, 1, 28, 12, 0, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, 2, 28, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Jan 15 + 1 month = Feb 15 (normal case)",
			start:    time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, 2, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Mar 31 + 1 month = Apr 30",
			start:    time.Date(2023, 3, 31, 12, 0, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, 4, 30, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Aug 31 + 3 months (quarterly) = Nov 30",
			start:    time.Date(2023, 8, 31, 12, 0, 0, 0, time.UTC),
			months:   3,
			expected: time.Date(2023, 11, 30, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Aug 31 + 6 months = Feb 28",
			start:    time.Date(2023, 8, 31, 12, 0, 0, 0, time.UTC),
			months:   6,
			expected: time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC), // 2024 is a leap year
		},
		{
			name:     "Dec 31 + 1 month = Jan 31 (year rollover)",
			start:    time.Date(2023, 12, 31, 12, 0, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2024, 1, 31, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addMonths(tt.start, tt.months)
			assert.Equal(t, tt.expected, result, "Expected %v but got %v", tt.expected, result)
		})
	}
}

// TestAddYears tests the year addition helper for edge cases
func TestAddYears(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		years    int
		expected time.Time
	}{
		{
			name:     "Feb 29 2024 + 1 year = Feb 28 2025 (leap to non-leap)",
			start:    time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
			years:    1,
			expected: time.Date(2025, 2, 28, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Feb 28 2023 + 1 year = Feb 28 2024 (normal case)",
			start:    time.Date(2023, 2, 28, 12, 0, 0, 0, time.UTC),
			years:    1,
			expected: time.Date(2024, 2, 28, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Feb 29 2024 + 4 years = Feb 29 2028 (leap to leap)",
			start:    time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
			years:    4,
			expected: time.Date(2028, 2, 29, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "Jan 15 2023 + 1 year = Jan 15 2024 (normal case)",
			start:    time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC),
			years:    1,
			expected: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addYears(tt.start, tt.years)
			assert.Equal(t, tt.expected, result, "Expected %v but got %v", tt.expected, result)
		})
	}
}

// TestCalculateNextReminderTimeMonthlyEdgeCases tests monthly recurrence edge cases
func TestCalculateNextReminderTimeMonthlyEdgeCases(t *testing.T) {
	reoccurFalse := false

	tests := []struct {
		name     string
		reminder models.Reminder
		expected time.Time
	}{
		{
			name: "Monthly from Jan 31 should go to Feb 28",
			reminder: models.Reminder{
				RemindAt:              time.Date(2023, 1, 31, 12, 0, 0, 0, time.UTC),
				Recurrence:            "monthly",
				ReoccurFromCompletion: &reoccurFalse,
			},
			expected: time.Date(2023, 2, 28, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "Monthly from Mar 31 should go to Apr 30",
			reminder: models.Reminder{
				RemindAt:              time.Date(2023, 3, 31, 12, 0, 0, 0, time.UTC),
				Recurrence:            "monthly",
				ReoccurFromCompletion: &reoccurFalse,
			},
			expected: time.Date(2023, 4, 30, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "Quarterly from Aug 31 should go to Nov 30",
			reminder: models.Reminder{
				RemindAt:              time.Date(2023, 8, 31, 12, 0, 0, 0, time.UTC),
				Recurrence:            "quarterly",
				ReoccurFromCompletion: &reoccurFalse,
			},
			expected: time.Date(2023, 11, 30, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "Yearly from Feb 29 leap year should go to Feb 28 non-leap",
			reminder: models.Reminder{
				RemindAt:              time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
				Recurrence:            "yearly",
				ReoccurFromCompletion: &reoccurFalse,
			},
			expected: time.Date(2025, 2, 28, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateNextReminderTime(tt.reminder)
			assert.Equal(t, tt.expected, result, "Expected %v but got %v", tt.expected, result)
		})
	}
}

// TestCalculateNextReminderTimeUTCConsistency tests that times are handled in UTC
func TestCalculateNextReminderTimeUTCConsistency(t *testing.T) {
	reoccurFalse := false

	// Create a reminder with a non-UTC timezone
	pst, _ := time.LoadLocation("America/Los_Angeles")
	remindAt := time.Date(2023, 1, 15, 9, 0, 0, 0, pst)

	reminder := models.Reminder{
		RemindAt:              remindAt,
		Recurrence:            "weekly",
		ReoccurFromCompletion: &reoccurFalse,
	}

	result := CalculateNextReminderTime(reminder)

	// Result should be in UTC
	assert.Equal(t, time.UTC, result.Location(), "Result should be in UTC")

	// Should be exactly 7 days later
	expectedUTC := remindAt.UTC().AddDate(0, 0, 7)
	assert.Equal(t, expectedUTC, result, "Should be 7 days after the original UTC time")
}

func TestSendReminders(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{Username: "reminder-user", Password: "password123", Email: "owner@example.com"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Jane",
		Lastname:  "Doe",
	}
	db.Create(&contact)

	// Create test reminder which should be sent
	reoccurFalse := false
	byMailTrue := true
	reminder := models.Reminder{
		UserID:                user.ID,
		ContactID:             &contact.ID,
		Message:               "Test reminder",
		ByMail:                &byMailTrue,
		RemindAt:              time.Now().Add(-1 * time.Hour), // already due today
		Recurrence:            "once",
		ReoccurFromCompletion: &reoccurFalse,
	}

	db.Create(&reminder)

	var (
		calledUser      models.User
		calledReminders []models.Reminder
		callCount       int
	)

	originalSender := sendReminderEmailFn
	sendReminderEmailFn = func(u models.User, reminders []models.Reminder, cfg config.Config, db *gorm.DB) error {
		calledUser = u
		calledReminders = reminders
		callCount++
		return nil
	}
	defer func() {
		sendReminderEmailFn = originalSender
	}()

	config := config.Config{
		UseResend:       true,
		ResendAPIKey:    "test_api_key",
		ResendFromEmail: "noreply@example.com",
		ReminderTime:    "12:00",
	}

	err := SendReminders(db, config)
	assert.NoError(t, err)

	assert.Equal(t, 1, callCount)
	assert.Equal(t, user.ID, calledUser.ID)
	if assert.Len(t, calledReminders, 1) {
		assert.Equal(t, reminder.ID, calledReminders[0].ID)
	}

	// After sending email, reminder should still exist but marked as email_sent=true
	var updatedReminder models.Reminder
	result := db.First(&updatedReminder, reminder.ID)
	assert.NoError(t, result.Error)
	assert.True(t, updatedReminder.EmailSent, "EmailSent should be true after email is sent")
	assert.NotNil(t, updatedReminder.LastSent, "LastSent should be set after email is sent")
}

// TestSendReminders_ExcludesCompletedAndAlreadySentReminders pins down Tier
// 3c item 11b (docs/fork-plan/95-backlog-and-priorities.md): SendReminders'
// eligibility filter (`completed = false AND email_sent = false`) had never
// been exercised with an already-completed or already-sent reminder sitting
// alongside a genuinely eligible one — only ever tested against reminders
// already in the eligible state. A broken filter here means silent
// re-spamming (already-sent reminders re-emailed) or silent starvation
// (eligible reminders wrongly excluded), neither of which shows up in normal
// monitoring.
func TestSendReminders_ExcludesCompletedAndAlreadySentReminders(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{Username: "reminder-filter-user", Password: "password123", Email: "owner2@example.com"}
	require.NoError(t, db.Create(&user).Error)

	contact := models.Contact{UserID: user.ID, Firstname: "Jane", Lastname: "Doe"}
	require.NoError(t, db.Create(&contact).Error)

	reoccurFalse := false
	byMailTrue := true
	makeReminder := func(message string, completed, emailSent bool) models.Reminder {
		r := models.Reminder{
			UserID:                user.ID,
			ContactID:             &contact.ID,
			Message:               message,
			ByMail:                &byMailTrue,
			RemindAt:              time.Now().Add(-1 * time.Hour),
			Recurrence:            "once",
			ReoccurFromCompletion: &reoccurFalse,
			Completed:             completed,
			EmailSent:             emailSent,
		}
		require.NoError(t, db.Create(&r).Error)
		return r
	}

	eligible := makeReminder("eligible", false, false)
	alreadyCompleted := makeReminder("already completed", true, false)
	alreadySent := makeReminder("already sent", false, true)

	var calledReminders []models.Reminder
	originalSender := sendReminderEmailFn
	sendReminderEmailFn = func(u models.User, reminders []models.Reminder, cfg config.Config, db *gorm.DB) error {
		calledReminders = reminders
		return nil
	}
	defer func() { sendReminderEmailFn = originalSender }()

	cfg := config.Config{UseResend: true, ResendAPIKey: "test_api_key", ResendFromEmail: "noreply@example.com", ReminderTime: "12:00"}
	require.NoError(t, SendReminders(db, cfg))

	if assert.Len(t, calledReminders, 1, "only the genuinely eligible reminder must be picked up") {
		assert.Equal(t, eligible.ID, calledReminders[0].ID)
	}

	var stillCompleted models.Reminder
	require.NoError(t, db.First(&stillCompleted, alreadyCompleted.ID).Error)
	assert.False(t, stillCompleted.EmailSent, "an already-completed reminder must not be touched by the send pass")

	var stillSent models.Reminder
	require.NoError(t, db.First(&stillSent, alreadySent.ID).Error)
	assert.False(t, stillSent.Completed, "an already-sent reminder must not be marked completed by the send pass")
}

func TestSendRemindersWithRateLimit_FirstRun(t *testing.T) {
	db, _ := setupRouter()

	// Set a short interval for testing
	originalInterval := ReminderMinInterval
	ReminderMinInterval = 100 * time.Millisecond
	defer func() { ReminderMinInterval = originalInterval }()

	user := models.User{Username: "rate-limit-user", Password: "password123", Email: "ratelimit@example.com"}
	db.Create(&user)

	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Rate",
		Lastname:  "Limit",
	}
	db.Create(&contact)

	byMailTrue := true
	reminder := models.Reminder{
		UserID:     user.ID,
		ContactID:  &contact.ID,
		Message:    "Rate limit test",
		ByMail:     &byMailTrue,
		RemindAt:   time.Now().Add(-1 * time.Hour),
		Recurrence: "weekly",
	}
	db.Create(&reminder)

	callCount := 0
	originalSender := sendReminderEmailFn
	sendReminderEmailFn = func(u models.User, reminders []models.Reminder, cfg config.Config, db *gorm.DB) error {
		callCount++
		return nil
	}
	defer func() { sendReminderEmailFn = originalSender }()

	cfg := config.Config{
		UseResend:       true,
		ResendAPIKey:    "test_api_key",
		ResendFromEmail: "noreply@example.com",
		ReminderTime:    "12:00",
	}

	// First run should execute
	err := SendRemindersWithRateLimit(db, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "First run should send reminders")

	// Verify job execution was recorded
	var job models.JobExecution
	err = db.Where("job_name = ?", models.JobNameDailyReminders).First(&job).Error
	assert.NoError(t, err)
	assert.NotZero(t, job.LastRunAt)
}

func TestSendRemindersWithRateLimit_RateLimited(t *testing.T) {
	db, _ := setupRouter()

	// Set a long interval to ensure rate limiting
	originalInterval := ReminderMinInterval
	ReminderMinInterval = 1 * time.Hour
	defer func() { ReminderMinInterval = originalInterval }()

	user := models.User{Username: "rate-limit-user2", Password: "password123", Email: "ratelimit2@example.com"}
	db.Create(&user)

	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Rate2",
		Lastname:  "Limit2",
	}
	db.Create(&contact)

	byMailTrue := true
	reminder := models.Reminder{
		UserID:     user.ID,
		ContactID:  &contact.ID,
		Message:    "Rate limit test 2",
		ByMail:     &byMailTrue,
		RemindAt:   time.Now().Add(-1 * time.Hour),
		Recurrence: "weekly",
	}
	db.Create(&reminder)

	callCount := 0
	originalSender := sendReminderEmailFn
	sendReminderEmailFn = func(u models.User, reminders []models.Reminder, cfg config.Config, db *gorm.DB) error {
		callCount++
		return nil
	}
	defer func() { sendReminderEmailFn = originalSender }()

	cfg := config.Config{
		UseResend:       true,
		ResendAPIKey:    "test_api_key",
		ResendFromEmail: "noreply@example.com",
		ReminderTime:    "12:00",
	}

	// First run should execute
	err := SendRemindersWithRateLimit(db, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "First run should send reminders")

	// Second run immediately after should be rate limited
	err = SendRemindersWithRateLimit(db, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "Second run should be rate limited - no additional sends")
}

func TestSendRemindersWithRateLimit_AllowsAfterInterval(t *testing.T) {
	db, _ := setupRouter()

	// Set a very short interval
	originalInterval := ReminderMinInterval
	ReminderMinInterval = 50 * time.Millisecond
	defer func() { ReminderMinInterval = originalInterval }()

	user := models.User{Username: "rate-limit-user3", Password: "password123", Email: "ratelimit3@example.com"}
	db.Create(&user)

	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Rate3",
		Lastname:  "Limit3",
	}
	db.Create(&contact)

	byMailTrue := true
	reminder := models.Reminder{
		UserID:     user.ID,
		ContactID:  &contact.ID,
		Message:    "Rate limit test 3",
		ByMail:     &byMailTrue,
		RemindAt:   time.Now().Add(-1 * time.Hour),
		Recurrence: "weekly",
	}
	db.Create(&reminder)

	callCount := 0
	originalSender := sendReminderEmailFn
	sendReminderEmailFn = func(u models.User, reminders []models.Reminder, cfg config.Config, db *gorm.DB) error {
		callCount++
		return nil
	}
	defer func() { sendReminderEmailFn = originalSender }()

	cfg := config.Config{
		UseResend:       true,
		ResendAPIKey:    "test_api_key",
		ResendFromEmail: "noreply@example.com",
		ReminderTime:    "12:00",
	}

	// First run
	err := SendRemindersWithRateLimit(db, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Record initial last run time
	var job models.JobExecution
	db.Where("job_name = ?", models.JobNameDailyReminders).First(&job)
	firstRunTime := job.LastRunAt

	// Wait for interval to pass
	time.Sleep(100 * time.Millisecond)

	// Create another reminder that's due now (since the first one was updated)
	reminder2 := models.Reminder{
		UserID:     user.ID,
		ContactID:  &contact.ID,
		Message:    "Rate limit test 3 - second",
		ByMail:     &byMailTrue,
		RemindAt:   time.Now().Add(-1 * time.Hour),
		Recurrence: "weekly",
	}
	db.Create(&reminder2)

	// Should now be allowed to run again
	err = SendRemindersWithRateLimit(db, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "Should allow run after interval passes")

	// Verify last run time was updated
	db.Where("job_name = ?", models.JobNameDailyReminders).First(&job)
	assert.True(t, job.LastRunAt.After(firstRunTime), "LastRunAt should be updated after second run")
}

// TestAcquireJobLock_LockedByAnotherInstance covers the branch where a
// different instance currently holds a fresh (non-stale) lock: acquisition
// must fail and leave the lock untouched.
func TestAcquireJobLock_LockedByAnotherInstance(t *testing.T) {
	db, _ := setupRouter()

	now := time.Now()
	lockedAt := now.Add(-1 * time.Minute) // fresh, well within the 5-minute stale timeout
	job := models.JobExecution{
		JobName:   models.JobNameDailyReminders,
		LastRunAt: now.Add(-2 * time.Hour), // old enough to clear the min-interval check
		LockedAt:  &lockedAt,
		LockedBy:  "some-other-instance-12345",
	}
	require.NoError(t, db.Create(&job).Error)

	acquired, err := acquireJobLock(db, models.JobNameDailyReminders, 1*time.Hour)
	assert.NoError(t, err)
	assert.False(t, acquired, "should not acquire a lock currently held by another instance")

	var reloaded models.JobExecution
	require.NoError(t, db.Where("job_name = ?", models.JobNameDailyReminders).First(&reloaded).Error)
	assert.Equal(t, "some-other-instance-12345", reloaded.LockedBy, "lock owner should be unchanged")
}

// TestAcquireJobLock_TakesOverStaleLock covers the branch where another
// instance's lock is older than the 5-minute staleness timeout: acquisition
// must succeed and take over the lock.
func TestAcquireJobLock_TakesOverStaleLock(t *testing.T) {
	db, _ := setupRouter()

	now := time.Now()
	lockedAt := now.Add(-10 * time.Minute) // past the 5-minute stale timeout
	job := models.JobExecution{
		JobName:   models.JobNameDailyReminders,
		LastRunAt: now.Add(-2 * time.Hour),
		LockedAt:  &lockedAt,
		LockedBy:  "crashed-instance-99999",
	}
	require.NoError(t, db.Create(&job).Error)

	acquired, err := acquireJobLock(db, models.JobNameDailyReminders, 1*time.Hour)
	assert.NoError(t, err)
	assert.True(t, acquired, "should take over a stale lock from a crashed instance")

	var reloaded models.JobExecution
	require.NoError(t, db.Where("job_name = ?", models.JobNameDailyReminders).First(&reloaded).Error)
	assert.NotEqual(t, "crashed-instance-99999", reloaded.LockedBy, "lock owner should have been taken over")
}

// TestReleaseJobLock_LockTakenByAnotherInstance covers the guard in
// releaseJobLock where the lock is no longer held by the calling instance
// (e.g. it was taken over as stale by someone else in the meantime): release
// must no-op without error and without clobbering the new owner's lock.
func TestReleaseJobLock_LockTakenByAnotherInstance(t *testing.T) {
	db, _ := setupRouter()

	now := time.Now()
	job := models.JobExecution{
		JobName:   models.JobNameDailyReminders,
		LastRunAt: now.Add(-2 * time.Hour),
		LockedAt:  &now,
		LockedBy:  "a-different-instance",
	}
	require.NoError(t, db.Create(&job).Error)

	err := releaseJobLock(db, models.JobNameDailyReminders, true)
	assert.NoError(t, err, "releasing a lock held by another instance should be a no-op, not an error")

	var reloaded models.JobExecution
	require.NoError(t, db.Where("job_name = ?", models.JobNameDailyReminders).First(&reloaded).Error)
	assert.Equal(t, "a-different-instance", reloaded.LockedBy, "lock owner should be unchanged by the no-op release")
	assert.NotNil(t, reloaded.LockedAt, "lock should still be held since our instance never owned it")
}

// TestFormatDateForUser covers all three date-format branches.
func TestFormatDateForUser(t *testing.T) {
	ts := time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, "03/07/2026", formatDateForUser(ts, "us"))
	assert.Equal(t, "2026-03-07", formatDateForUser(ts, "iso"))
	assert.Equal(t, "07.03.2026", formatDateForUser(ts, "eu"))
	assert.Equal(t, "07.03.2026", formatDateForUser(ts, ""), "unrecognized/empty format should fall back to EU default")
}

// TestFormatBirthdayForUser covers the empty-string guard, both the
// year-unknown (--MM-DD) and full (YYYY-MM-DD) branches across all three
// date formats, and the too-short fallbacks that return the raw string.
func TestFormatBirthdayForUser(t *testing.T) {
	assert.Equal(t, "", formatBirthdayForUser("", "us"), "empty birthday returns empty string")

	// Year-unknown format, all three date formats.
	assert.Equal(t, "03/07", formatBirthdayForUser("--03-07", "us"))
	assert.Equal(t, "03-07", formatBirthdayForUser("--03-07", "iso"))
	assert.Equal(t, "07.03.", formatBirthdayForUser("--03-07", "eu"))

	// Year-unknown but too short to extract month/day - returned as-is.
	assert.Equal(t, "--03", formatBirthdayForUser("--03", "us"))

	// Full YYYY-MM-DD format, all three date formats.
	assert.Equal(t, "03/07/2020", formatBirthdayForUser("2020-03-07", "us"))
	assert.Equal(t, "2020-03-07", formatBirthdayForUser("2020-03-07", "iso"))
	assert.Equal(t, "07.03.2020", formatBirthdayForUser("2020-03-07", "eu"))

	// Neither prefix pattern nor long enough - returned as-is.
	assert.Equal(t, "2020-3-7", formatBirthdayForUser("2020-3-7", "us"))
}

// TestSendReminderEmail_MissingEmailSkips covers the early-return guard when
// the user has no email address configured.
func TestSendReminderEmail_MissingEmailSkips(t *testing.T) {
	db, _ := setupRouter()
	user := models.User{Username: "no-email-user", Password: "password123"}
	require.NoError(t, db.Create(&user).Error)

	cfg := config.Config{}

	err := sendReminderEmail(user, nil, cfg, db)
	assert.NoError(t, err, "missing email should be skipped silently, not an error")
}

// TestSendReminderEmail_NoChannelConfiguredRendersWithoutSending exercises
// the full render path of sendReminderEmail (reminder items, birthday items
// including today/tomorrow/future badge branches, contact name lookup) while
// SendEmail's no-channel-configured guard prevents any real network call.
func TestSendReminderEmail_NoChannelConfiguredRendersWithoutSending(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{
		Username:   "email-render-user",
		Password:   "password123",
		Email:      "render@example.com",
		Language:   "en",
		DateFormat: "iso",
	}
	require.NoError(t, db.Create(&user).Error)

	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Jane",
		Lastname:  "Doe",
	}
	require.NoError(t, db.Create(&contact).Error)

	now := time.Now()
	reminder := models.Reminder{
		UserID:    user.ID,
		ContactID: &contact.ID,
		Message:   "Follow up",
		RemindAt:  now,
	}

	cfg := config.Config{} // UseResend=false, UseSMTP=false -> EmailEnabled() == false

	err := sendReminderEmail(user, []models.Reminder{reminder}, cfg, db)
	assert.NoError(t, err, "no channel configured should render but not error")
}

// TestCalculateNextReminderTime_OnceReturnsOriginal covers the "once"
// recurrence branch, which is a no-op (the reminder is deleted elsewhere).
func TestCalculateNextReminderTime_OnceReturnsOriginal(t *testing.T) {
	remindAt := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	reminder := models.Reminder{
		RemindAt:   remindAt,
		Recurrence: "once",
	}

	result := CalculateNextReminderTime(reminder)
	assert.Equal(t, remindAt, result)
}

// TestCalculateNextReminderTime_SixMonths covers the "six-months" recurrence
// branch.
func TestCalculateNextReminderTime_SixMonths(t *testing.T) {
	reoccurFalse := false
	reminder := models.Reminder{
		RemindAt:              time.Date(2023, 8, 31, 12, 0, 0, 0, time.UTC),
		Recurrence:            "six-months",
		ReoccurFromCompletion: &reoccurFalse,
	}

	result := CalculateNextReminderTime(reminder)
	assert.Equal(t, time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC), result) // 2024 is a leap year
}

// TestCalculateNextReminderTime_UnrecognizedRecurrence covers the default
// branch for an unknown/invalid recurrence value.
func TestCalculateNextReminderTime_UnrecognizedRecurrence(t *testing.T) {
	reoccurFalse := false
	remindAt := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	reminder := models.Reminder{
		RemindAt:              remindAt,
		Recurrence:            "bogus",
		ReoccurFromCompletion: &reoccurFalse,
	}

	result := CalculateNextReminderTime(reminder)
	assert.Equal(t, remindAt, result, "unrecognized recurrence should return the original RemindAt unchanged")
}

// TestCalculateNextReminderTime_ReoccurFromCompletionDefaultTrue_FutureRemindAt
// covers the reoccurFromCompletion=true path (including the nil-defaults-to-
// true case) when RemindAt is still in the future: baseTime should be the
// original RemindAt, not "now".
func TestCalculateNextReminderTime_ReoccurFromCompletionDefaultTrue_FutureRemindAt(t *testing.T) {
	remindAt := time.Now().UTC().Add(48 * time.Hour)
	reminder := models.Reminder{
		RemindAt:              remindAt,
		Recurrence:            "weekly",
		ReoccurFromCompletion: nil, // nil defaults to true
	}

	result := CalculateNextReminderTime(reminder)
	expected := remindAt.AddDate(0, 0, 7)
	assert.Equal(t, expected, result)
}

// TestCalculateNextReminderTime_ReoccurFromCompletionTrue_PastRemindAt covers
// the reoccurFromCompletion=true path when RemindAt is in the past: baseTime
// should be today (date only), not the stale original RemindAt.
func TestCalculateNextReminderTime_ReoccurFromCompletionTrue_PastRemindAt(t *testing.T) {
	reoccurTrue := true
	remindAt := time.Now().UTC().Add(-48 * time.Hour)
	reminder := models.Reminder{
		RemindAt:              remindAt,
		Recurrence:            "weekly",
		ReoccurFromCompletion: &reoccurTrue,
	}

	result := CalculateNextReminderTime(reminder)

	nowUTC := time.Now().UTC()
	today := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	expected := today.AddDate(0, 0, 7)
	assert.Equal(t, expected, result)
}

// TestSendReminders_BirthdayOnlyUserIncluded covers the branch in
// SendReminders that scans all users for a birthday falling today even when
// they have no due reminders, and includes them in the email/webhook run.
func TestSendReminders_BirthdayOnlyUserIncluded(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{Username: "birthday-only-user", Password: "password123", Email: "bdayonly@example.com"}
	require.NoError(t, db.Create(&user).Error)

	today := time.Now().UTC()
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "Birthday",
		Lastname:  "Person",
		Birthday:  today.AddDate(-30, 0, 0).Format("2006-01-02"), // same month/day, 30 years ago
	}
	require.NoError(t, db.Create(&contact).Error)

	var (
		calledUser      models.User
		calledReminders []models.Reminder
		callCount       int
	)
	originalSender := sendReminderEmailFn
	sendReminderEmailFn = func(u models.User, reminders []models.Reminder, cfg config.Config, db *gorm.DB) error {
		calledUser = u
		calledReminders = reminders
		callCount++
		return nil
	}
	defer func() { sendReminderEmailFn = originalSender }()

	cfg := config.Config{
		UseResend:       true,
		ResendAPIKey:    "test_api_key",
		ResendFromEmail: "noreply@example.com",
		ReminderTime:    "12:00",
	}

	err := SendReminders(db, cfg)
	assert.NoError(t, err)

	assert.Equal(t, 1, callCount, "birthday-only user (no due reminders) should still trigger an email")
	assert.Equal(t, user.ID, calledUser.ID)
	assert.Empty(t, calledReminders, "user has no reminders, only a birthday")
}

// TestSendReminders_EmailDisabledPreservesReminders covers the
// !config.EmailEnabled() branch: due reminders must be left with
// EmailSent=false (not mutated) so they're picked up again once email is
// configured.
func TestSendReminders_EmailDisabledPreservesReminders(t *testing.T) {
	db, _ := setupRouter()

	user := models.User{Username: "email-disabled-user", Password: "password123", Email: "disabled@example.com"}
	require.NoError(t, db.Create(&user).Error)

	contact := models.Contact{UserID: user.ID, Firstname: "No", Lastname: "Email"}
	require.NoError(t, db.Create(&contact).Error)

	byMailTrue := true
	reminder := models.Reminder{
		UserID:     user.ID,
		ContactID:  &contact.ID,
		Message:    "Should not be sent",
		ByMail:     &byMailTrue,
		RemindAt:   time.Now().Add(-1 * time.Hour),
		Recurrence: "once",
	}
	require.NoError(t, db.Create(&reminder).Error)

	cfg := config.Config{} // no UseResend/UseSMTP -> EmailEnabled() == false

	err := SendReminders(db, cfg)
	assert.NoError(t, err)

	var reloaded models.Reminder
	require.NoError(t, db.First(&reloaded, reminder.ID).Error)
	assert.False(t, reloaded.EmailSent, "reminder should be preserved (not marked sent) while email is disabled")
	assert.Nil(t, reloaded.LastSent)
}
