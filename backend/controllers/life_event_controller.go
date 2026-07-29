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

// verifyOwnedContact confirms vcardUID references a Contact owned by userID
// — the same ownership check every entity that references Contact.VCardUID
// needs (mirrors Activity's own ContactIDs ownership check).
func verifyOwnedContact(c *gin.Context, db *gorm.DB, userID uint, vcardUID string) bool {
	var contact models.Contact
	if err := db.Where("vcard_uid = ? AND user_id = ?", vcardUID, userID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("vcard_uid", vcardUID))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return false
	}
	return true
}

// CreateLifeEvent creates a new LifeEvent (life_event.go) for the
// authenticated user.
func CreateLifeEvent(c *gin.Context) {
	input, err := middleware.GetValidated[models.LifeEventInput](c)
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

	event := models.LifeEvent{
		UserID:           userID,
		EntityID:         input.EntityID,
		Type:             input.Type,
		Date:             input.Date,
		Description:      input.Description,
		Source:           input.Source,
		RelatedEntityIDs: input.RelatedEntityIDs,
	}
	if err := db.Create(&event).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save life event").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Life event created successfully", "life_event": event})
}

// GetLifeEvent returns one LifeEvent.
func GetLifeEvent(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var event models.LifeEvent
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Life event").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve life event").WithError(err))
		}
		return
	}

	c.JSON(http.StatusOK, event)
}

// ListLifeEvents returns the authenticated user's LifeEvents, paginated,
// optionally filtered by ?entity_id=<Contact.VCardUID>.
func ListLifeEvents(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)
	entityID := c.Query("entity_id")

	var events []models.LifeEvent
	var total int64

	baseQuery := db.Model(&models.LifeEvent{}).Where("user_id = ?", userID)
	if entityID != "" {
		baseQuery = baseQuery.Where("entity_id = ?", entityID)
	}

	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to count life events").WithError(err))
		return
	}

	if err := baseQuery.Session(&gorm.Session{}).
		Order("id DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&events).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve life events").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"life_events": events,
		"total":       total,
		"page":        pagination.Page,
		"limit":       pagination.Limit,
	})
}

// UpdateLifeEvent updates a LifeEvent.
func UpdateLifeEvent(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var event models.LifeEvent
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Life event").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve life event").WithError(err))
		}
		return
	}

	input, err := middleware.GetValidated[models.LifeEventInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	if !verifyOwnedContact(c, db, userID, input.EntityID) {
		return
	}

	event.EntityID = input.EntityID
	event.Type = input.Type
	event.Date = input.Date
	event.Description = input.Description
	event.Source = input.Source
	event.RelatedEntityIDs = input.RelatedEntityIDs

	if err := db.Save(&event).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save life event").WithError(err))
		return
	}

	c.JSON(http.StatusOK, event)
}

// DeleteLifeEvent deletes a LifeEvent.
func DeleteLifeEvent(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var event models.LifeEvent
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Life event").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve life event").WithError(err))
		}
		return
	}

	if err := db.Delete(&event).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete life event").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Life event deleted"})
}
