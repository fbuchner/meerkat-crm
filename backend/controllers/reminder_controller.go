package controllers

import (
	"errors"
	"net/http"
	apperrors "perema/errors"
	"perema/logger"
	"perema/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateReminder(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	contactID := c.Param("id")

	// Find the contact by the ID
	var contact models.Contact
	if err := db.First(&contact, contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", contactID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Bind the incoming JSON request to the Reminder struct
	var reminder models.Reminder
	if err := c.ShouldBindJSON(&reminder); err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error binding JSON for create reminder")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("reminder", err.Error()))
		return
	}

	// Assign the ContactID to the reminder to link it to the contact
	reminder.ContactID = &contact.ID

	// Set hours, minutes, seoncds to 0 to ensure reminders are found when comparing for "until date"
	reminder.RemindAt = time.Date(reminder.RemindAt.Year(),
		reminder.RemindAt.Month(),
		reminder.RemindAt.Day(), 0, 0, 0, 0, reminder.RemindAt.Location())

	// Save the new reminder to the database
	if err := db.Create(&reminder).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving reminder to database")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save reminder").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reminder created successfully", "reminder": reminder})
}

func GetReminder(c *gin.Context) {
	id := c.Param("id")
	var reminder models.Reminder
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&reminder, id).Error; err != nil {
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
	if err := db.First(&reminder, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Reminder").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve reminder").WithError(err))
		}
		return
	}

	var updatedReminder models.Reminder
	if err := c.ShouldBindJSON(&updatedReminder); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("reminder", err.Error()))
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
	reminder.ReocurrFromCompletion = updatedReminder.ReocurrFromCompletion
	reminder.ContactID = updatedReminder.ContactID

	db.Updates(&reminder)

	c.JSON(http.StatusOK, gin.H{"message": "Reminder updated successfully", "reminder": reminder})
}

func DeleteReminder(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	// Check if reminder exists first
	var reminder models.Reminder
	if err := db.First(&reminder, id).Error; err != nil {
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

	var contact models.Contact

	if err := db.Preload("Reminders").First(&contact, contactID).Error; err != nil {
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
