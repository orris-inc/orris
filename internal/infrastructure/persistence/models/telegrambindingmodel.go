package models

import "time"

// TelegramBindingModel is the GORM model for telegram_bindings table
type TelegramBindingModel struct {
	ID                   uint       `gorm:"primaryKey;autoIncrement"`
	SID                  string     `gorm:"column:sid;type:varchar(50);not null;uniqueIndex"`
	UserID               uint       `gorm:"column:user_id;not null;uniqueIndex"`
	TelegramUserID       int64      `gorm:"column:telegram_user_id;not null;uniqueIndex"`
	TelegramUsername     string     `gorm:"column:telegram_username;type:varchar(100)"`
	Language             string     `gorm:"column:language;type:varchar(5);default:zh"`
	NotifyExpiring       bool       `gorm:"column:notify_expiring;default:true"`
	NotifyTraffic        bool       `gorm:"column:notify_traffic;default:true"`
	ExpiringDays         int        `gorm:"column:expiring_days;default:3"`
	TrafficThreshold     int        `gorm:"column:traffic_threshold;default:80"`
	LastExpiringNotifyAt *time.Time `gorm:"column:last_expiring_notify_at"`
	LastTrafficNotifyAt  *time.Time `gorm:"column:last_traffic_notify_at"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for GORM
func (TelegramBindingModel) TableName() string {
	return "telegram_bindings"
}
