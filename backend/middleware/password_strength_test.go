package middleware

import (
	"math"
	"testing"
)

func TestCalculatePasswordEntropy(t *testing.T) {
	tests := []struct {
		name            string
		password        string
		expectedEntropy float64
		tolerance       float64
	}{
		{
			name:            "empty password",
			password:        "",
			expectedEntropy: 0,
			tolerance:       0,
		},
		{
			name:            "lowercase only - 8 chars",
			password:        "password",
			expectedEntropy: 8 * math.Log2(26), // ~37.6 bits
			tolerance:       0.1,
		},
		{
			name:            "lowercase only - 20 chars (passphrase)",
			password:        "correcthorsebattery",
			expectedEntropy: 19 * math.Log2(26), // ~89.4 bits
			tolerance:       0.1,
		},
		{
			name:            "mixed case - 12 chars",
			password:        "MyPassword12",
			expectedEntropy: 12 * math.Log2(62), // ~71.4 bits
			tolerance:       0.1,
		},
		{
			name:            "all character types - 10 chars",
			password:        "MyP@ssw0rd",
			expectedEntropy: 10 * math.Log2(94), // ~65.5 bits
			tolerance:       0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := CalculatePasswordEntropy(tt.password)
			diff := math.Abs(entropy - tt.expectedEntropy)
			if diff > tt.tolerance {
				t.Errorf("CalculatePasswordEntropy(%q) = %.2f, want %.2f (tolerance: %.2f)",
					tt.password, entropy, tt.expectedEntropy, tt.tolerance)
			}
		})
	}
}

func TestDetermineCharacterSetSize(t *testing.T) {
	tests := []struct {
		name            string
		password        string
		expectedCharSet int
	}{
		{
			name:            "lowercase only",
			password:        "hello",
			expectedCharSet: 26,
		},
		{
			name:            "uppercase only",
			password:        "HELLO",
			expectedCharSet: 26,
		},
		{
			name:            "mixed case",
			password:        "HelloWorld",
			expectedCharSet: 52, // 26 + 26
		},
		{
			name:            "digits only",
			password:        "12345",
			expectedCharSet: 10,
		},
		{
			name:            "letters and digits",
			password:        "hello123",
			expectedCharSet: 36, // 26 + 10
		},
		{
			name:            "all types",
			password:        "Hello123!@#",
			expectedCharSet: 94, // 26 + 26 + 10 + 32
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charSet := determineCharacterSetSize(tt.password)
			if charSet != tt.expectedCharSet {
				t.Errorf("determineCharacterSetSize(%q) = %d, want %d",
					tt.password, charSet, tt.expectedCharSet)
			}
		})
	}
}

func TestEvaluatePasswordStrength(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedValid bool
		expectedScore int
		minEntropy    float64
	}{
		{
			name:          "very weak - short and simple",
			password:      "pass",
			expectedValid: false,
			expectedScore: 0,
			minEntropy:    0,
		},
		{
			name:          "weak - common password",
			password:      "password",
			expectedValid: false,
			expectedScore: 2, // 37.6 bits
			minEntropy:    30,
		},
		{
			name:          "fair - short complex",
			password:      "P@ssw0rd",
			expectedValid: true, // 52.4 bits (meets 50 bit threshold)
			expectedScore: 3,
			minEntropy:    50,
		},
		{
			name:          "good - meets minimum well (50+ bits)",
			password:      "MySecureP@ss123",
			expectedValid: true,
			expectedScore: 4, // ~90 bits
			minEntropy:    50,
		},
		{
			name:          "very strong - long passphrase lowercase",
			password:      "correcthorsebatterystaple",
			expectedValid: true,
			expectedScore: 4,
			minEntropy:    100,
		},
		{
			name:          "very strong - long passphrase mixed",
			password:      "CorrectHorseBatteryStaple",
			expectedValid: true,
			expectedScore: 4,
			minEntropy:    120,
		},
		{
			name:          "strong - medium length high complexity",
			password:      "MyP@ssw0rd2024!Secure",
			expectedValid: true,
			expectedScore: 4,
			minEntropy:    110,
		},
		{
			name:          "very strong - 16 char lowercase passphrase",
			password:      "mylongpassphrase",
			expectedValid: true,
			expectedScore: 4, // ~75 bits
			minEntropy:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strength := EvaluatePasswordStrength(tt.password)

			if strength.IsValid != tt.expectedValid {
				t.Errorf("EvaluatePasswordStrength(%q).IsValid = %v, want %v",
					tt.password, strength.IsValid, tt.expectedValid)
			}

			if strength.Score != tt.expectedScore {
				t.Errorf("EvaluatePasswordStrength(%q).Score = %d, want %d",
					tt.password, strength.Score, tt.expectedScore)
			}

			if strength.Entropy < tt.minEntropy {
				t.Errorf("EvaluatePasswordStrength(%q).Entropy = %.2f, want at least %.2f",
					tt.password, strength.Entropy, tt.minEntropy)
			}
		})
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{
			name:      "valid - long lowercase passphrase",
			password:  "correcthorsebatterystaple",
			wantError: false,
		},
		{
			name:      "valid - medium mixed case",
			password:  "MySecurePassword2024",
			wantError: false,
		},
		{
			name:      "invalid - too short",
			password:  "short",
			wantError: true,
		},
		{
			name:      "invalid - common password",
			password:  "password123",
			wantError: true, // In common password list
		},
		{
			name:      "valid - above entropy threshold",
			password:  "Pass123!Secure",
			wantError: false, // 14 chars, all types = ~84 bits
		},
		{
			name:      "valid - 16 chars lowercase",
			password:  "thisisalongpassw",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			hasError := err != nil

			if hasError != tt.wantError {
				t.Errorf("ValidatePasswordStrength(%q) error = %v, wantError %v",
					tt.password, err, tt.wantError)
			}
		})
	}
}

func TestIsCommonPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		isCommon bool
	}{
		{
			name:     "common - password",
			password: "password",
			isCommon: true,
		},
		{
			name:     "common - 123456",
			password: "123456",
			isCommon: true,
		},
		{
			name:     "common - qwerty",
			password: "qwerty",
			isCommon: true,
		},
		{
			name:     "common - password with capitals",
			password: "Password",
			isCommon: true,
		},
		{
			name:     "common - password with numbers",
			password: "password123",
			isCommon: true,
		},
		{
			name:     "not common - unique password",
			password: "MyUniqueP@ssphrase2024",
			isCommon: false,
		},
		{
			name:     "not common - random string",
			password: "xjK9mP2nQ8rL4vW",
			isCommon: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCommonPassword(tt.password)
			if result != tt.isCommon {
				t.Errorf("IsCommonPassword(%q) = %v, want %v",
					tt.password, result, tt.isCommon)
			}
		})
	}
}

// Test that demonstrates entropy advantages of passphrases
func TestPassphraseVsComplexPassword(t *testing.T) {
	// Short complex password
	shortComplex := "P@ssw0rd!"
	shortStrength := EvaluatePasswordStrength(shortComplex)

	// Long simple passphrase
	longSimple := "correct horse battery staple"
	longStrength := EvaluatePasswordStrength(longSimple)

	t.Logf("Short complex (%s): %.2f bits, score %d, valid: %v",
		shortComplex, shortStrength.Entropy, shortStrength.Score, shortStrength.IsValid)
	t.Logf("Long simple (%s): %.2f bits, score %d, valid: %v",
		longSimple, longStrength.Entropy, longStrength.Score, longStrength.IsValid)

	// The passphrase should have higher entropy
	if longStrength.Entropy <= shortStrength.Entropy {
		t.Errorf("Passphrase should have higher entropy than short complex password")
	}

	// The passphrase should be valid
	if !longStrength.IsValid {
		t.Errorf("Long passphrase should be valid")
	}
}

func TestPasswordEntropyExamples(t *testing.T) {
	examples := []struct {
		password    string
		description string
	}{
		{"password", "Common weak password"},
		{"P@ssw0rd", "Short complex (8 chars, all types)"},
		{"MySecurePassword", "Medium mixed case"},
		{"MySecurePassword2024", "Medium mixed case with numbers"},
		{"correct horse battery", "Passphrase (3 words)"},
		{"correcthorsebatterystaple", "Passphrase (4 words, no spaces)"},
		{"CorrectHorseBatteryStaple", "Passphrase (mixed case)"},
		{"i love my cat very much today", "Long natural sentence"},
	}

	t.Log("\n=== Password Entropy Comparison ===")
	for _, ex := range examples {
		strength := EvaluatePasswordStrength(ex.password)
		t.Logf("%-35s | Length: %2d | Entropy: %6.2f bits | Score: %d | Valid: %v | %s",
			ex.password,
			len(ex.password),
			strength.Entropy,
			strength.Score,
			strength.IsValid,
			ex.description,
		)
	}
}
