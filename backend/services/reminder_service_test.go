package services

import (
	"perema/config"
	"perema/models"
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

	// Create a contact and relationships
	contact := models.Contact{
		Firstname: "Jane",
		Lastname:  "Doe",
	}
	db.Create(&contact)

	// Create test reminder which should be sent
	reminder := models.Reminder{
		ContactID:  &contact.ID,
		Message:    "Test reminder",
		ByMail:     true,
		RemindAt:   time.Now().Add(1 * time.Hour), // Tomorrow
		Recurrence: "Once",
	}

	db.Create(&reminder)

	config := config.Config{
		SendgridAPIKey:  "test_api_key",
		SendgridToEmail: "test_email@example.com",
		ReminderTime:    "12:00",
	}

	err := SendReminders(db, config)
	assert.NoError(t, err)

	// Check saved last_sent and other updates
	var updatedReminder models.Reminder
	db.First(&updatedReminder, reminder.ID)
	assert.NotNil(t, updatedReminder.LastSent)
}
