package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	e := &AppError{Message: "test message"}
	if e.Error() != "test message" {
		t.Errorf("Error() = %q, want %q", e.Error(), "test message")
	}

	e2 := &AppError{Message: "wrapped", Err: errors.New("inner")}
	if e2.Error() != "wrapped: inner" {
		t.Errorf("Error() = %q, want %q", e2.Error(), "wrapped: inner")
	}
}

func TestAppError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	e := &AppError{Message: "test", Err: inner}
	if !errors.Is(e, inner) {
		t.Error("Unwrap() should make errors.Is work")
	}
}

func TestAppError_WithDetails(t *testing.T) {
	e := NewError("TEST", "test", 400)
	e.WithDetails("key1", "val1").WithDetails("key2", 42)
	if e.Details["key1"] != "val1" {
		t.Errorf("Details['key1'] = %v, want 'val1'", e.Details["key1"])
	}
	if e.Details["key2"] != 42 {
		t.Errorf("Details['key2'] = %v, want 42", e.Details["key2"])
	}
	// nil Details map should be created on first call
	e2 := &AppError{Message: "no details"}
	e2.WithDetails("key", "val")
	if e2.Details["key"] != "val" {
		t.Error("WithDetails should create Details map if nil")
	}
}

func TestAppError_WithError(t *testing.T) {
	inner := errors.New("inner")
	e := NewError("TEST", "test", 400).WithError(inner)
	if e.Err != inner {
		t.Error("WithError should set Err field")
	}
}

func TestHTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name string
		e    *AppError
		want int
	}{
		{"Unauthorized", ErrUnauthorized("denied"), http.StatusUnauthorized},
		{"InvalidCredentials", ErrInvalidCredentials(), http.StatusUnauthorized},
		{"TokenExpired", ErrTokenExpired(), http.StatusUnauthorized},
		{"TokenInvalid", ErrTokenInvalid(), http.StatusUnauthorized},
		{"Forbidden", ErrForbidden("no"), http.StatusForbidden},
		{"NotFound", ErrNotFound("User"), http.StatusNotFound},
		{"AlreadyExists", ErrAlreadyExists("Circle"), http.StatusConflict},
		{"Conflict", ErrConflict("dup"), http.StatusConflict},
		{"Validation", ErrValidation("bad"), http.StatusBadRequest},
		{"InvalidInput", ErrInvalidInput("email", "invalid"), http.StatusBadRequest},
		{"MissingField", ErrMissingField("name"), http.StatusBadRequest},
		{"RateLimitExceeded", ErrRateLimitExceeded(), http.StatusTooManyRequests},
		{"Internal", ErrInternal("oops"), http.StatusInternalServerError},
		{"Database", ErrDatabase("save"), http.StatusInternalServerError},
		{"External", ErrExternal("stripe", "fail"), http.StatusServiceUnavailable},
		{"BusinessLogic", ErrBusinessLogic("nope"), http.StatusUnprocessableEntity},
		{"OperationFailed", ErrOperationFailed("sync", "timeout"), http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.e.HTTPStatus != tt.want {
				t.Errorf("%s: HTTPStatus = %d, want %d", tt.name, tt.e.HTTPStatus, tt.want)
			}
		})
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		e    *AppError
		want string
	}{
		{ErrUnauthorized("x"), ErrCodeUnauthorized},
		{ErrNotFound("X"), ErrCodeNotFound},
		{ErrValidation("x"), ErrCodeValidation},
		{ErrDatabase("x"), ErrCodeDatabase},
	}
	for _, tt := range tests {
		if tt.e.Code != tt.want {
			t.Errorf("Code = %q, want %q", tt.e.Code, tt.want)
		}
	}
}

func TestNewError(t *testing.T) {
	e := NewError("CUSTOM", "custom message", 418)
	if e.Code != "CUSTOM" {
		t.Errorf("Code = %q, want CUSTOM", e.Code)
	}
	if e.HTTPStatus != 418 {
		t.Errorf("HTTPStatus = %d, want 418", e.HTTPStatus)
	}
}

func TestErrNotFound_WithDetails(t *testing.T) {
	e := ErrNotFound("Contact").WithDetails("id", "abc")
	if e.Details["id"] != "abc" {
		t.Error("ErrNotFound should support Details chaining")
	}
}
