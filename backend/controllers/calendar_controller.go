package controllers

import (
	"errors"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/logger"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"mycorrhizal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxCalendarSubscriptionsPerUser = 10

func ListCalendarSubscriptions(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var subscriptions []models.CalendarSubscription
	if err := db.Where("user_id = ?", userID).Order("created_at ASC").Find(&subscriptions).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"calendars": toCalendarResponses(subscriptions)})
}

func CreateCalendarSubscription(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var count int64
	if err := db.Model(&models.CalendarSubscription{}).Where("user_id = ? AND deleted_at IS NULL", userID).Count(&count).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("count"))
		return
	}
	if count >= maxCalendarSubscriptionsPerUser {
		apperrors.AbortWithError(c, apperrors.ErrConflict("maximum of 10 calendar subscriptions per user reached"))
		return
	}

	input, appErr := middleware.GetValidated[models.CalendarSubscriptionInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	if _, err := services.NormalizeCalendarURL(input.URL); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("url", "must be an http, https, or webcal URL"))
		return
	}

	passwordEncrypted, err := services.EncryptCredential(currentConfig(c).JWTSecretKey, input.Password)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInternal("failed to store credentials"))
		return
	}

	subscription := models.CalendarSubscription{
		UserID:            userID,
		Name:              input.Name,
		URL:               input.URL,
		Username:          input.Username,
		PasswordEncrypted: passwordEncrypted,
		SyncEnabled:       input.SyncEnabled == nil || *input.SyncEnabled,
		PastDays:          5,
		FutureDays:        10,
	}
	if input.PastDays != nil {
		subscription.PastDays = *input.PastDays
	}
	if input.FutureDays != nil {
		subscription.FutureDays = *input.FutureDays
	}

	if err := db.Create(&subscription).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("insert"))
		return
	}

	c.JSON(http.StatusCreated, toCalendarResponse(subscription))
}

func UpdateCalendarSubscription(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	subscription, found := findCalendarSubscription(c, db, userID)
	if !found {
		return
	}

	input, appErr := middleware.GetValidated[models.CalendarSubscriptionInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	if _, err := services.NormalizeCalendarURL(input.URL); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("url", "must be an http, https, or webcal URL"))
		return
	}

	subscription.Name = input.Name
	subscription.URL = input.URL
	subscription.Username = input.Username
	if input.SyncEnabled != nil {
		subscription.SyncEnabled = *input.SyncEnabled
	}
	if input.PastDays != nil {
		subscription.PastDays = *input.PastDays
	}
	if input.FutureDays != nil {
		subscription.FutureDays = *input.FutureDays
	}

	if input.ClearPassword {
		subscription.PasswordEncrypted = ""
	} else if input.Password != "" {
		passwordEncrypted, err := services.EncryptCredential(currentConfig(c).JWTSecretKey, input.Password)
		if err != nil {
			apperrors.AbortWithError(c, apperrors.ErrInternal("failed to store credentials"))
			return
		}
		subscription.PasswordEncrypted = passwordEncrypted
	}

	if err := db.Save(&subscription).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("update"))
		return
	}

	c.JSON(http.StatusOK, toCalendarResponse(subscription))
}

func DeleteCalendarSubscription(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	subscription, found := findCalendarSubscription(c, db, userID)
	if !found {
		return
	}

	// Imported activities stay; only the subscription and its event links go.
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subscription_id = ?", subscription.ID).Delete(&models.CalendarEventLink{}).Error; err != nil {
			return err
		}
		return tx.Delete(&subscription).Error
	}); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("delete"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Calendar subscription deleted"})
}

func SyncCalendarSubscription(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	subscription, found := findCalendarSubscription(c, db, userID)
	if !found {
		return
	}

	cfg := currentConfig(c)
	service := services.NewCalendarSyncService(cfg.CalDAVBlockPrivateURLs)
	stats, err := service.SyncSubscription(c.Request.Context(), db, cfg, &subscription)
	if err != nil {
		logger.FromContext(c).Warn().Err(err).Uint("subscription_id", subscription.ID).Msg("Manual calendar sync failed")
		apperrors.AbortWithError(c, calendarSyncError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Calendar sync completed",
		"created": stats.Created,
		"updated": stats.Updated,
		"skipped": stats.Skipped,
	})
}

// calendarSyncError maps the sync service's sentinel errors to API errors.
func calendarSyncError(err error) *apperrors.AppError {
	switch {
	case errors.Is(err, services.ErrCalendarInvalidURL):
		return apperrors.ErrInvalidInput("url", "calendar URL is invalid")
	case errors.Is(err, services.ErrCalendarUnauthorized):
		return apperrors.ErrExternal("Calendar", "authentication failed - check username and password")
	case errors.Is(err, services.ErrCalendarNotFound):
		return apperrors.ErrExternal("Calendar", "calendar not found at the given URL")
	case errors.Is(err, services.ErrCalendarPrivateAddress):
		return apperrors.ErrExternal("Calendar", "calendar URL resolves to a blocked private address")
	case errors.Is(err, services.ErrCalendarTooLarge):
		return apperrors.ErrExternal("Calendar", "calendar response is too large")
	case errors.Is(err, services.ErrCalendarInvalidData):
		return apperrors.ErrExternal("Calendar", "calendar returned data that could not be parsed")
	case errors.Is(err, services.ErrCalendarUnreachable):
		return apperrors.ErrExternal("Calendar", "calendar could not be reached")
	default:
		return apperrors.ErrOperationFailed("Calendar sync", err.Error())
	}
}

func findCalendarSubscription(c *gin.Context, db *gorm.DB, userID uint) (models.CalendarSubscription, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("id", "must be a positive integer"))
		return models.CalendarSubscription{}, false
	}

	var subscription models.CalendarSubscription
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&subscription).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Calendar subscription"))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		}
		return models.CalendarSubscription{}, false
	}
	return subscription, true
}

func toCalendarResponse(sub models.CalendarSubscription) models.CalendarSubscriptionResponse {
	return models.CalendarSubscriptionResponse{
		ID:             sub.ID,
		Name:           sub.Name,
		URL:            sub.URL,
		Username:       sub.Username,
		HasPassword:    sub.PasswordEncrypted != "",
		SyncEnabled:    sub.SyncEnabled,
		PastDays:       sub.PastDays,
		FutureDays:     sub.FutureDays,
		LastSyncedAt:   sub.LastSyncedAt,
		LastSyncStatus: sub.LastSyncStatus,
		LastSyncError:  sub.LastSyncError,
		CreatedAt:      sub.CreatedAt,
	}
}

func toCalendarResponses(subs []models.CalendarSubscription) []models.CalendarSubscriptionResponse {
	out := make([]models.CalendarSubscriptionResponse, len(subs))
	for i, sub := range subs {
		out[i] = toCalendarResponse(sub)
	}
	return out
}
