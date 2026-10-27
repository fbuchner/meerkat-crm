package services

import (
	"errors"
	"meerkat/config"
	"meerkat/middleware"
	"meerkat/models"

	"golang.org/x/crypto/bcrypt"
)

var ErrPasswordTooLong = errors.New("password must not exceed 72 characters")

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	if len([]byte(password)) > 72 {
		return "", ErrPasswordTooLong
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func GenerateToken(user models.User, cfg *config.Config) (string, error) {
	return middleware.GenerateToken(user, cfg)
}
