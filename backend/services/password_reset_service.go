package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"meerkat/config"
	"meerkat/i18n"
	"meerkat/logger"
	"time"

	"github.com/resend/resend-go/v2"
)

const (
	passwordResetTokenBytes = 32
	passwordResetTTL        = time.Hour
)

// GeneratePasswordResetToken creates a secure token and its hashed representation.
func GeneratePasswordResetToken() (string, string, error) {
	raw := make([]byte, passwordResetTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}

	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := HashPasswordResetToken(token)
	return token, hash, nil
}

// HashPasswordResetToken hashes a reset token for database storage.
func HashPasswordResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// PasswordResetExpiry returns when a reset token should expire.
func PasswordResetExpiry() time.Time {
	return time.Now().Add(passwordResetTTL)
}

// SendPasswordResetEmail dispatches a reset email when Resend is configured.
// The lang parameter specifies the user's preferred language for the email content.
func SendPasswordResetEmail(email, token, lang string, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	if !cfg.UseResend {
		logger.Warn().Str("email", email).Msg("Resend disabled; password reset email not sent")
		return nil
	}

	// Default to English if language not set
	if lang == "" {
		lang = i18n.DefaultLanguage
	}

	client := resend.NewClient(cfg.ResendAPIKey)
	htmlBody := fmt.Sprintf(`<p>%s</p><p>%s</p><p><strong>%s</strong></p><p>%s</p>`,
		i18n.T(lang, "email.passwordReset.intro"),
		i18n.T(lang, "email.passwordReset.instruction"),
		token,
		i18n.T(lang, "email.passwordReset.ignore"),
	)

	params := &resend.SendEmailRequest{
		From:    cfg.ResendFromEmail,
		To:      []string{email},
		Subject: i18n.T(lang, "email.passwordReset.subject"),
		Html:    htmlBody,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return err
	}

	logger.Info().Str("email_id", sent.Id).Str("email", email).Str("language", lang).Msg("Password reset email sent")
	return nil
}
