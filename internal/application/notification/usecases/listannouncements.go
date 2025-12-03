package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ListAnnouncementsUseCase struct {
	repo            AnnouncementRepository
	markdownService dto.MarkdownService
	logger          logger.Interface
}

func NewListAnnouncementsUseCase(
	repo AnnouncementRepository,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *ListAnnouncementsUseCase {
	return &ListAnnouncementsUseCase{
		repo:            repo,
		markdownService: markdownService,
		logger:          logger,
	}
}

func (uc *ListAnnouncementsUseCase) Execute(ctx context.Context, limit, offset int) (*dto.ListResponse, error) {
	uc.logger.Infow("executing list announcements use case", "limit", limit, "offset", offset)

	announcements, total, err := uc.repo.FindAll(ctx, limit, offset)
	if err != nil {
		uc.logger.Errorw("failed to list announcements", "error", err)
		return nil, err
	}

	responses, err := dto.ToAnnouncementResponseList(announcements, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcements to responses", "error", err)
		return nil, err
	}

	return &dto.ListResponse{
		Items:  responses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (uc *ListAnnouncementsUseCase) ExecutePublished(ctx context.Context, limit, offset int) (*dto.ListResponse, error) {
	uc.logger.Infow("executing list published announcements use case", "limit", limit, "offset", offset)

	announcements, total, err := uc.repo.FindPublished(ctx, limit, offset)
	if err != nil {
		uc.logger.Errorw("failed to list published announcements", "error", err)
		return nil, err
	}

	responses, err := dto.ToAnnouncementResponseList(announcements, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcements to responses", "error", err)
		return nil, err
	}

	return &dto.ListResponse{
		Items:  responses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
