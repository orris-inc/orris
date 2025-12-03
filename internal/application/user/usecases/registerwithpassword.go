package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/value_objects"
	"github.com/orris-inc/orris/internal/shared/logger"
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
	userRepo       user.Repository
	passwordHasher user.PasswordHasher
	emailService   EmailService
	authHelper     *helpers.AuthHelper
	logger         logger.Interface
}

func NewRegisterWithPasswordUseCase(
	userRepo user.Repository,
	hasher user.PasswordHasher,
	emailService EmailService,
	authHelper *helpers.AuthHelper,
	logger logger.Interface,
) *RegisterWithPasswordUseCase {
	return &RegisterWithPasswordUseCase{
		userRepo:       userRepo,
		passwordHasher: hasher,
		emailService:   emailService,
		authHelper:     authHelper,
		logger:         logger,
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

	// Grant admin role to first user if applicable
	if err := uc.authHelper.GrantAdminAndSave(ctx, newUser); err != nil {
		uc.logger.Warnw("failed to grant admin role to first user", "error", err, "user_id", newUser.ID())
		// Continue despite error as user is already created
	}

	if err := uc.emailService.SendVerificationEmail(email.String(), token.Value()); err != nil {
		uc.logger.Warnw("failed to send verification email", "error", err, "email", email.String())
	}

	uc.logger.Infow("user registered successfully", "user_id", newUser.ID(), "email", email.String())

	return newUser, nil
}
