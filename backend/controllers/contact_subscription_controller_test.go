package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"meerkat/config"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedContactSubscription(db *gorm.DB, userID uint, url string) models.ContactSubscription {
	sub := models.ContactSubscription{
		UserID:      userID,
		Name:        "Test address book",
		URL:         url,
		SyncEnabled: true,
	}
	db.Create(&sub)
	return sub
}

func TestListContactSubscriptions(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.GET("/contact-subscriptions", ListContactSubscriptions)

	seedContactSubscription(db, user.ID, "https://example.com/addressbooks/a/")
	seedContactSubscription(db, user.ID, "https://example.com/addressbooks/b/")

	req, _ := http.NewRequest("GET", "/contact-subscriptions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Len(t, body["contact_subscriptions"], 2)
}

func TestCreateContactSubscription(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.POST("/contact-subscriptions", withValidated(func() any { return &models.ContactSubscriptionInput{} }), CreateContactSubscription)

	input := models.ContactSubscriptionInput{
		Name:     "My Nextcloud contacts",
		URL:      "https://nextcloud.example.com/remote.php/dav/addressbooks/users/alice/contacts/",
		Username: "alice",
		Password: "hunter2",
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/contact-subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var resp models.ContactSubscriptionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "My Nextcloud contacts", resp.Name)
	assert.True(t, resp.HasPassword)
	assert.True(t, resp.SyncEnabled)

	var stored models.ContactSubscription
	require.NoError(t, db.First(&stored, resp.ID).Error)
	assert.NotEmpty(t, stored.PasswordEncrypted)
	assert.NotEqual(t, "hunter2", stored.PasswordEncrypted, "password must be encrypted at rest")
}

func TestCreateContactSubscriptionRejectsInvalidURL(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.POST("/contact-subscriptions", withValidated(func() any { return &models.ContactSubscriptionInput{} }), CreateContactSubscription)

	input := models.ContactSubscriptionInput{Name: "Bad", URL: "ftp://example.com/x"}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/contact-subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateContactSubscriptionLimit(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.POST("/contact-subscriptions", withValidated(func() any { return &models.ContactSubscriptionInput{} }), CreateContactSubscription)

	for i := 0; i < maxContactSubscriptionsPerUser; i++ {
		seedContactSubscription(db, user.ID, "https://example.com/addressbooks/"+strconv.Itoa(i)+"/")
	}

	input := models.ContactSubscriptionInput{Name: "One Too Many", URL: "https://example.com/addressbooks/extra/"}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/contact-subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdateContactSubscription(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.PUT("/contact-subscriptions/:id", withValidated(func() any { return &models.ContactSubscriptionInput{} }), UpdateContactSubscription)

	sub := seedContactSubscription(db, user.ID, "https://example.com/addressbooks/a/")
	// Give it a sync token, as if a prior sync succeeded.
	db.Model(&sub).Update("sync_token", "abc123")

	falseVal := false
	update := models.ContactSubscriptionInput{
		Name:        "Renamed",
		URL:         "https://example.com/addressbooks/new/",
		SyncEnabled: &falseVal,
	}
	body, _ := json.Marshal(update)
	req, _ := http.NewRequest("PUT", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID)), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp models.ContactSubscriptionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Renamed", resp.Name)
	assert.False(t, resp.SyncEnabled)

	var stored models.ContactSubscription
	require.NoError(t, db.First(&stored, sub.ID).Error)
	assert.Empty(t, stored.SyncToken, "changing the URL should reset the stored sync token")
}

func TestDeleteContactSubscription(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.DELETE("/contact-subscriptions/:id", DeleteContactSubscription)

	sub := seedContactSubscription(db, user.ID, "https://example.com/addressbooks/a/")
	db.Create(&models.ContactSyncLink{SubscriptionID: sub.ID, UserID: user.ID, Href: "/a.vcf", ContactID: 1, ContentHash: "h"})

	req, _ := http.NewRequest("DELETE", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var deleted models.ContactSubscription
	result := db.Unscoped().First(&deleted, sub.ID)
	require.NoError(t, result.Error)
	assert.NotNil(t, deleted.DeletedAt)

	var linkCount int64
	db.Model(&models.ContactSyncLink{}).Where("subscription_id = ?", sub.ID).Count(&linkCount)
	assert.Equal(t, int64(0), linkCount, "sync links should be cleaned up on delete")
}

func TestContactSubscriptionUserIsolation(t *testing.T) {
	db, _ := setupRouter()
	var user1 models.User
	db.First(&user1)

	user2 := models.User{Username: "other", Password: "pass", Email: "other2@example.com"}
	db.Create(&user2)

	sub := seedContactSubscription(db, user1.ID, "https://example.com/addressbooks/a/")

	router := routerForUser(db, user2.ID)
	router.DELETE("/contact-subscriptions/:id", DeleteContactSubscription)

	req, _ := http.NewRequest("DELETE", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- SyncContactSubscription: the manual-sync HTTP route ---
//
// services/contact_sync_service_test.go already thoroughly tests
// ContactSyncService.SyncSubscription's own logic directly; these tests
// instead prove the controller/route wiring itself -- that hitting
// POST /contact-subscriptions/:id/sync over real gin routing actually
// invokes the service, reflects its result in the HTTP response, and
// persists the subscription's LastSyncStatus/LastSyncedAt bookkeeping to the
// database, mirroring contact_sync_service_test.go's own fake-CardDAV-server
// pattern (TestSyncSubscriptionFallsBackToFullRefetch /
// TestSyncSubscriptionRecordsErrorOnUnauthorized) at the HTTP boundary
// instead of calling the service function directly.

// addressMultistatusResponseForTest builds a minimal but valid CardDAV
// multistatus response for an addressbook-query REPORT, mirroring
// services/contact_sync_service_test.go's own (unexported, so not reusable
// across packages) addressMultistatusResponse helper.
func addressMultistatusResponseForTest(entries map[string]string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	sb.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">` + "\n")
	for href, cardText := range entries {
		escaped := strings.ReplaceAll(cardText, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		sb.WriteString(fmt.Sprintf(`<d:response><d:href>%s</d:href>
<d:propstat><d:prop><card:address-data>%s</card:address-data><d:getetag>&quot;%s-etag&quot;</d:getetag></d:prop>
<d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, href, escaped, href))
	}
	sb.WriteString(`</d:multistatus>`)
	return sb.String()
}

// routerForContactSync builds a router carrying db/userID/cfg -- routerForUser
// (webhook_controller_test.go) doesn't set "cfg", but SyncContactSubscription
// needs currentConfig(c).JWTSecretKey to decrypt the stored credential and
// CalDAVBlockPrivateURLs to build the sync HTTP client.
func routerForContactSync(db *gorm.DB, userID uint, cfg config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", userID)
		c.Set("cfg", cfg)
		c.Next()
	})
	return r
}

func TestSyncContactSubscription_Success(t *testing.T) {
	const vcardText = "BEGIN:VCARD\r\nVERSION:4.0\r\nUID:route-uid\r\nFN:Route Test\r\nN:Test;Route;;;\r\nEMAIL:route@example.com\r\nEND:VCARD\r\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			w.WriteHeader(http.StatusOK)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "sync-collection") {
			// Simulate a server that doesn't support RFC 6578 delta sync, so
			// the service falls back to a full addressbook-query refetch.
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		fmt.Fprint(w, addressMultistatusResponseForTest(map[string]string{
			"/addressbooks/test/route.vcf": vcardText,
		}))
	}))
	defer server.Close()

	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	cfg := config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
	sub := models.ContactSubscription{
		UserID:      user.ID,
		Name:        "Route sync test",
		URL:         server.URL + "/addressbooks/test/",
		SyncEnabled: true,
	}
	require.NoError(t, db.Create(&sub).Error)

	router := routerForContactSync(db, user.ID, cfg)
	router.POST("/contact-subscriptions/:id/sync", SyncContactSubscription)

	req, _ := http.NewRequest("POST", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID))+"/sync", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(1), body["created"], "response body: %s", w.Body.String())

	var stored models.ContactSubscription
	require.NoError(t, db.First(&stored, sub.ID).Error)
	assert.Equal(t, models.ContactSyncStatusSuccess, stored.LastSyncStatus)
	assert.NotNil(t, stored.LastSyncedAt, "LastSyncedAt should be set by hitting the real route")

	var contact models.Contact
	require.NoError(t, db.Where("user_id = ?", user.ID).First(&contact).Error)
	assert.Equal(t, "Route", contact.Firstname)
	assert.Equal(t, "Test", contact.Lastname)
}

func TestSyncContactSubscription_UnauthorizedReflectsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	cfg := config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
	encrypted, err := services.EncryptCredential(cfg.JWTSecretKey, "wrongpass")
	require.NoError(t, err)

	sub := models.ContactSubscription{
		UserID:            user.ID,
		Name:              "Route sync failure test",
		URL:               server.URL + "/addressbooks/test/",
		Username:          "carduser",
		PasswordEncrypted: encrypted,
		SyncEnabled:       true,
	}
	require.NoError(t, db.Create(&sub).Error)

	router := routerForContactSync(db, user.ID, cfg)
	router.POST("/contact-subscriptions/:id/sync", SyncContactSubscription)

	req, _ := http.NewRequest("POST", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID))+"/sync", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// contactSyncError maps ErrContactSyncUnauthorized to apperrors.ErrExternal,
	// which carries http.StatusServiceUnavailable.
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())

	var stored models.ContactSubscription
	require.NoError(t, db.First(&stored, sub.ID).Error)
	assert.Equal(t, models.ContactSyncStatusError, stored.LastSyncStatus, "the failed sync must still be recorded on the subscription")
	assert.NotEmpty(t, stored.LastSyncError)
	assert.NotNil(t, stored.LastSyncedAt)
}

func TestSyncContactSubscription_NotFoundForOtherUser(t *testing.T) {
	db, _ := setupRouter()
	var user1 models.User
	db.First(&user1)

	user2 := models.User{Username: "other-sync", Password: "pass", Email: "othersync@example.com"}
	require.NoError(t, db.Create(&user2).Error)

	sub := seedContactSubscription(db, user1.ID, "https://example.com/addressbooks/a/")

	cfg := config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
	router := routerForContactSync(db, user2.ID, cfg)
	router.POST("/contact-subscriptions/:id/sync", SyncContactSubscription)

	req, _ := http.NewRequest("POST", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID))+"/sync", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
