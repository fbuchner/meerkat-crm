package controllers

import (
	"errors"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateContact(c *gin.Context) {
	// Save to the database
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get validated input from validation middleware
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact", "validation data not found"))
		return
	}

	contactInput, ok := validated.(*models.ContactInput)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact", "invalid validation data type"))
		return
	}

	// Create contact from validated input
	contact := models.Contact{
		UserID:             userID,
		Firstname:          contactInput.Firstname,
		Lastname:           contactInput.Lastname,
		Nickname:           contactInput.Nickname,
		Gender:             contactInput.Gender,
		Email:              contactInput.Email,
		Phone:              contactInput.Phone,
		Birthday:           contactInput.Birthday,
		Address:            contactInput.Address,
		HowWeMet:           contactInput.HowWeMet,
		FoodPreference:     contactInput.FoodPreference,
		WorkInformation:    contactInput.WorkInformation,
		ContactInformation: contactInput.ContactInformation,
		Circles:            contactInput.Circles,
	}

	// Save the new contact to the database
	if err := db.Create(&contact).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving contact to database")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save contact").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact created successfully", "contact": contact})
}

func GetContacts(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)

	// Define allowed fields and parse requested fields with validation
	allowedFields := []string{"ID", "firstname", "lastname", "nickname", "gender", "email", "phone", "birthday", "address", "how_we_met", "food_preference", "work_information", "contact_information", "circles", "photo", "photo_thumbnail"}
	var selectedFields []string
	fields := c.Query("fields")
	if fields != "" {
		for _, field := range strings.Split(fields, ",") {
			if slices.Contains(allowedFields, field) { // Validate field
				selectedFields = append(selectedFields, field)
			}
		}
	} else {
		selectedFields = allowedFields // Use all allowed fields if none are specified
	}

	// Parse relationships to include with validation
	var relationshipMap = map[string]bool{
		"notes":         false,
		"activities":    false,
		"relationships": false,
		"reminders":     false,
	}
	includes := c.Query("includes")
	for _, rel := range strings.Split(includes, ",") {
		if _, exists := relationshipMap[rel]; exists {
			relationshipMap[rel] = true
		}
	}

	// Parse and validate sort parameters
	sortField := c.DefaultQuery("sort", "id")
	sortOrder := c.DefaultQuery("order", "desc")

	allowedSortFields := map[string]bool{"firstname": true, "lastname": true, "id": true}
	if !allowedSortFields[sortField] {
		sortField = "id"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	var contacts []models.Contact
	query := db.Model(&models.Contact{}).Where("user_id = ?", userID).Order(sortField + " " + sortOrder).Limit(pagination.Limit).Offset(pagination.Offset)

	if len(selectedFields) > 0 {
		query = query.Select(selectedFields)
	}

	// Apply search filter using parameterization
	if searchTerm := c.Query("search"); searchTerm != "" {
		searchTermParam := "%" + searchTerm + "%"
		query = query.Where("firstname LIKE ? OR lastname LIKE ? OR nickname LIKE ?", searchTermParam, searchTermParam, searchTermParam)
	}

	if circle := c.Query("circle"); circle != "" {
		query = query.Where("circles LIKE ?", "%"+circle+"%") // Using parameterization
	}

	// Preload requested relationships
	for rel, include := range relationshipMap {
		if include {
			switch rel {
			case "notes":
				query = query.Preload("Notes", "notes.user_id = ?", userID)
			case "activities":
				query = query.Preload("Activities", "activities.user_id = ?", userID)
			case "relationships":
				query = query.Preload("Relationships", "relationships.user_id = ?", userID)
			case "reminders":
				query = query.Preload("Reminders", "reminders.user_id = ?", userID)
			}
		}
	}

	// Execute query
	if err := query.Find(&contacts).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contacts").WithError(err))
		return
	}

	var total int64
	countQuery := db.Model(&models.Contact{}).Where("user_id = ?", userID)

	// Apply the same search filters to the count query
	if searchTerm := c.Query("search"); searchTerm != "" {
		searchTermParam := "%" + searchTerm + "%"
		countQuery = countQuery.Where("firstname LIKE ? OR lastname LIKE ? OR nickname LIKE ?", searchTermParam, searchTermParam, searchTermParam)
	}

	if circle := c.Query("circle"); circle != "" {
		countQuery = countQuery.Where("circles LIKE ?", "%"+circle+"%")
	}

	countQuery.Count(&total)

	// Map contacts to ContactResponse with thumbnail URLs
	contactResponses := make([]models.ContactResponse, len(contacts))
	for i, contact := range contacts {
		contactResponses[i] = models.ContactResponse{
			Contact:      contact,
			ThumbnailURL: contact.PhotoThumbnail, // Already base64 data URL
		}
	}

	// Respond with contacts and pagination metadata
	c.JSON(http.StatusOK, gin.H{
		"contacts": contactResponses,
		"total":    total,
		"page":     pagination.Page,
		"limit":    pagination.Limit,
	})
}

