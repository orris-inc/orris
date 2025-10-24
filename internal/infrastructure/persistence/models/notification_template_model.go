package models

import (
	"time"

	"gorm.io/gorm"
)

type NotificationTemplateModel struct {
	ID           uint   `gorm:"primaryKey"`
	TemplateType string `gorm:"size:50;not null;uniqueIndex"`
	Name         string `gorm:"size:100;not null"`
	Title        string `gorm:"size:255;not null"`
	Content      string `gorm:"type:longtext;not null"`
	Variables    string `gorm:"type:json"`
	Enabled      bool   `gorm:"default:true"`
	Version      int    `gorm:"not null;default:1"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (NotificationTemplateModel) TableName() string {
	return "notification_templates"
}

func (n *NotificationTemplateModel) BeforeCreate(tx *gorm.DB) error {
	if n.Version == 0 {
		n.Version = 1
	}
	return nil
}

func (n *NotificationTemplateModel) BeforeUpdate(tx *gorm.DB) error {
	tx.Statement.SetColumn("version", n.Version+1)
	return nil
}
