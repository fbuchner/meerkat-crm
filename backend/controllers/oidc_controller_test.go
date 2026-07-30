package controllers

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
	"mycorrhizal/models"
	"mycorrhizal/services"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- OIDCConfigHandler ---

func TestOIDCConfigHandler_Disabled(t *testing.T) {
	cfg := &config.Config{OIDC: config.OIDCConfig{Enabled: false}}
	_, router := setupRouter()
	router.GET("/config", OIDCConfigHandler(cfg))

	req, _ := http.NewRequest("GET", "/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["enabled"])
	assert.NotContains(t, body, "provider_name")
}

func TestOIDCConfigHandler_Enabled(t *testing.T) {
	cfg := &config.Config{OIDC: config.OIDCConfig{Enabled: true, ProviderURL: "https://accounts.google.com"}}
	_, router := setupRouter()
	router.GET("/config", OIDCConfigHandler(cfg))

	req, _ := http.NewRequest("GET", "/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["enabled"])
	assert.Equal(t, "accounts.google.com", body["provider_name"])
}

// --- OIDCLoginHandler ---

func TestOIDCLoginHandler(t *testing.T) {
	// Reuses the fake discovery-only IdP already established for logout
	// tests (newFakeOIDCProviderForLogout, logout_controller_test.go) since
	// BuildAuthURL only needs a real, discovery-initialized OIDCProvider --
	// it never hits /token.
	provider := newFakeOIDCProviderForLogout(t, "")
	cfg := &config.Config{CookieDomain: "", CookieSecure: false}

	_, router := setupRouter()
	router.GET("/login", OIDCLoginHandler(provider, cfg))

	req, _ := http.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)

	stateCookie := findCookie(w.Result().Cookies(), "oidc_state")
	require.NotNil(t, stateCookie)
	assert.NotEmpty(t, stateCookie.Value)
	nonceCookie := findCookie(w.Result().Cookies(), "oidc_nonce")
	require.NotNil(t, nonceCookie)
	assert.NotEmpty(t, nonceCookie.Value)
	pkceCookie := findCookie(w.Result().Cookies(), "oidc_pkce")
	require.NotNil(t, pkceCookie)
	assert.NotEmpty(t, pkceCookie.Value)
	assert.NotEqual(t, stateCookie.Value, nonceCookie.Value)

	loc := w.Header().Get("Location")
	parsedLoc, err := url.Parse(loc)
	require.NoError(t, err)
	q := parsedLoc.Query()
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "test-client", q.Get("client_id"))
	assert.Equal(t, stateCookie.Value, q.Get("state"))
	assert.Equal(t, nonceCookie.Value, q.Get("nonce"))
	assert.Equal(t, "S256", q.Get("code_challenge_method"))
	assert.NotEmpty(t, q.Get("code_challenge"))
}

// --- fakeCallbackIDP: a full OpenID Provider (discovery + JWKS + /token)
// backed by httptest.Server, issuing real RS256-signed ID tokens verified by
// go-oidc over the wire. Replicated here (package controllers) from
// services/oidc_service_test.go's fakeIDP, which cannot be imported
// cross-package -- see that file for the canonical version and rationale.

type fakeCallbackIDP struct {
	Server   *httptest.Server
	key      *rsa.PrivateKey
	kid      string
	clientID string

	// IDTokenClaims seeds the ID token issued by /token. Tests mutate this
	// directly to control what the callback handler receives.
	IDTokenClaims jwt.MapClaims

	// SigningKey, when set, signs the ID token instead of the IdP's own key --
	// used to simulate a token an attacker forged / that got corrupted in
	// transit, so ExchangeAndVerify's signature check can be exercised for
	// real rather than assumed.
	SigningKey *rsa.PrivateKey
}

