package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type OAuthUserInfo struct {
	Email         string
	Name          string
	Picture       string
	EmailVerified bool
	Provider      string
	ProviderID    string
}

type OAuthCallbackClient interface {
	ExchangeCode(ctx context.Context, code string, codeVerifier string) (accessToken string, err error)
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
	GetAuthURL(state string) (authURL string, codeVerifier string, err error)
}

type HandleOAuthCallbackCommand struct {
	Provider   string
	Code       string
	State      string
	DeviceName string
	DeviceType string
	IPAddress  string
	UserAgent  string
}

type HandleOAuthCallbackResult struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	IsNewUser    bool
}

type HandleOAuthCallbackUseCase struct {
	userRepo       user.Repository
	oauthRepo      user.OAuthAccountRepository
	sessionRepo    user.SessionRepository
	googleClient   OAuthCallbackClient
	githubClient   OAuthCallbackClient
	jwtService     JWTService
	oauthInitiator *InitiateOAuthLoginUseCase
	authHelper     *helpers.AuthHelper
	sessionConfig  config.SessionConfig
	logger         logger.Interface
}

func NewHandleOAuthCallbackUseCase(
	userRepo user.Repository,
	oauthRepo user.OAuthAccountRepository,
	sessionRepo user.SessionRepository,
	googleClient OAuthCallbackClient,
	githubClient OAuthCallbackClient,
	jwtService JWTService,
	oauthInitiator *InitiateOAuthLoginUseCase,
	authHelper *helpers.AuthHelper,
	sessionConfig config.SessionConfig,
	logger logger.Interface,
) *HandleOAuthCallbackUseCase {
	return &HandleOAuthCallbackUseCase{
		userRepo:       userRepo,
		oauthRepo:      oauthRepo,
		sessionRepo:    sessionRepo,
		googleClient:   googleClient,
		githubClient:   githubClient,
		jwtService:     jwtService,
		oauthInitiator: oauthInitiator,
		authHelper:     authHelper,
		sessionConfig:  sessionConfig,
		logger:         logger,
	}
}

