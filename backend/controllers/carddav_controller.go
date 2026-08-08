package controllers

import (
	"encoding/json"
	"errors"
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

// returns the user's CardDAV connection, or 404 when none is configured
// (the frontend treats that as the "not connected" state).
func GetCardDAVConnection(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var conn models.CardDAVConnection
	if err := db.Where("user_id = ?", userID).First(&conn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("CardDAV connection"))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		}
		return
	}

	c.JSON(http.StatusOK, toCardDAVConnectionResponse(conn))
}

// SaveCardDAVConnection creates or replaces the user's single CardDAV connection.
func SaveCardDAVConnection(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	input, appErr := middleware.GetValidated[models.CardDAVConnectionInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	// Store the normalized form (drops any inline credentials)
	normalized, err := services.NormalizeCardDAVURL(input.BaseURL)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("base_url", "must be an http or https URL"))
		return
	}
	baseURL := normalized.String()
	if !strings.HasPrefix(input.AddressBookPath, "/") {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("address_book_path", "must be an absolute path"))
		return
	}

	var conn models.CardDAVConnection
	lookupErr := db.Where("user_id = ?", userID).First(&conn).Error
	if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		return
	}
	isNew := errors.Is(lookupErr, gorm.ErrRecordNotFound)

	// Link state is scoped to one address book: switching servers or books
	// invalidates it, so it is wiped and rebuilt on the next sync (contacts
	// re-pair losslessly by UID).
	addressBookChanged := !isNew && (conn.BaseURL != baseURL || conn.AddressBookPath != input.AddressBookPath)

	conn.UserID = userID
	conn.BaseURL = baseURL
	conn.AddressBookPath = input.AddressBookPath
	conn.AddressBookName = input.AddressBookName
	conn.Username = input.Username
	if input.Direction != "" {
		conn.Direction = input.Direction
	} else if conn.Direction == "" {
		conn.Direction = models.CardDAVDirectionTwoWay
	}
	if input.SyncEnabled != nil {
		conn.SyncEnabled = *input.SyncEnabled
	} else if isNew {
		conn.SyncEnabled = true
	}

	if input.ClearPassword {
		conn.PasswordEncrypted = ""
	} else if input.Password != "" {
		passwordEncrypted, err := services.EncryptCredential(currentConfig(c).JWTSecretKey, input.Password)
		if err != nil {
			apperrors.AbortWithError(c, apperrors.ErrInternal("failed to store credentials"))
			return
		}
		conn.PasswordEncrypted = passwordEncrypted
	}
	if addressBookChanged {
		conn.SyncToken = ""
		conn.LastSyncedAt = nil
		conn.LastSyncStatus = ""
		conn.LastSyncError = ""
		conn.LastSyncStats = ""
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if addressBookChanged {
			if err := tx.Where("connection_id = ?", conn.ID).Delete(&models.CardDAVContactLink{}).Error; err != nil {
				return err
			}
		}
		return tx.Save(&conn).Error
	}); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("save"))
		return
	}

	status := http.StatusOK
	if isNew {
		status = http.StatusCreated
	}
	c.JSON(status, toCardDAVConnectionResponse(conn))
}

// removes the connection and its link state. Contacts are kept untouched.
func DeleteCardDAVConnection(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	conn, found := findCardDAVConnection(c, db, userID)
	if !found {
		return
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("connection_id = ?", conn.ID).Delete(&models.CardDAVContactLink{}).Error; err != nil {
			return err
		}
		return tx.Delete(&conn).Error
	}); err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("delete"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "CardDAV connection deleted"})
}

// lists the address books on a CardDAV server. With empty password the stored credentials are reused
// so re-discovery doesn't require retyping them — but only when the requested URL points at the
// same host they were stored for, so they cannot be walked over to another server by supplying a different base_url.
func DiscoverCardDAVAddressBooks(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	input, appErr := middleware.GetValidated[models.CardDAVDiscoverInput](c)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	cfg := currentConfig(c)
	normalized, err := services.NormalizeCardDAVURL(input.BaseURL)
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("base_url", "must be an http or https URL"))
		return
	}

	password := input.Password
	if password == "" {
		// Reuse the stored password only for the host it was stored for, so
		// re-discovering against a different server cannot carry it along.
		var conn models.CardDAVConnection
		if db.Where("user_id = ?", userID).First(&conn).Error == nil {
			storedURL, urlErr := services.NormalizeCardDAVURL(conn.BaseURL)
			if urlErr == nil && strings.EqualFold(storedURL.Hostname(), normalized.Hostname()) {
				if stored, decErr := services.DecryptCredential(cfg.JWTSecretKey, conn.PasswordEncrypted); decErr == nil {
					password = stored
				}
			}
		}
	}

	service := services.NewContactSyncService(cfg.CardDAVBlockPrivateURLs)
	books, err := service.Discover(c.Request.Context(), normalized.String(), input.Username, password)
	if err != nil {
		logger.FromContext(c).Warn().Err(err).Msg("CardDAV discovery failed")
		apperrors.AbortWithError(c, contactSyncError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"address_books": books})
}

