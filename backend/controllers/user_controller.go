package controllers

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"

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

	// Normalize email and username to lowercase for case-insensitive matching
	user.Email = strings.ToLower(user.Email)
	user.Username = strings.ToLower(user.Username)

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

// LoginInput represents the DTO for login requests
type LoginInput struct {
	Identifier string `json:"identifier"` // Can be username or email
	Email      string `json:"email"`      // Legacy field for backward compatibility
	Password   string `json:"password"`
}

func LoginUser(context *gin.Context, cfg *config.Config) {
	var input LoginInput
	var foundUser models.User

	err := context.ShouldBindJSON(&input)
	if err != nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", err.Error()))
		return
	}

	// Support both "identifier" and legacy "email" field
	identifier := input.Identifier
	if identifier == "" {
		identifier = input.Email
	}

	if identifier == "" {
		apperrors.AbortWithError(context, apperrors.ErrMissingField("identifier"))
		return
	}

	// Normalize to lowercase for case-insensitive matching
	identifier = strings.ToLower(identifier)

	if input.Password == "" {
		apperrors.AbortWithError(context, apperrors.ErrMissingField("password"))
		return
	}

	// Check per-account rate limiting before attempting authentication
	accountLimiter := middleware.GetAccountRateLimiter()
	if isLocked, remainingSecs := accountLimiter.IsLocked(identifier); isLocked {
		context.JSON(http.StatusTooManyRequests, gin.H{
			"error":          "Account temporarily locked",
			"message":        "Too many failed login attempts. Please try again later.",
			"retry_after":    remainingSecs,
			"retry_after_at": time.Now().Add(time.Duration(remainingSecs) * time.Second).Format(time.RFC3339),
		})
		context.Abort()
		return
	}

	db := context.MustGet("db").(*gorm.DB)

	// Check if identifier contains @ to determine if it's an email or username
	var query *gorm.DB
	if strings.Contains(identifier, "@") {
		query = db.Where("email = ?", identifier)
	} else {
		query = db.Where("username = ?", identifier)
	}

	if err := query.First(&foundUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Record failed attempt even for non-existent users to prevent enumeration
			accountLimiter.RecordFailedAttempt(identifier)
			apperrors.AbortWithError(context, apperrors.ErrInvalidCredentials())
		} else {
			apperrors.AbortWithError(context, apperrors.ErrDatabase("Failed to query user").WithError(err))
		}
		return
	}

	// Compare the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(input.Password)); err != nil {
		// Record failed attempt for password mismatch
		isLocked, lockoutSecs := accountLimiter.RecordFailedAttempt(identifier)
		if isLocked {
			context.JSON(http.StatusTooManyRequests, gin.H{
				"error":          "Account temporarily locked",
				"message":        "Too many failed login attempts. Please try again later.",
				"retry_after":    lockoutSecs,
				"retry_after_at": time.Now().Add(time.Duration(lockoutSecs) * time.Second).Format(time.RFC3339),
			})
			context.Abort()
			return
		}
		apperrors.AbortWithError(context, apperrors.ErrInvalidCredentials())
		return
	}

	// Successful login - clear any failed attempt tracking
	accountLimiter.RecordSuccessfulLogin(identifier)

	// Create JWT token
	tokenString, err := services.GenerateToken(foundUser, cfg)
	if err != nil {
		apperrors.AbortWithError(context, apperrors.ErrInternal("Could not generate token").WithError(err))
		return
	}

	// Return token and user preferences
	context.JSON(http.StatusOK, gin.H{
		"token":    tokenString,
		"language": foundUser.Language,
	})
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

