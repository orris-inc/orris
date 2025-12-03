package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetAnnouncementUseCase struct {
	repo            AnnouncementRepository
	markdownService dto.MarkdownService
	logger          logger.Interface
}

func NewGetAnnouncementUseCase(
	repo AnnouncementRepository,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *GetAnnouncementUseCase {
	return &GetAnnouncementUseCase{
		repo:            repo,
		markdownService: markdownService,
		logger:          logger,
	}
}

func (uc *GetAnnouncementUseCase) Execute(ctx context.Context, id uint) (*dto.AnnouncementResponse, error) {
	uc.logger.Infow("executing get announcement use case", "id", id)

	announcement, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to find announcement", "id", id, "error", err)
		return nil, errors.NewNotFoundError("announcement not found")
	}

	announcement.IncrementViewCount()

	if err := uc.repo.Update(ctx, announcement); err != nil {
		uc.logger.Warnw("failed to update view count", "id", id, "error", err)
	}

	response, err := dto.ToAnnouncementResponse(announcement, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcement to response", "error", err)
		return nil, err
	}

	return response, nil
}
