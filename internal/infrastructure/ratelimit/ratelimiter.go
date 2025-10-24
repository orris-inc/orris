package ratelimit

import "time"

type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
	BurstSize         int
}

type RateLimiter interface {
	Allow(key string, config RateLimitConfig) (bool, error)
	GetRemaining(key string, window time.Duration) (int64, error)
	Reset(key string) error
}
