package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/user/dto"
	domainUser "github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateUserUseCase handles the business logic for updating a user
type UpdateUserUseCase struct {
	userRepo domainUser.Repository
	logger   logger.Interface
}

// NewUpdateUserUseCase creates a new update user use case
func NewUpdateUserUseCase(
	userRepo domainUser.Repository,
	logger logger.Interface,
) *UpdateUserUseCase {
	return &UpdateUserUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Execute executes the update user use case
func (uc *UpdateUserUseCase) Execute(ctx context.Context, sid string, request dto.UpdateUserRequest) (*dto.UserResponse, error) {
	// Log the start of the use case
	uc.logger.Infow("executing update user use case", "sid", sid)

	// Retrieve the existing user
	userEntity, err := uc.userRepo.GetBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to get user", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if userEntity == nil {
		uc.logger.Warnw("user not found", "sid", sid)
		return nil, errors.NewNotFoundError("user not found")
	}

	// Update email if provided
	if request.Email != nil && *request.Email != userEntity.Email().String() {
		// Check if new email already exists
		existingUser, err := uc.userRepo.GetByEmail(ctx, *request.Email)
		if err != nil {
			uc.logger.Errorw("failed to check email existence", "email", *request.Email, "error", err)
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
		if existingUser != nil && existingUser.SID() != sid {
			return nil, errors.NewConflictError("email already in use", *request.Email)
		}

		// Create new email value object
		emailVO, err := vo.NewEmail(*request.Email)
		if err != nil {
			return nil, errors.NewValidationError(fmt.Sprintf("invalid email: %v", err))
		}

		// Update email in domain
		if err := userEntity.UpdateEmail(emailVO); err != nil {
			uc.logger.Errorw("failed to update email in domain", "error", err)
			return nil, fmt.Errorf("failed to update email: %w", err)
		}
	}

	// Update name if provided
	if request.Name != nil && *request.Name != userEntity.Name().String() {
		// Create new name value object
		nameVO, err := vo.NewName(*request.Name)
		if err != nil {
			return nil, errors.NewValidationError(fmt.Sprintf("invalid name: %v", err))
		}

		// Update name in domain
		if err := userEntity.UpdateName(nameVO); err != nil {
			uc.logger.Errorw("failed to update name in domain", "error", err)
			return nil, fmt.Errorf("failed to update name: %w", err)
		}
	}

	// Update status if provided
	if request.Status != nil {
		newStatus, err := vo.NewStatus(*request.Status)
		if err != nil {
			return nil, errors.NewValidationError(fmt.Sprintf("invalid status: %v", err))
		}

		// Apply status transition based on business rules
		switch *newStatus {
		case vo.StatusActive:
			if err := userEntity.Activate(); err != nil {
				return nil, errors.NewValidationError(fmt.Sprintf("cannot activate user: %v", err))
			}
		case vo.StatusInactive:
			if err := userEntity.Deactivate("User manually deactivated"); err != nil {
				return nil, errors.NewValidationError(fmt.Sprintf("cannot deactivate user: %v", err))
			}
		case vo.StatusSuspended:
			if err := userEntity.Suspend("User manually suspended"); err != nil {
				return nil, errors.NewValidationError(fmt.Sprintf("cannot suspend user: %v", err))
			}
		case vo.StatusDeleted:
			if err := userEntity.Delete(); err != nil {
				return nil, errors.NewValidationError(fmt.Sprintf("cannot delete user: %v", err))
			}
		default:
			return nil, errors.NewValidationError(fmt.Sprintf("unsupported status transition to: %s", newStatus))
		}
	}

	// Update role if provided
	if request.Role != nil {
		var newRole authorization.UserRole
		switch *request.Role {
		case constants.RoleUser:
			newRole = authorization.RoleUser
		case constants.RoleAdmin:
			newRole = authorization.RoleAdmin
		default:
			return nil, errors.NewValidationError(fmt.Sprintf("invalid role: %s", *request.Role))
		}
		userEntity.SetRole(newRole)
	}

	// Persist the updated user
	if err := uc.userRepo.Update(ctx, userEntity); err != nil {
		uc.logger.Errorw("failed to persist user updates", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to save user updates: %w", err)
	}

	// Map to response DTO with external SID
	displayInfo := userEntity.GetDisplayInfo()
	response := &dto.UserResponse{
		ID:          userEntity.SID(),
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

	uc.logger.Infow("user updated successfully", "id", response.ID)
	return response, nil
}

// ValidateRequest validates the update user request
func (uc *UpdateUserUseCase) ValidateRequest(request dto.UpdateUserRequest) error {
	// At least one field must be provided for update
	if request.Email == nil && request.Name == nil && request.Status == nil && request.Role == nil {
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

	// Validate status if provided
	if request.Status != nil {
		if _, err := vo.NewStatus(*request.Status); err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid status: %v", err))
		}
	}

	// Validate role if provided
	if request.Role != nil {
		if *request.Role != constants.RoleUser && *request.Role != constants.RoleAdmin {
			return errors.NewValidationError(fmt.Sprintf("invalid role: %s", *request.Role))
		}
	}

	return nil
}
