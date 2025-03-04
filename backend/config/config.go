package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBPath             string
	ReminderTime       string
	FrontendURL        string
	Port               string
	TrustedProxies     []string
	UseSendgrid        bool
	SendgridToEmail    string
	SendgridTemplateID string
	SendgridAPIKey     string
	JWTSecretKey       string
	JWTExpiryHours     int
}

func LoadConfig() *Config {

	defaultJWTExpiry := 24
	jwtExpiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", strconv.Itoa(defaultJWTExpiry)))
	if err != nil {
		log.Println("WARN: Invalid JWT expiry set. Please provide an integer value.")
		jwtExpiryHours = defaultJWTExpiry
	}

	cfg := &Config{
		DBPath:             getEnv("SQLITE_DB_PATH", "perema.db"),
		ReminderTime:       getEnv("REMINDER_TIME", "12:00"),
		FrontendURL:        getEnv("FRONTEND_URL", "*"),
		Port:               getEnv("PORT", "8080"),
		UseSendgrid:        true,
		SendgridAPIKey:     getEnv("SENDGRID_API_KEY", ""),
		SendgridTemplateID: getEnv("SENDGRID_BIRTHDAY_TEMPLATE_ID", ""),
		SendgridToEmail:    getEnv("SENDGRID_TO_EMAIL", ""),
		JWTSecretKey:       getEnv("JWT_SECRET_KEY", ""),
		JWTExpiryHours:     jwtExpiryHours,
		TrustedProxies:     getProxies(getEnv("TRUSTED_PROXIES", "")),
	}

	if cfg.SendgridAPIKey == "" || cfg.SendgridTemplateID == "" || cfg.SendgridToEmail == "" {
		cfg.UseSendgrid = false
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
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
