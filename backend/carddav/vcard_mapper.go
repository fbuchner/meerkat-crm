package carddav

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"meerkat/httputil"
	"meerkat/models"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/gen2brain/heic"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

// for unmapped vCard properties
type VCardExtra struct {
	Properties map[string][]vcard.Field `json:"properties,omitempty"`
}

// ContactToVCard converts a Contact to a vCard 3.0 card.
func ContactToVCard(contact *models.Contact, photoDir string) vcard.Card {
	card := make(vcard.Card)

	// Required: VERSION - use 3.0 for iOS compatibility
	card.SetValue(vcard.FieldVersion, "3.0")

	// UID - use VCardUID if set, otherwise generate a new one
	uid := contact.VCardUID
	if uid == "" {
		uid = generateUID()
	}
	card.SetValue(vcard.FieldUID, uid)

	// FN (formatted name) - required
	fn := strings.TrimSpace(contact.Firstname + " " + contact.Lastname)
	if fn == "" {
		fn = contact.Nickname
	}
	if fn == "" {
		fn = "Unknown"
	}
	card.SetValue(vcard.FieldFormattedName, fn)

	// N (structured name): FamilyName;GivenName;AdditionalName;HonorificPrefix;HonorificSuffix
	card.SetValue(vcard.FieldName, strings.Join([]string{
		contact.Lastname, contact.Firstname, contact.MiddleName, contact.Prefix, contact.Suffix,
	}, ";"))

	// NICKNAME
	if contact.Nickname != "" {
		card.SetValue(vcard.FieldNickname, contact.Nickname)
	}

	// EMAIL - emit every entry; fall back to the legacy scalar if the array is empty
	emails := contact.Emails
	if len(emails) == 0 && contact.Email != "" {
		emails = []models.ContactEmail{{Type: "home", Value: contact.Email}}
	}
	for _, e := range emails {
		if e.Value == "" {
			continue
		}
		card.Add(vcard.FieldEmail, &vcard.Field{
			Value:  e.Value,
			Params: emailParams(e.Type),
		})
	}

	// TEL (phone) - emit every entry; fall back to the legacy scalar if the array is empty
	phones := contact.Phones
	if len(phones) == 0 && contact.Phone != "" {
		phones = []models.ContactPhone{{Type: "cell", Value: contact.Phone}}
	}
	for _, p := range phones {
		if p.Value == "" {
			continue
		}
		card.Add(vcard.FieldTelephone, &vcard.Field{
			Value:  p.Value,
			Params: typeParams(p.Type),
		})
	}

	// ADR (address) - structured: POBox;Extended;Street;Locality;Region;Postal;Country
	addresses := contact.Addresses
	if len(addresses) == 0 && contact.Address != "" {
		addresses = []models.ContactAddress{{Type: "home", Street: contact.Address}}
	}
	for _, a := range addresses {
		if isEmptyAddress(a) {
			continue
		}
		// ADR components: POBox;Extended;Street;Locality;Region;Postal;Country
		comps := []string{"", "", a.Street, a.City, a.Region, a.Postal, a.Country}
		for i := range comps {
			comps[i] = escapeComponent(comps[i])
		}
		card.Add(vcard.FieldAddress, &vcard.Field{
			Value:  strings.Join(comps, ";"),
			Params: typeParams(a.Type),
		})
	}

	// URL (websites)
	for _, u := range contact.URLs {
		if u.Value == "" {
			continue
		}
		card.Add(vcard.FieldURL, &vcard.Field{
			Value:  u.Value,
			Params: typeParams(u.Type),
		})
	}

	// IMPP (instant messaging / social handles) - service goes in the X-SERVICE-TYPE param
	for _, im := range contact.IMPPs {
		if im.Value == "" {
			continue
		}
		params := vcard.Params{}
		if im.Type != "" {
			params["X-SERVICE-TYPE"] = []string{im.Type}
		}
		card.Add(vcard.FieldIMPP, &vcard.Field{Value: im.Value, Params: params})
	}

	// BDAY (birthday) - vCard 3.0 uses YYYY-MM-DD; store as-is (--MM-DD is also accepted)
	if contact.Birthday != "" {
		card.SetValue(vcard.FieldBirthday, contact.Birthday)
	}

	// ANNIVERSARY
	if contact.Anniversary != "" {
		card.SetValue(vcard.FieldAnniversary, contact.Anniversary)
	}

	// CATEGORIES (circles)
	if len(contact.Circles) > 0 {
		card.SetValue(vcard.FieldCategories, strings.Join(contact.Circles, ","))
	}

	// ORG (organization;department). Prefer the dedicated field; fall back to the
	// legacy WorkInformation so already-synced data is unaffected.
	org := contact.Organization
	if org == "" {
		org = contact.WorkInformation
	}
	if org != "" || contact.Department != "" {
		comps := []string{escapeComponent(org)}
		if contact.Department != "" {
			comps = append(comps, escapeComponent(contact.Department))
		}
		card.SetValue(vcard.FieldOrganization, strings.Join(comps, ";"))
	}

	// TITLE / ROLE
	if contact.JobTitle != "" {
		card.SetValue(vcard.FieldTitle, contact.JobTitle)
	}
	if contact.Role != "" {
		card.SetValue(vcard.FieldRole, contact.Role)
	}

	// PHOTO - read from disk, fall back to thumbnail
	// Include both vCard 3.0 and 4.0 parameters for maximum compatibility:
	// - ENCODING=b and TYPE=JPEG for vCard 3.0 (required by iOS)
	// - MEDIATYPE=image/jpeg for vCard 4.0
	photoData, mediaType := readContactPhoto(contact, photoDir)
	if photoData != "" {
		// Extract just the image type (e.g., "JPEG" from "image/jpeg")
		imageType := "JPEG"
		if strings.Contains(mediaType, "png") {
			imageType = "PNG"
		}
		card.Set(vcard.FieldPhoto, &vcard.Field{
			Value: photoData,
			Params: vcard.Params{
				"ENCODING": {"b"},       // vCard 3.0: base64 encoding
				"TYPE":     {imageType}, // vCard 3.0: image type
			},
		})
	}

	// Restore unmapped properties from VCardExtra. Skip any property that is now
	// mapped to a real column: legacy data may still carry it in vcard_extra (until
	// migration 000021 strips it), and emitting it here as well would duplicate the
	// value alongside the one written from the column above.
	if contact.VCardExtra != "" {
		var extra VCardExtra
		if err := json.Unmarshal([]byte(contact.VCardExtra), &extra); err == nil {
			mapped := mappedVCardFields()
			for name, fields := range extra.Properties {
				if mapped[name] {
					continue
				}
				for _, field := range fields {
					card.Add(name, &field)
				}
			}
		}
	}

	return card
}