// SyncCardDAVConnection starts a sync and returns immediately. A first sync of
// a large address book runs for minutes, far longer than an HTTP request should
// be held open, so the outcome is polled from GET /carddav/connection
// (sync_running, then last_sync_status and last_sync_stats) instead of awaited.
func SyncCardDAVConnection(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	conn, found := findCardDAVConnection(c, db, userID)
	if !found {
		return
	}

	if !services.StartContactSync(db, currentConfig(c), conn) {
		apperrors.AbortWithError(c, apperrors.ErrConflict("a contact sync is already running for this connection"))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Contact sync started"})
}

// contactSyncError maps the sync service's sentinel errors to API errors.
func contactSyncError(err error) *apperrors.AppError {
	switch {
	case errors.Is(err, services.ErrContactSyncInvalidURL):
		return apperrors.ErrInvalidInput("base_url", "CardDAV URL is invalid")
	case errors.Is(err, services.ErrContactSyncUnauthorized):
		return apperrors.ErrExternal("CardDAV", "authentication failed - check username and password")
	case errors.Is(err, services.ErrContactSyncForbidden):
		return apperrors.ErrExternal("CardDAV", "the server refused the request - check permissions")
	case errors.Is(err, services.ErrContactSyncNotFound):
		return apperrors.ErrExternal("CardDAV", "address book not found at the given URL")
	case errors.Is(err, services.ErrContactSyncPrivateAddress):
		return apperrors.ErrExternal("CardDAV", "URL resolves to a blocked private address")
	case errors.Is(err, services.ErrContactSyncTooLarge):
		return apperrors.ErrExternal("CardDAV", "server response is too large")
	case errors.Is(err, services.ErrContactSyncTooMany):
		return apperrors.ErrExternal("CardDAV", "address book has too many contacts to sync")
	case errors.Is(err, services.ErrContactSyncNoAddressBooks):
		return apperrors.ErrExternal("CardDAV", "no address books found on the server")
	case errors.Is(err, services.ErrContactSyncInvalidData):
		return apperrors.ErrExternal("CardDAV", "server returned data that could not be parsed")
	case errors.Is(err, services.ErrContactSyncUnreachable):
		return apperrors.ErrExternal("CardDAV", "server could not be reached")
	default:
		// Every caller logs the concrete error with a request ID. Internal
		// detail (database errors, internal hostnames) must not reach the client.
		return apperrors.ErrOperationFailed("Contact sync", "an unexpected error occurred")
	}
}

func findCardDAVConnection(c *gin.Context, db *gorm.DB, userID uint) (models.CardDAVConnection, bool) {
	var conn models.CardDAVConnection
	if err := db.Where("user_id = ?", userID).First(&conn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("CardDAV connection"))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("query"))
		}
		return models.CardDAVConnection{}, false
	}
	return conn, true
}

func toCardDAVConnectionResponse(conn models.CardDAVConnection) models.CardDAVConnectionResponse {
	// Passed through verbatim rather than re-encoded: the stats type lives in
	// services, which models cannot import. Validated so a malformed column can
	// never corrupt the response body.
	var stats json.RawMessage
	if conn.LastSyncStats != "" && json.Valid([]byte(conn.LastSyncStats)) {
		stats = json.RawMessage(conn.LastSyncStats)
	}

	return models.CardDAVConnectionResponse{
		ID:              conn.ID,
		BaseURL:         conn.BaseURL,
		AddressBookPath: conn.AddressBookPath,
		AddressBookName: conn.AddressBookName,
		Username:        conn.Username,
		HasPassword:     conn.PasswordEncrypted != "",
		Direction:       conn.Direction,
		SyncEnabled:     conn.SyncEnabled,
		SyncRunning:     services.IsContactSyncRunning(conn.ID),
		LastSyncedAt:    conn.LastSyncedAt,
		LastSyncStatus:  conn.LastSyncStatus,
		LastSyncError:   conn.LastSyncError,
		LastSyncStats:   stats,
		CreatedAt:       conn.CreatedAt,
	}
}
