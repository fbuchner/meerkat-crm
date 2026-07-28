package services

import (
	"encoding/json"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Providers disagree on the JSON type of email_verified. A plain bool field
// rejects the string form outright, and because idToken.Claims decodes the whole
// struct at once, one bad field fails every claim and the login with it.
func TestIDTokenClaimsEmailVerifiedAcceptsBothForms(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"boolean true", `{"email_verified": true}`, true},
		{"boolean false", `{"email_verified": false}`, false},
		{"string true", `{"email_verified": "true"}`, true},
		{"string false", `{"email_verified": "false"}`, false},
		{"absent", `{}`, false},
		{"null", `{"email_verified": null}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var claims idTokenClaims
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &claims),
				"claim decode must not fail on a well-formed provider response")
			assert.Equal(t, tt.want, bool(claims.EmailVerified))
		})
	}
}

// A genuinely malformed value should still be an error rather than silently
// becoming false, since email_verified gates account linking.
func TestIDTokenClaimsEmailVerifiedRejectsGarbage(t *testing.T) {
	var claims idTokenClaims
	assert.Error(t, json.Unmarshal([]byte(`{"email_verified": "yes"}`), &claims))
	assert.Error(t, json.Unmarshal([]byte(`{"email_verified": 1}`), &claims))
}

// The string form must not break the other claims decoded alongside it.
func TestIDTokenClaimsDecodeAlongsideStringEmailVerified(t *testing.T) {
	var claims idTokenClaims
	require.NoError(t, json.Unmarshal([]byte(`{
		"email": "ada@example.com",
		"email_verified": "true",
		"name": "Ada Lovelace"
	}`), &claims))

	assert.Equal(t, "ada@example.com", claims.Email)
	assert.True(t, bool(claims.EmailVerified))
	assert.Equal(t, "Ada Lovelace", claims.Name)
}

func TestIDTokenClaimsDisplayName(t *testing.T) {
	assert.Equal(t, "Ada Lovelace", idTokenClaims{Name: "Ada Lovelace", PreferredUsername: "ada"}.displayName())
	assert.Equal(t, "ada", idTokenClaims{PreferredUsername: "ada"}.displayName())
	assert.Equal(t, "", idTokenClaims{}.displayName())
}

// OIDC Core 3.1.3.7 steps 4-5. go-oidc checks that our client_id appears in
// `aud` but ignores `azp`, so these are ours to enforce.
func TestVerifyAuthorizedParty(t *testing.T) {
	provider := &OIDCProvider{}
	provider.oauth2Cfg.ClientID = "our-client"

	tests := []struct {
		name      string
		audience  []string
		azp       string
		wantError string
	}{
		{"single audience, no azp", []string{"our-client"}, "", ""},
		{"single audience, matching azp", []string{"our-client"}, "our-client", ""},
		{"multiple audiences with matching azp", []string{"our-client", "other"}, "our-client", ""},
		{"multiple audiences without azp", []string{"our-client", "other"}, "", "multiple audiences"},
		{"azp for a different client", []string{"our-client"}, "other-client", "is not this client"},
		{"multiple audiences, azp for another client", []string{"our-client", "other"}, "other", "is not this client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.verifyAuthorizedParty(&oidc.IDToken{Audience: tt.audience}, tt.azp)
			if tt.wantError == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}
