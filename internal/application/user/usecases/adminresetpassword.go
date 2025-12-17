package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type AdminResetPasswordCommand struct {
	UserID      uint
	NewPassword string
}

type AdminResetPasswordUseCase struct {
	userRepo       user.Repository
	sessionRepo    user.SessionRepository
	passwordHasher user.PasswordHasher
	emailService   EmailService
	logger         logger.Interface
}

func NewAdminResetPasswordUseCase(
	userRepo user.Repository,
	sessionRepo user.SessionRepository,
	hasher user.PasswordHasher,
	emailService EmailService,
	logger logger.Interface,
) *AdminResetPasswordUseCase {
	return &AdminResetPasswordUseCase{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		passwordHasher: hasher,
		emailService:   emailService,
		logger:         logger,
	}
}

func (uc *AdminResetPasswordUseCase) Execute(ctx context.Context, cmd AdminResetPasswordCommand) error {
	existingUser, err := uc.userRepo.GetByID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get user", "error", err, "user_id", cmd.UserID)
		return fmt.Errorf("failed to get user: %w", err)
	}
	if existingUser == nil {
		return errors.NewNotFoundError("user not found")
	}

	newPassword, err := vo.NewPassword(cmd.NewPassword)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	if err := existingUser.AdminResetPassword(newPassword, uc.passwordHasher); err != nil {
		uc.logger.Errorw("failed to reset password", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to reset password: %w", err)
	}

	// Invalidate all existing sessions for security
	if err := uc.sessionRepo.DeleteByUserID(existingUser.ID()); err != nil {
		uc.logger.Warnw("failed to delete user sessions", "error", err, "user_id", existingUser.ID())
	}

	if err := uc.userRepo.Update(ctx, existingUser); err != nil {
		uc.logger.Errorw("failed to update user", "error", err, "user_id", existingUser.ID())
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Notify user about password change
	if err := uc.emailService.SendPasswordChangedEmail(existingUser.Email().String()); err != nil {
		uc.logger.Warnw("failed to send password changed email", "error", err, "user_id", existingUser.ID())
	}

	uc.logger.Infow("admin reset password successfully", "user_id", existingUser.ID())

	return nil
}
