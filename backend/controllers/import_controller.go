package controllers

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"regexp"
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
	session models.ImportSession
	rows    [][]string
}

const (
	maxCSVSize    = 5 * 1024 * 1024 // 5MB
	maxCSVRows    = 1000
	sessionExpiry = 15 * time.Minute
	sampleRows    = 3 // Number of sample rows to return
)

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

// UploadCSVForImport handles CSV file upload and returns headers with suggested mappings
func UploadCSVForImport(c *gin.Context) {
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Cleanup expired sessions periodically
	go cleanupExpiredSessions()

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		log.Warn().Err(err).Msg("No file uploaded")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "No file uploaded"))
		return
	}

	// Check file size
	if file.Size > maxCSVSize {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", fmt.Sprintf("File too large. Maximum size is %d MB", maxCSVSize/(1024*1024))))
		return
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "File must be a CSV file"))
		return
	}

	// Open file
	f, err := file.Open()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open uploaded file")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to process file"))
		return
	}
	defer f.Close()

	// Parse CSV
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // Allow variable number of fields
	reader.LazyQuotes = true    // Be lenient with quotes

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse CSV")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "Invalid CSV format: "+err.Error()))
		return
	}

	if len(records) == 0 {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "CSV file is empty"))
		return
	}

	// First row is headers
	headers := records[0]
	if len(headers) == 0 {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "CSV file has no headers"))
		return
	}

	// Data rows (skip header)
	dataRows := records[1:]
	if len(dataRows) == 0 {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "CSV file has no data rows"))
		return
	}

	if len(dataRows) > maxCSVRows {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", fmt.Sprintf("Too many rows. Maximum is %d rows", maxCSVRows)))
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

	// Store session
	importSessionsLock.Lock()
	importSessions[sessionID] = &importSessionData{
		session: session,
		rows:    dataRows,
	}
	importSessionsLock.Unlock()

	// Get sample data for preview
	sampleData := make([][]string, 0, sampleRows)
	for i := 0; i < len(dataRows) && i < sampleRows; i++ {
		sampleData = append(sampleData, dataRows[i])
	}

	// Suggest column mappings
	suggestedMappings := suggestColumnMappings(headers)

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

// suggestColumnMappings guesses mappings based on CSV header names
func suggestColumnMappings(headers []string) []models.ColumnMapping {
	mappings := make([]models.ColumnMapping, len(headers))

	// Mapping rules (case-insensitive, supports common variations)
	headerToField := map[string]string{
		// English
		"firstname": "firstname", "first name": "firstname", "first": "firstname", "given name": "firstname",
		"lastname": "lastname", "last name": "lastname", "last": "lastname", "surname": "lastname", "family name": "lastname",
		"nickname": "nickname", "nick": "nickname", "alias": "nickname",
		"email": "email", "e-mail": "email", "mail": "email", "email address": "email",
		"phone": "phone", "telephone": "phone", "tel": "phone", "mobile": "phone", "cell": "phone", "phone number": "phone",
		"birthday": "birthday", "birth date": "birthday", "birthdate": "birthday", "dob": "birthday", "date of birth": "birthday",
		"address": "address", "street address": "address", "home address": "address",
		"gender": "gender", "sex": "gender",
		"how we met": "how_we_met", "how_we_met": "how_we_met", "notes": "how_we_met", "how i met": "how_we_met",
		"food": "food_preference", "food preference": "food_preference", "food_preference": "food_preference", "dietary": "food_preference", "diet": "food_preference",
		"work": "work_information", "work_information": "work_information", "job": "work_information", "company": "work_information", "occupation": "work_information", "employer": "work_information",
		"contact information": "contact_information", "contact_information": "contact_information", "other contact": "contact_information",
		"circles": "circles", "groups": "circles", "tags": "circles", "category": "circles", "categories": "circles",
		// German
		"vorname": "firstname",
		"nachname": "lastname", "familienname": "lastname",
		"spitzname": "nickname",
		"telefon": "phone", "handy": "phone", "mobiltelefon": "phone",
		"geburtstag": "birthday", "geburtsdatum": "birthday",
		"adresse": "address", "anschrift": "address",
		"geschlecht": "gender",
		"beruf": "work_information", "arbeit": "work_information", "firma": "work_information",
		"kreise": "circles", "gruppen": "circles",
	}

	for i, header := range headers {
		normalized := strings.ToLower(strings.TrimSpace(header))
		if field, ok := headerToField[normalized]; ok {
			mappings[i] = models.ColumnMapping{CSVColumn: header, ContactField: field}
		} else {
			mappings[i] = models.ColumnMapping{CSVColumn: header, ContactField: ""} // Unmapped
		}
	}

	return mappings
}

// PreviewImport applies mappings and returns preview with duplicate detection
func PreviewImport(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get validated input
	request, err := middleware.GetValidated[models.ImportPreviewRequest](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	// Get session
	importSessionsLock.RLock()
	sessionData, exists := importSessions[request.SessionID]
	importSessionsLock.RUnlock()

	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Import session expired or not found"))
		return
	}

	// Verify session belongs to user
	if sessionData.session.UserID != userID {
		apperrors.AbortWithError(c, apperrors.ErrUnauthorized("Session does not belong to current user"))
		return
	}

	// Check session expiry
	if time.Now().After(sessionData.session.ExpiresAt) {
		importSessionsLock.Lock()
		delete(importSessions, request.SessionID)
		importSessionsLock.Unlock()
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Import session expired"))
		return
	}

	// Build column index map from mappings
	columnIndex := make(map[string]int)
	for i, header := range sessionData.session.Headers {
		columnIndex[header] = i
	}

	fieldToColumnIndex := make(map[string]int)
	for _, mapping := range request.Mappings {
		if mapping.ContactField != "" {
			if idx, ok := columnIndex[mapping.CSVColumn]; ok {
				fieldToColumnIndex[mapping.ContactField] = idx
			}
		}
	}

	// Process each row
	var previews []models.ImportRowPreview
	var validCount, duplicateCount, errorCount int

	for rowIdx, row := range sessionData.rows {
		preview := models.ImportRowPreview{
			RowIndex:         rowIdx,
			ParsedContact:    make(map[string]interface{}),
			ValidationErrors: make([]string, 0), // Initialize as empty slice, not nil
			SuggestedAction:  "add",
		}

		// Parse fields from row
		for field, colIdx := range fieldToColumnIndex {
			if colIdx < len(row) {
				value := strings.TrimSpace(row[colIdx])
				if value != "" {
					preview.ParsedContact[field] = value
				}
			}
		}

		// Get key fields for validation and duplicate detection
		firstname := getStringField(preview.ParsedContact, "firstname")
		lastname := getStringField(preview.ParsedContact, "lastname")
		email := getStringField(preview.ParsedContact, "email")

		// Validate row
		validationErrors := validateImportRow(preview.ParsedContact)
		preview.ValidationErrors = validationErrors

		if len(validationErrors) > 0 {
			errorCount++
			preview.SuggestedAction = "skip"
		} else {
			validCount++

			// Detect duplicates
			duplicate := detectDuplicate(db, userID, firstname, lastname, email)
			if duplicate != nil {
				preview.DuplicateMatch = duplicate
				preview.SuggestedAction = "update"
				duplicateCount++
			}
		}

		previews = append(previews, preview)
	}

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
		Int("valid", validCount).
		Int("duplicates", duplicateCount).
		Int("errors", errorCount).
		Msg("Import preview generated")

	c.JSON(http.StatusOK, models.ImportPreviewResponse{
		SessionID:      request.SessionID,
		Rows:           previews,
		TotalRows:      len(previews),
		ValidRows:      validCount,
		DuplicateCount: duplicateCount,
		ErrorCount:     errorCount,
	})
}

// getStringField safely gets a string field from parsed contact
func getStringField(parsed map[string]interface{}, field string) string {
	if val, ok := parsed[field]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// validateImportRow validates a parsed row and returns errors
func validateImportRow(row map[string]interface{}) []string {
	errors := make([]string, 0) // Initialize as empty slice, not nil

	// Firstname is required
	firstname := getStringField(row, "firstname")
	if firstname == "" {
		errors = append(errors, "First name is required")
	}

	// Email format validation
	if email := getStringField(row, "email"); email != "" {
		if !middleware.ValidateEmail(email) {
			errors = append(errors, "Invalid email format")
		}
	}

	// Birthday format validation (YYYY-MM-DD or --MM-DD) - normalize first
	if birthday := getStringField(row, "birthday"); birthday != "" {
		normalized := normalizeBirthday(birthday)
		if !isValidBirthdayFormat(normalized) {
			errors = append(errors, "Invalid birthday format (expected YYYY-MM-DD or --MM-DD)")
		}
	}

	// Gender validation
	if gender := getStringField(row, "gender"); gender != "" {
		normalized := normalizeGender(gender)
		if normalized == "" {
			errors = append(errors, "Invalid gender value")
		}
	}

	// Phone validation
	if phone := getStringField(row, "phone"); phone != "" {
		if !isValidPhone(phone) {
			errors = append(errors, "Invalid phone format")
		}
	}

	return errors
}

// isValidBirthdayFormat checks birthday format (YYYY-MM-DD or --MM-DD)
func isValidBirthdayFormat(birthday string) bool {
	match, _ := regexp.MatchString(`^(--|\d{4}-)\d{2}-\d{2}$`, birthday)
	return match
}

// normalizeBirthday converts various birthday formats to the app's ISO format (YYYY-MM-DD or --MM-DD)
// Supported input formats:
// - YYYY-MM-DD (ISO format with year, e.g., "1958-06-29") - native format
// - --MM-DD (ISO format without year, e.g., "--04-20") - native format
// - DD.MM.YYYY (legacy format with year, e.g., "29.06.1958")
// - DD.MM. (legacy format without year, e.g., "29.06.")
func normalizeBirthday(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// Already in ISO format with year: YYYY-MM-DD - return as-is
	if match, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, input); match {
		return input
	}

	// Already in ISO format without year: --MM-DD - return as-is
	if match, _ := regexp.MatchString(`^--\d{2}-\d{2}$`, input); match {
		return input
	}

	// Legacy format with year: DD.MM.YYYY -> YYYY-MM-DD
	if match, _ := regexp.MatchString(`^\d{2}\.\d{2}\.\d{4}$`, input); match {
		day := input[0:2]
		month := input[3:5]
		year := input[6:10]
		return year + "-" + month + "-" + day
	}

	// Legacy format without year: DD.MM. -> --MM-DD
	if match, _ := regexp.MatchString(`^\d{2}\.\d{2}\.$`, input); match {
		day := input[0:2]
		month := input[3:5]
		return "--" + month + "-" + day
	}

	// Unknown format - return as-is (will fail validation)
	return input
}

// isValidPhone validates phone number format
func isValidPhone(phone string) bool {
	// Remove common formatting characters
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, phone)

	// Must have between 5 and 20 digits
	return len(cleaned) >= 5 && len(cleaned) <= 20
}

