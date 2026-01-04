package services

import (
	"fmt"
	"meerkat/config"
	"meerkat/logger"
	"meerkat/models"
	"os"
	"sort"
	"time"

	"github.com/resend/resend-go/v2"
	"gorm.io/gorm"
)

var sendReminderEmailFn = sendReminderEmail

// Default minimum interval between reminder job runs (prevents duplicates during restarts)
const DefaultReminderMinInterval = 1 * time.Hour

// ReminderMinInterval can be overridden for testing
var ReminderMinInterval = DefaultReminderMinInterval

// getInstanceID returns a unique identifier for this server instance
func getInstanceID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

// acquireJobLock attempts to acquire a lock for the given job.
// Returns true if the lock was acquired, false if the job was run recently
// or is currently locked by another instance.
func acquireJobLock(db *gorm.DB, jobName string, minInterval time.Duration) (bool, error) {
	now := time.Now()
	instanceID := getInstanceID()
	lockTimeout := 5 * time.Minute // Consider locks stale after 5 minutes

	return db.Transaction(func(tx *gorm.DB) error {
		var job models.JobExecution

		// Try to find existing job execution record
		err := tx.Where("job_name = ?", jobName).First(&job).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if err == gorm.ErrRecordNotFound {
			// First time running this job - create the record and acquire lock
			job = models.JobExecution{
				JobName:   jobName,
				LastRunAt: now,
				LockedAt:  &now,
				LockedBy:  instanceID,
			}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
			logger.Info().Str("job", jobName).Str("instance", instanceID).Msg("Acquired job lock (first run)")
			return nil
		}

		// Job exists - check if we should run
		timeSinceLastRun := now.Sub(job.LastRunAt)
		if timeSinceLastRun < minInterval {
			logger.Info().
				Str("job", jobName).
				Dur("since_last_run", timeSinceLastRun).
				Dur("min_interval", minInterval).
				Msg("Skipping job - ran too recently")
			return fmt.Errorf("job ran too recently")
		}

		// Check if another instance has the lock
		if job.LockedAt != nil {
			lockAge := now.Sub(*job.LockedAt)
			if lockAge < lockTimeout && job.LockedBy != instanceID {
				logger.Info().
					Str("job", jobName).
					Str("locked_by", job.LockedBy).
					Dur("lock_age", lockAge).
					Msg("Skipping job - locked by another instance")
				return fmt.Errorf("job locked by another instance")
			}
			// Lock is stale, we can take over
			if lockAge >= lockTimeout {
				logger.Warn().
					Str("job", jobName).
					Str("previous_instance", job.LockedBy).
					Dur("lock_age", lockAge).
					Msg("Taking over stale lock")
			}
		}

		// Acquire the lock
		job.LockedAt = &now
		job.LockedBy = instanceID
		if err := tx.Save(&job).Error; err != nil {
			return err
		}

		logger.Info().Str("job", jobName).Str("instance", instanceID).Msg("Acquired job lock")
		return nil
	}) == nil, nil
}

// releaseJobLock releases the lock and updates the last run time
func releaseJobLock(db *gorm.DB, jobName string, success bool) error {
	now := time.Now()
	instanceID := getInstanceID()

	return db.Transaction(func(tx *gorm.DB) error {
		var job models.JobExecution
		if err := tx.Where("job_name = ?", jobName).First(&job).Error; err != nil {
			return err
		}

		// Only update if we still hold the lock
		if job.LockedBy != instanceID {
			logger.Warn().
				Str("job", jobName).
				Str("expected", instanceID).
				Str("actual", job.LockedBy).
				Msg("Lock was taken by another instance")
			return nil
		}

		if success {
			job.LastRunAt = now
		}
		job.LockedAt = nil
		job.LockedBy = ""

		return tx.Save(&job).Error
	})
}