// RequestPasswordReset generates a reset token and sends instructions to the user.
func RequestPasswordReset(context *gin.Context, cfg *config.Config) {
	// Check if demo mode is enabled - password changes are disabled in demo
	if os.Getenv("DEMO_MODE") == "true" {
		apperrors.AbortWithError(context, apperrors.ErrForbidden("Password changes are disabled in demo mode"))
		return
	}

	log := logger.FromContext(context)

	validated, exists := context.Get("validated")
	if !exists {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", "validation data not found"))
		return
	}

	inputPtr, ok := validated.(*models.PasswordResetRequestInput)
	if !ok || inputPtr == nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", "invalid validation data type"))
		return
	}
	input := *inputPtr

	// Normalize email to lowercase for case-insensitive matching
	email := strings.ToLower(input.Email)

	db := context.MustGet("db").(*gorm.DB)

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			context.JSON(http.StatusOK, gin.H{"message": "If an account exists, password reset instructions were sent"})
			return
		}

		log.Error().Err(err).Msg("Failed to lookup user for password reset")
		apperrors.AbortWithError(context, apperrors.ErrDatabase("query user").WithError(err))
		return
	}

	token, hash, err := services.GeneratePasswordResetToken()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate password reset token")
		apperrors.AbortWithError(context, apperrors.ErrInternal("Could not generate password reset token").WithError(err))
		return
	}

	expires := services.PasswordResetExpiry()
	requested := time.Now()

	user.PasswordResetTokenHash = &hash
	user.PasswordResetExpiresAt = &expires
	user.PasswordResetRequestedAt = &requested

	if err := db.Save(&user).Error; err != nil {
		log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to persist password reset token")
		apperrors.AbortWithError(context, apperrors.ErrDatabase("update user").WithError(err))
		return
	}

	if err := services.SendPasswordResetEmail(user.Email, token, user.Language, cfg); err != nil {
		log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to send password reset email")
		apperrors.AbortWithError(context, apperrors.ErrExternal("email", "Failed to send password reset email").WithError(err))
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "If an account exists, password reset instructions were sent"})
}

// ConfirmPasswordReset validates the token and updates the password.
func ConfirmPasswordReset(context *gin.Context) {
	// Check if demo mode is enabled - password changes are disabled in demo
	if os.Getenv("DEMO_MODE") == "true" {
		apperrors.AbortWithError(context, apperrors.ErrForbidden("Password changes are disabled in demo mode"))
		return
	}

	log := logger.FromContext(context)

	validated, exists := context.Get("validated")
	if !exists {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", "validation data not found"))
		return
	}

	inputPtr, ok := validated.(*models.PasswordResetConfirmInput)
	if !ok || inputPtr == nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", "invalid validation data type"))
		return
	}
	input := *inputPtr

	db := context.MustGet("db").(*gorm.DB)

	tokenHash := services.HashPasswordResetToken(input.Token)

	var user models.User
	if err := db.Where("password_reset_token_hash = ?", tokenHash).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(context, apperrors.ErrInvalidInput("token", "Password reset token is invalid or expired"))
			return
		}

		log.Error().Err(err).Msg("Failed to lookup password reset token")
		apperrors.AbortWithError(context, apperrors.ErrDatabase("query user").WithError(err))
		return
	}

	if user.PasswordResetExpiresAt == nil || time.Now().After(*user.PasswordResetExpiresAt) {
		user.PasswordResetTokenHash = nil
		user.PasswordResetExpiresAt = nil
		user.PasswordResetRequestedAt = nil
		if err := db.Save(&user).Error; err != nil {
			log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to clear expired reset token")
		}
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("token", "Password reset token is invalid or expired"))
		return
	}

	hashedPassword, err := services.HashPassword(input.Password)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password during reset")
		apperrors.AbortWithError(context, apperrors.ErrInternal("Could not hash password").WithError(err))
		return
	}

	user.Password = hashedPassword
	user.PasswordResetTokenHash = nil
	user.PasswordResetExpiresAt = nil
	user.PasswordResetRequestedAt = nil

	if err := db.Save(&user).Error; err != nil {
		log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to persist password reset")
		apperrors.AbortWithError(context, apperrors.ErrDatabase("update user").WithError(err))
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Password reset successful"})
}

// ChangePassword lets authenticated users rotate their password.
// UpdateLanguageInput represents the request body for updating user language
type UpdateLanguageInput struct {
	Language string `json:"language" validate:"required,oneof=en de"`
}

