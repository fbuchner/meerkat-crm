package controllers

import (
	"net/http"
	"perema/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetRelationships retrieves all relationships for a given contact
func GetRelationships(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	contactID := c.Param("id")

	// Define a slice to hold the retrieved relationships
	var relationships []models.Relationship

	// Query the database for relationships belonging to the given contact ID, preloading only specific fields of RelatedContact
	if err := db.Preload("RelatedContact", func(db *gorm.DB) *gorm.DB {
		return db.Select("ID", "Firstname", "Lastname")
	}).Where("contact_id = ?", contactID).Find(&relationships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the retrieved relationships in JSON format
	c.JSON(http.StatusOK, gin.H{"relationships": relationships})
}

// CreateRelationship creates a new relationship for a given contact
func CreateRelationship(c *gin.Context) {
	// Retrieve the database instance from context
	db := c.MustGet("db").(*gorm.DB)

	// Get the contact ID from the request parameters
	contactIDParam := c.Param("id")
	contactID, err := strconv.Atoi(contactIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}

	// Bind the JSON input to a new Relationship object
	var relationship models.Relationship
	if err := c.ShouldBindJSON(&relationship); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set the ContactID to associate the relationship with the given contact
	relationship.ContactID = uint(contactID)
	// Save the new relationship to the database
	if err := db.Create(&relationship).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the created relationship in JSON format
	c.JSON(http.StatusCreated, gin.H{"relationship": relationship})
}

func UpdateRelationship(c *gin.Context) {
	id := c.Param("rid")
	var relationship models.Relationship
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&relationship, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Relationship not found"})
		return
	}

	var updatedRelationship models.Relationship
	if err := c.ShouldBindJSON(&updatedRelationship); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Updateable fields
	relationship.Name = updatedRelationship.Name
	relationship.Type = updatedRelationship.Type
	relationship.Gender = updatedRelationship.Gender
	relationship.Birthday = updatedRelationship.Birthday
	relationship.ContactID = updatedRelationship.ContactID
	relationship.RelatedContactID = updatedRelationship.RelatedContactID

	db.Updates(&relationship)

	c.JSON(http.StatusOK, relationship)
}

func DeleteRelationship(c *gin.Context) {
	id := c.Param("rid")
	db := c.MustGet("db").(*gorm.DB)
	if err := db.Delete(&models.Relationship{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Relationship not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Relationship deleted"})
}
