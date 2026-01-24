package controllers

import (
	"errors"
	apperrors "meerkat/errors"
	"meerkat/middleware"
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

	// Query the database for relationships belonging to the given contact ID, preloading related contact fields
	if err := db.Preload("RelatedContact", func(db *gorm.DB) *gorm.DB {
		return db.Select("ID", "Firstname", "Lastname", "Gender", "Birthday").Where("user_id = ?", userID)
	}).Where("user_id = ? AND contact_id = ?", userID, contactID).Find(&relationships).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationships").WithError(err))
		return
	}

	// Return the retrieved relationships in JSON format
	c.JSON(http.StatusOK, gin.H{"relationships": relationships})
}

// GetIncomingRelationships retrieves all relationships pointing to a given contact
func GetIncomingRelationships(c *gin.Context) {
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

	var incomingRelationships []models.Relationship

	// Query relationships where this contact is the target (RelatedContactID)
	if err := db.Preload("SourceContact", func(db *gorm.DB) *gorm.DB {
		return db.Select("ID", "Firstname", "Lastname").Where("user_id = ?", userID)
	}).Where("user_id = ? AND related_contact_id = ?", userID, contactID).Find(&incomingRelationships).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve incoming relationships").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"incoming_relationships": incomingRelationships})
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

	// Get validated input from validation middleware
	input, validationErr := middleware.GetValidated[models.RelationshipInput](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
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

	// Create the relationship model from input
	relationship := models.Relationship{
		Name:             input.Name,
		Type:             input.Type,
		Gender:           input.Gender,
		Birthday:         input.Birthday,
		RelatedContactID: input.RelatedContactID,
		ContactID:        uint(contactID),
		UserID:           userID,
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

	// Get validated input from validation middleware
	input, err := middleware.GetValidated[models.RelationshipInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	// Update fields from input (ContactID stays the same, comes from URL)
	relationship.Name = input.Name
	relationship.Type = input.Type
	relationship.Gender = input.Gender
	relationship.Birthday = input.Birthday
	relationship.RelatedContactID = input.RelatedContactID

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

	db.Save(&relationship)

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
