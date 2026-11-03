package services

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"meerkat/config"
	"meerkat/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
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

// oidcSigningKeys are generated once because RSA generation is slow, especially under -race.
var oidcSigningKeys = sync.OnceValue(func() [2]*rsa.PrivateKey {
	var keys [2]*rsa.PrivateKey
	for i := range keys {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		keys[i] = key
	}
	return keys
})

const fakeIdPClientID = "meerkat-client"

// fakeIdP is an OIDC provider that signs real ID tokens. tokenResponse decides what the token
// endpoint returns, so a test can serve whatever token it wants to see rejected.
type fakeIdP struct {
	server        *httptest.Server
	tokenResponse func(w http.ResponseWriter, r *http.Request)
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()

	idp := &fakeIdP{}
	mux := http.NewServeMux()
	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 idp.server.URL,
			"authorization_endpoint": idp.server.URL + "/authorize",
			"token_endpoint":         idp.server.URL + "/token",
			"jwks_uri":               idp.server.URL + "/jwks",
			"userinfo_endpoint":      idp.server.URL + "/userinfo",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		public := oidcSigningKeys()[0].PublicKey
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"kid": "test-key",
			"n":   base64.RawURLEncoding.EncodeToString(public.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(public.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		idp.tokenResponse(w, r)
	})
	return idp
}

// serveIDToken makes the token endpoint return rawIDToken alongside an access token.
func (idp *fakeIdP) serveIDToken(rawIDToken string) {
	idp.tokenResponse = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"id_token":     rawIDToken,
		})
	}
}

// claims returns a valid claim set, which each test then breaks in its own way.
func (idp *fakeIdP) claims() map[string]any {
	now := time.Now()
	return map[string]any{
		"iss":   idp.server.URL,
		"aud":   fakeIdPClientID,
		"sub":   "subject-1",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
		"nonce": "nonce-1",
	}
}

func (idp *fakeIdP) provider(t *testing.T) *OIDCProvider {
	t.Helper()
	provider, err := InitOIDCProvider(context.Background(), &config.Config{OIDC: config.OIDCConfig{
		ProviderURL:  idp.server.URL,
		ClientID:     fakeIdPClientID,
		ClientSecret: "client-secret",
		RedirectURL:  "https://crm.example.com/callback",
	}})
	require.NoError(t, err)
	return provider
}

// signIDToken builds an RS256 JWT by hand, which avoids a direct dependency on a JOSE library.
func signIDToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	encode := func(v any) string {
		raw, err := json.Marshal(v)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	signingInput := encode(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "test-key"}) + "." + encode(claims)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestInitOIDCProvider_Failures(t *testing.T) {
	t.Run("provider unreachable", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		server.Close()

		provider, err := InitOIDCProvider(context.Background(), &config.Config{OIDC: config.OIDCConfig{ProviderURL: server.URL}})

		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("no discovery document", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		defer server.Close()

		provider, err := InitOIDCProvider(context.Background(), &config.Config{OIDC: config.OIDCConfig{ProviderURL: server.URL}})

		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("discovery document for a different issuer", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"issuer": "https://attacker.example.com"})
		}))
		defer server.Close()

		provider, err := InitOIDCProvider(context.Background(), &config.Config{OIDC: config.OIDCConfig{ProviderURL: server.URL}})

		assert.Error(t, err)
		assert.Nil(t, provider)
	})
}

func TestBuildAuthURL_CarriesTheLoginParameters(t *testing.T) {
	idp := newFakeIdP(t)

	authURL, err := url.Parse(idp.provider(t).BuildAuthURL("state-1", "nonce-1"))

	require.NoError(t, err)
	assert.Equal(t, idp.server.URL+"/authorize", authURL.Scheme+"://"+authURL.Host+authURL.Path)
	query := authURL.Query()
	assert.Equal(t, "state-1", query.Get("state"))
	assert.Equal(t, "nonce-1", query.Get("nonce"))
	assert.Equal(t, fakeIdPClientID, query.Get("client_id"))
	assert.Equal(t, "https://crm.example.com/callback", query.Get("redirect_uri"))
	assert.Equal(t, "code", query.Get("response_type"))
	assert.ElementsMatch(t, []string{"openid", "email", "profile"}, strings.Fields(query.Get("scope")))
	assert.Empty(t, query.Get("client_secret"))
}

