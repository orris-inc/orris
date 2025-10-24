package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/notification/dto"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type NotificationFactory interface {
	CreateNotification(userID uint, notificationType, title, content string, relatedID *uint) (Notification, error)
}

type PublishAnnouncementUseCase struct {
	announcementRepo AnnouncementRepository
	notificationRepo NotificationRepository
	userRepo         UserRepository
	notificationFactory NotificationFactory
	markdownService  dto.MarkdownService
	logger           logger.Interface
}

func NewPublishAnnouncementUseCase(
	announcementRepo AnnouncementRepository,
	notificationRepo NotificationRepository,
	userRepo UserRepository,
	notificationFactory NotificationFactory,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *PublishAnnouncementUseCase {
	return &PublishAnnouncementUseCase{
		announcementRepo:    announcementRepo,
		notificationRepo:    notificationRepo,
		userRepo:            userRepo,
		notificationFactory: notificationFactory,
		markdownService:     markdownService,
		logger:              logger,
	}
}

func (uc *PublishAnnouncementUseCase) Execute(ctx context.Context, id uint, req dto.PublishAnnouncementRequest) (*dto.AnnouncementResponse, error) {
	uc.logger.Infow("executing publish announcement use case", "id", id, "send_notification", req.SendNotification)

	announcement, err := uc.announcementRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to find announcement", "id", id, "error", err)
		return nil, errors.NewNotFoundError("announcement not found")
	}

	if err := announcement.Publish(); err != nil {
		uc.logger.Errorw("failed to publish announcement", "id", id, "error", err)
		return nil, fmt.Errorf("failed to publish announcement: %w", err)
	}

	if err := uc.announcementRepo.Update(ctx, announcement); err != nil {
		uc.logger.Errorw("failed to persist announcement publication", "id", id, "error", err)
		return nil, fmt.Errorf("failed to save published announcement: %w", err)
	}

	if req.SendNotification {
		if err := uc.createNotificationsForAllUsers(ctx, announcement); err != nil {
			uc.logger.Errorw("failed to create notifications", "id", id, "error", err)
		}
	}

	response, err := dto.ToAnnouncementResponse(announcement, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert announcement to response", "error", err)
		return nil, err
	}

	uc.logger.Infow("announcement published successfully", "id", id)
	return response, nil
}

func (uc *PublishAnnouncementUseCase) createNotificationsForAllUsers(ctx context.Context, announcement Announcement) error {
	userIDs, err := uc.userRepo.FindAllActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch user IDs: %w", err)
	}

	if len(userIDs) == 0 {
		uc.logger.Infow("no active users to notify")
		return nil
	}

	notifications := make([]Notification, 0, len(userIDs))
	announcementID := announcement.ID()

	for _, userID := range userIDs {
		notification, err := uc.notificationFactory.CreateNotification(
			userID,
			"announcement",
			announcement.Title(),
			announcement.Content(),
			&announcementID,
		)
		if err != nil {
			uc.logger.Warnw("failed to create notification for user", "user_id", userID, "error", err)
			continue
		}
		notifications = append(notifications, notification)
	}

	if len(notifications) == 0 {
		return fmt.Errorf("no notifications were created")
	}

	if err := uc.notificationRepo.BulkCreate(ctx, notifications); err != nil {
		return fmt.Errorf("failed to bulk create notifications: %w", err)
	}

	uc.logger.Infow("notifications created for announcement", "announcement_id", announcement.ID(), "count", len(notifications))
	return nil
}
