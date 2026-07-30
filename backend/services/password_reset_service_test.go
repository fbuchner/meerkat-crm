package services

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"mycorrhizal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratePasswordResetToken verifies the shape and entropy of generated
// tokens rather than exact byte values, since the token is random by design.
func TestGeneratePasswordResetToken(t *testing.T) {
	t.Run("returns non-empty token and hash with no error", func(t *testing.T) {
		token, hash, err := GeneratePasswordResetToken()
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotEmpty(t, hash)
	})

	t.Run("hash matches HashPasswordResetToken(token)", func(t *testing.T) {
		token, hash, err := GeneratePasswordResetToken()
		require.NoError(t, err)
		assert.Equal(t, HashPasswordResetToken(token), hash)
	})

	t.Run("token decodes as base64 raw URL encoding of 32 bytes", func(t *testing.T) {
		token, _, err := GeneratePasswordResetToken()
		require.NoError(t, err)

		// base64.RawURLEncoding of 32 bytes is 43 chars (no padding).
		assert.Len(t, token, 43)

		// Token should be URL-safe: no '+', '/', or '=' padding characters.
		assert.NotContains(t, token, "+")
		assert.NotContains(t, token, "/")
		assert.NotContains(t, token, "=")
	})

	t.Run("hash is a 64-character lowercase hex string (sha256)", func(t *testing.T) {
		_, hash, err := GeneratePasswordResetToken()
		require.NoError(t, err)
		assert.Len(t, hash, 64)
		assert.Regexp(t, "^[0-9a-f]{64}$", hash)
	})

	t.Run("successive calls produce distinct tokens and hashes (entropy check)", func(t *testing.T) {
		const n = 100
		tokens := make(map[string]struct{}, n)
		hashes := make(map[string]struct{}, n)

		for i := 0; i < n; i++ {
			token, hash, err := GeneratePasswordResetToken()
			require.NoError(t, err)

			_, dupToken := tokens[token]
			assert.False(t, dupToken, "token collision detected across %d generations", n)
			tokens[token] = struct{}{}

			_, dupHash := hashes[hash]
			assert.False(t, dupHash, "hash collision detected across %d generations", n)
			hashes[hash] = struct{}{}
		}

		assert.Len(t, tokens, n)
		assert.Len(t, hashes, n)
	})
}

// TestHashPasswordResetToken verifies the hashing scheme mirrors the sha256
// hex-digest pattern used elsewhere in the codebase (e.g.
// controllers/api_token_controller.go's token hashing), and that it behaves
// correctly as a deterministic, DB-lookup-friendly one-way hash.
func TestHashPasswordResetToken(t *testing.T) {
	t.Run("is deterministic for the same input", func(t *testing.T) {
		token := "sample-token-value"
		h1 := HashPasswordResetToken(token)
		h2 := HashPasswordResetToken(token)
		assert.Equal(t, h1, h2)
	})

	t.Run("different tokens produce different hashes", func(t *testing.T) {
		h1 := HashPasswordResetToken("token-a")
		h2 := HashPasswordResetToken("token-b")
		assert.NotEqual(t, h1, h2)
	})

	t.Run("matches raw sha256 hex digest of the input", func(t *testing.T) {
		token := "another-sample-token"
		sum := sha256.Sum256([]byte(token))
		expected := hex.EncodeToString(sum[:])
		assert.Equal(t, expected, HashPasswordResetToken(token))
	})

	t.Run("handles empty string input without panicking", func(t *testing.T) {
		hash := HashPasswordResetToken("")
		assert.Len(t, hash, 64)
		sum := sha256.Sum256([]byte(""))
		assert.Equal(t, hex.EncodeToString(sum[:]), hash)
	})
}

// TestPasswordResetExpiry verifies the token expiry window matches the
// configured passwordResetTTL constant (1 hour).
func TestPasswordResetExpiry(t *testing.T) {
	before := time.Now()
	expiry := PasswordResetExpiry()
	after := time.Now()

	// expiry should be ~1 hour after "now", bounded by the calls to time.Now()
	// immediately before/after the call under test.
	assert.True(t, !expiry.Before(before.Add(passwordResetTTL)), "expiry should be at least passwordResetTTL after 'before'")
	assert.True(t, !expiry.After(after.Add(passwordResetTTL)), "expiry should be at most passwordResetTTL after 'after'")

	// Sanity: the configured TTL is exactly one hour.
	assert.Equal(t, time.Hour, passwordResetTTL)

	// Expiry should be in the future relative to now.
	assert.True(t, expiry.After(time.Now().Add(-time.Second)))
}

// TestSendPasswordResetEmail exercises the branches of SendPasswordResetEmail
// that don't require a live mailer transport: nil config, no email channel
// configured (no-op success), and a configured-but-unreachable channel
// (propagated error). The package has no mailer test seam/mock (SendEmail is
// called directly, unlike reminder_service.go's sendReminderEmailFn
// indirection), so the "happy path" of an actual delivery isn't exercised
// here without reaching out over the network.
func TestSendPasswordResetEmail(t *testing.T) {
	t.Run("returns error when config is nil", func(t *testing.T) {
		err := SendPasswordResetEmail("user@example.com", "sometoken", "en", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("no-ops successfully when no email channel is configured", func(t *testing.T) {
		cfg := &config.Config{}
		err := SendPasswordResetEmail("user@example.com", "sometoken", "en", cfg)
		assert.NoError(t, err)
	})

	t.Run("defaults to English when lang is empty and no channel configured", func(t *testing.T) {
		cfg := &config.Config{}
		err := SendPasswordResetEmail("user@example.com", "sometoken", "", cfg)
		assert.NoError(t, err)
	})

	t.Run("propagates error when configured channel fails to send", func(t *testing.T) {
		// Point SMTP at a local port nothing is listening on so the send
		// fails fast (connection refused) without reaching the network.
		cfg := &config.Config{
			UseSMTP:       true,
			SMTPHost:      "127.0.0.1",
			SMTPPort:      1,
			SMTPFromEmail: "noreply@example.com",
		}

		err := SendPasswordResetEmail("user@example.com", "sometoken", "en", cfg)
		require.Error(t, err)
	})

	t.Run("unknown language falls back without erroring", func(t *testing.T) {
		cfg := &config.Config{}
		err := SendPasswordResetEmail("user@example.com", "sometoken", "xx-not-a-real-lang", cfg)
		assert.NoError(t, err)
	})
}
