package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"mycorrhizal/config"
	"mycorrhizal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeOIDCProviderForLogout builds a real *services.OIDCProvider (via the
// exported InitOIDCProvider, pointed at a local fake discovery server) since
// OIDCProvider's internal field is unexported and unreachable from
// package controllers.
func newFakeOIDCProviderForLogout(t *testing.T, endSessionPath string) *services.OIDCProvider {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		endSessionJSON := ""
		if endSessionPath != "" {
			endSessionJSON = fmt.Sprintf(`,"end_session_endpoint":%q`, server.URL+endSessionPath)
		}
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"jwks_uri":%q%s}`,
			server.URL, server.URL+"/auth", server.URL+"/token", server.URL+"/keys", endSessionJSON)
	}))
	t.Cleanup(server.Close)

	provider, err := services.InitOIDCProvider(context.Background(), &config.Config{
		OIDC: config.OIDCConfig{
			ProviderURL:  server.URL,
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://mycorrhizal.example.com/api/v1/auth/oidc/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
	})
	require.NoError(t, err)
	return provider
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestLogoutUser_NoOIDC(t *testing.T) {
	cfg := &config.Config{}
	_, router := setupRouter()
	router.POST("/logout", func(c *gin.Context) {
		LogoutUser(c, cfg, nil)
	})

	req, _ := http.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Logged out successfully", body["message"])
	assert.NotContains(t, body, "redirect_url")

	authCookie := findCookie(w.Result().Cookies(), "auth_token")
	require.NotNil(t, authCookie)
	assert.Equal(t, -1, authCookie.MaxAge)
}

func TestLogoutUser_OIDCEnabledButLocalPasswordLogin(t *testing.T) {
	cfg := &config.Config{OIDC: config.OIDCConfig{PostLogoutRedirectURL: "https://mycorrhizal.example.com/login"}}
	provider := newFakeOIDCProviderForLogout(t, "/end_session")

	_, router := setupRouter()
	router.POST("/logout", func(c *gin.Context) {
		LogoutUser(c, cfg, provider)
	})

	// No id_token cookie attached -- this session was a local-password login,
	// even though OIDC is enabled globally.
	req, _ := http.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Logged out successfully", body["message"])
	assert.NotContains(t, body, "redirect_url", "no IdP session exists for a local-password login, so no redirect should be built")
}

func TestLogoutUser_OIDCSessionBuildsRedirect(t *testing.T) {
	cfg := &config.Config{OIDC: config.OIDCConfig{PostLogoutRedirectURL: "https://mycorrhizal.example.com/login"}}
	provider := newFakeOIDCProviderForLogout(t, "/end_session")

	_, router := setupRouter()
	router.POST("/logout", func(c *gin.Context) {
		LogoutUser(c, cfg, provider)
	})

	req, _ := http.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "id_token", Value: "test-raw-id-token"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Logged out successfully", body["message"])
	require.Contains(t, body, "redirect_url")

	redirectURL, err := url.Parse(body["redirect_url"])
	require.NoError(t, err)
	assert.Contains(t, redirectURL.String(), "/end_session")
	assert.Equal(t, "test-raw-id-token", redirectURL.Query().Get("id_token_hint"))
	assert.Equal(t, "https://mycorrhizal.example.com/login", redirectURL.Query().Get("post_logout_redirect_uri"))

	// Both cookies must be cleared regardless of the redirect_url outcome.
	authCookie := findCookie(w.Result().Cookies(), "auth_token")
	require.NotNil(t, authCookie)
	assert.Equal(t, -1, authCookie.MaxAge)
	idTokenCookie := findCookie(w.Result().Cookies(), "id_token")
	require.NotNil(t, idTokenCookie)
	assert.Equal(t, -1, idTokenCookie.MaxAge)
}

func TestLogoutUser_ProviderWithoutEndSessionEndpoint(t *testing.T) {
	cfg := &config.Config{OIDC: config.OIDCConfig{PostLogoutRedirectURL: "https://mycorrhizal.example.com/login"}}
	provider := newFakeOIDCProviderForLogout(t, "") // no end_session_endpoint advertised

	_, router := setupRouter()
	router.POST("/logout", func(c *gin.Context) {
		LogoutUser(c, cfg, provider)
	})

	req, _ := http.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "id_token", Value: "test-raw-id-token"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotContains(t, body, "redirect_url", "provider doesn't support RP-Initiated Logout, so no redirect should be built")
}
