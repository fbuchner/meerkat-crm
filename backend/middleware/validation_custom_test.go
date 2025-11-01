package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "perema/errors"
	"perema/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestValidationWithErrorSystem(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger(logger.Config{
		Level:  "error",
		Pretty: false,
	})

	gin.SetMode(gin.TestMode)

	type TestStruct struct {
		Email    string `json:"email" validate:"required,email"`
		Username string `json:"username" validate:"required,min=3,max=20,safe_string"`
		Phone    string `json:"phone" validate:"required,phone"`
	}

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
		expectedCode   string
		checkField     string
	}{
		{
			name: "valid_input",
			payload: map[string]string{
				"email":    "test@example.com",
				"username": "testuser",
				"phone":    "1234567890",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid_email",
			payload: map[string]string{
				"email":    "invalid-email",
				"username": "testuser",
				"phone":    "1234567890",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
			checkField:     "Email", // Note: Field names are capitalized in validation errors
		},
		{
			name: "missing_required_fields",
			payload: map[string]string{
				"email": "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
			checkField:     "Username", // Note: Field names are capitalized in validation errors
		},
		{
			name: "invalid_phone",
			payload: map[string]string{
				"email":    "test@example.com",
				"username": "testuser",
				"phone":    "123", // Too short
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
			checkField:     "Phone", // Note: Field names are capitalized in validation errors
		},
		{
			name: "unsafe_string",
			payload: map[string]string{
				"email":    "test@example.com",
				"username": "<script>alert('xss')</script>",
				"phone":    "1234567890",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
			checkField:     "Username", // Note: Field names are capitalized in validation errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(apperrors.ErrorHandlerMiddleware())

			router.POST("/test", ValidateJSONMiddleware(&TestStruct{}), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus != http.StatusOK {
				var response apperrors.ErrorResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCode, response.Error.Code)

				if tt.checkField != "" {
					assert.Contains(t, response.Error.Details, tt.checkField)
				}
			}
		})
	}
}

func TestValidationWithInvalidJSON(t *testing.T) {
	logger.InitLogger(logger.Config{
		Level:  "error",
		Pretty: false,
	})

	gin.SetMode(gin.TestMode)

	type TestStruct struct {
		Email string `json:"email" validate:"required,email"`
	}

	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestStruct{}), func(c *gin.Context) {
		t.Fatal("Handler should not be called")
	})

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response apperrors.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_INPUT", response.Error.Code)
}
