package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	telegramVerifyPrefix      = "telegram:verify:"
	telegramVerifyTTL         = 10 * time.Minute
	telegramVerifyCodeBytes   = 6 // 6 bytes = 48 bits of entropy (~281 trillion possibilities)
	telegramVerifyRatePrefix  = "telegram:verify:rate:"
	telegramVerifyMaxAttempts = 5                // Max failed attempts per telegram user
	telegramVerifyLockoutTTL  = 15 * time.Minute // Lockout duration after max attempts
)

// ErrVerifyRateLimited is returned when too many failed attempts are made
var ErrVerifyRateLimited = errors.New("too many failed verification attempts, please try again later")

// ErrVerifyCodeInvalid is returned when the verification code is invalid or expired
var ErrVerifyCodeInvalid = errors.New("verification code not found or expired")

// TelegramVerifyStore provides Redis-based verification code storage for Telegram binding
// with rate limiting to prevent brute-force attacks
type TelegramVerifyStore struct {
	client *redis.Client
}

// NewTelegramVerifyStore creates a new TelegramVerifyStore instance
func NewTelegramVerifyStore(client *redis.Client) *TelegramVerifyStore {
	return &TelegramVerifyStore{client: client}
}

// Generate generates a new verify code for the user and stores it in Redis
// Returns a 12-character hex code (48 bits of entropy) that expires in 10 minutes
func (s *TelegramVerifyStore) Generate(ctx context.Context, userID uint) (string, error) {
	// Generate a random 12-character code (6 bytes = 48 bits of entropy)
	// 48 bits provides ~281 trillion possibilities, resistant to brute-force attacks
	bytes := make([]byte, telegramVerifyCodeBytes)
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

// VerifyWithRateLimit verifies the code with rate limiting based on telegram user ID
// Returns the associated userID if successful (one-time use)
// Uses GETDEL to atomically get and delete the key
func (s *TelegramVerifyStore) VerifyWithRateLimit(ctx context.Context, code string, telegramUserID int64) (uint, error) {
	if code == "" {
		return 0, ErrVerifyCodeInvalid
	}

	// Check rate limit first
	rateKey := telegramVerifyRatePrefix + strconv.FormatInt(telegramUserID, 10)
	attempts, err := s.client.Get(ctx, rateKey).Int()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("failed to check rate limit: %w", err)
	}

	if attempts >= telegramVerifyMaxAttempts {
		return 0, ErrVerifyRateLimited
	}

	key := telegramVerifyPrefix + code

	// GETDEL is atomic: get the value and delete the key in one operation
	val, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Increment failed attempts counter
			s.incrementFailedAttempts(ctx, rateKey)
			return 0, ErrVerifyCodeInvalid
		}
		return 0, fmt.Errorf("failed to get verify code: %w", err)
	}

	// Success - clear the rate limit counter
	s.client.Del(ctx, rateKey)

	userID, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID in verify code: %w", err)
	}

	return uint(userID), nil
}

// Verify verifies the code without rate limiting (backward compatible)
// Prefer VerifyWithRateLimit for new code
func (s *TelegramVerifyStore) Verify(ctx context.Context, code string) (uint, error) {
	if code == "" {
		return 0, ErrVerifyCodeInvalid
	}

	key := telegramVerifyPrefix + code

	// GETDEL is atomic: get the value and delete the key in one operation
	val, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, ErrVerifyCodeInvalid
		}
		return 0, fmt.Errorf("failed to get verify code: %w", err)
	}

	userID, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID in verify code: %w", err)
	}

	return uint(userID), nil
}

// incrementFailedAttempts increments the failed attempts counter for a telegram user
func (s *TelegramVerifyStore) incrementFailedAttempts(ctx context.Context, rateKey string) {
	pipe := s.client.Pipeline()
	pipe.Incr(ctx, rateKey)
	pipe.Expire(ctx, rateKey, telegramVerifyLockoutTTL)
	_, _ = pipe.Exec(ctx)
}

// ClearRateLimit clears the rate limit for a telegram user (e.g., after successful bind via other means)
func (s *TelegramVerifyStore) ClearRateLimit(ctx context.Context, telegramUserID int64) error {
	rateKey := telegramVerifyRatePrefix + strconv.FormatInt(telegramUserID, 10)
	return s.client.Del(ctx, rateKey).Err()
}

// Delete removes the code (for cleanup)
func (s *TelegramVerifyStore) Delete(ctx context.Context, code string) error {
	if code == "" {
		return nil
	}
	key := telegramVerifyPrefix + code
	return s.client.Del(ctx, key).Err()
}
