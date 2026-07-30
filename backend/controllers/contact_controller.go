package controllers

import (
	"errors"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/logger"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"mycorrhizal/services"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// contactSummaryColumns is the fixed set of columns needed to build a
// ContactSummary (list-view) response. Selecting only these avoids the
// over-fetch (heavy JSON columns like card/emails/phones/addresses/...) that
// the removed fields= param used to exist to let callers opt out of (Gap 3
// in docs/fork-plan/50-integration-and-rebrand.md WP-71) — now that the list
// endpoint has a fixed slim shape, this is baked in rather than
// caller-configurable.
var contactSummaryColumns = []string{
	"id", "vcard_uid", "firstname", "lastname", "fn", "email", "phone", "birthday", "org",
	"photo", "photo_thumbnail", "archived",
}

func CreateContact(c *gin.Context) {
	// Save to the database
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get validated input from validation middleware. Per WP-71 item 3, this
	// is the new nested Card/CRM shape (models.ContactRecordInput), not the
	// old flat models.ContactInput.
	input, err := middleware.GetValidated[models.ContactRecordInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	contact := models.Contact{UserID: userID, Gender: input.Gender}
	models.ApplyRecordToContact(&contact, input.ToRecord(), currentConfig(c).ProfilePhotoDir)

	// Firstname is checked here (not via a struct tag on the nested input —
	// see ContactRecordInput's doc comment for why) because it's an
	// app-level invariant on Contact itself (existing `validate:"required"`
	// tag on models.Contact.Firstname, previously enforced indirectly by the
	// old flat ContactInput requiring it directly) that the new nested shape
	// has no simple field-level equivalent for: "at least one given-name
	// component, or a Name.Full, is present" is a derived condition, not a
	// single field.
	if contact.Firstname == "" {
		apperrors.AbortWithError(c, apperrors.ErrValidation("Request validation failed").WithDetails("card.name", "at least one name component (kind=given) or name.full is required"))
		return
	}

	if err := db.Create(&contact).Error; err != nil {
		logger.FromContext(c).Error().Err(err).Msg("Error saving contact to database")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to save contact").WithError(err))
		return
	}

	go services.TriggerWebhooks(db, currentConfig(c), userID, "contact.created", contact)
	c.JSON(http.StatusCreated, gin.H{"message": "Contact created successfully", "contact": models.NewContactRecordResponse(&contact, currentConfig(c).ProfilePhotoDir, db)})
}

// filters a contacts query by a free-text term
func applyContactSearch(query *gorm.DB, searchTerm string) *gorm.DB {
	like := "%" + searchTerm + "%"
	return query.Where(
		"firstname LIKE ? OR lastname LIKE ? OR nickname LIKE ? "+
			"OR (firstname || ' ' || lastname) LIKE ? OR (nickname || ' ' || lastname) LIKE ? "+
			"OR email LIKE ? OR phone LIKE ? "+
			"OR (json_valid(emails) AND EXISTS (SELECT 1 FROM json_each(contacts.emails) WHERE json_extract(json_each.value, '$.value') LIKE ?)) "+
			"OR (json_valid(phones) AND EXISTS (SELECT 1 FROM json_each(contacts.phones) WHERE json_extract(json_each.value, '$.value') LIKE ?))",
		like, like, like, like, like, like, like, like, like,
	)
}

func GetContacts(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)

	// NOTE: the fields= partial-projection param is gone (Gap 3 in
	// docs/fork-plan/50-integration-and-rebrand.md WP-71) — deliberately, not
	// an oversight. It is simply no longer read; a request that still sends
	// it is not rejected, it just has no effect. The fixed ContactSummary
	// shape (below) now serves the reason fields= existed (avoiding
	// over-fetch on a list view) — see contactSummaryColumns.

	// Parse relationships to include with validation. Per Gap 2, this
	// mechanic is preserved exactly as-is; only the per-item shape changes
	// (ContactSummary, extended to ContactSummaryWithRelations when any
	// includes= relation is requested).
	var relationshipMap = map[string]bool{
		"notes":         false,
		"activities":    false,
		"relationships": false,
		"reminders":     false,
	}
	includes := c.Query("includes")
	includesRequested := false
	for _, rel := range strings.Split(includes, ",") {
		if _, exists := relationshipMap[rel]; exists {
			relationshipMap[rel] = true
			includesRequested = true
		}
	}

	// Parse and validate sort parameters
	sortField := c.DefaultQuery("sort", "id")
	sortOrder := c.DefaultQuery("order", "desc")

	allowedSortFields := map[string]bool{"firstname": true, "lastname": true, "id": true, "random": true}
	if !allowedSortFields[sortField] {
		sortField = "id"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Parse archive filtering parameters
	includeArchived := c.Query("include_archived") == "true"
	archivedOnly := c.Query("archived") == "true"

	var contacts []models.Contact
	query := db.Model(&models.Contact{}).Where("user_id = ?", userID).
		Select(contactSummaryColumns).
		Limit(pagination.Limit).Offset(pagination.Offset)

	// Apply archive filtering
	if !includeArchived {
		if archivedOnly {
			query = query.Where("archived = ?", true)
		} else {
			query = query.Where("archived = ?", false)
		}
	}

	// Apply ordering - random uses RANDOM() function, others use column name
	// For search with include_archived, order non-archived first
	if includeArchived && c.Query("search") != "" {
		query = query.Order("archived ASC")
	}
	if sortField == "random" {
		query = query.Order("RANDOM()")
	} else {
		query = query.Order(sortField + " " + sortOrder)
	}

	// Apply search filter using parameterization
	if searchTerm := c.Query("search"); searchTerm != "" {
		query = applyContactSearch(query, searchTerm)
	}

	if circle := c.Query("circle"); circle != "" {
		query = query.Where("EXISTS (SELECT 1 FROM json_each(contacts.circles) WHERE json_each.value = ?)", circle)
	}

	// Preload requested relationships
	for rel, include := range relationshipMap {
		if include {
			switch rel {
			case "notes":
				query = query.Preload("Notes", "notes.user_id = ?", userID)
			case "activities":
				query = query.Preload("Activities", "activities.user_id = ?", userID)
			case "relationships":
				query = query.Preload("Relationships", "relationships.user_id = ?", userID)
			case "reminders":
				query = query.Preload("Reminders", "reminders.user_id = ?", userID)
			}
		}
	}

	// Execute query
	if err := query.Find(&contacts).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contacts").WithError(err))
		return
	}

	var total int64
	countQuery := db.Model(&models.Contact{}).Where("user_id = ?", userID)

	// Apply the same archive filter to the count query
	if !includeArchived {
		if archivedOnly {
			countQuery = countQuery.Where("archived = ?", true)
		} else {
			countQuery = countQuery.Where("archived = ?", false)
		}
	}

	// Apply the same search filters to the count query
	if searchTerm := c.Query("search"); searchTerm != "" {
		countQuery = applyContactSearch(countQuery, searchTerm)
	}

	if circle := c.Query("circle"); circle != "" {
		countQuery = countQuery.Where("EXISTS (SELECT 1 FROM json_each(contacts.circles) WHERE json_each.value = ?)", circle)
	}

	countQuery.Count(&total)

	// Map contacts to the slim ContactSummary shape (Gap 2/3): plain
	// ContactSummary normally, or ContactSummaryWithRelations when includes=
	// requested at least one relation — never the full Card, per WP-71's
	// binding "list returns []ContactSummary, not the full Card" rule.
	if includesRequested {
		items := make([]models.ContactSummaryWithRelations, len(contacts))
		for i := range contacts {
			items[i] = models.NewContactSummaryWithRelations(&contacts[i])
		}
		c.JSON(http.StatusOK, gin.H{
			"contacts": items,
			"total":    total,
			"page":     pagination.Page,
			"limit":    pagination.Limit,
		})
		return
	}

	items := make([]models.ContactSummary, len(contacts))
	for i := range contacts {
		items[i] = models.NewContactSummary(&contacts[i])
	}
	c.JSON(http.StatusOK, gin.H{
		"contacts": items,
		"total":    total,
		"page":     pagination.Page,
		"limit":    pagination.Limit,
	})
}

func GetContactsRandom(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var selectedFields = []string{"ID", "firstname", "lastname", "nickname", "circles", "photo_thumbnail"}

	var contacts []models.Contact
	query := db.Model(&models.Contact{}).Where("user_id = ?", userID).Where("archived = ?", false)

	if len(selectedFields) > 0 {
		query = query.Select(selectedFields)
	}

	// Get 5 random contacts
	query = query.Order("RANDOM()").Limit(5)

	// Execute query
	if err := query.Find(&contacts).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contacts").WithError(err))
		return
	}

	// Map to response with photo thumbnail
	contactResponses := make([]models.ContactResponse, len(contacts))
	for i, contact := range contacts {
		contactResponses[i] = models.ContactResponse{
			Contact:        contact,
			PhotoThumbnail: contact.PhotoThumbnail,
		}
	}

	// Respond with random contacts
	c.JSON(http.StatusOK, gin.H{
		"contacts": contactResponses,
	})
}

