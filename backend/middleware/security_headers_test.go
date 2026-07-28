package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		enableHSTS bool
	}{
		{name: "HSTS disabled (plain HTTP)", enableHSTS: false},
		{name: "HSTS enabled (HTTPS)", enableHSTS: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeadersMiddleware(tt.enableHSTS))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
			assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", w.Header().Get("Content-Security-Policy"))

			if tt.enableHSTS {
				assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
			} else {
				assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
			}
		})
	}
}

func TestSecurityHeadersMiddleware_CSPAppliesToAllResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// The CSP is defense-in-depth for any response, including error responses
	// and non-JSON payloads (e.g. the image proxy) - verify it isn't skipped
	// on a non-2xx response.
	router := gin.New()
	router.Use(SecurityHeadersMiddleware(false))
	router.GET("/not-found", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", w.Header().Get("Content-Security-Policy"))
}