// normalizeGender converts various gender inputs to valid enum values
func normalizeGender(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	switch lower {
	case "m", "male", "mann", "maennlich", "männlich", "masculin":
		return "male"
	case "f", "female", "frau", "weiblich", "feminin", "w":
		return "female"
	case "o", "other", "andere", "divers", "d":
		return "other"
	case "prefer not to say", "prefer_not_to_say", "keine angabe":
		return "prefer_not_to_say"
	default:
		return ""
	}
}

// detectDuplicate checks for existing contacts matching the parsed row
func detectDuplicate(db *gorm.DB, userID uint, firstname, lastname, email string) *models.DuplicateMatch {
	var existing models.Contact

	// Priority 1: Email match (if email provided)
	if email != "" {
		if err := db.Where("user_id = ? AND LOWER(email) = LOWER(?)", userID, email).First(&existing).Error; err == nil {
			return &models.DuplicateMatch{
				ExistingContactID: existing.ID,
				ExistingFirstname: existing.Firstname,
				ExistingLastname:  existing.Lastname,
				ExistingEmail:     existing.Email,
				MatchReason:       "email",
			}
		}
	}

	// Priority 2: Name match (firstname + lastname)
	if firstname != "" && lastname != "" {
		if err := db.Where("user_id = ? AND LOWER(firstname) = LOWER(?) AND LOWER(lastname) = LOWER(?)",
			userID, firstname, lastname).First(&existing).Error; err == nil {
			return &models.DuplicateMatch{
				ExistingContactID: existing.ID,
				ExistingFirstname: existing.Firstname,
				ExistingLastname:  existing.Lastname,
				ExistingEmail:     existing.Email,
				MatchReason:       "name",
			}
		}
	}

	return nil
}

