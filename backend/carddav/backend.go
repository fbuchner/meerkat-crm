package carddav

import (
	"bytes"
	"context"
	"fmt"
	"meerkat/logger"
	"meerkat/models"
	"net/http"
	"path"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	userIDKey   contextKey = "userID"
	usernameKey contextKey = "username"
	dbKey       contextKey = "db"
	photoDir    contextKey = "photoDir"
)

// Backend implements the carddav.Backend interface
type Backend struct {
	db       *gorm.DB
	photoDir string
}

// NewBackend creates a new CardDAV backend
func NewBackend(db *gorm.DB, photoDir string) *Backend {
	return &Backend{
		db:       db,
		photoDir: photoDir,
	}
}

// ContextWithUser adds user info to context for the backend
func ContextWithUser(ctx context.Context, userID uint, username string, db *gorm.DB, photoDirPath string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, usernameKey, username)
	ctx = context.WithValue(ctx, dbKey, db)
	ctx = context.WithValue(ctx, photoDir, photoDirPath)
	return ctx
}

func (b *Backend) getUserID(ctx context.Context) (uint, error) {
	userID, ok := ctx.Value(userIDKey).(uint)
	if !ok {
		return 0, fmt.Errorf("user not authenticated")
	}
	return userID, nil
}

func (b *Backend) getUsername(ctx context.Context) string {
	username, _ := ctx.Value(usernameKey).(string)
	return username
}

func (b *Backend) getDB(ctx context.Context) *gorm.DB {
	if db, ok := ctx.Value(dbKey).(*gorm.DB); ok {
		return db
	}
	return b.db
}

func (b *Backend) getPhotoDir(ctx context.Context) string {
	if dir, ok := ctx.Value(photoDir).(string); ok {
		return dir
	}
	return b.photoDir
}

// CurrentUserPrincipal returns the current user's principal URL
func (b *Backend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	username := b.getUsername(ctx)
	if username == "" {
		return "", fmt.Errorf("user not authenticated")
	}
	return "/carddav/principals/" + username + "/", nil
}

// AddressBookHomeSetPath returns the path to the address book home set
func (b *Backend) AddressBookHomeSetPath(ctx context.Context) (string, error) {
	username := b.getUsername(ctx)
	if username == "" {
		return "", fmt.Errorf("user not authenticated")
	}
	return "/carddav/addressbooks/" + username + "/", nil
}

// ListAddressBooks returns the list of address books for the current user
func (b *Backend) ListAddressBooks(ctx context.Context) ([]carddav.AddressBook, error) {
	username := b.getUsername(ctx)
	if username == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	return []carddav.AddressBook{
		{
			Path:        "/carddav/addressbooks/" + username + "/contacts/",
			Name:        "Contacts",
			Description: "Meerkat CRM Contacts",
			SupportedAddressData: []carddav.AddressDataType{
				{ContentType: "text/vcard", Version: "3.0"},
				{ContentType: "text/vcard", Version: "4.0"},
			},
		},
	}, nil
}

// GetAddressBook returns a specific address book
func (b *Backend) GetAddressBook(ctx context.Context, urlPath string) (*carddav.AddressBook, error) {
	username := b.getUsername(ctx)
	if username == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	expectedPath := "/carddav/addressbooks/" + username + "/contacts/"
	if urlPath != expectedPath && urlPath+"/" != expectedPath {
		return nil, fmt.Errorf("address book not found")
	}

	return &carddav.AddressBook{
		Path:        expectedPath,
		Name:        "Contacts",
		Description: "Meerkat CRM Contacts",
		SupportedAddressData: []carddav.AddressDataType{
			{ContentType: "text/vcard", Version: "3.0"},
			{ContentType: "text/vcard", Version: "4.0"},
		},
	}, nil
}

// CreateAddressBook creates a new address book (not supported - single address book per user)
func (b *Backend) CreateAddressBook(ctx context.Context, addressBook *carddav.AddressBook) error {
	return fmt.Errorf("creating address books is not supported")
}

