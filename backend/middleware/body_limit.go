package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Default body size limits
const (
	DefaultMaxBodySize = 10 << 20 // 10 MB
	MaxJSONBodySize    = 1 << 20  // 1 MB
)

// BodySizeLimitMiddleware limits the size of request bodies. For requests
// with a Content-Length header exceeding maxBytes, the middleware aborts
// with 413 immediately. For chunked/streaming bodies without Content-Length,
// the MaxBytesReader wrapper will terminate the read and Gin's JSON binding
// will produce a parse error (not a 413, but the request is still rejected).
func BodySizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if lengthStr := c.Request.Header.Get("Content-Length"); lengthStr != "" {
			if length, err := strconv.ParseInt(lengthStr, 10, 64); err == nil && length > maxBytes {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
				return
			}
		}
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
