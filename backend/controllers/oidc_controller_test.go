package controllers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"meerkat/config"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	oidcTestState = "state-value"
	oidcTestNonce = "nonce-value"
	oidcClientID  = "meerkat-client"
)

// fakeOIDCProvider is a real OIDC provider over httptest: discovery, token, JWKS and userinfo,
// signing RS256 ID tokens. Tests adjust what the token endpoint answers.
type fakeOIDCProvider struct {
	server *httptest.Server
	key    *rsa.PrivateKey

	mu          sync.Mutex
	claims      jwt.MapClaims
	signingKey  *rsa.PrivateKey
	tokenStatus int
	omitIDToken bool
	userInfo    map[string]any
}

func (p *fakeOIDCProvider) configure(change func(*fakeOIDCProvider)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	change(p)
}

func newFakeOIDCProvider(t *testing.T) *fakeOIDCProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	p := &fakeOIDCProvider{key: key, signingKey: key, tokenStatus: http.StatusOK}
	mux := http.NewServeMux()
	p.server = httptest.NewServer(mux)
	t.Cleanup(p.server.Close)

	writeJSON := func(w http.ResponseWriter, value any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(value)
	}
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                 p.server.URL,
			"authorization_endpoint": p.server.URL + "/authorize",
			"token_endpoint":         p.server.URL + "/token",
			"jwks_uri":               p.server.URL + "/jwks",
			"userinfo_endpoint":      p.server.URL + "/userinfo",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "test",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.tokenStatus != http.StatusOK {
			w.WriteHeader(p.tokenStatus)
			return
		}
		response := map[string]any{"access_token": "access", "token_type": "Bearer", "expires_in": 3600}
		if !p.omitIDToken {
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, p.claims)
			token.Header["kid"] = "test"
			signed, err := token.SignedString(p.signingKey)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			response["id_token"] = signed
		}
		writeJSON(w, response)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.userInfo == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSON(w, p.userInfo)
	})

	p.claims = p.idTokenClaims(jwt.MapClaims{"email": "jane@example.com", "email_verified": true, "name": "Jane"})
	return p
}

func (p *fakeOIDCProvider) idTokenClaims(extra jwt.MapClaims) jwt.MapClaims {
	claims := jwt.MapClaims{
		"iss": p.server.URL, "aud": oidcClientID, "sub": "subject-1", "nonce": oidcTestNonce,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}
	for key, value := range extra {
		claims[key] = value
	}
	return claims
}

type oidcFixture struct {
	db       *gorm.DB
	router   *gin.Engine
	fake     *fakeOIDCProvider
	cfg      *config.Config
	provider *services.OIDCProvider
}

func setupOIDC(t *testing.T, change func(*config.Config)) oidcFixture {
	t.Helper()
	fake := newFakeOIDCProvider(t)
	cfg := &config.Config{JWTSecretKey: testJWTSecret, JWTExpiryHours: 24, JWTMaxSessionHours: 720}
	cfg.OIDC = config.OIDCConfig{
		Enabled: true, ProviderURL: fake.server.URL, ClientID: oidcClientID, ClientSecret: "secret",
		RedirectURL: "https://app.example.com/api/v1/auth/oidc/callback",
	}
	if change != nil {
		change(cfg)
	}
	provider, err := services.InitOIDCProvider(context.Background(), cfg)
	require.NoError(t, err)

	db := newTestDB(t)
	router := routerAs(db, 0, *cfg)
	router.GET("/login", OIDCLoginHandler(provider, cfg))
	router.GET("/callback", OIDCCallbackHandler(provider, cfg))
	return oidcFixture{db: db, router: router, fake: fake, cfg: cfg, provider: provider}
}

func oidcCookies(state, nonce string) []*http.Cookie {
	var cookies []*http.Cookie
	if state != "" {
		cookies = append(cookies, &http.Cookie{Name: "oidc_state", Value: state})
	}
	if nonce != "" {
		cookies = append(cookies, &http.Cookie{Name: "oidc_nonce", Value: nonce})
	}
	return cookies
}

func (f oidcFixture) callback(query string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	return doRequest(f.router, http.MethodGet, "/callback?"+query, nil, "", cookies...)
}

func (f oidcFixture) validCallback() *httptest.ResponseRecorder {
	return f.callback("state="+oidcTestState+"&code=abc", oidcCookies(oidcTestState, oidcTestNonce)...)
}

func findCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func assertLoginRefused(t *testing.T, w *httptest.ResponseRecorder, wantError string) {
	t.Helper()
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login?error="+wantError, w.Header().Get("Location"))
	if cookie := findCookie(w, "auth_token"); cookie != nil {
		assert.Empty(t, cookie.Value, "a refused login must not set a session")
	}
}

func assertSessionFor(t *testing.T, f oidcFixture, w *httptest.ResponseRecorder, wantUserID uint) {
	t.Helper()
	require.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
	cookie := findCookie(w, "auth_token")
	require.NotNil(t, cookie, "a successful login must set the session cookie")
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, f.cfg.JWTExpiryHours*3600, cookie.MaxAge)

	token, err := jwt.Parse(cookie.Value, func(*jwt.Token) (any, error) { return []byte(testJWTSecret), nil })
	require.NoError(t, err)
	assert.EqualValues(t, wantUserID, token.Claims.(jwt.MapClaims)["user_id"])
}

