package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type MarkAnnouncementAsReadUseCase struct {
	announcementRepo     AnnouncementRepository
	userAnnouncementRepo UserAnnouncementReadRepository
	logger               logger.Interface
}

func NewMarkAnnouncementAsReadUseCase(
	announcementRepo AnnouncementRepository,
	userAnnouncementRepo UserAnnouncementReadRepository,
	logger logger.Interface,
) *MarkAnnouncementAsReadUseCase {
	return &MarkAnnouncementAsReadUseCase{
		announcementRepo:     announcementRepo,
		userAnnouncementRepo: userAnnouncementRepo,
		logger:               logger,
	}
}

// Execute marks a specific announcement as read for the user.
func (uc *MarkAnnouncementAsReadUseCase) Execute(ctx context.Context, userID uint, sid string) error {
	uc.logger.Infow("marking announcement as read", "user_id", userID, "sid", sid)

	// Verify announcement exists and is published
	announcement, err := uc.announcementRepo.FindBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to find announcement", "sid", sid, "error", err)
		return fmt.Errorf("failed to find announcement: %w", err)
	}

	if announcement == nil {
		uc.logger.Warnw("announcement not found", "sid", sid)
		return errors.NewNotFoundError("announcement not found")
	}

	// Only allow marking published announcements as read
	if announcement.Status() != "published" {
		uc.logger.Warnw("cannot mark non-published announcement as read", "sid", sid, "status", announcement.Status())
		return errors.NewValidationError("can only mark published announcements as read")
	}

	// Mark as read
	if err := uc.userAnnouncementRepo.MarkAsRead(ctx, userID, announcement.ID()); err != nil {
		uc.logger.Errorw("failed to mark announcement as read", "user_id", userID, "announcement_id", announcement.ID(), "error", err)
		return fmt.Errorf("failed to mark announcement as read: %w", err)
	}

	uc.logger.Infow("announcement marked as read", "user_id", userID, "sid", sid)
	return nil
}

// GetReadAnnouncementIDs returns the list of announcement IDs that the user has individually marked as read.
// Deprecated: Use GetReadStatusByIDs for better performance.
func (uc *MarkAnnouncementAsReadUseCase) GetReadAnnouncementIDs(ctx context.Context, userID uint) ([]uint, error) {
	return uc.userAnnouncementRepo.GetReadAnnouncementIDs(ctx, userID)
}

// GetReadStatusByIDs checks which announcements from the given list have been read by the user.
func (uc *MarkAnnouncementAsReadUseCase) GetReadStatusByIDs(ctx context.Context, userID uint, announcementIDs []uint) (map[uint]bool, error) {
	return uc.userAnnouncementRepo.GetReadStatusByIDs(ctx, userID, announcementIDs)
}
