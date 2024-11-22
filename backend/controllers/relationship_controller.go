package controllers

import (
	"net/http"
	"perema/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AddRelationshipToContact(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// Parse contact ID from the request parameters
	contactIDParam := c.Param("id")
	contactID, err := strconv.Atoi(contactIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
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

	// Bind the request body to the Relationship struct
	var relationship models.Relationship
	if err := c.ShouldBindJSON(&relationship); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// If the relationship is linked to an existing contact, validate the contact
	if relationship.ContactID != nil {
		var relatedContact models.Contact
		if err := db.First(&relatedContact, *relationship.ContactID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Related contact not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find related contact"})
			return
		}
		relationship.RelatedContact = &relatedContact
	}

	// Append the relationship to the contact
	contact.Relationships = append(contact.Relationships, relationship)

	// Save the contact with the new relationship
	if err := db.Save(&contact).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add relationship to contact"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Relationship added to contact successfully"})
}
