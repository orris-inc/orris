package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetUnreadCountUseCase struct {
	repo   NotificationRepository
	logger logger.Interface
}

func NewGetUnreadCountUseCase(
	repo NotificationRepository,
	logger logger.Interface,
) *GetUnreadCountUseCase {
	return &GetUnreadCountUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *GetUnreadCountUseCase) Execute(ctx context.Context, userID uint) (*dto.UnreadCountResponse, error) {
	uc.logger.Infow("executing get unread count use case", "user_id", userID)

	count, err := uc.repo.CountUnreadByUserID(ctx, userID)
	if err != nil {
		uc.logger.Errorw("failed to get unread count", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get unread count: %w", err)
	}

	return &dto.UnreadCountResponse{
		Count: count,
	}, nil
}
