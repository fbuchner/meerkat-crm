package controllers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mycorrhizal/contactmodel"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/jscontact"
	"mycorrhizal/logger"
	"mycorrhizal/models"
	"mycorrhizal/vcard3"
	"mycorrhizal/vcard4"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// csvFormulaLeaders are the characters spreadsheet applications treat as the
// start of a formula rather than text. Tab and carriage return are included
// because Excel strips leading whitespace before deciding.
const csvFormulaLeaders = "=+-@\t\r"

// csvSafe neutralizes spreadsheet formula injection.
//
// encoding/csv quotes delimiters and newlines, which makes the file parse
// correctly, but quoting does nothing about a value like
// `=HYPERLINK("http://attacker","click")` — Excel and LibreOffice still
// evaluate it on open. Contact data is not all self-authored: it arrives from
// CardDAV sync and VCF/CSV import, so a field can carry a payload chosen by
// someone else and fire when the user exports and opens the result.
//
// Prefixing with a single quote is the conventional fix: spreadsheets consume
// it as a "treat as text" marker, and any consumer that is not a spreadsheet
// sees one extra leading character rather than executable content.
func csvSafe(value string) string {
	if value == "" {
		return value
	}
	if strings.ContainsRune(csvFormulaLeaders, rune(value[0])) {
		return "'" + value
	}
	return value
}

// csvSafeRecord applies csvSafe to every field of a record.
func csvSafeRecord(record []string) []string {
	for i, field := range record {
		record[i] = csvSafe(field)
	}
	return record
}

