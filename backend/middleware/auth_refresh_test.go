package middleware

import (
	"meerkat/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestAuthCookieSlidingRefresh(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	cfg := &config.Config{JWTSecretKey: "test-secret-key-32-chars-minimum!", JWTExpiryHours: 10, JWTMaxSessionHours: 30}
	r := gin.New()
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
	_, err := jwt.ParseWithClaims(cookies[0].Value, claims, func(*jwt.Token) (any, error) { return []byte(cfg.JWTSecretKey), nil })
	assert.NoError(t, err)
	assert.Equal(t, "u", claims["username"])
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
}
