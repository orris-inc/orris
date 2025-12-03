package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/utils"
)

type rateLimiterEntry struct {
	count     int
	resetTime time.Time
}

type RateLimiter struct {
	mu      sync.RWMutex
	entries map[string]*rateLimiterEntry
	limit   int
	window  time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*rateLimiterEntry),
		limit:   limit,
		window:  window,
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.entries {
			if now.After(entry.resetTime) {
				delete(rl.entries, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		rl.mu.Lock()
		entry, exists := rl.entries[key]
		now := time.Now()

		if !exists || now.After(entry.resetTime) {
			rl.entries[key] = &rateLimiterEntry{
				count:     1,
				resetTime: now.Add(rl.window),
			}
			rl.mu.Unlock()
			c.Next()
			return
		}

		if entry.count >= rl.limit {
			rl.mu.Unlock()
			utils.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded, please try again later")
			c.Abort()
			return
		}

		entry.count++
		rl.mu.Unlock()

		c.Next()
	}
}
