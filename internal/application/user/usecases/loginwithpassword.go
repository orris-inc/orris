package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SecurityPolicyProvider provides security policy configuration
type SecurityPolicyProvider interface {
	GetSecurityPolicy(ctx context.Context) *user.SecurityPolicy
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type JWTService interface {
	Generate(userUUID string, sessionID string, role authorization.UserRole) (*TokenPair, error)
	Refresh(refreshToken string) (string, error)
}

type LoginWithPasswordCommand struct {
	Email      string
	Password   string
	RememberMe bool
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
	userRepo               user.Repository
	sessionRepo            user.SessionRepository
	passwordHasher         user.PasswordHasher
	jwtService             JWTService
	authHelper             *helpers.AuthHelper
	securityPolicyProvider SecurityPolicyProvider
	sessionConfig          config.SessionConfig
	logger                 logger.Interface
}

func NewLoginWithPasswordUseCase(
	userRepo user.Repository,
	sessionRepo user.SessionRepository,
	hasher user.PasswordHasher,
	jwtService JWTService,
	authHelper *helpers.AuthHelper,
	securityPolicyProvider SecurityPolicyProvider,
	sessionConfig config.SessionConfig,
	logger logger.Interface,
) *LoginWithPasswordUseCase {
	return &LoginWithPasswordUseCase{
		userRepo:               userRepo,
		sessionRepo:            sessionRepo,
		passwordHasher:         hasher,
		jwtService:             jwtService,
		authHelper:             authHelper,
		securityPolicyProvider: securityPolicyProvider,
		sessionConfig:          sessionConfig,
		logger:                 logger,
	}
}

func (uc *LoginWithPasswordUseCase) Execute(ctx context.Context, cmd LoginWithPasswordCommand) (*LoginWithPasswordResult, error) {
	existingUser, err := uc.userRepo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		uc.logger.Errorw("failed to get user by email", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Return generic error if user not found (security: don't reveal if email exists)
	if existingUser == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Validate user can login using unified helper (checks lock, password availability, status)
	if validationErr := uc.authHelper.ValidateUserCanLogin(existingUser); validationErr != nil {
		return nil, validationErr
	}

	// Verify password
	if err := existingUser.VerifyPassword(cmd.Password, uc.passwordHasher); err != nil {
		// Record failed login and save with security policy (non-critical operation)
		var securityPolicy *user.SecurityPolicy
		if uc.securityPolicyProvider != nil {
			securityPolicy = uc.securityPolicyProvider.GetSecurityPolicy(ctx)
		}
		uc.authHelper.RecordFailedLoginWithPolicyAndSave(ctx, existingUser, securityPolicy)
		return nil, fmt.Errorf("invalid email or password")
	}

	// Determine session duration based on remember me option
	sessionDuration := time.Duration(uc.sessionConfig.DefaultExpDays) * 24 * time.Hour
	if cmd.RememberMe {
		sessionDuration = time.Duration(uc.sessionConfig.RememberExpDays) * 24 * time.Hour
	}

	// Create session with tokens using unified helper
	sessionWithTokens, err := uc.authHelper.CreateAndSaveSessionWithTokens(
		existingUser.ID(),
		existingUser.SID(),
		helpers.DeviceInfo{
			DeviceName: cmd.DeviceName,
			DeviceType: cmd.DeviceType,
			IPAddress:  cmd.IPAddress,
			UserAgent:  cmd.UserAgent,
		},
		sessionDuration,
		func(userUUID string, sessionID string) (string, string, int64, error) {
			tokens, err := uc.jwtService.Generate(userUUID, sessionID, existingUser.Role())
			if err != nil {
				return "", "", 0, err
			}
			return tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresIn, nil
		},
	)
	if err != nil {
		return nil, err // Error already logged and wrapped in helper
	}

	// Save user after successful login (reset failed attempts) - non-critical
	uc.authHelper.SaveUserAfterSuccessfulLogin(ctx, existingUser)

	uc.logger.Infow("user logged in successfully", "user_id", existingUser.ID(), "session_id", sessionWithTokens.Session.ID)

	return &LoginWithPasswordResult{
		User:         existingUser,
		AccessToken:  sessionWithTokens.AccessToken,
		RefreshToken: sessionWithTokens.RefreshToken,
		ExpiresIn:    sessionWithTokens.ExpiresIn,
	}, nil
}
