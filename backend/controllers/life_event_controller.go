package controllers

import (
	"errors"
	"fmt"
	"mycorrhizal/contactmodel"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// defaultReminderHour is the hour (0-23) at which life-event reminders fire
// in the user's configured timezone. Centralized so a future per-user
// preference can reference it; not yet surfaced in the UI.
const defaultReminderHour = 9

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

// getContactIDByVCardUID resolves a VCardUID to a numeric Contact ID owned by
// the given user. Returns nil, nil if the contact does not exist; returns
// nil, error for any other DB failure (transient, connection, etc.) so the
// caller can decide whether to abort or fall through.
func getContactIDByVCardUID(db *gorm.DB, userID uint, vcardUID string) (*uint, error) {
	var contact models.Contact
	if err := db.Where("vcard_uid = ? AND user_id = ?", vcardUID, userID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &contact.ID, nil
}

// eventHasMonthDay returns true when the PartialDate has both month and day
// populated, which is required for yearly recurrence (T5b).
func eventHasMonthDay(d *contactmodel.PartialDate) bool {
	if d == nil || d.Month == nil || d.Day == nil {
		return false
	}
	m := *d.Month
	day := *d.Day
	if m < 1 || m > 12 || day < 1 || day > 31 {
		return false
	}
	// Validate real calendar: April 31 and Feb 30 normalize silently in
	// time.Date; reject them here so the user gets feedback instead of a
	// silently-wrong reminder date.
	check := time.Date(2000, time.Month(m), day, 0, 0, 0, 0, time.UTC)
	if check.Month() != time.Month(m) {
		return false
	}
	return true
}

// nextRemindAt computes the next yearly reminder timestamp from month/day,
// using the given location for timezone-aware date boundary calculation.
func nextRemindAt(month, day int, now time.Time, loc *time.Location) time.Time {
	d := time.Date(now.Year(), time.Month(month), day, defaultReminderHour, 0, 0, 0, loc)
	if !d.After(now.In(loc)) {
		d = d.AddDate(1, 0, 0)
	}
	return d
}

// syncLifeEventReminder keeps a materialised Reminder row in sync with the
// LifeEvent's Remind flag. Hard-deletes any existing reminder for this event
// (the row is machine-synthesized, not user-authored — no soft-delete), then
// creates a new yearly one if Remind is true and the event has a valid
// month/day date.
func syncLifeEventReminder(tx *gorm.DB, userID uint, event *models.LifeEvent, now time.Time, loc *time.Location) error {
	// Hard-delete: machine-synthesized rows are join-like, not user-authored.
	// Prevents soft-delete accumulation on every toggle/edit of Remind.
	if err := tx.Unscoped().Where("life_event_id = ?", event.ID).Delete(&models.Reminder{}).Error; err != nil {
		return err
	}

	if !event.Remind {
		return nil
	}
	if !eventHasMonthDay(event.Date) {
		return nil
	}

	contactID, err := getContactIDByVCardUID(tx, userID, event.EntityID)
	if err != nil {
		return err
	}
	if contactID == nil {
		return nil
	}

	month := *event.Date.Month
	day := *event.Date.Day
	remindAt := nextRemindAt(month, day, now, loc)

	contactName := event.EntityID // fallback: show VCardUID in message if name lookup fails
	var contact models.Contact
	if tx.Where("vcard_uid = ? AND user_id = ?", event.EntityID, userID).First(&contact).Error == nil {
		contactName = contact.Firstname + " " + contact.Lastname
	}

	reminder := models.Reminder{
		UserID:      userID,
		Message:     fmt.Sprintf("Life event — %s: %s", event.Type, contactName),
		RemindAt:    remindAt,
		Recurrence:  "yearly",
		ContactID:   contactID,
		LifeEventID: &event.ID,
	}
	if err := tx.Create(&reminder).Error; err != nil {
		return err
	}
	return nil
}

// CreateLifeEvent creates a new LifeEvent (life_event.go) for the
// authenticated user. Runs inside a transaction so the event and its
// materialised reminder stay in sync.
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

	cfg := currentConfig(c)
	loc := cfg.GetReminderLocation()
	now := time.Now().In(loc)

	var event models.LifeEvent
	txErr := db.Transaction(func(tx *gorm.DB) error {
		event = models.LifeEvent{
			UserID:           userID,
			EntityID:         input.EntityID,
			Type:             input.Type,
			Date:             input.Date,
			Description:      input.Description,
			Source:           input.Source,
			RelatedEntityIDs: input.RelatedEntityIDs,
			Remind:           input.Remind,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		return syncLifeEventReminder(tx, userID, &event, now, loc)
	})
	if txErr != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save life event").WithError(txErr))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Life event created successfully", "life_event": event})
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

// UpdateLifeEvent updates a LifeEvent inside a transaction so the event write
// and its materialised reminder stay in sync.
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

	cfg := currentConfig(c)
	loc := cfg.GetReminderLocation()
	now := time.Now().In(loc)

	event.EntityID = input.EntityID
	event.Type = input.Type
	event.Date = input.Date
	event.Description = input.Description
	event.Source = input.Source
	event.RelatedEntityIDs = input.RelatedEntityIDs
	event.Remind = input.Remind

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&event).Error; err != nil {
			return err
		}
		return syncLifeEventReminder(tx, userID, &event, now, loc)
	})
	if txErr != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save life event").WithError(txErr))
		return
	}

	c.JSON(http.StatusOK, event)
}

// DeleteLifeEvent deletes a LifeEvent inside a transaction. The materialised
// reminder is hard-deleted (machine-synthesized row); the event itself is
// soft-deleted (user-authored content).
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

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("life_event_id = ?", event.ID).Delete(&models.Reminder{}).Error; err != nil {
			return err
		}
		return tx.Delete(&event).Error
	})
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete life event").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Life event deleted"})
}
