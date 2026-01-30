package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

type UserAnnouncementReadRepository struct {
	db *gorm.DB
}

func NewUserAnnouncementReadRepository(db *gorm.DB) *UserAnnouncementReadRepository {
	return &UserAnnouncementReadRepository{db: db}
}

// MarkAsRead marks an announcement as read for a user.
// If already marked, it updates the read_at timestamp.
func (r *UserAnnouncementReadRepository) MarkAsRead(ctx context.Context, userID, announcementID uint) error {
	now := time.Now().UTC()
	model := &models.UserAnnouncementReadModel{
		UserID:         userID,
		AnnouncementID: announcementID,
		ReadAt:         now,
		CreatedAt:      now,
	}

	// Use ON DUPLICATE KEY UPDATE to handle upsert
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND announcement_id = ?", userID, announcementID).
		FirstOrCreate(model)

	if result.Error != nil {
		return fmt.Errorf("failed to mark announcement as read: %w", result.Error)
	}

	return nil
}

// IsRead checks if an announcement has been read by a user.
func (r *UserAnnouncementReadRepository) IsRead(ctx context.Context, userID, announcementID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.UserAnnouncementReadModel{}).
		Where("user_id = ? AND announcement_id = ?", userID, announcementID).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check if announcement is read: %w", err)
	}

	return count > 0, nil
}

// GetReadAnnouncementIDs returns all announcement IDs that have been read by a user.
// Deprecated: Use GetReadStatusByIDs for better performance when checking specific announcements.
func (r *UserAnnouncementReadRepository) GetReadAnnouncementIDs(ctx context.Context, userID uint) ([]uint, error) {
	var reads []models.UserAnnouncementReadModel
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&reads).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get read announcement IDs: %w", err)
	}

	ids := make([]uint, len(reads))
	for i, read := range reads {
		ids[i] = read.AnnouncementID
	}

	return ids, nil
}

// GetReadStatusByIDs checks which announcements from the given list have been read by the user.
// This is more efficient than GetReadAnnouncementIDs when checking a specific set of announcements.
func (r *UserAnnouncementReadRepository) GetReadStatusByIDs(ctx context.Context, userID uint, announcementIDs []uint) (map[uint]bool, error) {
	if len(announcementIDs) == 0 {
		return make(map[uint]bool), nil
	}

	var reads []models.UserAnnouncementReadModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND announcement_id IN ?", userID, announcementIDs).
		Find(&reads).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get read status: %w", err)
	}

	result := make(map[uint]bool, len(reads))
	for _, read := range reads {
		result[read.AnnouncementID] = true
	}

	return result, nil
}

// CountUnreadByUser counts announcements that are:
// 1. Published and not expired
// 2. Not individually marked as read
// 3. Published after user's announcements_read_at (if set)
func (r *UserAnnouncementReadRepository) CountUnreadByUser(
	ctx context.Context,
	userID uint,
	userReadAt *time.Time,
) (int64, error) {
	now := time.Now().UTC()

	// Subquery to get announcement IDs that user has individually read
	subQuery := r.db.Model(&models.UserAnnouncementReadModel{}).
		Select("announcement_id").
		Where("user_id = ?", userID)

	query := r.db.WithContext(ctx).
		Model(&models.AnnouncementModel{}).
		Where("status = ?", "published").
		Where("expires_at IS NULL OR expires_at > ?", now).
		Where("id NOT IN (?)", subQuery)

	// If user has a global read timestamp, only count announcements published after that
	if userReadAt != nil {
		query = query.Where("updated_at > ?", userReadAt.UTC())
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count unread announcements: %w", err)
	}

	return count, nil
}
