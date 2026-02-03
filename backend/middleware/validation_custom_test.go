package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "meerkat/errors"
	"meerkat/logger"

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
		Username string `json:"username" validate:"required,min=3,max=20"`
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

func TestValidateUniqueCircles(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger(logger.Config{
		Level:  "error",
		Pretty: false,
	})

	gin.SetMode(gin.TestMode)

	type TestContactStruct struct {
		Firstname string   `json:"firstname" validate:"required"`
		Circles   []string `json:"circles" validate:"unique_circles"`
	}

	tests := []struct {
		name           string
		circles        []string
		expectedStatus int
		expectedValid  bool
		description    string
	}{
		{
			name:           "valid_unique_circles",
			circles:        []string{"family", "work", "friends"},
			expectedStatus: http.StatusOK,
			expectedValid:  true,
			description:    "all circles are unique",
		},
		{
			name:           "valid_empty_circles",
			circles:        []string{},
			expectedStatus: http.StatusOK,
			expectedValid:  true,
			description:    "empty circles array is valid",
		},
		{
			name:           "invalid_duplicate_circles",
			circles:        []string{"family", "work", "family"},
			expectedStatus: http.StatusBadRequest,
			expectedValid:  false,
			description:    "duplicate circles should be invalid",
		},
		{
			name:           "invalid_case_insensitive_duplicates",
			circles:        []string{"family", "FAMILY", "Family"},
			expectedStatus: http.StatusBadRequest,
			expectedValid:  false,
			description:    "case-insensitive duplicates should be invalid",
		},
		{
			name:           "valid_with_whitespace_trimming",
			circles:        []string{" family ", "work", "friends "},
			expectedStatus: http.StatusOK,
			expectedValid:  true,
			description:    "whitespace should be trimmed and validation should pass",
		},
		{
			name:           "invalid_duplicate_after_trimming",
			circles:        []string{"family", " family ", "work"},
			expectedStatus: http.StatusBadRequest,
			expectedValid:  false,
			description:    "duplicates after trimming whitespace should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(apperrors.ErrorHandlerMiddleware())

			router.POST("/test", ValidateJSONMiddleware(&TestContactStruct{}), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			payload := map[string]interface{}{
				"firstname": "John",
				"circles":   tt.circles,
			}

			jsonPayload, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(jsonPayload))
			req.Header.Set("Content-Type", "application/json")

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code, "Status code mismatch for test: %s", tt.description)

			if !tt.expectedValid {
				var response apperrors.ErrorResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
				assert.Contains(t, response.Error.Details, "Circles", "Expected Circles field in validation errors")
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "success", response["message"])
			}
		})
	}
}
