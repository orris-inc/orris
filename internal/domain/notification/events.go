package notification

import "time"

type AnnouncementPublishedEvent struct {
	AnnouncementID   uint
	SendNotification bool
	PublishedAt      time.Time
}

type NotificationCreatedEvent struct {
	NotificationID uint
	UserID         uint
	CreatedAt      time.Time
}
