package services

import (
	"encoding/csv"
	"fmt"
	"io"
	"meerkat/carddav"
	"meerkat/middleware"
	"meerkat/models"
	"regexp"
	"strconv"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"
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
	PhotoURL       string // URL to fetch photo from (if not embedded)
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
		contact, photoData, photoMediaType, photoURL := carddav.VCardToContact(card, nil)

		// Generate UUID for contacts without one to avoid unique constraint violation
		if contact.VCardUID == "" {
			contact.VCardUID = uuid.New().String()
		}

		contacts = append(contacts, VCFContactData{
			Contact:        contact,
			PhotoData:      photoData,
			PhotoMediaType: photoMediaType,
			PhotoURL:       photoURL,
		})

		// Build preview
		preview := models.ImportRowPreview{
			RowIndex:         rowIdx,
			ParsedContact:    ContactToPreviewMap(contact),
			ValidationErrors: make([]string, 0),
			SuggestedAction:  "add",
		}

		// Validate contact
		validationErrors := ValidateImportedContact(contact)
		preview.ValidationErrors = validationErrors

		if len(validationErrors) > 0 {
			stats.ErrorCount++
			preview.SuggestedAction = "skip"
		} else {
			stats.ValidCount++

			// Detect duplicates
			duplicate := DetectDuplicate(db, userID, contact.Firstname, contact.Lastname, contact.Email, contact.Phone)
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

// headerToField holds case-insensitive rules for non-indexed headers.
var headerToField = map[string]string{
	// English
	"firstname": "firstname", "first name": "firstname", "first": "firstname", "given name": "firstname",
	"lastname": "lastname", "last name": "lastname", "last": "lastname", "surname": "lastname", "family name": "lastname",
	"middle name": "middle_name", "middle": "middle_name", "additional name": "middle_name",
	"name prefix": "prefix", "prefix": "prefix", "title prefix": "prefix",
	"name suffix": "suffix", "suffix": "suffix",
	"nickname": "nickname", "nick": "nickname", "alias": "nickname",
	"email": "email", "e-mail": "email", "mail": "email", "email address": "email",
	"phone": "phone", "telephone": "phone", "tel": "phone", "mobile": "phone", "cell": "phone", "phone number": "phone",
	"website": "url", "web site": "url", "url": "url", "homepage": "url",
	"birthday": "birthday", "birth date": "birthday", "birthdate": "birthday", "dob": "birthday", "date of birth": "birthday",
	"anniversary": "anniversary",
	"address":     "address_street", "street address": "address_street", "home address": "address_street", "street": "address_street",
	"city": "address_city", "town": "address_city",
	"region": "address_region", "state": "address_region", "province": "address_region",
	"postal code": "address_postal", "zip": "address_postal", "zip code": "address_postal", "postcode": "address_postal",
	"country": "address_country",
	"gender":  "gender", "sex": "gender",
	"organization": "organization", "organization name": "organization", "company": "organization", "employer": "organization",
	"department": "department", "organization department": "department",
	"job title": "job_title", "title": "job_title", "organization title": "job_title", "position": "job_title",
	"role": "role", "organization role": "role",
	"how we met": "how_we_met", "how_we_met": "how_we_met", "notes": "how_we_met", "how i met": "how_we_met",
	"food": "food_preference", "food preference": "food_preference", "food_preference": "food_preference", "dietary": "food_preference", "diet": "food_preference",
	"work": "work_information", "work_information": "work_information", "job": "work_information", "occupation": "work_information",
	"contact information": "contact_information", "contact_information": "contact_information", "other contact": "contact_information",
	"circles": "circles", "groups": "circles", "tags": "circles", "category": "circles", "categories": "circles", "labels": "circles",
	// German
	"vorname":  "firstname",
	"nachname": "lastname", "familienname": "lastname",
	"zweiter vorname": "middle_name",
	"spitzname":       "nickname",
	"telefon":         "phone", "handy": "phone", "mobiltelefon": "phone",
	"webseite": "url", "website (de)": "url",
	"geburtstag": "birthday", "geburtsdatum": "birthday",
	"jahrestag": "anniversary",
	"adresse":   "address_street", "anschrift": "address_street", "straße": "address_street", "strasse": "address_street",
	"stadt": "address_city", "ort": "address_city",
	"bundesland": "address_region",
	"plz":        "address_postal", "postleitzahl": "address_postal",
	"land":       "address_country",
	"geschlecht": "gender",
	"firma":      "organization", "unternehmen": "organization",
	"abteilung": "department",
	"beruf":     "work_information", "arbeit": "work_information",
	"kreise": "circles", "gruppen": "circles",
}

// indexedHeaderRe matches Google-style grouped columns, e.g. "E-mail 1 - Value",
// "Phone 2 - Label", "Address 1 - Postal Code".
var indexedHeaderRe = regexp.MustCompile(`^(.+?)\s+(\d+)\s*-\s*(.+)$`)

func suggestGroupedMapping(header string) (field string, group int, ok bool) {
	m := indexedHeaderRe.FindStringSubmatch(strings.TrimSpace(header))
	if m == nil {
		return "", 0, false
	}
	base := strings.ToLower(strings.TrimSpace(m[1]))
	idx, err := strconv.Atoi(m[2])
	if err != nil || idx < 1 {
		return "", 0, false
	}
	attr := strings.ToLower(strings.TrimSpace(m[3]))

	// Identify the value family.
	var family string
	switch {
	case base == "e-mail" || base == "email":
		family = "email"
	case base == "phone" || base == "telephone" || base == "tel":
		family = "phone"
	case base == "website" || base == "web site" || base == "url":
		family = "url"
	case base == "im" || base == "instant message" || base == "instant messaging":
		family = "impp"
	case base == "address":
		family = "address"
	default:
		return "", 0, false
	}

	switch family {
	case "address":
		switch attr {
		case "street", "formatted", "po box", "extended address", "address":
			field = "address_street"
		case "city", "locality":
			field = "address_city"
		case "region", "state", "province":
			field = "address_region"
		case "postal code", "zip", "zip code", "postcode":
			field = "address_postal"
		case "country":
			field = "address_country"
		case "label", "type":
			field = "address_label"
		default:
			return "", 0, false
		}
	default:
		switch attr {
		case "value", "address", "uri":
			field = family
		case "label", "type":
			field = family + "_label"
		default:
			return "", 0, false
		}
	}

	return field, idx - 1, true
}

// SuggestColumnMappings guesses mappings based on CSV header names
func SuggestColumnMappings(headers []string) []models.ColumnMapping {
	mappings := make([]models.ColumnMapping, len(headers))

	for i, header := range headers {
		// indexed columns (e.g. "E-mail 1 - Value") take priority
		if field, group, ok := suggestGroupedMapping(header); ok {
			mappings[i] = models.ColumnMapping{CSVColumn: header, ContactField: field, Group: group}
			continue
		}

		normalized := strings.ToLower(strings.TrimSpace(header))
		if field, ok := headerToField[normalized]; ok {
			mappings[i] = models.ColumnMapping{CSVColumn: header, ContactField: field}
		} else {
			mappings[i] = models.ColumnMapping{CSVColumn: header, ContactField: ""} // Unmapped
		}
	}

	return mappings
}

// GenerateCSVPreview applies mappings to CSV rows
func GenerateCSVPreview(db *gorm.DB, userID uint, rows [][]string, headers []string, mappings []models.ColumnMapping) ([]models.Contact, []models.ImportRowPreview, ImportStats) {
	contacts := make([]models.Contact, len(rows))
	var previews []models.ImportRowPreview
	var stats ImportStats

	for rowIdx, row := range rows {
		contact := BuildContactFromRow(userID, headers, row, mappings)
		contacts[rowIdx] = contact

		preview := models.ImportRowPreview{
			RowIndex:         rowIdx,
			ParsedContact:    ContactToPreviewMap(&contact),
			ValidationErrors: ValidateImportedContact(&contact),
			SuggestedAction:  "add",
		}

		if len(preview.ValidationErrors) > 0 {
			stats.ErrorCount++
			preview.SuggestedAction = "skip"
		} else {
			stats.ValidCount++

			// Detect duplicates using the denormalized primary scalars.
			duplicate := DetectDuplicate(db, userID, contact.Firstname, contact.Lastname, contact.Email, contact.Phone)
			if duplicate != nil {
				preview.DuplicateMatch = duplicate
				preview.SuggestedAction = "update"
				stats.DuplicateCount++
			}
		}

		previews = append(previews, preview)
	}

	return contacts, previews, stats
}

// ContactToPreviewMap converts a Contact to a preview map used for display in the
// wizard and for diffing in merge notes. It carries the denormalized primary scalars
// plus the new structured fields; multi-value arrays are summarized by their primary.
func ContactToPreviewMap(contact *models.Contact) map[string]interface{} {
	preview := make(map[string]interface{})
	set := func(key, value string) {
		if value != "" {
			preview[key] = value
		}
	}
	set("firstname", contact.Firstname)
	set("lastname", contact.Lastname)
	set("middle_name", contact.MiddleName)
	set("prefix", contact.Prefix)
	set("suffix", contact.Suffix)
	set("nickname", contact.Nickname)
	set("email", contact.Email)
	set("phone", contact.Phone)
	set("birthday", contact.Birthday)
	set("anniversary", contact.Anniversary)
	set("address", contact.Address)
	set("gender", contact.Gender)
	set("organization", contact.Organization)
	set("department", contact.Department)
	set("job_title", contact.JobTitle)
	set("role", contact.Role)
	set("work_information", contact.WorkInformation)
	if len(contact.Circles) > 0 {
		preview["circles"] = strings.Join(contact.Circles, ", ")
	}
	return preview
}

// ValidateImportedContact validates a contact built from either CSV or VCF and returns
// human-readable errors. Used by both import preview paths.
func ValidateImportedContact(contact *models.Contact) []string {
	errors := make([]string, 0)

	if contact.Firstname == "" {
		errors = append(errors, "First name is required")
	}

	for _, e := range contact.Emails {
		if e.Value != "" && !middleware.ValidateEmail(e.Value) {
			errors = append(errors, "Invalid email format")
			break
		}
	}

	for _, p := range contact.Phones {
		if p.Value != "" && !IsValidPhone(p.Value) {
			errors = append(errors, "Invalid phone format")
			break
		}
	}

	if contact.Birthday != "" && !IsValidBirthdayFormat(contact.Birthday) {
		errors = append(errors, "Invalid birthday format (expected YYYY-MM-DD or --MM-DD)")
	}

	if contact.Anniversary != "" && !IsValidBirthdayFormat(contact.Anniversary) {
		errors = append(errors, "Invalid anniversary format (expected YYYY-MM-DD or --MM-DD)")
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

// normalizePhoneForComparison removes all non-digit characters from a phone number for comparison
func normalizePhoneForComparison(phone string) string {
	var normalized strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			normalized.WriteRune(r)
		}
	}
	return normalized.String()
}

// DetectDuplicate checks for existing contacts matching the given fields
func DetectDuplicate(db *gorm.DB, userID uint, firstname, lastname, email, phone string) *models.DuplicateMatch {
	var existing models.Contact

	// Priority 1: Email match (if email provided)
	if email != "" {
		if err := db.Where("user_id = ? AND LOWER(email) = LOWER(?)", userID, email).First(&existing).Error; err == nil {
			return &models.DuplicateMatch{
				ExistingContactID: existing.ID,
				ExistingFirstname: existing.Firstname,
				ExistingLastname:  existing.Lastname,
				ExistingEmail:     existing.Email,
				ExistingPhone:     existing.Phone,
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
				ExistingPhone:     existing.Phone,
				MatchReason:       "name",
			}
		}
	}

	// Priority 3: Phone match (if phone provided)
	// Normalize phone numbers for comparison (strip non-digits)
	if phone != "" {
		normalizedPhone := normalizePhoneForComparison(phone)
		if len(normalizedPhone) >= 5 { // Only match if we have enough digits
			var contacts []models.Contact
			if err := db.Where("user_id = ? AND phone != ''", userID).Find(&contacts).Error; err == nil {
				for _, c := range contacts {
					if normalizePhoneForComparison(c.Phone) == normalizedPhone {
						return &models.DuplicateMatch{
							ExistingContactID: c.ID,
							ExistingFirstname: c.Firstname,
							ExistingLastname:  c.Lastname,
							ExistingEmail:     c.Email,
							ExistingPhone:     c.Phone,
							MatchReason:       "phone",
						}
					}
				}
			}
		}
	}

	return nil
}

// ParseCircles parses circles from a separated string
func ParseCircles(input string) []string {
	if input == "" {
		return nil
	}

	// Normalize ":::" separator to a comma so the splitter below handles it.
	normalized := strings.ReplaceAll(input, ":::", ",")

	var circles []string
	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ',' || r == ';'
	})

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" || strings.HasPrefix(trimmed, "*") {
			continue
		}
		circles = append(circles, trimmed)
	}

	return circles
}