// DeleteAddressBook deletes an address book (not supported)
func (b *Backend) DeleteAddressBook(ctx context.Context, urlPath string) error {
	return fmt.Errorf("deleting address books is not supported")
}

// GetAddressObject returns a single address object (contact)
func (b *Backend) GetAddressObject(ctx context.Context, urlPath string, req *carddav.AddressDataRequest) (*carddav.AddressObject, error) {
	userID, err := b.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Extract UID from path (e.g., /carddav/addressbooks/user/contacts/uid.vcf)
	uid := extractUIDFromPath(urlPath)
	if uid == "" {
		return nil, fmt.Errorf("invalid path")
	}

	db := b.getDB(ctx)
	var contact models.Contact

	// Try to find by vcard_uid first, then by ID
	err = db.Where("user_id = ? AND vcard_uid = ?", userID, uid).First(&contact).Error
	if err == gorm.ErrRecordNotFound {
		// Try parsing as numeric ID for backwards compatibility
		var id uint
		if _, scanErr := fmt.Sscanf(uid, "%d", &id); scanErr == nil {
			err = db.Where("user_id = ? AND id = ?", userID, id).First(&contact).Error
		}
	}
	if err != nil {
		return nil, fmt.Errorf("contact not found")
	}

	return b.contactToAddressObject(ctx, &contact), nil
}

// ListAddressObjects returns all address objects in an address book
func (b *Backend) ListAddressObjects(ctx context.Context, urlPath string, req *carddav.AddressDataRequest) ([]carddav.AddressObject, error) {
	userID, err := b.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	db := b.getDB(ctx)
	var contacts []models.Contact
	if err := db.Where("user_id = ?", userID).Find(&contacts).Error; err != nil {
		return nil, err
	}

	objects := make([]carddav.AddressObject, 0, len(contacts))
	for _, contact := range contacts {
		objects = append(objects, *b.contactToAddressObject(ctx, &contact))
	}

	return objects, nil
}

// QueryAddressObjects handles REPORT queries
func (b *Backend) QueryAddressObjects(ctx context.Context, urlPath string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	// Get all objects and filter using the library's Filter function
	objects, err := b.ListAddressObjects(ctx, urlPath, &query.DataRequest)
	if err != nil {
		return nil, err
	}

	return carddav.Filter(query, objects)
}

// PutAddressObject creates or updates an address object
func (b *Backend) PutAddressObject(ctx context.Context, urlPath string, card vcard.Card, opts *carddav.PutAddressObjectOptions) (*carddav.AddressObject, error) {
	userID, err := b.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	db := b.getDB(ctx)
	uid := extractUIDFromPath(urlPath)

	// Check for UID from card if not in path
	if uid == "" {
		uid = card.Value(vcard.FieldUID)
	}

	var contact models.Contact
	isNew := false

	// Try to find existing contact
	if uid != "" {
		err = db.Where("user_id = ? AND vcard_uid = ?", userID, uid).First(&contact).Error
		if err == gorm.ErrRecordNotFound {
			isNew = true
		} else if err != nil {
			return nil, err
		}
	} else {
		isNew = true
	}

	// Check ETag for conflict detection on updates (If-Match header)
	if !isNew && opts != nil && opts.IfMatch.IsSet() {
		matched, err := opts.IfMatch.MatchETag(contact.ETag)
		if err != nil || !matched {
			return nil, webdav.NewHTTPError(http.StatusPreconditionFailed, fmt.Errorf("ETag mismatch: resource has been modified"))
		}
	}

	// Convert vCard to contact
	updatedContact, photoData, photoMediaType, photoURL := VCardToContact(card, &contact)
	updatedContact.UserID = userID

	// Ensure VCardUID is set (RFC 6352 requires every contact to have a UID)
	if updatedContact.VCardUID == "" {
		updatedContact.VCardUID = uid
		if updatedContact.VCardUID == "" {
			updatedContact.VCardUID = card.Value(vcard.FieldUID)
		}
		if updatedContact.VCardUID == "" {
			// Generate a new UUID if none provided
			updatedContact.VCardUID = uuid.New().String()
		}
	}

	// Handle photo if provided (embedded base64)
	if len(photoData) > 0 {
		photoPath, thumbnailBase64, err := SaveContactPhoto(photoData, photoMediaType, b.getPhotoDir(ctx))
		if err != nil {
			logger.Warn().
				Err(err).
				Str("vcard_uid", updatedContact.VCardUID).
				Str("media_type", photoMediaType).
				Int("photo_size", len(photoData)).
				Msg("CardDAV: failed to save contact photo")
		} else {
			updatedContact.Photo = photoPath
			updatedContact.PhotoThumbnail = thumbnailBase64
		}
	} else if photoURL != "" {
		// Handle URL-based photo - fetch and save
		fetchedData, fetchedMediaType, err := FetchPhotoFromURL(photoURL)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("vcard_uid", updatedContact.VCardUID).
				Str("photo_url", photoURL).
				Msg("CardDAV: failed to fetch photo from URL")
		} else if len(fetchedData) > 0 {
			photoPath, thumbnailBase64, err := SaveContactPhoto(fetchedData, fetchedMediaType, b.getPhotoDir(ctx))
			if err != nil {
				logger.Warn().
					Err(err).
					Str("vcard_uid", updatedContact.VCardUID).
					Str("photo_url", photoURL).
					Msg("CardDAV: failed to save fetched photo")
			} else {
				updatedContact.Photo = photoPath
				updatedContact.PhotoThumbnail = thumbnailBase64
			}
		}
	}

	// Save contact
	if err := db.Save(updatedContact).Error; err != nil {
		return nil, err
	}

	return b.contactToAddressObject(ctx, updatedContact), nil
}

