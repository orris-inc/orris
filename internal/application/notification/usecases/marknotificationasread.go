package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type MarkNotificationAsReadUseCase struct {
	repo   NotificationRepository
	logger logger.Interface
}

func NewMarkNotificationAsReadUseCase(
	repo NotificationRepository,
	logger logger.Interface,
) *MarkNotificationAsReadUseCase {
	return &MarkNotificationAsReadUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *MarkNotificationAsReadUseCase) Execute(ctx context.Context, id uint, userID uint) error {
	uc.logger.Infow("executing mark notification as read use case", "id", id, "user_id", userID)

	notification, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to find notification", "id", id, "error", err)
		return errors.NewNotFoundError("notification not found")
	}

	if notification.UserID() != userID {
		uc.logger.Warnw("unauthorized access to notification", "id", id, "user_id", userID, "owner_id", notification.UserID())
		return errors.NewForbiddenError("you don't have permission to access this notification")
	}

	if err := notification.MarkAsRead(); err != nil {
		uc.logger.Errorw("failed to mark notification as read", "id", id, "error", err)
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	if err := uc.repo.Update(ctx, notification); err != nil {
		uc.logger.Errorw("failed to persist notification update", "id", id, "error", err)
		return fmt.Errorf("failed to save notification: %w", err)
	}

	uc.logger.Infow("notification marked as read", "id", id)
	return nil
}