func newFakeCallbackIDP(t *testing.T, clientID string) *fakeCallbackIDP {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	idp := &fakeCallbackIDP{
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
	t.Cleanup(idp.Server.Close)
	return idp
}

func (idp *fakeCallbackIDP) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"jwks_uri":%q}`,
			idp.Server.URL, idp.Server.URL+"/auth", idp.Server.URL+"/token", idp.Server.URL+"/keys")
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
	default:
		http.NotFound(w, r)
	}
}

func (idp *fakeCallbackIDP) signIDToken() (string, error) {
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

// newCallbackTestSetup builds a real *services.OIDCProvider (via the
// exported InitOIDCProvider, exactly like newFakeOIDCProviderForLogout) plus
// a matching *config.Config, pointed at idp.
func newCallbackTestSetup(t *testing.T, idp *fakeCallbackIDP) (*services.OIDCProvider, *config.Config) {
	t.Helper()

	cfg := &config.Config{
		OIDC: config.OIDCConfig{
			Enabled:      true,
			ProviderURL:  idp.Server.URL,
			ClientID:     idp.clientID,
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://mycorrhizal.example.com/api/v1/auth/oidc/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
		JWTSecretKey:   "test-jwt-secret-key-for-oidc-callback-tests",
		JWTExpiryHours: 24,
		CookieDomain:   "",
		CookieSecure:   false,
	}

	provider, err := services.InitOIDCProvider(context.Background(), cfg)
	require.NoError(t, err)
	return provider, cfg
}

// callbackRequest builds a GET /callback request carrying the given
// state/nonce/pkce cookies (only those with non-empty name are attached, so
// tests can omit one to exercise the "missing cookie" branches) and query
// parameters.
func callbackRequest(cookies map[string]string, query url.Values) *http.Request {
	req, _ := http.NewRequest("GET", "/callback?"+query.Encode(), nil)
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	return req
}

func fullCookieSet(state, nonce, pkce string) map[string]string {
	return map[string]string{"oidc_state": state, "oidc_nonce": nonce, "oidc_pkce": pkce}
}

func assertRedirectsTo(t *testing.T, w *httptest.ResponseRecorder, path string) {
	t.Helper()
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, path, w.Header().Get("Location"))
}

// --- OIDCCallbackHandler: pre-exchange validation (no real IdP needed) ---

func TestOIDCCallbackHandler_ProviderDenied(t *testing.T) {
	_, router := setupRouter()
	cfg := &config.Config{}
	router.GET("/callback", OIDCCallbackHandler(nil, cfg))

	req := callbackRequest(nil, url.Values{"error": {"access_denied"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_denied")
}

func TestOIDCCallbackHandler_MissingStateCookie(t *testing.T) {
	_, router := setupRouter()
	cfg := &config.Config{}
	router.GET("/callback", OIDCCallbackHandler(nil, cfg))

	req := callbackRequest(map[string]string{"oidc_nonce": "n", "oidc_pkce": "p"}, url.Values{"state": {"s"}, "code": {"c"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_error")
}

func TestOIDCCallbackHandler_MissingNonceCookie(t *testing.T) {
	_, router := setupRouter()
	cfg := &config.Config{}
	router.GET("/callback", OIDCCallbackHandler(nil, cfg))

	req := callbackRequest(map[string]string{"oidc_state": "s", "oidc_pkce": "p"}, url.Values{"state": {"s"}, "code": {"c"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_error")
}

func TestOIDCCallbackHandler_MissingPKCECookie(t *testing.T) {
	_, router := setupRouter()
	cfg := &config.Config{}
	router.GET("/callback", OIDCCallbackHandler(nil, cfg))

	req := callbackRequest(map[string]string{"oidc_state": "s", "oidc_nonce": "n"}, url.Values{"state": {"s"}, "code": {"c"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_error")
}

func TestOIDCCallbackHandler_StateMismatch(t *testing.T) {
	_, router := setupRouter()
	cfg := &config.Config{}
	router.GET("/callback", OIDCCallbackHandler(nil, cfg))

	req := callbackRequest(fullCookieSet("cookie-state", "n", "p"), url.Values{"state": {"different-state"}, "code": {"c"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_error")
}

func TestOIDCCallbackHandler_MissingCode(t *testing.T) {
	_, router := setupRouter()
	cfg := &config.Config{}
	router.GET("/callback", OIDCCallbackHandler(nil, cfg))

	req := callbackRequest(fullCookieSet("s", "n", "p"), url.Values{"state": {"s"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_error")
}

// State/nonce/PKCE cookies are cleared on every callback regardless of
// outcome -- verified once here since every other test only checks the
// redirect.
func TestOIDCCallbackHandler_ClearsCookiesEvenOnFailure(t *testing.T) {
	_, router := setupRouter()
	cfg := &config.Config{}
	router.GET("/callback", OIDCCallbackHandler(nil, cfg))

	req := callbackRequest(fullCookieSet("s", "n", "p"), url.Values{"state": {"wrong"}, "code": {"c"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	for _, name := range []string{"oidc_state", "oidc_nonce", "oidc_pkce"} {
		c := findCookie(w.Result().Cookies(), name)
		require.NotNil(t, c, "expected %s cookie to be cleared", name)
		assert.Equal(t, -1, c.MaxAge)
	}
}

// --- OIDCCallbackHandler: real exchange against the fake IdP ---

func TestOIDCCallbackHandler_ExchangeFailsOnForgedSignature(t *testing.T) {
	idp := newFakeCallbackIDP(t, "test-client")
	forgedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp.SigningKey = forgedKey

	provider, cfg := newCallbackTestSetup(t, idp)

	_, router := setupRouter()
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))

	req := callbackRequest(fullCookieSet("good-state", "good-nonce", "good-pkce"),
		url.Values{"state": {"good-state"}, "code": {"auth-code"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// This proves the callback handler genuinely calls ExchangeAndVerify's
	// real signature check (not a stub): a token signed by an unpublished
	// key must be rejected the same way TestExchangeAndVerify_RejectsTokenSignedByUnknownKey
	// proves it at the services layer.
	assertRedirectsTo(t, w, "/login?error=oidc_error")
}

func TestOIDCCallbackHandler_NonceMismatch(t *testing.T) {
	idp := newFakeCallbackIDP(t, "test-client")
	idp.IDTokenClaims["nonce"] = "nonce-from-idp"

	provider, cfg := newCallbackTestSetup(t, idp)

	_, router := setupRouter()
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))

	req := callbackRequest(fullCookieSet("good-state", "cookie-nonce-does-not-match", "good-pkce"),
		url.Values{"state": {"good-state"}, "code": {"auth-code"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_error")
}

func TestOIDCCallbackHandler_ClaimsRejectsMultipleAudienceWithoutAzp(t *testing.T) {
	idp := newFakeCallbackIDP(t, "test-client")
	idp.IDTokenClaims["nonce"] = "matching-nonce"
	idp.IDTokenClaims["aud"] = []string{"test-client", "some-other-client"}

	provider, cfg := newCallbackTestSetup(t, idp)

	_, router := setupRouter()
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))

	req := callbackRequest(fullCookieSet("good-state", "matching-nonce", "good-pkce"),
		url.Values{"state": {"good-state"}, "code": {"auth-code"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// ExchangeAndVerify's signature/aud-contains-client checks pass here (the
	// token really is validly signed and test-client is in aud); it's
	// ClaimsFor's OIDC Core 3.1.3.7 azp check that must reject this, proving
	// that path is wired into the callback handler for real.
	assertRedirectsTo(t, w, "/login?error=oidc_error")
}

func TestOIDCCallbackHandler_NoAccountAndAutoProvisionDisabled(t *testing.T) {
	idp := newFakeCallbackIDP(t, "test-client")
	idp.IDTokenClaims["nonce"] = "matching-nonce"
	idp.IDTokenClaims["email"] = "unknown-oidc-user@example.com"

	provider, cfg := newCallbackTestSetup(t, idp)
	cfg.OIDC.AllowAutoProvision = false

	_, router := setupRouter()
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))

	req := callbackRequest(fullCookieSet("good-state", "matching-nonce", "good-pkce"),
		url.Values{"state": {"good-state"}, "code": {"auth-code"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_no_account")
}

func TestOIDCCallbackHandler_NoEmailWithAutoProvisionEnabled(t *testing.T) {
	idp := newFakeCallbackIDP(t, "test-client")
	idp.IDTokenClaims["nonce"] = "matching-nonce"
	delete(idp.IDTokenClaims, "email")
	delete(idp.IDTokenClaims, "email_verified")
	// No UserInfo endpoint is advertised in discovery, so ClaimsFor's
	// fallback cannot recover an email either.

	provider, cfg := newCallbackTestSetup(t, idp)
	cfg.OIDC.AllowAutoProvision = true

	_, router := setupRouter()
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))

	req := callbackRequest(fullCookieSet("good-state", "matching-nonce", "good-pkce"),
		url.Values{"state": {"good-state"}, "code": {"auth-code"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/login?error=oidc_no_email")
}

func TestOIDCCallbackHandler_SuccessLinksExistingUserByOIDCSubject(t *testing.T) {
	idp := newFakeCallbackIDP(t, "test-client")
	idp.IDTokenClaims["nonce"] = "matching-nonce"
	idp.IDTokenClaims["sub"] = "existing-subject"

	provider, cfg := newCallbackTestSetup(t, idp)

	db, router := setupRouter()
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))

	subject := "existing-subject"
	providerURL := idp.Server.URL
	linkedUser := models.User{
		Username:     "already-linked",
		Password:     "",
		Email:        "already-linked@example.com",
		OIDCSubject:  &subject,
		OIDCProvider: &providerURL,
	}
	require.NoError(t, db.Create(&linkedUser).Error)

	req := callbackRequest(fullCookieSet("good-state", "matching-nonce", "good-pkce"),
		url.Values{"state": {"good-state"}, "code": {"auth-code"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/")

	authCookie := findCookie(w.Result().Cookies(), "auth_token")
	require.NotNil(t, authCookie)
	assert.NotEmpty(t, authCookie.Value)
	assert.Equal(t, 24*3600, authCookie.MaxAge)

	idTokenCookie := findCookie(w.Result().Cookies(), "id_token")
	require.NotNil(t, idTokenCookie)
	assert.NotEmpty(t, idTokenCookie.Value)

	// The retained id_token cookie must be the actual verified JWT from the
	// IdP, not a placeholder -- it's later used as id_token_hint for RP-
	// Initiated Logout.
	assert.Equal(t, 3, len(splitJWT(idTokenCookie.Value)), "expected a well-formed JWT (header.payload.signature)")
}

func TestOIDCCallbackHandler_SuccessLinksExistingUserByVerifiedEmail(t *testing.T) {
	idp := newFakeCallbackIDP(t, "test-client")
	idp.IDTokenClaims["nonce"] = "matching-nonce"
	idp.IDTokenClaims["sub"] = "brand-new-subject-not-yet-linked"
	idp.IDTokenClaims["email"] = "tester@example.com" // matches setupRouter's seeded user
	idp.IDTokenClaims["email_verified"] = true

	provider, cfg := newCallbackTestSetup(t, idp)

	db, router := setupRouter()
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))

	req := callbackRequest(fullCookieSet("good-state", "matching-nonce", "good-pkce"),
		url.Values{"state": {"good-state"}, "code": {"auth-code"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/")

	var user models.User
	require.NoError(t, db.Where("email = ?", "tester@example.com").First(&user).Error)
	require.NotNil(t, user.OIDCSubject)
	assert.Equal(t, "brand-new-subject-not-yet-linked", *user.OIDCSubject)
}

func TestOIDCCallbackHandler_SuccessAutoProvisionsNewUser(t *testing.T) {
	idp := newFakeCallbackIDP(t, "test-client")
	idp.IDTokenClaims["nonce"] = "matching-nonce"
	idp.IDTokenClaims["sub"] = "auto-provisioned-subject"
	idp.IDTokenClaims["email"] = "newperson@example.com"
	idp.IDTokenClaims["email_verified"] = true
	idp.IDTokenClaims["name"] = "New Person"

	provider, cfg := newCallbackTestSetup(t, idp)
	cfg.OIDC.AllowAutoProvision = true

	db, router := setupRouter()
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))

	req := callbackRequest(fullCookieSet("good-state", "matching-nonce", "good-pkce"),
		url.Values{"state": {"good-state"}, "code": {"auth-code"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertRedirectsTo(t, w, "/")

	var user models.User
	require.NoError(t, db.Where("email = ?", "newperson@example.com").First(&user).Error)
	require.NotNil(t, user.OIDCSubject)
	assert.Equal(t, "auto-provisioned-subject", *user.OIDCSubject)

	authCookie := findCookie(w.Result().Cookies(), "auth_token")
	require.NotNil(t, authCookie)
	assert.NotEmpty(t, authCookie.Value)
}

func splitJWT(s string) []string {
	var parts []string
	start := 0
	for i, r := range s {
		if r == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
