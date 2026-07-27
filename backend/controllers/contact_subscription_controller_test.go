package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mycorrhizal/config"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"mycorrhizal/services"
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
		fmt.Fprintf(&sb, `<d:response><d:href>%s</d:href>
<d:propstat><d:prop><card:address-data>%s</card:address-data><d:getetag>&quot;%s-etag&quot;</d:getetag></d:prop>
<d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, href, escaped, href)
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

// TestListContactSubscriptions_DBError exercises the db.Find error branch by
// closing the underlying *sql.DB out from under gorm before the request
// (mirrors export_controller_test.go's TestExportContactsAsVCF_DBError).
func TestListContactSubscriptions_DBError(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.GET("/contact-subscriptions", ListContactSubscriptions)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	req, _ := http.NewRequest("GET", "/contact-subscriptions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCreateContactSubscription_DBError exercises the subscription-count
// db.Count error branch (the first DB call CreateContactSubscription makes).
func TestCreateContactSubscription_DBError(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.POST("/contact-subscriptions", withValidated(func() any { return &models.ContactSubscriptionInput{} }), CreateContactSubscription)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	input := models.ContactSubscriptionInput{Name: "X", URL: "https://example.com/a/"}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/contact-subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFindContactSubscription_DBError exercises findContactSubscription's
// non-"record not found" DB error branch (used by Update/Delete/Sync).
func TestFindContactSubscription_DBError(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)
	sub := seedContactSubscription(db, user.ID, "https://example.com/a/")

	router := routerForUser(db, user.ID)
	router.DELETE("/contact-subscriptions/:id", DeleteContactSubscription)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	req, _ := http.NewRequest("DELETE", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestUpdateContactSubscription_RealValidation_MissingRequiredFields mirrors
// TestCreateContactSubscription_RealValidation_MissingRequiredFields for the
// update path's GetValidated error branch.
func TestUpdateContactSubscription_RealValidation_MissingRequiredFields(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)
	sub := seedContactSubscription(db, user.ID, "https://example.com/a/")

	router := routerForUser(db, user.ID)
	router.Use(apperrors.ErrorHandlerMiddleware())
	router.PUT("/contact-subscriptions/:id", middleware.ValidateJSONMiddleware(&models.ContactSubscriptionInput{}), UpdateContactSubscription)

	req, _ := http.NewRequest("PUT", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID)), strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// TestContactSyncError_AllSentinelsMapped exercises every branch of
// contactSyncError directly (pure function, no HTTP layer needed) — only
// ErrContactSyncUnauthorized was reachable through the existing HTTP-level
// tests above.
func TestContactSyncError_AllSentinelsMapped(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"InvalidURL", services.ErrContactSyncInvalidURL, http.StatusBadRequest},
		{"Unauthorized", services.ErrContactSyncUnauthorized, http.StatusServiceUnavailable},
		{"NotFound", services.ErrContactSyncNotFound, http.StatusServiceUnavailable},
		{"PrivateAddress", services.ErrContactSyncPrivateAddress, http.StatusServiceUnavailable},
		{"TooLarge", services.ErrContactSyncTooLarge, http.StatusServiceUnavailable},
		{"InvalidData", services.ErrContactSyncInvalidData, http.StatusServiceUnavailable},
		{"Unreachable", services.ErrContactSyncUnreachable, http.StatusServiceUnavailable},
		{"UnknownError_FallsBackToOperationFailed", fmt.Errorf("some other failure"), http.StatusUnprocessableEntity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appErr := contactSyncError(tc.err)
			require.NotNil(t, appErr)
			assert.Equal(t, tc.wantStatus, appErr.HTTPStatus, "status for %v", tc.err)
		})
	}
}

// TestContactSubscriptionHandlers_NoAuth_Unauthorized exercises the
// currentUserID !ok early-return every handler in this file checks first.
func TestContactSubscriptionHandlers_NoAuth_Unauthorized(t *testing.T) {
	db, _ := setupRouter()
	router := routerWithoutAuth(db)
	router.GET("/contact-subscriptions", ListContactSubscriptions)
	router.POST("/contact-subscriptions", withValidated(func() any { return &models.ContactSubscriptionInput{} }), CreateContactSubscription)
	router.PUT("/contact-subscriptions/:id", withValidated(func() any { return &models.ContactSubscriptionInput{} }), UpdateContactSubscription)
	router.DELETE("/contact-subscriptions/:id", DeleteContactSubscription)
	router.POST("/contact-subscriptions/:id/sync", SyncContactSubscription)

	for _, req := range []*http.Request{
		mustRequest(t, "GET", "/contact-subscriptions", nil),
		mustRequest(t, "POST", "/contact-subscriptions", strings.NewReader(`{"name":"x","url":"https://example.com"}`)),
		mustRequest(t, "PUT", "/contact-subscriptions/1", strings.NewReader(`{"name":"x","url":"https://example.com"}`)),
		mustRequest(t, "DELETE", "/contact-subscriptions/1", nil),
		mustRequest(t, "POST", "/contact-subscriptions/1/sync", nil),
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusOK, w.Code, "%s %s should not succeed without auth", req.Method, req.URL.Path)
		assert.NotEqual(t, http.StatusCreated, w.Code, "%s %s should not succeed without auth", req.Method, req.URL.Path)
	}
}

func mustRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, path, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// TestFindContactSubscription_NonNumericID_InvalidInput exercises the
// strconv.ParseUint error branch in findContactSubscription.
func TestFindContactSubscription_NonNumericID_InvalidInput(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.DELETE("/contact-subscriptions/:id", DeleteContactSubscription)

	req, _ := http.NewRequest("DELETE", "/contact-subscriptions/not-a-number", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// TestUpdateContactSubscription_ClearPassword exercises UpdateContactSubscription's
// ClearPassword branch.
func TestUpdateContactSubscription_ClearPassword(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	cfg := config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
	encrypted, err := services.EncryptCredential(cfg.JWTSecretKey, "hunter2")
	require.NoError(t, err)
	sub := models.ContactSubscription{UserID: user.ID, Name: "Has Password", URL: "https://example.com/a/", PasswordEncrypted: encrypted, SyncEnabled: true}
	require.NoError(t, db.Create(&sub).Error)

	router := routerForUser(db, user.ID)
	router.Use(func(c *gin.Context) { c.Set("cfg", cfg); c.Next() })
	router.PUT("/contact-subscriptions/:id", withValidated(func() any { return &models.ContactSubscriptionInput{} }), UpdateContactSubscription)

	input := models.ContactSubscriptionInput{Name: "Has Password", URL: sub.URL, ClearPassword: true}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest("PUT", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID)), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp models.ContactSubscriptionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.HasPassword)

	var stored models.ContactSubscription
	require.NoError(t, db.First(&stored, sub.ID).Error)
	assert.Empty(t, stored.PasswordEncrypted)
}

// TestUpdateContactSubscription_ReplacesPassword exercises the
// input.Password != "" branch (re-encrypting a new credential on update).
func TestUpdateContactSubscription_ReplacesPassword(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	cfg := config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!"}
	encrypted, err := services.EncryptCredential(cfg.JWTSecretKey, "oldpass")
	require.NoError(t, err)
	sub := models.ContactSubscription{UserID: user.ID, Name: "Sub", URL: "https://example.com/a/", PasswordEncrypted: encrypted, SyncEnabled: true}
	require.NoError(t, db.Create(&sub).Error)

	router := routerForUser(db, user.ID)
	router.Use(func(c *gin.Context) { c.Set("cfg", cfg); c.Next() })
	router.PUT("/contact-subscriptions/:id", withValidated(func() any { return &models.ContactSubscriptionInput{} }), UpdateContactSubscription)

	input := models.ContactSubscriptionInput{Name: "Sub", URL: sub.URL, Password: "newpass"}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest("PUT", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID)), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var stored models.ContactSubscription
	require.NoError(t, db.First(&stored, sub.ID).Error)
	assert.NotEqual(t, encrypted, stored.PasswordEncrypted)
	decrypted, err := services.DecryptCredential(cfg.JWTSecretKey, stored.PasswordEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "newpass", decrypted)
}

// TestUpdateContactSubscription_RejectsInvalidURL exercises the URL
// validation error branch on the update path (only the create path was
// previously tested).
func TestUpdateContactSubscription_RejectsInvalidURL(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	sub := seedContactSubscription(db, user.ID, "https://example.com/a/")

	router := routerForUser(db, user.ID)
	router.PUT("/contact-subscriptions/:id", withValidated(func() any { return &models.ContactSubscriptionInput{} }), UpdateContactSubscription)

	input := models.ContactSubscriptionInput{Name: "Sub", URL: "ftp://not-http.example.com"}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest("PUT", "/contact-subscriptions/"+strconv.Itoa(int(sub.ID)), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// TestCreateContactSubscription_RealValidation_MissingRequiredFields wires the
// real middleware.ValidateJSONMiddleware (not the withValidated bypass used
// elsewhere in this file) to prove the ContactSubscriptionInput struct tags
// (name/url required) are actually enforced end-to-end, matching the
// established pattern from contact_controller_validation_test.go.
func TestCreateContactSubscription_RealValidation_MissingRequiredFields(t *testing.T) {
	db, _ := setupRouter()
	var user models.User
	db.First(&user)

	router := routerForUser(db, user.ID)
	router.Use(apperrors.ErrorHandlerMiddleware())
	router.POST("/contact-subscriptions", middleware.ValidateJSONMiddleware(&models.ContactSubscriptionInput{}), CreateContactSubscription)

	req, _ := http.NewRequest("POST", "/contact-subscriptions", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var count int64
	db.Model(&models.ContactSubscription{}).Count(&count)
	assert.Equal(t, int64(0), count, "no subscription should be created when validation fails")
}