// SendRemindersWithRateLimit wraps SendReminders with distributed locking
// to prevent duplicate sends during rapid restarts
func SendRemindersWithRateLimit(db *gorm.DB, cfg config.Config) error {
	acquired, err := acquireJobLock(db, models.JobNameDailyReminders, ReminderMinInterval)
	if err != nil {
		logger.Error().Err(err).Msg("Error checking job lock")
		return err
	}

	if !acquired {
		logger.Info().Msg("Skipping reminder job - rate limited")
		return nil
	}

	// Run the actual reminder logic
	err = SendReminders(db, cfg)

	// Release the lock, marking success if no error
	if releaseErr := releaseJobLock(db, models.JobNameDailyReminders, err == nil); releaseErr != nil {
		logger.Error().Err(releaseErr).Msg("Error releasing job lock")
	}

	return err
}

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

	// Group reminders by user
	remindersByUser := make(map[uint][]models.Reminder)
	for _, reminder := range reminders {
		remindersByUser[reminder.UserID] = append(remindersByUser[reminder.UserID], reminder)
	}

	// Collect user IDs from reminders
	userIDSet := make(map[uint]bool)
	for userID := range remindersByUser {
		userIDSet[userID] = true
	}

	// Also include users who have birthdays today (even without reminders)
	// Check all users and use GetUpcomingBirthdays - if first result is today, include them
	var allUsers []models.User
	if err := db.Find(&allUsers).Error; err != nil {
		logger.Warn().Err(err).Msg("Failed to fetch all users for birthday check, continuing with reminders only")
	} else {
		now := time.Now()
		for _, user := range allUsers {
			if userIDSet[user.ID] {
				continue // Already included via reminders
			}
			birthdays, err := GetUpcomingBirthdays(db, user.ID)
			if err != nil {
				logger.Warn().Err(err).Uint("user_id", user.ID).Msg("Failed to fetch birthdays for user")
				continue
			}
			if len(birthdays) > 0 && DaysUntilBirthday(birthdays[0].Birthday, now) == 0 {
				userIDSet[user.ID] = true
			}
		}
	}

	// Convert set to slice
	userIDs := make([]uint, 0, len(userIDSet))
	for userID := range userIDSet {
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) == 0 {
		logger.Info().Msg("No reminders or birthdays to send for today")
		return nil
	}

	// Fetch all users we need to email
	var users []models.User
	if err := db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	userByID := make(map[uint]models.User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}

	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })

	// Short-circuit if email sending is disabled - preserve reminders for when it's enabled
	if !config.UseResend {
		logger.Info().Int("reminder_count", len(reminders)).Msg("Email sending disabled (UseResend=false), skipping reminder mutations to preserve them")
		return nil
	}

	var sendErrors int
	for _, userID := range userIDs {
		user, exists := userByID[userID]
		if !exists {
			logger.Warn().Uint("user_id", userID).Msg("Skipping email for missing user")
			continue
		}

		userReminders := remindersByUser[userID] // May be nil/empty for birthday-only users

		// Attempt to send email - if it fails, skip mutations for this user and continue to next
		if err := sendReminderEmailFn(user, userReminders, config, db); err != nil {
			logger.Error().Err(err).Uint("user_id", user.ID).Msg("Error sending daily email, skipping mutations for this user")
			sendErrors++
			continue // Don't mutate reminders if email failed - allows retry on next run
		}

		// Only mutate reminders after successful email send
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

	if sendErrors > 0 {
		logger.Warn().Int("failed_users", sendErrors).Int("total_users", len(userIDs)).Msg("Some emails failed to send")
	}

	return nil
}

