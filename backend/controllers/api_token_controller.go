package controllers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"strconv"
	"time"

	apperrors "meerkat/errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListApiTokens(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var tokens []models.ApiToken
	if err := db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tokens).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		return
	}

	response := make([]models.ApiTokenResponse, len(tokens))
	for i, t := range tokens {
		response[i] = models.ApiTokenResponse{
			ID:         t.ID,
			Name:       t.Name,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: t.LastUsedAt,
			RevokedAt:  t.RevokedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"tokens": response})
}

func CreateApiToken(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	input, appErr := middleware.GetValidated[models.ApiTokenInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	// Generate 32 random bytes → base64url → prepend "meerkat_"
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInternal("token generation failed"))
		return
	}
	plaintext := "meerkat_" + base64.RawURLEncoding.EncodeToString(rawBytes)

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))

	token := models.ApiToken{
		UserID:    userID,
		Name:      input.Name,
		TokenHash: hash,
	}
	if err := db.Create(&token).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("insert"))
		return
	}

	c.JSON(http.StatusCreated, models.ApiTokenCreateResponse{
		ApiTokenResponse: models.ApiTokenResponse{
			ID:         token.ID,
			Name:       token.Name,
			CreatedAt:  token.CreatedAt,
			LastUsedAt: token.LastUsedAt,
			RevokedAt:  token.RevokedAt,
		},
		Token: plaintext,
	})
}

func RevokeApiToken(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("id", "must be a positive integer"))
		return
	}

	var token models.ApiToken
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&token).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("API token"))
		return
	}

	now := time.Now()
	if err := db.Model(&token).Update("revoked_at", now).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("update"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token revoked successfully"})
}
