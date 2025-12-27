package specifications

import (
	"github.com/orris-inc/orris/internal/domain/notification"
	vo "github.com/orris-inc/orris/internal/domain/notification/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

type ActiveAnnouncements struct{}

func NewActiveAnnouncements() *ActiveAnnouncements {
	return &ActiveAnnouncements{}
}

func (s *ActiveAnnouncements) IsSatisfiedBy(entity interface{}) bool {
	a, ok := entity.(*notification.Announcement)
	if !ok {
		return false
	}

	if a.Status() != vo.AnnouncementStatusPublished {
		return false
	}

	now := biztime.NowUTC()

	if scheduledAt := a.ScheduledAt(); scheduledAt != nil && now.Before(*scheduledAt) {
		return false
	}

	if expiresAt := a.ExpiresAt(); expiresAt != nil && now.After(*expiresAt) {
		return false
	}

	return true
}
