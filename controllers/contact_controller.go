package controllers

import (
	"log"
	"net/http"
	"perema/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateContact(c *gin.Context) {
	var contact models.Contact
	if err := c.ShouldBindJSON(&contact); err != nil {
		log.Println("Error binding JSON for create contact:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle file upload
	//file, _ := c.FormFile("photo")
	//if file != nil {
	//	filePath := "./static/" + file.Filename
	//	c.SaveUploadedFile(file, filePath)
	//	contact.Photo = filePath
	//}

	// Save to the database
	db := c.MustGet("db").(*gorm.DB)

	// Save the new contact to the database
	if err := db.Create(&contact).Error; err != nil {
		log.Println("Error saving to database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save contact"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact created successfully", "contact": contact})
}

func GetAllContacts(c *gin.Context) {
	var contacts []models.Contact
	db := c.MustGet("db").(*gorm.DB)
	db.Find(&contacts)

	c.JSON(http.StatusOK, contacts)
}

func GetContact(c *gin.Context) {
	id := c.Param("id")
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&contact, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	c.JSON(http.StatusOK, contact)
}

func UpdateContact(c *gin.Context) {
	id := c.Param("id")
	var contact models.Contact
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&contact, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	if err := c.ShouldBindJSON(&contact); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Save(&contact)
	c.JSON(http.StatusOK, contact)
}

func DeleteContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	if err := db.Delete(&models.Contact{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted"})
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
