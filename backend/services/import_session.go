package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"meerkat/carddav"
	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/models"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// sessionExpiry is how long an in-progress import wizard session is kept server-side.
const sessionExpiry = 15 * time.Minute

// importSessionData holds the server-side state for an in-progress import wizard.
// Sessions are kept in memory only (not persisted) and are lost on restart.
type importSessionData struct {
	session     models.ImportSession
	rows        [][]string       // CSV rows (nil for VCF imports)
	importType  string           // "csv" or "vcf"
	vcfContacts []VCFContactData // VCF parsed contacts (nil for CSV imports)
	csvContacts []models.Contact // CSV contacts built during preview (nil for VCF imports)
}

// ImportSessionManager owns the lifecycle of in-progress import sessions: creation,
// retrieval/validation, preview generation, confirmation, and expiry. It is the single
// owner of import state so controllers only need to validate input and delegate.
type ImportSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*importSessionData
}

// NewImportSessionManager creates an empty session manager.
func NewImportSessionManager() *ImportSessionManager {
	return &ImportSessionManager{sessions: make(map[string]*importSessionData)}
}

// generateSessionID creates a random session ID.
func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// CleanupExpired removes expired import sessions. Safe to call from a goroutine.
func (m *ImportSessionManager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, data := range m.sessions {
		if now.After(data.session.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
}

// get retrieves and validates an import session, enforcing ownership and expiry.
func (m *ImportSessionManager) get(sessionID string, userID uint) (*importSessionData, *apperrors.AppError) {
	m.mu.RLock()
	sessionData, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return nil, apperrors.ErrNotFound("Import session expired or not found")
	}

	if sessionData.session.UserID != userID {
		return nil, apperrors.ErrUnauthorized("Session does not belong to current user")
	}

	if time.Now().After(sessionData.session.ExpiresAt) {
		m.mu.Lock()
		delete(m.sessions, sessionID)
		m.mu.Unlock()
		return nil, apperrors.ErrNotFound("Import session expired")
	}

	return sessionData, nil
}

// Delete removes a session, typically after a completed import.
func (m *ImportSessionManager) Delete(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
}

// CreateCSVSession stores a freshly parsed CSV upload and returns its session ID.
func (m *ImportSessionManager) CreateCSVSession(userID uint, headers []string, rows [][]string) string {
	sessionID := generateSessionID()
	now := time.Now()

	m.mu.Lock()
	m.sessions[sessionID] = &importSessionData{
		session: models.ImportSession{
			ID:        sessionID,
			UserID:    userID,
			Headers:   headers,
			Rows:      rows,
			CreatedAt: now,
			ExpiresAt: now.Add(sessionExpiry),
		},
		rows:       rows,
		importType: "csv",
	}
	m.mu.Unlock()

	return sessionID
}

// CreateVCFSession stores a freshly parsed VCF upload (whose preview is computed at
// upload time) and returns its session ID.
func (m *ImportSessionManager) CreateVCFSession(userID uint, vcfContacts []VCFContactData, previews []models.ImportRowPreview) string {
	sessionID := generateSessionID()
	now := time.Now()

	m.mu.Lock()
	m.sessions[sessionID] = &importSessionData{
		session: models.ImportSession{
			ID:            sessionID,
			UserID:        userID,
			CreatedAt:     now,
			ExpiresAt:     now.Add(sessionExpiry),
			PreviewRows:   previews,
			PreviewCached: true,
		},
		importType:  "vcf",
		vcfContacts: vcfContacts,
	}
	m.mu.Unlock()

	return sessionID
}

// PreviewCSV applies mappings to a CSV session, caches the built contacts and preview
// rows for the confirm step, and returns the preview response.
func (m *ImportSessionManager) PreviewCSV(db *gorm.DB, userID uint, req models.ImportPreviewRequest) (*models.ImportPreviewResponse, *apperrors.AppError) {
	sessionData, err := m.get(req.SessionID, userID)
	if err != nil {
		return nil, err
	}

	contacts, previews, stats := GenerateCSVPreview(db, userID, sessionData.rows, sessionData.session.Headers, req.Mappings)

	// Cache preview data and built contacts in the session for the confirm step.
	m.mu.Lock()
	if sd, exists := m.sessions[req.SessionID]; exists {
		sd.session.Mappings = req.Mappings
		sd.session.PreviewRows = previews
		sd.session.PreviewCached = true
		sd.csvContacts = contacts
	}
	m.mu.Unlock()

	return &models.ImportPreviewResponse{
		SessionID:      req.SessionID,
		Rows:           previews,
		TotalRows:      len(previews),
		ValidRows:      stats.ValidCount,
		DuplicateCount: stats.DuplicateCount,
		ErrorCount:     stats.ErrorCount,
	}, nil
}

// Confirm executes a CSV or VCF import using the per-row actions in req, then deletes
// the session. Photo processing is handled separately by ConfirmVCF.
func (m *ImportSessionManager) Confirm(db *gorm.DB, userID uint, req models.ImportConfirmRequest, log *zerolog.Logger) (*models.ImportResult, *apperrors.AppError) {
	sessionData, sessErr := m.get(req.SessionID, userID)
	if sessErr != nil {
		return nil, sessErr
	}

	if !sessionData.session.PreviewCached {
		return nil, apperrors.ErrInvalidInput("session", "Please generate a preview first")
	}

	actionMap := buildActionMap(req.Actions)
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
				} else {
					contact = sessionData.csvContacts[preview.RowIndex]
				}
				contact.UserID = userID

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
				if err := CreateMergeNote(tx, userID, existing.ID, &existing, preview.ParsedContact, importType); err != nil {
					log.Warn().Err(err).Uint("contact_id", existing.ID).Msg("Failed to create merge note")
				}

				if isVCFImport {
					MergeImportedContact(&existing, sessionData.vcfContacts[preview.RowIndex].Contact)
				} else {
					csvContact := sessionData.csvContacts[preview.RowIndex]
					MergeImportedContact(&existing, &csvContact)
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
		return nil, apperrors.ErrDatabase("Import failed").WithError(txErr)
	}

	m.Delete(req.SessionID)

	log.Info().
		Str("session_id", req.SessionID).
		Str("import_type", sessionData.importType).
		Int("created", result.Created).
		Int("updated", result.Updated).
		Int("skipped", result.Skipped).
		Int("errors", len(result.Errors)).
		Msg("Import completed")

	return &result, nil
}

