package controllers

import (
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SyncCalDAVActivities(c *gin.Context) {
	input, err := middleware.GetValidated[models.CalDAVSyncInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	result, syncErr := services.NewCalDAVSyncService().Sync(c.Request.Context(), db, userID, *input)
	if syncErr != nil {
		logger.FromContext(c).Warn().Err(syncErr).Msg("CalDAV activity sync failed")
		apperrors.AbortWithError(c, calDAVSyncError(syncErr))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "CalDAV sync completed",
		"created": result.Created,
		"skipped": result.Skipped,
	})
}

func calDAVSyncError(err error) *apperrors.AppError {
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid calendar URL"):
		return apperrors.ErrInvalidInput("url", message)
	case strings.Contains(message, "one or more contacts"):
		return apperrors.ErrNotFound("One or more contacts")
	case strings.Contains(message, "calendar service"):
		return apperrors.ErrExternal("CalDAV", message)
	default:
		return apperrors.ErrOperationFailed("CalDAV sync", message)
	}
}
