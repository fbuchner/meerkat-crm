package middleware

import (
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "string with null bytes",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "string with control characters",
			input:    "Hello\x01\x02World",
			expected: "HelloWorld",
		},
		{
			name:     "string with allowed whitespace",
			input:    "Hello\nWorld\t!",
			expected: "Hello\nWorld\t!",
		},
		{
			name:     "string with carriage return",
			input:    "Hello\rWorld",
			expected: "Hello\rWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		isValid bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			isValid: true,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			isValid: true,
		},
		{
			name:    "invalid email - no @",
			email:   "testexample.com",
			isValid: false,
		},
		{
			name:    "invalid email - no domain",
			email:   "test@",
			isValid: false,
		},
		{
			name:    "invalid email - no username",
			email:   "@example.com",
			isValid: false,
		},
		{
			name:    "empty email",
			email:   "",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateEmail(tt.email)
			if result != tt.isValid {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, result, tt.isValid)
			}
		})
	}
}

// TestValidateStruct_Phone tests phone validation through ValidateStruct
func TestValidateStruct_Phone(t *testing.T) {
	type TestStruct struct {
		Phone string `validate:"phone"`
	}

	tests := []struct {
		name    string
		phone   string
		isValid bool
	}{
		{
			name:    "valid 10 digit phone",
			phone:   "1234567890",
			isValid: true,
		},
		{
			name:    "valid phone with formatting",
			phone:   "+1 (234) 567-8900",
			isValid: true,
		},
		{
			name:    "valid international phone",
			phone:   "+33123456789",
			isValid: true,
		},
		{
			name:    "invalid - too short",
			phone:   "1234",
			isValid: false,
		},
		{
			name:    "invalid - too long (21+ digits)",
			phone:   "123456789012345678901", // 21 digits
			isValid: false,
		},
		{
			name:    "valid - letters stripped, enough digits remain",
			phone:   "123abc7890", // Letters stripped = 10 digits
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := TestStruct{Phone: tt.phone}
			errors := ValidateStruct(obj)
			hasErrors := len(errors) > 0
			if hasErrors == tt.isValid {
				t.Errorf("ValidateStruct with phone %q: hasErrors=%v, want isValid=%v", tt.phone, hasErrors, tt.isValid)
			}
		})
	}
}

// TestValidateStruct_Birthday tests birthday validation through ValidateStruct
func TestValidateStruct_Birthday(t *testing.T) {
	type TestStruct struct {
		Birthday string `validate:"birthday"`
	}

	tests := []struct {
		name    string
		date    string
		isValid bool
	}{
		{
			name:    "valid date - DD.MM.YYYY",
			date:    "15.01.1990",
			isValid: true,
		},
		{
			name:    "valid date without year - DD.MM.",
			date:    "15.01.",
			isValid: true, // Matches DD.MM. (year optional)
		},
		{
			name:    "valid leap year date",
			date:    "29.02.2000",
			isValid: true,
		},
		{
			name:    "invalid format - US style",
			date:    "01/15/1990",
			isValid: false,
		},
		{
			name:    "invalid format - ISO style",
			date:    "1990-01-15",
			isValid: false,
		},
		{
			name:    "invalid - not a date",
			date:    "not-a-date",
			isValid: false,
		},
		{
			name:    "empty string is valid (use omitempty or required for mandatory)",
			date:    "",
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := TestStruct{Birthday: tt.date}
			errors := ValidateStruct(obj)
			hasErrors := len(errors) > 0
			if hasErrors == tt.isValid {
				t.Errorf("ValidateStruct with birthday %q: hasErrors=%v, want isValid=%v", tt.date, hasErrors, tt.isValid)
			}
		})
	}
}

// TestValidateStruct_SafeString tests safe_string validation through ValidateStruct
func TestValidateStruct_SafeString(t *testing.T) {
	type TestStruct struct {
		Text string `validate:"safe_string"`
	}

	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{
			name:    "safe string",
			input:   "Hello World!",
			isValid: true,
		},
		{
			name:    "string with numbers and special chars",
			input:   "Test123 @#$%",
			isValid: true,
		},
		{
			name:    "SQL injection attempt - SELECT",
			input:   "'; SELECT * FROM users; --",
			isValid: false,
		},
		{
			name:    "SQL injection attempt - DROP",
			input:   "test'; DROP TABLE users; --",
			isValid: false,
		},
		{
			name:    "XSS attempt - script tag",
			input:   "<script>alert('xss')</script>",
			isValid: false,
		},
		{
			name:    "XSS attempt - javascript protocol",
			input:   "javascript:alert('xss')",
			isValid: false,
		},
		{
			name:    "SQL comment sequence",
			input:   "test -- comment",
			isValid: false,
		},
		{
			name:    "SQL block comment",
			input:   "test /* comment */",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := TestStruct{Text: tt.input}
			errors := ValidateStruct(obj)
			hasErrors := len(errors) > 0
			if hasErrors == tt.isValid {
				t.Errorf("ValidateStruct with text %q: hasErrors=%v, want isValid=%v", tt.input, hasErrors, tt.isValid)
			}
		})
	}
}
