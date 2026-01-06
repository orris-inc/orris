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
	adminTelegramVerifyPrefix      = "admin_telegram:verify:"
	adminTelegramVerifyTTL         = 10 * time.Minute
	adminTelegramVerifyCodeBytes   = 8 // 8 bytes = 64 bits of entropy (higher security for admin)
	adminTelegramVerifyRatePrefix  = "admin_telegram:verify:rate:"
	adminTelegramVerifyMaxAttempts = 3                // Max failed attempts (stricter for admin)
	adminTelegramVerifyLockoutTTL  = 30 * time.Minute // Longer lockout for admin (30 minutes)
)

// ErrAdminVerifyRateLimited is returned when too many failed attempts are made
var ErrAdminVerifyRateLimited = errors.New("too many failed verification attempts, please try again later")

// ErrAdminVerifyCodeInvalid is returned when the verification code is invalid or expired
var ErrAdminVerifyCodeInvalid = errors.New("verification code not found or expired")

// AdminTelegramVerifyStore provides Redis-based verification code storage for admin Telegram binding
// with rate limiting to prevent brute-force attacks
type AdminTelegramVerifyStore struct {
	client *redis.Client
}

// NewAdminTelegramVerifyStore creates a new AdminTelegramVerifyStore instance
func NewAdminTelegramVerifyStore(client *redis.Client) *AdminTelegramVerifyStore {
	return &AdminTelegramVerifyStore{client: client}
}

// Generate generates a new verify code for the admin user and stores it in Redis
// Returns a 16-character hex code (64 bits of entropy) that expires in 10 minutes
func (s *AdminTelegramVerifyStore) Generate(ctx context.Context, userID uint) (string, error) {
	// Generate a random 16-character code (8 bytes = 64 bits of entropy)
	// 64 bits provides ~18 quintillion possibilities, very resistant to brute-force attacks
	bytes := make([]byte, adminTelegramVerifyCodeBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	code := hex.EncodeToString(bytes)

	key := adminTelegramVerifyPrefix + code
	err := s.client.Set(ctx, key, userID, adminTelegramVerifyTTL).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store verify code: %w", err)
	}

	return code, nil
}

// VerifyWithRateLimit verifies the code with rate limiting based on telegram user ID
// Returns the associated userID if successful (one-time use)
// Uses GETDEL to atomically get and delete the key
func (s *AdminTelegramVerifyStore) VerifyWithRateLimit(ctx context.Context, code string, telegramUserID int64) (uint, error) {
	if code == "" {
		return 0, ErrAdminVerifyCodeInvalid
	}

	// Check rate limit first
	rateKey := adminTelegramVerifyRatePrefix + strconv.FormatInt(telegramUserID, 10)
	attempts, err := s.client.Get(ctx, rateKey).Int()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("failed to check rate limit: %w", err)
	}

	if attempts >= adminTelegramVerifyMaxAttempts {
		return 0, ErrAdminVerifyRateLimited
	}

	key := adminTelegramVerifyPrefix + code

	// GETDEL is atomic: get the value and delete the key in one operation
	val, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Increment failed attempts counter
			s.incrementFailedAttempts(ctx, rateKey)
			return 0, ErrAdminVerifyCodeInvalid
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
func (s *AdminTelegramVerifyStore) Verify(ctx context.Context, code string) (uint, error) {
	if code == "" {
		return 0, ErrAdminVerifyCodeInvalid
	}

	key := adminTelegramVerifyPrefix + code

	// GETDEL is atomic: get the value and delete the key in one operation
	val, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, ErrAdminVerifyCodeInvalid
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
func (s *AdminTelegramVerifyStore) incrementFailedAttempts(ctx context.Context, rateKey string) {
	pipe := s.client.Pipeline()
	pipe.Incr(ctx, rateKey)
	pipe.Expire(ctx, rateKey, adminTelegramVerifyLockoutTTL)
	_, _ = pipe.Exec(ctx)
}

// ClearRateLimit clears the rate limit for a telegram user (e.g., after successful bind via other means)
func (s *AdminTelegramVerifyStore) ClearRateLimit(ctx context.Context, telegramUserID int64) error {
	rateKey := adminTelegramVerifyRatePrefix + strconv.FormatInt(telegramUserID, 10)
	return s.client.Del(ctx, rateKey).Err()
}

// Delete removes the code (for cleanup)
func (s *AdminTelegramVerifyStore) Delete(ctx context.Context, code string) error {
	if code == "" {
		return nil
	}
	key := adminTelegramVerifyPrefix + code
	return s.client.Del(ctx, key).Err()
}