// parseCircles parses circles from comma or semicolon separated string
func parseCircles(input string) []string {
	if input == "" {
		return nil
	}

	// Support both comma and semicolon as separators
	var circles []string
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ';'
	})

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			circles = append(circles, trimmed)
		}
	}

	return circles
}

// ConfirmImport executes the import with user-specified actions per row
func ConfirmImport(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get validated input
	request, validationErr := middleware.GetValidated[models.ImportConfirmRequest](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
		return
	}

	// Get session
	importSessionsLock.RLock()
	sessionData, exists := importSessions[request.SessionID]
	importSessionsLock.RUnlock()

	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrNotFound("Import session expired or not found"))
		return
	}

	// Verify session belongs to user
	if sessionData.session.UserID != userID {
		apperrors.AbortWithError(c, apperrors.ErrUnauthorized("Session does not belong to current user"))
		return
	}

	// Check if preview was generated
	if !sessionData.session.PreviewCached {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("session", "Please generate a preview first"))
		return
	}

	// Build action map
	actionMap := make(map[int]string)
	for _, action := range request.Actions {
		actionMap[action.RowIndex] = action.Action
	}

	// Process import within a transaction
	result := models.ImportResult{
		Errors: []string{},
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, preview := range sessionData.session.PreviewRows {
			action, exists := actionMap[preview.RowIndex]
			if !exists {
				action = "skip" // Default to skip if not specified
			}

			result.TotalProcessed++

			switch action {
			case "skip":
				result.Skipped++

			case "add":
				// Create new contact
				contact := buildContactFromParsed(userID, preview.ParsedContact)
				if err := tx.Create(&contact).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to create contact: %v", preview.RowIndex+1, err))
					result.Skipped++
				} else {
					result.Created++
				}

			case "update":
				if preview.DuplicateMatch == nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Cannot update - no existing contact found", preview.RowIndex+1))
					result.Skipped++
					continue
				}

				// Fetch existing contact
				var existing models.Contact
				if err := tx.First(&existing, preview.DuplicateMatch.ExistingContactID).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to fetch existing contact: %v", preview.RowIndex+1, err))
					result.Skipped++
					continue
				}

				// Create note with old values before updating
				if err := createMergeNote(tx, userID, existing.ID, &existing, preview.ParsedContact); err != nil {
					log.Warn().Err(err).Uint("contact_id", existing.ID).Msg("Failed to create merge note")
					// Continue with update even if note creation fails
				}

				// Update contact fields
				updateContactFromParsed(&existing, preview.ParsedContact)
				if err := tx.Save(&existing).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to update contact: %v", preview.RowIndex+1, err))
					result.Skipped++
				} else {
					result.Updated++
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Import transaction failed")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Import failed").WithError(err))
		return
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
		Int("errors", len(result.Errors)).
		Msg("Import completed")

	c.JSON(http.StatusOK, result)
}

