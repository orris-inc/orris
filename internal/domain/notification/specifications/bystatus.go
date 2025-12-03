package specifications

import (
	"github.com/orris-inc/orris/internal/domain/notification"
	vo "github.com/orris-inc/orris/internal/domain/notification/value_objects"
)

type ByStatus struct {
	Status vo.AnnouncementStatus
}

func NewByStatus(status vo.AnnouncementStatus) *ByStatus {
	return &ByStatus{
		Status: status,
	}
}

func (s *ByStatus) IsSatisfiedBy(entity interface{}) bool {
	a, ok := entity.(*notification.Announcement)
	if !ok {
		return false
	}
	return a.Status() == s.Status
}
