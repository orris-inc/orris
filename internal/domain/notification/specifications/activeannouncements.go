package specifications

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/notification"
	vo "github.com/orris-inc/orris/internal/domain/notification/value_objects"
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

	now := time.Now()

	if scheduledAt := a.ScheduledAt(); scheduledAt != nil && now.Before(*scheduledAt) {
		return false
	}

	if expiresAt := a.ExpiresAt(); expiresAt != nil && now.After(*expiresAt) {
		return false
	}

	return true
}
