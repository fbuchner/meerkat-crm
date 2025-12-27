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

	var updatedReminder models.Reminder
	result := db.Unscoped().First(&updatedReminder, reminder.ID)
	assert.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
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
