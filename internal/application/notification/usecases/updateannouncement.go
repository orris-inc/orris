package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
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

func (uc *UpdateAnnouncementUseCase) Execute(ctx context.Context, sid string, req dto.UpdateAnnouncementRequest) (*dto.AnnouncementResponse, error) {
	uc.logger.Infow("executing update announcement use case", "sid", sid)

	announcement, err := uc.repo.FindBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to find announcement", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to find announcement: %w", err)
	}

	if announcement == nil {
		uc.logger.Warnw("announcement not found", "sid", sid)
		return nil, errors.NewNotFoundError("announcement not found")
	}

	// Apply updates from request, using current values as defaults
	title := announcement.Title()
	if req.Title != nil {
		title = *req.Title
	}

	content := announcement.Content()
	if req.Content != nil {
		content = *req.Content
	}

	priority := announcement.Priority()
	if req.Priority != nil {
		priority = *req.Priority
	}

	expiresAt := announcement.ExpiresAt()
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt
	}

	if err := announcement.Update(title, content, priority, expiresAt); err != nil {
		uc.logger.Errorw("failed to apply updates to announcement", "sid", sid, "error", err)
		return nil, errors.NewValidationError(fmt.Sprintf("failed to update announcement: %v", err))
	}

	if err := uc.repo.Update(ctx, announcement); err != nil {
		uc.logger.Errorw("failed to update announcement", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to update announcement: %w", err)
	}

	response, err := dto.ToAnnouncementResponse(announcement, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcement to response", "error", err)
		return nil, err
	}

	uc.logger.Infow("announcement updated successfully", "sid", sid)
	return response, nil
}