// DeleteAddressObject deletes an address object (soft delete)
func (b *Backend) DeleteAddressObject(ctx context.Context, urlPath string) error {
	userID, err := b.getUserID(ctx)
	if err != nil {
		return err
	}

	uid := extractUIDFromPath(urlPath)
	if uid == "" {
		return fmt.Errorf("invalid path")
	}

	db := b.getDB(ctx)

	// Find contact by vcard_uid or ID
	var contact models.Contact
	err = db.Where("user_id = ? AND vcard_uid = ?", userID, uid).First(&contact).Error
	if err == gorm.ErrRecordNotFound {
		var id uint
		if _, scanErr := fmt.Sscanf(uid, "%d", &id); scanErr == nil {
			err = db.Where("user_id = ? AND id = ?", userID, id).First(&contact).Error
		}
	}
	if err != nil {
		return fmt.Errorf("contact not found")
	}

	// Soft delete
	return db.Delete(&contact).Error
}

// contactToAddressObject converts a Contact to a CardDAV AddressObject
func (b *Backend) contactToAddressObject(ctx context.Context, contact *models.Contact) *carddav.AddressObject {
	username := b.getUsername(ctx)
	photoDir := b.getPhotoDir(ctx)

	// Determine UID for path
	uid := contact.VCardUID
	if uid == "" {
		uid = fmt.Sprintf("%d", contact.ID)
	}

	// Generate vCard
	card := ContactToVCard(contact, photoDir)

	// Serialize card to get content length
	var buf bytes.Buffer
	vcard.NewEncoder(&buf).Encode(card)

	return &carddav.AddressObject{
		Path:          "/carddav/addressbooks/" + username + "/contacts/" + uid + ".vcf",
		ModTime:       contact.UpdatedAt,
		ContentLength: int64(buf.Len()),
		ETag:          contact.ETag,
		Card:          card,
	}
}

// extractUIDFromPath extracts the UID from a CardDAV path
// e.g., /carddav/addressbooks/user/contacts/uid.vcf -> uid
func extractUIDFromPath(urlPath string) string {
	// Get the last path component
	base := path.Base(urlPath)

	// Remove .vcf extension if present
	base = strings.TrimSuffix(base, ".vcf")

	// Reserved path components that are not UIDs
	reserved := map[string]bool{
		"":             true,
		".":            true,
		"carddav":      true,
		"addressbooks": true,
		"principals":   true,
		"contacts":     true,
	}

	if reserved[base] {
		return ""
	}

	return base
}
