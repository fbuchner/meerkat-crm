package carddav

import (
	"bytes"
	"context"
	"fmt"
	"mycorrhizal/contactmodel"
	"mycorrhizal/logger"
	"mycorrhizal/models"
	"mycorrhizal/vcard3"
	"mycorrhizal/vcard4"
	"net/http"
	"os"
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
	userIDKey    contextKey = "userID"
	usernameKey  contextKey = "username"
	dbKey        contextKey = "db"
	photoDir     contextKey = "photoDir"
	acceptHeader contextKey = "acceptHeader"
)

// DefaultVCardVersion is the vCard version served by GetAddressObject/
// ListAddressObjects (and advertised as the export default) when a request
// doesn't otherwise indicate it needs 3.0 — see requestedVCardVersion.
//
// Per docs/fork-plan/50-integration-and-rebrand.md WP-73's content-
// negotiation step, "emit 4.0 by default, 3.0 for clients that require it".
// Kept as a package-level var read from CARDDAV_DEFAULT_VCARD_VERSION
// (rather than a new config.Config field) since backend/config is outside
// this WP's file scope (backend/photostore, backend/models,
// backend/controllers, backend/services, backend/carddav only) — the same
// self-contained env-var pattern as models.DefaultPhotoDir. This gives an
// operator a way to pin the whole deployment back to 3.0 (e.g. if a
// particular fleet of legacy CardDAV clients needs it) without a code change.
var DefaultVCardVersion = envOrDefault("CARDDAV_DEFAULT_VCARD_VERSION", "4.0")

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

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

// ContextWithUser adds user info to context for the backend. acceptHeaderValue
// is the incoming request's raw Accept header, used for vCard version
// negotiation (see requestedVCardVersion) — go-webdav's own Backend
// interface does not surface a requested content-type/version for
// GetAddressObject/ListAddressObjects (confirmed by reading go-webdav
// v0.7.0's server.go: decodeAddressDataReq parses but discards the RFC 6352
// <C:address-data content-type version> attributes from REPORT requests),
// so the plain HTTP Accept header on the underlying request is the only
// per-request negotiation signal available at this layer; the caller
// (Handler.GinHandler) is the only place with access to the raw
// *http.Request to read it from.
func ContextWithUser(ctx context.Context, userID uint, username string, db *gorm.DB, photoDirPath string, acceptHeaderValue string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, usernameKey, username)
	ctx = context.WithValue(ctx, dbKey, db)
	ctx = context.WithValue(ctx, photoDir, photoDirPath)
	ctx = context.WithValue(ctx, acceptHeader, acceptHeaderValue)
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

// requestedVCardVersion picks the vCard version to serve for this request:
// an explicit version=3(.0) in the Accept header wins (e.g.
// "text/vcard;version=3.0", the form RFC 6352 §10.4.1 examples use for
// content negotiation on a plain GET); otherwise DefaultVCardVersion. See
// ContextWithUser's doc comment for why the Accept header (rather than a
// protocol-level content-type/version request) is the negotiation signal
// available here.
func requestedVCardVersion(ctx context.Context) string {
	accept, _ := ctx.Value(acceptHeader).(string)
	accept = strings.ToLower(accept)
	if strings.Contains(accept, "version=3") {
		return "3.0"
	}
	if strings.Contains(accept, "version=4") {
		return "4.0"
	}
	return DefaultVCardVersion
}

// exporterForVersion returns the adapter that serializes a contactmodel.Record
// into the given vCard version ("3.0"/"3" -> vcard3, everything else,
// including "4.0"/"4"/"" -> vcard4).
func exporterForVersion(version string) contactmodel.Exporter {
	if strings.HasPrefix(strings.TrimSpace(version), "3") {
		return vcard3.Adapter{}
	}
	return vcard4.Adapter{}
}

