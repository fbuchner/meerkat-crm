package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"meerkat/logger"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runLogged serves one request through LoggingMiddleware and returns the decoded log entry.
func runLogged(t *testing.T, target string, handler gin.HandlerFunc) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	originalLogger, originalLevel := logger.Logger, zerolog.GlobalLevel()
	logger.Logger = zerolog.New(&buf)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	t.Cleanup(func() {
		logger.Logger = originalLogger
		zerolog.SetGlobalLevel(originalLevel)
	})

	router := gin.New()
	router.Use(LoggingMiddleware())
	router.GET("/items", handler)

	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(httptest.NewRecorder(), req)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), "expected exactly one log line, got %q", buf.String())
	return entry
}

func TestLoggingMiddleware_RecordsRequestDetails(t *testing.T) {
	entry := runLogged(t, "/items?page=2", func(c *gin.Context) {
		c.Set("request_id", "req-123")
		c.Set("userID", uint(42))
		c.Status(http.StatusOK)
	})

	assert.Equal(t, "info", entry["level"])
	assert.Equal(t, "GET", entry["method"])
	assert.Equal(t, "/items", entry["path"])
	assert.Equal(t, "page=2", entry["query"])
	assert.EqualValues(t, http.StatusOK, entry["status"])
	assert.Equal(t, "req-123", entry["request_id"])
	assert.EqualValues(t, 42, entry["user_id"])
	assert.Equal(t, "test-agent", entry["user_agent"])
}

func TestLoggingMiddleware_OmitsAbsentContext(t *testing.T) {
	entry := runLogged(t, "/items", func(c *gin.Context) { c.Status(http.StatusOK) })

	assert.NotContains(t, entry, "request_id")
	assert.NotContains(t, entry, "user_id")
	assert.NotContains(t, entry, "query")
	assert.NotContains(t, entry, "error")
}

func TestLoggingMiddleware_LevelFollowsStatus(t *testing.T) {
	tests := []struct {
		status int
		level  string
	}{
		{http.StatusNoContent, "info"},
		{http.StatusFound, "info"},
		{http.StatusBadRequest, "warn"},
		{http.StatusNotFound, "warn"},
		{http.StatusInternalServerError, "error"},
		{http.StatusBadGateway, "error"},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			entry := runLogged(t, "/items", func(c *gin.Context) { c.Status(tt.status) })

			assert.Equal(t, tt.level, entry["level"])
			assert.EqualValues(t, tt.status, entry["status"])
		})
	}
}

func TestLoggingMiddleware_IncludesHandlerErrors(t *testing.T) {
	entry := runLogged(t, "/items", func(c *gin.Context) {
		_ = c.Error(errors.New("database is on fire"))
		c.Status(http.StatusInternalServerError)
	})

	assert.Equal(t, "error", entry["level"])
	assert.Contains(t, entry["error"], "database is on fire")
}
