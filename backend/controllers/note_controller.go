package controllers

import (
	"errors"
	"net/http"
	apperrors "perema/errors"
	"perema/logger"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", contactID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Get validated input from validation middleware
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("note", "validation data not found"))
		return
	}

	noteInput, ok := validated.(*models.NoteInput)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("note", "invalid validation data type"))
		return
	}

	// Create note from validated input
	note := models.Note{
		Content:   noteInput.Content,
		Date:      noteInput.Date,
		ContactID: &contact.ID,
	}

	// Save the new note to the database
	if err := db.Create(&note).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving note to database")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save note").WithError(err))
		return
	}

	// Respond with success and the created note
	c.JSON(http.StatusOK, gin.H{"message": "Note created successfully", "note": note})
}

func CreateUnassignedNote(c *gin.Context) {
	// Get the database instance from the context
	db := c.MustGet("db").(*gorm.DB)

	// Get validated input from validation middleware
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("note", "validation data not found"))
		return
	}

	noteInput, ok := validated.(*models.NoteInput)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("note", "invalid validation data type"))
		return
	}

	// Create note from validated input
	note := models.Note{
		Content:   noteInput.Content,
		Date:      noteInput.Date,
		ContactID: noteInput.ContactID,
	}

	// Save the new note to the database
	if err := db.Create(&note).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving unassigned note to database")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save note").WithError(err))
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Note").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve note").WithError(err))
		}
		return
	}

	c.JSON(http.StatusOK, note)
}

func GetUnassignedNotes(c *gin.Context) {
	var notes []models.Note
	db := c.MustGet("db").(*gorm.DB)

	// Retrieve notes where contact_id is NULL
	if err := db.Where("contact_id IS NULL").Find(&notes).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve unassigned notes").WithError(err))
		return
	}

	c.JSON(http.StatusOK, notes)
}

func UpdateNote(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id := c.Param("id")
	var note models.Note

	// Retrieve the existing note from the database
	if err := db.First(&note, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Note").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve note").WithError(err))
		}
		return
	}

	// Get validated input from validation middleware
	validated, exists := c.Get("validated")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("note", "validation data not found"))
		return
	}

	updatedNote, ok := validated.(*models.NoteInput)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("note", "invalid validation data type"))
		return
	}

	// Updateable fields
	note.Content = updatedNote.Content
	note.Date = updatedNote.Date
	note.ContactID = updatedNote.ContactID

	db.Updates(&note)

	c.JSON(http.StatusOK, gin.H{"message": "Note updated successfully", "note": note})
}

func DeleteNote(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	// Check if note exists first
	var note models.Note
	if err := db.First(&note, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Note").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve note").WithError(err))
		}
		return
	}

	if err := db.Delete(&note).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete note").WithError(err))
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If no contact found, return a 404 error
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", contactID))
		} else {
			// For any other errors, return a 500 error
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// If successful, return the contact and its notes as JSON
	c.JSON(http.StatusOK, gin.H{
		"notes": contact.Notes,
	})
}
