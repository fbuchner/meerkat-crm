package middleware

import (
	"meerkat/config"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestAuthCookieSlidingRefresh(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	cfg := &config.Config{JWTSecretKey: "test-secret-key-32-chars-minimum!", JWTExpiryHours: 10, JWTMaxSessionHours: 30}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&models.User{}))
	user := models.User{Username: "renamed", Email: "u@example.com", Password: "x"}
	user.ID = 7
	assert.NoError(t, db.Create(&user).Error)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("db", db); c.Next() })
	r.Use(AuthMiddleware(cfg))
	r.GET("/p", func(c *gin.Context) { c.Status(http.StatusOK) })

	tokenAt := func(d, age time.Duration) string {
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": "u", "user_id": 7, "exp": time.Now().Add(d).Unix(),
			"auth_time": time.Now().Add(-age).Unix(),
		}).SignedString([]byte(cfg.JWTSecretKey))
		return tok
	}
	tokenWithExp := func(d time.Duration) string { return tokenAt(d, time.Hour) }
	call := func(tok string, cookie bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/p", nil)
		if cookie {
			req.AddCookie(&http.Cookie{Name: "auth_token", Value: tok})
		} else {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// fresh cookie: no refresh
	w := call(tokenWithExp(9*time.Hour), true)
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("Set-Cookie"))

	// old cookie: refreshed with same identity
	w = call(tokenWithExp(2*time.Hour), true)
	assert.Equal(t, 200, w.Code)
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.InDelta(t, 10*3600, cookies[0].MaxAge, 5)
	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(cookies[0].Value, claims, func(*jwt.Token) (any, error) { return []byte(cfg.JWTSecretKey), nil })
	assert.NoError(t, err)
	assert.Equal(t, "renamed", claims["username"], "the username comes from the database, not the old token")
	assert.EqualValues(t, 7, claims["user_id"])
	assert.InDelta(t, time.Now().Add(-time.Hour).Unix(), claims["auth_time"], 2)

	// near the session cap: new expiry is clamped to auth_time+30h
	w = call(tokenAt(2*time.Hour, 25*time.Hour), true)
	cookies = w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.InDelta(t, 5*3600, cookies[0].MaxAge, 5)

	// at the cap: no further extension
	w = call(tokenAt(2*time.Hour, 28*time.Hour), true)
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("Set-Cookie"))

	// legacy token without auth_time: never refreshed
	legacy, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "u", "user_id": 7, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(cfg.JWTSecretKey))
	w = call(legacy, true)
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("Set-Cookie"))

	// bearer header: never refreshed
	w = call(tokenWithExp(2*time.Hour), false)
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("Set-Cookie"))

	// password changed after sign-in: no refresh, the cookie keeps working until it expires
	changed := time.Now().Add(-time.Minute)
	assert.NoError(t, db.Model(&user).Update("password_changed_at", changed).Error)
	w = call(tokenWithExp(2*time.Hour), true)
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("Set-Cookie"))

	// signed in again after that password change: refreshed as usual
	w = call(tokenAt(2*time.Hour, time.Second), true)
	assert.Len(t, w.Result().Cookies(), 1)

	// deleted user: rejected, no refresh
	assert.NoError(t, db.Delete(&user).Error)
	w = call(tokenWithExp(2*time.Hour), true)
	assert.Equal(t, 401, w.Code)
	assert.Empty(t, w.Header().Get("Set-Cookie"))
}

func TestAuthJWTOfDeletedUserRejected(t *testing.T) {
	db, router := setupAuthTestRouter()
	cfg := &config.Config{JWTSecretKey: "test-secret-key-32-chars-minimum!"}
	var user models.User
	db.First(&user)
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username, "user_id": user.ID, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(cfg.JWTSecretKey))

	call := func() int {
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, call())
	db.Delete(&user)
	assert.Equal(t, http.StatusUnauthorized, call())
}
