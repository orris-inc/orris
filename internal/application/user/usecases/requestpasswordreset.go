package usecases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// rateLimitWindow is the time window for rate limiting password reset requests
	rateLimitWindow = 1 * time.Minute
	// rateLimitCleanupInterval is how often expired entries are cleaned up
	rateLimitCleanupInterval = 10 * time.Minute
)

type RequestPasswordResetCommand struct {
	Email string
}

type RequestPasswordResetUseCase struct {
	userRepo      user.Repository
	emailService  EmailService
	logger        logger.Interface
	rateLimiter   map[string]time.Time
	rateLimiterMu sync.Mutex
	lastCleanup   time.Time
}

func NewRequestPasswordResetUseCase(
	userRepo user.Repository,
	emailService EmailService,
	logger logger.Interface,
) *RequestPasswordResetUseCase {
	return &RequestPasswordResetUseCase{
		userRepo:     userRepo,
		emailService: emailService,
		logger:       logger,
		rateLimiter:  make(map[string]time.Time),
		lastCleanup:  biztime.NowUTC(),
	}
}

func (uc *RequestPasswordResetUseCase) Execute(ctx context.Context, cmd RequestPasswordResetCommand) error {
	// Check rate limit with mutex protection
	if err := uc.checkRateLimit(cmd.Email); err != nil {
		return err
	}

	existingUser, err := uc.userRepo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		uc.logger.Errorw("failed to get user by email", "error", err)
		return nil
	}
	if existingUser == nil {
		uc.logger.Infow("password reset requested for non-existent email", "email", cmd.Email)
		return nil
	}

	if !existingUser.HasPassword() {
		uc.logger.Infow("password reset requested for OAuth-only account", "user_id", existingUser.ID())
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
		uc.logger.Warnw("failed to send password reset email", "error", err, "email", cmd.Email)
	}

	// Record rate limit with mutex protection
	uc.recordRateLimit(cmd.Email)

	uc.logger.Infow("password reset requested", "user_id", existingUser.ID())

	return nil
}

// checkRateLimit checks if the email is rate limited and cleans up expired entries
func (uc *RequestPasswordResetUseCase) checkRateLimit(email string) error {
	uc.rateLimiterMu.Lock()
	defer uc.rateLimiterMu.Unlock()

	now := biztime.NowUTC()

	// Periodically cleanup expired entries to prevent memory leak
	if now.Sub(uc.lastCleanup) > rateLimitCleanupInterval {
		uc.cleanupExpiredEntries(now)
		uc.lastCleanup = now
	}

	if lastRequest, exists := uc.rateLimiter[email]; exists {
		if now.Sub(lastRequest) < rateLimitWindow {
			return fmt.Errorf("please wait before requesting another password reset")
		}
	}

	return nil
}

// recordRateLimit records the rate limit timestamp for an email
func (uc *RequestPasswordResetUseCase) recordRateLimit(email string) {
	uc.rateLimiterMu.Lock()
	defer uc.rateLimiterMu.Unlock()

	uc.rateLimiter[email] = biztime.NowUTC()
}

// cleanupExpiredEntries removes entries older than rateLimitCleanupInterval
// Must be called with rateLimiterMu held
func (uc *RequestPasswordResetUseCase) cleanupExpiredEntries(now time.Time) {
	for email, lastRequest := range uc.rateLimiter {
		if now.Sub(lastRequest) > rateLimitCleanupInterval {
			delete(uc.rateLimiter, email)
		}
	}
}
