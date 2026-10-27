package middleware

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"meerkat/config"
	"meerkat/logger"
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
		fromCookie := false

		// First, try to get token from httpOnly cookie
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenString = cookie
			fromCookie = true
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
			if err := db.Where("token_hash = ? AND revoked_at IS NULL", hash).First(&apiToken).Error; err != nil {
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

		if fromCookie {
			refreshCookie(c, cfg, token.Claims.(jwt.MapClaims))
		}

		c.Next()
	}
}

// refreshCookie re-issues the auth cookie once less than half of its lifetime is left,
// so active users stay logged in (sliding session).
func refreshCookie(c *gin.Context, cfg *config.Config, claims jwt.MapClaims) {
	exp, ok := claims["exp"].(float64)
	authTime, hasAuthTime := claims["auth_time"].(float64)
	// Tokens without auth_time (issued before the cap existed) just run out; the cap is off when unset.
	if !ok || !hasAuthTime || cfg.JWTMaxSessionHours <= 0 {
		return
	}
	lifetime := time.Duration(cfg.JWTExpiryHours) * time.Hour
	if time.Until(time.Unix(int64(exp), 0)) > lifetime/2 {
		return
	}
	userID, _ := c.Get("userID")
	id, _ := userID.(uint)
	username, _ := claims["username"].(string)
	user := models.User{Username: username}
	user.ID = id
	tokenString, newExp, err := signToken(user, cfg, time.Unix(int64(authTime), 0))
	if err != nil {
		logger.Logger.Warn().Err(err).Uint("user_id", id).Msg("Failed to refresh auth token")
		return
	}
	// At the session cap the new expiry stops growing; keep the current cookie instead of re-issuing it.
	if newExp.Unix() <= int64(exp) {
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", tokenString, int(time.Until(newExp).Seconds()), "/", cfg.CookieDomain, cfg.CookieSecure, true)
}

// GenerateToken signs a JWT for the user. Lives here (not in services) so the auth middleware can refresh tokens without an import cycle.
func GenerateToken(user models.User, cfg *config.Config) (string, error) {
	tokenString, _, err := signToken(user, cfg, time.Now())
	return tokenString, err
}

// signToken signs a JWT whose expiry is capped at authTime (the original login) + JWT_MAX_SESSION_HOURS,
// so refreshes cannot extend a session forever. authTime is carried through refreshes as the auth_time claim.
func signToken(user models.User, cfg *config.Config, authTime time.Time) (string, time.Time, error) {
	if cfg.JWTSecretKey == "" {
		return "", time.Time{}, errors.New("JWT secret key is empty")
	}
	if cfg.JWTExpiryHours <= 0 {
		return "", time.Time{}, errors.New("JWT expiry hours is invalid")
	}

	exp := time.Now().Add(time.Hour * time.Duration(cfg.JWTExpiryHours))
	if cfg.JWTMaxSessionHours > 0 {
		if limit := authTime.Add(time.Hour * time.Duration(cfg.JWTMaxSessionHours)); limit.Before(exp) {
			exp = limit
		}
	}

	// Note: is_admin is intentionally NOT included in the JWT (AdminMiddleware handles this)
	claims := jwt.MapClaims{
		"authorized": true,
		"username":   user.Username,
		"user_id":    user.ID,
		"auth_time":  authTime.Unix(),
		"exp":        exp.Unix(),
	}

	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSecretKey))
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenString, exp, nil
}
