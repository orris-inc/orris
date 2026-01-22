package cache

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// PasskeySignupSessionPrefix is the Redis key prefix for passkey signup sessions
	PasskeySignupSessionPrefix = "passkey:signup:session:"
	// PasskeySignupSessionTTL is the default TTL for passkey signup sessions (5 minutes)
	PasskeySignupSessionTTL = 5 * time.Minute
)

// PasskeySignupSession stores temporary registration data for passkey signup
type PasskeySignupSession struct {
	SessionToken string `json:"session_token"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	TempUserID   []byte `json:"temp_user_id"`
	CreatedAt    int64  `json:"created_at"` // Unix timestamp in milliseconds
}

// PasskeySignupSessionStore stores temporary passkey signup sessions
type PasskeySignupSessionStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewPasskeySignupSessionStore creates a new passkey signup session store
func NewPasskeySignupSessionStore(client *redis.Client) *PasskeySignupSessionStore {
	return &PasskeySignupSessionStore{
		client: client,
		prefix: PasskeySignupSessionPrefix,
		ttl:    PasskeySignupSessionTTL,
	}
}

// NewPasskeySignupSessionStoreWithConfig creates a new store with custom config
func NewPasskeySignupSessionStoreWithConfig(client *redis.Client, prefix string, ttl time.Duration) *PasskeySignupSessionStore {
	if prefix == "" {
		prefix = PasskeySignupSessionPrefix
	}
	if ttl == 0 {
		ttl = PasskeySignupSessionTTL
	}
	return &PasskeySignupSessionStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// GenerateSessionToken generates a cryptographically secure session token
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Store saves a passkey signup session
func (s *PasskeySignupSessionStore) Store(ctx context.Context, session *PasskeySignupSession) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}
	if session.SessionToken == "" {
		return errors.New("session token cannot be empty")
	}
	if session.Email == "" {
		return errors.New("email cannot be empty")
	}
	if session.Name == "" {
		return errors.New("name cannot be empty")
	}
	if len(session.TempUserID) == 0 {
		return errors.New("temp user ID cannot be empty")
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal signup session: %w", err)
	}

	key := s.buildKey(session.SessionToken)
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store signup session in Redis: %w", err)
	}

	return nil
}

// Get retrieves a passkey signup session by token (one-time use)
func (s *PasskeySignupSessionStore) Get(ctx context.Context, sessionToken string) (*PasskeySignupSession, error) {
	if sessionToken == "" {
		return nil, errors.New("session token cannot be empty")
	}

	key := s.buildKey(sessionToken)

	// Use GETDEL for one-time use semantics
	data, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("signup session not found or expired")
		}
		return nil, fmt.Errorf("failed to retrieve signup session from Redis: %w", err)
	}

	var session PasskeySignupSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signup session: %w", err)
	}

	return &session, nil
}

// Delete removes a signup session from the store
func (s *PasskeySignupSessionStore) Delete(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return errors.New("session token cannot be empty")
	}

	key := s.buildKey(sessionToken)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete signup session from Redis: %w", err)
	}

	return nil
}

// buildKey constructs the full Redis key
func (s *PasskeySignupSessionStore) buildKey(sessionToken string) string {
	return s.prefix + sessionToken
}
