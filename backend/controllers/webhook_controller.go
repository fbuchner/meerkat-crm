package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	apperrors "meerkat/errors"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListWebhooks(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var webhooks []models.Webhook
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&webhooks).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhooks": toWebhookResponses(webhooks)})
}

const maxWebhooksPerUser = 20

func CreateWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var count int64
	if err := db.Model(&models.Webhook{}).Where("user_id = ? AND deleted_at IS NULL", userID).Count(&count).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("count"))
		return
	}
	if count >= maxWebhooksPerUser {
		apperrors.AbortWithError(c, apperrors.ErrConflict("maximum of 20 webhooks per user reached"))
		return
	}

	input, appErr := middleware.GetValidated[models.WebhookInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	secret, err := generateSecret()
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInternal("secret generation failed"))
		return
	}

	wh := models.Webhook{
		UserID:   userID,
		Name:     input.Name,
		URL:      input.URL,
		Events:   input.Events,
		Secret:   secret,
		IsActive: input.IsActive,
	}
	if err := db.Create(&wh).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("insert"))
		return
	}

	c.JSON(http.StatusCreated, models.WebhookCreateResponse{
		ID:        wh.ID,
		Name:      wh.Name,
		URL:       wh.URL,
		Events:    wh.Events,
		IsActive:  wh.IsActive,
		CreatedAt: wh.CreatedAt,
		Secret:    secret,
	})
}

func GetWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	wh, found := findWebhook(c, db, userID)
	if !found {
		return
	}

	c.JSON(http.StatusOK, toWebhookResponse(wh))
}

func UpdateWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	wh, found := findWebhook(c, db, userID)
	if !found {
		return
	}

	input, appErr := middleware.GetValidated[models.WebhookInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	wh.Name = input.Name
	wh.URL = input.URL
	wh.Events = input.Events
	wh.IsActive = input.IsActive

	if err := db.Save(&wh).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("update"))
		return
	}

	c.JSON(http.StatusOK, toWebhookResponse(wh))
}

func DeleteWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	wh, found := findWebhook(c, db, userID)
	if !found {
		return
	}

	if err := db.Delete(&wh).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("delete"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted"})
}

func TestWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	wh, found := findWebhook(c, db, userID)
	if !found {
		return
	}

	delivery := services.TestWebhookDelivery(db, currentConfig(c), wh)
	c.JSON(http.StatusOK, gin.H{"delivery": toDeliveryResponse(delivery)})
}

func GetWebhookDeliveries(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	wh, found := findWebhook(c, db, userID)
	if !found {
		return
	}

	var deliveries []models.WebhookDelivery
	if err := db.Where("webhook_id = ?", wh.ID).Order("created_at DESC").Limit(50).Find(&deliveries).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"deliveries": toDeliveryResponses(deliveries)})
}

func findWebhook(c *gin.Context, db *gorm.DB, userID uint) (models.Webhook, bool) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("id", "must be a positive integer"))
		return models.Webhook{}, false
	}

	var wh models.Webhook
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&wh).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Webhook"))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		}
		return models.Webhook{}, false
	}
	return wh, true
}

func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func toWebhookResponse(wh models.Webhook) models.WebhookResponse {
	return models.WebhookResponse{
		ID:        wh.ID,
		Name:      wh.Name,
		URL:       wh.URL,
		Events:    wh.Events,
		IsActive:  wh.IsActive,
		CreatedAt: wh.CreatedAt,
	}
}

func toWebhookResponses(whs []models.Webhook) []models.WebhookResponse {
	out := make([]models.WebhookResponse, len(whs))
	for i, wh := range whs {
		out[i] = toWebhookResponse(wh)
	}
	return out
}

func toDeliveryResponse(d models.WebhookDelivery) models.WebhookDeliveryResponse {
	return models.WebhookDeliveryResponse{
		ID:          d.ID,
		WebhookID:   d.WebhookID,
		EventType:   d.EventType,
		StatusCode:  d.StatusCode,
		Error:       d.Error,
		Attempts:    d.Attempts,
		NextRetryAt: d.NextRetryAt,
		CreatedAt:   d.CreatedAt,
	}
}

func toDeliveryResponses(ds []models.WebhookDelivery) []models.WebhookDeliveryResponse {
	out := make([]models.WebhookDeliveryResponse, len(ds))
	for i, d := range ds {
		out[i] = toDeliveryResponse(d)
	}
	return out
}
