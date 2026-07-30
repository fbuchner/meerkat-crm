package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"mycorrhizal/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// newDiscoveryTestProvider builds an OIDCProvider via a real discovery fetch
// (oidc.NewProvider hitting /.well-known/openid-configuration), unlike
// newTestProvider (oidc_userinfo_test.go) which builds the *oidc.Provider
// directly via oidc.ProviderConfig and never populates its rawClaims -- that
// shortcut means end_session_endpoint (and any other field go-oidc doesn't
// explicitly parse) would never be resolvable through that harness.
func newDiscoveryTestProvider(t *testing.T, endSessionEndpoint string) (*OIDCProvider, func()) {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		endSessionJSON := ""
		if endSessionEndpoint != "" {
			endSessionJSON = fmt.Sprintf(`,"end_session_endpoint":%q`, server.URL+endSessionEndpoint)
		}
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"jwks_uri":%q%s}`,
			server.URL, server.URL+"/auth", server.URL+"/token", server.URL+"/keys", endSessionJSON)
	}))

	provider, err := oidc.NewProvider(context.Background(), server.URL)
	require.NoError(t, err)

	return &OIDCProvider{provider: provider}, server.Close
}

func TestEndSessionEndpoint_ResolvesFromDiscoveryDocument(t *testing.T) {
	p, cleanup := newDiscoveryTestProvider(t, "/end_session")
	defer cleanup()

	endpoint, err := p.EndSessionEndpoint()
	require.NoError(t, err)
	require.Contains(t, endpoint, "/end_session")
}

func TestEndSessionEndpoint_EmptyWhenProviderDoesNotAdvertiseOne(t *testing.T) {
	p, cleanup := newDiscoveryTestProvider(t, "")
	defer cleanup()

	endpoint, err := p.EndSessionEndpoint()
	require.NoError(t, err)
	require.Empty(t, endpoint)
}

// fakeIDP is a full OpenID Provider backed by httptest.Server: discovery,
// JWKS, token exchange and (optionally) UserInfo. Unlike newTestProvider and
// newDiscoveryTestProvider above, it issues a real RS256-signed ID token that
// go-oidc verifies against the JWKS over the wire -- this exercises
// InitOIDCProvider, ExchangeAndVerify and ClaimsFor against genuine
// cryptographic verification rather than a mocked-out success path.
type fakeIDP struct {
	Server   *httptest.Server
	key      *rsa.PrivateKey
	kid      string
	clientID string

	// IDTokenClaims seeds the ID token issued by /token. Tests may mutate
	// this directly (e.g. delete "email" to force ClaimsFor's UserInfo
	// fallback, or set a multi-valued "aud" to exercise azp checking).
	// "iss", "aud" (if unset), "exp" and "iat" are filled in at issuance
	// time, so callers only need to set what they care about.
	IDTokenClaims jwt.MapClaims

	// SigningKey, when set, signs the ID token instead of the IdP's own
	// key -- used to simulate a forged token from an untrusted party.
	SigningKey *rsa.PrivateKey

	// UserInfo, when non-nil, is served at /userinfo and advertised via
	// discovery's userinfo_endpoint. Must be set before InitOIDCProvider is
	// called, since the discovery document is only fetched once.
	UserInfo map[string]any
}

// newFakeIDP starts the fake IdP. clientID becomes the default "aud" claim on
// issued ID tokens, matching the client this IdP is meant to serve.
func newFakeIDP(t *testing.T, clientID string) *fakeIDP {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	idp := &fakeIDP{
		key:      key,
		kid:      "test-kid",
		clientID: clientID,
		IDTokenClaims: jwt.MapClaims{
			"sub":            "test-subject",
			"email":          "test-user@example.com",
			"email_verified": true,
			"name":           "Test User",
		},
	}
	idp.Server = httptest.NewServer(http.HandlerFunc(idp.handle))
	return idp
}

func (idp *fakeIDP) Close() { idp.Server.Close() }

func (idp *fakeIDP) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		userInfoJSON := ""
		if idp.UserInfo != nil {
			userInfoJSON = fmt.Sprintf(`,"userinfo_endpoint":%q`, idp.Server.URL+"/userinfo")
		}
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"jwks_uri":%q%s}`,
			idp.Server.URL, idp.Server.URL+"/auth", idp.Server.URL+"/token", idp.Server.URL+"/keys", userInfoJSON)
	case "/keys":
		n := base64.RawURLEncoding.EncodeToString(idp.key.PublicKey.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(idp.key.PublicKey.E)).Bytes())
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","kid":%q,"use":"sig","alg":"RS256","n":%q,"e":%q}]}`, idp.kid, n, e)
	case "/token":
		signed, err := idp.signIDToken()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, `{"access_token":"test-access-token","token_type":"Bearer","id_token":%q}`, signed)
	case "/userinfo":
		// Errors here surface as a broken response body rather than t.Fatal:
		// handlers run on the server's own goroutine, where failing the test
		// directly is not supported.
		if err := json.NewEncoder(w).Encode(idp.UserInfo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		http.NotFound(w, r)
	}
}

