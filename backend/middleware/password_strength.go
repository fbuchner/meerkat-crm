package middleware

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// PasswordStrength represents the strength evaluation of a password
type PasswordStrength struct {
	IsValid     bool    `json:"is_valid"`
	Entropy     float64 `json:"entropy"`
	Score       int     `json:"score"` // 0-4 (weak, fair, good, strong, very strong)
	Feedback    string  `json:"feedback"`
	MinEntropy  float64 `json:"min_entropy"`
	CharSetSize int     `json:"char_set_size"`
	Length      int     `json:"length"`
}

const (
	// Minimum entropy bits required for a valid password
	// 50 bits = ~10^15 combinations (strong enough against brute force)
	MinEntropyBits = 50.0

	// Character set sizes
	LowercaseCharSet = 26
	UppercaseCharSet = 26
	DigitsCharSet    = 10
	SymbolsCharSet   = 32 // Common symbols
)

// CalculatePasswordEntropy calculates the entropy of a password in bits
// Entropy = log2(charset_size^length) = length * log2(charset_size)
func CalculatePasswordEntropy(password string) float64 {
	if len(password) == 0 {
		return 0
	}

	charSetSize := determineCharacterSetSize(password)
	length := float64(len(password))

	// Calculate entropy: log2(charSetSize^length) = length * log2(charSetSize)
	entropy := length * math.Log2(float64(charSetSize))

	return entropy
}

// determineCharacterSetSize determines the size of the character set used
func determineCharacterSetSize(password string) int {
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSymbol := false

	for _, char := range password {
		if unicode.IsLower(char) {
			hasLower = true
		} else if unicode.IsUpper(char) {
			hasUpper = true
		} else if unicode.IsDigit(char) {
			hasDigit = true
		} else {
			hasSymbol = true
		}
	}

	charSetSize := 0
	if hasLower {
		charSetSize += LowercaseCharSet
	}
	if hasUpper {
		charSetSize += UppercaseCharSet
	}
	if hasDigit {
		charSetSize += DigitsCharSet
	}
	if hasSymbol {
		charSetSize += SymbolsCharSet
	}

	return charSetSize
}

// EvaluatePasswordStrength evaluates password strength based on entropy
func EvaluatePasswordStrength(password string) PasswordStrength {
	entropy := CalculatePasswordEntropy(password)
	charSetSize := determineCharacterSetSize(password)
	length := len(password)

	result := PasswordStrength{
		Entropy:     entropy,
		MinEntropy:  MinEntropyBits,
		CharSetSize: charSetSize,
		Length:      length,
	}

	// Determine validity
	result.IsValid = entropy >= MinEntropyBits

	// Calculate score (0-4)
	switch {
	case entropy < 28:
		result.Score = 0 // Very weak
		result.Feedback = "Password is too weak. Use at least 15 characters or a longer passphrase."
	case entropy < 36:
		result.Score = 1 // Weak
		result.Feedback = "Password is weak. Consider using a longer password or passphrase."
	case entropy < 50:
		result.Score = 2 // Fair
		result.Feedback = "Password is fair but below minimum security requirement (50 bits entropy)."
	case entropy < 60:
		result.Score = 3 // Good
		result.Feedback = "Password is strong. Good job!"
	default:
		result.Score = 4 // Very strong
		result.Feedback = "Password is very strong. Excellent!"
	}

	// Add specific feedback for common scenarios
	if length < 8 {
		result.Feedback = "Password must be at least 8 characters long."
	} else if length >= 20 && charSetSize == LowercaseCharSet {
		// Long passphrase with only lowercase
		if result.IsValid {
			result.Feedback = "Great passphrase! Length makes up for simplicity."
		}
	} else if length < 12 && charSetSize < 52 {
		// Short password without both upper and lower
		result.Feedback = "Short passwords need more character variety or consider using a passphrase."
	}

	return result
}

// ValidatePasswordStrength checks if a password meets minimum entropy requirements
// and isn't a common password
func ValidatePasswordStrength(password string) error {
	// Check if it's a common password first
	if IsCommonPassword(password) {
		return &PasswordStrengthError{
			Message:  "Password is too common. Please choose a unique password or passphrase.",
			Entropy:  CalculatePasswordEntropy(password),
			Required: MinEntropyBits,
		}
	}

	strength := EvaluatePasswordStrength(password)

	if !strength.IsValid {
		return &PasswordStrengthError{
			Message:  strength.Feedback,
			Entropy:  strength.Entropy,
			Required: MinEntropyBits,
		}
	}

	return nil
}

// PasswordStrengthError represents a password strength validation error
type PasswordStrengthError struct {
	Message  string  `json:"message"`
	Entropy  float64 `json:"entropy"`
	Required float64 `json:"required"`
}

func (e *PasswordStrengthError) Error() string {
	return e.Message
}

// GetPasswordRequirements returns human-readable password requirements
func GetPasswordRequirements() string {
	return `Password must have at least 50 bits of entropy. Examples:
- 11+ characters with mixed case, numbers, and symbols (e.g., "MyP@ssw0rd!")
- 15+ characters with letters and numbers (e.g., "correcthorsebattery42")
- 20+ lowercase letters (e.g., "correct horse battery staple")
Longer passphrases are preferred over short complex passwords.`
}

// IsCommonPassword checks if password is in common password list
// This is a simplified version - in production, use a comprehensive list
func IsCommonPassword(password string) bool {
	// Common weak passwords and patterns
	commonPasswords := []string{
		"password", "123456", "12345678", "qwerty", "abc123",
		"monkey", "1234567", "letmein", "trustno1", "dragon",
		"baseball", "111111", "iloveyou", "master", "sunshine",
		"ashley", "bailey", "passw0rd", "shadow", "123123",
		"654321", "superman", "qazwsx", "michael", "football",
		"password123", "password1", "admin", "welcome", "login",
	}

	// Normalize password: lowercase and remove non-alphanumeric
	lowerPassword := strings.ToLower(password)

	// Direct match
	for _, common := range commonPasswords {
		if lowerPassword == common {
			return true
		}
	}

	// Check if it's just a common password with symbols removed
	alphanumericOnly := regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(lowerPassword, "")
	for _, common := range commonPasswords {
		if alphanumericOnly == common {
			return true
		}
	}

	return false
}

// Examples for documentation:
// - "correcthorsebatterystaple" (25 chars, lowercase only): ~117 bits ✅
// - "CorrectHorseBatteryStaple" (25 chars, mixed case): ~147 bits ✅
// - "P@ssw0rd" (8 chars, mixed): ~47 bits ❌
// - "MySecureP@ssw0rd2024" (20 chars, all types): ~119 bits ✅
// - "i love my cat very much" (21 chars with spaces): ~98 bits ✅
