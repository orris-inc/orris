package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

type NotificationModel struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"not null;index:idx_user_read"`
	Type       string `gorm:"size:50;not null"`
	Title      string `gorm:"size:255;not null"`
	Content    string `gorm:"type:longtext;not null"`
	RelatedID  *uint
	ReadStatus string `gorm:"size:20;not null;default:'unread';index:idx_user_read"`
	ArchivedAt *time.Time
	CreatedAt  time.Time `gorm:"index"`
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (NotificationModel) TableName() string {
	return constants.TableNotifications
}

func (n *NotificationModel) BeforeCreate(tx *gorm.DB) error {
	if n.ReadStatus == "" {
		n.ReadStatus = "unread"
	}
	return nil
}