// buildContactFromParsed creates a new Contact from parsed import data
func buildContactFromParsed(userID uint, parsed map[string]interface{}) models.Contact {
	contact := models.Contact{
		UserID: userID,
	}

	if v := getStringField(parsed, "firstname"); v != "" {
		contact.Firstname = v
	}
	if v := getStringField(parsed, "lastname"); v != "" {
		contact.Lastname = v
	}
	if v := getStringField(parsed, "nickname"); v != "" {
		contact.Nickname = v
	}
	if v := getStringField(parsed, "email"); v != "" {
		contact.Email = v
	}
	if v := getStringField(parsed, "phone"); v != "" {
		contact.Phone = v
	}
	if v := getStringField(parsed, "birthday"); v != "" {
		contact.Birthday = normalizeBirthday(v)
	}
	if v := getStringField(parsed, "address"); v != "" {
		contact.Address = v
	}
	if v := getStringField(parsed, "gender"); v != "" {
		contact.Gender = normalizeGender(v)
	}
	if v := getStringField(parsed, "how_we_met"); v != "" {
		contact.HowWeMet = v
	}
	if v := getStringField(parsed, "food_preference"); v != "" {
		contact.FoodPreference = v
	}
	if v := getStringField(parsed, "work_information"); v != "" {
		contact.WorkInformation = v
	}
	if v := getStringField(parsed, "contact_information"); v != "" {
		contact.ContactInformation = v
	}
	if v := getStringField(parsed, "circles"); v != "" {
		contact.Circles = parseCircles(v)
	}

	return contact
}

