package carddav

import (
	"encoding/json"
	"fmt"
	"meerkat/models"
	"meerkat/photostore"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"
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

	// Apple-style custom labels (X-ABLabel) are emitted in their own property
	// group (item1, item2, ...). Allocate group names lazily as needed.
	groupCounter := 0
	nextGroup := func() string {
		groupCounter++
		return fmt.Sprintf("item%d", groupCounter)
	}

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
		// EMAIL always carries INTERNET (vCard 3.0) regardless of label.
		addTypedField(card, vcard.FieldEmail, &vcard.Field{Value: e.Value}, e.Type, nextGroup, "INTERNET")
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
		addTypedField(card, vcard.FieldTelephone, &vcard.Field{Value: p.Value}, p.Type, nextGroup)
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
		addTypedField(card, vcard.FieldAddress, &vcard.Field{Value: strings.Join(comps, ";")}, a.Type, nextGroup)
	}

	// URL (websites)
	for _, u := range contact.URLs {
		if u.Value == "" {
			continue
		}
		addTypedField(card, vcard.FieldURL, &vcard.Field{Value: u.Value}, u.Type, nextGroup)
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
	photoData, mediaType := photostore.ReadContactPhoto(contact.Photo, contact.PhotoThumbnail, photoDir)
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
				Type:  typeForField(card, f),
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
				Type:  typeForField(card, f),
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
				Type:    typeForField(card, f),
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
				Type:  typeForField(card, f),
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
		photoData, photoMediaType, photoURL = photostore.ExtractPhotoData(photoField)
	}

	// Store unmapped properties in VCardExtra
	extra := extractUnmappedProperties(card)
	if len(extra.Properties) > 0 {
		extraJSON, _ := json.Marshal(extra)
		contact.VCardExtra = string(extraJSON)
	}

	return contact, photoData, photoMediaType, photoURL
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

// fieldABLabel is the (non-standard but ubiquitous) property Apple Contacts and
// most CardDAV clients use to carry a human-readable custom label for a value,
// linked to that value via a shared property group (e.g. item1.X-ABLabel).
const fieldABLabel = "X-ABLabel"

// standardTypeTokens are the type values we round-trip as a vCard TYPE parameter.
// Any other (user-defined) label is emitted as a grouped X-ABLabel instead, which
// is how most CardDAV clients represent custom labels (RFC 2426).
var standardTypeTokens = map[string]bool{
	"home":   true,
	"work":   true,
	"cell":   true,
	"mobile": true,
	"fax":    true,
	"voice":  true,
	"pager":  true,
	"other":  true,
}

// reports whether t is a user-defined label rather than one of the standard vCard TYPE tokens.
func isCustomType(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return t != "" && !standardTypeTokens[t]
}

// addTypedField adds field to the card under fieldName, encoding the internal
// type token. Standard tokens become a vCard TYPE parameter; a custom label is
// emitted as a grouped X-ABLabel (Apple convention) so it survives round-trips
// and renders as a named label. baseTypeTokens are TYPE values
// always present regardless of the label (e.g. INTERNET for EMAIL).
func addTypedField(card vcard.Card, fieldName string, field *vcard.Field, t string, nextGroup func() string, baseTypeTokens ...string) {
	if isCustomType(t) {
		group := nextGroup()
		field.Group = group
		if len(baseTypeTokens) > 0 {
			field.Params = vcard.Params{vcard.ParamType: append([]string(nil), baseTypeTokens...)}
		}
		card.Add(fieldName, field)
		card.Add(fieldABLabel, &vcard.Field{Group: group, Value: strings.TrimSpace(t)})
		return
	}
	tokens := append(append([]string(nil), baseTypeTokens...), typeTokens(t)...)
	if len(tokens) > 0 {
		field.Params = vcard.Params{vcard.ParamType: tokens}
	}
	card.Add(fieldName, field)
}

// typeForField resolves the internal type token for an imported field, preferring
// a grouped X-ABLabel (custom or Apple pseudo-label) over the TYPE parameter.
func typeForField(card vcard.Card, field *vcard.Field) string {
	if label := labelFromGroup(card, field); label != "" {
		return normalizeLabel(label)
	}
	return typeFromField(field)
}

// labelFromGroup returns the X-ABLabel value attached to the field's property group.
func labelFromGroup(card vcard.Card, field *vcard.Field) string {
	if field.Group == "" {
		return ""
	}
	for _, lf := range card[fieldABLabel] {
		if strings.EqualFold(lf.Group, field.Group) {
			return strings.TrimSpace(lf.Value)
		}
	}
	return ""
}

// normalizeLabel maps Apple's pseudo-labels (e.g. "_$!<Home>!$_") back to our
// standard tokens; any other label is returned unchanged.
func normalizeLabel(label string) string {
	if strings.HasPrefix(label, "_$!<") && strings.HasSuffix(label, ">!$_") {
		inner := strings.ToLower(label[len("_$!<") : len(label)-len(">!$_")])
		if inner == "mobile" {
			return "cell"
		}
		return inner
	}
	return label
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
		fieldABLabel:             true,
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