func GetUpcomingBirthdays(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	birthdays, err := services.GetUpcomingBirthdays(db, userID, time.Now())
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve upcoming birthdays").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"birthdays": birthdays,
	})
}

func GetContact(c *gin.Context) {
	id := c.Param("id")

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	// NOTE: fields= is gone here too (Gap 3) — the detail endpoint now always
	// returns the full neutral Record/Card (ContactRecordResponse), which is
	// what fields= partial-fetching existed to approximate a slice of.
	//
	// Preload behavior: kept exactly as the pre-WP-71 "no fields=" branch
	// (always preload all four associations) — dedicated endpoints like
	// GET /contacts/:id/notes already exist and may make this redundant for
	// some callers, but changing that is a separate, larger decision this WP
	// doesn't need to make; preserving existing behavior here is the safer
	// default for backward compat.
	var contact models.Contact
	query := db.Where("user_id = ?", userID).
		Preload("Notes", "notes.user_id = ?", userID).
		Preload("Activities", "activities.user_id = ?", userID).
		Preload("Relationships", "relationships.user_id = ?", userID).
		Preload("Reminders", "reminders.user_id = ?", userID)

	if err := query.First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}
	c.JSON(http.StatusOK, models.NewContactRecordResponse(&contact, currentConfig(c).ProfilePhotoDir, db))
}

func UpdateContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Get validated input from validation middleware (new nested shape, see
	// CreateContact's comment).
	input, err := middleware.GetValidated[models.ContactRecordInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	contact.Gender = input.Gender
	models.ApplyRecordToContact(&contact, input.ToRecord(), currentConfig(c).ProfilePhotoDir)

	if contact.Firstname == "" {
		apperrors.AbortWithError(c, apperrors.ErrValidation("Request validation failed").WithDetails("card.name", "at least one name component (kind=given) or name.full is required"))
		return
	}

	if err := db.Save(&contact).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to update contact").WithError(err))
		return
	}

	go services.TriggerWebhooks(db, currentConfig(c), userID, "contact.updated", contact)
	c.JSON(http.StatusOK, models.NewContactRecordResponse(&contact, currentConfig(c).ProfilePhotoDir, db))
}

func DeleteContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Check if contact exists first
	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Start a transaction to ensure all deletes succeed together
	err := db.Transaction(func(tx *gorm.DB) error {
		// Manually delete associated reminders (soft delete doesn't trigger CASCADE)
		if err := tx.Where("contact_id = ? AND user_id = ?", id, userID).Delete(&models.Reminder{}).Error; err != nil {
			return err
		}

		// Manually delete associated notes
		if err := tx.Where("contact_id = ? AND user_id = ?", id, userID).Delete(&models.Note{}).Error; err != nil {
			return err
		}

		// Manually delete associated relationships
		if err := tx.Where("contact_id = ? AND user_id = ?", id, userID).Delete(&models.Relationship{}).Error; err != nil {
			return err
		}

		// Delete activity associations (many-to-many)
		if err := tx.Exec("DELETE FROM activity_contacts WHERE contact_id = ? AND activity_id IN (SELECT id FROM activities WHERE user_id = ?)", id, userID).Error; err != nil {
			return err
		}

		// Delete relationship-graph edges referencing this contact (source or target)
		if err := tx.Where("(source_id = ? OR target_id = ?) AND user_id = ?", contact.VCardUID, contact.VCardUID, userID).Delete(&models.RelationshipEdge{}).Error; err != nil {
			return err
		}

		// Delete this contact's household/circle memberships and tags (not the
		// household/circle/tag containers themselves -- other contacts may still
		// belong to them)
		if err := tx.Where("member_vcard_uid = ? AND user_id = ?", contact.VCardUID, userID).Delete(&models.HouseholdMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("member_vcard_uid = ? AND user_id = ?", contact.VCardUID, userID).Delete(&models.CircleMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("contact_vcard_uid = ? AND user_id = ?", contact.VCardUID, userID).Delete(&models.ContactTag{}).Error; err != nil {
			return err
		}

		// Delete this contact's life events and custom field values
		if err := tx.Where("entity_id = ? AND user_id = ?", contact.VCardUID, userID).Delete(&models.LifeEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("entity_id = ? AND user_id = ?", contact.VCardUID, userID).Delete(&models.FieldValue{}).Error; err != nil {
			return err
		}

		// Delete CardDAV contact sync links (a genuine Contact.ID FK, unlike the
		// VCardUID-based references above)
		if err := tx.Where("contact_id = ? AND user_id = ?", id, userID).Delete(&models.ContactSyncLink{}).Error; err != nil {
			return err
		}

		// Finally, delete the contact
		if err := tx.Delete(&contact).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete contact and associated data").WithError(err))
		return
	}

	// Cleanup profile photos after successful database transaction
	// This is done outside the transaction since file deletion cannot be rolled back
	deleteContactPhotos(c, contact)

	go services.TriggerWebhooks(db, currentConfig(c), userID, "contact.deleted", gin.H{"id": contact.ID})
	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted"})
}

// deleteContactPhotos removes the profile photo file for a contact
// Note: thumbnails are stored as base64 in the database, not as files
func deleteContactPhotos(c *gin.Context, contact models.Contact) {
	uploadDir := os.Getenv("PROFILE_PHOTO_DIR")
	if uploadDir == "" {
		return
	}

	log := logger.FromContext(c)

	// Delete main photo if it exists
	if contact.Photo != "" {
		photoPath := filepath.Join(uploadDir, contact.Photo)
		if err := os.Remove(photoPath); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", photoPath).Msg("Failed to delete contact photo")
		} else if err == nil {
			log.Debug().Str("path", photoPath).Msg("Deleted contact photo")
		}
	}

	// Delete legacy file-based thumbnail if it exists (not base64 data URL)
	if contact.PhotoThumbnail != "" && !strings.HasPrefix(contact.PhotoThumbnail, "data:") {
		thumbnailPath := filepath.Join(uploadDir, contact.PhotoThumbnail)
		if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", thumbnailPath).Msg("Failed to delete contact thumbnail")
		} else if err == nil {
			log.Debug().Str("path", thumbnailPath).Msg("Deleted contact thumbnail")
		}
	}
}

// GetCircles returns all unique circles associated with contacts.
func GetCircles(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var circleNames []string

	// Raw SQL query to extract unique circle names
	err := db.Raw(`SELECT DISTINCT json_each.value AS circle
	               FROM contacts, json_each(contacts.circles)
	               WHERE contacts.user_id = ?`, userID).Scan(&circleNames).Error
	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve circles").WithError(err))
		return
	}

	// Return the list of unique circle names
	c.JSON(http.StatusOK, circleNames)
}

// ArchiveContact archives a contact and deletes all its reminders
func ArchiveContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	// Archive contact and delete reminders in a transaction
	err := db.Transaction(func(tx *gorm.DB) error {
		// Delete all reminders for this contact
		if err := tx.Where("contact_id = ? AND user_id = ?", id, userID).Delete(&models.Reminder{}).Error; err != nil {
			return err
		}

		// Set archived to true
		if err := tx.Model(&contact).Update("archived", true).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to archive contact").WithError(err))
		return
	}

	contact.Archived = true
	c.JSON(http.StatusOK, contact)
}

// UnarchiveContact restores an archived contact
func UnarchiveContact(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var contact models.Contact
	if err := db.Where("user_id = ?", userID).First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Contact").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve contact").WithError(err))
		}
		return
	}

	if err := db.Model(&contact).Update("archived", false).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to unarchive contact").WithError(err))
		return
	}

	contact.Archived = false
	c.JSON(http.StatusOK, contact)
}