// importerForVCard sniffs the VERSION property already decoded onto card
// (go-webdav's Put handler parses the request body into a vcard.Card before
// calling Backend.PutAddressObject) and returns the matching adapter.
func importerForVCard(card vcard.Card) contactmodel.Importer {
	if strings.HasPrefix(strings.TrimSpace(card.Value(vcard.FieldVersion)), "3") {
		return vcard3.Adapter{}
	}
	return vcard4.Adapter{}
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
			Description: "Mycorrhizal CRM Contacts",
			// Advertise both versions (per WP-73: "advertise 4.0 (and/or 3.0)")
			// rather than 4.0-only: this is a capability announcement, not the
			// version actually served (that's requestedVCardVersion's job), so
			// advertising both keeps clients that specifically negotiate for
			// 3.0 informed that it's genuinely supported.
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
		Description: "Mycorrhizal CRM Contacts",
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

	// Route the incoming vCard through the vcard4/vcard3 adapters instead of
	// the legacy carddav.VCardToContact mapper (docs/fork-plan/
	// 50-integration-and-rebrand.md WP-73). go-webdav's Put handler has
	// already decoded the request body into `card` (a vcard.Card) before
	// calling us, so we re-encode it back to bytes for the adapter's Import
	// (which does its own go-vcard parsing) — this keeps a single
	// vCard-interpretation path (the P0 adapters + correspondence table)
	// rather than also reading go-webdav's own parsed vcard.Card fields
	// directly, per WP-73b's own note on this exact pattern.
	var buf bytes.Buffer
	if err := vcard.NewEncoder(&buf).Encode(card); err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, fmt.Errorf("carddav: failed to re-encode vCard: %w", err))
	}
	record, diags, err := importerForVCard(card).Import(buf.Bytes())
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, fmt.Errorf("carddav: failed to parse vCard: %w", err))
	}
	for _, d := range diags {
		logger.Debug().Str("severity", d.Severity).Str("concept", d.Concept).Msg("CardDAV PUT: " + d.Message)
	}

	// ApplyRecordToContact populates the flat legacy fields (for every other
	// reader that isn't adapter-aware yet) and the neutral Card/CRM/
	// Passthrough columns, and — given a real photoDir — persists an
	// embedded or remote Card.Media{Kind:"photo"} entry to disk and mirrors
	// it onto Contact.Photo/PhotoThumbnail (WP-73's photo-bridging
	// prerequisite), replacing the photoData/photoURL handling the legacy
	// mapper's return values used to require here.
	models.ApplyRecordToContact(&contact, record, b.getPhotoDir(ctx))
	contact.UserID = userID

	// Ensure VCardUID is set (RFC 6352 requires every contact to have a UID)
	if contact.VCardUID == "" {
		contact.VCardUID = uid
		if contact.VCardUID == "" {
			contact.VCardUID = card.Value(vcard.FieldUID)
		}
		if contact.VCardUID == "" {
			// Generate a new UUID if none provided
			contact.VCardUID = uuid.New().String()
		}
	}

	// Save contact
	if err := db.Save(&contact).Error; err != nil {
		return nil, err
	}

	return b.contactToAddressObject(ctx, &contact), nil
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

// contactToAddressObject converts a Contact to a CardDAV AddressObject.
//
// Per docs/fork-plan/50-integration-and-rebrand.md WP-73, this now builds the
// card via RecordFromContact + the vcard4/vcard3 adapters (chosen by
// requestedVCardVersion's content negotiation) instead of the legacy
// carddav.ContactToVCard mapper. The adapter's Export returns serialized
// vCard bytes; since carddav.AddressObject.Card is a parsed vcard.Card (the
// go-webdav library itself always does the final wire-encoding — see
// server.go's vcard.NewEncoder(w).Encode(ao.Card) — it never accepts raw
// bytes from the Backend), the bytes are decoded back into a vcard.Card here
// to satisfy that interface.
func (b *Backend) contactToAddressObject(ctx context.Context, contact *models.Contact) *carddav.AddressObject {
	username := b.getUsername(ctx)

	// Determine UID for path
	uid := contact.VCardUID
	if uid == "" {
		uid = fmt.Sprintf("%d", contact.ID)
	}

	version := requestedVCardVersion(ctx)
	record := models.RecordForContact(contact, b.getPhotoDir(ctx), b.db)
	data, diags, err := exporterForVersion(version).Export(record)

	card := make(vcard.Card)
	if err != nil {
		logger.Warn().Err(err).Str("vcard_uid", contact.VCardUID).Str("version", version).Msg("CardDAV: failed to export contact as vCard")
	} else {
		for _, d := range diags {
			logger.Debug().Str("severity", d.Severity).Str("concept", d.Concept).Msg("CardDAV export: " + d.Message)
		}
		decoded, decodeErr := vcard.NewDecoder(bytes.NewReader(data)).Decode()
		if decodeErr != nil {
			logger.Warn().Err(decodeErr).Str("vcard_uid", contact.VCardUID).Msg("CardDAV: failed to decode exported vCard")
		} else {
			card = decoded
		}
	}

	return &carddav.AddressObject{
		Path:    "/carddav/addressbooks/" + username + "/contacts/" + uid + ".vcf",
		ModTime: contact.UpdatedAt,
		ETag:    contact.ETag,
		Card:    card,
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
