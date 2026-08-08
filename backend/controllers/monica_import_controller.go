package controllers

import (
	"net/http"

	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// monicaImportSessions owns all in-progress Monica import wizard state,
// including the API tokens (in memory only). Controllers only validate HTTP
// input and delegate to the manager.
var monicaImportSessions = services.NewMonicaImportManager()

// ConnectMonicaImport validates the Monica URL and API token and returns the
// account's entity counts together with a new session ID.
func ConnectMonicaImport(c *gin.Context, cfg *config.Config) {
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	go monicaImportSessions.CleanupExpired()

	req, validationErr := middleware.GetValidated[models.MonicaConnectRequest](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
		return
	}

	resp, appErr := monicaImportSessions.Connect(c.Request.Context(), userID, *req, cfg.MonicaBlockPrivateURLs)
	if appErr != nil {
		log.Warn().Str("code", appErr.Code).Msg("Monica connect failed")
		apperrors.AbortWithError(c, appErr)
		return
	}

	log.Info().
		Str("session_id", resp.SessionID).
		Int("contacts", resp.Totals.Contacts).
		Msg("Monica import session connected")

	c.JSON(http.StatusOK, resp)
}

// StartMonicaFetch launches the background snapshot fetch for a session.
func StartMonicaFetch(c *gin.Context) {
	log := logger.FromContext(c)
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	req, validationErr := middleware.GetValidated[models.MonicaFetchRequest](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
		return
	}

	if appErr := monicaImportSessions.StartFetch(db, userID, *req, log); appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"session_id": req.SessionID})
}

// GetMonicaImportStatus reports fetch/import progress for polling.
func GetMonicaImportStatus(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		apperrors.AbortWithError(c, apperrors.ErrMissingField("session_id"))
		return
	}

	status, appErr := monicaImportSessions.Status(userID, sessionID)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetMonicaImportPreview returns the contact rows of a fetched snapshot.
func GetMonicaImportPreview(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		apperrors.AbortWithError(c, apperrors.ErrMissingField("session_id"))
		return
	}

	preview, appErr := monicaImportSessions.Preview(userID, sessionID)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	c.JSON(http.StatusOK, preview)
}

// ConfirmMonicaImport executes the import with the per-contact actions.
func ConfirmMonicaImport(c *gin.Context, cfg *config.Config) {
	log := logger.FromContext(c)
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	req, validationErr := middleware.GetValidated[models.MonicaConfirmRequest](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
		return
	}

	result, appErr := monicaImportSessions.Confirm(db, userID, *req, cfg, log)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CancelMonicaImport aborts a session, cancelling any running fetch and
// dropping the API token from memory.
func CancelMonicaImport(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		apperrors.AbortWithError(c, apperrors.ErrMissingField("session_id"))
		return
	}

	// Validate ownership before deleting.
	if _, appErr := monicaImportSessions.Status(userID, sessionID); appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}
	monicaImportSessions.Delete(sessionID)

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}
