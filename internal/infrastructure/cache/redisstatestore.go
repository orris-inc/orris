package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// StateInfo stores state-related information for OAuth flow
type StateInfo struct {
	CodeVerifier string    `json:"code_verifier"`
	CreatedAt    time.Time `json:"created_at"`
}

// RedisStateStore provides Redis-based state storage for OAuth flows
type RedisStateStore struct {
	client *redis.Client
	prefix string        // Key prefix, e.g., "oauth:state:"
	ttl    time.Duration // Expiration time for state keys
}

// NewRedisStateStore creates a new RedisStateStore instance
// Parameters:
//   - client: Redis client instance
//   - prefix: Key prefix for namespacing (e.g., "oauth:state:")
//   - ttl: Time-to-live for state keys (recommended: 10 minutes)
func NewRedisStateStore(client *redis.Client, prefix string, ttl time.Duration) *RedisStateStore {
	return &RedisStateStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// Set stores state and code_verifier in Redis with TTL
// The state will automatically expire after the configured TTL
// Returns an error if Redis operation fails
func (s *RedisStateStore) Set(ctx context.Context, state string, codeVerifier string) error {
	if state == "" {
		return errors.New("state cannot be empty")
	}
	if codeVerifier == "" {
		return errors.New("code_verifier cannot be empty")
	}

	stateInfo := StateInfo{
		CodeVerifier: codeVerifier,
		CreatedAt:    biztime.NowUTC(),
	}

	data, err := json.Marshal(stateInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal state info: %w", err)
	}

	key := s.buildKey(state)
	err = s.client.Set(ctx, key, data, s.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store state in redis: %w", err)
	}

	return nil
}

// VerifyAndGet verifies state exists and retrieves the code_verifier (one-time use)
// This method uses GETDEL command to atomically get and delete the key,
// ensuring the state can only be used once (preventing replay attacks)
// Returns StateInfo if found, or an error if:
//   - state not found (expired or already used)
//   - Redis operation fails
//   - JSON unmarshal fails
func (s *RedisStateStore) VerifyAndGet(ctx context.Context, state string) (*StateInfo, error) {
	if state == "" {
		return nil, errors.New("state cannot be empty")
	}

	key := s.buildKey(state)

	// GETDEL is atomic: get the value and delete the key in one operation
	// This ensures one-time use semantics
	data, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("state not found or expired")
		}
		return nil, fmt.Errorf("failed to retrieve state from redis: %w", err)
	}

	var stateInfo StateInfo
	err = json.Unmarshal([]byte(data), &stateInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal state info: %w", err)
	}

	return &stateInfo, nil
}

// buildKey constructs the full Redis key with prefix
func (s *RedisStateStore) buildKey(state string) string {
	return s.prefix + state
}
