package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/notification/dto"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type UpdateAnnouncementUseCase struct {
	repo            AnnouncementRepository
	markdownService dto.MarkdownService
	logger          logger.Interface
}

func NewUpdateAnnouncementUseCase(
	repo AnnouncementRepository,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *UpdateAnnouncementUseCase {
	return &UpdateAnnouncementUseCase{
		repo:            repo,
		markdownService: markdownService,
		logger:          logger,
	}
}

func (uc *UpdateAnnouncementUseCase) Execute(ctx context.Context, id uint, req dto.UpdateAnnouncementRequest) (*dto.AnnouncementResponse, error) {
	uc.logger.Infow("executing update announcement use case", "id", id)

	announcement, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to find announcement", "id", id, "error", err)
		return nil, errors.NewNotFoundError("announcement not found")
	}

	if err := uc.repo.Update(ctx, announcement); err != nil {
		uc.logger.Errorw("failed to update announcement", "id", id, "error", err)
		return nil, fmt.Errorf("failed to update announcement: %w", err)
	}

	response, err := dto.ToAnnouncementResponse(announcement, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcement to response", "error", err)
		return nil, err
	}

	uc.logger.Infow("announcement updated successfully", "id", id)
	return response, nil
}
