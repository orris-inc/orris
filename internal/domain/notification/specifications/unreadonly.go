package specifications

import (
	"github.com/orris-inc/orris/internal/domain/notification"
	vo "github.com/orris-inc/orris/internal/domain/notification/valueobjects"
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
