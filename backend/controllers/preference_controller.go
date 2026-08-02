package controllers

import (
	"errors"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreatePreference creates a new Preference (preference.go) for the
// authenticated user, scoped to a Contact they own via EntityID.
func CreatePreference(c *gin.Context) {
	input, err := middleware.GetValidated[models.PreferenceInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if !verifyOwnedContact(c, db, userID, input.EntityID) {
		return
	}

	pref := models.Preference{
		UserID:        userID,
		EntityID:      input.EntityID,
		Category:      input.Category,
		Key:           input.Key,
		Value:         input.Value,
		Source:        input.Source,
		Confidence:    input.Confidence,
		LastConfirmed: input.LastConfirmed,
	}
	if input.Sensitivity == "" {
		pref.Sensitivity = models.RelationshipSensitivityNormal
	} else {
		pref.Sensitivity = input.Sensitivity
	}

	if err := db.Create(&pref).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save preference").WithError(err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Preference created successfully", "preference": pref})
}

// GetPreference returns one Preference.
func GetPreference(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var pref models.Preference
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&pref).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Preference").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve preference").WithError(err))
		}
		return
	}

	c.JSON(http.StatusOK, pref)
}

// ListPreferences returns the authenticated user's Preferences, paginated,
// optionally filtered by ?entity_id=<Contact.VCardUID>.
func ListPreferences(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)
	entityID := c.Query("entity_id")

	var prefs []models.Preference
	var total int64

	baseQuery := db.Model(&models.Preference{}).Where("user_id = ?", userID)
	if entityID != "" {
		baseQuery = baseQuery.Where("entity_id = ?", entityID)
	}

	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to count preferences").WithError(err))
		return
	}

	if err := baseQuery.Session(&gorm.Session{}).
		Order("category, created_at").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&prefs).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve preferences").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"preferences": prefs,
		"total":       total,
		"page":        pagination.Page,
		"limit":       pagination.Limit,
	})
}

// UpdatePreference updates a Preference — full-replace semantics via the same
// PreferenceInput as create, matching UpdateLifeEvent's own precedent.
func UpdatePreference(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var pref models.Preference
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&pref).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Preference").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve preference").WithError(err))
		}
		return
	}

	input, err := middleware.GetValidated[models.PreferenceInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	if !verifyOwnedContact(c, db, userID, input.EntityID) {
		return
	}

	pref.EntityID = input.EntityID
	pref.Category = input.Category
	pref.Key = input.Key
	pref.Value = input.Value
	pref.Source = input.Source
	pref.Confidence = input.Confidence
	pref.LastConfirmed = input.LastConfirmed
	if input.Sensitivity == "" {
		pref.Sensitivity = models.RelationshipSensitivityNormal
	} else {
		pref.Sensitivity = input.Sensitivity
	}

	if err := db.Save(&pref).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save preference").WithError(err))
		return
	}

	c.JSON(http.StatusOK, pref)
}

// DeletePreference soft-deletes a Preference (user-authored content).
func DeletePreference(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var pref models.Preference
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&pref).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Preference").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve preference").WithError(err))
		}
		return
	}

	if err := db.Delete(&pref).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete preference").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preference deleted"})
}
