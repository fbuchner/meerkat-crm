package services

import (
	"meerkat/config"
	"meerkat/models"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRouter() (*gorm.DB, *gin.Engine) {
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&models.Contact{}, &models.Activity{}, &models.Note{}, models.Relationship{}, models.Reminder{}, models.User{}, models.JobExecution{})

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
