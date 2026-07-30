package carddav

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// newAuthTestRouter builds an in-memory-SQLite-backed gin router with
// BasicAuthMiddleware wired ahead of a probe route that echoes the
// authenticated userID, mirroring controllers/activity_controller_test.go's
// setupRouter pattern.
func newAuthTestRouter(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.ApiToken{}))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})
	router.Use(BasicAuthMiddleware())
	router.GET("/probe", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		c.JSON(http.StatusOK, gin.H{"userID": userID})
	})

	return db, router
}

func createTestToken(t *testing.T, db *gorm.DB, userID uint, scope string, opts ...func(*models.ApiToken)) string {
	t.Helper()
	rawBytes := make([]byte, 32)
	_, err := rand.Read(rawBytes)
	require.NoError(t, err)
	plaintext := "mycorrhizal_" + base64.RawURLEncoding.EncodeToString(rawBytes)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))

	expiresAt := time.Now().Add(90 * 24 * time.Hour)
	token := models.ApiToken{UserID: userID, Name: "test-token", TokenHash: hash, Scope: scope, ExpiresAt: &expiresAt}
	for _, opt := range opts {
		opt(&token)
	}
	require.NoError(t, db.Create(&token).Error)
	return plaintext
}

func doBasicAuthRequest(router *gin.Engine, username, password string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", "/probe", nil)
	req.SetBasicAuth(username, password)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestBasicAuthMiddleware_PasswordStillWorks(t *testing.T) {
	db, router := newAuthTestRouter(t)

	hashed, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)
	user := models.User{Username: "pw-user", Email: "pw-user@example.com", Password: string(hashed)}
	require.NoError(t, db.Create(&user).Error)

	w := doBasicAuthRequest(router, "pw-user", "correct-password")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBasicAuthMiddleware_FullScopeTokenAuthenticates(t *testing.T) {
	db, router := newAuthTestRouter(t)

	user := models.User{Username: "full-token-user", Email: "full-token-user@example.com", Password: "unused"}
	require.NoError(t, db.Create(&user).Error)
	plaintext := createTestToken(t, db, user.ID, "full")

	w := doBasicAuthRequest(router, "full-token-user", plaintext)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBasicAuthMiddleware_CardDAVScopeTokenAuthenticates(t *testing.T) {
	db, router := newAuthTestRouter(t)

	user := models.User{Username: "carddav-token-user", Email: "carddav-token-user@example.com", Password: "unused"}
	require.NoError(t, db.Create(&user).Error)
	plaintext := createTestToken(t, db, user.ID, "carddav")

	w := doBasicAuthRequest(router, "carddav-token-user", plaintext)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBasicAuthMiddleware_SSOUserWithEmptyPasswordAuthenticatesViaToken(t *testing.T) {
	db, router := newAuthTestRouter(t)

	// OIDC-provisioned users get Password: "" -- bcrypt can never match it,
	// so a scoped token is the only way for them to use CardDAV.
	user := models.User{Username: "sso-user", Email: "sso-user@example.com", Password: ""}
	require.NoError(t, db.Create(&user).Error)
	plaintext := createTestToken(t, db, user.ID, "carddav")

	w := doBasicAuthRequest(router, "sso-user", plaintext)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBasicAuthMiddleware_TokenForDifferentUserRejected(t *testing.T) {
	db, router := newAuthTestRouter(t)

	owner := models.User{Username: "owner-user", Email: "owner-user@example.com", Password: "unused"}
	require.NoError(t, db.Create(&owner).Error)
	victim := models.User{Username: "victim-user", Email: "victim-user@example.com", Password: "unused"}
	require.NoError(t, db.Create(&victim).Error)

	plaintext := createTestToken(t, db, owner.ID, "full")

	// Typed username belongs to victim, but the token belongs to owner.
	w := doBasicAuthRequest(router, "victim-user", plaintext)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBasicAuthMiddleware_RevokedTokenRejected(t *testing.T) {
	db, router := newAuthTestRouter(t)

	user := models.User{Username: "revoked-user", Email: "revoked-user@example.com", Password: "unused"}
	require.NoError(t, db.Create(&user).Error)
	now := time.Now()
	plaintext := createTestToken(t, db, user.ID, "full", func(a *models.ApiToken) { a.RevokedAt = &now })

	w := doBasicAuthRequest(router, "revoked-user", plaintext)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBasicAuthMiddleware_ExpiredTokenRejected(t *testing.T) {
	db, router := newAuthTestRouter(t)

	user := models.User{Username: "expired-user", Email: "expired-user@example.com", Password: "unused"}
	require.NoError(t, db.Create(&user).Error)
	past := time.Now().Add(-time.Hour)
	plaintext := createTestToken(t, db, user.ID, "full", func(a *models.ApiToken) { a.ExpiresAt = &past })

	w := doBasicAuthRequest(router, "expired-user", plaintext)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBasicAuthMiddleware_WrongPasswordAndNoValidTokenRejected(t *testing.T) {
	db, router := newAuthTestRouter(t)

	hashed, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)
	user := models.User{Username: "wrong-pw-user", Email: "wrong-pw-user@example.com", Password: string(hashed)}
	require.NoError(t, db.Create(&user).Error)

	w := doBasicAuthRequest(router, "wrong-pw-user", "totally-wrong")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
