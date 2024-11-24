package controllers

import (
	"log"
	"net/http"
	"perema/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetRelationships(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// Get contact ID from query parameter
	contactIDParam := c.Param("id")
	if contactIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Contact ID is required"})
		return
	}

	// Convert contact ID to integer
	contactID, err := strconv.Atoi(contactIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}

	// Find the contact by ID and preload relationships and related contacts
	var contact models.Contact
	if err := db.Preload("Relationships").Preload("Relationships.RelatedContact").First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
			return
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find contact"})
		return
	}

	// Prepare response to include only necessary fields from RelatedContact
	relationshipsResponse := make([]map[string]interface{}, len(contact.Relationships))
	for i, rel := range contact.Relationships {
		relationshipsResponse[i] = map[string]interface{}{
			"name":       rel.Name,
			"type":       rel.Type,
			"gender":     rel.Gender,
			"birthday":   rel.Birthday,
			"contact_id": rel.ContactID,
		}

		// If RelatedContact is not nil, include only selected fields
		if rel.RelatedContact != nil {
			relationshipsResponse[i]["related_contact"] = map[string]interface{}{
				"firstname": rel.RelatedContact.Firstname,
				"lastname":  rel.RelatedContact.Lastname,
				"gender":    rel.RelatedContact.Gender,
			}
		} else {
			relationshipsResponse[i]["related_contact"] = nil
		}
	}

	// Return the relationships for the contact
	c.JSON(http.StatusOK, gin.H{"relationships": relationshipsResponse})
}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
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
	}

	// Start a transaction to save the relationship properly
	err = db.Transaction(func(tx *gorm.DB) error {
		// Append the relationship to the contact
		contact.Relationships = append(contact.Relationships, relationship)

		// Save the contact with the new relationship
		if err := tx.Save(&contact).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add relationship to contact"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Relationship added to contact successfully"})
}
