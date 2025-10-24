package usecases

import (
	"context"

	"orris/internal/application/notification/dto"
	"orris/internal/shared/logger"
)

type ListNotificationsUseCase struct {
	repo            NotificationRepository
	markdownService dto.MarkdownService
	logger          logger.Interface
}

func NewListNotificationsUseCase(
	repo NotificationRepository,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *ListNotificationsUseCase {
	return &ListNotificationsUseCase{
		repo:            repo,
		markdownService: markdownService,
		logger:          logger,
	}
}

func (uc *ListNotificationsUseCase) Execute(ctx context.Context, req dto.ListNotificationsRequest) (*dto.ListResponse, error) {
	uc.logger.Infow("executing list notifications use case", "user_id", req.UserID, "status", req.Status)

	var notifications []Notification
	var total int64
	var err error

	if req.Status == "unread" {
		notifications, total, err = uc.repo.FindUnreadByUserID(ctx, req.UserID, req.Limit, req.Offset)
	} else {
		notifications, total, err = uc.repo.FindByUserID(ctx, req.UserID, req.Limit, req.Offset)
	}

	if err != nil {
		uc.logger.Errorw("failed to list notifications", "user_id", req.UserID, "error", err)
		return nil, err
	}

	responses, err := dto.ToNotificationResponseList(notifications, uc.markdownService)
	if err != nil {
		uc.logger.Errorw("failed to convert notifications to responses", "error", err)
		return nil, err
	}

	return &dto.ListResponse{
		Items:  responses,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}
