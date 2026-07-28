package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mycorrhizal/config"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupContactSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Contact{},
		&models.ContactSubscription{}, &models.ContactSyncLink{},
	))
	return db
}

func contactSyncTestConfig() config.Config {
	return config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
}

func createContactSyncTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	user := models.User{Username: "tester", Password: "password123!A", Email: "tester@example.com"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func newContactTestSubscription(t *testing.T, db *gorm.DB, cfg config.Config, userID uint, url, username, password string) *models.ContactSubscription {
	t.Helper()
	encrypted, err := EncryptCredential(cfg.JWTSecretKey, password)
	require.NoError(t, err)
	sub := models.ContactSubscription{
		UserID:            userID,
		Name:              "Test address book",
		URL:               url,
		Username:          username,
		PasswordEncrypted: encrypted,
		SyncEnabled:       true,
	}
	require.NoError(t, db.Create(&sub).Error)
	return &sub
}

// parseTestCard decodes raw vCard text into a vcard.Card, the same type
// AddressObject.Card carries in real go-webdav responses.
func parseTestCard(t *testing.T, raw string) vcard.Card {
	t.Helper()
	card, err := vcard.NewDecoder(strings.NewReader(raw)).Decode()
	require.NoError(t, err)
	return card
}

// N is Family;Given;Additional;Prefix;Suffix per RFC 6350 §6.2.2.
const testVCard4Template = "BEGIN:VCARD\r\nVERSION:4.0\r\nUID:%s\r\nFN:%s %s\r\nN:%s;%s;;;\r\nEMAIL:%s\r\nEND:VCARD\r\n"

func testCard(t *testing.T, uid, firstname, lastname, email string) vcard.Card {
	t.Helper()
	return parseTestCard(t, fmt.Sprintf(testVCard4Template, uid, firstname, lastname, lastname, firstname, email))
}

// 1x1 transparent PNG, small enough to embed inline in a test vCard's PHOTO.
const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func testCardWithPhoto(t *testing.T, uid, fn string) vcard.Card {
	t.Helper()
	raw := "BEGIN:VCARD\r\nVERSION:4.0\r\nUID:" + uid + "\r\nFN:" + fn + "\r\nN:" + fn + ";;;;\r\n" +
		"PHOTO:data:image/png;base64," + testPNGBase64 + "\r\nEND:VCARD\r\n"
	return parseTestCard(t, raw)
}

// --- reconcileContactSync: the reconciliation logic, tested directly ---

func TestReconcileContactSyncCreatesNewContact(t *testing.T) {
	db := setupContactSyncTestDB(t)
	cfg := contactSyncTestConfig()
	user := createContactSyncTestUser(t, db)
	sub := newContactTestSubscription(t, db, cfg, user.ID, "https://example.com/addressbooks/test/", "", "")

	obj := carddav.AddressObject{Path: "/addressbooks/test/alice.vcf", ETag: "\"etag-1\"", Card: testCard(t, "alice-uid", "Alice", "Wonderland", "alice@example.com")}

	stats, err := reconcileContactSync(db, sub, []carddav.AddressObject{obj}, nil, false, "")
	require.NoError(t, err)
	assert.Equal(t, ContactSyncStats{Created: 1}, stats)

	var link models.ContactSyncLink
	require.NoError(t, db.Where("subscription_id = ? AND href = ?", sub.ID, obj.Path).First(&link).Error)
	assert.Equal(t, obj.ETag, link.ETag)
	assert.NotEmpty(t, link.ContentHash)

	var contact models.Contact
	require.NoError(t, db.First(&contact, link.ContactID).Error)
	assert.Equal(t, user.ID, contact.UserID)
	assert.Equal(t, "Alice", contact.Firstname)
	assert.Equal(t, "Wonderland", contact.Lastname)
	assert.Equal(t, "alice@example.com", contact.Email)
	assert.False(t, contact.Archived)
}

func TestReconcileContactSyncUpdatesExistingLinkedContact(t *testing.T) {
	db := setupContactSyncTestDB(t)
	cfg := contactSyncTestConfig()
	user := createContactSyncTestUser(t, db)
	sub := newContactTestSubscription(t, db, cfg, user.ID, "https://example.com/addressbooks/test/", "", "")

	href := "/addressbooks/test/bob.vcf"
	first := carddav.AddressObject{Path: href, ETag: "\"etag-1\"", Card: testCard(t, "bob-uid", "Bob", "Builder", "bob@example.com")}
	stats, err := reconcileContactSync(db, sub, []carddav.AddressObject{first}, nil, false, "")
	require.NoError(t, err)
	require.Equal(t, 1, stats.Created)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Where("user_id = ?", user.ID).Count(&count).Error)
	require.Equal(t, int64(1), count)

	// Same href, changed content (new email, new ETag) -> must update the
	// SAME Contact row, not create a second one.
	second := carddav.AddressObject{Path: href, ETag: "\"etag-2\"", Card: testCard(t, "bob-uid", "Bob", "Builder", "bob.new@example.com")}
	stats, err = reconcileContactSync(db, sub, []carddav.AddressObject{second}, nil, false, "")
	require.NoError(t, err)
	assert.Equal(t, ContactSyncStats{Updated: 1}, stats)

	require.NoError(t, db.Model(&models.Contact{}).Where("user_id = ?", user.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count, "update must not create a duplicate contact")

	var link models.ContactSyncLink
	require.NoError(t, db.Where("subscription_id = ? AND href = ?", sub.ID, href).First(&link).Error)
	assert.Equal(t, "\"etag-2\"", link.ETag)

	var contact models.Contact
	require.NoError(t, db.First(&contact, link.ContactID).Error)
	assert.Equal(t, "bob.new@example.com", contact.Email)
}

func TestReconcileContactSyncSkipsUnchangedContent(t *testing.T) {
	db := setupContactSyncTestDB(t)
	cfg := contactSyncTestConfig()
	user := createContactSyncTestUser(t, db)
	sub := newContactTestSubscription(t, db, cfg, user.ID, "https://example.com/addressbooks/test/", "", "")

	href := "/addressbooks/test/carol.vcf"
	obj := carddav.AddressObject{Path: href, ETag: "\"etag-1\"", Card: testCard(t, "carol-uid", "Carol", "Danvers", "carol@example.com")}
	_, err := reconcileContactSync(db, sub, []carddav.AddressObject{obj}, nil, false, "")
	require.NoError(t, err)

	// Re-sync identical content under a new ETag (some servers bump ETag on
	// every fetch); content hash is unchanged, so it should be a no-op skip.
	obj.ETag = "\"etag-1-again\""
	stats, err := reconcileContactSync(db, sub, []carddav.AddressObject{obj}, nil, false, "")
	require.NoError(t, err)
	assert.Equal(t, ContactSyncStats{Skipped: 1}, stats)
}

func TestReconcileContactSyncArchivesDeletedContact(t *testing.T) {
	db := setupContactSyncTestDB(t)
	cfg := contactSyncTestConfig()
	user := createContactSyncTestUser(t, db)
	sub := newContactTestSubscription(t, db, cfg, user.ID, "https://example.com/addressbooks/test/", "", "")

	href := "/addressbooks/test/dave.vcf"
	obj := carddav.AddressObject{Path: href, ETag: "\"etag-1\"", Card: testCard(t, "dave-uid", "Dave", "Grohl", "dave@example.com")}
	_, err := reconcileContactSync(db, sub, []carddav.AddressObject{obj}, nil, false, "")
	require.NoError(t, err)

	var link models.ContactSyncLink
	require.NoError(t, db.Where("subscription_id = ? AND href = ?", sub.ID, href).First(&link).Error)
	contactID := link.ContactID

	stats, err := reconcileContactSync(db, sub, nil, []string{href}, false, "")
	require.NoError(t, err)
	assert.Equal(t, ContactSyncStats{Archived: 1}, stats)

	var contact models.Contact
	require.NoError(t, db.First(&contact, contactID).Error)
	assert.True(t, contact.Archived)

	err = db.Where("subscription_id = ? AND href = ?", sub.ID, href).First(&models.ContactSyncLink{}).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "the sync link should be removed once the contact is archived")
}

