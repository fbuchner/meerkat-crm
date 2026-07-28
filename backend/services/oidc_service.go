package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"mycorrhizal/config"
	"mycorrhizal/logger"
	"mycorrhizal/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

var ErrOIDCUserNotFound = errors.New("no account found for OIDC identity")

// ErrOIDCNoEmail means the provider supplied no email address in either the ID
// token or UserInfo, so no account can be provisioned. Usually a missing
// "email" scope, or a provider that serves email only from UserInfo while its
// UserInfo endpoint is unreachable.
var ErrOIDCNoEmail = errors.New("OIDC provider returned no email address")

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

// GeneratePKCEVerifier returns a fresh PKCE code verifier.
func GeneratePKCEVerifier() string {
	return oauth2.GenerateVerifier()
}

// BuildAuthURL constructs the provider's authorization URL with the given state,
// nonce and PKCE verifier. PKCE is sent even though this is a confidential
// client: OAuth 2.1 requires it for every client type, and it binds the
// authorization code to this specific login attempt so a stolen or injected
// code cannot be redeemed elsewhere.
func (p *OIDCProvider) BuildAuthURL(state, nonce, pkceVerifier string) string {
	return p.oauth2Cfg.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(pkceVerifier),
	)
}

// ExchangeAndVerify exchanges an authorization code for tokens and verifies the
// ID token. The OAuth2 token is returned alongside it because the UserInfo
// request needs its access token — see ClaimsFor.
func (p *OIDCProvider) ExchangeAndVerify(ctx context.Context, code, pkceVerifier string) (*oidc.IDToken, *oauth2.Token, error) {
	token, err := p.oauth2Cfg.Exchange(ctx, code, oauth2.VerifierOption(pkceVerifier))
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
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Provider      string
}

// ClaimsFor normalizes the caller's identity from a verified ID token, falling
// back to the UserInfo endpoint for anything the ID token left out.
//
// The fallback is not optional in practice. Nothing in OIDC requires a provider
// to put `email`/`name` in the ID token — several (Authelia among them)
// deliberately return only `sub` there and serve the rest from UserInfo. Without
// this, those providers yield an empty email, which skips account linking and
// produces users the database cannot even store more than one of
// (users.email is UNIQUE NOT NULL).
func (p *OIDCProvider) ClaimsFor(ctx context.Context, idToken *oidc.IDToken, token *oauth2.Token, providerURL string) (*OIDCClaims, error) {
	var raw struct {
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&raw); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	claims := &OIDCClaims{
		Subject:       idToken.Subject,
		Email:         raw.Email,
		EmailVerified: raw.EmailVerified,
		Name:          raw.Name,
		Provider:      providerURL,
	}

	if claims.Email == "" || claims.Name == "" {
		if err := p.enrichFromUserInfo(ctx, idToken.Subject, token, claims); err != nil {
			// Not fatal on its own: if the ID token already carried an email we
			// can still proceed. FindOrProvisionUser rejects the case where we
			// end up with nothing usable.
			logger.Warn().Err(err).Str("subject", idToken.Subject).Msg("OIDC: UserInfo lookup failed")
		}
	}

	return claims, nil
}

// enrichFromUserInfo fills blank claims from the provider's UserInfo endpoint.
func (p *OIDCProvider) enrichFromUserInfo(ctx context.Context, subject string, token *oauth2.Token, claims *OIDCClaims) error {
	if token == nil {
		return errors.New("no oauth2 token available for UserInfo request")
	}

	info, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return fmt.Errorf("userinfo request failed: %w", err)
	}

	// OIDC Core 5.3.2: the UserInfo `sub` MUST match the ID token's `sub`.
	// Skipping this check would let a provider (or anything able to swap the
	// access token) attach one identity's profile to another's session.
	if info.Subject != subject {
		return fmt.Errorf("userinfo subject %q does not match id_token subject %q", info.Subject, subject)
	}

	if claims.Email == "" && info.Email != "" {
		claims.Email = info.Email
		claims.EmailVerified = info.EmailVerified
	}

	if claims.Name == "" {
		var extra struct {
			Name              string `json:"name"`
			PreferredUsername string `json:"preferred_username"`
		}
		if err := info.Claims(&extra); err == nil {
			if extra.Name != "" {
				claims.Name = extra.Name
			} else if extra.PreferredUsername != "" {
				claims.Name = extra.PreferredUsername
			}
		}
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

	// users.email is UNIQUE NOT NULL, so provisioning without one inserts an
	// empty string that the *next* such user then collides with, locking
	// everyone after the first out. Refuse up front with a diagnosable error
	// rather than creating an account that cannot receive mail or reset its
	// password anyway.
	if claims.Email == "" {
		return nil, ErrOIDCNoEmail
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
