package controllers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"meerkat/carddav"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/models"
	"net/http"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ExportData exports all user data as CSV files in a combined format
func ExportData(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Fetch user to get custom field names
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch user for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch user"))
		return
	}
	customFieldNames := user.CustomFieldNames
	if customFieldNames == nil {
		customFieldNames = []string{}
	}

	// Fetch all user data
	var contacts []models.Contact
	if err := db.Where("user_id = ?", userID).
		Preload("Relationships").
		Order("firstname ASC, lastname ASC").
		Find(&contacts).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch contacts for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch contacts"))
		return
	}

	var activities []models.Activity
	if err := db.Where("user_id = ?", userID).
		Preload("Contacts").
		Order("date DESC").
		Find(&activities).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch activities for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch activities"))
		return
	}

	var notes []models.Note
	if err := db.Where("user_id = ?", userID).
		Preload("Contact").
		Order("date DESC").
		Find(&notes).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch notes for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch notes"))
		return
	}

	var reminders []models.Reminder
	if err := db.Where("user_id = ?", userID).
		Preload("Contact").
		Order("remind_at ASC").
		Find(&reminders).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch reminders for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch reminders"))
		return
	}

	// Generate combined CSV content
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write contacts section
	buf.WriteString("=== CONTACTS ===\n")
	writer.Flush()

	contactHeaders := []string{
		"ID", "Firstname", "Lastname", "Nickname", "Gender", "Email", "Phone",
		"Birthday", "Address", "How We Met", "Food Preference", "Work Information",
		"Contact Information", "Circles", "Created At", "Updated At",
	}
	// Add custom field names as additional headers
	contactHeaders = append(contactHeaders, customFieldNames...)
	if err := writer.Write(contactHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write contact headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, contact := range contacts {
		record := []string{
			fmt.Sprintf("%d", contact.ID),
			contact.Firstname,
			contact.Lastname,
			contact.Nickname,
			contact.Gender,
			contact.Email,
			contact.Phone,
			contact.Birthday,
			contact.Address,
			contact.HowWeMet,
			contact.FoodPreference,
			contact.WorkInformation,
			contact.ContactInformation,
			strings.Join(contact.Circles, "; "),
			contact.CreatedAt.Format(time.RFC3339),
			contact.UpdatedAt.Format(time.RFC3339),
		}
		// Add custom field values
		for _, fieldName := range customFieldNames {
			value := ""
			if contact.CustomFields != nil {
				value = contact.CustomFields[fieldName]
			}
			record = append(record, value)
		}
		if err := writer.Write(record); err != nil {
			log.Error().Err(err).Msg("Failed to write contact record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Write relationships section
	buf.WriteString("\n=== RELATIONSHIPS ===\n")

	relationshipHeaders := []string{
		"ID", "Contact ID", "Contact Name", "Name", "Type", "Gender", "Birthday",
		"Related Contact ID", "Related Contact Name", "Created At", "Updated At",
	}
	if err := writer.Write(relationshipHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write relationship headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, contact := range contacts {
		for _, rel := range contact.Relationships {
			relatedContactName := ""
			if rel.RelatedContact != nil {
				relatedContactName = fmt.Sprintf("%s %s", rel.RelatedContact.Firstname, rel.RelatedContact.Lastname)
			}
			relatedContactID := ""
			if rel.RelatedContactID != nil {
				relatedContactID = fmt.Sprintf("%d", *rel.RelatedContactID)
			}

			record := []string{
				fmt.Sprintf("%d", rel.ID),
				fmt.Sprintf("%d", contact.ID),
				fmt.Sprintf("%s %s", contact.Firstname, contact.Lastname),
				rel.Name,
				rel.Type,
				rel.Gender,
				rel.Birthday,
				relatedContactID,
				relatedContactName,
				rel.CreatedAt.Format(time.RFC3339),
				rel.UpdatedAt.Format(time.RFC3339),
			}
			if err := writer.Write(record); err != nil {
				log.Error().Err(err).Msg("Failed to write relationship record")
				apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
				return
			}
		}
	}
	writer.Flush()

	// Write activities section
	buf.WriteString("\n=== ACTIVITIES ===\n")

	activityHeaders := []string{
		"ID", "Title", "Description", "Location", "Date", "Contact Names", "Created At", "Updated At",
	}
	if err := writer.Write(activityHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write activity headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, activity := range activities {
		contactNames := make([]string, len(activity.Contacts))
		for i, contact := range activity.Contacts {
			contactNames[i] = fmt.Sprintf("%s %s", contact.Firstname, contact.Lastname)
		}
		record := []string{
			fmt.Sprintf("%d", activity.ID),
			activity.Title,
			activity.Description,
			activity.Location,
			activity.Date.Format(time.RFC3339),
			strings.Join(contactNames, "; "),
			activity.CreatedAt.Format(time.RFC3339),
			activity.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			log.Error().Err(err).Msg("Failed to write activity record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Write notes section
	buf.WriteString("\n=== NOTES ===\n")

	noteHeaders := []string{
		"ID", "Contact ID", "Contact Name", "Content", "Date", "Created At", "Updated At",
	}
	if err := writer.Write(noteHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write note headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, note := range notes {
		contactID := ""
		contactName := ""
		if note.ContactID != nil {
			contactID = fmt.Sprintf("%d", *note.ContactID)
			contactName = fmt.Sprintf("%s %s", note.Contact.Firstname, note.Contact.Lastname)
		}
		record := []string{
			fmt.Sprintf("%d", note.ID),
			contactID,
			contactName,
			note.Content,
			note.Date.Format(time.RFC3339),
			note.CreatedAt.Format(time.RFC3339),
			note.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			log.Error().Err(err).Msg("Failed to write note record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Write reminders section
	buf.WriteString("\n=== REMINDERS ===\n")

	reminderHeaders := []string{
		"ID", "Contact ID", "Contact Name", "Message", "Remind At", "Recurrence",
		"By Mail", "Reoccur From Completion", "Completed", "Last Sent", "Created At", "Updated At",
	}
	if err := writer.Write(reminderHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write reminder headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, reminder := range reminders {
		contactID := ""
		contactName := ""
		if reminder.ContactID != nil {
			contactID = fmt.Sprintf("%d", *reminder.ContactID)
			contactName = fmt.Sprintf("%s %s", reminder.Contact.Firstname, reminder.Contact.Lastname)
		}

		byMail := "false"
		if reminder.ByMail != nil && *reminder.ByMail {
			byMail = "true"
		}

		reoccurFromCompletion := "true"
		if reminder.ReoccurFromCompletion != nil && !*reminder.ReoccurFromCompletion {
			reoccurFromCompletion = "false"
		}

		lastSent := ""
		if reminder.LastSent != nil {
			lastSent = reminder.LastSent.Format(time.RFC3339)
		}

		record := []string{
			fmt.Sprintf("%d", reminder.ID),
			contactID,
			contactName,
			reminder.Message,
			reminder.RemindAt.Format(time.RFC3339),
			reminder.Recurrence,
			byMail,
			reoccurFromCompletion,
			fmt.Sprintf("%t", reminder.Completed),
			lastSent,
			reminder.CreatedAt.Format(time.RFC3339),
			reminder.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			log.Error().Err(err).Msg("Failed to write reminder record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Check for any CSV writer errors
	if err := writer.Error(); err != nil {
		log.Error().Err(err).Msg("CSV writer error")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("meerkat-export-%s.csv", time.Now().Format("2006-01-02"))

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))

	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())

	log.Info().
		Int("contacts", len(contacts)).
		Int("activities", len(activities)).
		Int("notes", len(notes)).
		Int("reminders", len(reminders)).
		Msg("Data export completed successfully")
}

// ExportContactsAsVCF exports all user contacts as a VCF (vCard) file
func ExportContactsAsVCF(c *gin.Context, photoDir string) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Fetch all user contacts
	var contacts []models.Contact
	if err := db.Where("user_id = ?", userID).
		Order("firstname ASC, lastname ASC").
		Find(&contacts).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch contacts for VCF export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch contacts"))
		return
	}

	// Generate VCF content
	var buf bytes.Buffer
	encoder := vcard.NewEncoder(&buf)

	for _, contact := range contacts {
		card := carddav.ContactToVCard(&contact, photoDir)
		if err := encoder.Encode(card); err != nil {
			log.Error().Err(err).Uint("contact_id", contact.ID).Msg("Failed to encode contact as vCard")
			// Continue with other contacts instead of failing completely
			continue
		}
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("meerkat-contacts-%s.vcf", time.Now().Format("2006-01-02"))

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/vcard; charset=utf-8")
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))

	c.Data(http.StatusOK, "text/vcard; charset=utf-8", buf.Bytes())

	log.Info().
		Int("contacts", len(contacts)).
		Msg("VCF export completed successfully")
}
