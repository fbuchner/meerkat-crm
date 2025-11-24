package services

import (
	"meerkat/config"
	"meerkat/models"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	password := "mypassword123"

	hashedPassword, err := HashPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)

	// Verify that the hashed password matches the original password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	assert.NoError(t, err)
}

func TestHashPassword_Error(t *testing.T) {
	// Simulate an error by providing an empty password
	_, err := HashPassword("")

	assert.Error(t, err)
}

func TestGenerateToken(t *testing.T) {
	config := config.Config{
		JWTSecretKey:   "mysecretkey",
		JWTExpiryHours: 24,
	}

	user := models.User{
		Username: "testuser",
	}

	tokenString, err := GenerateToken(user, &config)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(config.JWTSecretKey), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, user.Username, claims["username"])
	assert.True(t, claims["authorized"].(bool)) // Check if claims authorize is true
}

func TestGenerateToken_Error(t *testing.T) {
	config := config.Config{
		JWTSecretKey: "",
	}

	user := models.User{
		Username: "testuser",
	}

	// Attempts to generate a token should fail
	_, err := GenerateToken(user, &config)

	assert.Error(t, err)
}