// addrEntry accumulates the components of one structured address while building a row.
type addrEntry struct {
	label   string
	street  string
	city    string
	region  string
	postal  string
	country string
}

func (a addrEntry) isEmpty() bool {
	return strings.TrimSpace(a.street+a.city+a.region+a.postal+a.country) == ""
}

// BuildContactFromRow assembles a full multi-value Contact from a single CSV row using
// the column mappings. Scalars are set directly; value/label/part columns sharing a
// (family, Group) assemble into one ContactEmail/Phone/Address/URL/IMPP entry.
func BuildContactFromRow(userID uint, headers []string, row []string, mappings []models.ColumnMapping) models.Contact {
	columnIndex := make(map[string]int, len(headers))
	for i, header := range headers {
		columnIndex[header] = i
	}

	// cellValue returns the trimmed value for a mapped column, or "" when out of range.
	cellValue := func(m models.ColumnMapping) string {
		idx, ok := columnIndex[m.CSVColumn]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	contact := models.Contact{UserID: userID}

	// Multi-value accumulators keyed by group index. Ordered slices preserve appearance.
	emailVals := map[int]string{}
	emailLabels := map[int]string{}
	phoneVals := map[int]string{}
	phoneLabels := map[int]string{}
	urlVals := map[int]string{}
	urlLabels := map[int]string{}
	imppVals := map[int]string{}
	imppLabels := map[int]string{}
	addrs := map[int]*addrEntry{}
	var emailGroups, phoneGroups, urlGroups, imppGroups, addrGroups []int

	addrFor := func(g int) *addrEntry {
		if a, ok := addrs[g]; ok {
			return a
		}
		a := &addrEntry{}
		addrs[g] = a
		addrGroups = append(addrGroups, g)
		return a
	}
	// if two columns manually mapped to same value like "Email", it bumps to the
	// next free group so both values survive rather than overwriting
	putValue := func(vals map[int]string, order *[]int, g int, v string) {
		if v == "" {
			return
		}
		if cur, ok := vals[g]; ok && cur != "" {
			for {
				g++
				if _, taken := vals[g]; !taken {
					break
				}
			}
		}
		if _, seen := vals[g]; !seen {
			*order = append(*order, g)
		}
		vals[g] = v
	}

	for _, m := range mappings {
		if m.ContactField == "" {
			continue
		}
		v := cellValue(m)
		switch m.ContactField {
		case "firstname":
			contact.Firstname = v
		case "lastname":
			contact.Lastname = v
		case "middle_name":
			contact.MiddleName = v
		case "prefix":
			contact.Prefix = v
		case "suffix":
			contact.Suffix = v
		case "nickname":
			contact.Nickname = v
		case "gender":
			if v != "" {
				contact.Gender = NormalizeGender(v)
			}
		case "birthday":
			if v != "" {
				contact.Birthday = NormalizeBirthday(v)
			}
		case "anniversary":
			if v != "" {
				contact.Anniversary = NormalizeBirthday(v)
			}
		case "organization":
			contact.Organization = v
		case "department":
			contact.Department = v
		case "job_title":
			contact.JobTitle = v
		case "role":
			contact.Role = v
		case "how_we_met":
			contact.HowWeMet = v
		case "food_preference":
			contact.FoodPreference = v
		case "work_information":
			contact.WorkInformation = v
		case "contact_information":
			contact.ContactInformation = v
		case "circles":
			if v != "" {
				contact.Circles = ParseCircles(v)
			}
		case "email":
			putValue(emailVals, &emailGroups, m.Group, v)
		case "email_label":
			emailLabels[m.Group] = v
		case "phone":
			putValue(phoneVals, &phoneGroups, m.Group, v)
		case "phone_label":
			phoneLabels[m.Group] = v
		case "url":
			putValue(urlVals, &urlGroups, m.Group, v)
		case "url_label":
			urlLabels[m.Group] = v
		case "impp":
			putValue(imppVals, &imppGroups, m.Group, v)
		case "impp_label":
			imppLabels[m.Group] = v
		case "address_street":
			addrFor(m.Group).street = v
		case "address_city":
			addrFor(m.Group).city = v
		case "address_region":
			addrFor(m.Group).region = v
		case "address_postal":
			addrFor(m.Group).postal = v
		case "address_country":
			addrFor(m.Group).country = v
		case "address_label":
			addrFor(m.Group).label = v
		}
	}

	for _, g := range emailGroups {
		if v := emailVals[g]; v != "" {
			contact.Emails = append(contact.Emails, models.ContactEmail{Type: normalizeImportType(emailLabels[g], "home"), Value: v})
		}
	}
	for _, g := range phoneGroups {
		if v := phoneVals[g]; v != "" {
			contact.Phones = append(contact.Phones, models.ContactPhone{Type: normalizeImportType(phoneLabels[g], "cell"), Value: v})
		}
	}
	for _, g := range urlGroups {
		if v := urlVals[g]; v != "" {
			contact.URLs = append(contact.URLs, models.ContactURL{Type: normalizeImportType(urlLabels[g], "home"), Value: v})
		}
	}
	for _, g := range imppGroups {
		if v := imppVals[g]; v != "" {
			contact.IMPPs = append(contact.IMPPs, models.ContactIMPP{Type: normalizeImportType(imppLabels[g], ""), Value: v})
		}
	}
	for _, g := range addrGroups {
		a := addrs[g]
		if a.isEmpty() {
			continue
		}
		contact.Addresses = append(contact.Addresses, models.ContactAddress{
			Type:    normalizeImportType(a.label, "home"),
			Street:  a.street,
			City:    a.city,
			Region:  a.region,
			Postal:  a.postal,
			Country: a.country,
		})
	}

	// Mirror the primary entries into the denormalized scalars so duplicate detection works
	if len(contact.Emails) > 0 {
		contact.Email = contact.Emails[0].Value
	}
	if len(contact.Phones) > 0 {
		contact.Phone = contact.Phones[0].Value
	}
	if len(contact.Addresses) > 0 {
		contact.Address = models.FormatAddress(contact.Addresses[0])
	}

	return contact
}

// clean up a type/label token (e.g. Google's "* Home")
func normalizeImportType(label, def string) string {
	t := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(label), "*")))
	switch t {
	case "":
		return def
	case "mobile":
		return "cell"
	default:
		return t
	}
}