func TestGenerateStateToken_IsRandomHex(t *testing.T) {
	first, err := GenerateStateToken()
	require.NoError(t, err)
	second, err := GenerateStateToken()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	raw, err := hex.DecodeString(first)
	require.NoError(t, err)
	assert.Len(t, raw, 32)
}

func TestExchangeAndVerify_AcceptsValidToken(t *testing.T) {
	idp := newFakeIdP(t)
	claims := idp.claims()
	claims["email"] = "jane@example.com"
	claims["email_verified"] = true
	claims["name"] = "Jane Doe"
	claims["preferred_username"] = "jane"

	var postedCode, postedSecret string
	rawIDToken := signIDToken(t, oidcSigningKeys()[0], claims)
	idp.serveIDToken(rawIDToken)
	respond := idp.tokenResponse
	idp.tokenResponse = func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		postedCode = r.PostForm.Get("code")
		_, postedSecret, _ = r.BasicAuth()
		respond(w, r)
	}

	idToken, token, err := idp.provider(t).ExchangeAndVerify(context.Background(), "auth-code")

	require.NoError(t, err)
	assert.Equal(t, "auth-code", postedCode)
	assert.Equal(t, "client-secret", postedSecret)
	assert.Equal(t, "test-access-token", token.AccessToken)
	assert.Equal(t, "subject-1", idToken.Subject)
	assert.Equal(t, "nonce-1", idToken.Nonce, "the controller compares this against its cookie")

	extracted, err := ExtractClaims(idToken, "https://idp.example.com")
	require.NoError(t, err)
	assert.Equal(t, &OIDCClaims{
		Subject:           "subject-1",
		Email:             "jane@example.com",
		EmailVerified:     true,
		Name:              "Jane Doe",
		PreferredUsername: "jane",
		Provider:          "https://idp.example.com",
	}, extracted)
}

