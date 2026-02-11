package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type RefreshTokenCommand struct {
	RefreshToken string
}

type RefreshTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	RememberMe   bool
}

type RefreshTokenUseCase struct {
	userRepo      user.Repository
	sessionRepo   user.SessionRepository
	jwtService    JWTService
	authHelper    *helpers.AuthHelper
	sessionConfig config.SessionConfig
	logger        logger.Interface
}

func NewRefreshTokenUseCase(
	userRepo user.Repository,
	sessionRepo user.SessionRepository,
	jwtService JWTService,
	authHelper *helpers.AuthHelper,
	sessionConfig config.SessionConfig,
	logger logger.Interface,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		userRepo:      userRepo,
		sessionRepo:   sessionRepo,
		jwtService:    jwtService,
		authHelper:    authHelper,
		sessionConfig: sessionConfig,
		logger:        logger,
	}
}

func (uc *RefreshTokenUseCase) Execute(ctx context.Context, cmd RefreshTokenCommand) (*RefreshTokenResult, error) {
	refreshTokenHash := uc.authHelper.HashToken(cmd.RefreshToken)

	session, err := uc.sessionRepo.GetByRefreshTokenHash(refreshTokenHash)
	if err != nil {
		uc.logger.Errorw("failed to get session", "error", err)
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	if session.IsExpired() {
		return nil, fmt.Errorf("session has expired")
	}

	// Validate user status - ensure user is still active before issuing new token
	existingUser, err := uc.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get user", "error", err, "user_id", session.UserID)
		return nil, fmt.Errorf("failed to validate user")
	}
	if existingUser == nil {
		uc.logger.Warnw("user not found during token refresh", "user_id", session.UserID)
		return nil, fmt.Errorf("user not found")
	}

	// Check if user can still perform actions (not suspended, inactive, or deleted)
	if validationErr := uc.authHelper.ValidateUserCanPerformAction(existingUser); validationErr != nil {
		uc.logger.Warnw("user cannot perform actions during token refresh",
			"user_id", session.UserID,
			"status", existingUser.Status(),
			"error", validationErr.Message,
		)
		return nil, fmt.Errorf("account is not active")
	}

	tokenPair, err := uc.jwtService.Refresh(cmd.RefreshToken)
	if err != nil {
		uc.logger.Errorw("failed to refresh token", "error", err)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update both access and refresh token hashes (refresh token rotation)
	uc.authHelper.SetSessionTokens(session, tokenPair.AccessToken, tokenPair.RefreshToken)
	session.UpdateActivity()

	// Extend session expiration on refresh to keep active users logged in
	// Use remember or default duration based on session's remember me setting
	expDays := uc.sessionConfig.DefaultExpDays
	if session.RememberMe {
		expDays = uc.sessionConfig.RememberExpDays
	}
	sessionDuration := time.Duration(expDays) * 24 * time.Hour
	session.ExtendExpiration(sessionDuration)

	if err := uc.sessionRepo.Update(session); err != nil {
		uc.logger.Errorw("failed to update session", "error", err)
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	uc.logger.Debugw("token refreshed successfully", "user_id", session.UserID, "session_id", session.ID)

	return &RefreshTokenResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		RememberMe:   session.RememberMe,
	}, nil
}
