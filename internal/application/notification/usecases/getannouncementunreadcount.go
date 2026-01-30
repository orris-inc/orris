package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetAnnouncementUnreadCountUseCase struct {
	userAnnouncementRepo UserAnnouncementReadRepository
	logger               logger.Interface
}

func NewGetAnnouncementUnreadCountUseCase(
	userAnnouncementRepo UserAnnouncementReadRepository,
	logger logger.Interface,
) *GetAnnouncementUnreadCountUseCase {
	return &GetAnnouncementUnreadCountUseCase{
		userAnnouncementRepo: userAnnouncementRepo,
		logger:               logger,
	}
}

// Execute returns the count of unread announcements for the user.
// An announcement is considered unread if:
// 1. It is published and not expired
// 2. It was not individually marked as read by the user
// 3. It was published after user's global announcements_read_at timestamp (if set)
func (uc *GetAnnouncementUnreadCountUseCase) Execute(ctx context.Context, userID uint, userReadAt *time.Time) (int64, error) {
	count, err := uc.userAnnouncementRepo.CountUnreadByUser(ctx, userID, userReadAt)
	if err != nil {
		uc.logger.Errorw("failed to count unread announcements", "user_id", userID, "error", err)
		return 0, err
	}
	return count, nil
}
