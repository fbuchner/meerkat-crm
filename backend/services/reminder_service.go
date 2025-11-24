package services

import (
	"fmt"
	"meerkat/config"
	"meerkat/logger"
	"meerkat/models"
	"time"

	"github.com/resend/resend-go/v2"
	"gorm.io/gorm"
)

func SendReminders(db *gorm.DB, config config.Config) error {
	logger.Info().Msg("Sending reminders...")
	var reminders []models.Reminder
	// Get the current time
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	// Fetch reminders that are:
	// - Set to be sent by email
	// - Due today or before
	// - Not completed
	// - Either never sent, or last sent was before today
	err := db.Where("by_mail = ? AND remind_at <= ? AND completed = ? AND (last_sent IS NULL OR last_sent <= ?)",
		true, endOfDay, false, endOfDay).Find(&reminders).Error
	if err != nil {
		return fmt.Errorf("failed to fetch reminders: %w", err)
	}

	if len(reminders) == 0 {
		logger.Info().Msg("No reminders to send for today")
		return nil
	}

	// Prepare email notification
	err = sendReminderEmail(reminders, config, db)
	if err != nil {
		logger.Error().Err(err).Msg("Error sending daily reminder email")
		return err
	}

	// Update last_sent for reminders and handle "once" reminders
	for _, reminder := range reminders {
		if reminder.Recurrence == "once" {
			// Delete "once" reminders after sending
			if err := db.Delete(&reminder).Error; err != nil {
				logger.Error().Err(err).Uint("reminder_id", reminder.ID).Msg("Failed to delete 'once' reminder after sending")
			} else {
				logger.Info().Uint("reminder_id", reminder.ID).Msg("Deleted 'once' reminder after sending")
			}
		} else {
			// Update recurring reminders
			reminder.LastSent = new(time.Time)
			*reminder.LastSent = time.Now()
			reminder.RemindAt = CalculateNextReminderTime(reminder)
			logger.Debug().Time("next_remind_at", reminder.RemindAt).Uint("reminder_id", reminder.ID).Msg("Updated reminder time")
			if err := db.Save(&reminder).Error; err != nil {
				logger.Error().Err(err).Uint("reminder_id", reminder.ID).Msg("Failed to update reminder after sending")
			}
		}
	}

	return nil
}

// Send email using Resend with daily reminders
func sendReminderEmail(reminders []models.Reminder, config config.Config, db *gorm.DB) error {
	// Build the HTML content
	htmlContent := "<h1>Your Reminders for Today:</h1><ul>"
	for _, reminder := range reminders {
		contactName := "Unknown" // Default value for contact's name
		if reminder.ContactID != nil {
			var contact models.Contact
			if err := db.First(&contact, *reminder.ContactID).Error; err == nil {
				contactName = contact.Firstname + " " + contact.Lastname
			}
		}
		htmlContent += fmt.Sprintf("<li>%s - %s (Contact: %s)</li>", reminder.RemindAt.Format("02.01.2006"), reminder.Message, contactName)
	}
	htmlContent += "</ul>"

	logger.Debug().Str("html_content", htmlContent).Int("reminder_count", len(reminders)).Msg("Sending reminder email")

	// Initialize Resend client
	client := resend.NewClient(config.ResendAPIKey)

	// Prepare email parameters
	params := &resend.SendEmailRequest{
		From:    config.ResendFromEmail,
		To:      []string{config.ResendToEmail},
		Subject: "Your Daily Reminders",
		Html:    htmlContent,
	}

	// Send email
	sent, err := client.Emails.Send(params)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to send reminder email")
		return err
	}

	logger.Info().Str("email_id", sent.Id).Msg("Reminder email sent successfully")

	return nil
}

// CalculateNextReminderTime determines the next reminder date based on recurrence settings
func CalculateNextReminderTime(reminder models.Reminder) time.Time {
	// Determine the base time to use for calculation
	now := time.Now()
	var baseTime time.Time
	if reminder.ReocurrFromCompletion {
		if reminder.RemindAt.After(now) {
			// For reminders in the future, use the original remind at time (e.g. if I already complete a monthly reminder set for next week I am remindet again next week in one month)
			baseTime = reminder.RemindAt
		} else {
			// For reminders in the past use now as reference (if I complete a weekly reminder that was due last week, the next reminder is in one week from today)
			baseTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		}
	} else {
		baseTime = reminder.RemindAt
	}

	switch reminder.Recurrence {
	case "once":
		// Will be deleted anyway
		return reminder.RemindAt
	case "weekly":
		return baseTime.AddDate(0, 0, 7)
	case "monthly":
		return baseTime.AddDate(0, 1, 0)
	case "quarterly":
		return baseTime.AddDate(0, 3, 0)
	case "six-months":
		return baseTime.AddDate(0, 6, 0)
	case "yearly":
		return baseTime.AddDate(1, 0, 0)
	default:
		// If the recurrence type is unrecognized, return the original RemindAt
		logger.Warn().Str("recurrence", reminder.Recurrence).Uint("reminder_id", reminder.ID).Msg("Unrecognized recurrence type")
		return reminder.RemindAt
	}
}
