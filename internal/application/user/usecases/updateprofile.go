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

// UpdateProfileUseCase handles updating a user's profile information
type UpdateProfileUseCase struct {
	userRepo domainUser.Repository
	logger   logger.Interface
}

// NewUpdateProfileUseCase creates a new instance of UpdateProfileUseCase
func NewUpdateProfileUseCase(userRepo domainUser.Repository, logger logger.Interface) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Execute updates the user's profile with the provided information
func (uc *UpdateProfileUseCase) Execute(ctx context.Context, userID uint, request dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	uc.logger.Infow("executing update profile use case", "user_id", userID)

	// Retrieve the existing user
	userEntity, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		uc.logger.Errorw("failed to get user", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if userEntity == nil {
		uc.logger.Warnw("user not found", "user_id", userID)
		return nil, errors.NewNotFoundError("user not found")
	}

	// Update email if provided
	if request.Email != nil && *request.Email != userEntity.Email().String() {
		// Check if email is already in use
		exists, err := uc.userRepo.ExistsByEmail(ctx, *request.Email)
		if err != nil {
			uc.logger.Errorw("failed to check email existence", "email", *request.Email, "error", err)
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
		if exists {
			uc.logger.Warnw("email already in use", "email", *request.Email)
			return nil, errors.NewValidationError("email already in use")
		}

		// Create and update email
		email, err := vo.NewEmail(*request.Email)
		if err != nil {
			uc.logger.Warnw("invalid email format", "email", *request.Email, "error", err)
			return nil, errors.NewValidationError(fmt.Sprintf("invalid email: %v", err))
		}

		if err := userEntity.UpdateEmail(email); err != nil {
			uc.logger.Errorw("failed to update email", "error", err)
			return nil, fmt.Errorf("failed to update email: %w", err)
		}
	}

	// Update name if provided
	if request.Name != nil && *request.Name != userEntity.Name().String() {
		name, err := vo.NewName(*request.Name)
		if err != nil {
			uc.logger.Warnw("invalid name format", "name", *request.Name, "error", err)
			return nil, errors.NewValidationError(fmt.Sprintf("invalid name: %v", err))
		}

		if err := userEntity.UpdateName(name); err != nil {
			uc.logger.Errorw("failed to update name", "error", err)
			return nil, fmt.Errorf("failed to update name: %w", err)
		}
	}

	// Persist the updated user
	if err := uc.userRepo.Update(ctx, userEntity); err != nil {
		uc.logger.Errorw("failed to persist user updates", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to save user updates: %w", err)
	}

	uc.logger.Infow("profile updated successfully", "user_id", userID)

	// Convert to response
	return uc.mapToResponse(userEntity), nil
}

// mapToResponse maps a user entity to a response DTO
func (uc *UpdateProfileUseCase) mapToResponse(userEntity *domainUser.User) *dto.UserResponse {
	displayInfo := userEntity.GetDisplayInfo()

	return &dto.UserResponse{
		ID:          userEntity.ID(),
		Email:       userEntity.Email().String(),
		Name:        userEntity.Name().String(),
		DisplayName: displayInfo.DisplayName,
		Initials:    displayInfo.Initials,
		Role:        string(userEntity.Role()),
		Status:      userEntity.Status().String(),
		CreatedAt:   userEntity.CreatedAt(),
		UpdatedAt:   userEntity.UpdatedAt(),
		Metadata: dto.UserMetadata{
			IsBusinessEmail:      userEntity.IsBusinessEmail(),
			CanPerformActions:    userEntity.CanPerformActions(),
			RequiresVerification: userEntity.RequiresVerification(),
			EmailDomain:          userEntity.Email().Domain(),
		},
	}
}

// ValidateRequest validates the update profile request
func (uc *UpdateProfileUseCase) ValidateRequest(request dto.UpdateProfileRequest) error {
	// At least one field must be provided for update
	if request.Email == nil && request.Name == nil {
		return errors.NewValidationError("at least one field must be provided for update")
	}

	// Validate email if provided
	if request.Email != nil && *request.Email == "" {
		return errors.NewValidationError("email cannot be empty")
	}

	// Validate name if provided
	if request.Name != nil && *request.Name == "" {
		return errors.NewValidationError("name cannot be empty")
	}

	return nil
}
