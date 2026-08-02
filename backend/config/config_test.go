package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// validConfig returns a Config that passes Validate() with no errors, so
// individual tests can mutate just the field(s) they care about.
func validConfig() *Config {
	return &Config{
		DBPath:           "test.db",
		ReminderTime:     "12:00",
		ReminderTimezone: "UTC",
		FrontendURL:      "https://crm.example.com",
		Port:             "8080",
		JWTSecretKey:     "a-very-long-jwt-secret-key-that-is-32-chars",
		JWTExpiryHours:   96,
		ReadTimeout:      15,
		WriteTimeout:     15,
		IdleTimeout:      60,
		ProfilePhotoDir:  "/var/data/photos",
	}
}

func hasFieldError(errs []ValidationError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}

func TestValidate_ValidConfigHasNoErrors(t *testing.T) {
	cfg := validConfig()
	errs := cfg.Validate()
	assert.Empty(t, errs)
}

func TestValidate_FrontendURLWildcard(t *testing.T) {
	tests := []struct {
		name        string
		ginMode     string
		expectError bool
	}{
		{name: "wildcard with GIN_MODE unset (dev)", ginMode: "", expectError: false},
		{name: "wildcard with GIN_MODE=debug", ginMode: "debug", expectError: false},
		{name: "wildcard with GIN_MODE=test", ginMode: "test", expectError: false},
		{name: "wildcard with GIN_MODE=release", ginMode: "release", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GIN_MODE", tt.ginMode)

			cfg := validConfig()
			cfg.FrontendURL = "*"
			errs := cfg.Validate()

			if tt.expectError {
				assert.True(t, hasFieldError(errs, "FRONTEND_URL"), "expected a FRONTEND_URL validation error, got: %v", errs)
			} else {
				assert.False(t, hasFieldError(errs, "FRONTEND_URL"), "did not expect a FRONTEND_URL validation error, got: %v", errs)
			}
		})
	}
}

func TestValidate_SpecificFrontendURLAllowedInRelease(t *testing.T) {
	t.Setenv("GIN_MODE", "release")

	cfg := validConfig()
	cfg.FrontendURL = "https://crm.example.com"
	errs := cfg.Validate()

	assert.False(t, hasFieldError(errs, "FRONTEND_URL"), "a specific FRONTEND_URL should be allowed in release mode, got: %v", errs)
}

func TestValidate_EmptyFrontendURLStillRejected(t *testing.T) {
	t.Setenv("GIN_MODE", "")

	cfg := validConfig()
	cfg.FrontendURL = ""
	errs := cfg.Validate()

	assert.True(t, hasFieldError(errs, "FRONTEND_URL"), "empty FRONTEND_URL should still be rejected regardless of GIN_MODE, got: %v", errs)
}

func TestGetScopesEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "empty defaults to openid/email/profile", input: "", expected: []string{"openid", "email", "profile"}},
		{name: "single scope", input: "openid", expected: []string{"openid"}},
		{name: "comma-separated with whitespace trimmed", input: " openid, email ", expected: []string{"openid", "email"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getScopesEnv(tt.input))
		})
	}
}

func TestValidate_JWTSecretKey(t *testing.T) {
	cfg := validConfig()
	cfg.JWTSecretKey = "short"
	errs := cfg.Validate()
	assert.True(t, hasFieldError(errs, "JWT_SECRET_KEY"), "short JWT secret should error")

	cfg.JWTSecretKey = ""
	errs = cfg.Validate()
	assert.True(t, hasFieldError(errs, "JWT_SECRET_KEY"), "empty JWT secret should error")
}

func TestValidate_SQLiteDBPath(t *testing.T) {
	cfg := validConfig()
	cfg.DBPath = ""
	errs := cfg.Validate()
	assert.True(t, hasFieldError(errs, "SQLITE_DB_PATH"), "empty DB path should error")
}

func TestValidate_Port(t *testing.T) {
	cfg := validConfig()
	cfg.Port = "0"
	errs := cfg.Validate()
	assert.True(t, hasFieldError(errs, "PORT"), "port 0 should error")

	cfg.Port = "not-a-number"
	errs = cfg.Validate()
	assert.True(t, hasFieldError(errs, "PORT"), "non-numeric port should error")
}

func TestValidate_ReminderTime(t *testing.T) {
	cfg := validConfig()
	cfg.ReminderTime = "25:00"
	errs := cfg.Validate()
	assert.True(t, hasFieldError(errs, "REMINDER_TIME"), "invalid hour should error")

	cfg.ReminderTime = "12:99"
	errs = cfg.Validate()
	assert.True(t, hasFieldError(errs, "REMINDER_TIME"), "invalid minute should error")

	cfg.ReminderTime = "abc"
	errs = cfg.Validate()
	assert.True(t, hasFieldError(errs, "REMINDER_TIME"), "non-time string should error")
}

func TestValidate_JWTExpiryHours(t *testing.T) {
	cfg := validConfig()
	cfg.JWTExpiryHours = 0
	errs := cfg.Validate()
	assert.True(t, hasFieldError(errs, "JWT_EXPIRY_HOURS"), "zero expiry should error")

	cfg.JWTExpiryHours = -1
	errs = cfg.Validate()
	assert.True(t, hasFieldError(errs, "JWT_EXPIRY_HOURS"), "negative expiry should error")
}

func TestValidate_Timeouts(t *testing.T) {
	cfg := validConfig()
	cfg.ReadTimeout = 0
	errs := cfg.Validate()
	assert.True(t, hasFieldError(errs, "HTTP_READ_TIMEOUT"), "zero read timeout should error")

	cfg = validConfig()
	cfg.WriteTimeout = -1
	errs = cfg.Validate()
	assert.True(t, hasFieldError(errs, "HTTP_WRITE_TIMEOUT"), "negative write timeout should error")
}

func TestValidate_ReminderTimezone(t *testing.T) {
	cfg := validConfig()
	cfg.ReminderTimezone = "NotAReal/Timezone"
	errs := cfg.Validate()
	assert.True(t, hasFieldError(errs, "REMINDER_TIMEZONE"), "invalid timezone should error")
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-secret-key-that-is-long-enough-32")
	t.Setenv("PROFILE_PHOTO_DIR", "/tmp/photos")
	t.Setenv("SQLITE_DB_PATH", "/tmp/test.db")
	t.Setenv("FRONTEND_URL", "http://localhost:5173")

	cfg := LoadConfig()
	assert.NotNil(t, cfg)

	// Default timeout values
	assert.Equal(t, 15, cfg.ReadTimeout)
	assert.Equal(t, 15, cfg.WriteTimeout)
	assert.Equal(t, 60, cfg.IdleTimeout)

	// Default retention
	assert.Equal(t, 30, cfg.DeleteRetentionDays)
}

func TestLoadConfig_DeleteRetentionDays(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-secret-key-that-is-long-enough-32")
	t.Setenv("PROFILE_PHOTO_DIR", "/tmp/photos")
	t.Setenv("SQLITE_DB_PATH", "/tmp/test.db")
	t.Setenv("FRONTEND_URL", "http://localhost:5173")
	t.Setenv("DELETED_RETENTION_DAYS", "14")

	cfg := LoadConfig()
	assert.Equal(t, 14, cfg.DeleteRetentionDays)
}
