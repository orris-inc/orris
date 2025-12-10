package auth

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

// ConnectionTokenInfo stores token-related information
type ConnectionTokenInfo struct {
	EntryAgentID string    `json:"entry_agent_id"`
	ExitAgentID  string    `json:"exit_agent_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// AgentConnectionTokenService handles generation and verification of
// short-term connection tokens for agent-to-agent tunnel establishment.
// Tokens are stored in Redis and can only be used once.
type AgentConnectionTokenService struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewAgentConnectionTokenService creates a new AgentConnectionTokenService instance.
// client: Redis client for token storage.
// ttl: token expiration time (recommended: 5 minutes).
func NewAgentConnectionTokenService(client *redis.Client, ttl time.Duration) *AgentConnectionTokenService {
	return &AgentConnectionTokenService{
		client: client,
		prefix: "conn_token:",
		ttl:    ttl,
	}
}

// Generate creates a new random connection token and stores it in Redis.
// entryAgentID: the ID of the entry agent initiating the connection.
// exitAgentID: the ID of the exit agent accepting the connection.
// Returns a random token string or an error.
func (s *AgentConnectionTokenService) Generate(entryAgentID, exitAgentID string) (string, error) {
	// Generate random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(bytes)

	// Store token info in Redis
	info := ConnectionTokenInfo{
		EntryAgentID: entryAgentID,
		ExitAgentID:  exitAgentID,
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token info: %w", err)
	}

	ctx := context.Background()
	key := s.prefix + token
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("failed to store token in redis: %w", err)
	}

	return token, nil
}

// Verify validates a connection token and returns its info (one-time use).
// Uses GETDEL to atomically get and delete the token, preventing replay attacks.
// Returns the token info if valid, or an error if invalid/expired/already used.
func (s *AgentConnectionTokenService) Verify(token string) (*ConnectionTokenInfo, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	ctx := context.Background()
	key := s.prefix + token

	// GETDEL is atomic: get the value and delete the key in one operation
	data, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("token not found or expired")
		}
		return nil, fmt.Errorf("failed to retrieve token from redis: %w", err)
	}

	var info ConnectionTokenInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token info: %w", err)
	}

	return &info, nil
}