func TestOIDCConfigHandler(t *testing.T) {
	router := gin.New()
	router.GET("/disabled", OIDCConfigHandler(&config.Config{}))
	enabled := &config.Config{}
	enabled.OIDC.Enabled = true
	enabled.OIDC.ProviderURL = "https://sso.example.com/realm"
	enabled.OIDC.ClientSecret = "must-stay-private"
	router.GET("/enabled", OIDCConfigHandler(enabled))

	w := doRequest(router, http.MethodGet, "/disabled", nil, "")
	assert.Equal(t, map[string]any{"enabled": false}, decodeJSON[map[string]any](t, w))

	w = doRequest(router, http.MethodGet, "/enabled", nil, "")
	assert.Equal(t, map[string]any{"enabled": true, "provider_name": "sso.example.com"}, decodeJSON[map[string]any](t, w))
}

func TestOIDCLogin_RedirectsToProviderWithFreshStateAndNonce(t *testing.T) {
	f := setupOIDC(t, func(cfg *config.Config) { cfg.CookieSecure = true })

	w := doRequest(f.router, http.MethodGet, "/login", nil, "")
	require.Equal(t, http.StatusFound, w.Code)

	target, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, f.fake.server.URL+"/authorize", target.Scheme+"://"+target.Host+target.Path)
	assert.Equal(t, oidcClientID, target.Query().Get("client_id"))
	assert.Equal(t, f.cfg.OIDC.RedirectURL, target.Query().Get("redirect_uri"))
	assert.Contains(t, target.Query().Get("scope"), "openid")

	state, nonce := findCookie(w, "oidc_state"), findCookie(w, "oidc_nonce")
	require.NotNil(t, state)
	require.NotNil(t, nonce)
	assert.Equal(t, state.Value, target.Query().Get("state"))
	assert.Equal(t, nonce.Value, target.Query().Get("nonce"))
	assert.NotEqual(t, state.Value, nonce.Value)
	for _, cookie := range []*http.Cookie{state, nonce} {
		assert.True(t, cookie.HttpOnly)
		assert.True(t, cookie.Secure)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.Equal(t, "/api/v1/auth/oidc/callback", cookie.Path)
	}

	second := doRequest(f.router, http.MethodGet, "/login", nil, "")
	assert.NotEqual(t, state.Value, findCookie(second, "oidc_state").Value, "state must be random per login")
}

func TestOIDCCallback_RefusedBeforeTheProviderIsAsked(t *testing.T) {
	for name, tc := range map[string]struct {
		query, state, nonce, wantError string
	}{
		"provider reported an error": {"error=access_denied&state=" + oidcTestState, oidcTestState, oidcTestNonce, "oidc_denied"},
		"missing state cookie":       {"state=" + oidcTestState + "&code=abc", "", oidcTestNonce, "oidc_error"},
		"missing nonce cookie":       {"state=" + oidcTestState + "&code=abc", oidcTestState, "", "oidc_error"},
		"state mismatch":             {"state=forged&code=abc", oidcTestState, oidcTestNonce, "oidc_error"},
		"empty state parameter":      {"code=abc", oidcTestState, oidcTestNonce, "oidc_error"},
		"missing code":               {"state=" + oidcTestState, oidcTestState, oidcTestNonce, "oidc_error"},
	} {
		t.Run(name, func(t *testing.T) {
			f := setupOIDC(t, func(cfg *config.Config) { cfg.OIDC.AllowAutoProvision = true })

			w := f.callback(tc.query, oidcCookies(tc.state, tc.nonce)...)

			assertLoginRefused(t, w, tc.wantError)
			var users int64
			require.NoError(t, f.db.Model(&models.User{}).Count(&users).Error)
			assert.Zero(t, users, "no account may be provisioned")
		})
	}
}

func TestOIDCCallback_ProviderOrTokenFailures(t *testing.T) {
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	for name, breakIt := range map[string]func(*fakeOIDCProvider){
		"code exchange fails": func(p *fakeOIDCProvider) { p.tokenStatus = http.StatusBadRequest },
		"no id_token":         func(p *fakeOIDCProvider) { p.omitIDToken = true },
		"forged signature":    func(p *fakeOIDCProvider) { p.signingKey = otherKey },
		"wrong audience":      func(p *fakeOIDCProvider) { p.claims = p.idTokenClaims(jwt.MapClaims{"aud": "another-app"}) },
		"wrong issuer": func(p *fakeOIDCProvider) {
			p.claims = p.idTokenClaims(jwt.MapClaims{"iss": "https://evil.example.com"})
		},
		"expired token": func(p *fakeOIDCProvider) {
			p.claims = p.idTokenClaims(jwt.MapClaims{"exp": time.Now().Add(-time.Hour).Unix()})
		},
		"nonce mismatch": func(p *fakeOIDCProvider) { p.claims = p.idTokenClaims(jwt.MapClaims{"nonce": "replayed"}) },
	} {
		t.Run(name, func(t *testing.T) {
			f := setupOIDC(t, func(cfg *config.Config) { cfg.OIDC.AllowAutoProvision = true })
			f.fake.configure(func(p *fakeOIDCProvider) {
				p.claims = p.idTokenClaims(jwt.MapClaims{"email": "jane@example.com", "email_verified": true, "name": "Jane"})
				breakIt(p)
			})

			w := f.validCallback()

			assertLoginRefused(t, w, "oidc_error")
			var users int64
			require.NoError(t, f.db.Model(&models.User{}).Count(&users).Error)
			assert.Zero(t, users, "an unverified token must not provision an account")
		})
	}
}

