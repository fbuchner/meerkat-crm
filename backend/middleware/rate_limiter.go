package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter manages rate limiters for different IP addresses
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit // requests per second
	b   int        // burst size
}

// NewIPRateLimiter creates a new IP-based rate limiter
// r: requests per second (e.g., 10 = 10 requests per second)
// b: burst size (e.g., 20 = allow bursts up to 20 requests)
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

// GetLimiter returns the rate limiter for the given IP address
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

// CleanupStaleEntries removes rate limiters that haven't been used recently
// This prevents memory leaks from accumulating limiters for old IPs
func (i *IPRateLimiter) CleanupStaleEntries() {
	i.mu.Lock()
	defer i.mu.Unlock()

	for ip := range i.ips {
		// If the limiter has full tokens (hasn't been used), remove it
		if i.ips[ip].Tokens() == float64(i.b) {
			delete(i.ips, ip)
		}
	}
}

// Global rate limiters for different use cases
var (
	// Strict rate limiter for authentication endpoints (login, register)
	// 5 requests per minute with burst of 10
	authLimiter = NewIPRateLimiter(rate.Every(12*time.Second), 10)

	// General API rate limiter
	// 100 requests per minute with burst of 200
	apiLimiter = NewIPRateLimiter(rate.Every(600*time.Millisecond), 200)
)

// Start cleanup routine to prevent memory leaks
func init() {
	// Clean up stale entries every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			authLimiter.CleanupStaleEntries()
			apiLimiter.CleanupStaleEntries()
		}
	}()
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP address
		ip := c.ClientIP()

		// Get the rate limiter for this IP
		rateLimiter := limiter.GetLimiter(ip)

		// Check if request is allowed
		if !rateLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthRateLimitMiddleware applies strict rate limiting for authentication endpoints
func AuthRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(authLimiter)
}

// APIRateLimitMiddleware applies general rate limiting for API endpoints
func APIRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(apiLimiter)
}
