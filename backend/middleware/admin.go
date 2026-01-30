package middleware

import (
	"net/http"

	"meerkat/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminMiddleware checks if the authenticated user has admin privileges.
// Must be used AFTER AuthMiddleware which sets "userID" in context.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDValue, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		userID, ok := userIDValue.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}

		db := c.MustGet("db").(*gorm.DB)

		var user models.User
		if err := db.Select("is_admin").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if !user.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