// VCardToContact converts a vCard to a Contact, updating existing fields
// Returns the updated contact, photo data if embedded, media type, and photo URL if remote
func VCardToContact(card vcard.Card, existing *models.Contact) (*models.Contact, []byte, string, string) {
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
		contact.MiddleName = name.AdditionalName
		contact.Prefix = name.HonorificPrefix
		contact.Suffix = name.HonorificSuffix
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

	// EMAIL - import every entry with its type
	if fields := card[vcard.FieldEmail]; len(fields) > 0 {
		contact.Emails = contact.Emails[:0]
		for _, f := range fields {
			if f.Value == "" {
				continue
			}
			contact.Emails = append(contact.Emails, models.ContactEmail{
				Type:  typeFromField(f),
				Value: f.Value,
			})
		}
		if len(contact.Emails) > 0 {
			contact.Email = contact.Emails[0].Value
		}
	}

	// TEL - import every entry with its type
	if fields := card[vcard.FieldTelephone]; len(fields) > 0 {
		contact.Phones = contact.Phones[:0]
		for _, f := range fields {
			if f.Value == "" {
				continue
			}
			contact.Phones = append(contact.Phones, models.ContactPhone{
				Type:  typeFromField(f),
				Value: f.Value,
			})
		}
		if len(contact.Phones) > 0 {
			contact.Phone = contact.Phones[0].Value
		}
	}

	// ADR - import every structured address. Parsed manually (rather than via
	// card.Addresses()) so our "\;" escaping of embedded semicolons round-trips;
	// go-vcard's helper splits naively on every ";".
	if fields := card[vcard.FieldAddress]; len(fields) > 0 {
		contact.Addresses = contact.Addresses[:0]
		for _, f := range fields {
			// ADR components: POBox;Extended;Street;Locality;Region;Postal;Country
			comps := splitComponents(f.Value)
			ca := models.ContactAddress{
				Type:    typeFromField(f),
				Street:  strings.TrimSpace(strings.Join(nonEmpty(component(comps, 2), component(comps, 1)), " ")),
				City:    component(comps, 3),
				Region:  component(comps, 4),
				Postal:  component(comps, 5),
				Country: component(comps, 6),
			}
			if !isEmptyAddress(ca) {
				contact.Addresses = append(contact.Addresses, ca)
			}
		}
		if len(contact.Addresses) > 0 {
			contact.Address = models.FormatAddress(contact.Addresses[0])
		}
	}

	// URL - import every entry
	if fields := card[vcard.FieldURL]; len(fields) > 0 {
		contact.URLs = contact.URLs[:0]
		for _, f := range fields {
			if f.Value == "" {
				continue
			}
			contact.URLs = append(contact.URLs, models.ContactURL{
				Type:  typeFromField(f),
				Value: f.Value,
			})
		}
	}

	// IMPP - import every entry; service comes from X-SERVICE-TYPE or TYPE
	if fields := card[vcard.FieldIMPP]; len(fields) > 0 {
		contact.IMPPs = contact.IMPPs[:0]
		for _, f := range fields {
			if f.Value == "" {
				continue
			}
			service := f.Params.Get("X-SERVICE-TYPE")
			if service == "" {
				service = typeFromField(f)
			}
			contact.IMPPs = append(contact.IMPPs, models.ContactIMPP{
				Type:  service,
				Value: f.Value,
			})
		}
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

	// ORG -> Organization (+ Department after the ';' separator)
	if org := card.Value(vcard.FieldOrganization); org != "" {
		comps := splitComponents(org)
		contact.Organization = strings.TrimSpace(component(comps, 0))
		if d := strings.TrimSpace(component(comps, 1)); d != "" {
			contact.Department = d
		}
	}

	// TITLE / ROLE
	if title := card.Value(vcard.FieldTitle); title != "" {
		contact.JobTitle = title
	}
	if role := card.Value(vcard.FieldRole); role != "" {
		contact.Role = role
	}

	// ANNIVERSARY
	if anniv := card.Value(vcard.FieldAnniversary); anniv != "" {
		contact.Anniversary = normalizeBirthday(anniv)
	}

	// Extract photo data for separate processing
	var photoData []byte
	var photoMediaType string
	var photoURL string
	if photoField := card.Get(vcard.FieldPhoto); photoField != nil {
		photoData, photoMediaType, photoURL = extractPhotoData(photoField)
	}

	// Store unmapped properties in VCardExtra
	extra := extractUnmappedProperties(card)
	if len(extra.Properties) > 0 {
		extraJSON, _ := json.Marshal(extra)
		contact.VCardExtra = string(extraJSON)
	}

	return contact, photoData, photoMediaType, photoURL
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
	case strings.Contains(mediaType, "heic") || strings.Contains(mediaType, "heif"):
		img, err = heic.Decode(reader)
	default:
		// Check for HEIC magic bytes if media type is unknown
		// HEIC files have "ftyp" followed by "heic", "heix", "hevc", "hevx", or "mif1" at byte 4
		if len(photoData) >= 12 && string(photoData[4:8]) == "ftyp" {
			brand := string(photoData[8:12])
			if brand == "heic" || brand == "heix" || brand == "hevc" || brand == "hevx" || brand == "mif1" {
				img, err = heic.Decode(reader)
			}
		}
		// Try to decode as JPEG first, then PNG, then HEIC
		if img == nil {
			img, err = jpeg.Decode(reader)
			if err != nil {
				reader.Seek(0, 0)
				img, err = png.Decode(reader)
				if err != nil {
					reader.Seek(0, 0)
					img, err = heic.Decode(reader)
				}
			}
		}
	}
	if err != nil {
		return "", "", err
	}

	// Crop to centered square if rectangular
	img = cropToSquare(img)

	// Generate unique filename
	baseFilename := uuid.New().String()
	photoPath := baseFilename + "_photo.jpg"

	// Ensure directory exists
	if err := os.MkdirAll(photoDir, os.ModePerm); err != nil {
		return "", "", err
	}

	// Save photo (max 400x400, only downscale - smaller images keep their size)
	const maxPhotoSize = 400
	bounds := img.Bounds()
	photoImg := img
	if bounds.Dx() > maxPhotoSize || bounds.Dy() > maxPhotoSize {
		photoImg = resize.Resize(maxPhotoSize, maxPhotoSize, img, resize.Lanczos3)
	}
	fullPhotoPath := filepath.Join(photoDir, photoPath)
	outFile, err := os.Create(fullPhotoPath)
	if err != nil {
		return "", "", err
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, photoImg, &jpeg.Options{Quality: 85}); err != nil {
		return "", "", err
	}

	// Create thumbnail and encode as base64 data URL (48x48)
	thumbnail := resize.Resize(48, 48, img, resize.Lanczos3)
	var thumbnailBuf bytes.Buffer
	if err := jpeg.Encode(&thumbnailBuf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return "", "", err
	}
	thumbnailBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumbnailBuf.Bytes())

	return photoPath, thumbnailBase64, nil
}

