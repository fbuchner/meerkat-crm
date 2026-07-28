package middleware

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"mycorrhizal/config"
	"mycorrhizal/logger"
	"mycorrhizal/models"
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

		// Handle API tokens (mycorrhizal_ prefix)
		if strings.HasPrefix(tokenString, "mycorrhizal_") {
			db := c.MustGet("db").(*gorm.DB)
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))
			var apiToken models.ApiToken
			// A NULL expires_at means "no expiry" and only occurs for rows
			// predating that column; tokens minted through the API always
			// carry one.
			if err := db.Where(
				"token_hash = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)",
				hash, time.Now(),
			).First(&apiToken).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}
			c.Set("userID", apiToken.UserID)
			c.Set("isAPIToken", true)
			go func(id uint) {
				if err := db.Model(&models.ApiToken{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error; err != nil {
					logger.Logger.Warn().Err(err).Uint("api_token_id", id).Msg("Failed to update api token last_used_at")
				}
			}(apiToken.ID)
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

		// jwt.Parse always yields MapClaims, but handle the other case rather
		// than falling through to c.Next() with no userID set.
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if username, exists := claims["username"].(string); exists {
			c.Set("username", username)
		}

		userID, ok := uintClaim(claims, "user_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Tokens minted before token versioning existed carry no such claim.
		// Reject them rather than defaulting to 0, so the upgrade forces one
		// re-login instead of silently trusting unversioned tokens forever.
		tokenVersion, ok := uintClaim(claims, "token_version")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// JWTs are stateless, so the only way a password change can end an
		// existing session is to compare against server-side state.
		db := c.MustGet("db").(*gorm.DB)
		var user models.User
		if err := db.Select("token_version").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		if user.TokenVersion != tokenVersion {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired, please sign in again"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

// uintClaim reads a numeric claim as uint. JSON round-tripping makes every
// number a float64, but tokens built in-process still hold their original type,
// so both are accepted.
func uintClaim(claims jwt.MapClaims, key string) (uint, bool) {
	value, exists := claims[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case uint:
		return v, true
	default:
		return 0, false
	}
}