// UpdateLanguage updates the authenticated user's language preference
func UpdateLanguage(context *gin.Context) {
	log := logger.FromContext(context)

	var input UpdateLanguageInput
	if err := context.ShouldBindJSON(&input); err != nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("language", "Invalid language value"))
		return
	}

	// Validate language is supported
	if input.Language != "en" && input.Language != "de" {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("language", "Unsupported language. Supported: en, de"))
		return
	}

	usernameValue, exists := context.Get("username")
	if !exists {
		apperrors.AbortWithError(context, apperrors.ErrUnauthorized("Authentication required"))
		return
	}

	username, ok := usernameValue.(string)
	if !ok || username == "" {
		apperrors.AbortWithError(context, apperrors.ErrUnauthorized("Authentication required"))
		return
	}

	db := context.MustGet("db").(*gorm.DB)

	var user models.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(context, apperrors.ErrUnauthorized("Authentication required"))
			return
		}

		log.Error().Err(err).Msg("Failed to lookup user for language update")
		apperrors.AbortWithError(context, apperrors.ErrDatabase("query user").WithError(err))
		return
	}

	user.Language = input.Language
	if err := db.Save(&user).Error; err != nil {
		log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to update user language")
		apperrors.AbortWithError(context, apperrors.ErrDatabase("update user").WithError(err))
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Language updated successfully", "language": user.Language})
}

// UpdateCustomFieldNames updates the authenticated user's custom field definitions
func UpdateCustomFieldNames(c *gin.Context) {
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Get validated input
	input, err := middleware.GetValidated[models.CustomFieldNamesInput](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		log.Error().Err(err).Uint("user_id", userID).Msg("Failed to lookup user for custom field names update")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query user").WithError(err))
		return
	}

	user.CustomFieldNames = input.Names
	if err := db.Save(&user).Error; err != nil {
		log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to update user custom field names")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("update user").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Custom field names updated successfully",
		"custom_field_names": user.CustomFieldNames,
	})
}

// GetCustomFieldNames returns the authenticated user's custom field definitions
func GetCustomFieldNames(c *gin.Context) {
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		log.Error().Err(err).Uint("user_id", userID).Msg("Failed to lookup user for custom field names")
		apperrors.AbortWithError(c, apperrors.ErrDatabase("query user").WithError(err))
		return
	}

	// Return empty array instead of null if not set
	names := user.CustomFieldNames
	if names == nil {
		names = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"custom_field_names": names,
	})
}

func ChangePassword(context *gin.Context) {
	// Check if demo mode is enabled - password changes are disabled in demo
	if os.Getenv("DEMO_MODE") == "true" {
		apperrors.AbortWithError(context, apperrors.ErrForbidden("Password changes are disabled in demo mode"))
		return
	}

	log := logger.FromContext(context)

	validated, exists := context.Get("validated")
	if !exists {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", "validation data not found"))
		return
	}

	inputPtr, ok := validated.(*models.ChangePasswordInput)
	if !ok || inputPtr == nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("", "invalid validation data type"))
		return
	}
	input := *inputPtr

	usernameValue, exists := context.Get("username")
	if !exists {
		apperrors.AbortWithError(context, apperrors.ErrUnauthorized("Authentication required"))
		return
	}

	username, ok := usernameValue.(string)
	if !ok || username == "" {
		apperrors.AbortWithError(context, apperrors.ErrUnauthorized("Authentication required"))
		return
	}

	db := context.MustGet("db").(*gorm.DB)

	var user models.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(context, apperrors.ErrUnauthorized("Authentication required"))
			return
		}

		log.Error().Err(err).Msg("Failed to lookup user for password change")
		apperrors.AbortWithError(context, apperrors.ErrDatabase("query user").WithError(err))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.CurrentPassword)); err != nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("current_password", "Current password is incorrect"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.NewPassword)); err == nil {
		apperrors.AbortWithError(context, apperrors.ErrInvalidInput("new_password", "New password must differ from current password"))
		return
	}

	hashedPassword, err := services.HashPassword(input.NewPassword)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password during change")
		apperrors.AbortWithError(context, apperrors.ErrInternal("Could not hash password").WithError(err))
		return
	}

	user.Password = hashedPassword
	user.PasswordResetTokenHash = nil
	user.PasswordResetExpiresAt = nil
	user.PasswordResetRequestedAt = nil

	if err := db.Save(&user).Error; err != nil {
		log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to persist password change")
		apperrors.AbortWithError(context, apperrors.ErrDatabase("update user").WithError(err))
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
