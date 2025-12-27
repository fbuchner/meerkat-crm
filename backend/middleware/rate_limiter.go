package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Account lockout configuration
const (
	// MaxLoginAttempts before account lockout kicks in
	MaxLoginAttempts = 5
	// BaseLockoutDuration is the initial lockout period (doubles with each subsequent failure)
	BaseLockoutDuration = 1 * time.Minute
	// MaxLockoutDuration caps the exponential backoff
	MaxLockoutDuration = 30 * time.Minute
	// AccountLockoutTTL is how long to remember failed attempts after last failure
	AccountLockoutTTL = 1 * time.Hour
)

// AccountLockoutEntry tracks failed login attempts per account
type AccountLockoutEntry struct {
	FailedAttempts int
	LockedUntil    time.Time
	LastAttempt    time.Time
}

// AccountRateLimiter manages per-account login rate limiting with exponential backoff
type AccountRateLimiter struct {
	accounts map[string]*AccountLockoutEntry
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewAccountRateLimiter creates a new account-based rate limiter
func NewAccountRateLimiter(ttl time.Duration) *AccountRateLimiter {
	return &AccountRateLimiter{
		accounts: make(map[string]*AccountLockoutEntry),
		ttl:      ttl,
	}
}

// IsLocked checks if an account is currently locked out
// Returns (isLocked, remainingLockoutSeconds)
func (a *AccountRateLimiter) IsLocked(identifier string) (bool, int) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entry, exists := a.accounts[identifier]
	if !exists {
		return false, 0
	}

	now := time.Now()
	if entry.LockedUntil.After(now) {
		remaining := int(entry.LockedUntil.Sub(now).Seconds())
		return true, remaining
	}

	return false, 0
}

// RecordFailedAttempt records a failed login attempt and applies exponential backoff
// Returns (isNowLocked, lockoutDurationSeconds)
func (a *AccountRateLimiter) RecordFailedAttempt(identifier string) (bool, int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	entry, exists := a.accounts[identifier]

	if !exists {
		entry = &AccountLockoutEntry{
			FailedAttempts: 0,
			LastAttempt:    now,
		}
		a.accounts[identifier] = entry
	}

	entry.FailedAttempts++
	entry.LastAttempt = now

	// Apply lockout if we've exceeded max attempts
	if entry.FailedAttempts >= MaxLoginAttempts {
		// Calculate exponential backoff: base * 2^(attempts - maxAttempts)
		exponent := entry.FailedAttempts - MaxLoginAttempts
		lockoutDuration := BaseLockoutDuration * time.Duration(1<<exponent)

		// Cap at max lockout duration
		if lockoutDuration > MaxLockoutDuration {
			lockoutDuration = MaxLockoutDuration
		}

		entry.LockedUntil = now.Add(lockoutDuration)
		return true, int(lockoutDuration.Seconds())
	}

	return false, 0
}

// RecordSuccessfulLogin clears the failed attempt counter for an account
func (a *AccountRateLimiter) RecordSuccessfulLogin(identifier string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.accounts, identifier)
}

// GetFailedAttempts returns the current failed attempt count for an account
func (a *AccountRateLimiter) GetFailedAttempts(identifier string) int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entry, exists := a.accounts[identifier]
	if !exists {
		return 0
	}
	return entry.FailedAttempts
}

// CleanupStaleAccountEntries removes entries that haven't had activity within TTL
func (a *AccountRateLimiter) CleanupStaleAccountEntries() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	for identifier, entry := range a.accounts {
		// Remove entries where:
		// 1. Lockout has expired AND
		// 2. Last attempt was more than TTL ago
		if entry.LockedUntil.Before(now) && now.Sub(entry.LastAttempt) > a.ttl {
			delete(a.accounts, identifier)
		}
	}
}

// EntryCount returns the number of tracked accounts (for testing/monitoring)
func (a *AccountRateLimiter) EntryCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.accounts)
}

// Default TTL for rate limiter entries (10 minutes of inactivity)
const defaultLimiterTTL = 10 * time.Minute

// limiterEntry wraps a rate limiter with its last access time
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// IPRateLimiter manages rate limiters for different IP addresses
type IPRateLimiter struct {
	ips map[string]*limiterEntry
	mu  *sync.RWMutex
	r   rate.Limit    // requests per second
	b   int           // burst size
	ttl time.Duration // time-to-live for inactive entries
}

// NewIPRateLimiter creates a new IP-based rate limiter
// r: requests per second (e.g., 10 = 10 requests per second)
// b: burst size (e.g., 20 = allow bursts up to 20 requests)
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*limiterEntry),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
		ttl: defaultLimiterTTL,
	}
}

// NewIPRateLimiterWithTTL creates a new IP-based rate limiter with custom TTL
func NewIPRateLimiterWithTTL(r rate.Limit, b int, ttl time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*limiterEntry),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
		ttl: ttl,
	}
}

// GetLimiter returns the rate limiter for the given IP address
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	entry, exists := i.ips[ip]
	if !exists {
		entry = &limiterEntry{
			limiter:    rate.NewLimiter(i.r, i.b),
			lastAccess: time.Now(),
		}
		i.ips[ip] = entry
	} else {
		// Update last access time on each access
		entry.lastAccess = time.Now()
	}

	return entry.limiter
}

// CleanupStaleEntries removes rate limiters that haven't been accessed within the TTL
// This prevents memory leaks from accumulating limiters for old IPs
func (i *IPRateLimiter) CleanupStaleEntries() {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now()
	for ip, entry := range i.ips {
		// Remove entries that haven't been accessed within the TTL
		if now.Sub(entry.lastAccess) > i.ttl {
			delete(i.ips, ip)
		}
	}
}

// EntryCount returns the number of IP entries currently tracked (for testing/monitoring)
func (i *IPRateLimiter) EntryCount() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return len(i.ips)
}

// Global rate limiters for different use cases
var (
	// Strict rate limiter for authentication endpoints (login, register)
	// 5 requests per minute with burst of 10
	authLimiter = NewIPRateLimiter(rate.Every(12*time.Second), 10)

	// General API rate limiter
	// 100 requests per minute with burst of 500
	apiLimiter = NewIPRateLimiter(rate.Every(600*time.Millisecond), 500)

	// Per-account rate limiter for login attempts
	// Tracks failed attempts per username/email with exponential backoff
	accountLimiter = NewAccountRateLimiter(AccountLockoutTTL)
)

// GetAccountRateLimiter returns the global account rate limiter for login attempts
func GetAccountRateLimiter() *AccountRateLimiter {
	return accountLimiter
}

// Start cleanup routine to prevent memory leaks
func init() {
	// Clean up stale entries every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			authLimiter.CleanupStaleEntries()
			apiLimiter.CleanupStaleEntries()
			accountLimiter.CleanupStaleAccountEntries()
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
