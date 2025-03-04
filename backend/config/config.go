package config

import (
	"os"
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
}

func LoadConfig() *Config {
	cfg := &Config{
		DBPath:             getEnv("SQLITE_DB_PATH", "perema.db"),
		ReminderTime:       getEnv("REMINDER_TIME", "08:00"),
		FrontendURL:        getEnv("FRONTEND_URL", "*"),
		Port:               getEnv("PORT", "8080"),
		UseSendgrid:        true,
		SendgridAPIKey:     getEnv("SENDGRID_API_KEY", ""),
		SendgridTemplateID: getEnv("SENDGRID_BIRTHDAY_TEMPLATE_ID", ""),
		SendgridToEmail:    getEnv("SENDGRID_TO_EMAIL", ""),
		JWTSecretKey:       getEnv("JWT_SECRET_KEY", ""),
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
	return strings.Split(proxies, ",")
}
