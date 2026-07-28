package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mycorrhizal/config"
	"mycorrhizal/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// newTestProvider builds an OIDCProvider whose UserInfo endpoint returns the
// given payload, without needing live discovery.
func newTestProvider(t *testing.T, userInfo map[string]any) (*OIDCProvider, func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(userInfo))
	}))

	cfg := oidc.ProviderConfig{
		IssuerURL:   server.URL,
		AuthURL:     server.URL + "/auth",
		TokenURL:    server.URL + "/token",
		UserInfoURL: server.URL,
		JWKSURL:     server.URL + "/keys",
	}

	return &OIDCProvider{provider: cfg.NewProvider(context.Background())}, server.Close
}

func testToken() *oauth2.Token {
	return &oauth2.Token{AccessToken: "test-access-token", TokenType: "Bearer"}
}

// The Authelia case from upstream issue #189: the ID token carries only `sub`,
// and email/name have to come from UserInfo.
func TestEnrichFromUserInfoFillsMissingClaims(t *testing.T) {
	provider, cleanup := newTestProvider(t, map[string]any{
		"sub":            "user-123",
		"email":          "ada@example.com",
		"email_verified": true,
		"name":           "Ada Lovelace",
	})
	defer cleanup()

	claims := &OIDCClaims{Subject: "user-123"}
	require.NoError(t, provider.enrichFromUserInfo(context.Background(), "user-123", testToken(), claims))

	assert.Equal(t, "ada@example.com", claims.Email)
	assert.True(t, claims.EmailVerified)
	assert.Equal(t, "Ada Lovelace", claims.Name)
}

// OIDC Core 5.3.2: a UserInfo response whose `sub` differs from the ID token's
// must be discarded. Accepting it would bind one identity's profile — and
// therefore, via email matching, potentially one identity's account — to a
// different subject's session.
func TestEnrichFromUserInfoRejectsSubjectMismatch(t *testing.T) {
	provider, cleanup := newTestProvider(t, map[string]any{
		"sub":            "attacker-999",
		"email":          "victim@example.com",
		"email_verified": true,
		"name":           "Victim",
	})
	defer cleanup()

	claims := &OIDCClaims{Subject: "user-123"}
	err := provider.enrichFromUserInfo(context.Background(), "user-123", testToken(), claims)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")

	// Nothing from the mismatched response may leak into the claims.
	assert.Empty(t, claims.Email)
	assert.Empty(t, claims.Name)
	assert.False(t, claims.EmailVerified)
}

// An ID token that already carries an email must not have it overwritten.
func TestEnrichFromUserInfoDoesNotOverwriteExistingEmail(t *testing.T) {
	provider, cleanup := newTestProvider(t, map[string]any{
		"sub":            "user-123",
		"email":          "other@example.com",
		"email_verified": true,
	})
	defer cleanup()

	claims := &OIDCClaims{Subject: "user-123", Email: "original@example.com"}
	require.NoError(t, provider.enrichFromUserInfo(context.Background(), "user-123", testToken(), claims))

	assert.Equal(t, "original@example.com", claims.Email)
}

// Falls back to preferred_username when the provider has no `name`.
func TestEnrichFromUserInfoUsesPreferredUsername(t *testing.T) {
	provider, cleanup := newTestProvider(t, map[string]any{
		"sub":                "user-123",
		"email":              "ada@example.com",
		"preferred_username": "ada",
	})
	defer cleanup()

	claims := &OIDCClaims{Subject: "user-123"}
	require.NoError(t, provider.enrichFromUserInfo(context.Background(), "user-123", testToken(), claims))

	assert.Equal(t, "ada", claims.Name)
}

func TestEnrichFromUserInfoRequiresToken(t *testing.T) {
	provider, cleanup := newTestProvider(t, map[string]any{"sub": "user-123"})
	defer cleanup()

	claims := &OIDCClaims{Subject: "user-123"}
	assert.Error(t, provider.enrichFromUserInfo(context.Background(), "user-123", nil, claims))
}

// setupUserDB gives an in-memory database with just the users table.
func setupUserDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	return db
}

// Provisioning without an email used to insert an empty string into a UNIQUE
// NOT NULL column, so the first such login succeeded and every later one failed
// on the constraint. Refusing up front turns that into a diagnosable error.
func TestFindOrProvisionUserRejectsMissingEmail(t *testing.T) {
	db := setupUserDB(t)
	cfg := &config.Config{OIDC: config.OIDCConfig{AllowAutoProvision: true}}

	claims := &OIDCClaims{Subject: "user-123", Provider: "https://idp.example.com", Name: "No Email"}
	user, err := FindOrProvisionUser(db, claims, cfg)

	require.ErrorIs(t, err, ErrOIDCNoEmail)
	assert.Nil(t, user)

	var count int64
	require.NoError(t, db.Model(&models.User{}).Count(&count).Error)
	assert.Zero(t, count, "no user should be created without an email")
}

func TestFindOrProvisionUserCreatesWithEmail(t *testing.T) {
	db := setupUserDB(t)
	cfg := &config.Config{OIDC: config.OIDCConfig{AllowAutoProvision: true}}

	claims := &OIDCClaims{
		Subject:       "user-123",
		Provider:      "https://idp.example.com",
		Email:         "ada@example.com",
		EmailVerified: true,
		Name:          "Ada Lovelace",
	}
	user, err := FindOrProvisionUser(db, claims, cfg)

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "ada@example.com", user.Email)
	assert.Equal(t, "ada", user.Username)
}

// Two different subjects that both lack an email must both be refused, rather
// than the first succeeding and the second hitting a UNIQUE violation.
func TestFindOrProvisionUserRejectsSecondMissingEmail(t *testing.T) {
	db := setupUserDB(t)
	cfg := &config.Config{OIDC: config.OIDCConfig{AllowAutoProvision: true}}

	for _, subject := range []string{"user-1", "user-2"} {
		_, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  subject,
			Provider: "https://idp.example.com",
		}, cfg)
		require.ErrorIs(t, err, ErrOIDCNoEmail, "subject %s", subject)
	}
}