func TestReconcileContactSyncFullRefetchComputesDeletions(t *testing.T) {
	db := setupContactSyncTestDB(t)
	cfg := contactSyncTestConfig()
	user := createContactSyncTestUser(t, db)
	sub := newContactTestSubscription(t, db, cfg, user.ID, "https://example.com/addressbooks/test/", "", "")

	href1 := "/addressbooks/test/erin.vcf"
	href2 := "/addressbooks/test/frank.vcf"
	obj1 := carddav.AddressObject{Path: href1, ETag: "\"e1\"", Card: testCard(t, "erin-uid", "Erin", "Esurance", "erin@example.com")}
	obj2 := carddav.AddressObject{Path: href2, ETag: "\"e2\"", Card: testCard(t, "frank-uid", "Frank", "Castle", "frank@example.com")}
	_, err := reconcileContactSync(db, sub, []carddav.AddressObject{obj1, obj2}, nil, false, "")
	require.NoError(t, err)

	var linkFrank models.ContactSyncLink
	require.NoError(t, db.Where("subscription_id = ? AND href = ?", sub.ID, href2).First(&linkFrank).Error)

	// Simulate a full refetch (fallback path) where the server no longer
	// lists frank.vcf at all -- no explicit Deleted entry is available in
	// this mode, so reconcileContactSync must diff against existing links
	// itself.
	stats, err := reconcileContactSync(db, sub, []carddav.AddressObject{obj1}, nil, true, "")
	require.NoError(t, err)
	assert.Equal(t, ContactSyncStats{Skipped: 1, Archived: 1}, stats)

	var frankContact models.Contact
	require.NoError(t, db.First(&frankContact, linkFrank.ContactID).Error)
	assert.True(t, frankContact.Archived)

	err = db.Where("subscription_id = ? AND href = ?", sub.ID, href2).First(&models.ContactSyncLink{}).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestReconcileContactSyncPhotoRoundTrips confirms the wiring into WP-73's
// photo bridge (models.ApplyRecordToContact's photoDir handling) -- not
// re-testing that bridge's own internals, just that a synced vCard's PHOTO
// ends up on Contact.Photo/PhotoThumbnail when photoDir is supplied.
func TestReconcileContactSyncPhotoRoundTrips(t *testing.T) {
	db := setupContactSyncTestDB(t)
	cfg := contactSyncTestConfig()
	user := createContactSyncTestUser(t, db)
	sub := newContactTestSubscription(t, db, cfg, user.ID, "https://example.com/addressbooks/test/", "", "")
	photoDir := t.TempDir()

	obj := carddav.AddressObject{Path: "/addressbooks/test/grace.vcf", ETag: "\"e1\"", Card: testCardWithPhoto(t, "grace-uid", "Grace Hopper")}
	stats, err := reconcileContactSync(db, sub, []carddav.AddressObject{obj}, nil, false, photoDir)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Created)

	var link models.ContactSyncLink
	require.NoError(t, db.Where("subscription_id = ? AND href = ?", sub.ID, obj.Path).First(&link).Error)
	var contact models.Contact
	require.NoError(t, db.First(&contact, link.ContactID).Error)

	assert.NotEmpty(t, contact.Photo, "synced PHOTO should have been persisted to disk via photostore")
	assert.True(t, strings.HasPrefix(contact.PhotoThumbnail, "data:image/"), "PhotoThumbnail = %q", contact.PhotoThumbnail)
}

