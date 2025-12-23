package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	telegramVerifyPrefix = "telegram:verify:"
	telegramVerifyTTL    = 10 * time.Minute
)

// TelegramVerifyStore provides Redis-based verification code storage for Telegram binding
type TelegramVerifyStore struct {
	client *redis.Client
}

// NewTelegramVerifyStore creates a new TelegramVerifyStore instance
func NewTelegramVerifyStore(client *redis.Client) *TelegramVerifyStore {
	return &TelegramVerifyStore{client: client}
}

// Generate generates a new verify code for the user and stores it in Redis
// Returns a 6-character hex code that expires in 10 minutes
func (s *TelegramVerifyStore) Generate(ctx context.Context, userID uint) (string, error) {
	// Generate a random 6-character code
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	code := hex.EncodeToString(bytes)

	key := telegramVerifyPrefix + code
	err := s.client.Set(ctx, key, userID, telegramVerifyTTL).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store verify code: %w", err)
	}

	return code, nil
}

// Verify verifies the code and returns the associated userID (one-time use)
// Uses GETDEL to atomically get and delete the key
func (s *TelegramVerifyStore) Verify(ctx context.Context, code string) (uint, error) {
	if code == "" {
		return 0, fmt.Errorf("verify code cannot be empty")
	}

	key := telegramVerifyPrefix + code

	// GETDEL is atomic: get the value and delete the key in one operation
	val, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("verify code not found or expired")
		}
		return 0, fmt.Errorf("failed to get verify code: %w", err)
	}

	userID, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID in verify code: %w", err)
	}

	return uint(userID), nil
}

// Delete removes the code (for cleanup)
func (s *TelegramVerifyStore) Delete(ctx context.Context, code string) error {
	if code == "" {
		return nil
	}
	key := telegramVerifyPrefix + code
	return s.client.Del(ctx, key).Err()
}
