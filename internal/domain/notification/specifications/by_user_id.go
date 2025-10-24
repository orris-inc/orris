package specifications

import "orris/internal/domain/notification"

type ByUserID struct {
	UserID uint
}

func NewByUserID(userID uint) *ByUserID {
	return &ByUserID{
		UserID: userID,
	}
}

func (s *ByUserID) IsSatisfiedBy(entity interface{}) bool {
	n, ok := entity.(*notification.Notification)
	if !ok {
		return false
	}
	return n.UserID() == s.UserID
}
