package carddav

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"meerkat/models"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

// VCardExtra stores unmapped vCard properties
type VCardExtra struct {
	Properties map[string][]vcard.Field `json:"properties,omitempty"`
}

// ContactToVCard converts a Contact to a vCard 4.0
func ContactToVCard(contact *models.Contact, photoDir string) vcard.Card {
	card := make(vcard.Card)

	// Required: VERSION
	card.SetValue(vcard.FieldVersion, "4.0")

	// UID - use VCardUID if set, otherwise generate a new one
	uid := contact.VCardUID
	if uid == "" {
		uid = generateUID()
	}
	card.SetValue(vcard.FieldUID, uid)

	// FN (formatted name) - required in vCard 4.0
	fn := strings.TrimSpace(contact.Firstname + " " + contact.Lastname)
	if fn == "" {
		fn = contact.Nickname
	}
	if fn == "" {
		fn = "Unknown"
	}
	card.SetValue(vcard.FieldFormattedName, fn)

	// N (structured name)
	card.Set(vcard.FieldName, &vcard.Field{
		Value: contact.Lastname + ";" + contact.Firstname + ";;;",
	})

	// NICKNAME
	if contact.Nickname != "" {
		card.SetValue(vcard.FieldNickname, contact.Nickname)
	}

	// EMAIL
	if contact.Email != "" {
		card.Set(vcard.FieldEmail, &vcard.Field{
			Value:  contact.Email,
			Params: vcard.Params{vcard.ParamType: {"INTERNET"}},
		})
	}

	// TEL (phone) - default to CELL (mobile) as most contacts have mobile numbers
	if contact.Phone != "" {
		card.Set(vcard.FieldTelephone, &vcard.Field{
			Value:  contact.Phone,
			Params: vcard.Params{vcard.ParamType: {"CELL"}},
		})
	}

	// ADR (address)
	if contact.Address != "" {
		// Store as unstructured address (extended address field)
		card.Set(vcard.FieldAddress, &vcard.Field{
			Value: ";;" + contact.Address + ";;;;",
		})
	}

	// BDAY (birthday) - convert to vCard 4.0 format
	if contact.Birthday != "" {
		bday := contact.Birthday
		// Convert --MM-DD to --MMDD for vCard 4.0 compatibility
		if len(bday) == 7 && bday[0] == '-' && bday[1] == '-' && bday[4] == '-' {
			bday = bday[:4] + bday[5:] // --06-12 -> --0612
		}
		card.SetValue(vcard.FieldBirthday, bday)
	}

	// GENDER
	if contact.Gender != "" {
		gender := mapGenderToVCard(contact.Gender)
		if gender != "" {
			card.SetValue(vcard.FieldGender, gender)
		}
	}

	// CATEGORIES (circles)
	if len(contact.Circles) > 0 {
		card.SetValue(vcard.FieldCategories, strings.Join(contact.Circles, ","))
	}

	// ORG (work information)
	if contact.WorkInformation != "" {
		card.SetValue(vcard.FieldOrganization, contact.WorkInformation)
	}

	// PHOTO - read from disk, fall back to thumbnail
	// Note: In vCard 4.0, base64 encoding is implicit, so ENCODING parameter is not used
	photoData, mediaType := readContactPhoto(contact, photoDir)
	if photoData != "" {
		card.Set(vcard.FieldPhoto, &vcard.Field{
			Value: photoData,
			Params: vcard.Params{
				"MEDIATYPE": {mediaType},
			},
		})
	}

	// Restore unmapped properties from VCardExtra
	if contact.VCardExtra != "" {
		var extra VCardExtra
		if err := json.Unmarshal([]byte(contact.VCardExtra), &extra); err == nil {
			for name, fields := range extra.Properties {
				for _, field := range fields {
					card.Add(name, &field)
				}
			}
		}
	}

	return card
}

