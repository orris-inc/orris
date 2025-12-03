package usecases

import (
	"fmt"

	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type RefreshTokenCommand struct {
	RefreshToken string
}

type RefreshTokenResult struct {
	AccessToken string
	ExpiresIn   int64
}

type RefreshTokenUseCase struct {
	sessionRepo user.SessionRepository
	jwtService  JWTService
	authHelper  *helpers.AuthHelper
	logger      logger.Interface
}

func NewRefreshTokenUseCase(
	sessionRepo user.SessionRepository,
	jwtService JWTService,
	authHelper *helpers.AuthHelper,
	logger logger.Interface,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		sessionRepo: sessionRepo,
		jwtService:  jwtService,
		authHelper:  authHelper,
		logger:      logger,
	}
}

func (uc *RefreshTokenUseCase) Execute(cmd RefreshTokenCommand) (*RefreshTokenResult, error) {
	refreshTokenHash := uc.authHelper.HashToken(cmd.RefreshToken)

	session, err := uc.sessionRepo.GetByRefreshTokenHash(refreshTokenHash)
	if err != nil {
		uc.logger.Errorw("failed to get session", "error", err)
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	if session.IsExpired() {
		return nil, fmt.Errorf("session has expired")
	}

	newAccessToken, err := uc.jwtService.Refresh(cmd.RefreshToken)
	if err != nil {
		uc.logger.Errorw("failed to refresh token", "error", err)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	uc.authHelper.UpdateSessionAccessToken(session, newAccessToken)
	session.UpdateActivity()

	if err := uc.sessionRepo.Update(session); err != nil {
		uc.logger.Errorw("failed to update session", "error", err)
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	uc.logger.Infow("token refreshed successfully", "user_id", session.UserID, "session_id", session.ID)

	return &RefreshTokenResult{
		AccessToken: newAccessToken,
		ExpiresIn:   900,
	}, nil
}
