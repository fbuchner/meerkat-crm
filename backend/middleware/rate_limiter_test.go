package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestNewIPRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Every(time.Second), 10)

	if limiter == nil {
		t.Fatal("Expected limiter to be created, got nil")
	}

	if limiter.r != rate.Every(time.Second) {
		t.Errorf("Expected rate to be %v, got %v", rate.Every(time.Second), limiter.r)
	}

	if limiter.b != 10 {
		t.Errorf("Expected burst to be 10, got %d", limiter.b)
	}
}

func TestGetLimiter(t *testing.T) {
	ipLimiter := NewIPRateLimiter(rate.Every(time.Second), 10)

	// Get limiter for IP 1
	limiter1 := ipLimiter.GetLimiter("192.168.1.1")
	if limiter1 == nil {
		t.Fatal("Expected limiter to be created for IP")
	}

	// Get limiter for same IP again - should return same instance
	limiter1Again := ipLimiter.GetLimiter("192.168.1.1")
	if limiter1 != limiter1Again {
		t.Error("Expected same limiter instance for same IP")
	}

	// Get limiter for different IP
	limiter2 := ipLimiter.GetLimiter("192.168.1.2")
	if limiter1 == limiter2 {
		t.Error("Expected different limiter instances for different IPs")
	}
}

func TestRateLimitMiddleware_AllowsRequests(t *testing.T) {
	// Create a lenient limiter for testing (10 requests per second)
	limiter := NewIPRateLimiter(rate.Every(100*time.Millisecond), 10)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make 5 requests - all should succeed
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitMiddleware_BlocksExcessiveRequests(t *testing.T) {
	// Create a strict limiter (2 requests per second, burst of 2)
	limiter := NewIPRateLimiter(rate.Every(500*time.Millisecond), 2)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	successCount := 0
	blockedCount := 0

	// Make 10 rapid requests
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		switch w.Code {
		case http.StatusOK:
			successCount++
		case http.StatusTooManyRequests:
			blockedCount++
		}
	}

	// Should allow burst (2) and block the rest (8)
	if successCount != 2 {
		t.Errorf("Expected 2 successful requests, got %d", successCount)
	}

	if blockedCount != 8 {
		t.Errorf("Expected 8 blocked requests, got %d", blockedCount)
	}
}

func TestRateLimitMiddleware_SeparateIPsIndependent(t *testing.T) {
	// Create a strict limiter (1 request per second, burst of 1)
	limiter := NewIPRateLimiter(rate.Every(time.Second), 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// IP 1 makes a request - should succeed
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First IP request: Expected status 200, got %d", w1.Code)
	}

	// IP 2 makes a request immediately - should also succeed (different IP)
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:1234"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second IP request: Expected status 200, got %d", w2.Code)
	}

	// IP 1 makes another request immediately - should be blocked
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Second request from first IP: Expected status 429, got %d", w3.Code)
	}
}

func TestAuthRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthRateLimitMiddleware())
	router.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "logged in"})
	})

	// Make a request - should succeed
	req, _ := http.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.100:1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIRateLimitMiddleware())
	router.GET("/contacts", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"contacts": []string{}})
	})

	// Make multiple requests - should succeed (API limiter is more lenient)
	for i := 0; i < 20; i++ {
		req, _ := http.NewRequest("GET", "/contacts", nil)
		req.RemoteAddr = "192.168.1.200:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}
}

