package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"perema/config"
	"perema/models"
	"perema/services"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegisterUser(t *testing.T) {
	_, router := setupRouter()
	router.POST("/register", RegisterUser)

	// Create a new user
	newUser := models.User{
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: "password123",
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
	router.POST("/register", RegisterUser)

	// Invalid input (no email)
	invalidUser := models.User{
		Username: "invaliduser",
		Password: "password123",
	}

	jsonValue, _ := json.Marshal(invalidUser)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Invalid input", responseBody["error"])
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
		Password: "password123",
	}
	hashedPassword, _ := services.HashPassword(newUser.Password)
	newUser.Password = hashedPassword
	db.Create(&newUser)

	// Now try to login
	loginUser := models.User{
		Email:    "testuser@example.com",
		Password: "password123",
	}

	jsonValue, _ := json.Marshal(loginUser)
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
	loginUser := models.User{
		Email:    "wronguser@example.com",
		Password: "wrongpassword",
	}

	jsonValue, _ := json.Marshal(loginUser)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Invalid credentials", responseBody["error"])
}

func TestLoginUser_InvalidInput(t *testing.T) {
	config := config.Config{
		JWTSecretKey: "mysecretkey",
	}
	_, router := setupRouter()
	router.POST("/login", func(c *gin.Context) {
		LoginUser(c, &config)
	})

	// Trying to login with invalid input (missing email)
	invalidUser := models.User{
		Password: "password123",
	}

	jsonValue, _ := json.Marshal(invalidUser)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Invalid input", responseBody["error"])
}
