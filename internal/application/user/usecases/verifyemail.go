package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/user"
	"orris/internal/shared/logger"
)

type VerifyEmailCommand struct {
	Token string
}

type VerifyEmailUseCase struct {
	userRepo user.Repository
	logger   logger.Interface
}

func NewVerifyEmailUseCase(userRepo user.Repository, logger logger.Interface) *VerifyEmailUseCase {
	return &VerifyEmailUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (uc *VerifyEmailUseCase) Execute(ctx context.Context, cmd VerifyEmailCommand) error {
	existingUser, err := uc.userRepo.GetByVerificationToken(ctx, cmd.Token)
	if err != nil {
		uc.logger.Errorw("failed to get user by verification token", "error", err)
		return fmt.Errorf("invalid or expired verification token")
	}
	if existingUser == nil {
		return fmt.Errorf("invalid or expired verification token")
	}

	if err := existingUser.VerifyEmail(cmd.Token); err != nil {
		uc.logger.Errorw("failed to verify email", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to verify email: %w", err)
	}

	if err := existingUser.Activate(); err != nil {
		uc.logger.Errorw("failed to activate user", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to activate user: %w", err)
	}

	if err := uc.userRepo.Update(ctx, existingUser); err != nil {
		uc.logger.Errorw("failed to update user", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to update user: %w", err)
	}

	uc.logger.Infow("email verified successfully", "user_id", existingUser.ID())

	return nil
}
