package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// newTestProvider spins up a minimal OIDC discovery + userinfo endpoint and returns a
// provider pointed at it. userInfo is served as-is from the userinfo endpoint.
func newTestProvider(t *testing.T, userInfo map[string]any) *OIDCProvider {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/authorize",
			"token_endpoint":         server.URL + "/token",
			"jwks_uri":               server.URL + "/jwks",
			"userinfo_endpoint":      server.URL + "/userinfo",
		})
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfo)
	})

	provider, err := oidc.NewProvider(context.Background(), server.URL)
	require.NoError(t, err)

	return &OIDCProvider{provider: provider}
}

func testToken() *oauth2.Token {
	return &oauth2.Token{AccessToken: "test-access-token", TokenType: "Bearer"}
}

func TestEnrichFromUserInfo_FillsMissingClaims(t *testing.T) {
	// Authelia-style: the ID token carries only `sub`.
	provider := newTestProvider(t, map[string]any{
		"sub":                "authelia-subject",
		"email":              "jane@example.com",
		"email_verified":     true,
		"name":               "Jane Doe",
		"preferred_username": "jane",
	})

	claims := &OIDCClaims{Subject: "authelia-subject", Provider: "https://auth.example.com"}

	err := provider.EnrichFromUserInfo(context.Background(), testToken(), claims)

	assert.NoError(t, err)
	assert.Equal(t, "jane@example.com", claims.Email)
	assert.True(t, claims.EmailVerified)
	assert.Equal(t, "Jane Doe", claims.Name)
	assert.Equal(t, "jane", claims.PreferredUsername)
}

func TestEnrichFromUserInfo_DoesNotOverwriteIDTokenClaims(t *testing.T) {
	provider := newTestProvider(t, map[string]any{
		"sub":                "subject-1",
		"email":              "userinfo@example.com",
		"name":               "Userinfo Name",
		"preferred_username": "userinfo_name",
	})

	claims := &OIDCClaims{
		Subject:       "subject-1",
		Email:         "idtoken@example.com",
		EmailVerified: true,
		Name:          "ID Token Name",
	}

	err := provider.EnrichFromUserInfo(context.Background(), testToken(), claims)

	assert.NoError(t, err)
	assert.Equal(t, "idtoken@example.com", claims.Email)
	assert.True(t, claims.EmailVerified)
	assert.Equal(t, "ID Token Name", claims.Name)
	// Absent from the ID token, so it is still taken from userinfo.
	assert.Equal(t, "userinfo_name", claims.PreferredUsername)
}

func TestEnrichFromUserInfo_RejectsSubjectMismatch(t *testing.T) {
	provider := newTestProvider(t, map[string]any{
		"sub":   "someone-else",
		"email": "attacker@example.com",
	})

	claims := &OIDCClaims{Subject: "subject-1"}

	err := provider.EnrichFromUserInfo(context.Background(), testToken(), claims)

	assert.Error(t, err)
	assert.Empty(t, claims.Email)
}

func TestNeedsUserInfo(t *testing.T) {
	tests := []struct {
		name   string
		claims OIDCClaims
		want   bool
	}{
		{"empty claims", OIDCClaims{Subject: "s"}, true},
		{"email only", OIDCClaims{Subject: "s", Email: "a@b.com"}, true},
		{"name only", OIDCClaims{Subject: "s", Name: "A B"}, true},
		{"email and name", OIDCClaims{Subject: "s", Email: "a@b.com", Name: "A B"}, false},
		{"email and preferred_username", OIDCClaims{Subject: "s", Email: "a@b.com", PreferredUsername: "ab"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.claims.NeedsUserInfo())
		})
	}
}

func TestDeriveUsername(t *testing.T) {
	tests := []struct {
		name   string
		claims OIDCClaims
		want   string
	}{
		{"prefers preferred_username", OIDCClaims{PreferredUsername: "Jane", Email: "other@example.com", Name: "Jane Doe"}, "jane"},
		{"keeps local part of a UPN-style preferred_username", OIDCClaims{PreferredUsername: "jane.doe@corp.example.com", Name: "Jane Doe"}, "jane_doe"},
		{"falls back to email local-part", OIDCClaims{Email: "Jane.Doe@example.com", Name: "Jane Doe"}, "jane_doe"},
		{"falls back to name", OIDCClaims{Name: "Jane Doe"}, "jane_doe"},
		{"strips unsupported characters", OIDCClaims{PreferredUsername: "jane+tag!"}, "janetag"},
		{"defaults when nothing usable", OIDCClaims{}, "oidc_user"},
		{"defaults when claims sanitize to empty", OIDCClaims{PreferredUsername: "!!!", Name: "***"}, "oidc_user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveUsername(&tt.claims))
		})
	}
}