// VCardToContact converts a vCard to a Contact, updating existing fields
// Returns the updated contact and photo data if present (for separate processing)
func VCardToContact(card vcard.Card, existing *models.Contact) (*models.Contact, []byte, string) {
	contact := existing
	if contact == nil {
		contact = &models.Contact{}
	}

	// UID
	if uid := card.Value(vcard.FieldUID); uid != "" {
		contact.VCardUID = uid
	}

	// N (structured name) - prefer over FN
	if name := card.Name(); name != nil {
		contact.Firstname = name.GivenName
		contact.Lastname = name.FamilyName
	} else if fn := card.Value(vcard.FieldFormattedName); fn != "" {
		// Fall back to FN - try to split
		parts := strings.SplitN(fn, " ", 2)
		contact.Firstname = parts[0]
		if len(parts) > 1 {
			contact.Lastname = parts[1]
		}
	}

	// NICKNAME
	if nickname := card.Value(vcard.FieldNickname); nickname != "" {
		contact.Nickname = nickname
	}

	// EMAIL - take first one
	if emails := card.Values(vcard.FieldEmail); len(emails) > 0 {
		contact.Email = emails[0]
	}

	// TEL - take first one
	if phones := card.Values(vcard.FieldTelephone); len(phones) > 0 {
		contact.Phone = phones[0]
	}

	// ADR - combine into single string
	if addresses := card.Addresses(); len(addresses) > 0 {
		addr := addresses[0]
		parts := []string{}
		if addr.StreetAddress != "" {
			parts = append(parts, addr.StreetAddress)
		}
		if addr.ExtendedAddress != "" {
			parts = append(parts, addr.ExtendedAddress)
		}
		if addr.Locality != "" {
			parts = append(parts, addr.Locality)
		}
		if addr.Region != "" {
			parts = append(parts, addr.Region)
		}
		if addr.PostalCode != "" {
			parts = append(parts, addr.PostalCode)
		}
		if addr.Country != "" {
			parts = append(parts, addr.Country)
		}
		contact.Address = strings.Join(parts, ", ")
	}

	// BDAY
	if bday := card.Value(vcard.FieldBirthday); bday != "" {
		contact.Birthday = normalizeBirthday(bday)
	}

	// GENDER
	if gender := card.Value(vcard.FieldGender); gender != "" {
		contact.Gender = mapGenderFromVCard(gender)
	}

	// CATEGORIES -> Circles
	if categories := card.Value(vcard.FieldCategories); categories != "" {
		contact.Circles = strings.Split(categories, ",")
		for i, c := range contact.Circles {
			contact.Circles[i] = strings.TrimSpace(c)
		}
	}

	// ORG -> WorkInformation
	if org := card.Value(vcard.FieldOrganization); org != "" {
		contact.WorkInformation = org
	}

	// Extract photo data for separate processing
	var photoData []byte
	var photoMediaType string
	if photoField := card.Get(vcard.FieldPhoto); photoField != nil {
		photoData, photoMediaType = extractPhotoData(photoField)
	}

	// Store unmapped properties in VCardExtra
	extra := extractUnmappedProperties(card)
	if len(extra.Properties) > 0 {
		extraJSON, _ := json.Marshal(extra)
		contact.VCardExtra = string(extraJSON)
	}

	return contact, photoData, photoMediaType
}

// SaveContactPhoto saves photo data to disk and generates thumbnail
// Returns the photo filename and base64 thumbnail data URL
func SaveContactPhoto(photoData []byte, mediaType string, photoDir string) (string, string, error) {
	if len(photoData) == 0 {
		return "", "", nil
	}

	// Detect content type if not provided
	if mediaType == "" {
		mediaType = http.DetectContentType(photoData)
	}

	// Decode the image
	var img image.Image
	var err error

	reader := bytes.NewReader(photoData)
	switch {
	case strings.Contains(mediaType, "jpeg") || strings.Contains(mediaType, "jpg"):
		img, err = jpeg.Decode(reader)
	case strings.Contains(mediaType, "png"):
		img, err = png.Decode(reader)
	default:
		// Try to decode as JPEG first, then PNG
		img, err = jpeg.Decode(reader)
		if err != nil {
			reader.Seek(0, 0)
			img, err = png.Decode(reader)
		}
	}
	if err != nil {
		return "", "", err
	}

	// Generate unique filename
	baseFilename := uuid.New().String()
	photoPath := baseFilename + "_photo.jpg"

	// Ensure directory exists
	if err := os.MkdirAll(photoDir, os.ModePerm); err != nil {
		return "", "", err
	}

	// Save resized photo (125x125 to match existing behavior)
	resizedPhoto := resize.Resize(125, 125, img, resize.Lanczos3)
	fullPhotoPath := filepath.Join(photoDir, photoPath)
	outFile, err := os.Create(fullPhotoPath)
	if err != nil {
		return "", "", err
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, resizedPhoto, &jpeg.Options{Quality: 85}); err != nil {
		return "", "", err
	}

	// Create thumbnail and encode as base64 data URL (48x48 to match existing behavior)
	thumbnail := resize.Resize(48, 48, img, resize.Lanczos3)
	var thumbnailBuf bytes.Buffer
	if err := jpeg.Encode(&thumbnailBuf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return "", "", err
	}
	thumbnailBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumbnailBuf.Bytes())

	return photoPath, thumbnailBase64, nil
}

