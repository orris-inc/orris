package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/application/user/helpers"
	"orris/internal/domain/user"
	"orris/internal/shared/logger"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type JWTService interface {
	Generate(userID uint, sessionID string) (*TokenPair, error)
	Refresh(refreshToken string) (string, error)
}

type LoginWithPasswordCommand struct {
	Email      string
	Password   string
	DeviceName string
	DeviceType string
	IPAddress  string
	UserAgent  string
}

type LoginWithPasswordResult struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type LoginWithPasswordUseCase struct {
	userRepo       user.Repository
	sessionRepo    user.SessionRepository
	passwordHasher user.PasswordHasher
	jwtService     JWTService
	authHelper     *helpers.AuthHelper
	logger         logger.Interface
}

func NewLoginWithPasswordUseCase(
	userRepo user.Repository,
	sessionRepo user.SessionRepository,
	hasher user.PasswordHasher,
	jwtService JWTService,
	authHelper *helpers.AuthHelper,
	logger logger.Interface,
) *LoginWithPasswordUseCase {
	return &LoginWithPasswordUseCase{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		passwordHasher: hasher,
		jwtService:     jwtService,
		authHelper:     authHelper,
		logger:         logger,
	}
}

func (uc *LoginWithPasswordUseCase) Execute(ctx context.Context, cmd LoginWithPasswordCommand) (*LoginWithPasswordResult, error) {
	existingUser, err := uc.userRepo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		uc.logger.Errorw("failed to get user by email", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if existingUser == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if existingUser.IsLocked() {
		return nil, fmt.Errorf("account is temporarily locked due to too many failed login attempts")
	}

	if !existingUser.HasPassword() {
		return nil, fmt.Errorf("password login not available for this account")
	}

	if err := existingUser.VerifyPassword(cmd.Password, uc.passwordHasher); err != nil {
		if updateErr := uc.userRepo.Update(ctx, existingUser); updateErr != nil {
			uc.logger.Errorw("failed to update user after failed login", "error", updateErr)
		}
		return nil, fmt.Errorf("invalid email or password")
	}

	if !existingUser.CanPerformActions() {
		return nil, fmt.Errorf("account is not active")
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	session, err := user.NewSession(
		existingUser.ID(),
		cmd.DeviceName,
		cmd.DeviceType,
		cmd.IPAddress,
		cmd.UserAgent,
		expiresAt,
	)
	if err != nil {
		uc.logger.Errorw("failed to create session", "error", err)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	tokens, err := uc.jwtService.Generate(existingUser.ID(), session.ID)
	if err != nil {
		uc.logger.Errorw("failed to generate JWT tokens", "error", err)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	session.TokenHash = uc.authHelper.HashToken(tokens.AccessToken)
	session.RefreshTokenHash = uc.authHelper.HashToken(tokens.RefreshToken)

	if err := uc.sessionRepo.Create(session); err != nil {
		uc.logger.Errorw("failed to create session in database", "error", err)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	if err := uc.userRepo.Update(ctx, existingUser); err != nil {
		uc.logger.Errorw("failed to update user", "error", err)
	}

	uc.logger.Infow("user logged in successfully", "user_id", existingUser.ID(), "session_id", session.ID)

	return &LoginWithPasswordResult{
		User:         existingUser,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}
