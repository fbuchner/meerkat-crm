package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	DBPath          string
	ReminderTime    string
	FrontendURL     string
	Port            string
	TrustedProxies  []string
	UseResend       bool
	ResendAPIKey    string
	ResendFromEmail string
	ResendToEmail   string
	JWTSecretKey    string
	JWTExpiryHours  int
	ReadTimeout     int    // HTTP server read timeout in seconds
	WriteTimeout    int    // HTTP server write timeout in seconds
	IdleTimeout     int    // HTTP server idle timeout in seconds
	ProfilePhotoDir string // Directory for storing profile photos (must be absolute path)
}

func LoadConfig() *Config {

	defaultJWTExpiry := 96
	jwtExpiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", strconv.Itoa(defaultJWTExpiry)))
	if err != nil {
		log.Println("WARN: Invalid JWT expiry set. Please provide an integer value.")
		jwtExpiryHours = defaultJWTExpiry
	}

	// Parse timeout values with defaults
	readTimeout := getIntEnv("HTTP_READ_TIMEOUT", 15)
	writeTimeout := getIntEnv("HTTP_WRITE_TIMEOUT", 15)
	idleTimeout := getIntEnv("HTTP_IDLE_TIMEOUT", 60)

	cfg := &Config{
		DBPath:          getEnv("SQLITE_DB_PATH", "meerkat.db"),
		ReminderTime:    getEnv("REMINDER_TIME", "12:00"),
		FrontendURL:     getEnv("FRONTEND_URL", "*"),
		Port:            getEnv("PORT", "8080"),
		UseResend:       true,
		ResendAPIKey:    getEnv("RESEND_API_KEY", ""),
		ResendFromEmail: getEnv("RESEND_FROM_EMAIL", ""),
		JWTSecretKey:    getEnv("JWT_SECRET_KEY", ""),
		JWTExpiryHours:  jwtExpiryHours,
		TrustedProxies:  getProxies(getEnv("TRUSTED_PROXIES", "")),
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ProfilePhotoDir: getEnv("PROFILE_PHOTO_DIR", ""),
	}

	if cfg.ResendAPIKey == "" || cfg.ResendFromEmail == "" {
		cfg.UseResend = false
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("WARN: Invalid integer value for %s: %s. Using default: %d", key, value, fallback)
			return fallback
		}
		return intValue
	}
	return fallback
}

func getProxies(proxies string) []string {
	if proxies == "" {
		return nil
	}

	proxyList := strings.Split(proxies, ",")
	for i, proxy := range proxyList {
		proxyList[i] = strings.TrimSpace(proxy) // Remove whitespaces
	}
	return proxyList
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("Configuration Error [%s]: %s", e.Field, e.Message)
}

