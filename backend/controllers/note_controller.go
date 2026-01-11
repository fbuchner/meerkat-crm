package controllers

import (
	"errors"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateNote(c *gin.Context) {
	// Get the database instance from the context
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get contact ID from the request URL
	contactID := c.Param("id")

	// Find the contact by the ID
	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", contactID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Get validated input from validation middleware
	noteInput, err := middleware.GetValidated[models.NoteInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	// Create note from validated input
	note := models.Note{
		UserID:    userID,
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

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get validated input from validation middleware
	noteInput, err := middleware.GetValidated[models.NoteInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	// Create note from validated input
	note := models.Note{
		UserID:    userID,
		Content:   noteInput.Content,
		Date:      noteInput.Date,
		ContactID: noteInput.ContactID,
	}

	if note.ContactID != nil {
		var contact models.Contact
		if err := db.Where("user_id = ?", userID).First(&contact, *note.ContactID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact"))
			} else {
				apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
			}
			return
		}
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

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := db.Where("user_id = ?", userID).First(&note, id).Error; err != nil {
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

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))

	baseQuery := db.Model(&models.Note{}).
		Where("notes.user_id = ? AND contact_id IS NULL", userID)

	if search != "" {
		like := "%" + search + "%"
		baseQuery = baseQuery.Where("LOWER(content) LIKE ?", like)
	}

	countQuery := baseQuery.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to count notes").WithError(err))
		return
	}

	if err := baseQuery.Session(&gorm.Session{}).
		Order("notes.date DESC, notes.id DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&notes).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve unassigned notes").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notes": notes,
		"total": total,
		"page":  pagination.Page,
		"limit": pagination.Limit,
	})
}

func UpdateNote(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var note models.Note

	// Retrieve the existing note from the database
	if err := db.Where("user_id = ?", userID).First(&note, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Note").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve note").WithError(err))
		}
		return
	}

	// Get validated input from validation middleware
	updatedNote, err := middleware.GetValidated[models.NoteInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	// Updateable fields
	note.Content = updatedNote.Content
	note.Date = updatedNote.Date
	note.ContactID = updatedNote.ContactID

	if note.ContactID != nil {
		var contact models.Contact
		if err := db.Where("user_id = ?", userID).First(&contact, *note.ContactID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact"))
			} else {
				apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
			}
			return
		}
	}

	db.Updates(&note)

	c.JSON(http.StatusOK, gin.H{"message": "Note updated successfully", "note": note})
}

func DeleteNote(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Check if note exists first
	var note models.Note
	if err := db.Where("user_id = ?", userID).First(&note, id).Error; err != nil {
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

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Initialize a variable to store the contact
	var contact models.Contact

	// Fetch the contact and preload associated notes
	if err := db.Preload("Notes", "notes.user_id = ?", userID).Where("user_id = ?", userID).First(&contact, contactID).Error; err != nil {
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
