package controllers

import (
	"errors"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetDuplicateContacts lists pairs of contacts that probably describe the same person.
func GetDuplicateContacts(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pairs, err := services.FindDuplicateContacts(db, userID)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to detect duplicate contacts").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"duplicates": pairs})
}

// DismissDuplicateContacts marks two contacts as not duplicates of each other.
func DismissDuplicateContacts(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	input, appErr := middleware.GetValidated[models.DuplicateDismissInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	if err := services.DismissDuplicatePair(db, userID, input.ContactAID, input.ContactBID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact"))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to dismiss duplicate").WithError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Duplicate dismissed"})
}

// GetContactMergePreview reports the conflicting fields and the data that
// merging the contact in ?source_id= into the contact in the URL would move.
func GetContactMergePreview(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	targetID, ok := parseIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}
	sourceID, ok := parseIDParam(c, c.Query("source_id"), "source_id")
	if !ok {
		return
	}

	preview, err := services.PreviewContactMerge(db, userID, targetID, sourceID)
	if err != nil {
		abortMergeError(c, err, "Failed to preview contact merge")
		return
	}

	c.JSON(http.StatusOK, preview)
}

// MergeContact merges the source contact from the body into the contact in the
// URL and deletes the source.
func MergeContact(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	targetID, ok := parseIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	input, appErr := middleware.GetValidated[models.ContactMergeInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	result, err := services.MergeContacts(db, userID, targetID, input.SourceID, input.Resolutions)
	if err != nil {
		abortMergeError(c, err, "Failed to merge contacts")
		return
	}

	// File deletion cannot be rolled back, so it waits for the commit.
	deleteContactPhotos(c, result.DiscardedPhoto)

	logger.FromContext(c).Info().Uint("target_id", targetID).Uint("source_id", result.SourceID).Msg("Merged contacts")

	cfg := currentConfig(c)
	go services.TriggerWebhooks(db, cfg, userID, "contact.updated", result.Contact)
	go services.TriggerWebhooks(db, cfg, userID, "contact.deleted", gin.H{"id": result.SourceID})
	c.JSON(http.StatusOK, result.Contact)
}

func parseIDParam(c *gin.Context, raw, field string) (uint, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput(field, "must be a positive integer"))
		return 0, false
	}
	return uint(id), true
}

func abortMergeError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, services.ErrMergeSameContact):
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("source_id", "must differ from the target contact"))
	case errors.Is(err, gorm.ErrRecordNotFound):
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact"))
	default:
		apperrors.AbortWithError(c, apperrors.ErrDatabase(operation).WithError(err))
	}
}
