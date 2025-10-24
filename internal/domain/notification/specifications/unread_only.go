package specifications

import (
	"orris/internal/domain/notification"
	vo "orris/internal/domain/notification/value_objects"
)

type UnreadOnly struct{}

func NewUnreadOnly() *UnreadOnly {
	return &UnreadOnly{}
}

func (s *UnreadOnly) IsSatisfiedBy(entity interface{}) bool {
	n, ok := entity.(*notification.Notification)
	if !ok {
		return false
	}
	return n.ReadStatus() == vo.ReadStatusUnread
}
