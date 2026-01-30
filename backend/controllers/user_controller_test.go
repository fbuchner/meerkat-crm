package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meerkat/config"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

const (
	strongPassword        = "CorrectHorseBattery42!"
	strongPasswordAlt     = "TrulySecurePassphrase99#"
	strongPasswordAnother = "UltraSafePassphrase88$"
)

func TestRegisterUser(t *testing.T) {
	_, router := setupRouter()
	router.POST("/register", middleware.ValidateJSONMiddleware(&models.UserRegistrationInput{}), RegisterUser)

	// Create a new user using the registration DTO
	newUser := models.UserRegistrationInput{
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: strongPassword,
	}

	jsonValue, _ := json.Marshal(newUser)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "User registered successfully", responseBody["message"])
}

func TestRegisterUser_InvalidInput(t *testing.T) {
	_, router := setupRouter()
	router.POST("/register", middleware.ValidateJSONMiddleware(&models.UserRegistrationInput{}), RegisterUser)

	// Invalid input (no email)
	invalidUser := models.UserRegistrationInput{
		Username: "invaliduser",
		Password: strongPassword,
	}

	jsonValue, _ := json.Marshal(invalidUser)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	errorDetail := response["error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errorDetail["code"])
}

func TestLoginUser(t *testing.T) {
	config := config.Config{
		JWTSecretKey:   "mysecretkey",
		JWTExpiryHours: 24,
	}

	db, router := setupRouter()
	router.POST("/login", func(c *gin.Context) {
		LoginUser(c, &config)
	})

	// First, register a user to test login
	newUser := models.User{
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: strongPassword,
	}
	hashedPassword, _ := services.HashPassword(newUser.Password)
	newUser.Password = hashedPassword
	db.Create(&newUser)

	// Now try to login with email
	loginData := map[string]string{
		"identifier": "testuser@example.com",
		"password":   strongPassword,
	}

	jsonValue, _ := json.Marshal(loginData)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Contains(t, responseBody, "token")
}

func TestLoginUser_WithUsername(t *testing.T) {
	config := config.Config{
		JWTSecretKey:   "mysecretkey",
		JWTExpiryHours: 24,
	}

	db, router := setupRouter()
	router.POST("/login", func(c *gin.Context) {
		LoginUser(c, &config)
	})

	// First, register a user to test login
	newUser := models.User{
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: strongPassword,
	}
	hashedPassword, _ := services.HashPassword(newUser.Password)
	newUser.Password = hashedPassword
	db.Create(&newUser)

	// Now try to login with username
	loginData := map[string]string{
		"identifier": "testuser",
		"password":   strongPassword,
	}

	jsonValue, _ := json.Marshal(loginData)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Contains(t, responseBody, "token")
}

func TestLoginUser_LegacyEmailField(t *testing.T) {
	config := config.Config{
		JWTSecretKey:   "mysecretkey",
		JWTExpiryHours: 24,
	}

	db, router := setupRouter()
	router.POST("/login", func(c *gin.Context) {
		LoginUser(c, &config)
	})

	// First, register a user to test login
	newUser := models.User{
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: strongPassword,
	}
	hashedPassword, _ := services.HashPassword(newUser.Password)
	newUser.Password = hashedPassword
	db.Create(&newUser)

	// Now try to login with legacy email field (backward compatibility)
	loginData := map[string]string{
		"email":    "testuser@example.com",
		"password": strongPassword,
	}

	jsonValue, _ := json.Marshal(loginData)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Contains(t, responseBody, "token")
}

func TestLoginUser_InvalidCredentials(t *testing.T) {
	config := config.Config{
		JWTSecretKey:   "mysecretkey",
		JWTExpiryHours: 24,
	}
	_, router := setupRouter()
	router.POST("/login", func(c *gin.Context) {
		LoginUser(c, &config)
	})
	// Try to login with unregistered user
	loginData := map[string]string{
		"identifier": "wronguser@example.com",
		"password":   "wrongpassword",
	}

	jsonValue, _ := json.Marshal(loginData)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	errorDetail := response["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_CREDENTIALS", errorDetail["code"])
}

func TestLoginUser_InvalidInput(t *testing.T) {
	config := config.Config{
		JWTSecretKey: "mysecretkey",
	}
	_, router := setupRouter()
	router.POST("/login", func(c *gin.Context) {
		LoginUser(c, &config)
	})

	// Trying to login with invalid input (missing identifier)
	invalidData := map[string]string{
		"password": strongPassword,
	}

	jsonValue, _ := json.Marshal(invalidData)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	errorDetail := response["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_FIELD", errorDetail["code"])
}

func TestRequestPasswordReset_Succeeds(t *testing.T) {
	cfg := config.Config{
		FrontendURL: "http://localhost:3000",
		UseResend:   false,
	}

	db, router := setupRouter()

	hashed, _ := services.HashPassword(strongPassword)
	user := models.User{
		Username: "resetuser",
		Email:    "reset@example.com",
		Password: hashed,
	}
	db.Create(&user)

	router.POST("/password-reset/request", func(c *gin.Context) {
		c.Set("validated", &models.PasswordResetRequestInput{Email: "reset@example.com"})
		RequestPasswordReset(c, &cfg)
	})

	req, _ := http.NewRequest("POST", "/password-reset/request", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated models.User
	db.Where("email = ?", "reset@example.com").First(&updated)
	if assert.NotNil(t, updated.PasswordResetTokenHash) {
		assert.NotEmpty(t, *updated.PasswordResetTokenHash)
	}
	if assert.NotNil(t, updated.PasswordResetExpiresAt) {
		assert.True(t, updated.PasswordResetExpiresAt.After(time.Now().Add(-time.Minute)))
	}
	if assert.NotNil(t, updated.PasswordResetRequestedAt) {
		assert.True(t, updated.PasswordResetRequestedAt.After(time.Now().Add(-time.Minute)))
	}
}

func TestConfirmPasswordReset_Succeeds(t *testing.T) {
	db, router := setupRouter()

	initialPassword, _ := services.HashPassword(strongPassword)
	token, tokenHash, _ := services.GeneratePasswordResetToken()
	expires := services.PasswordResetExpiry()
	requested := time.Now()

	user := models.User{
		Username:                 "confirmuser",
		Email:                    "confirm@example.com",
		Password:                 initialPassword,
		PasswordResetTokenHash:   &tokenHash,
		PasswordResetExpiresAt:   &expires,
		PasswordResetRequestedAt: &requested,
	}
	db.Create(&user)

	router.POST("/password-reset/confirm", func(c *gin.Context) {
		c.Set("validated", &models.PasswordResetConfirmInput{Token: token, Password: strongPasswordAlt})
		ConfirmPasswordReset(c)
	})

	req, _ := http.NewRequest("POST", "/password-reset/confirm", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated models.User
	db.Where("email = ?", "confirm@example.com").First(&updated)
	assert.Nil(t, updated.PasswordResetTokenHash)
	assert.Nil(t, updated.PasswordResetExpiresAt)
	assert.Nil(t, updated.PasswordResetRequestedAt)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte(strongPasswordAlt)))
}

func TestChangePassword_Succeeds(t *testing.T) {
	db, router := setupRouter()

	initialPassword, _ := services.HashPassword(strongPassword)
	user := models.User{
		Username: "changeme",
		Email:    "change@example.com",
		Password: initialPassword,
	}
	db.Create(&user)

	router.POST("/change-password", func(c *gin.Context) {
		c.Set("username", "changeme")
		c.Set("validated", &models.ChangePasswordInput{
			CurrentPassword: strongPassword,
			NewPassword:     strongPasswordAnother,
		})
		ChangePassword(c)
	})

	req, _ := http.NewRequest("POST", "/change-password", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated models.User
	db.Where("username = ?", "changeme").First(&updated)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte(strongPasswordAnother)))
	assert.Nil(t, updated.PasswordResetTokenHash)
	assert.Nil(t, updated.PasswordResetExpiresAt)
	assert.Nil(t, updated.PasswordResetRequestedAt)
}
