package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/value_objects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ResetPasswordCommand struct {
	Token       string
	NewPassword string
}

type ResetPasswordUseCase struct {
	userRepo       user.Repository
	sessionRepo    user.SessionRepository
	passwordHasher user.PasswordHasher
	emailService   EmailService
	logger         logger.Interface
}

func NewResetPasswordUseCase(
	userRepo user.Repository,
	sessionRepo user.SessionRepository,
	hasher user.PasswordHasher,
	emailService EmailService,
	logger logger.Interface,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		passwordHasher: hasher,
		emailService:   emailService,
		logger:         logger,
	}
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, cmd ResetPasswordCommand) error {
	existingUser, err := uc.userRepo.GetByPasswordResetToken(ctx, cmd.Token)
	if err != nil {
		uc.logger.Errorw("failed to get user by reset token", "error", err)
		return fmt.Errorf("invalid or expired reset token")
	}
	if existingUser == nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	newPassword, err := vo.NewPassword(cmd.NewPassword)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	if err := existingUser.ResetPassword(cmd.Token, newPassword, uc.passwordHasher); err != nil {
		uc.logger.Errorw("failed to reset password", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to reset password: %w", err)
	}

	if err := uc.sessionRepo.DeleteByUserID(existingUser.ID()); err != nil {
		uc.logger.Warnw("failed to delete user sessions", "error", err, "user_id", existingUser.ID())
	}

	if err := uc.userRepo.Update(ctx, existingUser); err != nil {
		uc.logger.Errorw("failed to update user", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to update user: %w", err)
	}

	if err := uc.emailService.SendPasswordChangedEmail(existingUser.Email().String()); err != nil {
		uc.logger.Warnw("failed to send password changed email", "error", err, "user_id", existingUser.ID())
	}

	uc.logger.Infow("password reset successfully", "user_id", existingUser.ID())

	return nil
}
