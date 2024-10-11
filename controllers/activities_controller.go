package controllers

import (
	"log"
	"net/http"
	"perema/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateActivity(c *gin.Context) {
	var activity models.Activity
	if err := c.ShouldBindJSON(&activity); err != nil {
		log.Println("Error binding JSON for create activity:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save to the database
	db := c.MustGet("db").(*gorm.DB)

	// Save the new activity to the database
	if err := db.Create(&activity).Error; err != nil {
		log.Println("Error saving to database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity created successfully", "activity": activity})
}

func GetActivity(c *gin.Context) {
	id := c.Param("id")
	var activity models.Activity
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&activity, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func UpdateActivity(c *gin.Context) {
	id := c.Param("id")
	var activity models.Activity
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&activity, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}

	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Save(&activity)
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

func AddContactToActivity(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// Parse activity ID from the request parameters
	activityIDParam := c.Param("activity_id")
	activityID, err := strconv.Atoi(activityIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity ID"})
		return
	}

	// Parse contact ID from the request parameters
	contactIDParam := c.Param("contact_id")
	contactID, err := strconv.Atoi(contactIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}

	// Find the activity by ID
	var activity models.Activity
	if err := db.First(&activity, activityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find activity"})
		return
	}

	// Find the contact by ID
	var contact models.Contact
	if err := db.First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find contact"})
		return
	}

	// Add the contact to the activity
	if err := db.Model(&activity).Association("Contacts").Append(&contact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contact to activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact added to activity successfully"})
}

func RemoveContactFromActivity(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// Parse activity ID from the request parameters
	activityIDParam := c.Param("activity_id")
	activityID, err := strconv.Atoi(activityIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity ID"})
		return
	}

	// Parse contact ID from the request parameters
	contactIDParam := c.Param("contact_id")
	contactID, err := strconv.Atoi(contactIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}

	// Find the activity by ID
	var activity models.Activity
	if err := db.First(&activity, activityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find activity"})
		return
	}

	// Find the contact by ID
	var contact models.Contact
	if err := db.First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find contact"})
		return
	}

	// Remove the contact from the activity
	if err := db.Model(&activity).Association("Contacts").Delete(&contact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove contact from activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact removed from activity successfully"})
}

func GetActivitiesForContact(c *gin.Context) {
	// Get contact ID from the request URL
	contactID := c.Param("id")

	// Get the database instance from the context
	db := c.MustGet("db").(*gorm.DB)

	// Initialize a variable to store the contact
	var contact models.Contact

	// Find the contact and preload associated activities
	if err := db.Preload("Activities").First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// If no contact found, return a 404 error
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		} else {
			// For any other errors, return a 500 error
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// If successful, return the contact and its notes as JSON
	c.JSON(http.StatusOK, gin.H{
		"activities": contact.Activities,
	})
}
