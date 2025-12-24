package controllers

import (
	"errors"
	apperrors "meerkat/errors"
	"meerkat/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetRelationships retrieves all relationships for a given contact
func GetRelationships(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	contactID := c.Param("id")

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

	// Define a slice to hold the retrieved relationships
	var relationships []models.Relationship

	// Query the database for relationships belonging to the given contact ID, preloading only specific fields of RelatedContact
	if err := db.Preload("RelatedContact", func(db *gorm.DB) *gorm.DB {
		return db.Select("ID", "Firstname", "Lastname").Where("user_id = ?", userID)
	}).Where("user_id = ? AND contact_id = ?", userID, contactID).Find(&relationships).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationships").WithError(err))
		return
	}

	// Return the retrieved relationships in JSON format
	c.JSON(http.StatusOK, gin.H{"relationships": relationships})
}

// CreateRelationship creates a new relationship for a given contact
func CreateRelationship(c *gin.Context) {
	// Retrieve the database instance from context
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get the contact ID from the request parameters
	contactIDParam := c.Param("id")
	contactID, err := strconv.Atoi(contactIDParam)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("contact_id", "Invalid contact ID"))
		return
	}

	// Bind the JSON input to a new Relationship object
	var relationship models.Relationship
	if err := c.ShouldBindJSON(&relationship); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("relationship", err.Error()))
		return
	}

	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", contactIDParam))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Set the ContactID to associate the relationship with the given contact
	relationship.ContactID = uint(contactID)
	relationship.UserID = userID

	if relationship.RelatedContactID != nil {
		var relatedContact models.Contact
		if err := db.Where("user_id = ?", userID).First(&relatedContact, *relationship.RelatedContactID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				apperrors.AbortWithError(c, apperrors.ErrNotFound("Related contact").WithDetails("id", strconv.FormatUint(uint64(*relationship.RelatedContactID), 10)))
			} else {
				apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve related contact").WithError(err))
			}
			return
		}
	}
	// Save the new relationship to the database
	if err := db.Create(&relationship).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save relationship").WithError(err))
		return
	}

	// Return the created relationship in JSON format
	c.JSON(http.StatusCreated, gin.H{"relationship": relationship})
}

func UpdateRelationship(c *gin.Context) {
	id := c.Param("rid")
	var relationship models.Relationship
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := db.Where("user_id = ?", userID).First(&relationship, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Relationship").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationship").WithError(err))
		}
		return
	}

	var updatedRelationship models.Relationship
	if err := c.ShouldBindJSON(&updatedRelationship); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("relationship", err.Error()))
		return
	}

	// Updateable fields
	relationship.Name = updatedRelationship.Name
	relationship.Type = updatedRelationship.Type
	relationship.Gender = updatedRelationship.Gender
	relationship.Birthday = updatedRelationship.Birthday
	relationship.ContactID = updatedRelationship.ContactID
	relationship.RelatedContactID = updatedRelationship.RelatedContactID

	if relationship.ContactID != 0 {
		var contact models.Contact
		if err := db.Where("user_id = ?", userID).First(&contact, relationship.ContactID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", strconv.FormatUint(uint64(relationship.ContactID), 10)))
			} else {
				apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
			}
			return
		}
	}

	if relationship.RelatedContactID != nil {
		var relatedContact models.Contact
		if err := db.Where("user_id = ?", userID).First(&relatedContact, *relationship.RelatedContactID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				apperrors.AbortWithError(c, apperrors.ErrNotFound("Related contact").WithDetails("id", strconv.FormatUint(uint64(*relationship.RelatedContactID), 10)))
			} else {
				apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve related contact").WithError(err))
			}
			return
		}
	}

	db.Updates(&relationship)

	c.JSON(http.StatusOK, relationship)
}

func DeleteRelationship(c *gin.Context) {
	id := c.Param("rid")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Check if relationship exists first
	var relationship models.Relationship
	if err := db.Where("user_id = ?", userID).First(&relationship, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Relationship").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationship").WithError(err))
		}
		return
	}

	if err := db.Delete(&relationship).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete relationship").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Relationship deleted"})
}
