package controllers

import (
	"errors"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateReminder(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	contactID := c.Param("id")

	// Find the contact by the ID
	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", contactID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Get the validated reminder from context (already bound by ValidateJSONMiddleware)
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("reminder", "validation data not found"))
		return
	}

	// Type assert to Reminder
	reminder, ok := validated.(*models.Reminder)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("reminder", "invalid data type"))
		return
	}

	// Assign the ContactID to the reminder to link it to the contact
	reminder.ContactID = &contact.ID
	reminder.UserID = userID

	// Set hours, minutes, seconds to 0 to ensure reminders are found when comparing for "until date"
	reminder.RemindAt = time.Date(reminder.RemindAt.Year(),
		reminder.RemindAt.Month(),
		reminder.RemindAt.Day(), 0, 0, 0, 0, reminder.RemindAt.Location())

	// Save the new reminder to the database
	if err := db.Create(&reminder).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving reminder to database")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save reminder").WithError(err))
		return
	}

	// Clear the Contact association to avoid including it in the response
	reminder.Contact = models.Contact{}

	c.JSON(http.StatusOK, gin.H{"message": "Reminder created successfully", "reminder": reminder})
}

func GetReminder(c *gin.Context) {
	id := c.Param("id")
	var reminder models.Reminder
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := db.Where("user_id = ?", userID).First(&reminder, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Reminder").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve reminder").WithError(err))
		}
		return
	}

	c.JSON(http.StatusOK, reminder)
}

func UpdateReminder(c *gin.Context) {
	id := c.Param("id")
	var reminder models.Reminder
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := db.Where("user_id = ?", userID).First(&reminder, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Reminder").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve reminder").WithError(err))
		}
		return
	}

	// Get the validated reminder from context (already bound by ValidateJSONMiddleware)
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("reminder", "validation data not found"))
		return
	}

	// Type assert to Reminder
	updatedReminder, ok := validated.(*models.Reminder)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("reminder", "invalid data type"))
		return
	}

	// Updateable fields
	reminder.Message = updatedReminder.Message
	reminder.ByMail = updatedReminder.ByMail
	reminder.RemindAt = time.Date(updatedReminder.RemindAt.Year(),
		updatedReminder.RemindAt.Month(),
		updatedReminder.RemindAt.Day(), 0, 0, 0, 0,
		updatedReminder.RemindAt.Location())
	reminder.Recurrence = updatedReminder.Recurrence
	reminder.ReoccurFromCompletion = updatedReminder.ReoccurFromCompletion
	reminder.ContactID = updatedReminder.ContactID

	if reminder.ContactID != nil {
		var contact models.Contact
		if err := db.Where("user_id = ?", userID).First(&contact, *reminder.ContactID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact"))
			} else {
				apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
			}
			return
		}
	}

	db.Updates(&reminder)

	// Clear the Contact association to avoid including it in the response
	reminder.Contact = models.Contact{}

	c.JSON(http.StatusOK, gin.H{"message": "Reminder updated successfully", "reminder": reminder})
}

func DeleteReminder(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Check if reminder exists first
	var reminder models.Reminder
	if err := db.Where("user_id = ?", userID).First(&reminder, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Reminder").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve reminder").WithError(err))
		}
		return
	}

	if err := db.Delete(&reminder).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete reminder").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reminder deleted"})
}

func GetRemindersForContact(c *gin.Context) {
	contactID := c.Param("id")

	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var contact models.Contact

	if err := db.Preload("Reminders", "reminders.user_id = ?", userID).Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", contactID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reminders": contact.Reminders,
	})
}

// GetAllReminders returns all reminders across all contacts for the current user
func GetAllReminders(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var reminders []models.Reminder

	// Get all reminders, ordered by remind_at date
	// Don't preload Contact to avoid validation issues with invalid contact data
	if err := db.Where("user_id = ?", userID).Order("remind_at ASC").Find(&reminders).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve reminders").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reminders": reminders,
	})
}

// GetUpcomingReminders returns all reminders due within the next 7 days when that set exceeds five, otherwise it ensures at least five upcoming reminders overall
func GetUpcomingReminders(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	sevenDaysFromNow := time.Now().AddDate(0, 0, 7)

	// Get reminders due (or overdue) within the next 7 days
	var remindersNext7Days []models.Reminder
	if err := db.Where("user_id = ? AND remind_at <= ? AND completed = ?", userID, sevenDaysFromNow, false).
		Order("remind_at ASC").
		Find(&remindersNext7Days).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve reminders").WithError(err))
		return
	}

	// If more than five reminders are due soon, return all of them
	if len(remindersNext7Days) > 5 {
		c.JSON(http.StatusOK, gin.H{
			"reminders": remindersNext7Days,
		})
		return
	}

	// Otherwise, ensure we return at least the next five reminders overall
	var remindersNext5 []models.Reminder
	if err := db.Where("user_id = ? AND completed = ?", userID, false).
		Order("remind_at ASC").
		Limit(5).
		Find(&remindersNext5).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve reminders").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reminders": remindersNext5,
	})
}

// CompleteReminder marks a reminder as completed
func CompleteReminder(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var reminder models.Reminder
	if err := db.Where("user_id = ?", userID).First(&reminder, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Reminder").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve reminder").WithError(err))
		}
		return
	}

	// Mark as completed
	reminder.Completed = true
	reminder.LastSent = new(time.Time)
	*reminder.LastSent = time.Now()

	// If reoccur from completion, calculate next reminder time
	// Default to true if not specified (nil)
	reoccurFromCompletion := reminder.ReoccurFromCompletion == nil || *reminder.ReoccurFromCompletion
	if reoccurFromCompletion && reminder.Recurrence != "once" {
		reminder.RemindAt = services.CalculateNextReminderTime(reminder)
		// Reset completed flag for recurring reminders
		reminder.Completed = false

		logger.FromContext(c).Info().
			Time("next_remind_at", reminder.RemindAt).
			Uint("reminder_id", reminder.ID).
			Msg("Reminder completed, next occurrence scheduled")
	}

	// Delete "once" reminders after completion
	if reminder.Recurrence == "once" {
		if err := db.Delete(&reminder).Error; err != nil {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete completed 'once' reminder").WithError(err))
			return
		}

		logger.FromContext(c).Info().Uint("reminder_id", reminder.ID).Msg("Deleted 'once' reminder after completion")
		c.JSON(http.StatusOK, gin.H{"message": "Reminder completed and deleted"})
		return
	}

	// Save the updated reminder
	if err := db.Save(&reminder).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to update reminder").WithError(err))
		return
	}

	// Clear the Contact association to avoid including it in the response
	reminder.Contact = models.Contact{}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Reminder completed successfully",
		"reminder": reminder,
	})
}