func GetContactsRandom(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var selectedFields = []string{"ID", "firstname", "lastname", "nickname", "circles", "photo_thumbnail"}

	var contacts []models.Contact
	query := db.Model(&models.Contact{}).Where("user_id = ?", userID)

	if len(selectedFields) > 0 {
		query = query.Select(selectedFields)
	}

	// Get 5 random contacts
	query = query.Order("RANDOM()").Limit(5)

	// Execute query
	if err := query.Find(&contacts).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contacts").WithError(err))
		return
	}

	// Map to response with thumbnail_url
	contactResponses := make([]models.ContactResponse, len(contacts))
	for i, contact := range contacts {
		contactResponses[i] = models.ContactResponse{
			Contact:      contact,
			ThumbnailURL: contact.PhotoThumbnail,
		}
	}

	// Respond with random contacts
	c.JSON(http.StatusOK, gin.H{
		"contacts": contactResponses,
	})
}

func GetUpcomingBirthdays(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	birthdays, err := services.GetUpcomingBirthdays(db, userID)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve upcoming birthdays").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"birthdays": birthdays,
	})
}

func GetContact(c *gin.Context) {
	id := c.Param("id")

	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)
	if err := db.
		Where("user_id = ?", userID).
		Preload("Notes", "notes.user_id = ?", userID).
		Preload("Activities", "activities.user_id = ?", userID).
		Preload("Relationships", "relationships.user_id = ?", userID).
		Preload("Reminders", "reminders.user_id = ?", userID).
		First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}
	c.JSON(http.StatusOK, contact)
}

func UpdateContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Get validated input from validation middleware
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact", "validation data not found"))
		return
	}

	contactInput, ok := validated.(*models.ContactInput)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact", "invalid validation data type"))
		return
	}

	// Updateable fields
	contact.Firstname = contactInput.Firstname
	contact.Lastname = contactInput.Lastname
	contact.Nickname = contactInput.Nickname
	contact.Gender = contactInput.Gender
	contact.Email = contactInput.Email
	contact.Phone = contactInput.Phone
	contact.Birthday = contactInput.Birthday
	contact.Address = contactInput.Address
	contact.HowWeMet = contactInput.HowWeMet
	contact.FoodPreference = contactInput.FoodPreference
	contact.WorkInformation = contactInput.WorkInformation
	contact.ContactInformation = contactInput.ContactInformation
	contact.Circles = contactInput.Circles

	db.Save(&contact)

	c.JSON(http.StatusOK, contact)
}

func DeleteContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Check if contact exists first
	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Start a transaction to ensure all deletes succeed together
	err := db.Transaction(func(tx *gorm.DB) error {
		// Manually delete associated reminders (soft delete doesn't trigger CASCADE)
		if err := tx.Where("contact_id = ? AND user_id = ?", id, userID).Delete(&models.Reminder{}).Error; err != nil {
			return err
		}

		// Manually delete associated notes
		if err := tx.Where("contact_id = ? AND user_id = ?", id, userID).Delete(&models.Note{}).Error; err != nil {
			return err
		}

		// Manually delete associated relationships
		if err := tx.Where("contact_id = ? AND user_id = ?", id, userID).Delete(&models.Relationship{}).Error; err != nil {
			return err
		}

		// Delete activity associations (many-to-many)
		if err := tx.Exec("DELETE FROM activity_contacts WHERE contact_id = ? AND activity_id IN (SELECT id FROM activities WHERE user_id = ?)", id, userID).Error; err != nil {
			return err
		}

		// Finally, delete the contact
		if err := tx.Delete(&contact).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete contact and associated data").WithError(err))
		return
	}

	// Cleanup profile photos after successful database transaction
	// This is done outside the transaction since file deletion cannot be rolled back
	deleteContactPhotos(c, contact)

	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted"})
}

// deleteContactPhotos removes the profile photo file for a contact
// Note: thumbnails are stored as base64 in the database, not as files
func deleteContactPhotos(c *gin.Context, contact models.Contact) {
	uploadDir := os.Getenv("PROFILE_PHOTO_DIR")
	if uploadDir == "" {
		return
	}

	log := logger.FromContext(c)

	// Delete main photo if it exists
	if contact.Photo != "" {
		photoPath := filepath.Join(uploadDir, contact.Photo)
		if err := os.Remove(photoPath); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", photoPath).Msg("Failed to delete contact photo")
		} else if err == nil {
			log.Debug().Str("path", photoPath).Msg("Deleted contact photo")
		}
	}

	// Delete legacy file-based thumbnail if it exists (not base64 data URL)
	if contact.PhotoThumbnail != "" && !strings.HasPrefix(contact.PhotoThumbnail, "data:") {
		thumbnailPath := filepath.Join(uploadDir, contact.PhotoThumbnail)
		if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", thumbnailPath).Msg("Failed to delete contact thumbnail")
		} else if err == nil {
			log.Debug().Str("path", thumbnailPath).Msg("Deleted contact thumbnail")
		}
	}
}

// GetCircles returns all unique circles associated with contacts.
func GetCircles(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var circleNames []string

	// Raw SQL query to extract unique circle names
	err := db.Raw(`SELECT DISTINCT json_each.value AS circle
	               FROM contacts, json_each(contacts.circles)
	               WHERE contacts.user_id = ?`, userID).Scan(&circleNames).Error
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circles").WithError(err))
		return
	}

	// Return the list of unique circle names
	c.JSON(http.StatusOK, circleNames)
}