func (idp *fakeIDP) signIDToken() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{}
	for k, v := range idp.IDTokenClaims {
		claims[k] = v
	}
	if _, ok := claims["iss"]; !ok {
		claims["iss"] = idp.Server.URL
	}
	if _, ok := claims["aud"]; !ok {
		claims["aud"] = idp.clientID
	}
	claims["exp"] = now.Add(time.Hour).Unix()
	claims["iat"] = now.Unix()

	key := idp.key
	if idp.SigningKey != nil {
		key = idp.SigningKey
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = idp.kid
	return tok.SignedString(key)
}

// testOIDCConfig returns a config.Config pointed at the fake IdP, matching
// what LoadConfig would produce from OIDC_* env vars.
func testOIDCConfig(providerURL, clientID string) *config.Config {
	return &config.Config{
		OIDC: config.OIDCConfig{
			Enabled:      true,
			ProviderURL:  providerURL,
			ClientID:     clientID,
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://app.example.com/auth/oidc/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
	}
}

func TestInitOIDCProvider_Success(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	defer idp.Close()

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, "test-client", provider.oauth2Cfg.ClientID)
	assert.Equal(t, "test-client-secret", provider.oauth2Cfg.ClientSecret)
	assert.Equal(t, idp.Server.URL+"/auth", provider.oauth2Cfg.Endpoint.AuthURL)
	assert.Equal(t, idp.Server.URL+"/token", provider.oauth2Cfg.Endpoint.TokenURL)
	assert.Equal(t, []string{"openid", "email", "profile"}, provider.oauth2Cfg.Scopes)
	require.NotNil(t, provider.verifier)
}

func TestInitOIDCProvider_FailsOnUnreachableDiscovery(t *testing.T) {
	// A bare, unrouted httptest server 404s on the discovery path, which is
	// the same failure shape as a mistyped/unreachable OIDC_PROVIDER_URL.
	badServer := httptest.NewServer(http.NotFoundHandler())
	defer badServer.Close()

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(badServer.URL, "test-client"))
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "failed to initialize OIDC provider")
}

func TestExchangeAndVerify_Success(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	defer idp.Close()

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)

	idToken, token, rawIDToken, err := provider.ExchangeAndVerify(context.Background(), "test-code", "test-pkce-verifier")
	require.NoError(t, err)
	require.NotNil(t, idToken)
	require.NotNil(t, token)

	assert.Equal(t, "test-subject", idToken.Subject)
	assert.Equal(t, "test-access-token", token.AccessToken)
	assert.NotEmpty(t, rawIDToken)

	// The raw ID token is retained for id_token_hint (RP-Initiated Logout),
	// so it must actually be the verified JWT, not some other string.
	reVerified, err := provider.verifier.Verify(context.Background(), rawIDToken)
	require.NoError(t, err)
	assert.Equal(t, idToken.Subject, reVerified.Subject)
}

