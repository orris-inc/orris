package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type DeleteNotificationUseCase struct {
	repo   NotificationRepository
	logger logger.Interface
}

func NewDeleteNotificationUseCase(
	repo NotificationRepository,
	logger logger.Interface,
) *DeleteNotificationUseCase {
	return &DeleteNotificationUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *DeleteNotificationUseCase) Execute(ctx context.Context, id uint, userID uint) error {
	uc.logger.Infow("executing delete notification use case", "id", id, "user_id", userID)

	notification, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to find notification", "id", id, "error", err)
		return errors.NewNotFoundError("notification not found")
	}

	if notification.UserID() != userID {
		uc.logger.Warnw("unauthorized access to notification", "id", id, "user_id", userID, "owner_id", notification.UserID())
		return errors.NewForbiddenError("you don't have permission to access this notification")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.logger.Errorw("failed to delete notification", "id", id, "error", err)
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	uc.logger.Infow("notification deleted successfully", "id", id)
	return nil
}
