package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"meerkat/config"
	"meerkat/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

var ErrOIDCUserNotFound = errors.New("no account found for OIDC identity")

// OIDCProvider holds the initialized OIDC provider and OAuth2 config.
type OIDCProvider struct {
	provider  *oidc.Provider
	oauth2Cfg oauth2.Config
	verifier  *oidc.IDTokenVerifier
}

// InitOIDCProvider fetches the OIDC discovery document and builds the OAuth2 config.
func InitOIDCProvider(ctx context.Context, cfg *config.Config) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, cfg.OIDC.ProviderURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	oauth2Cfg := oauth2.Config{
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.OIDC.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.OIDC.ClientID})

	return &OIDCProvider{
		provider:  provider,
		oauth2Cfg: oauth2Cfg,
		verifier:  verifier,
	}, nil
}

// GenerateStateToken returns a 32-byte random hex string for CSRF protection.
func GenerateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// BuildAuthURL constructs the provider's authorization URL with the given state and nonce.
func (p *OIDCProvider) BuildAuthURL(state, nonce string) string {
	return p.oauth2Cfg.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce))
}

// ExchangeAndVerify exchanges an authorization code for tokens and verifies the ID token.
func (p *OIDCProvider) ExchangeAndVerify(ctx context.Context, code string) (*oidc.IDToken, error) {
	token, err := p.oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("missing id_token in token response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify id_token: %w", err)
	}

	return idToken, nil
}

// OIDCClaims holds the normalized user identity from an ID token.
type OIDCClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Provider      string
}

// ExtractClaims parses standard claims out of a verified ID token.
func ExtractClaims(idToken *oidc.IDToken, providerURL string) (*OIDCClaims, error) {
	var raw struct {
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&raw); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	return &OIDCClaims{
		Subject:       idToken.Subject,
		Email:         raw.Email,
		EmailVerified: raw.EmailVerified,
		Name:          raw.Name,
		Provider:      providerURL,
	}, nil
}

// FindOrProvisionUser finds an existing user by OIDC subject/email, or creates one
// when auto-provisioning is enabled. Returns ErrOIDCUserNotFound if the user cannot
// be found and auto-provisioning is disabled.
func FindOrProvisionUser(db *gorm.DB, claims *OIDCClaims, cfg *config.Config) (*models.User, error) {
	var user models.User

	// 1. Look up by oidc_subject + oidc_provider (fastest path on subsequent logins)
	err := db.Where("oidc_subject = ? AND oidc_provider = ?", claims.Subject, claims.Provider).First(&user).Error
	if err == nil {
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("database error looking up OIDC subject: %w", err)
	}

	// 2. Match by email and link the OIDC identity to the existing account.
	// Require email_verified to prevent account takeover via unverified email claims,
	// unless OIDC_TRUST_EMAIL is set (safe for self-hosted trusted providers).
	if claims.Email != "" && (claims.EmailVerified || cfg.OIDC.TrustEmail) {
		err = db.Where("email = ?", strings.ToLower(claims.Email)).First(&user).Error
		if err == nil {
			user.OIDCSubject = &claims.Subject
			user.OIDCProvider = &claims.Provider
			if saveErr := db.Save(&user).Error; saveErr != nil {
				return nil, fmt.Errorf("failed to link OIDC identity to existing account: %w", saveErr)
			}
			return &user, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("database error looking up email: %w", err)
		}
	}

	// 3. Auto-provision a new user if enabled
	if !cfg.OIDC.AllowAutoProvision {
		return nil, ErrOIDCUserNotFound
	}

	username := deriveUsername(claims)
	base := username
	for i := 1; i <= 100; i++ {
		var count int64
		db.Model(&models.User{}).Where("username = ?", username).Count(&count)
		if count == 0 {
			break
		}
		if i == 100 {
			return nil, fmt.Errorf("could not derive a unique username for OIDC user")
		}
		username = fmt.Sprintf("%s%d", base, i)
	}

	email := strings.ToLower(claims.Email)
	newUser := models.User{
		Username:     username,
		Password:     "", // OIDC-only accounts have no password
		Email:        email,
		OIDCSubject:  &claims.Subject,
		OIDCProvider: &claims.Provider,
	}

	if err := db.Create(&newUser).Error; err != nil {
		return nil, fmt.Errorf("failed to create OIDC user: %w", err)
	}

	return &newUser, nil
}

// deriveUsername builds a clean lowercase username from email local-part or display name.
func deriveUsername(claims *OIDCClaims) string {
	if claims.Email != "" {
		parts := strings.Split(claims.Email, "@")
		if len(parts) > 0 && parts[0] != "" {
			cleaned := strings.Map(func(r rune) rune {
				switch {
				case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
					return r
				case r == '_' || r == '-' || r == '.':
					return '_'
				default:
					return -1
				}
			}, strings.ToLower(parts[0]))
			if cleaned != "" {
				return cleaned
			}
		}
	}
	if claims.Name != "" {
		return strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
				return r
			}
			return '_'
		}, strings.ToLower(claims.Name))
	}
	return "oidc_user"
}

// ProviderName extracts a human-readable name from a provider URL (e.g. "accounts.google.com" → "accounts.google.com").
func ProviderName(providerURL string) string {
	u, err := url.Parse(providerURL)
	if err != nil || u.Host == "" {
		return "SSO"
	}
	return u.Host
}