// photoTask queues photo processing for a contact, deferred until after the import
// transaction commits since it involves file I/O and network requests.
type photoTask struct {
	contactID      uint
	photoData      []byte
	photoMediaType string
	photoURL       string // URL to fetch photo from (if not embedded)
}

// ConfirmVCF executes a VCF import with photo processing, then deletes the session.
func (m *ImportSessionManager) ConfirmVCF(db *gorm.DB, userID uint, req models.ImportConfirmRequest, cfg *config.Config, log *zerolog.Logger) (*models.ImportResult, *apperrors.AppError) {
	sessionData, sessErr := m.get(req.SessionID, userID)
	if sessErr != nil {
		return nil, sessErr
	}

	if sessionData.importType != "vcf" {
		return nil, apperrors.ErrInvalidInput("session", "This endpoint is only for VCF imports")
	}

	if !sessionData.session.PreviewCached {
		return nil, apperrors.ErrInvalidInput("session", "Please generate a preview first")
	}

	actionMap := buildActionMap(req.Actions)
	result := models.ImportResult{Errors: []string{}}
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

				if err := CreateMergeNote(tx, userID, existing.ID, &existing, preview.ParsedContact, "VCF"); err != nil {
					log.Warn().Err(err).Uint("contact_id", existing.ID).Msg("Failed to create merge note")
				}

				MergeImportedContact(&existing, vcfData.Contact)

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
		return nil, apperrors.ErrDatabase("Import failed").WithError(txErr)
	}

	// Process photos outside the transaction (file I/O and network requests).
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

	m.Delete(req.SessionID)

	log.Info().
		Str("session_id", req.SessionID).
		Int("created", result.Created).
		Int("updated", result.Updated).
		Int("skipped", result.Skipped).
		Int("photos_processed", len(photoTasks)).
		Int("errors", len(result.Errors)).
		Msg("VCF import completed")

	return &result, nil
}

// buildActionMap indexes per-row import actions by row index.
func buildActionMap(actions []models.RowImportAction) map[int]string {
	actionMap := make(map[int]string, len(actions))
	for _, action := range actions {
		actionMap[action.RowIndex] = action.Action
	}
	return actionMap
}
