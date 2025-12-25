package middleware

import (
	apperrors "meerkat/errors"
	"meerkat/logger"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Global validator instance
var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validators
	validate.RegisterValidation("phone", validatePhone)
	validate.RegisterValidation("birthday", validateBirthday)
	validate.RegisterValidation("safe_string", validateSafeString)
	validate.RegisterValidation("strong_password", validateStrongPassword)
	validate.RegisterValidation("unique_circles", validateUniqueCircles)
	validate.RegisterValidation("no_at_sign", validateNoAtSign)
}

// ValidationError represents a validation error response
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateStruct validates a struct and returns formatted errors
func ValidateStruct(obj interface{}) []ValidationError {
	var errors []ValidationError

	err := validate.Struct(obj)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, ValidationError{
				Field:   err.Field(),
				Message: formatValidationError(err),
			})
		}
	}

	return errors
}

// formatValidationError creates user-friendly error messages
func formatValidationError(err validator.FieldError) string {
	field := err.Field()

	switch err.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		return field + " must be at least " + err.Param() + " characters"
	case "max":
		return field + " must be at most " + err.Param() + " characters"
	case "phone":
		return field + " must be a valid phone number"
	case "birthday":
		return field + " must be in DD.MM.YYYY format (YYYY optional)"
	case "safe_string":
		return field + " contains invalid characters"
	case "strong_password":
		return field + " is too weak. Use a longer password (15+ characters) or a passphrase (20+ characters). Avoid common passwords."
	case "unique_circles":
		return field + " cannot contain duplicate circles"
	case "no_at_sign":
		return field + " cannot contain the @ character"
	case "url":
		return field + " must be a valid URL"
	default:
		return field + " is invalid"
	}
}

// SanitizeString removes potentially dangerous characters and trims whitespace
func SanitizeString(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Remove control characters except newlines and tabs
	s = removeControlChars(s)

	return s
}

// removeControlChars removes control characters except newlines and tabs
func removeControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			return -1
		}
		return r
	}, s)
}

// validatePhone validates phone number format
// Accepts: +1234567890, (123) 456-7890, 123-456-7890, etc.
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true // Allow empty (use 'required' tag if needed)
	}

	// Remove common formatting characters
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '+' {
			return r
		}
		return -1
	}, phone)

	// Must have between 5 and 20 digits
	if len(cleaned) < 5 || len(cleaned) > 20 {
		return false
	}

	return true
}

// validateBirthday validates date format (DD.MM.YYYY or DD.MM.)
func validateBirthday(fl validator.FieldLevel) bool {
	birthday := fl.Field().String()
	if birthday == "" {
		return true // Allow empty (use 'required' tag if needed)
	}

	// Check format DD.MM.YYYY or DD.MM. (YYYY optional)
	match, _ := regexp.MatchString(`^\d{2}\.\d{2}\.(\d{4})?$`, birthday)
	if !match {
		return false
	}

	// Additional validation could check if date is valid
	return true
}

// validateSafeString checks for potentially dangerous characters
func validateSafeString(fl validator.FieldLevel) bool {
	str := fl.Field().String()

	// Check for SQL injection patterns
	dangerous := []string{
		"--", "/*", "*/", "xp_", "sp_", "exec", "execute",
		"script", "javascript:", "onerror", "onload",
	}

	lowerStr := strings.ToLower(str)
	for _, pattern := range dangerous {
		if strings.Contains(lowerStr, pattern) {
			return false
		}
	}

	return true
}

// validateStrongPassword checks password strength based on entropy
func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Check if it's a common password
	if IsCommonPassword(password) {
		return false
	}

	// Check entropy
	err := ValidatePasswordStrength(password)
	return err == nil
}

// validateUniqueCircles checks that all circles in the slice are unique (case-insensitive)
func validateUniqueCircles(fl validator.FieldLevel) bool {
	circles := fl.Field()
	if circles.Kind() != reflect.Slice {
		return true // Not a slice, let other validators handle this
	}

	seen := make(map[string]bool)
	for i := 0; i < circles.Len(); i++ {
		circle := circles.Index(i)
		if circle.Kind() != reflect.String {
			continue // Skip non-string elements
		}

		circleValue := strings.ToLower(strings.TrimSpace(circle.String()))
		if circleValue == "" {
			continue // Skip empty strings
		}

		if seen[circleValue] {
			return false // Duplicate found
		}
		seen[circleValue] = true
	}

	return true
}

// validateNoAtSign checks that a string field does not contain the @ character
func validateNoAtSign(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return !strings.Contains(value, "@")
}

// ValidateEmail validates email format with regex
func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateJSONMiddleware validates JSON request body against a struct
// Usage: router.POST("/endpoint", ValidateJSONMiddleware(&MyStruct{}), handler)
func ValidateJSONMiddleware(template interface{}) gin.HandlerFunc {
	// Get the type of the template to create new instances per request
	templateType := reflect.TypeOf(template)
	if templateType.Kind() == reflect.Ptr {
		templateType = templateType.Elem()
	}

	return func(c *gin.Context) {
		// Create a new instance of the struct for each request
		obj := reflect.New(templateType).Interface()

		if err := c.ShouldBindJSON(obj); err != nil {
			logger.FromContext(c).Warn().Err(err).Msg("Invalid JSON in request body")

			appErr := apperrors.ErrInvalidInput("request body", err.Error())
			apperrors.AbortWithError(c, appErr)
			return
		}

		// Validate the struct
		if validationErrors := ValidateStruct(obj); len(validationErrors) > 0 {
			logger.FromContext(c).Warn().Interface("validation_errors", validationErrors).Msg("Validation failed")

			// Build detailed validation error
			appErr := apperrors.ErrValidation("Request validation failed")
			for _, ve := range validationErrors {
				appErr.WithDetails(ve.Field, ve.Message)
			}
			apperrors.AbortWithError(c, appErr)
			return
		}

		// Store validated object in context
		c.Set("validated", obj)
		c.Next()
	}
}

// SanitizeContactInput sanitizes contact input fields
func SanitizeContactInput(contact interface{}) {
	// Use reflection to sanitize string fields
	// This is a placeholder - actual implementation would use reflection
	// to iterate over struct fields and sanitize string fields
}
