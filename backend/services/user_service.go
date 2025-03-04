package services

import (
	"errors"
	"perema/config"
	"perema/models"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func GenerateToken(user models.User, cfg *config.Config) (string, error) {
	JWTSecretKey := cfg.JWTSecretKey
	if JWTSecretKey == "" {
		return "", errors.New("JWT secret key is empty")
	}

	JWTExpiryHours := cfg.JWTExpiryHours
	if JWTExpiryHours <= 0 {
		return "", errors.New("JWT expiry hours is invalid")
	}

	claims := jwt.MapClaims{
		"authorized": true,
		"username":   user.Username,
		"exp":        time.Now().Add(time.Hour * time.Duration(JWTExpiryHours)).Unix(), // Token valid for 1 hour
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(JWTSecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
