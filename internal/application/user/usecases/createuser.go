package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/user/dto"
	"orris/internal/domain/shared/events"
	domainUser "orris/internal/domain/user"
	"orris/internal/domain/user/specifications"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// CreateUserUseCase handles the business logic for creating a user
type CreateUserUseCase struct {
	userFactory     *domainUser.UserFactory
	userRepo        domainUser.RepositoryWithSpecifications
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

// NewCreateUserUseCase creates a new create user use case
func NewCreateUserUseCase(
	userRepo domainUser.RepositoryWithSpecifications,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *CreateUserUseCase {
	return &CreateUserUseCase{
		userFactory:     domainUser.NewUserFactory(),
		userRepo:        userRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

// Execute executes the create user use case
func (uc *CreateUserUseCase) Execute(ctx context.Context, request dto.CreateUserRequest) (*dto.UserResponse, error) {
	// Log the start of the use case
	uc.logger.Infow("executing create user use case", "email", request.Email)

	// Check if user already exists using specification
	emailSpec := specifications.NewEmailSpecification(request.Email)
	existingUsers, err := uc.userRepo.FindBySpecification(ctx, emailSpec, 1)
	if err != nil {
		uc.logger.Errorw("database error while checking for existing user", "email", request.Email, "error", err)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if len(existingUsers) > 0 {
		uc.logger.Warnw("user with email already exists", "email", request.Email)
		return nil, errors.NewConflictError("user with this email already exists", request.Email)
	}

	// Create user using factory
	userEntity, err := uc.userFactory.CreateUser(request.Email, request.Name)
	if err != nil {
		uc.logger.Errorw("failed to create user entity", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Apply business rules via domain service if needed
	// For example, auto-activate users from trusted domains
	if userEntity.IsBusinessEmail() {
		if err := userEntity.Activate(); err != nil {
			uc.logger.Warnw("failed to auto-activate business user", "email", request.Email, "error", err)
			// Don't fail the operation, just log the warning
		}
	}

	// Persist the user
	if err := uc.userRepo.Create(ctx, userEntity); err != nil {
		uc.logger.Errorw("failed to persist user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	// Map to response DTO
	response := &dto.UserResponse{
		ID:        userEntity.ID(),
		Email:     userEntity.Email().String(),
		Name:      userEntity.Name().String(),
		Status:    userEntity.Status().String(),
		CreatedAt: userEntity.CreatedAt(),
		UpdatedAt: userEntity.UpdatedAt(),
	}

	uc.logger.Infow("user created successfully", "id", response.ID, "email", response.Email)
	return response, nil
}

// ValidateRequest validates the create user request
func (uc *CreateUserUseCase) ValidateRequest(request dto.CreateUserRequest) error {
	if request.Email == "" {
		return errors.NewValidationError("email is required")
	}
	if request.Name == "" {
		return errors.NewValidationError("name is required")
	}
	return nil
}
