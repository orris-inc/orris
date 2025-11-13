package migrations

import (
	"orris/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

// CreateNotificationsTable creates the notifications table
func CreateNotificationsTable(db *gorm.DB) error {
	return db.AutoMigrate(&models.NotificationModel{})
}

// CreateAnnouncementsTable creates the announcements table
func CreateAnnouncementsTable(db *gorm.DB) error {
	return db.AutoMigrate(&models.AnnouncementModel{})
}

// CreateNotificationTemplatesTable creates the notification_templates table
func CreateNotificationTemplatesTable(db *gorm.DB) error {
	return db.AutoMigrate(&models.NotificationTemplateModel{})
}
