package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ResetPasswordCommand struct {
	Token       string
	NewPassword string
}

type ResetPasswordUseCase struct {
	userRepo               user.Repository
	sessionRepo            user.SessionRepository
	passwordHasher         user.PasswordHasher
	emailService           EmailService
	passwordPolicyProvider PasswordPolicyProvider
	logger                 logger.Interface
}

func NewResetPasswordUseCase(
	userRepo user.Repository,
	sessionRepo user.SessionRepository,
	hasher user.PasswordHasher,
	emailService EmailService,
	passwordPolicyProvider PasswordPolicyProvider,
	logger logger.Interface,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		userRepo:               userRepo,
		sessionRepo:            sessionRepo,
		passwordHasher:         hasher,
		emailService:           emailService,
		passwordPolicyProvider: passwordPolicyProvider,
		logger:                 logger,
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

	// Get password policy from settings
	var passwordPolicy *vo.PasswordPolicy
	if uc.passwordPolicyProvider != nil {
		passwordPolicy = uc.passwordPolicyProvider.GetPasswordPolicy(ctx)
	}

	newPassword, err := vo.NewPasswordWithPolicy(cmd.NewPassword, passwordPolicy)
	if err != nil {
		return err
	}

	if err := existingUser.ResetPassword(cmd.Token, newPassword, uc.passwordHasher); err != nil {
		uc.logger.Errorw("failed to reset password", "error", err, "user_id", existingUser.ID())
		return err
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
