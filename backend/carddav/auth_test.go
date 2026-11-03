package carddav

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"meerkat/middleware"
	"meerkat/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const authTestPassword = "correct horse battery staple"

// authFixture is a router with BasicAuthMiddleware in front of a handler that records whether
// it ran and which identity it saw. The account rate limiter is a process-wide singleton, so
// every test must use its own username and email.
type authFixture struct {
	db      *gorm.DB
	router  *gin.Engine
	reached bool
	userID  uint
	name    string
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.ApiToken{}))

	f := &authFixture{db: db, router: gin.New()}
	f.router.Use(func(c *gin.Context) { c.Set("db", db) })
	f.router.Use(BasicAuthMiddleware())
	f.router.Any("/carddav", func(c *gin.Context) {
		f.reached = true
		f.userID = c.GetUint("userID")
		f.name = c.GetString("username")
		c.Status(http.StatusNoContent)
	})
	return f
}

// seedUser creates a user with a known password and clears the limiter state for its identifiers.
func (f *authFixture) seedUser(t *testing.T, username, email string) models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(authTestPassword), bcrypt.MinCost)
	require.NoError(t, err)
	user := models.User{Username: username, Email: email, Password: string(hash)}
	require.NoError(t, f.db.Create(&user).Error)
	forgetLimiterState(t, username, email)
	return user
}

func forgetLimiterState(t *testing.T, identifiers ...string) {
	t.Helper()
	limiter := middleware.GetAccountRateLimiter()
	forget := func() {
		for _, id := range identifiers {
			limiter.RecordSuccessfulLogin(id)
		}
	}
	forget()
	t.Cleanup(forget)
}

func (f *authFixture) request(username, password string) *httptest.ResponseRecorder {
	f.reached = false
	req := httptest.NewRequest(http.MethodGet, "/carddav", nil)
	req.SetBasicAuth(username, password)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func assertChallenged(t *testing.T, f *authFixture, rec *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `Basic realm="CardDAV"`, rec.Header().Get("WWW-Authenticate"))
	assert.False(t, f.reached, "the handler must not run without valid credentials")
}

func TestBasicAuth_NoCredentialsAreChallenged(t *testing.T) {
	f := newAuthFixture(t)

	req := httptest.NewRequest(http.MethodGet, "/carddav", nil)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assertChallenged(t, f, rec)
}

func TestBasicAuth_ValidCredentials(t *testing.T) {
	f := newAuthFixture(t)
	user := f.seedUser(t, "valid_login", "valid_login@example.com")

	tests := []struct {
		name       string
		identifier string
	}{
		{"username", "valid_login"},
		{"email", "valid_login@example.com"},
		{"username in other case", "Valid_Login"},
		{"email in other case", "VALID_LOGIN@Example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forgetLimiterState(t, tt.identifier)

			rec := f.request(tt.identifier, authTestPassword)

			assert.Equal(t, http.StatusNoContent, rec.Code)
			assert.True(t, f.reached)
			assert.Equal(t, user.ID, f.userID)
			assert.Equal(t, "valid_login", f.name)
		})
	}
}

func TestBasicAuth_WrongPasswordIsRefusedAndCounted(t *testing.T) {
	f := newAuthFixture(t)
	f.seedUser(t, "wrong_pw", "wrong_pw@example.com")

	rec := f.request("wrong_pw", "not the password")

	assertChallenged(t, f, rec)
	assert.Equal(t, 1, middleware.GetAccountRateLimiter().GetFailedAttempts("wrong_pw"))
}

func TestBasicAuth_EmptyPasswordIsRefused(t *testing.T) {
	f := newAuthFixture(t)
	f.seedUser(t, "empty_pw", "empty_pw@example.com")

	assertChallenged(t, f, f.request("empty_pw", ""))
}

func TestBasicAuth_SuccessClearsFailedAttempts(t *testing.T) {
	f := newAuthFixture(t)
	f.seedUser(t, "recovers", "recovers@example.com")
	limiter := middleware.GetAccountRateLimiter()

	f.request("recovers", "typo")
	f.request("recovers", "typo")
	require.Equal(t, 2, limiter.GetFailedAttempts("recovers"))

	rec := f.request("recovers", authTestPassword)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Zero(t, limiter.GetFailedAttempts("recovers"))
}