// --- SyncSubscription: full plumbing against a fake CardDAV server ---

// addressMultistatusResponse builds a minimal but valid CardDAV multistatus
// response for an addressbook-query/multiget REPORT, mirroring
// calendar_sync_service_test.go's multistatusResponse helper.
func addressMultistatusResponse(entries map[string]string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	sb.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">` + "\n")
	for href, cardText := range entries {
		escaped := strings.ReplaceAll(cardText, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		fmt.Fprintf(&sb, `<d:response><d:href>%s</d:href>
<d:propstat><d:prop><card:address-data>%s</card:address-data><d:getetag>&quot;%s-etag&quot;</d:getetag></d:prop>
<d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, href, escaped, href)
	}
	sb.WriteString(`</d:multistatus>`)
	return sb.String()
}

// TestSyncSubscriptionFallsBackToFullRefetch exercises SyncSubscription's
// real HTTP plumbing end to end: sync-collection REPORT fails (501, as a
// server that doesn't support RFC 6578 would respond), so the service must
// fall back to a full addressbook-query refetch and still reconcile the
// result into a real Contact row.
func TestSyncSubscriptionFallsBackToFullRefetch(t *testing.T) {
	var sawSyncCollectionAttempt, sawQueryFallback, sawAuth bool
	const vcardText = "BEGIN:VCARD\r\nVERSION:4.0\r\nUID:helen-uid\r\nFN:Helen Keller\r\nN:Keller;Helen;;;\r\nEMAIL:helen@example.com\r\nEND:VCARD\r\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("carduser:cardsecret"))
		if r.Header.Get("Authorization") == expectedAuth {
			sawAuth = true
		}

		if r.Method != "REPORT" {
			w.WriteHeader(http.StatusOK)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "sync-collection") {
			sawSyncCollectionAttempt = true
			w.WriteHeader(http.StatusNotImplemented)
			return
		}

		sawQueryFallback = true
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		fmt.Fprint(w, addressMultistatusResponse(map[string]string{
			"/addressbooks/test/helen.vcf": vcardText,
		}))
	}))
	defer server.Close()

	db := setupContactSyncTestDB(t)
	cfg := contactSyncTestConfig()
	user := createContactSyncTestUser(t, db)
	sub := newContactTestSubscription(t, db, cfg, user.ID, server.URL+"/addressbooks/test/", "carduser", "cardsecret")

	service := NewContactSyncService(false)
	stats, err := service.SyncSubscription(context.Background(), db, cfg, sub)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.Created)
	assert.True(t, sawSyncCollectionAttempt, "should have attempted sync-collection first")
	assert.True(t, sawQueryFallback, "should have fallen back to addressbook-query")
	assert.True(t, sawAuth, "should have sent Basic auth credentials")

	var contact models.Contact
	require.NoError(t, db.Where("user_id = ?", user.ID).First(&contact).Error)
	assert.Equal(t, "Helen", contact.Firstname)
	assert.Equal(t, "Keller", contact.Lastname)

	require.NoError(t, db.First(sub, sub.ID).Error)
	assert.Equal(t, models.ContactSyncStatusSuccess, sub.LastSyncStatus)
	assert.NotNil(t, sub.LastSyncedAt)
}

func TestSyncSubscriptionRecordsErrorOnUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	db := setupContactSyncTestDB(t)
	cfg := contactSyncTestConfig()
	user := createContactSyncTestUser(t, db)
	sub := newContactTestSubscription(t, db, cfg, user.ID, server.URL+"/addressbooks/test/", "carduser", "wrongpass")

	service := NewContactSyncService(false)
	_, err := service.SyncSubscription(context.Background(), db, cfg, sub)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContactSyncUnauthorized)

	require.NoError(t, db.First(sub, sub.ID).Error)
	assert.Equal(t, models.ContactSyncStatusError, sub.LastSyncStatus)
	assert.NotEmpty(t, sub.LastSyncError)
}

// --- contactPrivateBlockingDialContext ---

func TestContactPrivateBlockingDialContextRejectsPrivateAddress(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:80", "10.0.0.1:80"} {
		conn, err := contactPrivateBlockingDialContext(context.Background(), "tcp", addr)
		assert.Nil(t, conn)
		assert.ErrorIs(t, err, ErrContactSyncPrivateAddress, "addr = %q", addr)
	}
}

func TestContactPrivateBlockingDialContextUnresolvableHostWrapsUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// "*.invalid" is reserved by RFC 2606 and never resolves; whether that
	// fails as NXDOMAIN or (in a network-less sandbox) as a resolver error,
	// LookupIP returns an error either way, exercising the same branch.
	conn, err := contactPrivateBlockingDialContext(ctx, "tcp", "no-such-host.invalid:80")
	assert.Nil(t, conn)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContactSyncUnreachable)
}

func TestContactPrivateBlockingDialContextMalformedAddrPropagatesSplitHostPortError(t *testing.T) {
	conn, err := contactPrivateBlockingDialContext(context.Background(), "tcp", "no-port-here")
	assert.Nil(t, conn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing port")
	// Must not be misclassified as one of the sync sentinels.
	assert.False(t, errors.Is(err, ErrContactSyncPrivateAddress))
	assert.False(t, errors.Is(err, ErrContactSyncUnreachable))
}

// TestNewContactSyncServiceBlocksPrivateURLsWiring proves the dial-context
// function is actually wired into the http.Client's Transport when
// blockPrivateURLs is enabled, not just correct in isolation: a real request
// to a local httptest.Server (which listens on 127.0.0.1) must be refused.
func TestNewContactSyncServiceBlocksPrivateURLsWiring(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewContactSyncService(true)
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	_, doErr := service.client.Do(req)
	require.Error(t, doErr)
	assert.ErrorIs(t, doErr, ErrContactSyncPrivateAddress)
}

// --- classifyContactSyncError ---

func TestClassifyContactSyncErrorPassesSentinelThrough(t *testing.T) {
	err := classifyContactSyncError(ErrContactSyncUnauthorized)
	assert.ErrorIs(t, err, ErrContactSyncUnauthorized)
}

func TestClassifyContactSyncErrorWrapsArbitraryError(t *testing.T) {
	original := errors.New("boom: connection reset")
	err := classifyContactSyncError(original)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContactSyncUnreachable)
	assert.Contains(t, err.Error(), "boom: connection reset")
}
