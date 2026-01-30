package models

import (
	"time"

	"github.com/orris-inc/orris/internal/shared/constants"
)

type UserAnnouncementReadModel struct {
	ID             uint      `gorm:"primaryKey"`
	UserID         uint      `gorm:"not null;index"`
	AnnouncementID uint      `gorm:"not null;index"`
	ReadAt         time.Time `gorm:"not null"`
	CreatedAt      time.Time
}

func (UserAnnouncementReadModel) TableName() string {
	return constants.TableUserAnnouncementReads
}
