package middleware

import (
	"github.com/gin-gonic/gin"
)

//	Sets common HTTP security headers on all responses.
//
// enableHSTS should only be true when the server is reached via HTTPS
// (e.g. behind a TLS-terminating proxy), otherwise browsers may refuse
// plain-HTTP access for the duration of the max-age.
func SecurityHeadersMiddleware(enableHSTS bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		if enableHSTS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}
