package controllers

import (
	"net/http"
	"perema/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateContact(c *gin.Context) {
	var contact models.Contact
	if err := c.ShouldBindJSON(&contact); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle file upload
	file, _ := c.FormFile("photo")
	if file != nil {
		filePath := "./static/" + file.Filename
		c.SaveUploadedFile(file, filePath)
		contact.Photo = filePath
	}

	// Save to the database
	db := c.MustGet("db").(*gorm.DB)
	db.Create(&contact)

	c.JSON(http.StatusOK, contact)
}

func CreateRelationship(c *gin.Context) {
	var relationship models.Relationship
	if err := c.ShouldBindJSON(&relationship); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	// If ContactID is provided, make sure it refers to a valid existing contact
	if relationship.ContactID != nil {
		var existingContact models.Contact
		if err := db.First(&existingContact, *relationship.ContactID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Related contact not found"})
			return
		}
		relationship.RelatedContact = &existingContact
	}

	db.Create(&relationship)
	c.JSON(http.StatusOK, relationship)
}
