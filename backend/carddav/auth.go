package carddav

import (
	"meerkat/logger"
	"meerkat/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BasicAuthMiddleware provides HTTP Basic Authentication for CardDAV
// It supports both username and email as the login identifier
func BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="CardDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		db := c.MustGet("db").(*gorm.DB)
		var user models.User

		// Try to find user by username or email
		err := db.Where("username = ? OR email = ?", username, username).First(&user).Error
		if err != nil {
			logger.Warn().
				Str("identifier", username).
				Str("ip", c.ClientIP()).
				Msg("CardDAV auth failed: user not found")
			c.Header("WWW-Authenticate", `Basic realm="CardDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Validate password
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			logger.Warn().
				Str("identifier", username).
				Str("ip", c.ClientIP()).
				Uint("user_id", user.ID).
				Msg("CardDAV auth failed: invalid password")
			c.Header("WWW-Authenticate", `Basic realm="CardDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Set user info in context for downstream handlers
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
		c.Set("user", &user)

		c.Next()
	}
}
