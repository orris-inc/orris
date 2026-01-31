package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/user/dto"
	domainUser "github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// PasswordHasher is the interface for hashing passwords (re-exported from domain)
type PasswordHasher = domainUser.PasswordHasher

// AdminNewUserNotifier is the interface for notifying admins about new users
type AdminNewUserNotifier interface {
	NotifyNewUser(ctx context.Context, cmd AdminNewUserCommand) error
}

// AdminNewUserCommand contains data for new user notification
type AdminNewUserCommand struct {
	UserID    uint
	UserSID   string
	Email     string
	Name      string
	Source    string
	CreatedAt time.Time
}

// CreateUserUseCase handles the business logic for creating a user
type CreateUserUseCase struct {
	userRepo               domainUser.Repository
	passwordHasher         PasswordHasher
	passwordPolicyProvider PasswordPolicyProvider
	adminNotifier          AdminNewUserNotifier // Optional, can be nil
	logger                 logger.Interface
}

// NewCreateUserUseCase creates a new create user use case
func NewCreateUserUseCase(
	userRepo domainUser.Repository,
	passwordHasher PasswordHasher,
	passwordPolicyProvider PasswordPolicyProvider,
	logger logger.Interface,
) *CreateUserUseCase {
	return &CreateUserUseCase{
		userRepo:               userRepo,
		passwordHasher:         passwordHasher,
		passwordPolicyProvider: passwordPolicyProvider,
		logger:                 logger,
	}
}

// SetAdminNotifier sets the admin notifier (optional dependency injection)
func (uc *CreateUserUseCase) SetAdminNotifier(notifier AdminNewUserNotifier) {
	uc.adminNotifier = notifier
}

// Execute executes the create user use case
func (uc *CreateUserUseCase) Execute(ctx context.Context, request dto.CreateUserRequest) (*dto.UserResponse, error) {
	// Log the start of the use case
	uc.logger.Infow("executing create user use case", "email", request.Email)

	// Check if user already exists using GetByEmail
	existingUser, err := uc.userRepo.GetByEmail(ctx, request.Email)
	if err != nil {
		uc.logger.Errorw("database error while checking for existing user", "email", request.Email, "error", err)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil {
		uc.logger.Warnw("user with email already exists", "email", request.Email)
		return nil, errors.NewConflictError("user with this email already exists", request.Email)
	}

	// Create value objects
	email, err := vo.NewEmail(request.Email)
	if err != nil {
		uc.logger.Errorw("invalid email", "error", err)
		return nil, errors.NewValidationError("invalid email", err.Error())
	}

	name, err := vo.NewName(request.Name)
	if err != nil {
		uc.logger.Errorw("invalid name", "error", err)
		return nil, errors.NewValidationError("invalid name", err.Error())
	}

	// Get password policy from settings
	var passwordPolicy *vo.PasswordPolicy
	if uc.passwordPolicyProvider != nil {
		passwordPolicy = uc.passwordPolicyProvider.GetPasswordPolicy(ctx)
	}

	password, err := vo.NewPasswordWithPolicy(request.Password, passwordPolicy)
	if err != nil {
		uc.logger.Errorw("invalid password", "error", err)
		return nil, errors.NewValidationError("invalid password", err.Error())
	}

	// Create user using constructor with SID generator
	userEntity, err := domainUser.NewUser(email, name, id.NewUserID)
	if err != nil {
		uc.logger.Errorw("failed to create user entity", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Set password for the user
	if err := userEntity.SetPassword(password, uc.passwordHasher); err != nil {
		uc.logger.Errorw("failed to set password", "error", err)
		return nil, fmt.Errorf("failed to set password: %w", err)
	}

	// Persist the user
	if err := uc.userRepo.Create(ctx, userEntity); err != nil {
		uc.logger.Errorw("failed to persist user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
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

	uc.logger.Infow("user created successfully", "id", response.ID, "email", response.Email)

	// Notify admins about new user (async, non-blocking)
	if uc.adminNotifier != nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := uc.adminNotifier.NotifyNewUser(notifyCtx, AdminNewUserCommand{
				UserID:    userEntity.ID(),
				UserSID:   userEntity.SID(),
				Email:     userEntity.Email().String(),
				Name:      userEntity.Name().String(),
				Source:    "admin_create",
				CreatedAt: userEntity.CreatedAt(),
			}); err != nil {
				uc.logger.Warnw("failed to notify admins about new user", "user_id", userEntity.ID(), "error", err)
			}
		}()
	}

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
	if request.Password == "" {
		return errors.NewValidationError("password is required")
	}
	return nil
}