// readContactPhoto reads photo from disk or falls back to thumbnail
func readContactPhoto(contact *models.Contact, photoDir string) (string, string) {
	// Try to read full photo from disk
	if contact.Photo != "" && photoDir != "" {
		photoPath := filepath.Join(photoDir, contact.Photo)
		data, err := os.ReadFile(photoPath)
		if err == nil {
			mediaType := http.DetectContentType(data)
			return base64.StdEncoding.EncodeToString(data), mediaType
		}
	}

	// Fall back to thumbnail (already base64)
	if contact.PhotoThumbnail != "" && strings.HasPrefix(contact.PhotoThumbnail, "data:") {
		// Parse data URL: data:image/jpeg;base64,<data>
		parts := strings.SplitN(contact.PhotoThumbnail, ",", 2)
		if len(parts) == 2 {
			// Extract media type from first part
			mediaType := "image/jpeg"
			if strings.Contains(parts[0], "image/png") {
				mediaType = "image/png"
			}
			return parts[1], mediaType
		}
	}

	return "", ""
}

// extractPhotoData extracts binary photo data from a vCard PHOTO field
func extractPhotoData(field *vcard.Field) ([]byte, string) {
	if field == nil || field.Value == "" {
		return nil, ""
	}

	value := field.Value
	mediaType := ""

	// Check for MEDIATYPE or TYPE parameter
	if mt := field.Params.Get("MEDIATYPE"); mt != "" {
		mediaType = mt
	} else if t := field.Params.Get("TYPE"); t != "" {
		mediaType = "image/" + strings.ToLower(t)
	}

	// Check if it's a data URI
	if strings.HasPrefix(value, "data:") {
		parts := strings.SplitN(value, ",", 2)
		if len(parts) == 2 {
			if mediaType == "" && strings.Contains(parts[0], "image/") {
				// Extract media type from data URI
				start := strings.Index(parts[0], "image/")
				end := strings.Index(parts[0][start:], ";")
				if end == -1 {
					mediaType = parts[0][start:]
				} else {
					mediaType = parts[0][start : start+end]
				}
			}
			value = parts[1]
		}
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		// Try URL-safe base64
		data, err = base64.URLEncoding.DecodeString(value)
		if err != nil {
			return nil, ""
		}
	}

	return data, mediaType
}

// generateUID creates a UID for a contact
func generateUID() string {
	return uuid.New().String()
}

// mapGenderToVCard converts internal gender to vCard format
func mapGenderToVCard(gender string) string {
	switch gender {
	case "male":
		return "M"
	case "female":
		return "F"
	case "other":
		return "O"
	case "prefer_not_to_say":
		return "N"
	default:
		return ""
	}
}

// mapGenderFromVCard converts vCard gender to internal format
func mapGenderFromVCard(gender string) string {
	switch strings.ToUpper(gender) {
	case "M":
		return "male"
	case "F":
		return "female"
	case "O":
		return "other"
	case "N", "U":
		return "prefer_not_to_say"
	default:
		return ""
	}
}

// normalizeBirthday ensures birthday is in YYYY-MM-DD or --MM-DD format for storage
func normalizeBirthday(bday string) string {
	// Already in correct format (YYYY-MM-DD)
	if len(bday) == 10 && bday[4] == '-' && bday[7] == '-' {
		return bday
	}
	// Already in correct format (--MM-DD)
	if len(bday) == 7 && bday[0] == '-' && bday[1] == '-' && bday[4] == '-' {
		return bday
	}

	// Handle YYYYMMDD format (vCard 3.0)
	if len(bday) == 8 && bday[0] != '-' {
		return bday[:4] + "-" + bday[4:6] + "-" + bday[6:]
	}

	// Handle --MMDD format (vCard 4.0 without year) -> convert to --MM-DD
	if len(bday) == 6 && bday[0] == '-' && bday[1] == '-' {
		return bday[:4] + "-" + bday[4:] // --0612 -> --06-12
	}

	return bday
}

// extractUnmappedProperties extracts vCard properties not mapped to Contact fields
func extractUnmappedProperties(card vcard.Card) VCardExtra {
	mappedFields := map[string]bool{
		vcard.FieldVersion:       true,
		vcard.FieldUID:           true,
		vcard.FieldFormattedName: true,
		vcard.FieldName:          true,
		vcard.FieldNickname:      true,
		vcard.FieldEmail:         true,
		vcard.FieldTelephone:     true,
		vcard.FieldAddress:       true,
		vcard.FieldBirthday:      true,
		vcard.FieldGender:        true,
		vcard.FieldCategories:    true,
		vcard.FieldOrganization:  true,
		vcard.FieldPhoto:         true,
	}

	extra := VCardExtra{
		Properties: make(map[string][]vcard.Field),
	}

	for name, fields := range card {
		if !mappedFields[name] {
			for _, field := range fields {
				extra.Properties[name] = append(extra.Properties[name], *field)
			}
		}
	}

	return extra
}
