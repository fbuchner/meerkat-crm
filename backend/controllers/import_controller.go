package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"meerkat/carddav"
	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Import session storage (simple in-memory for single-server deployment)
var (
	importSessions     = make(map[string]*importSessionData)
	importSessionsLock sync.RWMutex
)

type importSessionData struct {
	session     models.ImportSession
	rows        [][]string                // CSV rows (nil for VCF imports)
	importType  string                    // "csv" or "vcf"
	vcfContacts []services.VCFContactData // VCF parsed contacts (nil for CSV imports)
}

const sessionExpiry = 15 * time.Minute

// cleanupExpiredSessions removes expired import sessions
func cleanupExpiredSessions() {
	importSessionsLock.Lock()
	defer importSessionsLock.Unlock()

	now := time.Now()
	for id, data := range importSessions {
		if now.After(data.session.ExpiresAt) {
			delete(importSessions, id)
		}
	}
}

// generateSessionID creates a random session ID
func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getSession retrieves and validates an import session
func getSession(sessionID string, userID uint) (*importSessionData, *apperrors.AppError) {
	importSessionsLock.RLock()
	sessionData, exists := importSessions[sessionID]
	importSessionsLock.RUnlock()

	if !exists {
		return nil, apperrors.ErrNotFound("Import session expired or not found")
	}

	if sessionData.session.UserID != userID {
		return nil, apperrors.ErrUnauthorized("Session does not belong to current user")
	}

	if time.Now().After(sessionData.session.ExpiresAt) {
		importSessionsLock.Lock()
		delete(importSessions, sessionID)
		importSessionsLock.Unlock()
		return nil, apperrors.ErrNotFound("Import session expired")
	}

	return sessionData, nil
}

// UploadCSVForImport handles CSV file upload and returns headers with suggested mappings
func UploadCSVForImport(c *gin.Context) {
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	go cleanupExpiredSessions()

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		log.Warn().Err(err).Msg("No file uploaded")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "No file uploaded"))
		return
	}

	if file.Size > services.MaxCSVSize {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", fmt.Sprintf("File too large. Maximum size is %d MB", services.MaxCSVSize/(1024*1024))))
		return
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "File must be a CSV file"))
		return
	}

	f, err := file.Open()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open uploaded file")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to process file"))
		return
	}
	defer f.Close()

	// Parse CSV using service
	headers, dataRows, err := services.ParseCSV(f)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse CSV")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", err.Error()))
		return
	}

	// Generate session
	sessionID := generateSessionID()
	now := time.Now()

	session := models.ImportSession{
		ID:        sessionID,
		UserID:    userID,
		Headers:   headers,
		Rows:      dataRows,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionExpiry),
	}

	importSessionsLock.Lock()
	importSessions[sessionID] = &importSessionData{
		session:    session,
		rows:       dataRows,
		importType: "csv",
	}
	importSessionsLock.Unlock()

	// Get sample data for preview
	sampleData := make([][]string, 0, services.SampleRows)
	for i := 0; i < len(dataRows) && i < services.SampleRows; i++ {
		sampleData = append(sampleData, dataRows[i])
	}

	// Suggest column mappings using service
	suggestedMappings := services.SuggestColumnMappings(headers)

	log.Info().
		Str("session_id", sessionID).
		Int("headers", len(headers)).
		Int("rows", len(dataRows)).
		Msg("CSV uploaded successfully")

	c.JSON(http.StatusOK, models.ImportUploadResponse{
		SessionID:         sessionID,
		Headers:           headers,
		SuggestedMappings: suggestedMappings,
		RowCount:          len(dataRows),
		SampleData:        sampleData,
	})
}

// UploadVCFForImport handles VCF file upload and returns preview directly (no mapping needed)
func UploadVCFForImport(c *gin.Context, cfg *config.Config) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	go cleanupExpiredSessions()

	file, err := c.FormFile("file")
	if err != nil {
		log.Warn().Err(err).Msg("No file uploaded")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "No file uploaded"))
		return
	}

	if file.Size > services.MaxVCFSize {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", fmt.Sprintf("File too large. Maximum size is %d MB", services.MaxVCFSize/(1024*1024))))
		return
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".vcf") {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "File must be a VCF file"))
		return
	}

	f, err := file.Open()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open uploaded file")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to process file"))
		return
	}
	defer f.Close()

	// Parse VCF using service
	vcfContacts, previews, stats, err := services.ParseVCF(f, db, userID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse VCF")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", err.Error()))
		return
	}

	// Generate session
	sessionID := generateSessionID()
	now := time.Now()

	session := models.ImportSession{
		ID:            sessionID,
		UserID:        userID,
		CreatedAt:     now,
		ExpiresAt:     now.Add(sessionExpiry),
		PreviewRows:   previews,
		PreviewCached: true,
	}

	importSessionsLock.Lock()
	importSessions[sessionID] = &importSessionData{
		session:     session,
		importType:  "vcf",
		vcfContacts: vcfContacts,
	}
	importSessionsLock.Unlock()

	log.Info().
		Str("session_id", sessionID).
		Int("contacts", len(vcfContacts)).
		Int("valid", stats.ValidCount).
		Int("duplicates", stats.DuplicateCount).
		Int("errors", stats.ErrorCount).
		Msg("VCF uploaded and parsed successfully")

	c.JSON(http.StatusOK, models.ImportPreviewResponse{
		SessionID:      sessionID,
		Rows:           previews,
		TotalRows:      len(previews),
		ValidRows:      stats.ValidCount,
		DuplicateCount: stats.DuplicateCount,
		ErrorCount:     stats.ErrorCount,
	})
}