// updateContactFromParsed updates an existing contact with parsed import data
func updateContactFromParsed(contact *models.Contact, parsed map[string]interface{}) {
	// Only update fields that have values in the import
	if v := getStringField(parsed, "firstname"); v != "" {
		contact.Firstname = v
	}
	if v := getStringField(parsed, "lastname"); v != "" {
		contact.Lastname = v
	}
	if v := getStringField(parsed, "nickname"); v != "" {
		contact.Nickname = v
	}
	if v := getStringField(parsed, "email"); v != "" {
		contact.Email = v
	}
	if v := getStringField(parsed, "phone"); v != "" {
		contact.Phone = v
	}
	if v := getStringField(parsed, "birthday"); v != "" {
		contact.Birthday = normalizeBirthday(v)
	}
	if v := getStringField(parsed, "address"); v != "" {
		contact.Address = v
	}
	if v := getStringField(parsed, "gender"); v != "" {
		contact.Gender = normalizeGender(v)
	}
	if v := getStringField(parsed, "how_we_met"); v != "" {
		contact.HowWeMet = v
	}
	if v := getStringField(parsed, "food_preference"); v != "" {
		contact.FoodPreference = v
	}
	if v := getStringField(parsed, "work_information"); v != "" {
		contact.WorkInformation = v
	}
	if v := getStringField(parsed, "contact_information"); v != "" {
		contact.ContactInformation = v
	}
	if v := getStringField(parsed, "circles"); v != "" {
		contact.Circles = parseCircles(v)
	}
}

// createMergeNote creates a note documenting what was changed during import
func createMergeNote(db *gorm.DB, userID uint, contactID uint, original *models.Contact, newValues map[string]interface{}) error {
	var changes []string

	fieldLabels := map[string]struct {
		label    string
		original string
	}{
		"firstname":          {"First Name", original.Firstname},
		"lastname":           {"Last Name", original.Lastname},
		"nickname":           {"Nickname", original.Nickname},
		"email":              {"Email", original.Email},
		"phone":              {"Phone", original.Phone},
		"birthday":           {"Birthday", original.Birthday},
		"address":            {"Address", original.Address},
		"gender":             {"Gender", original.Gender},
		"how_we_met":         {"How We Met", original.HowWeMet},
		"food_preference":    {"Food Preferences", original.FoodPreference},
		"work_information":   {"Work Information", original.WorkInformation},
		"contact_information": {"Contact Information", original.ContactInformation},
	}

	// Compare each field that has a new value
	for field, info := range fieldLabels {
		newVal := getStringField(newValues, field)
		if newVal != "" && info.original != newVal {
			if info.original != "" {
				changes = append(changes, fmt.Sprintf("- %s: %s → %s", info.label, info.original, newVal))
			} else {
				changes = append(changes, fmt.Sprintf("- %s: (empty) → %s", info.label, newVal))
			}
		}
	}

	// Handle circles separately
	if newCirclesStr := getStringField(newValues, "circles"); newCirclesStr != "" {
		oldCircles := strings.Join(original.Circles, ", ")
		if oldCircles != newCirclesStr {
			if oldCircles != "" {
				changes = append(changes, fmt.Sprintf("- Circles: %s → %s", oldCircles, newCirclesStr))
			} else {
				changes = append(changes, fmt.Sprintf("- Circles: (empty) → %s", newCirclesStr))
			}
		}
	}

	if len(changes) == 0 {
		return nil // No actual changes
	}

	content := "CSV Import updated this contact.\n\nChanges made:\n" + strings.Join(changes, "\n")

	note := models.Note{
		UserID:    userID,
		ContactID: &contactID,
		Content:   content,
		Date:      time.Now(),
	}

	return db.Create(&note).Error
}
