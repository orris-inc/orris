package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/user"
	vo "orris/internal/domain/user/value_objects"
	"orris/internal/shared/authorization"
	"orris/internal/shared/logger"
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
	ExchangeCode(ctx context.Context, code string) (accessToken string, err error)
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
	GetAuthURL(state string) string
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
	roleRepo interface{},
	permissionService interface{},
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
		logger:         logger,
	}
}

func (uc *HandleOAuthCallbackUseCase) Execute(ctx context.Context, cmd HandleOAuthCallbackCommand) (*HandleOAuthCallbackResult, error) {
	if !uc.oauthInitiator.VerifyState(cmd.State) {
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

	accessToken, err := client.ExchangeCode(ctx, cmd.Code)
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

			isFirstUser, err := uc.isFirstUser(ctx)
			if err != nil {
				uc.logger.Errorw("failed to check if first user", "error", err)
			} else if isFirstUser {
				existingUser.SetRole(authorization.RoleAdmin)
				if err := uc.userRepo.Update(ctx, existingUser); err != nil {
					uc.logger.Errorw("failed to update user role to admin", "error", err, "user_id", existingUser.ID())
				} else {
					uc.logger.Infow("admin role assigned to first user", "user_id", existingUser.ID())
				}
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

	session.TokenHash = hashToken(tokens.AccessToken)
	session.RefreshTokenHash = hashToken(tokens.RefreshToken)

	if err := uc.sessionRepo.Create(session); err != nil {
		uc.logger.Errorw("failed to create session in database", "error", err)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	uc.logger.Infow("OAuth login successful", "user_id", existingUser.ID(), "provider", cmd.Provider, "is_new_user", isNewUser)

	return &HandleOAuthCallbackResult{
		User:         existingUser,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		IsNewUser:    isNewUser,
	}, nil
}

func (uc *HandleOAuthCallbackUseCase) isFirstUser(ctx context.Context) (bool, error) {
	filter := user.ListFilter{Page: 1, PageSize: 1}
	_, total, err := uc.userRepo.List(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}
	return total == 1, nil
}