// ExportData exports all user data as CSV files in a combined format
func ExportData(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Fetch user to get custom field names
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch user for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch user"))
		return
	}
	customFieldNames := user.CustomFieldNames
	if customFieldNames == nil {
		customFieldNames = []string{}
	}

	// Fetch all user data
	var contacts []models.Contact
	if err := db.Where("user_id = ?", userID).
		Order("firstname ASC, lastname ASC").
		Find(&contacts).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch contacts for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch contacts"))
		return
	}

	// §3d WP4: relationships now come from RelationshipEdge, not the legacy
	// models.Relationship table. Names are resolved via a VCardUID map built
	// from the contacts already fetched above, since an edge only carries
	// its endpoints' VCardUID, not a nested contact.
	var relationshipEdges []models.RelationshipEdge
	if err := db.Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&relationshipEdges).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch relationship edges for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch relationship edges"))
		return
	}
	type contactRef struct {
		ID   uint
		Name string
	}
	contactByVCardUID := make(map[string]contactRef, len(contacts))
	for _, contact := range contacts {
		contactByVCardUID[contact.VCardUID] = contactRef{
			ID:   contact.ID,
			Name: fmt.Sprintf("%s %s", contact.Firstname, contact.Lastname),
		}
	}

	var activities []models.Activity
	if err := db.Where("user_id = ?", userID).
		Preload("Contacts").
		Order("date DESC").
		Find(&activities).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch activities for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch activities"))
		return
	}

	var notes []models.Note
	if err := db.Where("user_id = ?", userID).
		Preload("Contact").
		Order("date DESC").
		Find(&notes).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch notes for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch notes"))
		return
	}

	var reminders []models.Reminder
	if err := db.Where("user_id = ?", userID).
		Preload("Contact").
		Order("remind_at ASC").
		Find(&reminders).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch reminders for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch reminders"))
		return
	}

	// T20a: the "Food Preference" column now sources from the structured
	// preferences table (category=food) rather than the retired free-text
	// Contact.FoodPreference. Deliberately includes every sensitivity — this
	// is the user's own full personal-data backup, the same choice the
	// relationships section below documents for RelationshipEdge.
	var foodPreferences []models.Preference
	if err := db.Where("user_id = ? AND category = ?", userID, models.PreferenceCategoryFood).
		Find(&foodPreferences).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch food preferences for export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch food preferences"))
		return
	}
	foodByVCardUID := make(map[string][]string, len(foodPreferences))
	for _, pref := range foodPreferences {
		foodByVCardUID[pref.EntityID] = append(foodByVCardUID[pref.EntityID], pref.Value)
	}

	// Generate combined CSV content
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write contacts section
	buf.WriteString("=== CONTACTS ===\n")
	writer.Flush()

	contactHeaders := []string{
		"ID", "Firstname", "Lastname", "Nickname", "Gender", "Email", "Phone",
		"Birthday", "Address", "How We Met", "Food Preference", "Work Information",
		"Contact Information", "Circles", "Created At", "Updated At",
	}
	// Add custom field names as additional headers. These are user-defined, so
	// the header row needs the same treatment as the data rows.
	contactHeaders = append(contactHeaders, customFieldNames...)
	if err := writer.Write(csvSafeRecord(contactHeaders)); err != nil {
		log.Error().Err(err).Msg("Failed to write contact headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, contact := range contacts {
		record := []string{
			fmt.Sprintf("%d", contact.ID),
			contact.Firstname,
			contact.Lastname,
			contact.Nickname,
			contact.Gender,
			contact.Email,
			contact.Phone,
			contact.Birthday,
			contact.Address,
			contact.HowWeMet,
			strings.Join(foodByVCardUID[contact.VCardUID], "; "),
			contact.WorkInformation,
			contact.ContactInformation,
			strings.Join(contact.Circles, "; "),
			contact.CreatedAt.Format(time.RFC3339),
			contact.UpdatedAt.Format(time.RFC3339),
		}
		// Add custom field values
		for _, fieldName := range customFieldNames {
			value := ""
			if contact.CustomFields != nil {
				value = contact.CustomFields[fieldName]
			}
			record = append(record, value)
		}
		if err := writer.Write(csvSafeRecord(record)); err != nil {
			log.Error().Err(err).Msg("Failed to write contact record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Write relationships section. §3d WP4: reads RelationshipEdge, not the
	// legacy models.Relationship table. Deliberately includes every
	// status/sensitivity — this is the user's own full personal-data backup,
	// not a share to another party, so unlike RecordForContact's vCard/
	// JSContact export path (which filters non-normal sensitivity because
	// that projection can leave the instance), nothing here is held back.
	buf.WriteString("\n=== RELATIONSHIPS ===\n")

	relationshipHeaders := []string{
		"ID", "Source Contact ID", "Source Contact Name", "Type", "Target Contact ID",
		"Target Contact Name", "Status", "Sensitivity", "Source", "Created At", "Updated At",
	}
	if err := writer.Write(relationshipHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write relationship headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, edge := range relationshipEdges {
		sourceContactID := edge.SourceID
		sourceContactName := edge.SourceID
		if ref, ok := contactByVCardUID[edge.SourceID]; ok {
			sourceContactID = fmt.Sprintf("%d", ref.ID)
			sourceContactName = ref.Name
		}
		targetContactID := edge.TargetID
		targetContactName := edge.TargetID
		if ref, ok := contactByVCardUID[edge.TargetID]; ok {
			targetContactID = fmt.Sprintf("%d", ref.ID)
			targetContactName = ref.Name
		}

		record := []string{
			edge.ID,
			sourceContactID,
			sourceContactName,
			edge.Type,
			targetContactID,
			targetContactName,
			edge.Status,
			edge.Sensitivity,
			edge.Source,
			edge.CreatedAt.Format(time.RFC3339),
			edge.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(csvSafeRecord(record)); err != nil {
			log.Error().Err(err).Msg("Failed to write relationship record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Write activities section
	buf.WriteString("\n=== ACTIVITIES ===\n")

	activityHeaders := []string{
		"ID", "Title", "Description", "Location", "Date", "Contact Names", "Created At", "Updated At",
	}
	if err := writer.Write(activityHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write activity headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, activity := range activities {
		contactNames := make([]string, len(activity.Contacts))
		for i, contact := range activity.Contacts {
			contactNames[i] = fmt.Sprintf("%s %s", contact.Firstname, contact.Lastname)
		}
		record := []string{
			fmt.Sprintf("%d", activity.ID),
			activity.Title,
			activity.Description,
			activity.Location,
			activity.Date.Format(time.RFC3339),
			strings.Join(contactNames, "; "),
			activity.CreatedAt.Format(time.RFC3339),
			activity.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(csvSafeRecord(record)); err != nil {
			log.Error().Err(err).Msg("Failed to write activity record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Write notes section
	buf.WriteString("\n=== NOTES ===\n")

	noteHeaders := []string{
		"ID", "Contact ID", "Contact Name", "Content", "Date", "Created At", "Updated At",
	}
	if err := writer.Write(noteHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write note headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, note := range notes {
		contactID := ""
		contactName := ""
		if note.ContactID != nil {
			contactID = fmt.Sprintf("%d", *note.ContactID)
			contactName = fmt.Sprintf("%s %s", note.Contact.Firstname, note.Contact.Lastname)
		}
		record := []string{
			fmt.Sprintf("%d", note.ID),
			contactID,
			contactName,
			note.Content,
			note.Date.Format(time.RFC3339),
			note.CreatedAt.Format(time.RFC3339),
			note.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(csvSafeRecord(record)); err != nil {
			log.Error().Err(err).Msg("Failed to write note record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Write reminders section
	buf.WriteString("\n=== REMINDERS ===\n")

	reminderHeaders := []string{
		"ID", "Contact ID", "Contact Name", "Message", "Remind At", "Recurrence",
		"By Mail", "Reoccur From Completion", "Completed", "Last Sent", "Created At", "Updated At",
	}
	if err := writer.Write(reminderHeaders); err != nil {
		log.Error().Err(err).Msg("Failed to write reminder headers")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	for _, reminder := range reminders {
		contactID := ""
		contactName := ""
		if reminder.ContactID != nil {
			contactID = fmt.Sprintf("%d", *reminder.ContactID)
			contactName = fmt.Sprintf("%s %s", reminder.Contact.Firstname, reminder.Contact.Lastname)
		}

		byMail := "false"
		if reminder.ByMail != nil && *reminder.ByMail {
			byMail = "true"
		}

		reoccurFromCompletion := "true"
		if reminder.ReoccurFromCompletion != nil && !*reminder.ReoccurFromCompletion {
			reoccurFromCompletion = "false"
		}

		lastSent := ""
		if reminder.LastSent != nil {
			lastSent = reminder.LastSent.Format(time.RFC3339)
		}

		record := []string{
			fmt.Sprintf("%d", reminder.ID),
			contactID,
			contactName,
			reminder.Message,
			reminder.RemindAt.Format(time.RFC3339),
			reminder.Recurrence,
			byMail,
			reoccurFromCompletion,
			fmt.Sprintf("%t", reminder.Completed),
			lastSent,
			reminder.CreatedAt.Format(time.RFC3339),
			reminder.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(csvSafeRecord(record)); err != nil {
			log.Error().Err(err).Msg("Failed to write reminder record")
			apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
			return
		}
	}
	writer.Flush()

	// Check for any CSV writer errors
	if err := writer.Error(); err != nil {
		log.Error().Err(err).Msg("CSV writer error")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("mycorrhizal-export-%s.csv", time.Now().Format("2006-01-02"))

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))

	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())

	log.Info().
		Int("contacts", len(contacts)).
		Int("activities", len(activities)).
		Int("notes", len(notes)).
		Int("reminders", len(reminders)).
		Msg("Data export completed successfully")
}

// ExportContactsAsVCF exports all user contacts as a VCF (vCard) file.
//
// Per docs/fork-plan/50-integration-and-rebrand.md WP-71 Gap 4, this now
// routes through the vcard4/vcard3 adapters instead of the legacy
// carddav.ContactToVCard mapper. ?version=3 (or "3.0") selects vCard 3.0;
// anything else (including absent) defaults to 4.0, per the "advertise 4.0
// by default" precedent this WP sets.
//
// contactmodel.Record is built via models.RecordForContact, not
// RecordFromContact directly — the persisted contact.Card already carries
// any data with no flat-field home (SpeakToAs, PersonalInfo, ...); calling
// RecordFromContact fresh here would silently drop it from the export. See
// RecordForContact's doc comment; this was a real bug found and fixed
// across three call sites while auditing WP-73's work.
//
// photoDir (config.Config.ProfilePhotoDir, from routes.go's call site) is
// forwarded through RecordForContact: per
// docs/fork-plan/50-integration-and-rebrand.md WP-73's photo-bridging
// prerequisite, Contact.Photo/PhotoThumbnail bridges into a
// Card.Media{Kind:"photo"} entry, which the vcard4/vcard3 adapters encode
// as an embedded PHOTO property — closing WP-71's previously-documented
// "VCF export doesn't embed photos" gap.
func ExportContactsAsVCF(c *gin.Context, photoDir string) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Fetch all user contacts
	var contacts []models.Contact
	if err := db.Where("user_id = ?", userID).
		Order("firstname ASC, lastname ASC").
		Find(&contacts).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch contacts for VCF export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch contacts"))
		return
	}

	version := "4"
	var exporter contactmodel.Exporter = vcard4.Adapter{}
	if v := strings.TrimPrefix(c.Query("version"), "v"); v == "3" || v == "3.0" {
		version = "3"
		exporter = vcard3.Adapter{}
	}

	// Generate VCF content: one full vCard block per contact, concatenated
	// (the standard shape for a multi-contact .vcf file).
	var buf bytes.Buffer
	for _, contact := range contacts {
		record := models.RecordForContact(&contact, photoDir, db)
		data, diags, err := exporter.Export(record)
		if err != nil {
			log.Error().Err(err).Uint("contact_id", contact.ID).Msg("Failed to encode contact as vCard")
			// Continue with other contacts instead of failing completely
			continue
		}
		for _, d := range diags {
			log.Debug().Str("severity", d.Severity).Str("concept", d.Concept).Uint("contact_id", contact.ID).Msg(d.Message)
		}
		buf.Write(data)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("mycorrhizal-contacts-%s.vcf", time.Now().Format("2006-01-02"))

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/vcard; charset=utf-8")
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))

	c.Data(http.StatusOK, "text/vcard; charset=utf-8", buf.Bytes())

	log.Info().
		Int("contacts", len(contacts)).
		Str("version", version).
		Msg("VCF export completed successfully")
}

// ExportContactsAsJSContact exports all user contacts as a single JSON
// document: a JSON array of RFC 9553 JSContact Card objects (one per
// contact) — the "Card set" option from WP-71's task list, chosen over a
// single merged document since each contact is an independent Card with its
// own @type/uid, not sub-objects of one another.
//
// Reads photoDir from currentConfig(c).ProfilePhotoDir directly (rather than
// as an explicit parameter, unlike ExportContactsAsVCF) since this handler is
// registered directly in routes.go with the plain gin.HandlerFunc signature,
// not via a photoDir-carrying closure; this avoids a routes.go signature
// change for a one-line lookup this handler can already make itself.
func ExportContactsAsJSContact(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)
	photoDir := currentConfig(c).ProfilePhotoDir

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var contacts []models.Contact
	if err := db.Where("user_id = ?", userID).
		Order("firstname ASC, lastname ASC").
		Find(&contacts).Error; err != nil {
		log.Error().Err(err).Msg("Failed to fetch contacts for JSContact export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to fetch contacts"))
		return
	}

	adapter := jscontact.Adapter{}
	cards := make([]json.RawMessage, 0, len(contacts))
	for _, contact := range contacts {
		record := models.RecordForContact(&contact, photoDir, db)
		data, diags, err := adapter.Export(record)
		if err != nil {
			log.Error().Err(err).Uint("contact_id", contact.ID).Msg("Failed to encode contact as JSContact")
			continue
		}
		for _, d := range diags {
			log.Debug().Str("severity", d.Severity).Str("concept", d.Concept).Uint("contact_id", contact.ID).Msg(d.Message)
		}
		cards = append(cards, json.RawMessage(data))
	}

	payload, err := json.Marshal(cards)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal JSContact export")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to generate export"))
		return
	}

	filename := fmt.Sprintf("mycorrhizal-contacts-%s.jscontact.json", time.Now().Format("2006-01-02"))

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/jscontact+json; charset=utf-8")
	c.Header("Content-Length", fmt.Sprintf("%d", len(payload)))

	c.Data(http.StatusOK, "application/jscontact+json; charset=utf-8", payload)

	log.Info().
		Int("contacts", len(contacts)).
		Msg("JSContact export completed successfully")
}
