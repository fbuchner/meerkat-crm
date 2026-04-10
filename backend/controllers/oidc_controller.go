package controllers

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"meerkat/config"
	"meerkat/logger"
	"meerkat/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// returns whether OIDC is enabled and a provider name hint.
func OIDCConfigHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := gin.H{"enabled": cfg.OIDC.Enabled}
		if cfg.OIDC.Enabled {
			resp["provider_name"] = services.ProviderName(cfg.OIDC.ProviderURL)
		}
		c.JSON(http.StatusOK, resp)
	}
}

// generates a random state and nonce, stores in cookies, then redirects the browser
func OIDCLoginHandler(provider *services.OIDCProvider, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		state, err := services.GenerateStateToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
			return
		}
		nonce, err := services.GenerateStateToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
			return
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("oidc_state", state, 600, "/api/v1/auth/oidc/callback", cfg.CookieDomain, cfg.CookieSecure, true)
		c.SetCookie("oidc_nonce", nonce, 600, "/api/v1/auth/oidc/callback", cfg.CookieDomain, cfg.CookieSecure, true)

		c.Redirect(http.StatusFound, provider.BuildAuthURL(state, nonce))
	}
}

// handles the provider redirect (validates state, exchanges code, finds/creates user, sets the auth cookie, redirects to /).
func OIDCCallbackHandler(provider *services.OIDCProvider, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logger.FromContext(c)

		// Provider-side errors (e.g. user denied consent)
		if errParam := c.Query("error"); errParam != "" {
			c.Redirect(http.StatusFound, "/login?error=oidc_denied")
			return
		}

		// Retrieve and immediately clear the state and nonce cookies.
		stateCookie, err := c.Cookie("oidc_state")
		nonceCookie, nonceErr := c.Cookie("oidc_nonce")
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("oidc_state", "", -1, "/api/v1/auth/oidc/callback", cfg.CookieDomain, cfg.CookieSecure, true)
		c.SetCookie("oidc_nonce", "", -1, "/api/v1/auth/oidc/callback", cfg.CookieDomain, cfg.CookieSecure, true)

		if err != nil || stateCookie == "" {
			log.Warn().Msg("OIDC callback: missing state cookie")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}
		if nonceErr != nil || nonceCookie == "" {
			log.Warn().Msg("OIDC callback: missing nonce cookie")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}

		stateParam := c.Query("state")
		if subtle.ConstantTimeCompare([]byte(stateCookie), []byte(stateParam)) != 1 {
			log.Warn().Msg("OIDC callback: state mismatch")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}

		code := c.Query("code")
		if code == "" {
			log.Warn().Msg("OIDC callback: missing code")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}

		idToken, err := provider.ExchangeAndVerify(c.Request.Context(), code)
		if err != nil {
			log.Error().Err(err).Msg("OIDC token exchange/verification failed")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}

		if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonceCookie)) != 1 {
			log.Warn().Msg("OIDC callback: nonce mismatch")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}

		claims, err := services.ExtractClaims(idToken, cfg.OIDC.ProviderURL)
		if err != nil {
			log.Error().Err(err).Msg("OIDC: failed to extract claims")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}

		db := c.MustGet("db").(*gorm.DB)

		user, err := services.FindOrProvisionUser(db, claims, cfg)
		if err != nil {
			if errors.Is(err, services.ErrOIDCUserNotFound) {
				c.Redirect(http.StatusFound, "/login?error=oidc_no_account")
				return
			}
			log.Error().Err(err).Msg("OIDC: failed to find or provision user")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}

		tokenString, err := services.GenerateToken(*user, cfg)
		if err != nil {
			log.Error().Err(err).Uint("user_id", user.ID).Msg("OIDC: failed to generate JWT")
			c.Redirect(http.StatusFound, "/login?error=oidc_error")
			return
		}

		maxAge := cfg.JWTExpiryHours * 3600
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", tokenString, maxAge, "/", cfg.CookieDomain, cfg.CookieSecure, true)

		c.Redirect(http.StatusFound, "/")
	}
}