func TestOIDCCallback_ClearsStateCookies(t *testing.T) {
	f := setupOIDC(t, nil)

	w := f.callback("state=forged&code=abc", oidcCookies(oidcTestState, oidcTestNonce)...)

	for _, name := range []string{"oidc_state", "oidc_nonce"} {
		cookie := findCookie(w, name)
		require.NotNil(t, cookie, name)
		assert.Negative(t, cookie.MaxAge, "%s must not be reusable after a callback", name)
	}
}

func TestOIDCCallback_LinksVerifiedEmailToExistingAccount(t *testing.T) {
	f := setupOIDC(t, nil)
	existing := newTestUser(t, f.db, "jane", false)

	assertSessionFor(t, f, f.validCallback(), existing.ID)

	var reloaded models.User
	require.NoError(t, f.db.First(&reloaded, existing.ID).Error)
	require.NotNil(t, reloaded.OIDCSubject)
	assert.Equal(t, "subject-1", *reloaded.OIDCSubject)
}

func TestOIDCCallback_RedirectTargetCannotBeChosenByTheCaller(t *testing.T) {
	f := setupOIDC(t, nil)
	newTestUser(t, f.db, "jane", false)

	w := f.callback("state="+oidcTestState+"&code=abc&redirect=https://evil.example.com&next=//evil.example.com",
		oidcCookies(oidcTestState, oidcTestNonce)...)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
}

func TestOIDCCallback_UnverifiedEmailDoesNotTakeOverAnAccount(t *testing.T) {
	f := setupOIDC(t, nil)
	victim := newTestUser(t, f.db, "jane", false)
	f.fake.configure(func(p *fakeOIDCProvider) {
		p.claims = p.idTokenClaims(jwt.MapClaims{"email": "jane@example.com", "email_verified": false, "name": "Jane"})
	})

	assertLoginRefused(t, f.validCallback(), "oidc_no_account")

	var reloaded models.User
	require.NoError(t, f.db.First(&reloaded, victim.ID).Error)
	assert.Nil(t, reloaded.OIDCSubject)
}

func TestOIDCCallback_UnknownUserWithoutAutoProvision(t *testing.T) {
	f := setupOIDC(t, nil)

	assertLoginRefused(t, f.validCallback(), "oidc_no_account")

	var users int64
	require.NoError(t, f.db.Model(&models.User{}).Count(&users).Error)
	assert.Zero(t, users)
}

func TestOIDCCallback_AutoProvisionsFromUserInfoWhenTheIDTokenIsBare(t *testing.T) {
	f := setupOIDC(t, func(cfg *config.Config) { cfg.OIDC.AllowAutoProvision = true })
	f.fake.configure(func(p *fakeOIDCProvider) {
		p.claims = p.idTokenClaims(nil)
		p.userInfo = map[string]any{"sub": "subject-1", "email": "New.Person@Example.com", "preferred_username": "new.person"}
	})

	w := f.validCallback()

	var user models.User
	require.NoError(t, f.db.Where("oidc_subject = ?", "subject-1").First(&user).Error)
	assert.Equal(t, "new_person", user.Username)
	assert.Equal(t, "new.person@example.com", user.Email)
	assert.False(t, user.IsAdmin, "provisioned accounts must not be admins")
	assertSessionFor(t, f, w, user.ID)
}

func TestOIDCCallback_FailingUserInfoFallsBackToIDTokenClaims(t *testing.T) {
	f := setupOIDC(t, func(cfg *config.Config) { cfg.OIDC.AllowAutoProvision = true })
	f.fake.configure(func(p *fakeOIDCProvider) {
		p.claims = p.idTokenClaims(jwt.MapClaims{"name": "Only Name"})
		p.userInfo = nil
	})

	w := f.validCallback()

	var user models.User
	require.NoError(t, f.db.Where("oidc_subject = ?", "subject-1").First(&user).Error)
	assert.Equal(t, "only_name", user.Username)
	assertSessionFor(t, f, w, user.ID)
}

func TestOIDCCallback_DatabaseFailureIsAGenericError(t *testing.T) {
	f := setupOIDC(t, nil)
	sqlDB, err := f.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := f.validCallback()

	assertLoginRefused(t, w, "oidc_error")
	assert.False(t, strings.Contains(w.Header().Get("Location"), "database"))
}
