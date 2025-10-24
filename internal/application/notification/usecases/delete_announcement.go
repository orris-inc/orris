package usecases

import (
	"context"
	"fmt"

	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type DeleteAnnouncementUseCase struct {
	repo   AnnouncementRepository
	logger logger.Interface
}

func NewDeleteAnnouncementUseCase(
	repo AnnouncementRepository,
	logger logger.Interface,
) *DeleteAnnouncementUseCase {
	return &DeleteAnnouncementUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *DeleteAnnouncementUseCase) Execute(ctx context.Context, id uint) error {
	uc.logger.Infow("executing delete announcement use case", "id", id)

	announcement, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to find announcement", "id", id, "error", err)
		return errors.NewNotFoundError("announcement not found")
	}

	if err := announcement.Archive(); err != nil {
		uc.logger.Errorw("failed to archive announcement", "id", id, "error", err)
		return fmt.Errorf("failed to archive announcement: %w", err)
	}

	if err := uc.repo.Update(ctx, announcement); err != nil {
		uc.logger.Errorw("failed to persist announcement deletion", "id", id, "error", err)
		return fmt.Errorf("failed to delete announcement: %w", err)
	}

	uc.logger.Infow("announcement deleted successfully", "id", id)
	return nil
}