// merges fields from an imported contact into an existing one, overwriting only non-empty incoming values
func MergeImportedContact(existing *models.Contact, incoming *models.Contact) {
	if incoming.Firstname != "" {
		existing.Firstname = incoming.Firstname
	}
	if incoming.Lastname != "" {
		existing.Lastname = incoming.Lastname
	}
	if incoming.Nickname != "" {
		existing.Nickname = incoming.Nickname
	}
	if incoming.Email != "" {
		existing.Email = incoming.Email
	}
	if incoming.Phone != "" {
		existing.Phone = incoming.Phone
	}
	if incoming.Birthday != "" {
		existing.Birthday = incoming.Birthday
	}
	if incoming.Address != "" {
		existing.Address = incoming.Address
	}
	if incoming.Gender != "" {
		existing.Gender = incoming.Gender
	}
	if incoming.WorkInformation != "" {
		existing.WorkInformation = incoming.WorkInformation
	}
	if incoming.HowWeMet != "" {
		existing.HowWeMet = incoming.HowWeMet
	}
	if incoming.FoodPreference != "" {
		existing.FoodPreference = incoming.FoodPreference
	}
	if incoming.ContactInformation != "" {
		existing.ContactInformation = incoming.ContactInformation
	}
	if len(incoming.Circles) > 0 {
		existing.Circles = incoming.Circles
	}
	// Multi-valued and structured vCard fields
	if len(incoming.Emails) > 0 {
		existing.Emails = incoming.Emails
	}
	if len(incoming.Phones) > 0 {
		existing.Phones = incoming.Phones
	}
	if len(incoming.Addresses) > 0 {
		existing.Addresses = incoming.Addresses
	}
	if len(incoming.URLs) > 0 {
		existing.URLs = incoming.URLs
	}
	if len(incoming.IMPPs) > 0 {
		existing.IMPPs = incoming.IMPPs
	}
	if incoming.MiddleName != "" {
		existing.MiddleName = incoming.MiddleName
	}
	if incoming.Prefix != "" {
		existing.Prefix = incoming.Prefix
	}
	if incoming.Suffix != "" {
		existing.Suffix = incoming.Suffix
	}
	if incoming.Organization != "" {
		existing.Organization = incoming.Organization
	}
	if incoming.Department != "" {
		existing.Department = incoming.Department
	}
	if incoming.JobTitle != "" {
		existing.JobTitle = incoming.JobTitle
	}
	if incoming.Role != "" {
		existing.Role = incoming.Role
	}
	if incoming.Anniversary != "" {
		existing.Anniversary = incoming.Anniversary
	}
	if incoming.VCardExtra != "" {
		existing.VCardExtra = incoming.VCardExtra
	}
	if incoming.VCardUID != "" {
		existing.VCardUID = incoming.VCardUID
	}
}

// creates a note documenting what was changed during import
func CreateMergeNote(db *gorm.DB, userID uint, contactID uint, original *models.Contact, newValues map[string]interface{}, importType string) error {
	var changes []string

	fieldLabels := map[string]struct {
		label    string
		original string
	}{
		"firstname":           {"First Name", original.Firstname},
		"lastname":            {"Last Name", original.Lastname},
		"middle_name":         {"Middle Name", original.MiddleName},
		"prefix":              {"Prefix", original.Prefix},
		"suffix":              {"Suffix", original.Suffix},
		"nickname":            {"Nickname", original.Nickname},
		"email":               {"Email", original.Email},
		"phone":               {"Phone", original.Phone},
		"birthday":            {"Birthday", original.Birthday},
		"anniversary":         {"Anniversary", original.Anniversary},
		"address":             {"Address", original.Address},
		"gender":              {"Gender", original.Gender},
		"organization":        {"Organization", original.Organization},
		"department":          {"Department", original.Department},
		"job_title":           {"Job Title", original.JobTitle},
		"role":                {"Role", original.Role},
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
