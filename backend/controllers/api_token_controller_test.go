package controllers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListApiTokens_Empty(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	router.GET("/api-tokens", ListApiTokens)

	req, _ := http.NewRequest("GET", "/api-tokens", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Tokens []models.ApiTokenResponse `json:"tokens"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Empty(t, body.Tokens)
}

func TestListApiTokens_ReturnsScopedTokens(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	var user models.User
	db.First(&user)

	// Token for a second user — must NOT appear in the response
	other := models.User{Username: "other", Email: "other@example.com", Password: "x"}
	db.Create(&other)

	db.Create(&models.ApiToken{UserID: user.ID, Name: "my-token", TokenHash: "hash1"})
	db.Create(&models.ApiToken{UserID: other.ID, Name: "other-token", TokenHash: "hash2"})

	router.GET("/api-tokens", ListApiTokens)

	req, _ := http.NewRequest("GET", "/api-tokens", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Tokens []models.ApiTokenResponse `json:"tokens"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Tokens, 1)
	assert.Equal(t, "my-token", body.Tokens[0].Name)
}

func TestListApiTokens_OrderedByCreatedAtDesc(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	var user models.User
	db.First(&user)

	db.Create(&models.ApiToken{UserID: user.ID, Name: "first", TokenHash: "hash-first"})
	db.Create(&models.ApiToken{UserID: user.ID, Name: "second", TokenHash: "hash-second"})

	router.GET("/api-tokens", ListApiTokens)

	req, _ := http.NewRequest("GET", "/api-tokens", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Tokens []models.ApiTokenResponse `json:"tokens"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Tokens, 2)
	assert.Equal(t, "second", body.Tokens[0].Name)
	assert.Equal(t, "first", body.Tokens[1].Name)
}

func TestListApiTokens_RevokedTokensAreIncluded(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	var user models.User
	db.First(&user)

	now := time.Now()
	db.Create(&models.ApiToken{UserID: user.ID, Name: "active", TokenHash: "h1"})
	db.Create(&models.ApiToken{UserID: user.ID, Name: "revoked", TokenHash: "h2", RevokedAt: &now})

	router.GET("/api-tokens", ListApiTokens)

	req, _ := http.NewRequest("GET", "/api-tokens", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Tokens []models.ApiTokenResponse `json:"tokens"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Tokens, 2)

	var revokedFound bool
	for _, tok := range body.Tokens {
		if tok.Name == "revoked" {
			assert.NotNil(t, tok.RevokedAt)
			revokedFound = true
		}
	}
	assert.True(t, revokedFound)
}

func TestCreateApiToken_Success(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	router.POST("/api-tokens", withValidated(func() any { return &models.ApiTokenInput{} }), CreateApiToken)

	input := models.ApiTokenInput{Name: "ci-token"}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/api-tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.ApiTokenCreateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ci-token", resp.Name)
	assert.NotZero(t, resp.ID)
	assert.Contains(t, resp.Token, "meerkat_")
	assert.Nil(t, resp.LastUsedAt)
	assert.Nil(t, resp.RevokedAt)

	// Plaintext is never stored; only the SHA-256 hash is persisted
	var stored models.ApiToken
	require.NoError(t, db.First(&stored, resp.ID).Error)
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(resp.Token)))
	assert.Equal(t, expectedHash, stored.TokenHash)
	assert.NotContains(t, stored.TokenHash, "meerkat_")
}

func TestCreateApiToken_MissingName(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	router.POST("/api-tokens", middleware.ValidateJSONMiddleware(&models.ApiTokenInput{}), CreateApiToken)

	body, _ := json.Marshal(map[string]string{"name": ""})
	req, _ := http.NewRequest("POST", "/api-tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRevokeApiToken_Success(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	var user models.User
	db.First(&user)

	token := models.ApiToken{UserID: user.ID, Name: "to-revoke", TokenHash: "somehash"}
	db.Create(&token)

	router.DELETE("/api-tokens/:id", RevokeApiToken)

	req, _ := http.NewRequest("DELETE", "/api-tokens/"+strconv.Itoa(int(token.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Token revoked successfully", resp["message"])

	var updated models.ApiToken
	db.First(&updated, token.ID)
	assert.NotNil(t, updated.RevokedAt)
	assert.WithinDuration(t, time.Now(), *updated.RevokedAt, 5*time.Second)
}

func TestRevokeApiToken_NotFound(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	router.DELETE("/api-tokens/:id", RevokeApiToken)

	req, _ := http.NewRequest("DELETE", "/api-tokens/9999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRevokeApiToken_WrongUser(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	// setupRouter seeds a user and sets its ID in the context.
	// Create a second user and put their token in the DB.
	other := models.User{Username: "attacker", Email: "attacker@example.com", Password: "x"}
	db.Create(&other)

	token := models.ApiToken{UserID: other.ID, Name: "victim-token", TokenHash: "victimhash"}
	db.Create(&token)

	router.DELETE("/api-tokens/:id", RevokeApiToken)

	req, _ := http.NewRequest("DELETE", "/api-tokens/"+strconv.Itoa(int(token.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Must not find a token belonging to a different user
	assert.Equal(t, http.StatusNotFound, w.Code)

	var unchanged models.ApiToken
	db.First(&unchanged, token.ID)
	assert.Nil(t, unchanged.RevokedAt)
}

func TestRevokeApiToken_InvalidID(t *testing.T) {
	db, router := setupRouter()
	db.AutoMigrate(&models.ApiToken{})

	router.DELETE("/api-tokens/:id", RevokeApiToken)

	req, _ := http.NewRequest("DELETE", "/api-tokens/not-a-number", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
