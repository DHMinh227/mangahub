package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int           // max requests
	window   time.Duration // time window
}

var limiter = &rateLimiter{
	requests: make(map[string][]time.Time),
	limit:    100,             // 100 requests
	window:   1 * time.Minute, // per minute
}

// cleanup old entries periodically
func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiter.mu.Lock()
			now := time.Now()
			for ip, times := range limiter.requests {
				var valid []time.Time
				for _, t := range times {
					if now.Sub(t) < limiter.window {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(limiter.requests, ip)
				} else {
					limiter.requests[ip] = valid
				}
			}
			limiter.mu.Unlock()
		}
	}()
}

func (r *rateLimiter) isAllowed(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	// Filter requests within the window
	var valid []time.Time
	for _, t := range r.requests[ip] {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= r.limit {
		r.requests[ip] = valid
		return false
	}

	// Add current request
	r.requests[ip] = append(valid, now)
	return true
}

// RateLimitMiddleware limits requests per IP
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.isAllowed(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// StrictRateLimitMiddleware for sensitive endpoints (login, register)
// 10 requests per minute
func StrictRateLimitMiddleware() gin.HandlerFunc {
	strictLimiter := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    10,
		window:   1 * time.Minute,
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !strictLimiter.isAllowed(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many attempts, try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
