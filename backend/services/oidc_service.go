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

// ExchangeAndVerify exchanges an authorization code for tokens and verifies the ID token, OAuth2 token is returned alongside
func (p *OIDCProvider) ExchangeAndVerify(ctx context.Context, code string) (*oidc.IDToken, *oauth2.Token, error) {
	token, err := p.oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, nil, errors.New("missing id_token in token response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify id_token: %w", err)
	}

	return idToken, token, nil
}

// OIDCClaims holds the normalized user identity from an ID token.
type OIDCClaims struct {
	Subject           string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	Provider          string
}

// standard claims we care about, shared by the ID token and userinfo response.
type profileClaims struct {
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

// ExtractClaims parses standard claims out of a verified ID token.
func ExtractClaims(idToken *oidc.IDToken, providerURL string) (*OIDCClaims, error) {
	var raw profileClaims
	if err := idToken.Claims(&raw); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	return &OIDCClaims{
		Subject:           idToken.Subject,
		Email:             raw.Email,
		EmailVerified:     raw.EmailVerified,
		Name:              raw.Name,
		PreferredUsername: raw.PreferredUsername,
		Provider:          providerURL,
	}, nil
}

// NeedsUserInfo reports whether the ID token lacked the identity claims we need to
// provision an account, meaning the userinfo endpoint should be queried.
func (c *OIDCClaims) NeedsUserInfo() bool {
	return c.Email == "" || (c.Name == "" && c.PreferredUsername == "")
}

// queries the provider's userinfo endpoint and fills in any identity claims missing from the ID token. Providers such as Authelia only return `sub` in the
func (p *OIDCProvider) EnrichFromUserInfo(ctx context.Context, token *oauth2.Token, claims *OIDCClaims) error {
	userInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return fmt.Errorf("failed to fetch userinfo: %w", err)
	}

	// OIDC Core 5.3.2: the userinfo subject MUST match the ID token subject.
	if userInfo.Subject != claims.Subject {
		return fmt.Errorf("userinfo subject %q does not match id_token subject %q", userInfo.Subject, claims.Subject)
	}

	var raw profileClaims
	if err := userInfo.Claims(&raw); err != nil {
		return fmt.Errorf("failed to parse userinfo claims: %w", err)
	}

	if claims.Email == "" && raw.Email != "" {
		claims.Email = raw.Email
		claims.EmailVerified = raw.EmailVerified
	}
	if claims.Name == "" {
		claims.Name = raw.Name
	}
	if claims.PreferredUsername == "" {
		claims.PreferredUsername = raw.PreferredUsername
	}

	return nil
}

// FindOrProvisionUser finds an existing user by OIDC subject/email, or creates one
// when auto-provisioning is enabled. Returns ErrOIDCUserNotFound if the user cannot
// be found and auto-provisioning is disabled.
func FindOrProvisionUser(db *gorm.DB, claims *OIDCClaims, cfg *config.Config) (*models.User, error) {
	var user models.User

	// 1. Look up by oidc_subject + oidc_provider (fastest path on subsequent logins)
	err := db.Where("oidc_subject = ? AND oidc_provider = ?", claims.Subject, claims.Provider).First(&user).Error
	if err == nil {
		// Backfill the email for accounts provisioned before the provider supplied one
		// (e.g. when the ID token carried no email claim). A failure here must not
		// block the login, so it is only best-effort.
		if user.Email == "" && claims.Email != "" {
			user.Email = strings.ToLower(claims.Email)
			if saveErr := db.Save(&user).Error; saveErr != nil {
				user.Email = ""
			}
		}
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

// deriveUsername builds a clean lowercase username from the preferred_username claim,
// the email local-part, or the display name, in that order.
func deriveUsername(claims *OIDCClaims) string {
	// preferred_username is a UPN on some providers (Entra ID), so treat it like an
	// address and keep the local part — usernames may not contain "@".
	if cleaned := sanitizeUsername(localPart(claims.PreferredUsername)); cleaned != "" {
		return cleaned
	}
	if cleaned := sanitizeUsername(localPart(claims.Email)); cleaned != "" {
		return cleaned
	}
	if cleaned := sanitizeUsername(claims.Name); cleaned != "" {
		return cleaned
	}
	return "oidc_user"
}

// localPart returns everything before the first "@", or the input unchanged if there is none.
func localPart(s string) string {
	if i := strings.Index(s, "@"); i >= 0 {
		return s[:i]
	}
	return s
}

// sanitizeUsername lowercases and strips anything that is not alphanumeric, mapping
// separators to underscores.
func sanitizeUsername(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-' || r == '.' || r == ' ':
			return '_'
		default:
			return -1
		}
	}, strings.ToLower(s))
}

// ProviderName extracts a human-readable name from a provider URL
func ProviderName(providerURL string) string {
	u, err := url.Parse(providerURL)
	if err != nil || u.Host == "" {
		return "SSO"
	}
	return u.Host
}
