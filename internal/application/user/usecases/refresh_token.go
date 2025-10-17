package usecases

import (
	"fmt"

	"orris/internal/domain/user"
	"orris/internal/shared/logger"
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
	logger      logger.Interface
}

func NewRefreshTokenUseCase(
	sessionRepo user.SessionRepository,
	jwtService JWTService,
	logger logger.Interface,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		sessionRepo: sessionRepo,
		jwtService:  jwtService,
		logger:      logger,
	}
}

func (uc *RefreshTokenUseCase) Execute(cmd RefreshTokenCommand) (*RefreshTokenResult, error) {
	refreshTokenHash := hashToken(cmd.RefreshToken)

	session, err := uc.sessionRepo.GetByTokenHash(refreshTokenHash)
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

	session.TokenHash = hashToken(newAccessToken)
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
