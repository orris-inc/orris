package usecases

import (
	"context"
	"fmt"

	permissionApp "orris/internal/application/permission"
	"orris/internal/domain/permission"
	"orris/internal/domain/user"
	vo "orris/internal/domain/user/value_objects"
	"orris/internal/shared/logger"
)

type EmailService interface {
	SendVerificationEmail(to, token string) error
	SendPasswordResetEmail(to, token string) error
	SendPasswordChangedEmail(to string) error
}

type RegisterWithPasswordCommand struct {
	Email    string
	Name     string
	Password string
}

type RegisterWithPasswordUseCase struct {
	userRepo          user.Repository
	permissionService *permissionApp.Service
	roleRepo          permission.RoleRepository
	passwordHasher    user.PasswordHasher
	emailService      EmailService
	logger            logger.Interface
}

func NewRegisterWithPasswordUseCase(
	userRepo user.Repository,
	roleRepo permission.RoleRepository,
	hasher user.PasswordHasher,
	emailService EmailService,
	permissionService *permissionApp.Service,
	logger logger.Interface,
) *RegisterWithPasswordUseCase {
	return &RegisterWithPasswordUseCase{
		userRepo:          userRepo,
		roleRepo:          roleRepo,
		permissionService: permissionService,
		passwordHasher:    hasher,
		emailService:      emailService,
		logger:            logger,
	}
}

func (uc *RegisterWithPasswordUseCase) Execute(ctx context.Context, cmd RegisterWithPasswordCommand) (*user.User, error) {
	email, err := vo.NewEmail(cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	exists, err := uc.userRepo.ExistsByEmail(ctx, email.String())
	if err != nil {
		uc.logger.Errorw("failed to check email existence", "error", err)
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user with email %s already exists", cmd.Email)
	}

	name, err := vo.NewName(cmd.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}

	password, err := vo.NewPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	newUser, err := user.NewUser(email, name)
	if err != nil {
		uc.logger.Errorw("failed to create user aggregate", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := newUser.SetPassword(password, uc.passwordHasher); err != nil {
		uc.logger.Errorw("failed to set password", "error", err)
		return nil, fmt.Errorf("failed to set password: %w", err)
	}

	token, err := newUser.GenerateEmailVerificationToken()
	if err != nil {
		uc.logger.Errorw("failed to generate verification token", "error", err)
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		uc.logger.Errorw("failed to create user in database", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	isFirstUser, err := uc.isFirstUser(ctx)
	if err != nil {
		uc.logger.Errorw("failed to check if first user", "error", err)
	} else if isFirstUser {
		if err := uc.assignAdminRoleToFirstUser(ctx, newUser.ID()); err != nil {
			uc.logger.Errorw("failed to assign admin role to first user", "error", err, "user_id", newUser.ID())
		} else {
			uc.logger.Infow("admin role assigned to first user", "user_id", newUser.ID())
		}
	}

	if err := uc.emailService.SendVerificationEmail(email.String(), token.Value()); err != nil {
		uc.logger.Warnw("failed to send verification email", "error", err, "email", email.String())
	}

	uc.logger.Infow("user registered successfully", "user_id", newUser.ID(), "email", email.String())

	return newUser, nil
}

func (uc *RegisterWithPasswordUseCase) isFirstUser(ctx context.Context) (bool, error) {
	filter := user.ListFilter{Page: 1, PageSize: 1}
	_, total, err := uc.userRepo.List(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}
	return total == 1, nil
}

func (uc *RegisterWithPasswordUseCase) assignAdminRoleToFirstUser(ctx context.Context, userID uint) error {
	adminRole, err := uc.roleRepo.GetBySlug(ctx, "admin")
	if err != nil {
		return fmt.Errorf("failed to get admin role: %w", err)
	}

	if err := uc.permissionService.AssignRoleToUser(ctx, userID, []uint{adminRole.ID()}); err != nil {
		return fmt.Errorf("failed to assign admin role: %w", err)
	}

	return nil
}
