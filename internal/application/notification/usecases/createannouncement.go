package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type AnnouncementFactory interface {
	CreateAnnouncement(title, content, announcementType string, creatorID uint, priority int) (Announcement, error)
}

type CreateAnnouncementUseCase struct {
	repo            AnnouncementRepository
	factory         AnnouncementFactory
	markdownService dto.MarkdownService
	logger          logger.Interface
}

func NewCreateAnnouncementUseCase(
	repo AnnouncementRepository,
	factory AnnouncementFactory,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *CreateAnnouncementUseCase {
	return &CreateAnnouncementUseCase{
		repo:            repo,
		factory:         factory,
		markdownService: markdownService,
		logger:          logger,
	}
}

func (uc *CreateAnnouncementUseCase) Execute(ctx context.Context, req dto.CreateAnnouncementRequest) (*dto.AnnouncementResponse, error) {
	uc.logger.Infow("executing create announcement use case", "title", req.Title, "creator_id", req.CreatorID)

	announcement, err := uc.factory.CreateAnnouncement(req.Title, req.Content, req.Type, req.CreatorID, req.Priority)
	if err != nil {
		uc.logger.Errorw("failed to create announcement entity", "error", err)
		return nil, errors.NewValidationError(fmt.Sprintf("failed to create announcement: %v", err))
	}

	if err := uc.repo.Create(ctx, announcement); err != nil {
		uc.logger.Errorw("failed to persist announcement", "error", err)
		return nil, fmt.Errorf("failed to save announcement: %w", err)
	}

	response, err := dto.ToAnnouncementResponse(announcement, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcement to response", "error", err)
		return nil, err
	}

	uc.logger.Infow("announcement created successfully", "id", announcement.ID())
	return response, nil
}
