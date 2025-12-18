package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

type NotificationTemplateModel struct {
	ID           uint   `gorm:"primaryKey"`
	TemplateType string `gorm:"size:50;not null;uniqueIndex"`
	Name         string `gorm:"size:100;not null"`
	Title        string `gorm:"size:255;not null"`
	Content      string `gorm:"type:longtext;not null"`
	Variables    string `gorm:"type:json"`
	Enabled      bool   `gorm:"default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (NotificationTemplateModel) TableName() string {
	return constants.TableNotificationTemplates
}
