package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter represents a rate limiter
type RateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter manages rate limiters for different IPs
type IPRateLimiter struct {
	ips map[string]*RateLimiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter creates a new IP rate limiter
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*RateLimiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	// Clean up old entries every minute
	go i.cleanupRoutine()

	return i
}

// AddIP creates a new rate limiter for an IP
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)

	i.ips[ip] = &RateLimiter{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}

// GetLimiter returns the rate limiter for an IP
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.ips[ip]

	if !exists {
		i.mu.Unlock()
		return i.AddIP(ip)
	}

	// Update last seen time
	limiter.lastSeen = time.Now()
	i.mu.Unlock()

	return limiter.limiter
}

// cleanupRoutine removes old entries
func (i *IPRateLimiter) cleanupRoutine() {
	for {
		time.Sleep(time.Minute)

		i.mu.Lock()
		for ip, limiter := range i.ips {
			if time.Since(limiter.lastSeen) > 3*time.Minute {
				delete(i.ips, ip)
			}
		}
		i.mu.Unlock()
	}
}

var limiter = NewIPRateLimiter(rate.Every(time.Second), 10) // 10 requests per second

// RateLimit returns a rate limiting middleware
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := limiter.GetLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitWithConfig returns a rate limiting middleware with custom config
func RateLimitWithConfig(requestsPerSecond int, burst int) gin.HandlerFunc {
	customLimiter := NewIPRateLimiter(rate.Every(time.Second/time.Duration(requestsPerSecond)), burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := customLimiter.GetLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}