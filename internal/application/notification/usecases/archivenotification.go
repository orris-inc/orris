package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ArchiveNotificationUseCase struct {
	repo   NotificationRepository
	logger logger.Interface
}

func NewArchiveNotificationUseCase(
	repo NotificationRepository,
	logger logger.Interface,
) *ArchiveNotificationUseCase {
	return &ArchiveNotificationUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *ArchiveNotificationUseCase) Execute(ctx context.Context, id uint, userID uint) error {
	uc.logger.Infow("executing archive notification use case", "id", id, "user_id", userID)

	notification, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to find notification", "id", id, "error", err)
		return errors.NewNotFoundError("notification not found")
	}

	if notification.UserID() != userID {
		uc.logger.Warnw("unauthorized access to notification", "id", id, "user_id", userID, "owner_id", notification.UserID())
		return errors.NewForbiddenError("you don't have permission to access this notification")
	}

	if err := notification.Archive(); err != nil {
		uc.logger.Errorw("failed to archive notification", "id", id, "error", err)
		return fmt.Errorf("failed to archive notification: %w", err)
	}

	if err := uc.repo.Update(ctx, notification); err != nil {
		uc.logger.Errorw("failed to persist notification archival", "id", id, "error", err)
		return fmt.Errorf("failed to save notification: %w", err)
	}

	uc.logger.Infow("notification archived successfully", "id", id)
	return nil
}
