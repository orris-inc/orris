package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
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

func (uc *DeleteAnnouncementUseCase) Execute(ctx context.Context, sid string) error {
	uc.logger.Infow("executing delete announcement use case", "sid", sid)

	announcement, err := uc.repo.FindBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to find announcement", "sid", sid, "error", err)
		return fmt.Errorf("failed to find announcement: %w", err)
	}

	if announcement == nil {
		uc.logger.Warnw("announcement not found", "sid", sid)
		return errors.NewNotFoundError("announcement not found")
	}

	if err := announcement.Archive(); err != nil {
		uc.logger.Errorw("failed to archive announcement", "sid", sid, "error", err)
		return fmt.Errorf("failed to archive announcement: %w", err)
	}

	if err := uc.repo.Update(ctx, announcement); err != nil {
		uc.logger.Errorw("failed to persist announcement deletion", "sid", sid, "error", err)
		return fmt.Errorf("failed to delete announcement: %w", err)
	}

	uc.logger.Infow("announcement deleted successfully", "sid", sid)
	return nil
}