func TestBasicAuth_UnknownUserIsRefusedAndCounted(t *testing.T) {
	f := newAuthFixture(t)
	forgetLimiterState(t, "nobody_here")

	rec := f.request("Nobody_Here", authTestPassword)

	assertChallenged(t, f, rec)
	// Counted like a real account, so lockout behaviour does not reveal which accounts exist.
	assert.Equal(t, 1, middleware.GetAccountRateLimiter().GetFailedAttempts("nobody_here"))
}

func TestBasicAuth_DeletedUserIsRefused(t *testing.T) {
	f := newAuthFixture(t)
	user := f.seedUser(t, "deleted_user", "deleted_user@example.com")

	require.Equal(t, http.StatusNoContent, f.request("deleted_user", authTestPassword).Code)

	require.NoError(t, f.db.Delete(&user).Error)

	assertChallenged(t, f, f.request("deleted_user", authTestPassword))
	forgetLimiterState(t, "deleted_user@example.com")
	assertChallenged(t, f, f.request("deleted_user@example.com", authTestPassword))
}

func TestBasicAuth_AccountWithoutPasswordCannotLogIn(t *testing.T) {
	f := newAuthFixture(t)
	subject, provider := "sub-1", "https://idp.example.com"
	user := models.User{Username: "sso_only", Email: "sso_only@example.com", OIDCSubject: &subject, OIDCProvider: &provider}
	require.NoError(t, f.db.Create(&user).Error)
	forgetLimiterState(t, "sso_only")

	assertChallenged(t, f, f.request("sso_only", ""))
	assertChallenged(t, f, f.request("sso_only", "anything"))
}

func TestBasicAuth_EmptyIdentifierDoesNotMatchAccountWithoutEmail(t *testing.T) {
	f := newAuthFixture(t)
	// OIDC accounts can be provisioned before the provider supplies an email.
	user := models.User{Username: "no_email_user", Email: ""}
	require.NoError(t, f.db.Create(&user).Error)
	forgetLimiterState(t, "")

	assertChallenged(t, f, f.request("", ""))
}

// CardDAV authenticates with the account password only. API tokens are for the REST API, so
// one must not open the contact store, either as the Basic password or as a bearer token.
func TestBasicAuth_APITokenIsNotACredential(t *testing.T) {
	f := newAuthFixture(t)
	user := f.seedUser(t, "token_owner", "token_owner@example.com")
	const plaintext = "mcrm_plaintext-token"
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))
	require.NoError(t, f.db.Create(&models.ApiToken{UserID: user.ID, Name: "phone", TokenHash: hash}).Error)

	t.Run("as Basic password", func(t *testing.T) {
		assertChallenged(t, f, f.request("token_owner", plaintext))
	})

	t.Run("as bearer token", func(t *testing.T) {
		f.reached = false
		req := httptest.NewRequest(http.MethodGet, "/carddav", nil)
		req.Header.Set("Authorization", "Bearer "+plaintext)
		rec := httptest.NewRecorder()
		f.router.ServeHTTP(rec, req)

		assertChallenged(t, f, rec)
	})
}

func TestBasicAuth_LocksAccountAfterRepeatedFailures(t *testing.T) {
	f := newAuthFixture(t)
	f.seedUser(t, "locked_out", "locked_out@example.com")

	for i := 1; i < middleware.MaxLoginAttempts; i++ {
		assertChallenged(t, f, f.request("locked_out", "wrong"))
	}

	// The attempt that trips the lock already reports it.
	rec := f.request("locked_out", "wrong")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.False(t, f.reached)

	t.Run("correct password is refused while locked", func(t *testing.T) {
		rec := f.request("locked_out", authTestPassword)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.False(t, f.reached)
		retryAfter, err := strconv.Atoi(rec.Header().Get("Retry-After"))
		require.NoError(t, err)
		assert.Positive(t, retryAfter)
	})

	t.Run("lock applies regardless of identifier case", func(t *testing.T) {
		rec := f.request("LOCKED_OUT", authTestPassword)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.False(t, f.reached)
	})
}
