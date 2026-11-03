package controllers

import (
	"errors"
	"fmt"
	"io"
	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type carddavFixture struct {
	db       *gorm.DB
	cfg      config.Config
	user     models.User
	stranger models.User
	router   *gin.Engine
	// strangerRouter acts as a second user against the same database.
	strangerRouter *gin.Engine
}

func setupCardDAV(t *testing.T) carddavFixture {
	t.Helper()
	db := newTestDB(t)
	cfg := config.Config{JWTSecretKey: testJWTSecret}
	user := newTestUser(t, db, "owner", false)
	stranger := newTestUser(t, db, "stranger", false)

	build := func(userID uint) *gin.Engine {
		router := routerAs(db, userID, cfg)
		router.GET("/carddav/connection", GetCardDAVConnection)
		router.PUT("/carddav/connection", middleware.ValidateJSONMiddleware(&models.CardDAVConnectionInput{}), SaveCardDAVConnection)
		router.DELETE("/carddav/connection", DeleteCardDAVConnection)
		router.POST("/carddav/discover", middleware.ValidateJSONMiddleware(&models.CardDAVDiscoverInput{}), DiscoverCardDAVAddressBooks)
		router.POST("/carddav/connection/sync", SyncCardDAVConnection)
		return router
	}
	return carddavFixture{db: db, cfg: cfg, user: user, stranger: stranger, router: build(user.ID), strangerRouter: build(stranger.ID)}
}

func (f carddavFixture) connection(t *testing.T, userID uint) models.CardDAVConnection {
	t.Helper()
	var conn models.CardDAVConnection
	require.NoError(t, f.db.Where("user_id = ?", userID).First(&conn).Error)
	return conn
}

func (f carddavFixture) seedConnection(t *testing.T, baseURL, password string) models.CardDAVConnection {
	t.Helper()
	encrypted, err := services.EncryptCredential(testJWTSecret, password)
	require.NoError(t, err)
	conn := models.CardDAVConnection{
		UserID: f.user.ID, BaseURL: baseURL, AddressBookPath: "/books/personal/", Username: "owner",
		PasswordEncrypted: encrypted, Direction: models.CardDAVDirectionTwoWay, SyncEnabled: true,
	}
	require.NoError(t, f.db.Create(&conn).Error)
	return conn
}

func TestSaveCardDAVConnection_CreatesWithDefaultsAndEncryptsThePassword(t *testing.T) {
	f := setupCardDAV(t)

	w := doJSON(t, f.router, http.MethodPut, "/carddav/connection", gin.H{
		"base_url": "https://user:inline-secret@dav.example.com/dav#frag", "address_book_path": "/books/personal/",
		"address_book_name": "Personal", "username": "owner", "password": "hunter2",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	assert.NotContains(t, w.Body.String(), "hunter2")
	assert.NotContains(t, w.Body.String(), "inline-secret")
	response := decodeJSON[models.CardDAVConnectionResponse](t, w)
	assert.True(t, response.HasPassword)
	assert.Equal(t, models.CardDAVDirectionTwoWay, response.Direction)
	assert.True(t, response.SyncEnabled)
	assert.Equal(t, "https://dav.example.com/dav", response.BaseURL, "inline credentials and fragments are dropped")
	assert.NotContains(t, w.Body.String(), f.connection(t, f.user.ID).PasswordEncrypted)

	conn := f.connection(t, f.user.ID)
	assert.NotContains(t, conn.PasswordEncrypted, "hunter2", "the password must not be stored in plain text")
	decrypted, err := services.DecryptCredential(testJWTSecret, conn.PasswordEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "hunter2", decrypted)
}

func TestSaveCardDAVConnection_CanBeCreatedWithSyncDisabled(t *testing.T) {
	f := setupCardDAV(t)

	w := doJSON(t, f.router, http.MethodPut, "/carddav/connection", gin.H{"base_url": "https://dav.example.com/", "address_book_path": "/x/", "sync_enabled": false})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	assert.False(t, f.connection(t, f.user.ID).SyncEnabled, "the user opted out of background syncing")
}

func TestSaveCardDAVConnection_PasswordHandlingOnUpdate(t *testing.T) {
	f := setupCardDAV(t)
	f.seedConnection(t, "https://dav.example.com/", "original")
	base := func(extra gin.H) gin.H {
		body := gin.H{"base_url": "https://dav.example.com/", "address_book_path": "/books/personal/"}
		for key, value := range extra {
			body[key] = value
		}
		return body
	}
	stored := func() string {
		decrypted, err := services.DecryptCredential(testJWTSecret, f.connection(t, f.user.ID).PasswordEncrypted)
		require.NoError(t, err)
		return decrypted
	}

	w := doJSON(t, f.router, http.MethodPut, "/carddav/connection", base(nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "original", stored(), "an empty password keeps the stored one")
	assert.True(t, decodeJSON[models.CardDAVConnectionResponse](t, w).HasPassword)

	w = doJSON(t, f.router, http.MethodPut, "/carddav/connection", base(gin.H{"password": "replaced"}))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "replaced", stored())

	w = doJSON(t, f.router, http.MethodPut, "/carddav/connection", base(gin.H{"clear_password": true, "password": "ignored"}))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Empty(t, f.connection(t, f.user.ID).PasswordEncrypted, "clearing wins")
	assert.False(t, decodeJSON[models.CardDAVConnectionResponse](t, w).HasPassword)
}

func TestSaveCardDAVConnection_ChangingServerDropsTheStoredPassword(t *testing.T) {
	f := setupCardDAV(t)
	f.seedConnection(t, "https://dav.example.com/", "original")

	w := doJSON(t, f.router, http.MethodPut, "/carddav/connection", gin.H{"base_url": "https://evil.example.org/", "address_book_path": "/books/personal/"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.Empty(t, f.connection(t, f.user.ID).PasswordEncrypted, "credentials stored for one host must not follow the connection to another")
}

func TestSaveCardDAVConnection_ChangingPortOrSchemeDropsTheStoredPassword(t *testing.T) {
	for _, target := range []string{"https://dav.example.com:8443/", "http://dav.example.com/"} {
		t.Run(target, func(t *testing.T) {
			f := setupCardDAV(t)
			f.seedConnection(t, "https://dav.example.com/", "original")

			w := doJSON(t, f.router, http.MethodPut, "/carddav/connection", gin.H{"base_url": target, "address_book_path": "/books/personal/"})
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			assert.Empty(t, f.connection(t, f.user.ID).PasswordEncrypted)
		})
	}
}

func TestSaveCardDAVConnection_ChangingAddressBookResetsSyncState(t *testing.T) {
	f := setupCardDAV(t)
	conn := f.seedConnection(t, "https://dav.example.com/", "original")
	now := time.Now()
	require.NoError(t, f.db.Model(&conn).Updates(map[string]any{
		"sync_token": "tok", "last_synced_at": now, "last_sync_status": "success", "last_sync_stats": `{"errors":0}`,
	}).Error)
	contact := newTestContact(t, f.db, f.user.ID, "Linked")
	require.NoError(t, f.db.Create(&models.CardDAVContactLink{ConnectionID: conn.ID, UserID: f.user.ID, ContactID: contact.ID, RemoteUID: "u", RemotePath: "/p"}).Error)
	links := func() (n int64) {
		require.NoError(t, f.db.Model(&models.CardDAVContactLink{}).Count(&n).Error)
		return n
	}

	// Editing only the name keeps everything.
	w := doJSON(t, f.router, http.MethodPut, "/carddav/connection", gin.H{"base_url": "https://dav.example.com/", "address_book_path": "/books/personal/", "address_book_name": "Renamed", "sync_enabled": false, "direction": "pull"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.EqualValues(t, 1, links())
	reloaded := f.connection(t, f.user.ID)
	assert.Equal(t, "tok", reloaded.SyncToken)
	assert.Equal(t, "Renamed", reloaded.AddressBookName)
	assert.False(t, reloaded.SyncEnabled)
	assert.Equal(t, "pull", reloaded.Direction)

	// Pointing at another book invalidates the link state, but never the contacts.
	w = doJSON(t, f.router, http.MethodPut, "/carddav/connection", gin.H{"base_url": "https://dav.example.com/", "address_book_path": "/books/work/"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Zero(t, links())
	reloaded = f.connection(t, f.user.ID)
	assert.Empty(t, reloaded.SyncToken)
	assert.Nil(t, reloaded.LastSyncedAt)
	assert.Empty(t, reloaded.LastSyncStats)
	assert.Equal(t, "pull", reloaded.Direction, "direction survives when not sent")
	var kept models.Contact
	assert.NoError(t, f.db.First(&kept, contact.ID).Error)
}

func TestSaveCardDAVConnection_RejectsInvalidInput(t *testing.T) {
	for name, body := range map[string]gin.H{
		"unsupported scheme": {"base_url": "ftp://dav.example.com/", "address_book_path": "/books/"},
		"relative book path": {"base_url": "https://dav.example.com/", "address_book_path": "books/"},
		"missing book path":  {"base_url": "https://dav.example.com/"},
		"unknown direction":  {"base_url": "https://dav.example.com/", "address_book_path": "/books/", "direction": "sideways"},
	} {
		t.Run(name, func(t *testing.T) {
			f := setupCardDAV(t)

			w := doJSON(t, f.router, http.MethodPut, "/carddav/connection", body)

			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var count int64
			require.NoError(t, f.db.Model(&models.CardDAVConnection{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

func TestCardDAVConnection_IsPrivateToItsOwner(t *testing.T) {
	f := setupCardDAV(t)
	conn := f.seedConnection(t, "https://dav.example.com/", "original")
	contact := newTestContact(t, f.db, f.user.ID, "Linked")
	require.NoError(t, f.db.Create(&models.CardDAVContactLink{ConnectionID: conn.ID, UserID: f.user.ID, ContactID: contact.ID, RemoteUID: "u", RemotePath: "/p"}).Error)

	assert.Equal(t, http.StatusNotFound, doRequest(f.strangerRouter, http.MethodGet, "/carddav/connection", nil, "").Code)
	assert.Equal(t, http.StatusNotFound, doRequest(f.strangerRouter, http.MethodDelete, "/carddav/connection", nil, "").Code)
	assert.Equal(t, http.StatusNotFound, doRequest(f.strangerRouter, http.MethodPost, "/carddav/connection/sync", nil, "").Code)

	// Saving as the stranger creates their own connection and leaves the owner's alone.
	w := doJSON(t, f.strangerRouter, http.MethodPut, "/carddav/connection", gin.H{"base_url": "https://other.example.com/", "address_book_path": "/x/"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	reloaded := f.connection(t, f.user.ID)
	assert.Equal(t, "https://dav.example.com/", reloaded.BaseURL)
	assert.NotEmpty(t, reloaded.PasswordEncrypted)

	var links int64
	require.NoError(t, f.db.Model(&models.CardDAVContactLink{}).Count(&links).Error)
	assert.EqualValues(t, 1, links)
}

func TestGetCardDAVConnection_ReturnsNoSecrets(t *testing.T) {
	f := setupCardDAV(t)
	conn := f.seedConnection(t, "https://dav.example.com/", "original")
	require.NoError(t, f.db.Model(&conn).Updates(map[string]any{"last_sync_stats": `{"errors":2}`, "sync_token": "secret-token"}).Error)

	w := doRequest(f.router, http.MethodGet, "/carddav/connection", nil, "")
	require.Equal(t, http.StatusOK, w.Code)

	assert.NotContains(t, w.Body.String(), conn.PasswordEncrypted)
	assert.NotContains(t, w.Body.String(), "original")
	assert.NotContains(t, w.Body.String(), "secret-token")
	response := decodeJSON[models.CardDAVConnectionResponse](t, w)
	assert.True(t, response.HasPassword)
	assert.False(t, response.SyncRunning)
	assert.JSONEq(t, `{"errors":2}`, string(response.LastSyncStats))

	// A corrupt stats column must not corrupt the response.
	require.NoError(t, f.db.Model(&conn).Update("last_sync_stats", "{not json").Error)
	w = doRequest(f.router, http.MethodGet, "/carddav/connection", nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "null", string(decodeJSON[models.CardDAVConnectionResponse](t, w).LastSyncStats))
}

func TestDeleteCardDAVConnection_KeepsContacts(t *testing.T) {
	f := setupCardDAV(t)
	conn := f.seedConnection(t, "https://dav.example.com/", "original")
	contact := newTestContact(t, f.db, f.user.ID, "Linked")
	require.NoError(t, f.db.Create(&models.CardDAVContactLink{ConnectionID: conn.ID, UserID: f.user.ID, ContactID: contact.ID, RemoteUID: "u", RemotePath: "/p"}).Error)

	w := doRequest(f.router, http.MethodDelete, "/carddav/connection", nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var connections, links int64
	require.NoError(t, f.db.Model(&models.CardDAVConnection{}).Count(&connections).Error)
	require.NoError(t, f.db.Model(&models.CardDAVContactLink{}).Count(&links).Error)
	assert.Zero(t, connections, "stored credentials must be gone")
	assert.Zero(t, links)
	var kept models.Contact
	assert.NoError(t, f.db.First(&kept, contact.ID).Error)

	assert.Equal(t, http.StatusNotFound, doRequest(f.router, http.MethodGet, "/carddav/connection", nil, "").Code)
}

// fakeDAVServer answers the three discovery PROPFINDs and records the credentials it saw.
func fakeDAVServer(t *testing.T) (*httptest.Server, *atomic.Value) {
	t.Helper()
	var seen atomic.Value
	seen.Store("")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, password, ok := r.BasicAuth(); ok {
			seen.Store(user + ":" + password)
		}
		body, _ := io.ReadAll(r.Body)
		var prop string
		href := r.URL.Path
		switch {
		case strings.Contains(string(body), "current-user-principal"):
			prop = `<D:current-user-principal><D:href>/principals/me/</D:href></D:current-user-principal>`
		case strings.Contains(string(body), "addressbook-home-set"):
			prop = `<C:addressbook-home-set><D:href>/books/</D:href></C:addressbook-home-set>`
		default:
			href = "/books/personal/"
			prop = `<D:resourcetype><D:collection/><C:addressbook/></D:resourcetype><D:displayname>Personal</D:displayname>`
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		fmt.Fprintf(w, `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
<D:response><D:href>%s</D:href><D:propstat><D:prop>%s</D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response>
</D:multistatus>`, href, prop)
	}))
	t.Cleanup(server.Close)
	return server, &seen
}

func TestDiscoverCardDAV_ListsAddressBooksWithSuppliedCredentials(t *testing.T) {
	f := setupCardDAV(t)
	server, seen := fakeDAVServer(t)

	w := doJSON(t, f.router, http.MethodPost, "/carddav/discover", gin.H{"base_url": server.URL + "/", "username": "u", "password": "typed"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	books := decodeJSON[struct {
		AddressBooks []models.DiscoveredAddressBook `json:"address_books"`
	}](t, w).AddressBooks
	require.Len(t, books, 1)
	assert.Equal(t, "/books/personal/", books[0].Path)
	assert.Equal(t, "u:typed", seen.Load())
}

func TestDiscoverCardDAV_StoredPasswordOnlyGoesToItsOwnHost(t *testing.T) {
	f := setupCardDAV(t)
	server, seen := fakeDAVServer(t)
	f.seedConnection(t, server.URL+"/", "stored-secret")

	// Same host: the stored password is reused.
	w := doJSON(t, f.router, http.MethodPost, "/carddav/discover", gin.H{"base_url": server.URL + "/", "username": "u"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u:stored-secret", seen.Load())

	// Different host (same server, reached by another name): it must not travel.
	seen.Store("")
	otherHost := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	w = doJSON(t, f.router, http.MethodPost, "/carddav/discover", gin.H{"base_url": otherHost + "/", "username": "u"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.NotContains(t, seen.Load(), "stored-secret")

	// Another user never gets the owner's password, even for the same host.
	seen.Store("")
	w = doJSON(t, f.strangerRouter, http.MethodPost, "/carddav/discover", gin.H{"base_url": server.URL + "/", "username": "u"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.NotContains(t, seen.Load(), "stored-secret")
}

func TestDiscoverCardDAV_Failures(t *testing.T) {
	f := setupCardDAV(t)
	unauthorized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthorized.Close()

	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, "/carddav/discover", gin.H{"base_url": "ftp://dav.example.com/"}).Code)
	assert.Equal(t, http.StatusBadRequest, doJSON(t, f.router, http.MethodPost, "/carddav/discover", gin.H{}).Code)

	w := doJSON(t, f.router, http.MethodPost, "/carddav/discover", gin.H{"base_url": unauthorized.URL + "/", "password": "wrong"})
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "authentication failed")
}

func TestContactSyncError_MapsSentinelsAndHidesInternals(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{services.ErrContactSyncInvalidURL, http.StatusBadRequest},
		{services.ErrContactSyncUnauthorized, http.StatusServiceUnavailable},
		{services.ErrContactSyncForbidden, http.StatusServiceUnavailable},
		{services.ErrContactSyncNotFound, http.StatusServiceUnavailable},
		{services.ErrContactSyncPrivateAddress, http.StatusServiceUnavailable},
		{services.ErrContactSyncTooLarge, http.StatusServiceUnavailable},
		{services.ErrContactSyncTooMany, http.StatusServiceUnavailable},
		{services.ErrContactSyncNoAddressBooks, http.StatusServiceUnavailable},
		{services.ErrContactSyncInvalidData, http.StatusServiceUnavailable},
		{services.ErrContactSyncUnreachable, http.StatusServiceUnavailable},
	} {
		wrapped := fmt.Errorf("while talking to internal-host-1: %w", tc.err)
		appErr := contactSyncError(wrapped)
		assert.Equal(t, tc.status, appErr.HTTPStatus, tc.err.Error())
		assert.NotContains(t, appErr.Message, "internal-host-1", tc.err.Error())
	}

	unexpected := contactSyncError(errors.New("db at 10.0.0.5 exploded"))
	assert.Equal(t, apperrors.ErrCodeOperationFailed, unexpected.Code)
	assert.NotContains(t, unexpected.Message, "10.0.0.5", "internal detail must not reach the client")
}

func waitForContactSync(t *testing.T, connID uint) {
	t.Helper()
	require.Eventually(t, func() bool { return !services.IsContactSyncRunning(connID) }, 10*time.Second, 5*time.Millisecond)
}

func TestSyncCardDAVConnection_RunsInBackgroundAndRecordsTheOutcome(t *testing.T) {
	f := setupCardDAV(t)
	var sawPassword atomic.Value
	sawPassword.Store("")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, password, ok := r.BasicAuth(); ok {
			sawPassword.Store(password)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	conn := f.seedConnection(t, server.URL+"/", "stored-secret")

	w := doRequest(f.router, http.MethodPost, "/carddav/connection/sync", nil, "")
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	waitForContactSync(t, conn.ID)

	reloaded := f.connection(t, f.user.ID)
	assert.Equal(t, "stored-secret", sawPassword.Load(), "the stored password must be decrypted for the server")
	assert.Equal(t, models.CardDAVSyncStatusError, reloaded.LastSyncStatus)
	assert.NotEmpty(t, reloaded.LastSyncError)
}

func TestSyncCardDAVConnection_SecondRequestWhileRunningConflicts(t *testing.T) {
	f := setupCardDAV(t)
	release := make(chan struct{})
	var once atomic.Bool
	reached := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if once.CompareAndSwap(false, true) {
			close(reached)
		}
		<-release
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	conn := f.seedConnection(t, server.URL+"/", "stored-secret")

	require.Equal(t, http.StatusAccepted, doRequest(f.router, http.MethodPost, "/carddav/connection/sync", nil, "").Code)
	<-reached

	w := doRequest(f.router, http.MethodPost, "/carddav/connection/sync", nil, "")
	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	running := doRequest(f.router, http.MethodGet, "/carddav/connection", nil, "")
	assert.True(t, decodeJSON[models.CardDAVConnectionResponse](t, running).SyncRunning)

	close(release)
	waitForContactSync(t, conn.ID)
}
