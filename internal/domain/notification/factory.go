package notification

import (
	"time"

	vo "github.com/orris-inc/orris/internal/domain/notification/value_objects"
)

func CreateAnnouncement(
	title string,
	content string,
	announcementType vo.AnnouncementType,
	creatorID uint,
	priority int,
	scheduledAt *time.Time,
	expiresAt *time.Time,
) (*Announcement, error) {
	return NewAnnouncement(title, content, announcementType, creatorID, priority, scheduledAt, expiresAt)
}

func CreateNotification(
	userID uint,
	notificationType vo.NotificationType,
	title string,
	content string,
	relatedID *uint,
) (*Notification, error) {
	return NewNotification(userID, notificationType, title, content, relatedID)
}

func CreateNotificationTemplate(
	templateType vo.TemplateType,
	name string,
	title string,
	content string,
	variables []string,
) (*NotificationTemplate, error) {
	return NewNotificationTemplate(templateType, name, title, content, variables)
}

func CreateSystemNotification(userID uint, title, content string) (*Notification, error) {
	return NewNotification(userID, vo.NotificationTypeSystem, title, content, nil)
}

func CreateActivityNotification(userID uint, title, content string, relatedID uint) (*Notification, error) {
	return NewNotification(userID, vo.NotificationTypeActivity, title, content, &relatedID)
}

func CreateSubscriptionNotification(userID uint, title, content string, relatedID uint) (*Notification, error) {
	return NewNotification(userID, vo.NotificationTypeSubscription, title, content, &relatedID)
}

func CreateSystemAnnouncement(title, content string, creatorID uint, priority int, expiresAt *time.Time) (*Announcement, error) {
	return NewAnnouncement(title, content, vo.AnnouncementTypeSystem, creatorID, priority, nil, expiresAt)
}

func CreateMaintenanceAnnouncement(title, content string, creatorID uint, priority int, scheduledAt, expiresAt *time.Time) (*Announcement, error) {
	return NewAnnouncement(title, content, vo.AnnouncementTypeMaintenance, creatorID, priority, scheduledAt, expiresAt)
}

func CreateEventAnnouncement(title, content string, creatorID uint, priority int, scheduledAt, expiresAt *time.Time) (*Announcement, error) {
	return NewAnnouncement(title, content, vo.AnnouncementTypeEvent, creatorID, priority, scheduledAt, expiresAt)
}