func TestExchangeAndVerify_RejectsTokenSignedByUnknownKey(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	defer idp.Close()

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)

	// Simulate a forged ID token: signed by a key the IdP never published to
	// its own JWKS endpoint. If ExchangeAndVerify accepted this, signature
	// verification would be a no-op and any attacker who could intercept the
	// token response could forge a login.
	forgedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp.SigningKey = forgedKey

	idToken, token, rawIDToken, err := provider.ExchangeAndVerify(context.Background(), "test-code", "test-pkce-verifier")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify id_token")
	assert.Nil(t, idToken)
	assert.Nil(t, token)
	assert.Empty(t, rawIDToken)
}

func TestExchangeAndVerify_RejectsTamperedClaims(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	defer idp.Close()

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)

	// A token issued for a different audience must be rejected outright,
	// proving the verifier actually checks claims rather than just the
	// signature.
	idp.IDTokenClaims["aud"] = "someone-elses-client"

	idToken, token, rawIDToken, err := provider.ExchangeAndVerify(context.Background(), "test-code", "test-pkce-verifier")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify id_token")
	assert.Nil(t, idToken)
	assert.Nil(t, token)
	assert.Empty(t, rawIDToken)
}

func TestExchangeAndVerify_MissingIDToken(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	defer idp.Close()

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)

	// Override /token to respond without an id_token field at all, as a
	// pure-OAuth2 (non-OIDC) token endpoint would.
	idp.Server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			fmt.Fprint(w, `{"access_token":"test-access-token","token_type":"Bearer"}`)
		default:
			idp.handle(w, r)
		}
	})

	idToken, token, rawIDToken, err := provider.ExchangeAndVerify(context.Background(), "test-code", "test-pkce-verifier")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing id_token")
	assert.Nil(t, idToken)
	assert.Nil(t, token)
	assert.Empty(t, rawIDToken)
}

func TestExchangeAndVerify_TokenEndpointError(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	defer idp.Close()

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)

	idp.Server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		idp.handle(w, r)
	})

	idToken, token, rawIDToken, err := provider.ExchangeAndVerify(context.Background(), "bad-code", "test-pkce-verifier")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to exchange code")
	assert.Nil(t, idToken)
	assert.Nil(t, token)
	assert.Empty(t, rawIDToken)
}

func TestClaimsFor_UsesIDTokenClaimsWithoutUserInfoFallback(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	defer idp.Close()
	// No UserInfo configured: if ClaimsFor needed the fallback, this test
	// would fail with a UserInfo error rather than silently succeeding.

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)

	idToken, token, _, err := provider.ExchangeAndVerify(context.Background(), "test-code", "test-pkce-verifier")
	require.NoError(t, err)

	claims, err := provider.ClaimsFor(context.Background(), idToken, token, idp.Server.URL)
	require.NoError(t, err)
	assert.Equal(t, "test-subject", claims.Subject)
	assert.Equal(t, "test-user@example.com", claims.Email)
	assert.True(t, claims.EmailVerified)
	assert.Equal(t, "Test User", claims.Name)
	assert.Equal(t, idp.Server.URL, claims.Provider)
}

// The Authelia case (also covered directly against enrichFromUserInfo in
// oidc_userinfo_test.go), exercised here through the full ExchangeAndVerify +
// ClaimsFor path: an ID token carrying only `sub` must still yield a complete
// identity by falling back to UserInfo.
func TestClaimsFor_FallsBackToUserInfoThroughFullFlow(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	idp.UserInfo = map[string]any{
		"sub":            "test-subject",
		"email":          "ada@example.com",
		"email_verified": true,
		"name":           "Ada Lovelace",
	}
	defer idp.Close()

	idp.IDTokenClaims = jwt.MapClaims{"sub": "test-subject"}

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)

	idToken, token, _, err := provider.ExchangeAndVerify(context.Background(), "test-code", "test-pkce-verifier")
	require.NoError(t, err)

	claims, err := provider.ClaimsFor(context.Background(), idToken, token, idp.Server.URL)
	require.NoError(t, err)
	assert.Equal(t, "ada@example.com", claims.Email)
	assert.True(t, claims.EmailVerified)
	assert.Equal(t, "Ada Lovelace", claims.Name)
}

