package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/user/dto"
	domainUser "github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/value_objects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ChangePasswordUseCase handles changing a user's password
type ChangePasswordUseCase struct {
	userRepo       domainUser.Repository
	sessionRepo    domainUser.SessionRepository
	passwordHasher domainUser.PasswordHasher
	logger         logger.Interface
}

// NewChangePasswordUseCase creates a new instance of ChangePasswordUseCase
func NewChangePasswordUseCase(
	userRepo domainUser.Repository,
	sessionRepo domainUser.SessionRepository,
	passwordHasher domainUser.PasswordHasher,
	logger logger.Interface,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		passwordHasher: passwordHasher,
		logger:         logger,
	}
}

// Execute changes the user's password
func (uc *ChangePasswordUseCase) Execute(ctx context.Context, userID uint, request dto.ChangePasswordRequest) error {
	uc.logger.Infow("executing change password use case", "user_id", userID)

	// Retrieve the existing user
	userEntity, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		uc.logger.Errorw("failed to get user", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get user: %w", err)
	}
	if userEntity == nil {
		uc.logger.Warnw("user not found", "user_id", userID)
		return errors.NewNotFoundError("user not found")
	}

	// Validate old password
	oldPassword, err := vo.NewPassword(request.OldPassword)
	if err != nil {
		uc.logger.Warnw("invalid old password format", "error", err)
		return errors.NewValidationError("invalid old password format")
	}

	// Validate new password
	newPassword, err := vo.NewPassword(request.NewPassword)
	if err != nil {
		uc.logger.Warnw("invalid new password format", "error", err)
		return errors.NewValidationError(fmt.Sprintf("invalid new password: %v", err))
	}

	// Change password (includes verification of old password)
	if err := userEntity.ChangePassword(oldPassword, newPassword, uc.passwordHasher); err != nil {
		uc.logger.Warnw("failed to change password", "user_id", userID, "error", err)
		return errors.NewValidationError(err.Error())
	}

	// Persist the updated user
	if err := uc.userRepo.Update(ctx, userEntity); err != nil {
		uc.logger.Errorw("failed to persist user updates", "user_id", userID, "error", err)
		return fmt.Errorf("failed to save user updates: %w", err)
	}

	// Optionally logout all devices (if requested)
	if request.LogoutAllDevices {
		if err := uc.sessionRepo.DeleteByUserID(userID); err != nil {
			// Log error but don't fail the operation
			uc.logger.Warnw("failed to delete user sessions", "user_id", userID, "error", err)
		} else {
			uc.logger.Infow("logged out all devices", "user_id", userID)
		}
	}

	uc.logger.Infow("password changed successfully", "user_id", userID)

	return nil
}

// ValidateRequest validates the change password request
func (uc *ChangePasswordUseCase) ValidateRequest(request dto.ChangePasswordRequest) error {
	// Validate old password is provided
	if request.OldPassword == "" {
		return errors.NewValidationError("old password is required")
	}

	// Validate new password is provided
	if request.NewPassword == "" {
		return errors.NewValidationError("new password is required")
	}

	// Validate password meets minimum requirements
	if len(request.NewPassword) < 8 {
		return errors.NewValidationError("new password must be at least 8 characters long")
	}

	// Validate new password is different from old password
	if request.OldPassword == request.NewPassword {
		return errors.NewValidationError("new password must be different from old password")
	}

	return nil
}
