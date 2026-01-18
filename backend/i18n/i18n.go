package i18n

import (
	"embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

// translations holds all loaded translations indexed by language code
var translations = make(map[string]map[string]interface{})
var translationsMu sync.RWMutex
var initialized bool

// SupportedLanguages lists all supported language codes
var SupportedLanguages = []string{"en", "de"}

// DefaultLanguage is the fallback language
const DefaultLanguage = "en"

// Init loads all translation files. Call this at application startup.
func Init() error {
	translationsMu.Lock()
	defer translationsMu.Unlock()

	if initialized {
		return nil
	}

	for _, lang := range SupportedLanguages {
		data, err := localesFS.ReadFile("locales/" + lang + ".json")
		if err != nil {
			return err
		}

		var t map[string]interface{}
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
		translations[lang] = t
	}

	initialized = true
	return nil
}

// T returns the translated string for the given key and language.
// Keys use dot notation, e.g., "email.reminder.subject".
// If the key is not found, it returns the key itself.
// Optional params map can be used for {{placeholder}} substitution.
func T(lang, key string, params ...map[string]string) string {
	translationsMu.RLock()
	defer translationsMu.RUnlock()

	// Normalize language (take first part before hyphen, e.g., "en-US" -> "en")
	lang = normalizeLanguage(lang)

	// Try the requested language first
	if result := lookup(lang, key); result != "" {
		return interpolate(result, params...)
	}

	// Fall back to default language
	if lang != DefaultLanguage {
		if result := lookup(DefaultLanguage, key); result != "" {
			return interpolate(result, params...)
		}
	}

	// Return the key if nothing found
	return key
}

// normalizeLanguage normalizes the language code
func normalizeLanguage(lang string) string {
	if lang == "" {
		return DefaultLanguage
	}

	// Take first part before hyphen (e.g., "en-US" -> "en")
	lang = strings.ToLower(strings.Split(lang, "-")[0])

	// Check if supported
	for _, supported := range SupportedLanguages {
		if lang == supported {
			return lang
		}
	}

	return DefaultLanguage
}

// lookup finds a value by dot-separated key in the translations map
func lookup(lang, key string) string {
	t, ok := translations[lang]
	if !ok {
		return ""
	}

	parts := strings.Split(key, ".")
	var current interface{} = t

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = m[part]
		if !ok {
			return ""
		}
	}

	if str, ok := current.(string); ok {
		return str
	}

	return ""
}

// interpolate replaces {{placeholder}} with values from params
func interpolate(s string, params ...map[string]string) string {
	if len(params) == 0 {
		return s
	}

	result := s
	for key, value := range params[0] {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

// IsValidLanguage checks if a language code is supported
func IsValidLanguage(lang string) bool {
	normalized := normalizeLanguage(lang)
	for _, supported := range SupportedLanguages {
		if normalized == supported {
			return true
		}
	}
	return false
}
