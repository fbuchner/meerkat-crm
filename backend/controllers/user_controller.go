package controllers

import (
	"errors"
	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func RegisterUser(context *gin.Context) {
	// Get validated user from middleware
	validated, exists := context.Get("validated")
	if !exists {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", "validation data not found"))
		return
	}

	userPtr, ok := validated.(*models.User)
	if !ok {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", "invalid validation data type"))
		return
	}
	user := *userPtr

	if user.Email == "" {
		apperrors.AbortWithError(context, apperrors.ErrMissingField("email"))
		return
	}

	if user.Password == "" {
		apperrors.AbortWithError(context, apperrors.ErrMissingField("password"))
		return
	}

	hashedPassword, err := services.HashPassword(user.Password)
	if err != nil {
		apperrors.AbortWithError(context, apperrors.ErrInternal("Could not hash password").WithError(err))
		return
	}
	user.Password = hashedPassword

	db := context.MustGet("db").(*gorm.DB)
	if err := db.Create(&user).Error; err != nil {
		apperrors.AbortWithError(context, apperrors.ErrAlreadyExists("User").WithDetails("email", user.Email))
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func LoginUser(context *gin.Context, cfg *config.Config) {
	var user models.User
	var foundUser models.User

	err := context.ShouldBindJSON(&user)
	if err != nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", err.Error()))
		return
	}

	if user.Email == "" {
		apperrors.AbortWithError(context, apperrors.ErrMissingField("email"))
		return
	}

	if user.Password == "" {
		apperrors.AbortWithError(context, apperrors.ErrMissingField("password"))
		return
	}

	db := context.MustGet("db").(*gorm.DB)
	if err := db.Where("email = ?", user.Email).First(&foundUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(context, apperrors.ErrInvalidCredentials())
		} else {
			apperrors.AbortWithError(context, apperrors.ErrDatabase("Failed to query user").WithError(err))
		}
		return
	}

	// Compare the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(user.Password)); err != nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidCredentials())
		return
	}

	// Create JWT token
	tokenString, err := services.GenerateToken(foundUser, cfg)
	if err != nil {
		apperrors.AbortWithError(context, apperrors.ErrInternal("Could not generate token").WithError(err))
		return
	}

	context.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// CheckPasswordStrength evaluates password strength without registration
func CheckPasswordStrength(context *gin.Context) {
	var request struct {
		Password string `json:"password" binding:"required"`
	}

	if err := context.ShouldBindJSON(&request); err != nil {
		apperrors.AbortWithError(context, apperrors.ErrMissingField("password"))
		return
	}

	strength := middleware.EvaluatePasswordStrength(request.Password)
	context.JSON(http.StatusOK, strength)
}
