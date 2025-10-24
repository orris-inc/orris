package usecases

import (
	"context"
	"fmt"

	"orris/internal/shared/logger"
)

type MarkAllAsReadUseCase struct {
	repo   NotificationRepository
	logger logger.Interface
}

func NewMarkAllAsReadUseCase(
	repo NotificationRepository,
	logger logger.Interface,
) *MarkAllAsReadUseCase {
	return &MarkAllAsReadUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *MarkAllAsReadUseCase) Execute(ctx context.Context, userID uint) error {
	uc.logger.Infow("executing mark all notifications as read use case", "user_id", userID)

	if err := uc.repo.MarkAllAsReadByUserID(ctx, userID); err != nil {
		uc.logger.Errorw("failed to mark all notifications as read", "user_id", userID, "error", err)
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	uc.logger.Infow("all notifications marked as read", "user_id", userID)
	return nil
}
