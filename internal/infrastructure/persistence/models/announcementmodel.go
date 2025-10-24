package models

import (
	"time"

	"gorm.io/gorm"
)

type AnnouncementModel struct {
	ID          uint   `gorm:"primaryKey"`
	Title       string `gorm:"size:255;not null"`
	Content     string `gorm:"type:longtext;not null"`
	Type        string `gorm:"size:50;not null;default:'system'"`
	Status      string `gorm:"size:50;not null;default:'draft';index"`
	CreatorID   uint   `gorm:"not null;index"`
	Priority    int    `gorm:"default:3"`
	ScheduledAt *time.Time
	ExpiresAt   *time.Time
	ViewCount   int `gorm:"default:0"`
	Version     int `gorm:"not null;default:1"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (AnnouncementModel) TableName() string {
	return "announcements"
}

func (a *AnnouncementModel) BeforeCreate(tx *gorm.DB) error {
	if a.Status == "" {
		a.Status = "draft"
	}
	if a.Type == "" {
		a.Type = "system"
	}
	if a.Priority == 0 {
		a.Priority = 3
	}
	if a.Version == 0 {
		a.Version = 1
	}
	return nil
}

func (a *AnnouncementModel) BeforeUpdate(tx *gorm.DB) error {
	tx.Statement.SetColumn("version", a.Version+1)
	return nil
}
