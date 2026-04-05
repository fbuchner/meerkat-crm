package middleware

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"meerkat/config"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestRouter() (*gorm.DB, *gin.Engine) {
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db")
	}
	db.AutoMigrate(&models.User{}, &models.ApiToken{})

	user := models.User{Username: "authtest", Email: "authtest@example.com", Password: "password"}
	if err := db.Create(&user).Error; err != nil {
		panic("failed to seed user")
	}

	cfg := &config.Config{JWTSecretKey: "test-secret-key-32-chars-minimum!"}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})
	router.Use(AuthMiddleware(cfg))

	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		isAPIToken, _ := c.Get("isAPIToken")
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "is_api_token": isAPIToken})
	})

	return db, router
}

func hashToken(plaintext string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))
}

func TestAuthMiddleware_ValidApiToken(t *testing.T) {
	db, router := setupAuthTestRouter()

	var user models.User
	db.First(&user)

	plaintext := "meerkat_validtoken123456789"
	db.Create(&models.ApiToken{
		UserID:    user.ID,
		Name:      "test",
		TokenHash: hashToken(plaintext),
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, float64(user.ID), body["user_id"])
	assert.Equal(t, true, body["is_api_token"])
}

func TestAuthMiddleware_RevokedApiToken(t *testing.T) {
	db, router := setupAuthTestRouter()

	var user models.User
	db.First(&user)

	plaintext := "meerkat_revokedtoken9876"
	now := time.Now()
	db.Create(&models.ApiToken{
		UserID:    user.ID,
		Name:      "revoked",
		TokenHash: hashToken(plaintext),
		RevokedAt: &now,
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_UnknownApiToken(t *testing.T) {
	_, router := setupAuthTestRouter()

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer meerkat_doesnotexistXXXX")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ApiToken_UpdatesLastUsedAt(t *testing.T) {
	db, router := setupAuthTestRouter()

	var user models.User
	db.First(&user)

	plaintext := "meerkat_lastusedupdatetoken"
	token := models.ApiToken{
		UserID:    user.ID,
		Name:      "track",
		TokenHash: hashToken(plaintext),
	}
	db.Create(&token)

	assert.Nil(t, token.LastUsedAt)

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// The update runs in a goroutine — give it a moment to complete
	time.Sleep(50 * time.Millisecond)

	var updated models.ApiToken
	db.First(&updated, token.ID)
	assert.NotNil(t, updated.LastUsedAt)
	assert.WithinDuration(t, time.Now(), *updated.LastUsedAt, 5*time.Second)
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	_, router := setupAuthTestRouter()

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminMiddleware_BlocksApiToken(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db")
	}
	db.AutoMigrate(&models.User{}, &models.ApiToken{})

	user := models.User{Username: "admintest", Email: "admin@example.com", Password: "pw", IsAdmin: true}
	db.Create(&user)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Simulate what AuthMiddleware sets for an API token
		c.Set("db", db)
		c.Set("userID", user.ID)
		c.Set("isAPIToken", true)
		c.Next()
	})
	router.Use(AdminMiddleware())

	router.GET("/admin/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, "API tokens cannot access admin endpoints", body["error"])
}

func TestAdminMiddleware_AllowsAdminUser(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db")
	}
	db.AutoMigrate(&models.User{})

	user := models.User{Username: "superadmin", Email: "super@example.com", Password: "pw", IsAdmin: true}
	db.Create(&user)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", user.ID)
		// isAPIToken NOT set — regular JWT session
		c.Next()
	})
	router.Use(AdminMiddleware())

	router.GET("/admin/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminMiddleware_BlocksNonAdminUser(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db")
	}
	db.AutoMigrate(&models.User{})

	user := models.User{Username: "regular", Email: "regular@example.com", Password: "pw", IsAdmin: false}
	db.Create(&user)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", user.ID)
		c.Next()
	})
	router.Use(AdminMiddleware())

	router.GET("/admin/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