// PreviewImport applies mappings and returns preview with duplicate detection
func PreviewImport(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	request, err := middleware.GetValidated[models.ImportPreviewRequest](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	sessionData, err := getSession(request.SessionID, userID)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	// Generate preview using service
	previews, stats := services.GenerateCSVPreview(db, userID, sessionData.rows, sessionData.session.Headers, request.Mappings)

	// Cache preview data in session for confirm step
	importSessionsLock.Lock()
	if sd, exists := importSessions[request.SessionID]; exists {
		sd.session.Mappings = request.Mappings
		sd.session.PreviewRows = previews
		sd.session.PreviewCached = true
	}
	importSessionsLock.Unlock()

	log.Info().
		Str("session_id", request.SessionID).
		Int("total", len(previews)).
		Int("valid", stats.ValidCount).
		Int("duplicates", stats.DuplicateCount).
		Int("errors", stats.ErrorCount).
		Msg("Import preview generated")

	c.JSON(http.StatusOK, models.ImportPreviewResponse{
		SessionID:      request.SessionID,
		Rows:           previews,
		TotalRows:      len(previews),
		ValidRows:      stats.ValidCount,
		DuplicateCount: stats.DuplicateCount,
		ErrorCount:     stats.ErrorCount,
	})
}

// ConfirmImport executes the import with user-specified actions per row
func ConfirmImport(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	request, validationErr := middleware.GetValidated[models.ImportConfirmRequest](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
		return
	}

	sessionData, err := getSession(request.SessionID, userID)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	if !sessionData.session.PreviewCached {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("session", "Please generate a preview first"))
		return
	}

	// Build action map
	actionMap := make(map[int]string)
	for _, action := range request.Actions {
		actionMap[action.RowIndex] = action.Action
	}

	result := models.ImportResult{Errors: []string{}}
	isVCFImport := sessionData.importType == "vcf"

	txErr := db.Transaction(func(tx *gorm.DB) error {
		for _, preview := range sessionData.session.PreviewRows {
			action := actionMap[preview.RowIndex]
			if action == "" {
				action = "skip"
			}

			result.TotalProcessed++

			switch action {
			case "skip":
				result.Skipped++

			case "add":
				var contact models.Contact
				if isVCFImport {
					contact = *sessionData.vcfContacts[preview.RowIndex].Contact
					contact.UserID = userID
				} else {
					contact = services.BuildContactFromParsed(userID, preview.ParsedContact)
				}

				if err := tx.Create(&contact).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to create contact: %v", preview.RowIndex+1, err))
					result.Skipped++
				} else {
					result.Created++
					if isVCFImport {
						sessionData.vcfContacts[preview.RowIndex].Contact.ID = contact.ID
					}
				}

			case "update":
				if preview.DuplicateMatch == nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Cannot update - no existing contact found", preview.RowIndex+1))
					result.Skipped++
					continue
				}

				var existing models.Contact
				if err := tx.First(&existing, preview.DuplicateMatch.ExistingContactID).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to fetch existing contact: %v", preview.RowIndex+1, err))
					result.Skipped++
					continue
				}

				importType := "CSV"
				if isVCFImport {
					importType = "VCF"
				}
				if err := services.CreateMergeNote(tx, userID, existing.ID, &existing, preview.ParsedContact, importType); err != nil {
					log.Warn().Err(err).Uint("contact_id", existing.ID).Msg("Failed to create merge note")
				}

				if isVCFImport {
					services.UpdateContactFromVCF(&existing, sessionData.vcfContacts[preview.RowIndex].Contact)
				} else {
					services.UpdateContactFromParsed(&existing, preview.ParsedContact)
				}

				if err := tx.Save(&existing).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to update contact: %v", preview.RowIndex+1, err))
					result.Skipped++
				} else {
					result.Updated++
					if isVCFImport {
						sessionData.vcfContacts[preview.RowIndex].Contact.ID = existing.ID
					}
				}
			}
		}

		return nil
	})

	if txErr != nil {
		log.Error().Err(txErr).Msg("Import transaction failed")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Import failed").WithError(txErr))
		return
	}

	// Clean up session
	importSessionsLock.Lock()
	delete(importSessions, request.SessionID)
	importSessionsLock.Unlock()

	log.Info().
		Str("session_id", request.SessionID).
		Str("import_type", sessionData.importType).
		Int("created", result.Created).
		Int("updated", result.Updated).
		Int("skipped", result.Skipped).
		Int("errors", len(result.Errors)).
		Msg("Import completed")

	c.JSON(http.StatusOK, result)
}

