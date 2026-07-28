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
