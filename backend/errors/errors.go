package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a custom application error
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Err        error                  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap implements the errors.Unwrap interface
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithError wraps an underlying error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// Error codes
const (
	// Authentication & Authorization
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid       = "TOKEN_INVALID"
	ErrCodeForbidden          = "FORBIDDEN"

	// Resource errors
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeAlreadyExists = "ALREADY_EXISTS"
	ErrCodeConflict      = "CONFLICT"

	// Validation errors
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeInvalidInput = "INVALID_INPUT"
	ErrCodeMissingField = "MISSING_FIELD"

	// Rate limiting
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"

	// Internal errors
	ErrCodeInternal = "INTERNAL_ERROR"
	ErrCodeDatabase = "DATABASE_ERROR"
	ErrCodeExternal = "EXTERNAL_SERVICE_ERROR"

	// Business logic errors
	ErrCodeBusinessLogic   = "BUSINESS_LOGIC_ERROR"
	ErrCodeOperationFailed = "OPERATION_FAILED"
)

// NewError creates a new AppError
func NewError(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

// --- Authentication & Authorization Errors ---

// ErrUnauthorized returns an unauthorized error
func ErrUnauthorized(message string) *AppError {
	if message == "" {
		message = "Authentication required"
	}
	return NewError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// ErrInvalidCredentials returns an invalid credentials error
func ErrInvalidCredentials() *AppError {
	return NewError(ErrCodeInvalidCredentials, "Invalid email or password", http.StatusUnauthorized)
}

// ErrTokenExpired returns a token expired error
func ErrTokenExpired() *AppError {
	return NewError(ErrCodeTokenExpired, "Authentication token has expired", http.StatusUnauthorized)
}

// ErrTokenInvalid returns a token invalid error
func ErrTokenInvalid() *AppError {
	return NewError(ErrCodeTokenInvalid, "Authentication token is invalid", http.StatusUnauthorized)
}

// ErrForbidden returns a forbidden error
func ErrForbidden(message string) *AppError {
	if message == "" {
		message = "You don't have permission to access this resource"
	}
	return NewError(ErrCodeForbidden, message, http.StatusForbidden)
}

// --- Resource Errors ---

// ErrNotFound returns a not found error
func ErrNotFound(resource string) *AppError {
	message := "Resource not found"
	if resource != "" {
		message = fmt.Sprintf("%s not found", resource)
	}
	return NewError(ErrCodeNotFound, message, http.StatusNotFound)
}

// ErrAlreadyExists returns an already exists error
func ErrAlreadyExists(resource string) *AppError {
	message := "Resource already exists"
	if resource != "" {
		message = fmt.Sprintf("%s already exists", resource)
	}
	return NewError(ErrCodeAlreadyExists, message, http.StatusConflict)
}

// ErrConflict returns a conflict error
func ErrConflict(message string) *AppError {
	if message == "" {
		message = "Resource conflict occurred"
	}
	return NewError(ErrCodeConflict, message, http.StatusConflict)
}

// --- Validation Errors ---

// ErrValidation returns a validation error
func ErrValidation(message string) *AppError {
	if message == "" {
		message = "Validation failed"
	}
	return NewError(ErrCodeValidation, message, http.StatusBadRequest)
}

// ErrInvalidInput returns an invalid input error
func ErrInvalidInput(field, reason string) *AppError {
	message := "Invalid input"
	if field != "" {
		message = fmt.Sprintf("Invalid value for field '%s'", field)
	}
	err := NewError(ErrCodeInvalidInput, message, http.StatusBadRequest)
	if reason != "" {
		err.WithDetails("reason", reason)
	}
	if field != "" {
		err.WithDetails("field", field)
	}
	return err
}

// ErrMissingField returns a missing field error
func ErrMissingField(field string) *AppError {
	message := "Required field is missing"
	if field != "" {
		message = fmt.Sprintf("Required field '%s' is missing", field)
	}
	err := NewError(ErrCodeMissingField, message, http.StatusBadRequest)
	if field != "" {
		err.WithDetails("field", field)
	}
	return err
}

// --- Rate Limiting Errors ---

// ErrRateLimitExceeded returns a rate limit exceeded error
func ErrRateLimitExceeded() *AppError {
	return NewError(ErrCodeRateLimitExceeded, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
}

// --- Internal Errors ---

// ErrInternal returns an internal server error
func ErrInternal(message string) *AppError {
	if message == "" {
		message = "An internal error occurred"
	}
	return NewError(ErrCodeInternal, message, http.StatusInternalServerError)
}

// ErrDatabase returns a database error
func ErrDatabase(operation string) *AppError {
	message := "Database operation failed"
	if operation != "" {
		message = fmt.Sprintf("Database %s operation failed", operation)
	}
	err := NewError(ErrCodeDatabase, message, http.StatusInternalServerError)
	if operation != "" {
		err.WithDetails("operation", operation)
	}
	return err
}

// ErrExternal returns an external service error
func ErrExternal(service, message string) *AppError {
	if message == "" {
		message = "External service error"
	}
	if service != "" {
		message = fmt.Sprintf("%s service error: %s", service, message)
	}
	err := NewError(ErrCodeExternal, message, http.StatusServiceUnavailable)
	if service != "" {
		err.WithDetails("service", service)
	}
	return err
}

// --- Business Logic Errors ---

// ErrBusinessLogic returns a business logic error
func ErrBusinessLogic(message string) *AppError {
	if message == "" {
		message = "Business logic constraint violated"
	}
	return NewError(ErrCodeBusinessLogic, message, http.StatusUnprocessableEntity)
}

// ErrOperationFailed returns an operation failed error
func ErrOperationFailed(operation, reason string) *AppError {
	message := "Operation failed"
	if operation != "" {
		message = fmt.Sprintf("%s operation failed", operation)
	}
	err := NewError(ErrCodeOperationFailed, message, http.StatusUnprocessableEntity)
	if operation != "" {
		err.WithDetails("operation", operation)
	}
	if reason != "" {
		err.WithDetails("reason", reason)
	}
	return err
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts an AppError from an error, or creates a generic internal error
func GetAppError(err error) *AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return ErrInternal("An unexpected error occurred").WithError(err)
}