func TestExchangeAndVerify_RejectsBadTokens(t *testing.T) {
	idp := newFakeIdP(t)
	valid := signIDToken(t, oidcSigningKeys()[0], idp.claims())
	parts := strings.Split(valid, ".")

	with := func(mutate func(map[string]any)) string {
		claims := idp.claims()
		mutate(claims)
		return signIDToken(t, oidcSigningKeys()[0], claims)
	}

	unsignedHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))

	tests := []struct {
		name  string
		token string
	}{
		{"signed by another key", signIDToken(t, oidcSigningKeys()[1], idp.claims())},
		{"tampered payload", parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"admin"}`)) + "." + parts[2]},
		{"unsigned", unsignedHeader + "." + parts[1] + "."},
		{"not a JWT", "not-a-jwt"},
		{"wrong issuer", with(func(c map[string]any) { c["iss"] = "https://attacker.example.com" })},
		{"wrong audience", with(func(c map[string]any) { c["aud"] = "another-client" })},
		{"expired", with(func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() })},
		{"missing expiry", with(func(c map[string]any) { delete(c, "exp") })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idp.serveIDToken(tt.token)

			idToken, token, err := idp.provider(t).ExchangeAndVerify(context.Background(), "auth-code")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to verify id_token")
			assert.Nil(t, idToken)
			assert.Nil(t, token)
		})
	}
}

func TestExchangeAndVerify_TokenEndpointFailures(t *testing.T) {
	idp := newFakeIdP(t)

	t.Run("code rejected", func(t *testing.T) {
		idp.tokenResponse = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"invalid_grant"}`)
		}

		_, _, err := idp.provider(t).ExchangeAndVerify(context.Background(), "used-code")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to exchange code")
	})

	t.Run("no id_token in the response", func(t *testing.T) {
		idp.tokenResponse = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"test-access-token","token_type":"Bearer"}`)
		}

		_, _, err := idp.provider(t).ExchangeAndVerify(context.Background(), "auth-code")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing id_token")
	})
}

func TestEnrichFromUserInfo_Failures(t *testing.T) {
	t.Run("endpoint refuses the access token", func(t *testing.T) {
		provider := newTestProvider(t, map[string]any{"sub": "subject-1", "email": "jane@example.com"})
		claims := &OIDCClaims{Subject: "subject-1"}

		err := provider.EnrichFromUserInfo(context.Background(), &oauth2.Token{AccessToken: "wrong", TokenType: "Bearer"}, claims)

		require.Error(t, err)
		assert.Empty(t, claims.Email)
	})

	t.Run("claims of the wrong type", func(t *testing.T) {
		provider := newTestProvider(t, map[string]any{"sub": "subject-1", "email": "jane@example.com", "name": 42})
		claims := &OIDCClaims{Subject: "subject-1"}

		err := provider.EnrichFromUserInfo(context.Background(), testToken(), claims)

		require.Error(t, err)
		assert.Empty(t, claims.Email)
	})
}

func TestProviderName(t *testing.T) {
	assert.Equal(t, "auth.example.com", ProviderName("https://auth.example.com/realms/main"))
	assert.Equal(t, "auth.example.com:8443", ProviderName("https://auth.example.com:8443"))
	assert.Equal(t, "SSO", ProviderName("not a url"))
	assert.Equal(t, "SSO", ProviderName("://broken"))
	assert.Equal(t, "SSO", ProviderName(""))
}

const testOIDCProvider = "https://idp.example.com"

func seedLocalUser(t *testing.T, db *gorm.DB, username, email string) models.User {
	t.Helper()
	user := models.User{Username: username, Email: email, Password: "existing-hash"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func seedOIDCUser(t *testing.T, db *gorm.DB, username, email, subject string) models.User {
	t.Helper()
	provider := testOIDCProvider
	user := models.User{Username: username, Email: email, OIDCSubject: &subject, OIDCProvider: &provider}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func countUsers(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Unscoped().Model(&models.User{}).Count(&count).Error)
	return count
}

func oidcConfig(autoProvision, trustEmail bool) *config.Config {
	return &config.Config{OIDC: config.OIDCConfig{AllowAutoProvision: autoProvision, TrustEmail: trustEmail}}
}

func TestFindOrProvisionUser_KnownSubject(t *testing.T) {
	t.Run("returns the linked account", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		existing := seedOIDCUser(t, db, "jane", "jane@example.com", "subject-1")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "subject-1",
			Provider: testOIDCProvider,
			Email:    "renamed@example.com",
		}, oidcConfig(false, false))

		require.NoError(t, err)
		assert.Equal(t, existing.ID, user.ID)
		assert.Equal(t, "jane@example.com", user.Email, "an existing email is not replaced by the claim")
		assert.EqualValues(t, 1, countUsers(t, db))
	})

	t.Run("same subject from another provider is a different identity", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		seedOIDCUser(t, db, "jane", "jane@example.com", "subject-1")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "subject-1",
			Provider: "https://other-idp.example.com",
		}, oidcConfig(false, false))

		assert.ErrorIs(t, err, ErrOIDCUserNotFound)
		assert.Nil(t, user)
	})

	t.Run("backfills a missing email in lower case", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		existing := seedOIDCUser(t, db, "jane", "", "subject-1")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "subject-1",
			Provider: testOIDCProvider,
			Email:    "Jane@Example.com",
		}, oidcConfig(false, false))

		require.NoError(t, err)
		assert.Equal(t, "jane@example.com", user.Email)
		var stored models.User
		require.NoError(t, db.First(&stored, existing.ID).Error)
		assert.Equal(t, "jane@example.com", stored.Email)
	})

	t.Run("backfill conflict does not block the login", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		seedLocalUser(t, db, "taken", "taken@example.com")
		existing := seedOIDCUser(t, db, "jane", "", "subject-1")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "subject-1",
			Provider: testOIDCProvider,
			Email:    "taken@example.com",
		}, oidcConfig(false, false))

		require.NoError(t, err)
		assert.Equal(t, existing.ID, user.ID)
		assert.Empty(t, user.Email)
		var stored models.User
		require.NoError(t, db.First(&stored, existing.ID).Error)
		assert.Empty(t, stored.Email)
	})
}

func TestFindOrProvisionUser_EmailLinking(t *testing.T) {
	t.Run("verified email links the identity to the existing account", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		existing := seedLocalUser(t, db, "jane", "jane@example.com")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:       "subject-1",
			Provider:      testOIDCProvider,
			Email:         "JANE@example.com",
			EmailVerified: true,
		}, oidcConfig(false, false))

		require.NoError(t, err)
		assert.Equal(t, existing.ID, user.ID)
		assert.EqualValues(t, 1, countUsers(t, db))

		var stored models.User
		require.NoError(t, db.First(&stored, existing.ID).Error)
		require.NotNil(t, stored.OIDCSubject)
		require.NotNil(t, stored.OIDCProvider)
		assert.Equal(t, "subject-1", *stored.OIDCSubject)
		assert.Equal(t, testOIDCProvider, *stored.OIDCProvider)
		assert.Equal(t, "existing-hash", stored.Password, "linking keeps the password login working")

		// The next login takes the subject path, even with no email claim at all.
		again, err := FindOrProvisionUser(db, &OIDCClaims{Subject: "subject-1", Provider: testOIDCProvider}, oidcConfig(false, false))
		require.NoError(t, err)
		assert.Equal(t, existing.ID, again.ID)
	})

	t.Run("unverified email must not take over an account", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		existing := seedLocalUser(t, db, "admin", "admin@example.com")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "attacker",
			Provider: testOIDCProvider,
			Email:    "admin@example.com",
		}, oidcConfig(false, false))

		assert.ErrorIs(t, err, ErrOIDCUserNotFound)
		assert.Nil(t, user)
		var stored models.User
		require.NoError(t, db.First(&stored, existing.ID).Error)
		assert.Nil(t, stored.OIDCSubject)
	})

	t.Run("unverified email is not linked by auto-provisioning either", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		existing := seedLocalUser(t, db, "admin", "admin@example.com")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "attacker",
			Provider: testOIDCProvider,
			Email:    "admin@example.com",
		}, oidcConfig(true, false))

		// The email is taken, so no account can be created; what matters is no access.
		require.Error(t, err)
		assert.Nil(t, user)
		assert.EqualValues(t, 1, countUsers(t, db))
		var stored models.User
		require.NoError(t, db.First(&stored, existing.ID).Error)
		assert.Nil(t, stored.OIDCSubject)
	})

	t.Run("OIDC_TRUST_EMAIL links an unverified email", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		existing := seedLocalUser(t, db, "jane", "jane@example.com")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "subject-1",
			Provider: testOIDCProvider,
			Email:    "jane@example.com",
		}, oidcConfig(false, true))

		require.NoError(t, err)
		assert.Equal(t, existing.ID, user.ID)
		require.NotNil(t, user.OIDCSubject)
		assert.Equal(t, "subject-1", *user.OIDCSubject)
	})

	t.Run("no email claim never matches an account without an email", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		seedOIDCUser(t, db, "no_email", "", "someone-else")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "subject-1",
			Provider: testOIDCProvider,
		}, oidcConfig(false, true))

		assert.ErrorIs(t, err, ErrOIDCUserNotFound)
		assert.Nil(t, user)
	})

	t.Run("deleted account is not linked", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		existing := seedLocalUser(t, db, "gone", "gone@example.com")
		require.NoError(t, db.Delete(&existing).Error)

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:       "subject-1",
			Provider:      testOIDCProvider,
			Email:         "gone@example.com",
			EmailVerified: true,
		}, oidcConfig(false, false))

		assert.ErrorIs(t, err, ErrOIDCUserNotFound)
		assert.Nil(t, user)
	})

	t.Run("deleted account cannot log back in through its old subject", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		existing := seedOIDCUser(t, db, "gone", "gone@example.com", "subject-1")
		require.NoError(t, db.Delete(&existing).Error)

		user, err := FindOrProvisionUser(db, &OIDCClaims{Subject: "subject-1", Provider: testOIDCProvider}, oidcConfig(false, false))

		assert.ErrorIs(t, err, ErrOIDCUserNotFound)
		assert.Nil(t, user)
	})
}

func TestFindOrProvisionUser_AutoProvision(t *testing.T) {
	t.Run("disabled creates nothing", func(t *testing.T) {
		db := newImportSessionTestDB(t)

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:       "subject-1",
			Provider:      testOIDCProvider,
			Email:         "new@example.com",
			EmailVerified: true,
		}, oidcConfig(false, false))

		assert.ErrorIs(t, err, ErrOIDCUserNotFound)
		assert.Nil(t, user)
		assert.Zero(t, countUsers(t, db))
	})

	t.Run("creates an unprivileged account with no password", func(t *testing.T) {
		db := newImportSessionTestDB(t)

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:           "subject-1",
			Provider:          testOIDCProvider,
			Email:             "New.User@Example.com",
			EmailVerified:     true,
			PreferredUsername: "New.User",
		}, oidcConfig(true, false))

		require.NoError(t, err)
		var stored models.User
		require.NoError(t, db.First(&stored, user.ID).Error)
		assert.Equal(t, "new_user", stored.Username)
		assert.Equal(t, "new.user@example.com", stored.Email)
		assert.False(t, stored.IsAdmin)
		assert.Empty(t, stored.Password)
		assert.Error(t, stored.CheckPassword(""), "an account with no password must not accept an empty one")
		require.NotNil(t, stored.OIDCSubject)
		assert.Equal(t, "subject-1", *stored.OIDCSubject)
		assert.Equal(t, testOIDCProvider, *stored.OIDCProvider)

		// A second login finds the same account rather than creating another.
		again, err := FindOrProvisionUser(db, &OIDCClaims{Subject: "subject-1", Provider: testOIDCProvider}, oidcConfig(true, false))
		require.NoError(t, err)
		assert.Equal(t, user.ID, again.ID)
		assert.EqualValues(t, 1, countUsers(t, db))
	})

	t.Run("works without an email claim", func(t *testing.T) {
		db := newImportSessionTestDB(t)

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:  "subject-1",
			Provider: testOIDCProvider,
			Name:     "Jane Doe",
		}, oidcConfig(true, false))

		require.NoError(t, err)
		assert.Equal(t, "jane_doe", user.Username)
		assert.Empty(t, user.Email)
	})

	t.Run("adds a suffix when the username is taken", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		seedLocalUser(t, db, "jane", "jane1@example.com")
		seedLocalUser(t, db, "jane1", "jane2@example.com")

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:           "subject-1",
			Provider:          testOIDCProvider,
			Email:             "jane3@example.com",
			PreferredUsername: "jane",
		}, oidcConfig(true, false))

		require.NoError(t, err)
		assert.Equal(t, "jane2", user.Username)
	})

	t.Run("gives up when no username variant is free", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		seedLocalUser(t, db, "jane", "jane@example.com")
		for i := 1; i < 100; i++ {
			seedLocalUser(t, db, fmt.Sprintf("jane%d", i), fmt.Sprintf("jane%d@example.com", i))
		}

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:           "subject-1",
			Provider:          testOIDCProvider,
			Email:             "new@example.com",
			PreferredUsername: "jane",
		}, oidcConfig(true, false))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unique username")
		assert.Nil(t, user)
	})

	t.Run("a second provider user without an email can also sign up", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		_, err := FindOrProvisionUser(db, &OIDCClaims{Subject: "subject-1", Provider: testOIDCProvider, Name: "Jane"}, oidcConfig(true, false))
		require.NoError(t, err)

		user, err := FindOrProvisionUser(db, &OIDCClaims{Subject: "subject-2", Provider: testOIDCProvider, Name: "Joe"}, oidcConfig(true, false))

		require.NoError(t, err)
		assert.Equal(t, "joe", user.Username)
	})

	t.Run("username held by a deleted account gets a suffix", func(t *testing.T) {
		db := newImportSessionTestDB(t)
		gone := seedLocalUser(t, db, "jane", "gone@example.com")
		require.NoError(t, db.Delete(&gone).Error)

		user, err := FindOrProvisionUser(db, &OIDCClaims{
			Subject:           "subject-1",
			Provider:          testOIDCProvider,
			Email:             "new@example.com",
			PreferredUsername: "jane",
		}, oidcConfig(true, false))

		require.NoError(t, err)
		assert.Equal(t, "jane1", user.Username)
	})
}

func TestFindOrProvisionUser_DatabaseFailureIsNotReportedAsUnknownUser(t *testing.T) {
	db := newImportSessionTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	user, err := FindOrProvisionUser(db, &OIDCClaims{Subject: "subject-1", Provider: testOIDCProvider}, oidcConfig(true, false))

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrOIDCUserNotFound)
	assert.Contains(t, err.Error(), "database error")
	assert.Nil(t, user)
}
