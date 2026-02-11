package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/domain/user"
	apperrors "github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// rateLimitWindow is the time window for rate limiting password reset requests
	rateLimitWindow = 1 * time.Minute
	// rateLimitKeyPrefix is the Redis key prefix for password reset rate limiting
	rateLimitKeyPrefix = "pwreset:ratelimit:"
)

type RequestPasswordResetCommand struct {
	Email string
}

type RequestPasswordResetUseCase struct {
	userRepo     user.Repository
	emailService EmailService
	redisClient  *redis.Client
	logger       logger.Interface
}

func NewRequestPasswordResetUseCase(
	userRepo user.Repository,
	emailService EmailService,
	redisClient *redis.Client,
	logger logger.Interface,
) *RequestPasswordResetUseCase {
	return &RequestPasswordResetUseCase{
		userRepo:     userRepo,
		emailService: emailService,
		redisClient:  redisClient,
		logger:       logger,
	}
}

func (uc *RequestPasswordResetUseCase) Execute(ctx context.Context, cmd RequestPasswordResetCommand) error {
	// Check rate limit via Redis
	if err := uc.checkRateLimit(ctx, cmd.Email); err != nil {
		return err
	}

	existingUser, err := uc.userRepo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		uc.logger.Errorw("failed to get user by email", "error", err)
		return nil
	}
	if existingUser == nil {
		uc.logger.Warnw("password reset requested for non-existent email")
		return nil
	}

	if !existingUser.HasPassword() {
		uc.logger.Warnw("password reset requested for OAuth-only account", "user_id", existingUser.ID())
		return nil
	}

	token, err := existingUser.GeneratePasswordResetToken()
	if err != nil {
		uc.logger.Errorw("failed to generate reset token", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	if err := uc.userRepo.Update(ctx, existingUser); err != nil {
		uc.logger.Errorw("failed to update user", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to update user: %w", err)
	}

	if err := uc.emailService.SendPasswordResetEmail(cmd.Email, token.Value()); err != nil {
		uc.logger.Warnw("failed to send password reset email", "error", err)
	}

	// Record rate limit via Redis
	uc.recordRateLimit(ctx, cmd.Email)

	uc.logger.Infow("password reset requested", "user_id", existingUser.ID())

	return nil
}

// checkRateLimit checks if the email is rate limited using Redis
func (uc *RequestPasswordResetUseCase) checkRateLimit(ctx context.Context, email string) error {
	key := rateLimitKeyPrefix + email

	exists, err := uc.redisClient.Exists(ctx, key).Result()
	if err != nil {
		// If Redis is unavailable, allow the request to avoid blocking password resets entirely
		uc.logger.Warnw("failed to check rate limit from Redis, allowing request", "error", err)
		return nil
	}

	if exists > 0 {
		return apperrors.NewValidationError("please wait before requesting another password reset")
	}

	return nil
}

// recordRateLimit records the rate limit timestamp for an email in Redis with automatic expiry
func (uc *RequestPasswordResetUseCase) recordRateLimit(ctx context.Context, email string) {
	key := rateLimitKeyPrefix + email

	if err := uc.redisClient.Set(ctx, key, "1", rateLimitWindow).Err(); err != nil {
		uc.logger.Warnw("failed to record rate limit in Redis", "error", err)
	}
}
