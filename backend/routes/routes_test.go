package routes

import (
	"fmt"
	"meerkat/config"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// publicRoutes is every route reachable without logging in. Adding to this list is a security
// decision: anything not listed here must reject anonymous requests.
var publicRoutes = map[string]bool{
	"GET /health":                          true,
	"GET /api/v1/auth/oidc/config":         true,
	"GET /api/v1/auth/oidc/login":          true,
	"GET /api/v1/auth/oidc/callback":       true,
	"POST /api/v1/register":                true,
	"POST /api/v1/login":                   true,
	"POST /api/v1/logout":                  true,
	"POST /api/v1/check-password-strength": true,
	"POST /api/v1/password-reset/request":  true,
	"POST /api/v1/password-reset/confirm":  true,
	"GET /.well-known/carddav":             true,
}

var routeParam = regexp.MustCompile(`:[^/]+`)

const testSecret = "test-jwt-secret-key-with-32-chars!!"

func testConfig() *config.Config {
	cfg := &config.Config{
		JWTSecretKey: testSecret, JWTExpiryHours: 24, JWTMaxSessionHours: 720,
		CardDAVEnabled: true, ProfilePhotoDir: "unused",
	}
	cfg.OIDC.Enabled = true
	cfg.OIDC.ProviderURL = "https://sso.example.com"
	return cfg
}

func newRouter(t *testing.T, cfg *config.Config, oidc *services.OIDCProvider) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.ApiToken{}, &models.Contact{}))

	router := gin.New()
	// The same injection main.go does ahead of the routes.
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("cfg", *cfg)
		c.Next()
	})
	RegisterRoutes(router, cfg, db, oidc)
	return router, db
}

// concretePath turns a route pattern into a path the router will match.
func concretePath(pattern string) string {
	path := routeParam.ReplaceAllString(pattern, "1")
	if i := strings.Index(path, "*"); i >= 0 {
		path = path[:i] + "x"
	}
	return path
}

func routeKey(route gin.RouteInfo) string {
	return route.Method + " " + route.Path
}

// requestCounter gives every request its own client IP, so the walks cannot trip the rate limiters
// and report a 429 where a 401 was expected, however many routes get added.
var requestCounter atomic.Uint32

func request(router http.Handler, method, path string, prepare func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	n := requestCounter.Add(1)
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", byte(n>>16), byte(n>>8), byte(n))
	if prepare != nil {
		prepare(req)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// routesReachableAnonymously walks the router, so a new route is covered without touching this file.
func routesReachableAnonymously(router *gin.Engine) []string {
	var exposed []string
	for _, route := range router.Routes() {
		if publicRoutes[routeKey(route)] {
			continue
		}
		w := request(router, route.Method, concretePath(route.Path), nil)
		if w.Code != http.StatusUnauthorized {
			exposed = append(exposed, routeKey(route))
		}
	}
	return exposed
}

func TestRoutes_EverythingButThePublicListRejectsAnonymousRequests(t *testing.T) {
	router, _ := newRouter(t, testConfig(), &services.OIDCProvider{})

	require.Greater(t, len(router.Routes()), 100, "the walk must actually see the real route table")
	assert.Empty(t, routesReachableAnonymously(router))
}

func TestRoutes_WalkCatchesARouteOutsideTheAuthGroup(t *testing.T) {
	router, _ := newRouter(t, testConfig(), &services.OIDCProvider{})
	router.GET("/api/v1/forgot-the-auth-group/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	assert.Equal(t, []string{"GET /api/v1/forgot-the-auth-group/:id"}, routesReachableAnonymously(router))
}

func TestRoutes_PublicListHasNoStaleEntries(t *testing.T) {
	router, _ := newRouter(t, testConfig(), &services.OIDCProvider{})

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[routeKey(route)] = true
	}
	for route := range publicRoutes {
		assert.True(t, registered[route], "%s no longer exists; remove it from publicRoutes", route)
	}
}

func TestRoutes_AnonymousRequestsDoNotGetPastAuthWithBadCredentials(t *testing.T) {
	router, _ := newRouter(t, testConfig(), nil)

	for name, prepare := range map[string]func(*http.Request){
		"garbage bearer":    func(r *http.Request) { r.Header.Set("Authorization", "Bearer not-a-token") },
		"non-bearer scheme": func(r *http.Request) { r.Header.Set("Authorization", "Basic abc") },
		"unknown api token": func(r *http.Request) { r.Header.Set("Authorization", "Bearer meerkat_unknown") },
		"garbage cookie":    func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "auth_token", Value: "garbage"}) },
	} {
		assert.Equal(t, http.StatusUnauthorized, request(router, http.MethodGet, "/api/v1/contacts", prepare).Code, name)
	}
}

func TestRoutes_OptionalRoutesAreOnlyRegisteredWhenEnabled(t *testing.T) {
	cfg := testConfig()
	cfg.CardDAVEnabled = false
	cfg.OIDC.Enabled = false
	router, _ := newRouter(t, cfg, nil)

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[routeKey(route)] = true
		assert.False(t, strings.HasPrefix(route.Path, "/carddav"), route.Path)
	}
	assert.True(t, registered["GET /api/v1/auth/oidc/config"], "the config endpoint tells the frontend OIDC is off")
	assert.False(t, registered["GET /api/v1/auth/oidc/login"])
	assert.False(t, registered["GET /api/v1/auth/oidc/callback"])
	assert.False(t, registered["GET /.well-known/carddav"])

	// Enabled without a provider (failed initialization) must not register handlers that would panic.
	cfg = testConfig()
	router, _ = newRouter(t, cfg, nil)
	assert.Equal(t, http.StatusNotFound, request(router, http.MethodGet, "/api/v1/auth/oidc/login", nil).Code)
}

func TestRoutes_AdminGroupIsClosedToOtherAuthenticatedUsers(t *testing.T) {
	cfg := testConfig()
	router, db := newRouter(t, cfg, nil)
	user := models.User{Username: "regular", Email: "regular@example.com", Password: "x"}
	require.NoError(t, db.Create(&user).Error)
	token, err := middleware.GenerateToken(user, cfg)
	require.NoError(t, err)
	asUser := func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }

	checked := 0
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, "/api/v1/admin") {
			checked++
			assert.Equal(t, http.StatusForbidden, request(router, route.Method, concretePath(route.Path), asUser).Code, routeKey(route))
		}
	}
	assert.GreaterOrEqual(t, checked, 5)

	// The same token does open the regular protected routes, so the 403 above is about admin rights.
	assert.Equal(t, http.StatusOK, request(router, http.MethodGet, "/api/v1/users/me", asUser).Code)
}

func TestRoutes_PublicEndpointsAnswerWithoutCredentials(t *testing.T) {
	router, _ := newRouter(t, testConfig(), &services.OIDCProvider{})

	config := request(router, http.MethodGet, "/api/v1/auth/oidc/config", nil)
	assert.Equal(t, http.StatusOK, config.Code)
	assert.Contains(t, config.Body.String(), `"enabled":true`)

	assert.Equal(t, http.StatusOK, request(router, http.MethodPost, "/api/v1/logout", nil).Code)

	wellKnown := request(router, http.MethodGet, "/.well-known/carddav", nil)
	assert.Less(t, wellKnown.Code, 400)
	assert.NotEqual(t, http.StatusUnauthorized, wellKnown.Code)
}
