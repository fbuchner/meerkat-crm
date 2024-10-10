package controllers

import (
	"log"
	"net/http"
	"perema/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateNote(c *gin.Context) {
	// Get the database instance from the context
	db := c.MustGet("db").(*gorm.DB)

	// Get contact ID from the request URL
	contactID := c.Param("id")

	// Find the contact by the ID
	var contact models.Contact
	if err := db.First(&contact, contactID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Bind the incoming JSON request to the Note struct
	var note models.Note
	if err := c.ShouldBindJSON(&note); err != nil {
		log.Println("Error binding JSON for create note:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assign the ContactID to the note to link it to the contact
	note.ContactID = contact.ID

	// Save the new note to the database
	if err := db.Create(&note).Error; err != nil {
		log.Println("Error saving to database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save note"})
		return
	}

	// Respond with success and the created note
	c.JSON(http.StatusOK, gin.H{"message": "Note created successfully", "note": note})
}

func GetNote(c *gin.Context) {
	id := c.Param("id")
	var note models.Note
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	c.JSON(http.StatusOK, note)
}

func UpdateNote(c *gin.Context) {
	id := c.Param("id")
	var note models.Note
	db := c.MustGet("db").(*gorm.DB)
	if err := db.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	if err := c.ShouldBindJSON(&note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Save(&note)
	c.JSON(http.StatusOK, note)
}

func DeleteNote(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	if err := db.Delete(&models.Note{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Note deleted"})
}

// GetNotesForContact retrieves all notes for a given contact
func GetNotesForContact(c *gin.Context) {
	// Get contact ID from the request URL
	contactID := c.Param("id")

	// Get the database instance from the context
	db := c.MustGet("db").(*gorm.DB)

	// Initialize a variable to store the contact
	var contact models.Contact

	// Fetch the contact and preload associated notes
	if err := db.Preload("Notes").First(&contact, contactID).Error; err != nil {
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
		"notes": contact.Notes,
	})
}
