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

	db.AutoMigrate(&models.Contact{}, &models.Activity{}, &models.Note{}, models.Relationship{}, models.Reminder{}, models.User{})

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
	reminder := models.Reminder{
		UserID:                user.ID,
		ContactID:             &contact.ID,
		Message:               "Test reminder",
		ByMail:                true,
		RemindAt:              time.Now().Add(-1 * time.Hour), // already due today
		Recurrence:            "once",
		ReoccurFromCompletion: false,
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
