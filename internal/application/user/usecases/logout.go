package usecases

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type LogoutCommand struct {
	SessionID string
}

type LogoutUseCase struct {
	sessionRepo user.SessionRepository
	logger      logger.Interface
}

func NewLogoutUseCase(sessionRepo user.SessionRepository, logger logger.Interface) *LogoutUseCase {
	return &LogoutUseCase{
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

func (uc *LogoutUseCase) Execute(cmd LogoutCommand) error {
	if err := uc.sessionRepo.Delete(cmd.SessionID); err != nil {
		uc.logger.Errorw("failed to delete session", "error", err, "session_id", cmd.SessionID)
		return fmt.Errorf("failed to logout: %w", err)
	}

	uc.logger.Infow("user logged out successfully", "session_id", cmd.SessionID)

	return nil
}
