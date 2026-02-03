package controllers

import (
	"errors"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateActivity(c *gin.Context) {
	// Get validated input from validation middleware
	activityInput, err := middleware.GetValidated[models.ActivityInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	// Fetch the contacts from the database using the ContactIDs
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var contacts []models.Contact
	if len(activityInput.ContactIDs) > 0 {
		if err := db.Where("user_id = ? AND id IN ?", userID, activityInput.ContactIDs).Find(&contacts).Error; err != nil {
			logger.FromContext(c).Error().Err(err).Uints("contact_ids", activityInput.ContactIDs).Msg("Error finding contacts with IDs")
			apperrors.AbortWithError(c, apperrors.ErrNotFound("One or more contacts"))
			return
		}

		if len(contacts) != len(activityInput.ContactIDs) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("One or more contacts"))
			return
		}
	}

	// Create a new activity without the associations initially
	activity := models.Activity{
		UserID:      userID,
		Title:       activityInput.Title,
		Date:        activityInput.Date,
		Description: activityInput.Description,
		Location:    activityInput.Location,
	}

	// Save the new activity to the database
	if err := db.Create(&activity).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving activity to database")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save activity").WithError(err))
		return
	}

	// Update the activity's contacts association
	if len(contacts) > 0 {
		if err := db.Model(&activity).Association("Contacts").Append(contacts); err != nil {
			logger.FromContext(c).Error().Err(err).Msg("Error creating relationship between activity and contacts")
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to associate contacts with activity").WithError(err))
			return
		}
	}

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"message": "Activity created successfully", "activity": activity})
}

func GetActivity(c *gin.Context) {
	id := c.Param("id")
	var activity models.Activity
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := db.Preload("Contacts", func(db *gorm.DB) *gorm.DB {
		return db.Select("ID", "Firstname", "Lastname", "PhotoThumbnail", "Circles").Where("user_id = ?", userID)
	}).Where("user_id = ?", userID).First(&activity, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Activity").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve activity").WithError(err))
		}
		return
	}

	c.JSON(http.StatusOK, activity)
}

func GetActivities(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get pagination parameters from query
	pagination := GetPaginationParams(c)

	includeContacts := c.DefaultQuery("include", "") == "contacts"
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	var activities []models.Activity
	var total int64

	baseQuery := db.Model(&models.Activity{}).
		Where("activities.user_id = ?", userID)

	// Apply date filters
	if fromDateStr != "" {
		if fromDate, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			baseQuery = baseQuery.Where("activities.date >= ?", fromDate)
		}
	}
	if toDateStr != "" {
		if toDate, err := time.Parse("2006-01-02", toDateStr); err == nil {
			// Add one day to include the entire end date
			toDate = toDate.AddDate(0, 0, 1)
			baseQuery = baseQuery.Where("activities.date < ?", toDate)
		}
	}

	if search != "" {
		like := "%" + search + "%"
		searchClause := db.Where("LOWER(activities.title) LIKE ?", like).
			Or("LOWER(activities.description) LIKE ?", like).
			Or("LOWER(activities.location) LIKE ?", like).
			Or("LOWER(COALESCE(contacts.firstname, '')) LIKE ?", like).
			Or("LOWER(COALESCE(contacts.lastname, '')) LIKE ?", like).
			Or("LOWER(COALESCE(contacts.nickname, '')) LIKE ?", like)

		baseQuery = baseQuery.
			Select("DISTINCT activities.*").
			Joins("LEFT JOIN activity_contacts ON activity_contacts.activity_id = activities.id").
			Joins("LEFT JOIN contacts ON contacts.id = activity_contacts.contact_id AND contacts.user_id = ?", userID).
			Where(searchClause)
	}

	countQuery := baseQuery.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to count activities").WithError(err))
		return
	}

	query := baseQuery.Session(&gorm.Session{}).
		Order("activities.date DESC, activities.id DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset)

	if includeContacts {
		query = query.Preload("Contacts", func(db *gorm.DB) *gorm.DB {
			return db.Select("ID", "Firstname", "Lastname", "PhotoThumbnail", "Circles").Where("user_id = ?", userID)
		})
		logger.FromContext(c).Debug().Msg("Preloading contacts for activities")
	}

	if err := query.Find(&activities).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve activities").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"total":      total,
		"page":       pagination.Page,
		"limit":      pagination.Limit,
	})
}

func UpdateActivity(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var activity models.Activity
	if err := db.Preload("Contacts", "contacts.user_id = ?", userID).Where("user_id = ?", userID).First(&activity, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Activity").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve activity").WithError(err))
		}
		return
	}

	// Get validated input from validation middleware
	activityInput, err := middleware.GetValidated[models.ActivityInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	// Updateable fields
	activity.Title = activityInput.Title
	activity.Description = activityInput.Description
	activity.Location = activityInput.Location
	activity.Date = activityInput.Date

	// Update contacts association if contact_ids are provided
	if activityInput.ContactIDs != nil {
		// Fetch the new contacts from the database
		var contacts []models.Contact
		if len(activityInput.ContactIDs) > 0 {
			if err := db.Where("user_id = ? AND id IN ?", userID, activityInput.ContactIDs).Find(&contacts).Error; err != nil {
				logger.FromContext(c).Error().Err(err).Uints("contact_ids", activityInput.ContactIDs).Msg("Error finding contacts with IDs")
				apperrors.AbortWithError(c, apperrors.ErrNotFound("One or more contacts"))
				return
			}

			if len(contacts) != len(activityInput.ContactIDs) {
				apperrors.AbortWithError(c, apperrors.ErrNotFound("One or more contacts"))
				return
			}
		}

		// Replace the existing contacts association
		if err := db.Model(&activity).Association("Contacts").Replace(contacts); err != nil {
			logger.FromContext(c).Error().Err(err).Msg("Error updating contacts association")
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to update contacts association").WithError(err))
			return
		}
	}

	// Save the activity
	if err := db.Save(&activity).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving activity")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save activity").WithError(err))
		return
	}

	// Reload the activity with contacts to return complete data
	if err := db.Preload("Contacts", "contacts.user_id = ?", userID).Where("user_id = ?", userID).First(&activity, id).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to reload activity").WithError(err))
		return
	}

	c.JSON(http.StatusOK, activity)
}

func DeleteActivity(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Check if activity exists first
	var activity models.Activity
	if err := db.Where("user_id = ?", userID).First(&activity, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Activity").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve activity").WithError(err))
		}
		return
	}

	if err := db.Delete(&activity).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete activity").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity deleted"})
}

func GetActivitiesForContact(c *gin.Context) {
	// Get contact ID from the request URL
	contactID := c.Param("id")

	// Get the database instance from the context
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", contactID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	var activities []models.Activity
	// Eager load contacts associated with each activity
	if err := db.Preload("Contacts", func(db *gorm.DB) *gorm.DB {
		return db.Select("ID", "Firstname", "Lastname", "PhotoThumbnail", "Circles").Where("user_id = ?", userID)
	}).
		Model(&models.Activity{}).
		Where("activities.user_id = ?", userID).
		Joins("JOIN activity_contacts ON activities.id = activity_contacts.activity_id").
		Where("activity_contacts.contact_id = ?", contact.ID).
		Find(&activities).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve activities for contact").WithError(err))
		return
	}

	// If successful, return the contact and its notes as JSON
	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
	})
}
