package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ArchiveAnnouncementUseCase struct {
	repo            AnnouncementRepository
	markdownService dto.MarkdownService
	logger          logger.Interface
}

func NewArchiveAnnouncementUseCase(
	repo AnnouncementRepository,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *ArchiveAnnouncementUseCase {
	return &ArchiveAnnouncementUseCase{
		repo:            repo,
		markdownService: markdownService,
		logger:          logger,
	}
}

func (uc *ArchiveAnnouncementUseCase) Execute(ctx context.Context, sid string) (*dto.AnnouncementResponse, error) {
	uc.logger.Infow("executing archive announcement use case", "sid", sid)

	announcement, err := uc.repo.FindBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to find announcement by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to find announcement: %w", err)
	}

	if announcement == nil {
		uc.logger.Warnw("announcement not found", "sid", sid)
		return nil, errors.NewNotFoundError("announcement not found")
	}

	if err := announcement.Archive(); err != nil {
		uc.logger.Errorw("failed to archive announcement", "sid", sid, "error", err)
		return nil, errors.NewValidationError(fmt.Sprintf("failed to archive announcement: %v", err))
	}

	if err := uc.repo.Update(ctx, announcement); err != nil {
		uc.logger.Errorw("failed to update archived announcement", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to save archived announcement: %w", err)
	}

	response, err := dto.ToAnnouncementResponse(announcement, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcement to response", "sid", sid, "error", err)
		return nil, err
	}

	uc.logger.Infow("announcement archived successfully", "sid", sid)
	return response, nil
}