// OIDC Core 3.1.3.7: an ID token with multiple audiences and no azp claim
// must be rejected by ClaimsFor, not just by verifyAuthorizedParty in
// isolation (oidc_claims_test.go) -- this proves the check is actually wired
// into the real call path.
func TestClaimsFor_RejectsMissingAzpWithMultipleAudiences(t *testing.T) {
	idp := newFakeIDP(t, "test-client")
	defer idp.Close()

	idp.IDTokenClaims["aud"] = []string{"test-client", "some-other-client"}

	provider, err := InitOIDCProvider(context.Background(), testOIDCConfig(idp.Server.URL, "test-client"))
	require.NoError(t, err)

	idToken, token, _, err := provider.ExchangeAndVerify(context.Background(), "test-code", "test-pkce-verifier")
	require.NoError(t, err)

	claims, err := provider.ClaimsFor(context.Background(), idToken, token, idp.Server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple audiences")
	assert.Nil(t, claims)
}

func TestBuildAuthURL(t *testing.T) {
	provider := &OIDCProvider{}
	provider.oauth2Cfg = oauth2.Config{
		ClientID: "test-client",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://idp.example.com/auth",
			TokenURL: "https://idp.example.com/token",
		},
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid", "email"},
	}

	authURL := provider.BuildAuthURL("test-state", "test-nonce", "test-pkce-verifier")

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	assert.Equal(t, "idp.example.com", parsed.Host)
	assert.Equal(t, "/auth", parsed.Path)

	q := parsed.Query()
	assert.Equal(t, "test-client", q.Get("client_id"))
	assert.Equal(t, "test-state", q.Get("state"))
	assert.Equal(t, "test-nonce", q.Get("nonce"))
	assert.Equal(t, "https://app.example.com/callback", q.Get("redirect_uri"))
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "S256", q.Get("code_challenge_method"))
	assert.NotEmpty(t, q.Get("code_challenge"))
	// PKCE: the challenge is a derived hash of the verifier, not the verifier
	// itself -- if BuildAuthURL ever regressed to sending the verifier
	// plaintext, this would catch it.
	assert.NotEqual(t, "test-pkce-verifier", q.Get("code_challenge"))
}

func TestGenerateStateToken(t *testing.T) {
	a, err := GenerateStateToken()
	require.NoError(t, err)
	b, err := GenerateStateToken()
	require.NoError(t, err)

	// 32 random bytes, hex-encoded.
	assert.Len(t, a, 64)
	assert.Len(t, b, 64)
	assert.NotEqual(t, a, b, "state tokens must be unpredictable per login attempt")

	for _, r := range a {
		assert.True(t, (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f'), "expected lowercase hex, got %q", a)
	}
}

func TestGeneratePKCEVerifier(t *testing.T) {
	a := GeneratePKCEVerifier()
	b := GeneratePKCEVerifier()

	assert.NotEmpty(t, a)
	assert.NotEqual(t, a, b, "PKCE verifiers must be unique per login attempt")
	// RFC 7636 3.1: 43-128 characters from the unreserved character set.
	assert.GreaterOrEqual(t, len(a), 43)
	assert.LessOrEqual(t, len(a), 128)
}

func TestProviderName(t *testing.T) {
	tests := []struct {
		name        string
		providerURL string
		want        string
	}{
		{"standard issuer URL", "https://accounts.google.com", "accounts.google.com"},
		{"issuer with path", "https://idp.example.com/realms/main", "idp.example.com"},
		{"issuer with port", "https://idp.example.com:8443", "idp.example.com:8443"},
		{"empty URL falls back to SSO", "", "SSO"},
		{"unparseable URL falls back to SSO", "://not-a-url", "SSO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ProviderName(tt.providerURL))
		})
	}
}
