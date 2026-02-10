package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type NotificationFactory interface {
	CreateNotification(userID uint, notificationType, title, content string, relatedID *uint) (Notification, error)
}

type PublishAnnouncementUseCase struct {
	announcementRepo AnnouncementRepository
	markdownService  dto.MarkdownService
	logger           logger.Interface
}

func NewPublishAnnouncementUseCase(
	announcementRepo AnnouncementRepository,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *PublishAnnouncementUseCase {
	return &PublishAnnouncementUseCase{
		announcementRepo: announcementRepo,
		markdownService:  markdownService,
		logger:           logger,
	}
}

func (uc *PublishAnnouncementUseCase) Execute(ctx context.Context, sid string) (*dto.AnnouncementResponse, error) {
	uc.logger.Infow("executing publish announcement use case", "sid", sid)

	announcement, err := uc.announcementRepo.FindBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to find announcement", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to find announcement: %w", err)
	}

	if announcement == nil {
		uc.logger.Warnw("announcement not found", "sid", sid)
		return nil, errors.NewNotFoundError("announcement not found")
	}

	if err := announcement.Publish(); err != nil {
		uc.logger.Errorw("failed to publish announcement", "sid", sid, "error", err)
		return nil, err
	}

	if err := uc.announcementRepo.Update(ctx, announcement); err != nil {
		uc.logger.Errorw("failed to persist announcement publication", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to save published announcement: %w", err)
	}

	response, err := dto.ToAnnouncementResponse(announcement, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcement to response", "error", err)
		return nil, err
	}

	uc.logger.Infow("announcement published successfully", "sid", sid)
	return response, nil
}
