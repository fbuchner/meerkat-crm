package services

import (
	"fmt"
	"log"
	"perema/config"
	"perema/models"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gorm.io/gorm"
)

func SendReminders(db *gorm.DB, config config.Config) error {
	log.Println("Sending reminders...")
	var reminders []models.Reminder
	// Get the current time
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	err := db.Where("by_mail = ? AND remind_at <= ? AND (last_sent IS NULL OR last_sent <= ?)", true, endOfDay, endOfDay).Find(&reminders).Error
	if err != nil {
		return fmt.Errorf("failed to fetch reminders: %w", err)
	}

	if len(reminders) == 0 {
		log.Println("No reminders to send for today.")
		return nil
	}

	// Prepare email notification
	err = sendReminderEmail(reminders, config, db)
	if err != nil {
		log.Printf("Error sending daily reminder email: %v", err)
		return err
	}

	// Update last_sent for reminders
	for _, reminder := range reminders {
		reminder.LastSent = new(time.Time)
		*reminder.LastSent = time.Now()
		reminder.RemindAt = calculateNextReminderTime(reminder)
		log.Println("Reminder new remind at: ", reminder.RemindAt)
		if err := db.Save(&reminder).Error; err != nil {
			log.Printf("Failed to update reminder after sending for ID %d: %v", reminder.ID, err)
		}
	}

	return nil
}

// Send email using SendGrid with daily reminders
func sendReminderEmail(reminders []models.Reminder, config config.Config, db *gorm.DB) error {
	from := mail.NewEmail("Meerkat CRM", config.SendgridToEmail)
	subject := "Your Daily Reminders"
	to := mail.NewEmail("CRM User", config.SendgridToEmail)

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

	log.Println(htmlContent)

	message := mail.NewSingleEmail(from, subject, to, "", htmlContent)
	client := sendgrid.NewSendClient(config.SendgridAPIKey)
	res, err := client.Send(message)

	if err != nil {
		log.Printf("Failed to send reminder email: %v", err)
		return err
	}

	log.Printf("Reminder email sent successfully with status code: %d", res.StatusCode)

	return nil
}

func calculateNextReminderTime(reminder models.Reminder) time.Time {
	// Determine the base time to use for calculation
	now := time.Now()
	var baseTime time.Time
	if reminder.ReocurrFromCompletion {
		baseTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else {
		baseTime = reminder.RemindAt
	}

	switch reminder.Recurrence {
	case "No recurrence":
		return reminder.RemindAt
	case "Quarterly":
		return baseTime.AddDate(0, 3, 0)
	case "Six-months":
		return baseTime.AddDate(0, 6, 0)
	case "Yearly":
		return baseTime.AddDate(1, 0, 0)
	default:
		// If the recurrence type is unrecognized, return the original RemindAt or handle the error
		return reminder.RemindAt
	}
}