// Send email using Resend with daily reminders and upcoming birthdays
func sendReminderEmail(user models.User, reminders []models.Reminder, config config.Config, db *gorm.DB) error {
	if user.Email == "" {
		logger.Warn().Uint("user_id", user.ID).Msg("Skipping reminder email because user email is missing")
		return nil
	}

	// Build the HTML content for reminders
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

	// Add upcoming birthdays section
	birthdays, err := GetUpcomingBirthdays(db, user.ID)
	if err != nil {
		logger.Warn().Err(err).Uint("user_id", user.ID).Msg("Failed to fetch birthdays for email, continuing without them")
	} else if len(birthdays) > 0 {
		now := time.Now()
		htmlContent += "<h1>Upcoming Birthdays:</h1><ul>"
		for _, birthday := range birthdays {
			days := DaysUntilBirthday(birthday.Birthday, now)
			var daysText string
			if days == 0 {
				daysText = "Today!"
			} else if days == 1 {
				daysText = "Tomorrow"
			} else {
				daysText = fmt.Sprintf("In %d days", days)
			}

			if birthday.Type == "relationship" {
				htmlContent += fmt.Sprintf("<li>%s (%s) - %s's %s - %s</li>",
					birthday.Birthday, daysText, birthday.AssociatedContactName, birthday.RelationshipType, birthday.Name)
			} else {
				htmlContent += fmt.Sprintf("<li>%s (%s) - %s</li>",
					birthday.Birthday, daysText, birthday.Name)
			}
		}
		htmlContent += "</ul>"
	}

	logger.Debug().Str("html_content", htmlContent).Int("reminder_count", len(reminders)).Int("birthday_count", len(birthdays)).Uint("user_id", user.ID).Msg("Sending reminder email")

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

// addMonths adds the specified number of months to a date, clamping to the last
// valid day of the target month to handle edge cases like Jan 31 + 1 month -> Feb 28/29
func addMonths(t time.Time, months int) time.Time {
	// Get the original day of month
	originalDay := t.Day()

	// Add months using Go's AddDate (which may overflow into next month)
	result := t.AddDate(0, months, 0)

	// If the day changed unexpectedly (overflow occurred), clamp to last day of target month
	// For example: Jan 31 + 1 month = March 3 (in non-leap year), we want Feb 28
	if result.Day() != originalDay {
		// Go back to the last day of the previous month (the intended target month)
		result = result.AddDate(0, 0, -result.Day())
	}

	return result
}

// addYears adds the specified number of years to a date, handling Feb 29 edge case
func addYears(t time.Time, years int) time.Time {
	originalDay := t.Day()
	result := t.AddDate(years, 0, 0)

	// Handle Feb 29 -> Feb 28 transition for leap year edge case
	if result.Day() != originalDay {
		result = result.AddDate(0, 0, -result.Day())
	}

	return result
}

// CalculateNextReminderTime determines the next reminder date based on recurrence settings.
// All calculations are done in UTC to ensure consistency.
func CalculateNextReminderTime(reminder models.Reminder) time.Time {
	// Normalize to UTC for consistent calculations
	now := time.Now().UTC()
	remindAtUTC := reminder.RemindAt.UTC()

	var baseTime time.Time
	// Default to true if not specified (nil)
	reoccurFromCompletion := reminder.ReoccurFromCompletion == nil || *reminder.ReoccurFromCompletion
	if reoccurFromCompletion {
		if remindAtUTC.After(now) {
			// For reminders in the future, use the original remind at time (e.g. if I already complete a monthly reminder set for next week I am reminded again next week in one month)
			baseTime = remindAtUTC
		} else {
			// For reminders in the past use now as reference (if I complete a weekly reminder that was due last week, the next reminder is in one week from today)
			baseTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		}
	} else {
		baseTime = remindAtUTC
	}

	switch reminder.Recurrence {
	case "once":
		// Will be deleted anyway
		return reminder.RemindAt
	case "weekly":
		return baseTime.AddDate(0, 0, 7)
	case "monthly":
		return addMonths(baseTime, 1)
	case "quarterly":
		return addMonths(baseTime, 3)
	case "six-months":
		return addMonths(baseTime, 6)
	case "yearly":
		return addYears(baseTime, 1)
	default:
		// If the recurrence type is unrecognized, return the original RemindAt
		logger.Warn().Str("recurrence", reminder.Recurrence).Uint("reminder_id", reminder.ID).Msg("Unrecognized recurrence type")
		return reminder.RemindAt
	}
}
