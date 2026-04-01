package middleware

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"meerkat/config"
	"meerkat/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// First, try to get token from httpOnly cookie
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenString = cookie
		} else {
			// Fall back to Authorization header (for API clients like CardDAV)
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
				c.Abort()
				return
			}

			// Check if Authorization header is formatted properly
			if !strings.HasPrefix(authHeader, "Bearer ") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must start with Bearer"})
				c.Abort()
				return
			}

			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// Handle API tokens (meerkat_ prefix)
		if strings.HasPrefix(tokenString, "meerkat_") {
			db := c.MustGet("db").(*gorm.DB)
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))
			var apiToken models.ApiToken
			if err := db.Where("token_hash = ? AND revoked_at IS NULL AND deleted_at IS NULL", hash).First(&apiToken).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}
			c.Set("userID", apiToken.UserID)
			c.Set("username", "")
			c.Set("isAPIToken", true)
			go db.Model(&models.ApiToken{}).Where("id = ?", apiToken.ID).Update("last_used_at", time.Now())
			c.Next()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(cfg.JWTSecretKey), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
				c.Abort()
				return
			}
			if errors.Is(err, jwt.ErrTokenMalformed) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Malformed token"})
				c.Abort()
				return
			}
			if errors.Is(err, jwt.ErrSignatureInvalid) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token signature"})
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if username, exists := claims["username"].(string); exists {
				c.Set("username", username)
			}

			if idValue, exists := claims["user_id"]; exists {
				switch v := idValue.(type) {
				case float64:
					c.Set("userID", uint(v))
				case int:
					c.Set("userID", uint(v))
				case uint:
					c.Set("userID", v)
				default:
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
					c.Abort()
					return
				}
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