func (uc *HandleOAuthCallbackUseCase) Execute(ctx context.Context, cmd HandleOAuthCallbackCommand) (*HandleOAuthCallbackResult, error) {
	// Verify state and retrieve code_verifier from Redis
	stateInfo, err := uc.oauthInitiator.VerifyStateAndGetVerifier(ctx, cmd.State)
	if err != nil {
		uc.logger.Warnw("state verification failed",
			"provider", cmd.Provider,
			"error", err,
		)
		return nil, fmt.Errorf("invalid or expired state parameter")
	}

	var client OAuthCallbackClient
	switch cmd.Provider {
	case "google":
		client = uc.googleClient
	case "github":
		client = uc.githubClient
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", cmd.Provider)
	}

	// Exchange authorization code for access token using code_verifier
	accessToken, err := client.ExchangeCode(ctx, cmd.Code, stateInfo.CodeVerifier)
	if err != nil {
		uc.logger.Errorw("failed to exchange code", "error", err, "provider", cmd.Provider)
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	userInfo, err := client.GetUserInfo(ctx, accessToken)
	if err != nil {
		uc.logger.Errorw("failed to get user info", "error", err, "provider", cmd.Provider)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	oauthAccount, err := uc.oauthRepo.GetByProviderAndUserID(cmd.Provider, userInfo.ProviderID)
	if err != nil {
		uc.logger.Errorw("failed to get oauth account", "error", err)
		return nil, fmt.Errorf("failed to get oauth account: %w", err)
	}

	var existingUser *user.User
	isNewUser := false

	if oauthAccount != nil {
		existingUser, err = uc.userRepo.GetByID(ctx, oauthAccount.UserID)
		if err != nil {
			uc.logger.Errorw("failed to get user", "error", err, "user_id", oauthAccount.UserID)
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		oauthAccount.RecordLogin()
		if updateErr := uc.oauthRepo.Update(oauthAccount); updateErr != nil {
			uc.logger.Warnw("failed to update oauth account", "error", updateErr)
		}
	} else {
		existingUser, err = uc.userRepo.GetByEmail(ctx, userInfo.Email)
		if err != nil {
			uc.logger.Errorw("failed to get user by email", "error", err)
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}

		if existingUser == nil {
			email, err := vo.NewEmail(userInfo.Email)
			if err != nil {
				return nil, fmt.Errorf("invalid email: %w", err)
			}

			name, err := vo.NewName(userInfo.Name)
			if err != nil {
				return nil, fmt.Errorf("invalid name: %w", err)
			}

			existingUser, err = user.NewUser(email, name)
			if err != nil {
				uc.logger.Errorw("failed to create user", "error", err)
				return nil, fmt.Errorf("failed to create user: %w", err)
			}

			if userInfo.EmailVerified {
				if err := existingUser.Activate(); err != nil {
					uc.logger.Warnw("failed to activate user", "error", err)
				}
			}

			if err := uc.userRepo.Create(ctx, existingUser); err != nil {
				uc.logger.Errorw("failed to create user in database", "error", err)
				return nil, fmt.Errorf("failed to create user: %w", err)
			}

			isNewUser = true

			// Grant admin role to first user if applicable
			if err := uc.authHelper.GrantAdminAndSave(ctx, existingUser); err != nil {
				uc.logger.Warnw("failed to grant admin role to first user", "error", err, "user_id", existingUser.ID())
				// Continue despite error as user is already created
			}
		}

		newOAuthAccount, err := user.NewOAuthAccount(existingUser.ID(), cmd.Provider, userInfo.ProviderID, userInfo.Email)
		if err != nil {
			uc.logger.Errorw("failed to create oauth account", "error", err)
			return nil, fmt.Errorf("failed to create oauth account: %w", err)
		}

		newOAuthAccount.ProviderUsername = userInfo.Name
		newOAuthAccount.ProviderAvatarURL = userInfo.Picture

		if err := uc.oauthRepo.Create(newOAuthAccount); err != nil {
			uc.logger.Errorw("failed to create oauth account in database", "error", err)
			return nil, fmt.Errorf("failed to create oauth account: %w", err)
		}
	}

	// Validate user can perform actions using unified helper (checks account status)
	if validationErr := uc.authHelper.ValidateUserCanPerformAction(existingUser); validationErr != nil {
		return nil, validationErr
	}

	// OAuth login uses remember me duration by default (user convenience)
	sessionDuration := time.Duration(uc.sessionConfig.RememberExpDays) * 24 * time.Hour

	// Create session with tokens using unified helper
	sessionWithTokens, err := uc.authHelper.CreateAndSaveSessionWithTokens(
		existingUser.ID(),
		helpers.DeviceInfo{
			DeviceName: cmd.DeviceName,
			DeviceType: cmd.DeviceType,
			IPAddress:  cmd.IPAddress,
			UserAgent:  cmd.UserAgent,
		},
		sessionDuration,
		func(userID uint, sessionID string) (string, string, int64, error) {
			tokens, err := uc.jwtService.Generate(userID, sessionID, existingUser.Role())
			if err != nil {
				return "", "", 0, err
			}
			return tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresIn, nil
		},
	)
	if err != nil {
		return nil, err // Error already logged and wrapped in helper
	}

	uc.logger.Infow("OAuth login successful", "user_id", existingUser.ID(), "provider", cmd.Provider, "is_new_user", isNewUser)

	return &HandleOAuthCallbackResult{
		User:         existingUser,
		AccessToken:  sessionWithTokens.AccessToken,
		RefreshToken: sessionWithTokens.RefreshToken,
		ExpiresIn:    sessionWithTokens.ExpiresIn,
		IsNewUser:    isNewUser,
	}, nil
}
