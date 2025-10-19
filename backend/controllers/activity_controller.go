package controllers

import (
	"log"
	"net/http"
	"perema/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateActivity(c *gin.Context) {
	var requestBody struct {
		Title       string    `json:"title"`
		Date        time.Time `json:"date"`
		Description string    `json:"description"`
		Location    string    `json:"location"`
		ContactIDs  []uint    `json:"contact_ids"` // Accept an array of contact IDs for many-to-many association
	}

	// Bind the incoming JSON to the requestBody
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		log.Println("Error binding JSON for create activity:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch the contacts from the database using the ContactIDs
	db := c.MustGet("db").(*gorm.DB)
	var contacts []models.Contact
	if len(requestBody.ContactIDs) > 0 {
		if err := db.Where("id IN ?", requestBody.ContactIDs).Find(&contacts).Error; err != nil {
			log.Println("Error finding contacts with IDs:", requestBody.ContactIDs, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "One or more contacts not found"})
			return
		}
	}

	// Create a new activity without the associations initially
	activity := models.Activity{
		Title:       requestBody.Title,
		Date:        requestBody.Date,
		Description: requestBody.Description,
		Location:    requestBody.Location,
	}

	// Save the new activity to the database
	if err := db.Create(&activity).Error; err != nil {
		log.Println("Error saving activity to the database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save activity"})
		return
	}

	// Update the activity's contacts association
	if len(contacts) > 0 {
		if err := db.Model(&activity).Association("Contacts").Append(contacts); err != nil {
			log.Println("Error creating relationship between activity and contacts:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate contacts with activity"})
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
	if err := db.Preload("Contacts").First(&activity, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func GetActivities(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB).Debug()

	// Get pagination parameters from query
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 25
	}

	// Calculate offset
	offset := (page - 1) * limit

	includeContacts := c.DefaultQuery("include", "") == "contacts"

	var activities []models.Activity
	var total int64

	// Get the total count of activities
	db.Model(&models.Activity{}).Count(&total)

	// Build the query with optional preloading and ordering by date in descending order
	query := db.Model(&models.Activity{}).
		Order("date DESC").
		Limit(limit).
		Offset(offset)

	if includeContacts {
		query = query.Preload("Contacts")
		log.Println("Preloading contacts")
	}

	// Execute the query
	if err := query.Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve activities"})
		return
	}

	// Include pagination metadata in response
	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"total":      total,
		"page":       page,
		"limit":      limit,
	})
}

func UpdateActivity(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	var activity models.Activity
	if err := db.Preload("Contacts").First(&activity, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	var requestBody struct {
		Title       string    `json:"title"`
		Date        time.Time `json:"date"`
		Description string    `json:"description"`
		Location    string    `json:"location"`
		ContactIDs  []uint    `json:"contact_ids"` // Accept an array of contact IDs for many-to-many association
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Updateable fields
	activity.Title = requestBody.Title
	activity.Description = requestBody.Description
	activity.Location = requestBody.Location
	activity.Date = requestBody.Date

	// Update contacts association if contact_ids are provided
	if requestBody.ContactIDs != nil {
		// Fetch the new contacts from the database
		var contacts []models.Contact
		if len(requestBody.ContactIDs) > 0 {
			if err := db.Where("id IN ?", requestBody.ContactIDs).Find(&contacts).Error; err != nil {
				log.Println("Error finding contacts with IDs:", requestBody.ContactIDs, err)
				c.JSON(http.StatusNotFound, gin.H{"error": "One or more contacts not found"})
				return
			}
		}

		// Replace the existing contacts association
		if err := db.Model(&activity).Association("Contacts").Replace(contacts); err != nil {
			log.Println("Error updating contacts association:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contacts association"})
			return
		}
	}

	// Save the activity
	if err := db.Save(&activity).Error; err != nil {
		log.Println("Error saving activity:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save activity"})
		return
	}

	// Reload the activity with contacts to return complete data
	if err := db.Preload("Contacts").First(&activity, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload activity"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func DeleteActivity(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	if err := db.Delete(&models.Activity{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity deleted"})
}

func GetActivitiesForContact(c *gin.Context) {
	// Get contact ID from the request URL
	contactID := c.Param("id")

	// Get the database instance from the context
	db := c.MustGet("db").(*gorm.DB)

	var activities []models.Activity
	// Eager load contacts associated with each activity
	if err := db.Preload("Contacts").
		Model(&models.Activity{}).
		Joins("JOIN activity_contacts ON activities.id = activity_contacts.activity_id").
		Where("activity_contacts.contact_id = ?", contactID).
		Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If successful, return the contact and its notes as JSON
	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
	})
}
