package errors

import (
	"fmt"
	"meerkat/logger"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents the JSON structure returned to clients
type ErrorResponse struct {
	Error     ErrorDetail `json:"error"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorHandlerMiddleware handles panics and formats error responses
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				stackTrace := string(debug.Stack())

				log := logger.FromContext(c)
				log.Error().
					Str("panic", fmt.Sprintf("%v", err)).
					Str("stack_trace", stackTrace).
					Msg("Panic recovered")

				// Create internal error response
				appErr := ErrInternal("An unexpected error occurred")
				appErr.WithDetails("panic", fmt.Sprintf("%v", err))

				RespondWithError(c, appErr)
			}
		}()

		c.Next()

		// Check if there were any errors during request processing
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last().Err

			// Convert to AppError
			appErr := GetAppError(err)

			// Only respond if we haven't already written a response
			if !c.Writer.Written() {
				RespondWithError(c, appErr)
			}
		}
	}
}

// RespondWithError sends a formatted error response
func RespondWithError(c *gin.Context, err *AppError) {
	requestID, _ := c.Get("request_id")
	requestIDStr, _ := requestID.(string)

	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Log the error with context
	LogError(c, err)

	c.JSON(err.HTTPStatus, response)
	c.Abort()
}

// LogError logs an error with context
func LogError(c *gin.Context, err *AppError) {
	log := logger.FromContext(c)

	event := log.Error().
		Str("code", err.Code).
		Int("status", err.HTTPStatus).
		Str("error", err.Message)

	// Include underlying error if present
	if err.Err != nil {
		event = event.Err(err.Err)
	}

	// Include details if present
	if len(err.Details) > 0 {
		event = event.Interface("details", err.Details)
	}

	event.Msg("Request error")
}

// AbortWithError aborts the request and responds with an error
func AbortWithError(c *gin.Context, err *AppError) {
	RespondWithError(c, err)
}

// AbortWithCustomError creates and responds with a custom error
func AbortWithCustomError(c *gin.Context, code, message string, httpStatus int) {
	err := NewError(code, message, httpStatus)
	RespondWithError(c, err)
}
