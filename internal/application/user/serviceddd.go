package user

import (
	"context"

	"orris/internal/application/user/dto"
	"orris/internal/application/user/usecases"
	domainUser "orris/internal/domain/user"
	"orris/internal/shared/logger"
)

// ServiceDDD is the application service that orchestrates use cases
type ServiceDDD struct {
	createUserUC         *usecases.CreateUserUseCase
	updateUserUC         *usecases.UpdateUserUseCase
	getUserUC            *usecases.GetUserUseCase
	updateProfileUC      *usecases.UpdateProfileUseCase
	changePasswordUC     *usecases.ChangePasswordUseCase
	logger               logger.Interface
}

// NewServiceDDD creates a new DDD application service
func NewServiceDDD(
	userRepo domainUser.Repository,
	sessionRepo domainUser.SessionRepository,
	passwordHasher domainUser.PasswordHasher,
	logger logger.Interface,
) *ServiceDDD {
	return &ServiceDDD{
		createUserUC:         usecases.NewCreateUserUseCase(userRepo, logger),
		updateUserUC:         usecases.NewUpdateUserUseCase(userRepo, logger),
		getUserUC:            usecases.NewGetUserUseCase(userRepo, logger),
		updateProfileUC:      usecases.NewUpdateProfileUseCase(userRepo, logger),
		changePasswordUC:     usecases.NewChangePasswordUseCase(userRepo, sessionRepo, passwordHasher, logger),
		logger:               logger,
	}
}

// CreateUser creates a new user
func (s *ServiceDDD) CreateUser(ctx context.Context, request dto.CreateUserRequest) (*dto.UserResponse, error) {
	if err := s.createUserUC.ValidateRequest(request); err != nil {
		return nil, err
	}
	return s.createUserUC.Execute(ctx, request)
}

// UpdateUser updates an existing user
func (s *ServiceDDD) UpdateUser(ctx context.Context, id uint, request dto.UpdateUserRequest) (*dto.UserResponse, error) {
	if err := s.updateUserUC.ValidateRequest(request); err != nil {
		return nil, err
	}
	return s.updateUserUC.Execute(ctx, id, request)
}

// GetUserByID retrieves a user by ID
func (s *ServiceDDD) GetUserByID(ctx context.Context, id uint) (*dto.UserResponse, error) {
	return s.getUserUC.ExecuteByID(ctx, id)
}

// GetUserByEmail retrieves a user by email
func (s *ServiceDDD) GetUserByEmail(ctx context.Context, email string) (*dto.UserResponse, error) {
	return s.getUserUC.ExecuteByEmail(ctx, email)
}

// ListUsers retrieves a paginated list of users
func (s *ServiceDDD) ListUsers(ctx context.Context, request dto.ListUsersRequest) (*dto.ListUsersResponse, error) {
	return s.getUserUC.ExecuteList(ctx, request)
}

// DeleteUser deletes a user by ID
func (s *ServiceDDD) DeleteUser(ctx context.Context, id uint) error {
	// For now, use the update use case to mark as deleted
	// This could be extracted to a separate DeleteUserUseCase later
	updateRequest := dto.UpdateUserRequest{
		Status: &[]string{"deleted"}[0],
	}
	_, err := s.updateUserUC.Execute(ctx, id, updateRequest)
	return err
}

// UpdateProfile updates the current user's profile (name, email)
func (s *ServiceDDD) UpdateProfile(ctx context.Context, userID uint, request dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	if err := s.updateProfileUC.ValidateRequest(request); err != nil {
		return nil, err
	}
	return s.updateProfileUC.Execute(ctx, userID, request)
}

// ChangePassword changes the current user's password
func (s *ServiceDDD) ChangePassword(ctx context.Context, userID uint, request dto.ChangePasswordRequest) error {
	if err := s.changePasswordUC.ValidateRequest(request); err != nil {
		return err
	}
	return s.changePasswordUC.Execute(ctx, userID, request)
}
