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
	limiter := NewIPRateLimiter(rate.Every(time.Second), 10)

	// Add some limiters
	limiter.GetLimiter("192.168.1.1")
	limiter.GetLimiter("192.168.1.2")
	limiter.GetLimiter("192.168.1.3")

	// Check we have 3 entries
	limiter.mu.RLock()
	count := len(limiter.ips)
	limiter.mu.RUnlock()

	if count != 3 {
		t.Errorf("Expected 3 limiters, got %d", count)
	}

	// Cleanup (all should be removed as they haven't been used)
	limiter.CleanupStaleEntries()

	// Check entries were cleaned up
	limiter.mu.RLock()
	countAfter := len(limiter.ips)
	limiter.mu.RUnlock()

	if countAfter != 0 {
		t.Errorf("Expected 0 limiters after cleanup, got %d", countAfter)
	}
}