// cropToSquare crops an image to a centered square
// If the image is already square, it returns the original image unchanged
func cropToSquare(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Already square, return as-is
	if width == height {
		return img
	}

	// Calculate the size of the square (use the smaller dimension)
	size := width
	if height < width {
		size = height
	}

	// Calculate crop offset to center the square
	offsetX := (width - size) / 2
	offsetY := (height - size) / 2

	// Create a new RGBA image for the cropped result
	cropped := image.NewRGBA(image.Rect(0, 0, size, size))

	// Copy the centered square region
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			cropped.Set(x, y, img.At(bounds.Min.X+offsetX+x, bounds.Min.Y+offsetY+y))
		}
	}

	return cropped
}

// FetchPhotoFromURL fetches a photo from a URL with SSRF protection
// Returns the photo data, media type, and any error
func FetchPhotoFromURL(photoURL string) ([]byte, string, error) {
	return httputil.FetchImageFromURL(photoURL)
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
// Returns: photoData (bytes), mediaType (string), photoURL (string if URL-based photo)
func extractPhotoData(field *vcard.Field) ([]byte, string, string) {
	if field == nil || field.Value == "" {
		return nil, "", ""
	}

	value := field.Value
	mediaType := ""

	// Check for MEDIATYPE or TYPE parameter
	if mt := field.Params.Get("MEDIATYPE"); mt != "" {
		mediaType = mt
	} else if t := field.Params.Get("TYPE"); t != "" {
		mediaType = "image/" + strings.ToLower(t)
	}

	// Check if it's a URL (http/https)
	// Google VCF format may have spaces in URLs that need to be removed
	cleanValue := strings.ReplaceAll(value, " ", "")
	cleanValue = strings.ReplaceAll(cleanValue, "\n", "")
	cleanValue = strings.ReplaceAll(cleanValue, "\r", "")

	if strings.HasPrefix(cleanValue, "http://") || strings.HasPrefix(cleanValue, "https://") {
		// It's a URL-based photo, return the URL for later fetching
		return nil, mediaType, cleanValue
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
		return nil, "", ""
	}

	return data, mediaType, ""
}

// generateUID creates a UID for a contact
func generateUID() string {
	return uuid.New().String()
}

// typeTokens maps our internal lowercase type token to vCard TYPE param tokens.
func typeTokens(t string) []string {
	up := strings.ToUpper(strings.TrimSpace(t))
	if up == "" {
		return nil
	}
	switch up {
	case "CELL", "MOBILE":
		return []string{"CELL", "VOICE"}
	default:
		return []string{up}
	}
}

// typeParams builds a vCard Params containing the TYPE for a value, or nil if untyped.
func typeParams(t string) vcard.Params {
	tokens := typeTokens(t)
	if len(tokens) == 0 {
		return nil
	}
	return vcard.Params{vcard.ParamType: tokens}
}

// emailParams builds TYPE params for an EMAIL, always including INTERNET (vCard 3.0).
func emailParams(t string) vcard.Params {
	tokens := append([]string{"INTERNET"}, typeTokens(t)...)
	return vcard.Params{vcard.ParamType: tokens}
}

// typeFromField extracts our internal lowercase type token from a vCard field's
// TYPE params, ignoring transport/preference markers.
func typeFromField(field *vcard.Field) string {
	if field == nil {
		return ""
	}
	for _, raw := range field.Params[vcard.ParamType] {
		for _, t := range strings.Split(raw, ",") {
			u := strings.ToUpper(strings.TrimSpace(t))
			switch u {
			case "", "INTERNET", "VOICE", "PREF":
				continue
			case "CELL", "MOBILE":
				return "cell"
			default:
				return strings.ToLower(u)
			}
		}
	}
	return ""
}

// isEmptyAddress reports whether a structured address has no content.
func isEmptyAddress(a models.ContactAddress) bool {
	return strings.TrimSpace(a.Street+a.City+a.Region+a.Postal+a.Country) == ""
}

// escapeComponent escapes a single value before it is joined into a vCard
// structured value (e.g. ORG or ADR) with ";" separators. go-vcard's encoder only
// escapes "\", "\n" and "," (not the structured ";" separator, per its formatValue),
// so we escape "\" and ";" ourselves; splitComponents reverses it on the way back.
// "\" must be escaped first so the backslash we add for ";" is not doubled.
func escapeComponent(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	return s
}

// splitComponents splits a vCard structured value on its unescaped ";" separators
// and unescapes each component, reversing escapeComponent. By the time it runs,
// go-vcard's decoder has already applied its own value-level unescaping (\\, \n, \,),
// so the only escape left to honor here is the "\;" we emit for embedded semicolons.
func splitComponents(s string) []string {
	var parts []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			cur.WriteByte(s[i+1])
			i++
			continue
		}
		if s[i] == ';' {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(s[i])
	}
	return append(parts, cur.String())
}

// component returns the i-th element of a structured value, or "" when absent.
func component(comps []string, i int) string {
	if i < len(comps) {
		return comps[i]
	}
	return ""
}

// nonEmpty returns the non-empty strings from the provided values, preserving order.
func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
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

// mappedVCardFields lists the vCard property names that map to dedicated Contact
// columns. Such properties must not also be stored in / restored from vcard_extra,
// otherwise they would be emitted twice on export.
func mappedVCardFields() map[string]bool {
	return map[string]bool{
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
		vcard.FieldURL:           true,
		vcard.FieldIMPP:          true,
		vcard.FieldTitle:         true,
		vcard.FieldRole:          true,
		vcard.FieldAnniversary:   true,
	}
}

// extractUnmappedProperties extracts vCard properties not mapped to Contact fields
func extractUnmappedProperties(card vcard.Card) VCardExtra {
	mappedFields := mappedVCardFields()

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
