package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateJWTMaxSessionHours(t *testing.T) {
	tests := []struct {
		name    string
		expiry  int
		max     int
		wantErr bool
	}{
		{"default", 96, 720, false},
		{"equal to expiry", 96, 96, false},
		{"upper bound", 96, 8760, false},
		{"below expiry", 96, 95, true},
		{"above one year", 96, 8761, true},
		{"zero", 96, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{JWTExpiryHours: tt.expiry, JWTMaxSessionHours: tt.max}
			found := false
			for _, e := range cfg.Validate() {
				if e.Field == "JWT_MAX_SESSION_HOURS" {
					found = true
				}
			}
			assert.Equal(t, tt.wantErr, found)
		})
	}
}
