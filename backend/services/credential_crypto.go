package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// credentialKey derives a stable AES-256 key from the JWT secret. Stored
// credentials become undecryptable if JWT_SECRET_KEY changes; callers must
// treat decryption failures as "credentials need to be re-entered".
func credentialKey(jwtSecret string) []byte {
	sum := sha256.Sum256([]byte("mycorrhizal-credential-encryption:" + jwtSecret))
	return sum[:]
}

// EncryptCredential encrypts a secret with AES-256-GCM for storage at rest.
// An empty plaintext encrypts to an empty string.
func EncryptCredential(jwtSecret, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(credentialKey(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptCredential reverses EncryptCredential.
func DecryptCredential(jwtSecret, encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("stored credential is corrupted")
	}

	block, err := aes.NewCipher(credentialKey(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(raw) < gcm.NonceSize() {
		return "", errors.New("stored credential is corrupted")
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("failed to decrypt stored credential (was the JWT secret changed?)")
	}

	return string(plaintext), nil
}