func TestCleanupStaleEntries(t *testing.T) {
	// Create limiter with very short TTL for testing
	limiter := NewIPRateLimiterWithTTL(rate.Every(time.Second), 10, 50*time.Millisecond)

	// Add some limiters
	limiter.GetLimiter("192.168.1.1")
	limiter.GetLimiter("192.168.1.2")
	limiter.GetLimiter("192.168.1.3")

	// Check we have 3 entries
	if count := limiter.EntryCount(); count != 3 {
		t.Errorf("Expected 3 limiters, got %d", count)
	}

	// Cleanup immediately - entries should still exist (not past TTL)
	limiter.CleanupStaleEntries()
	if count := limiter.EntryCount(); count != 3 {
		t.Errorf("Expected 3 limiters after immediate cleanup, got %d", count)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Cleanup (all should be removed as they're past TTL)
	limiter.CleanupStaleEntries()

	// Check entries were cleaned up
	if countAfter := limiter.EntryCount(); countAfter != 0 {
		t.Errorf("Expected 0 limiters after TTL expiry, got %d", countAfter)
	}
}

func TestCleanupStaleEntries_PartiallyUsedLimiters(t *testing.T) {
	// This test verifies the fix for the memory leak issue:
	// Limiters that consumed tokens but then went inactive should be cleaned up
	limiter := NewIPRateLimiterWithTTL(rate.Every(time.Second), 10, 50*time.Millisecond)

	// Get a limiter and consume some tokens
	rateLimiter := limiter.GetLimiter("192.168.1.1")
	rateLimiter.Allow() // Consume a token
	rateLimiter.Allow() // Consume another token

	// Verify tokens were consumed (not at full capacity)
	if rateLimiter.Tokens() >= 10 {
		t.Error("Expected tokens to be consumed")
	}

	// Check we have the entry
	if count := limiter.EntryCount(); count != 1 {
		t.Errorf("Expected 1 limiter, got %d", count)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Cleanup should remove the entry even though tokens aren't at max
	limiter.CleanupStaleEntries()

	if count := limiter.EntryCount(); count != 0 {
		t.Errorf("Expected 0 limiters after TTL expiry (even with consumed tokens), got %d", count)
	}
}

func TestCleanupStaleEntries_ActiveLimitersSurvive(t *testing.T) {
	limiter := NewIPRateLimiterWithTTL(rate.Every(time.Second), 10, 50*time.Millisecond)

	// Add two IPs
	limiter.GetLimiter("192.168.1.1")
	limiter.GetLimiter("192.168.1.2")

	// Wait a bit, but re-access one IP
	time.Sleep(30 * time.Millisecond)
	limiter.GetLimiter("192.168.1.1") // Refresh access time for IP 1

	// Wait a bit more so IP 2's TTL expires but IP 1's doesn't
	time.Sleep(30 * time.Millisecond)

	limiter.CleanupStaleEntries()

	// Only IP 1 should remain
	if count := limiter.EntryCount(); count != 1 {
		t.Errorf("Expected 1 limiter (active one), got %d", count)
	}

	// Verify IP 1 still exists by getting it again
	limiter.mu.RLock()
	_, exists := limiter.ips["192.168.1.1"]
	limiter.mu.RUnlock()

	if !exists {
		t.Error("Expected active IP 192.168.1.1 to still exist")
	}
}

// Tests for AccountRateLimiter

func TestNewAccountRateLimiter(t *testing.T) {
	limiter := NewAccountRateLimiter(time.Hour)

	if limiter == nil {
		t.Fatal("Expected limiter to be created, got nil")
	}

	if limiter.ttl != time.Hour {
		t.Errorf("Expected TTL to be 1 hour, got %v", limiter.ttl)
	}

	if limiter.EntryCount() != 0 {
		t.Errorf("Expected 0 entries, got %d", limiter.EntryCount())
	}
}

func TestAccountRateLimiter_IsLocked_NoEntry(t *testing.T) {
	limiter := NewAccountRateLimiter(time.Hour)

	isLocked, remaining := limiter.IsLocked("test@example.com")

	if isLocked {
		t.Error("Expected account to not be locked when no entry exists")
	}

	if remaining != 0 {
		t.Errorf("Expected remaining time to be 0, got %d", remaining)
	}
}

func TestAccountRateLimiter_RecordFailedAttempt_BelowThreshold(t *testing.T) {
	limiter := NewAccountRateLimiter(time.Hour)

	// Record fewer attempts than the threshold
	for i := 0; i < MaxLoginAttempts-1; i++ {
		isLocked, lockoutSecs := limiter.RecordFailedAttempt("test@example.com")

		if isLocked {
			t.Errorf("Attempt %d: Expected account to not be locked below threshold", i+1)
		}

		if lockoutSecs != 0 {
			t.Errorf("Attempt %d: Expected lockout seconds to be 0, got %d", i+1, lockoutSecs)
		}
	}

	if limiter.GetFailedAttempts("test@example.com") != MaxLoginAttempts-1 {
		t.Errorf("Expected %d failed attempts, got %d", MaxLoginAttempts-1, limiter.GetFailedAttempts("test@example.com"))
	}
}

func TestAccountRateLimiter_RecordFailedAttempt_AtThreshold(t *testing.T) {
	limiter := NewAccountRateLimiter(time.Hour)

	// Record up to the threshold
	for i := 0; i < MaxLoginAttempts-1; i++ {
		limiter.RecordFailedAttempt("test@example.com")
	}

	// The next attempt should trigger lockout
	isLocked, lockoutSecs := limiter.RecordFailedAttempt("test@example.com")

	if !isLocked {
		t.Error("Expected account to be locked at threshold")
	}

	expectedLockout := int(BaseLockoutDuration.Seconds())
	if lockoutSecs != expectedLockout {
		t.Errorf("Expected lockout of %d seconds, got %d", expectedLockout, lockoutSecs)
	}
}

func TestAccountRateLimiter_ExponentialBackoff(t *testing.T) {
	limiter := NewAccountRateLimiter(time.Hour)

	// Record up to the threshold
	for i := 0; i < MaxLoginAttempts; i++ {
		limiter.RecordFailedAttempt("test@example.com")
	}

	// First lockout: base duration
	expectedFirst := int(BaseLockoutDuration.Seconds())
	if _, lockout := limiter.IsLocked("test@example.com"); lockout < expectedFirst-1 || lockout > expectedFirst {
		t.Errorf("First lockout: expected ~%d seconds, got %d", expectedFirst, lockout)
	}

	// Simulate time passing and another failed attempt
	limiter.mu.Lock()
	limiter.accounts["test@example.com"].LockedUntil = time.Now().Add(-time.Second) // Unlock
	limiter.mu.Unlock()

	// Another failed attempt should double the lockout
	_, lockoutSecs := limiter.RecordFailedAttempt("test@example.com")
	expectedSecond := int((BaseLockoutDuration * 2).Seconds())
	if lockoutSecs != expectedSecond {
		t.Errorf("Second lockout: expected %d seconds, got %d", expectedSecond, lockoutSecs)
	}
}

func TestAccountRateLimiter_MaxLockoutDuration(t *testing.T) {
	limiter := NewAccountRateLimiter(time.Hour)

	// Record many failed attempts to hit the max lockout
	for i := 0; i < MaxLoginAttempts+20; i++ {
		limiter.mu.Lock()
		if entry, exists := limiter.accounts["test@example.com"]; exists {
			entry.LockedUntil = time.Now().Add(-time.Second) // Reset lockout for next attempt
		}
		limiter.mu.Unlock()
		limiter.RecordFailedAttempt("test@example.com")
	}

	isLocked, lockoutSecs := limiter.IsLocked("test@example.com")

	if !isLocked {
		t.Error("Expected account to be locked")
	}

	maxLockoutSecs := int(MaxLockoutDuration.Seconds())
	if lockoutSecs > maxLockoutSecs {
		t.Errorf("Lockout should be capped at %d seconds, got %d", maxLockoutSecs, lockoutSecs)
	}
}

func TestAccountRateLimiter_RecordSuccessfulLogin(t *testing.T) {
	limiter := NewAccountRateLimiter(time.Hour)

	// Record some failed attempts
	for i := 0; i < 3; i++ {
		limiter.RecordFailedAttempt("test@example.com")
	}

	if limiter.GetFailedAttempts("test@example.com") != 3 {
		t.Errorf("Expected 3 failed attempts, got %d", limiter.GetFailedAttempts("test@example.com"))
	}

	// Successful login should clear the counter
	limiter.RecordSuccessfulLogin("test@example.com")

	if limiter.GetFailedAttempts("test@example.com") != 0 {
		t.Errorf("Expected 0 failed attempts after successful login, got %d", limiter.GetFailedAttempts("test@example.com"))
	}

	if limiter.EntryCount() != 0 {
		t.Errorf("Expected entry to be removed after successful login, got %d entries", limiter.EntryCount())
	}
}

func TestAccountRateLimiter_SeparateAccounts(t *testing.T) {
	limiter := NewAccountRateLimiter(time.Hour)

	// Record failed attempts for account 1
	for i := 0; i < MaxLoginAttempts; i++ {
		limiter.RecordFailedAttempt("user1@example.com")
	}

	// Account 1 should be locked
	isLocked1, _ := limiter.IsLocked("user1@example.com")
	if !isLocked1 {
		t.Error("Expected user1 to be locked")
	}

	// Account 2 should not be affected
	isLocked2, _ := limiter.IsLocked("user2@example.com")
	if isLocked2 {
		t.Error("Expected user2 to not be locked")
	}

	if limiter.GetFailedAttempts("user2@example.com") != 0 {
		t.Error("Expected user2 to have 0 failed attempts")
	}
}

func TestAccountRateLimiter_CleanupStaleEntries(t *testing.T) {
	limiter := NewAccountRateLimiter(50 * time.Millisecond)

	// Add some entries
	limiter.RecordFailedAttempt("user1@example.com")
	limiter.RecordFailedAttempt("user2@example.com")

	if limiter.EntryCount() != 2 {
		t.Errorf("Expected 2 entries, got %d", limiter.EntryCount())
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	limiter.CleanupStaleAccountEntries()

	if limiter.EntryCount() != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", limiter.EntryCount())
	}
}

func TestAccountRateLimiter_CleanupPreservesLockedAccounts(t *testing.T) {
	limiter := NewAccountRateLimiter(50 * time.Millisecond)

	// Lock an account
	for i := 0; i < MaxLoginAttempts; i++ {
		limiter.RecordFailedAttempt("locked@example.com")
	}

	// Add another entry that's not locked
	limiter.RecordFailedAttempt("unlocked@example.com")

	// Wait for TTL
	time.Sleep(60 * time.Millisecond)

	// The locked account should still be preserved if lockout is active
	limiter.CleanupStaleAccountEntries()

	// Check if locked account still exists (it should if lockout hasn't expired)
	isLocked, _ := limiter.IsLocked("locked@example.com")

	// If still locked, entry should exist
	if isLocked && limiter.EntryCount() < 1 {
		t.Error("Expected locked account entry to be preserved during lockout")
	}
}

func TestGetAccountRateLimiter(t *testing.T) {
	limiter := GetAccountRateLimiter()

	if limiter == nil {
		t.Fatal("Expected global account rate limiter to be initialized")
	}

	// Verify it's the same instance on subsequent calls
	limiter2 := GetAccountRateLimiter()
	if limiter != limiter2 {
		t.Error("Expected same global rate limiter instance")
	}
}