// ConfirmVCFImport executes VCF import with photo processing
func ConfirmVCFImport(c *gin.Context, cfg *config.Config) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	request, validationErr := middleware.GetValidated[models.ImportConfirmRequest](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
		return
	}

	sessionData, err := getSession(request.SessionID, userID)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	if sessionData.importType != "vcf" {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("session", "This endpoint is only for VCF imports"))
		return
	}

	if !sessionData.session.PreviewCached {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("session", "Please generate a preview first"))
		return
	}

	actionMap := make(map[int]string)
	for _, action := range request.Actions {
		actionMap[action.RowIndex] = action.Action
	}

	result := models.ImportResult{Errors: []string{}}

	type photoTask struct {
		contactID      uint
		photoData      []byte
		photoMediaType string
		photoURL       string // URL to fetch photo from (if not embedded)
	}
	var photoTasks []photoTask

	txErr := db.Transaction(func(tx *gorm.DB) error {
		for _, preview := range sessionData.session.PreviewRows {
			action := actionMap[preview.RowIndex]
			if action == "" {
				action = "skip"
			}

			result.TotalProcessed++
			vcfData := sessionData.vcfContacts[preview.RowIndex]

			switch action {
			case "skip":
				result.Skipped++

			case "add":
				contact := *vcfData.Contact
				contact.UserID = userID

				if err := tx.Create(&contact).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to create contact: %v", preview.RowIndex+1, err))
					result.Skipped++
				} else {
					result.Created++
					// Queue photo processing (either embedded data or URL)
					if len(vcfData.PhotoData) > 0 || vcfData.PhotoURL != "" {
						photoTasks = append(photoTasks, photoTask{
							contactID:      contact.ID,
							photoData:      vcfData.PhotoData,
							photoMediaType: vcfData.PhotoMediaType,
							photoURL:       vcfData.PhotoURL,
						})
					}
				}

			case "update":
				if preview.DuplicateMatch == nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Cannot update - no existing contact found", preview.RowIndex+1))
					result.Skipped++
					continue
				}

				var existing models.Contact
				if err := tx.First(&existing, preview.DuplicateMatch.ExistingContactID).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to fetch existing contact: %v", preview.RowIndex+1, err))
					result.Skipped++
					continue
				}

				if err := services.CreateMergeNote(tx, userID, existing.ID, &existing, preview.ParsedContact, "VCF"); err != nil {
					log.Warn().Err(err).Uint("contact_id", existing.ID).Msg("Failed to create merge note")
				}

				services.UpdateContactFromVCF(&existing, vcfData.Contact)

				if err := tx.Save(&existing).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to update contact: %v", preview.RowIndex+1, err))
					result.Skipped++
				} else {
					result.Updated++
					// Queue photo processing only if contact doesn't already have a photo
					if existing.Photo == "" && (len(vcfData.PhotoData) > 0 || vcfData.PhotoURL != "") {
						photoTasks = append(photoTasks, photoTask{
							contactID:      existing.ID,
							photoData:      vcfData.PhotoData,
							photoMediaType: vcfData.PhotoMediaType,
							photoURL:       vcfData.PhotoURL,
						})
					}
				}
			}
		}

		return nil
	})

	if txErr != nil {
		log.Error().Err(txErr).Msg("VCF import transaction failed")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Import failed").WithError(txErr))
		return
	}

	// Process photos outside transaction (file I/O and network requests)
	for _, task := range photoTasks {
		var photoData []byte
		var mediaType string

		if len(task.photoData) > 0 {
			// Use embedded photo data
			photoData = task.photoData
			mediaType = task.photoMediaType
		} else if task.photoURL != "" {
			// Fetch photo from URL
			var err error
			photoData, mediaType, err = carddav.FetchPhotoFromURL(task.photoURL)
			if err != nil {
				log.Warn().Err(err).Uint("contact_id", task.contactID).Str("photo_url", task.photoURL).Msg("Failed to fetch photo from URL")
				continue
			}
		}

		if len(photoData) == 0 {
			continue
		}

		photoPath, thumbnailData, err := carddav.SaveContactPhoto(photoData, mediaType, cfg.ProfilePhotoDir)
		if err != nil {
			log.Warn().Err(err).Uint("contact_id", task.contactID).Msg("Failed to save imported photo")
			continue
		}

		if err := db.Model(&models.Contact{}).Where("id = ?", task.contactID).Updates(map[string]interface{}{
			"photo":           photoPath,
			"photo_thumbnail": thumbnailData,
		}).Error; err != nil {
			log.Warn().Err(err).Uint("contact_id", task.contactID).Msg("Failed to update contact with photo")
		}
	}

	// Clean up session
	importSessionsLock.Lock()
	delete(importSessions, request.SessionID)
	importSessionsLock.Unlock()

	log.Info().
		Str("session_id", request.SessionID).
		Int("created", result.Created).
		Int("updated", result.Updated).
		Int("skipped", result.Skipped).
		Int("photos_processed", len(photoTasks)).
		Int("errors", len(result.Errors)).
		Msg("VCF import completed")

	c.JSON(http.StatusOK, result)
}