// Validate checks if the configuration is valid and returns detailed errors if not
func (c *Config) Validate() []ValidationError {
	var errors []ValidationError

	// Validate JWT Secret Key - critical for security
	if c.JWTSecretKey == "" {
		errors = append(errors, ValidationError{
			Field:   "JWT_SECRET_KEY",
			Message: "JWT secret key is required for authentication. Set JWT_SECRET_KEY environment variable.",
		})
	} else if len(c.JWTSecretKey) < 32 {
		errors = append(errors, ValidationError{
			Field:   "JWT_SECRET_KEY",
			Message: fmt.Sprintf("JWT secret key is too short (%d characters). Must be at least 32 characters for security.", len(c.JWTSecretKey)),
		})
	}

	// Validate Database Path
	if c.DBPath == "" {
		errors = append(errors, ValidationError{
			Field:   "SQLITE_DB_PATH",
			Message: "Database path cannot be empty. Set SQLITE_DB_PATH environment variable.",
		})
	}

	// Validate Profile Photo Directory - must be set and absolute path for security
	if c.ProfilePhotoDir == "" {
		errors = append(errors, ValidationError{
			Field:   "PROFILE_PHOTO_DIR",
			Message: "Profile photo directory is required. Set PROFILE_PHOTO_DIR environment variable to an absolute path.",
		})
	} else if !filepath.IsAbs(c.ProfilePhotoDir) {
		errors = append(errors, ValidationError{
			Field:   "PROFILE_PHOTO_DIR",
			Message: fmt.Sprintf("Profile photo directory '%s' must be an absolute path for security.", c.ProfilePhotoDir),
		})
	}

	// Validate Port
	if c.Port == "" {
		errors = append(errors, ValidationError{
			Field:   "PORT",
			Message: "Server port cannot be empty. Set PORT environment variable.",
		})
	} else {
		portNum, err := strconv.Atoi(c.Port)
		if err != nil || portNum < 1 || portNum > 65535 {
			errors = append(errors, ValidationError{
				Field:   "PORT",
				Message: fmt.Sprintf("Invalid port number '%s'. Must be between 1 and 65535.", c.Port),
			})
		}
	}

	// Validate Reminder Time format (HH:MM)
	timePattern := regexp.MustCompile(`^([0-1][0-9]|2[0-3]):[0-5][0-9]$`)
	if !timePattern.MatchString(c.ReminderTime) {
		errors = append(errors, ValidationError{
			Field:   "REMINDER_TIME",
			Message: fmt.Sprintf("Invalid time format '%s'. Must be in HH:MM format (e.g., 12:00).", c.ReminderTime),
		})
	}

	// Validate Frontend URL
	if c.FrontendURL == "" {
		errors = append(errors, ValidationError{
			Field:   "FRONTEND_URL",
			Message: "Frontend URL cannot be empty. Set FRONTEND_URL environment variable (use '*' for development).",
		})
	}

	// Validate JWT Expiry Hours
	if c.JWTExpiryHours < 1 || c.JWTExpiryHours > 8760 {
		errors = append(errors, ValidationError{
			Field:   "JWT_EXPIRY_HOURS",
			Message: fmt.Sprintf("Invalid JWT expiry hours '%d'. Must be between 1 and 8760 (1 year).", c.JWTExpiryHours),
		})
	}

	// Validate HTTP Timeouts (in seconds)
	if c.ReadTimeout < 1 || c.ReadTimeout > 300 {
		errors = append(errors, ValidationError{
			Field:   "HTTP_READ_TIMEOUT",
			Message: fmt.Sprintf("Invalid read timeout '%d'. Must be between 1 and 300 seconds.", c.ReadTimeout),
		})
	}
	if c.WriteTimeout < 1 || c.WriteTimeout > 300 {
		errors = append(errors, ValidationError{
			Field:   "HTTP_WRITE_TIMEOUT",
			Message: fmt.Sprintf("Invalid write timeout '%d'. Must be between 1 and 300 seconds.", c.WriteTimeout),
		})
	}
	if c.IdleTimeout < 1 || c.IdleTimeout > 300 {
		errors = append(errors, ValidationError{
			Field:   "HTTP_IDLE_TIMEOUT",
			Message: fmt.Sprintf("Invalid idle timeout '%d'. Must be between 1 and 300 seconds.", c.IdleTimeout),
		})
	}

	// Validate Trusted Proxies format (IP addresses or CIDR notation)
	for _, proxy := range c.TrustedProxies {
		if proxy == "" {
			continue
		}
		// Check if it's a valid IP address
		if ip := net.ParseIP(proxy); ip != nil {
			continue
		}
		// Check if it's a valid CIDR
		if _, _, err := net.ParseCIDR(proxy); err == nil {
			continue
		}
		errors = append(errors, ValidationError{
			Field:   "TRUSTED_PROXIES",
			Message: fmt.Sprintf("Invalid proxy '%s'. Must be a valid IP address or CIDR notation.", proxy),
		})
	}

	// Validate Resend configuration if emails are enabled
	if c.UseResend {
		if c.ResendAPIKey == "" {
			errors = append(errors, ValidationError{
				Field:   "RESEND_API_KEY",
				Message: "Resend API key is required when email is enabled.",
			})
		}
		if c.ResendFromEmail == "" {
			errors = append(errors, ValidationError{
				Field:   "RESEND_FROM_EMAIL",
				Message: "Resend sender email is required when email is enabled.",
			})
		}
	}

	return errors
}

// ValidateOrPanic validates the configuration and panics with detailed error message if invalid
func (c *Config) ValidateOrPanic() {
	errors := c.Validate()
	if len(errors) > 0 {
		log.Println("❌ Configuration validation failed:")
		log.Println("")
		for _, err := range errors {
			log.Printf("  • %s\n", err.Error())
		}
		log.Println("")
		log.Println("Please fix the configuration errors above and restart the server.")
		log.Println("Refer to backend/.env.example for configuration examples.")
		panic("Configuration validation failed")
	}
	log.Println("✓ Configuration validated successfully")
}
