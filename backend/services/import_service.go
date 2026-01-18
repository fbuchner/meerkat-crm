package services

import (
	"encoding/csv"
	"fmt"
	"io"
	"meerkat/carddav"
	"meerkat/middleware"
	"meerkat/models"
	"regexp"
	"strings"

	"github.com/emersion/go-vcard"
	"gorm.io/gorm"
)

// Import limits
const (
	MaxCSVSize     = 5 * 1024 * 1024  // 5MB
	MaxVCFSize     = 10 * 1024 * 1024 // 10MB (VCF files can include embedded photos)
	MaxCSVRows     = 1000
	MaxVCFContacts = 1000
	SampleRows     = 3 // Number of sample rows to return
)

// VCFContactData holds parsed VCF contact data with photo for import
type VCFContactData struct {
	Contact        *models.Contact
	PhotoData      []byte
	PhotoMediaType string
}

// ParseCSV reads and parses a CSV file, returning headers and data rows
func ParseCSV(reader io.Reader) (headers []string, rows [][]string, err error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields
	csvReader.LazyQuotes = true    // Be lenient with quotes

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid CSV format: %w", err)
	}

	if len(records) == 0 {
		return nil, nil, fmt.Errorf("CSV file is empty")
	}

	headers = records[0]
	if len(headers) == 0 {
		return nil, nil, fmt.Errorf("CSV file has no headers")
	}

	rows = records[1:]
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("CSV file has no data rows")
	}

	if len(rows) > MaxCSVRows {
		return nil, nil, fmt.Errorf("too many rows: maximum is %d rows", MaxCSVRows)
	}

	return headers, rows, nil
}

// ParseVCF reads and parses a VCF file, returning contact data and previews
func ParseVCF(reader io.Reader, db *gorm.DB, userID uint) (contacts []VCFContactData, previews []models.ImportRowPreview, stats ImportStats, err error) {
	decoder := vcard.NewDecoder(reader)

	for rowIdx := 0; ; rowIdx++ {
		card, decodeErr := decoder.Decode()
		if decodeErr == io.EOF {
			break
		}
		if decodeErr != nil {
			// Skip malformed vCards but continue parsing
			previews = append(previews, models.ImportRowPreview{
				RowIndex:         rowIdx,
				ParsedContact:    make(map[string]interface{}),
				ValidationErrors: []string{fmt.Sprintf("Failed to parse vCard: %v", decodeErr)},
				SuggestedAction:  "skip",
			})
			stats.ErrorCount++
			continue
		}

		if rowIdx >= MaxVCFContacts {
			return nil, nil, stats, fmt.Errorf("too many contacts: maximum is %d contacts", MaxVCFContacts)
		}

		// Convert vCard to Contact using existing carddav mapper
		contact, photoData, photoMediaType := carddav.VCardToContact(card, nil)

		contacts = append(contacts, VCFContactData{
			Contact:        contact,
			PhotoData:      photoData,
			PhotoMediaType: photoMediaType,
		})

		// Build preview
		preview := models.ImportRowPreview{
			RowIndex:         rowIdx,
			ParsedContact:    ContactToPreviewMap(contact),
			ValidationErrors: make([]string, 0),
			SuggestedAction:  "add",
		}

		// Validate the contact
		validationErrors := ValidateVCFContact(contact)
		preview.ValidationErrors = validationErrors

		if len(validationErrors) > 0 {
			stats.ErrorCount++
			preview.SuggestedAction = "skip"
		} else {
			stats.ValidCount++

			// Detect duplicates
			duplicate := DetectDuplicate(db, userID, contact.Firstname, contact.Lastname, contact.Email)
			if duplicate != nil {
				preview.DuplicateMatch = duplicate
				preview.SuggestedAction = "update"
				stats.DuplicateCount++
			}
		}

		previews = append(previews, preview)
	}

	if len(contacts) == 0 {
		return nil, nil, stats, fmt.Errorf("VCF file contains no valid contacts")
	}

	return contacts, previews, stats, nil
}

// ImportStats holds statistics about an import operation
type ImportStats struct {
	ValidCount     int
	DuplicateCount int
	ErrorCount     int
}

