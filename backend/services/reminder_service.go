package services

import (
	"fmt"
	"meerkat/config"
	"meerkat/logger"
	"meerkat/models"
	"sort"
	"time"

	"github.com/resend/resend-go/v2"
	"gorm.io/gorm"
)

var sendReminderEmailFn = sendReminderEmail

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

	remindersByUser := make(map[uint][]models.Reminder)
	for _, reminder := range reminders {
		remindersByUser[reminder.UserID] = append(remindersByUser[reminder.UserID], reminder)
	}

	userIDs := make([]uint, 0, len(remindersByUser))
	for userID := range remindersByUser {
		userIDs = append(userIDs, userID)
	}

	var users []models.User
	if err := db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users for reminders: %w", err)
	}

	userByID := make(map[uint]models.User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}

	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })

	for _, userID := range userIDs {
		user, exists := userByID[userID]
		if !exists {
			logger.Warn().Uint("user_id", userID).Msg("Skipping reminders for missing user")
			continue
		}

		userReminders := remindersByUser[userID]

		if config.UseResend {
			if err := sendReminderEmailFn(user, userReminders, config, db); err != nil {
				logger.Error().Err(err).Uint("user_id", user.ID).Msg("Error sending daily reminder email")
				return err
			}
		}

		for _, reminder := range userReminders {
			if reminder.Recurrence == "once" {
				// Use Unscoped to hard delete the reminder permanently
				if err := db.Unscoped().Delete(&reminder).Error; err != nil {
					logger.Error().Err(err).Uint("reminder_id", reminder.ID).Msg("Failed to delete 'once' reminder after sending")
				} else {
					logger.Info().Uint("reminder_id", reminder.ID).Msg("Deleted 'once' reminder after sending")
				}
				continue
			}

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
func sendReminderEmail(user models.User, reminders []models.Reminder, config config.Config, db *gorm.DB) error {
	if user.Email == "" {
		logger.Warn().Uint("user_id", user.ID).Msg("Skipping reminder email because user email is missing")
		return nil
	}

	// Build the HTML content
	htmlContent := "<h1>Your Reminders for Today:</h1><ul>"
	for _, reminder := range reminders {
		contactName := "Unknown" // Default value for contact's name
		if reminder.ContactID != nil {
			var contact models.Contact
			if err := db.Where("user_id = ?", reminder.UserID).First(&contact, *reminder.ContactID).Error; err == nil {
				contactName = contact.Firstname + " " + contact.Lastname
			}
		}
		htmlContent += fmt.Sprintf("<li>%s - %s (Contact: %s)</li>", reminder.RemindAt.Format("02.01.2006"), reminder.Message, contactName)
	}
	htmlContent += "</ul>"

	logger.Debug().Str("html_content", htmlContent).Int("reminder_count", len(reminders)).Uint("user_id", user.ID).Msg("Sending reminder email")

	// Initialize Resend client
	client := resend.NewClient(config.ResendAPIKey)

	// Prepare email parameters
	params := &resend.SendEmailRequest{
		From:    config.ResendFromEmail,
		To:      []string{user.Email},
		Subject: "Your Daily Reminders",
		Html:    htmlContent,
	}

	// Send email
	sent, err := client.Emails.Send(params)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to send reminder email")
		return err
	}

	logger.Info().Str("email_id", sent.Id).Uint("user_id", user.ID).Msg("Reminder email sent successfully")

	return nil
}

// CalculateNextReminderTime determines the next reminder date based on recurrence settings
func CalculateNextReminderTime(reminder models.Reminder) time.Time {
	// Determine the base time to use for calculation
	now := time.Now()
	var baseTime time.Time
	// Default to true if not specified (nil)
	reoccurFromCompletion := reminder.ReoccurFromCompletion == nil || *reminder.ReoccurFromCompletion
	if reoccurFromCompletion {
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
