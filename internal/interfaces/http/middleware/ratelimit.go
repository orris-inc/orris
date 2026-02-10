package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/shared/utils"
)

// RateLimiter provides Redis-backed IP rate limiting using a fixed-window counter.
// Each IP gets a counter key with TTL equal to the window duration.
// This works correctly in multi-instance deployments since all instances share Redis.
type RateLimiter struct {
	redisClient *redis.Client
	limit       int
	window      time.Duration
}

// NewRateLimiter creates a new Redis-backed rate limiter.
// limit is the maximum number of requests allowed per window.
// window is the duration of the fixed time window.
func NewRateLimiter(redisClient *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
		limit:       limit,
		window:      window,
	}
}

// Limit returns a Gin middleware that enforces the rate limit per client IP.
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		windowBucket := time.Now().Unix() / int64(rl.window.Seconds())
		key := fmt.Sprintf("ratelimit:ip:%s:%d", clientIP, windowBucket)

		ctx := context.Background()

		// Use INCR to atomically increment the counter and check if this is the first request
		count, err := rl.redisClient.Incr(ctx, key).Result()
		if err != nil {
			// If Redis is unavailable, allow the request to avoid blocking all traffic
			c.Next()
			return
		}

		// Set TTL on the key for the first request in this window
		if count == 1 {
			rl.redisClient.Expire(ctx, key, rl.window+time.Second)
		}

		if count > int64(rl.limit) {
			utils.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}