// SuggestColumnMappings guesses mappings based on CSV header names
func SuggestColumnMappings(headers []string) []models.ColumnMapping {
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
		"vorname":  "firstname",
		"nachname": "lastname", "familienname": "lastname",
		"spitzname": "nickname",
		"telefon":   "phone", "handy": "phone", "mobiltelefon": "phone",
		"geburtstag": "birthday", "geburtsdatum": "birthday",
		"adresse": "address", "anschrift": "address",
		"geschlecht": "gender",
		"beruf":      "work_information", "arbeit": "work_information", "firma": "work_information",
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

// GenerateCSVPreview applies mappings to CSV rows and returns preview with duplicate detection
func GenerateCSVPreview(db *gorm.DB, userID uint, rows [][]string, headers []string, mappings []models.ColumnMapping) ([]models.ImportRowPreview, ImportStats) {
	// Build column index map from mappings
	columnIndex := make(map[string]int)
	for i, header := range headers {
		columnIndex[header] = i
	}

	fieldToColumnIndex := make(map[string]int)
	for _, mapping := range mappings {
		if mapping.ContactField != "" {
			if idx, ok := columnIndex[mapping.CSVColumn]; ok {
				fieldToColumnIndex[mapping.ContactField] = idx
			}
		}
	}

	var previews []models.ImportRowPreview
	var stats ImportStats

	for rowIdx, row := range rows {
		preview := models.ImportRowPreview{
			RowIndex:         rowIdx,
			ParsedContact:    make(map[string]interface{}),
			ValidationErrors: make([]string, 0),
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
		firstname := GetStringField(preview.ParsedContact, "firstname")
		lastname := GetStringField(preview.ParsedContact, "lastname")
		email := GetStringField(preview.ParsedContact, "email")

		// Validate row
		validationErrors := ValidateImportRow(preview.ParsedContact)
		preview.ValidationErrors = validationErrors

		if len(validationErrors) > 0 {
			stats.ErrorCount++
			preview.SuggestedAction = "skip"
		} else {
			stats.ValidCount++

			// Detect duplicates
			duplicate := DetectDuplicate(db, userID, firstname, lastname, email)
			if duplicate != nil {
				preview.DuplicateMatch = duplicate
				preview.SuggestedAction = "update"
				stats.DuplicateCount++
			}
		}

		previews = append(previews, preview)
	}

	return previews, stats
}

// ContactToPreviewMap converts a Contact to a preview map for display
func ContactToPreviewMap(contact *models.Contact) map[string]interface{} {
	preview := make(map[string]interface{})
	if contact.Firstname != "" {
		preview["firstname"] = contact.Firstname
	}
	if contact.Lastname != "" {
		preview["lastname"] = contact.Lastname
	}
	if contact.Nickname != "" {
		preview["nickname"] = contact.Nickname
	}
	if contact.Email != "" {
		preview["email"] = contact.Email
	}
	if contact.Phone != "" {
		preview["phone"] = contact.Phone
	}
	if contact.Birthday != "" {
		preview["birthday"] = contact.Birthday
	}
	if contact.Address != "" {
		preview["address"] = contact.Address
	}
	if contact.Gender != "" {
		preview["gender"] = contact.Gender
	}
	if contact.WorkInformation != "" {
		preview["work_information"] = contact.WorkInformation
	}
	if len(contact.Circles) > 0 {
		preview["circles"] = strings.Join(contact.Circles, ", ")
	}
	return preview
}

// ValidateVCFContact validates a contact parsed from VCF
func ValidateVCFContact(contact *models.Contact) []string {
	errors := make([]string, 0)

	if contact.Firstname == "" {
		errors = append(errors, "First name is required")
	}

	if contact.Email != "" && !middleware.ValidateEmail(contact.Email) {
		errors = append(errors, "Invalid email format")
	}

	if contact.Birthday != "" && !IsValidBirthdayFormat(contact.Birthday) {
		errors = append(errors, "Invalid birthday format")
	}

	return errors
}

// ValidateImportRow validates a parsed row and returns errors
func ValidateImportRow(row map[string]interface{}) []string {
	errors := make([]string, 0)

	// Firstname is required
	firstname := GetStringField(row, "firstname")
	if firstname == "" {
		errors = append(errors, "First name is required")
	}

	// Email format validation
	if email := GetStringField(row, "email"); email != "" {
		if !middleware.ValidateEmail(email) {
			errors = append(errors, "Invalid email format")
		}
	}

	// Birthday format validation (YYYY-MM-DD or --MM-DD) - normalize first
	if birthday := GetStringField(row, "birthday"); birthday != "" {
		normalized := NormalizeBirthday(birthday)
		if !IsValidBirthdayFormat(normalized) {
			errors = append(errors, "Invalid birthday format (expected YYYY-MM-DD or --MM-DD)")
		}
	}

	// Gender validation
	if gender := GetStringField(row, "gender"); gender != "" {
		normalized := NormalizeGender(gender)
		if normalized == "" {
			errors = append(errors, "Invalid gender value")
		}
	}

	// Phone validation
	if phone := GetStringField(row, "phone"); phone != "" {
		if !IsValidPhone(phone) {
			errors = append(errors, "Invalid phone format")
		}
	}

	return errors
}

// GetStringField safely gets a string field from parsed contact
func GetStringField(parsed map[string]interface{}, field string) string {
	if val, ok := parsed[field]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// IsValidBirthdayFormat checks birthday format (YYYY-MM-DD or --MM-DD)
func IsValidBirthdayFormat(birthday string) bool {
	match, _ := regexp.MatchString(`^(--|\d{4}-)\d{2}-\d{2}$`, birthday)
	return match
}

// NormalizeBirthday converts various birthday formats to the app's ISO format (YYYY-MM-DD or --MM-DD)
// Supported input formats:
// - YYYY-MM-DD (ISO format with year, e.g., "1958-06-29") - native format
// - --MM-DD (ISO format without year, e.g., "--04-20") - native format
// - DD.MM.YYYY (legacy format with year, e.g., "29.06.1958")
// - DD.MM. (legacy format without year, e.g., "29.06.")
func NormalizeBirthday(input string) string {
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

// IsValidPhone validates phone number format
func IsValidPhone(phone string) bool {
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

// NormalizeGender converts various gender inputs to valid enum values
func NormalizeGender(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	switch lower {
	case "m", "male", "mann", "maennlich", "männlich", "masculin":
		return "male"
	case "f", "female", "frau", "weiblich", "feminin", "w":
		return "female"
	case "o", "other", "andere", "divers", "d":
		return "other"
	case "prefer not to say", "prefer_not_to_say", "keine angabe":
		return "other"
	default:
		return ""
	}
}

// DetectDuplicate checks for existing contacts matching the given fields
func DetectDuplicate(db *gorm.DB, userID uint, firstname, lastname, email string) *models.DuplicateMatch {
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

// ParseCircles parses circles from comma or semicolon separated string
func ParseCircles(input string) []string {
	if input == "" {
		return nil
	}

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

// BuildContactFromParsed creates a new Contact from parsed import data
func BuildContactFromParsed(userID uint, parsed map[string]interface{}) models.Contact {
	contact := models.Contact{
		UserID: userID,
	}

	if v := GetStringField(parsed, "firstname"); v != "" {
		contact.Firstname = v
	}
	if v := GetStringField(parsed, "lastname"); v != "" {
		contact.Lastname = v
	}
	if v := GetStringField(parsed, "nickname"); v != "" {
		contact.Nickname = v
	}
	if v := GetStringField(parsed, "email"); v != "" {
		contact.Email = v
	}
	if v := GetStringField(parsed, "phone"); v != "" {
		contact.Phone = v
	}
	if v := GetStringField(parsed, "birthday"); v != "" {
		contact.Birthday = NormalizeBirthday(v)
	}
	if v := GetStringField(parsed, "address"); v != "" {
		contact.Address = v
	}
	if v := GetStringField(parsed, "gender"); v != "" {
		contact.Gender = NormalizeGender(v)
	}
	if v := GetStringField(parsed, "how_we_met"); v != "" {
		contact.HowWeMet = v
	}
	if v := GetStringField(parsed, "food_preference"); v != "" {
		contact.FoodPreference = v
	}
	if v := GetStringField(parsed, "work_information"); v != "" {
		contact.WorkInformation = v
	}
	if v := GetStringField(parsed, "contact_information"); v != "" {
		contact.ContactInformation = v
	}
	if v := GetStringField(parsed, "circles"); v != "" {
		contact.Circles = ParseCircles(v)
	}

	return contact
}

// UpdateContactFromParsed updates an existing contact with parsed import data
func UpdateContactFromParsed(contact *models.Contact, parsed map[string]interface{}) {
	if v := GetStringField(parsed, "firstname"); v != "" {
		contact.Firstname = v
	}
	if v := GetStringField(parsed, "lastname"); v != "" {
		contact.Lastname = v
	}
	if v := GetStringField(parsed, "nickname"); v != "" {
		contact.Nickname = v
	}
	if v := GetStringField(parsed, "email"); v != "" {
		contact.Email = v
	}
	if v := GetStringField(parsed, "phone"); v != "" {
		contact.Phone = v
	}
	if v := GetStringField(parsed, "birthday"); v != "" {
		contact.Birthday = NormalizeBirthday(v)
	}
	if v := GetStringField(parsed, "address"); v != "" {
		contact.Address = v
	}
	if v := GetStringField(parsed, "gender"); v != "" {
		contact.Gender = NormalizeGender(v)
	}
	if v := GetStringField(parsed, "how_we_met"); v != "" {
		contact.HowWeMet = v
	}
	if v := GetStringField(parsed, "food_preference"); v != "" {
		contact.FoodPreference = v
	}
	if v := GetStringField(parsed, "work_information"); v != "" {
		contact.WorkInformation = v
	}
	if v := GetStringField(parsed, "contact_information"); v != "" {
		contact.ContactInformation = v
	}
	if v := GetStringField(parsed, "circles"); v != "" {
		contact.Circles = ParseCircles(v)
	}
}

// UpdateContactFromVCF updates an existing contact with VCF contact data
func UpdateContactFromVCF(existing *models.Contact, vcf *models.Contact) {
	if vcf.Firstname != "" {
		existing.Firstname = vcf.Firstname
	}
	if vcf.Lastname != "" {
		existing.Lastname = vcf.Lastname
	}
	if vcf.Nickname != "" {
		existing.Nickname = vcf.Nickname
	}
	if vcf.Email != "" {
		existing.Email = vcf.Email
	}
	if vcf.Phone != "" {
		existing.Phone = vcf.Phone
	}
	if vcf.Birthday != "" {
		existing.Birthday = vcf.Birthday
	}
	if vcf.Address != "" {
		existing.Address = vcf.Address
	}
	if vcf.Gender != "" {
		existing.Gender = vcf.Gender
	}
	if vcf.WorkInformation != "" {
		existing.WorkInformation = vcf.WorkInformation
	}
	if len(vcf.Circles) > 0 {
		existing.Circles = vcf.Circles
	}
	if vcf.VCardExtra != "" {
		existing.VCardExtra = vcf.VCardExtra
	}
	if vcf.VCardUID != "" {
		existing.VCardUID = vcf.VCardUID
	}
}

// CreateMergeNote creates a note documenting what was changed during import
func CreateMergeNote(db *gorm.DB, userID uint, contactID uint, original *models.Contact, newValues map[string]interface{}, importType string) error {
	var changes []string

	fieldLabels := map[string]struct {
		label    string
		original string
	}{
		"firstname":           {"First Name", original.Firstname},
		"lastname":            {"Last Name", original.Lastname},
		"nickname":            {"Nickname", original.Nickname},
		"email":               {"Email", original.Email},
		"phone":               {"Phone", original.Phone},
		"birthday":            {"Birthday", original.Birthday},
		"address":             {"Address", original.Address},
		"gender":              {"Gender", original.Gender},
		"how_we_met":          {"How We Met", original.HowWeMet},
		"food_preference":     {"Food Preferences", original.FoodPreference},
		"work_information":    {"Work Information", original.WorkInformation},
		"contact_information": {"Contact Information", original.ContactInformation},
	}

	for field, info := range fieldLabels {
		newVal := GetStringField(newValues, field)
		if newVal != "" && info.original != newVal {
			if info.original != "" {
				changes = append(changes, fmt.Sprintf("- %s: %s → %s", info.label, info.original, newVal))
			} else {
				changes = append(changes, fmt.Sprintf("- %s: (empty) → %s", info.label, newVal))
			}
		}
	}

	if newCirclesStr := GetStringField(newValues, "circles"); newCirclesStr != "" {
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
		return nil
	}

	content := fmt.Sprintf("%s Import updated this contact.\n\nChanges made:\n%s", importType, strings.Join(changes, "\n"))

	note := models.Note{
		UserID:    userID,
		ContactID: &contactID,
		Content:   content,
	}

	return db.Create(&note).Error
}
