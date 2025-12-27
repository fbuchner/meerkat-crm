package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Default body size limits
const (
	DefaultMaxBodySize = 10 << 20 // 10 MB
	MaxJSONBodySize    = 1 << 20  // 1 MB
)

// BodySizeLimitMiddleware limits the size of request bodies to prevent DoS attacks.
func BodySizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Wrap the request body with a size limiter
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func JSONBodySizeLimitMiddleware() gin.HandlerFunc {
	return BodySizeLimitMiddleware(MaxJSONBodySize)
}

func DefaultBodySizeLimitMiddleware() gin.HandlerFunc {
	return BodySizeLimitMiddleware(DefaultMaxBodySize)
}
