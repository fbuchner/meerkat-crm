package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBodySizeLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		maxBytes       int64
		bodySize       int
		expectedStatus int
	}{
		{
			name:           "body within limit",
			maxBytes:       1024,
			bodySize:       512,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "body at exact limit",
			maxBytes:       1024,
			bodySize:       1024,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "body exceeds limit",
			maxBytes:       1024,
			bodySize:       2048,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "empty body",
			maxBytes:       1024,
			bodySize:       0,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(BodySizeLimitMiddleware(tt.maxBytes))
			router.POST("/test", func(c *gin.Context) {
				// Try to read the body - this triggers the size check
				body := make([]byte, tt.bodySize+1)
				_, err := c.Request.Body.Read(body)
				if err != nil && err.Error() == "http: request body too large" {
					c.AbortWithStatus(http.StatusRequestEntityTooLarge)
					return
				}
				c.Status(http.StatusOK)
			})

			body := bytes.NewReader(bytes.Repeat([]byte("x"), tt.bodySize))
			req := httptest.NewRequest(http.MethodPost, "/test", body)
			req.Header.Set("Content-Type", "application/octet-stream")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestJSONBodySizeLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(JSONBodySizeLimitMiddleware())
	router.POST("/test", func(c *gin.Context) {
		body := make([]byte, MaxJSONBodySize+1)
		_, err := c.Request.Body.Read(body)
		if err != nil && err.Error() == "http: request body too large" {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	// Test body within limit
	t.Run("within limit", func(t *testing.T) {
		body := strings.NewReader(`{"test": "data"}`)
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test body exceeding 1MB limit
	t.Run("exceeds limit", func(t *testing.T) {
		// Create a body larger than 1MB
		largeBody := bytes.Repeat([]byte("x"), MaxJSONBodySize+1)
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})
}

func TestDefaultBodySizeLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Verify the default limit is 10MB
	assert.Equal(t, int64(10<<20), int64(DefaultMaxBodySize))

	router := gin.New()
	router.Use(DefaultBodySizeLimitMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Test small body passes
	body := strings.NewReader(`{"test": "data"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
