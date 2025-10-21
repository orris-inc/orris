package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	client.FlushDB(ctx)

	t.Cleanup(func() {
		client.FlushDB(ctx)
		client.Close()
	})

	return client
}

func TestRedisRateLimiter_Allow_PerMinute(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 5,
		RequestsPerHour:   0,
		RequestsPerDay:    0,
	}

	key := "test-key-minute"

	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(key, config)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	allowed, err := limiter.Allow(key, config)
	require.NoError(t, err)
	assert.False(t, allowed, "6th request should be denied")
}

func TestRedisRateLimiter_Allow_PerHour(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 0,
		RequestsPerHour:   3,
		RequestsPerDay:    0,
	}

	key := "test-key-hour"

	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(key, config)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	allowed, err := limiter.Allow(key, config)
	require.NoError(t, err)
	assert.False(t, allowed, "4th request should be denied")
}

func TestRedisRateLimiter_Allow_PerDay(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 0,
		RequestsPerHour:   0,
		RequestsPerDay:    10,
	}

	key := "test-key-day"

	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(key, config)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	allowed, err := limiter.Allow(key, config)
	require.NoError(t, err)
	assert.False(t, allowed, "11th request should be denied")
}

func TestRedisRateLimiter_Allow_MultipleWindows(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 5,
		RequestsPerHour:   10,
		RequestsPerDay:    20,
	}

	key := "test-key-multi"

	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(key, config)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	allowed, err := limiter.Allow(key, config)
	require.NoError(t, err)
	assert.False(t, allowed, "6th request should be denied by minute limit")
}

func TestRedisRateLimiter_Allow_DifferentKeys(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 2,
		RequestsPerHour:   0,
		RequestsPerDay:    0,
	}

	key1 := "test-key-1"
	key2 := "test-key-2"

	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(key1, config)
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	allowed, err := limiter.Allow(key1, config)
	require.NoError(t, err)
	assert.False(t, allowed, "key1 should be rate limited")

	allowed, err = limiter.Allow(key2, config)
	require.NoError(t, err)
	assert.True(t, allowed, "key2 should not be affected")
}

func TestRedisRateLimiter_GetRemaining(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 5,
		RequestsPerHour:   0,
		RequestsPerDay:    0,
	}

	key := "test-key-remaining"

	remaining, err := limiter.GetRemaining(key, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(0), remaining)

	for i := 0; i < 3; i++ {
		_, err := limiter.Allow(key, config)
		require.NoError(t, err)
	}

	remaining, err = limiter.GetRemaining(key, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(3), remaining)
}

func TestRedisRateLimiter_Reset(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 2,
		RequestsPerHour:   0,
		RequestsPerDay:    0,
	}

	key := "test-key-reset"

	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(key, config)
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	allowed, err := limiter.Allow(key, config)
	require.NoError(t, err)
	assert.False(t, allowed)

	err = limiter.Reset(key)
	require.NoError(t, err)

	allowed, err = limiter.Allow(key, config)
	require.NoError(t, err)
	assert.True(t, allowed, "should be allowed after reset")
}

func TestRedisRateLimiter_SlidingWindow(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 3,
		RequestsPerHour:   0,
		RequestsPerDay:    0,
	}

	key := "test-key-sliding"

	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(key, config)
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	allowed, err := limiter.Allow(key, config)
	require.NoError(t, err)
	assert.False(t, allowed)

	time.Sleep(2 * time.Second)

	allowed, err = limiter.Allow(key, config)
	require.NoError(t, err)
	assert.False(t, allowed, "should still be limited in sliding window")
}

func TestRedisRateLimiter_ZeroLimits(t *testing.T) {
	client := setupTestRedis(t)
	limiter := NewRedisRateLimiter(client)

	config := RateLimitConfig{
		RequestsPerMinute: 0,
		RequestsPerHour:   0,
		RequestsPerDay:    0,
	}

	key := "test-key-zero"

	allowed, err := limiter.Allow(key, config)
	require.NoError(t, err)
	assert.True(t, allowed, "zero limits should allow all requests")
}

func BenchmarkRedisRateLimiter_Allow(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	client.FlushDB(ctx)
	defer client.Close()

	limiter := NewRedisRateLimiter(client)
	config := RateLimitConfig{
		RequestsPerMinute: 1000,
		RequestsPerHour:   10000,
		RequestsPerDay:    100000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow("bench-key", config)
	}
}

func BenchmarkRedisRateLimiter_GetRemaining(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	client.FlushDB(ctx)
	defer client.Close()

	limiter := NewRedisRateLimiter(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.GetRemaining("bench-key", time.Minute)
	}
}
