package models

import "time"

// SessionModel represents the database persistence model for sessions.
type SessionModel struct {
	ID               string    `gorm:"primarykey;size:64"`
	UserID           uint      `gorm:"not null;index"`
	DeviceName       string    `gorm:"size:255"`
	DeviceType       string    `gorm:"size:50"`
	IPAddress        string    `gorm:"size:45"`
	UserAgent        string    `gorm:"size:512"`
	TokenHash        string    `gorm:"size:255;index"`
	RefreshTokenHash string    `gorm:"size:255;index"`
	ExpiresAt        time.Time `gorm:"not null;index"`
	LastActivityAt   time.Time `gorm:"not null"`
	CreatedAt        time.Time
}

// TableName specifies the table name for GORM
func (SessionModel) TableName() string {
	return "sessions"
}
