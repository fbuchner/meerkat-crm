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

func init() {
	gin.SetMode(gin.TestMode)
	logger.InitLogger(logger.Config{
		Level:  "error",
		Pretty: false,
	})
}

type TestValidationStruct struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=20,safe_string"`
	Phone    string `json:"phone" validate:"required,phone"`
	Birthday string `json:"birthday" validate:"omitempty,birthday"`
	Password string `json:"password" validate:"required,strong_password"`
}

func TestValidateJSONMiddleware_ValidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
		validated, exists := c.Get("validated")
		assert.True(t, exists)
		assert.NotNil(t, validated)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	payload := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"phone":    "1234567890",
		"birthday": "15.06.1990",
		"password": "StrongP@ssw0rd123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestValidateJSONMiddleware_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
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
	assert.Contains(t, response.Error.Message, "request body")
}

func TestValidateJSONMiddleware_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
		t.Fatal("Handler should not be called")
	})

	payload := map[string]string{
		"email": "test@example.com",
		// Missing username, phone, password
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response apperrors.AppError
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", response.Code)
	assert.NotNil(t, response.Details)

	// Check that all missing fields are reported
	assert.Contains(t, response.Details, "username")
	assert.Contains(t, response.Details, "phone")
	assert.Contains(t, response.Details, "password")
}

func TestValidateJSONMiddleware_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
		t.Fatal("Handler should not be called")
	})

	payload := map[string]string{
		"email":    "invalid-email",
		"username": "testuser",
		"phone":    "1234567890",
		"password": "StrongP@ssw0rd123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response apperrors.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(t, response.Error.Details, "email")
}

func TestValidateJSONMiddleware_InvalidPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
		t.Fatal("Handler should not be called")
	})

	payload := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"phone":    "123", // Too short
		"password": "StrongP@ssw0rd123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response apperrors.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(t, response.Error.Details, "phone")
}

func TestValidateJSONMiddleware_InvalidBirthday(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
		t.Fatal("Handler should not be called")
	})

	payload := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"phone":    "1234567890",
		"birthday": "1990-06-15", // Wrong format, should be DD.MM.YYYY
		"password": "StrongP@ssw0rd123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response apperrors.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(t, response.Error.Details, "birthday")
}

func TestValidateJSONMiddleware_WeakPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
		t.Fatal("Handler should not be called")
	})

	payload := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"phone":    "1234567890",
		"password": "weak", // Too weak
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response apperrors.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(t, response.Error.Details, "password")
}

func TestValidateJSONMiddleware_UnsafeString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
		t.Fatal("Handler should not be called")
	})

	payload := map[string]string{
		"email":    "test@example.com",
		"username": "<script>alert('xss')</script>",
		"phone":    "1234567890",
		"password": "StrongP@ssw0rd123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response apperrors.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(t, response.Error.Details, "username")
}

func TestValidateJSONMiddleware_MultipleErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apperrors.ErrorHandlerMiddleware())

	router.POST("/test", ValidateJSONMiddleware(&TestValidationStruct{}), func(c *gin.Context) {
		t.Fatal("Handler should not be called")
	})

	payload := map[string]string{
		"email":    "invalid-email",
		"username": "ab", // Too short
		"phone":    "12", // Too short
		"password": "weak",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response apperrors.AppError
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", response.Code)

	// Check that all invalid fields are reported
	assert.Contains(t, response.Details, "email")
	assert.Contains(t, response.Details, "username")
	assert.Contains(t, response.Details, "phone")
	assert.Contains(t, response.Details, "password")
}
