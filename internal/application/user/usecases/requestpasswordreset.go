package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/user"
	"orris/internal/shared/logger"
)

type RequestPasswordResetCommand struct {
	Email string
}

type RequestPasswordResetUseCase struct {
	userRepo     user.Repository
	emailService EmailService
	logger       logger.Interface
	rateLimiter  map[string]time.Time
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
	}
}

func (uc *RequestPasswordResetUseCase) Execute(ctx context.Context, cmd RequestPasswordResetCommand) error {
	if lastRequest, exists := uc.rateLimiter[cmd.Email]; exists {
		if time.Since(lastRequest) < 1*time.Minute {
			return fmt.Errorf("please wait before requesting another password reset")
		}
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

	uc.rateLimiter[cmd.Email] = time.Now()

	uc.logger.Infow("password reset requested", "user_id", existingUser.ID())

	return nil
}
