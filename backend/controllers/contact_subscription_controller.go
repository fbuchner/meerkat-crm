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

// maxContactSubscriptionsPerUser mirrors maxCalendarSubscriptionsPerUser
// (calendar_controller.go)'s cap for the same reason: bound how many
// external servers one user's account can fan out sync requests to.
const maxContactSubscriptionsPerUser = 10

func ListContactSubscriptions(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var subscriptions []models.ContactSubscription
	if err := db.Where("user_id = ?", userID).Order("created_at ASC").Find(&subscriptions).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"contact_subscriptions": toContactSubscriptionResponses(subscriptions)})
}

func CreateContactSubscription(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var count int64
	if err := db.Model(&models.ContactSubscription{}).Where("user_id = ? AND deleted_at IS NULL", userID).Count(&count).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("count"))
		return
	}
	if count >= maxContactSubscriptionsPerUser {
		apperrors.AbortWithError(c, apperrors.ErrConflict("maximum of 10 contact subscriptions per user reached"))
		return
	}

	input, appErr := middleware.GetValidated[models.ContactSubscriptionInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	if _, err := services.NormalizeContactSubscriptionURL(input.URL); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("url", "must be an http or https URL"))
		return
	}

	passwordEncrypted, err := services.EncryptCredential(currentConfig(c).JWTSecretKey, input.Password)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInternal("failed to store credentials"))
		return
	}

	subscription := models.ContactSubscription{
		UserID:            userID,
		Name:              input.Name,
		URL:               input.URL,
		Username:          input.Username,
		PasswordEncrypted: passwordEncrypted,
		SyncEnabled:       input.SyncEnabled == nil || *input.SyncEnabled,
	}

	if err := db.Create(&subscription).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("insert"))
		return
	}

	c.JSON(http.StatusCreated, toContactSubscriptionResponse(subscription))
}

func UpdateContactSubscription(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	subscription, found := findContactSubscription(c, db, userID)
	if !found {
		return
	}

	input, appErr := middleware.GetValidated[models.ContactSubscriptionInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	if _, err := services.NormalizeContactSubscriptionURL(input.URL); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("url", "must be an http or https URL"))
		return
	}

	// The subscription's URL changed identity in CardDAV terms, so a
	// previously stored sync-token (scoped to the old collection) would no
	// longer be valid; reset it so the next sync performs a fresh full sync
	// via sync-collection's own initial-sync behavior (empty token).
	if subscription.URL != input.URL {
		subscription.SyncToken = ""
	}

	subscription.Name = input.Name
	subscription.URL = input.URL
	subscription.Username = input.Username
	if input.SyncEnabled != nil {
		subscription.SyncEnabled = *input.SyncEnabled
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

	c.JSON(http.StatusOK, toContactSubscriptionResponse(subscription))
}

func DeleteContactSubscription(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	subscription, found := findContactSubscription(c, db, userID)
	if !found {
		return
	}

	// Synced contacts stay (they are real, user-owned Contact rows per
	// WP-73b's decision); only the subscription and its sync links go.
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subscription_id = ?", subscription.ID).Delete(&models.ContactSyncLink{}).Error; err != nil {
			return err
		}
		return tx.Delete(&subscription).Error
	}); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("delete"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact subscription deleted"})
}

func SyncContactSubscription(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	subscription, found := findContactSubscription(c, db, userID)
	if !found {
		return
	}

	cfg := currentConfig(c)
	service := services.NewContactSyncService(cfg.CalDAVBlockPrivateURLs)
	stats, err := service.SyncSubscription(c.Request.Context(), db, cfg, &subscription)
	if err != nil {
		logger.FromContext(c).Warn().Err(err).Uint("subscription_id", subscription.ID).Msg("Manual contact sync failed")
		apperrors.AbortWithError(c, contactSyncError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Contact sync completed",
		"created":  stats.Created,
		"updated":  stats.Updated,
		"archived": stats.Archived,
		"skipped":  stats.Skipped,
	})
}

// contactSyncError maps the sync service's sentinel errors to API errors.
func contactSyncError(err error) *apperrors.AppError {
	switch {
	case errors.Is(err, services.ErrContactSyncInvalidURL):
		return apperrors.ErrInvalidInput("url", "contact subscription URL is invalid")
	case errors.Is(err, services.ErrContactSyncUnauthorized):
		return apperrors.ErrExternal("CardDAV", "authentication failed - check username and password")
	case errors.Is(err, services.ErrContactSyncNotFound):
		return apperrors.ErrExternal("CardDAV", "address book not found at the given URL")
	case errors.Is(err, services.ErrContactSyncPrivateAddress):
		return apperrors.ErrExternal("CardDAV", "URL resolves to a blocked private address")
	case errors.Is(err, services.ErrContactSyncTooLarge):
		return apperrors.ErrExternal("CardDAV", "server response is too large")
	case errors.Is(err, services.ErrContactSyncInvalidData):
		return apperrors.ErrExternal("CardDAV", "server returned data that could not be parsed")
	case errors.Is(err, services.ErrContactSyncUnreachable):
		return apperrors.ErrExternal("CardDAV", "server could not be reached")
	default:
		return apperrors.ErrOperationFailed("Contact sync", err.Error())
	}
}

func findContactSubscription(c *gin.Context, db *gorm.DB, userID uint) (models.ContactSubscription, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("id", "must be a positive integer"))
		return models.ContactSubscription{}, false
	}

	var subscription models.ContactSubscription
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&subscription).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact subscription"))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		}
		return models.ContactSubscription{}, false
	}
	return subscription, true
}

func toContactSubscriptionResponse(sub models.ContactSubscription) models.ContactSubscriptionResponse {
	return models.ContactSubscriptionResponse{
		ID:             sub.ID,
		Name:           sub.Name,
		URL:            sub.URL,
		Username:       sub.Username,
		HasPassword:    sub.PasswordEncrypted != "",
		SyncEnabled:    sub.SyncEnabled,
		LastSyncedAt:   sub.LastSyncedAt,
		LastSyncStatus: sub.LastSyncStatus,
		LastSyncError:  sub.LastSyncError,
		CreatedAt:      sub.CreatedAt,
	}
}

func toContactSubscriptionResponses(subs []models.ContactSubscription) []models.ContactSubscriptionResponse {
	out := make([]models.ContactSubscriptionResponse, len(subs))
	for i, sub := range subs {
		out[i] = toContactSubscriptionResponse(sub)
	}
	return out
}
