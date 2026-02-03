package carddav

import (
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BasicAuthMiddleware provides HTTP Basic Authentication for CardDAV
// It supports both username and email as the login identifier
// Includes account-based rate limiting to prevent brute force attacks
func BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="CardDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Normalize identifier for consistent rate limiting
		identifier := strings.ToLower(username)

		// Check per-account rate limiting before attempting authentication
		accountLimiter := middleware.GetAccountRateLimiter()
		if isLocked, remainingSecs := accountLimiter.IsLocked(identifier); isLocked {
			logger.Warn().
				Str("identifier", identifier).
				Str("ip", c.ClientIP()).
				Int("retry_after", remainingSecs).
				Msg("CardDAV auth blocked: account temporarily locked")
			c.Header("WWW-Authenticate", `Basic realm="CardDAV"`)
			c.Header("Retry-After", strconv.Itoa(remainingSecs))
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		db := c.MustGet("db").(*gorm.DB)
		var user models.User

		// Try to find user by username or email
		err := db.Where("username = ? OR email = ?", identifier, identifier).First(&user).Error
		if err != nil {
			// Record failed attempt even for non-existent users to prevent enumeration
			accountLimiter.RecordFailedAttempt(identifier)
			logger.Warn().
				Str("identifier", identifier).
				Str("ip", c.ClientIP()).
				Msg("CardDAV auth failed: user not found")
			c.Header("WWW-Authenticate", `Basic realm="CardDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Validate password
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			// Record failed attempt for password mismatch
			isLocked, _ := accountLimiter.RecordFailedAttempt(identifier)
			logger.Warn().
				Str("identifier", identifier).
				Str("ip", c.ClientIP()).
				Uint("user_id", user.ID).
				Bool("now_locked", isLocked).
				Msg("CardDAV auth failed: invalid password")
			c.Header("WWW-Authenticate", `Basic realm="CardDAV"`)
			if isLocked {
				c.AbortWithStatus(http.StatusTooManyRequests)
			} else {
				c.AbortWithStatus(http.StatusUnauthorized)
			}
			return
		}

		// Successful login - clear any failed attempt tracking
		accountLimiter.RecordSuccessfulLogin(identifier)

		// Set user info in context for downstream handlers
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
		c.Set("user", &user)

		c.Next()
	}
}
