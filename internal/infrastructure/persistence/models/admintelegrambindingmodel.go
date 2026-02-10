package models

import "time"

// AdminTelegramBindingModel is the GORM model for admin_telegram_bindings table
type AdminTelegramBindingModel struct {
	ID                             uint       `gorm:"primaryKey;autoIncrement"`
	SID                            string     `gorm:"column:sid;type:varchar(50);not null;uniqueIndex"`
	UserID                         uint       `gorm:"column:user_id;not null;uniqueIndex"`
	TelegramUserID                 int64      `gorm:"column:telegram_user_id;not null;uniqueIndex"`
	TelegramUsername               string     `gorm:"column:telegram_username;type:varchar(100)"`
	Language                       string     `gorm:"column:language;type:varchar(5);default:zh"`
	NotifyNodeOffline              bool       `gorm:"column:notify_node_offline;default:true"`
	NotifyAgentOffline             bool       `gorm:"column:notify_agent_offline;default:true"`
	NotifyNewUser                  bool       `gorm:"column:notify_new_user;default:true"`
	NotifyPaymentSuccess           bool       `gorm:"column:notify_payment_success;default:true"`
	NotifyDailySummary             bool       `gorm:"column:notify_daily_summary;default:true"`
	NotifyWeeklySummary            bool       `gorm:"column:notify_weekly_summary;default:true"`
	OfflineThresholdMinutes        int        `gorm:"column:offline_threshold_minutes;default:5"`
	NotifyResourceExpiring         bool       `gorm:"column:notify_resource_expiring;default:true"`
	ResourceExpiringDays           int        `gorm:"column:resource_expiring_days;default:7"`
	DailySummaryHour               int        `gorm:"column:daily_summary_hour;default:9"`
	WeeklySummaryHour              int        `gorm:"column:weekly_summary_hour;default:9"`
	WeeklySummaryWeekday           int        `gorm:"column:weekly_summary_weekday;default:1"`
	OfflineCheckIntervalMinutes    int        `gorm:"column:offline_check_interval_minutes;default:5"`
	LastNodeOfflineNotifyAt        *time.Time `gorm:"column:last_node_offline_notify_at"`
	LastAgentOfflineNotifyAt       *time.Time `gorm:"column:last_agent_offline_notify_at"`
	LastDailySummaryAt             *time.Time `gorm:"column:last_daily_summary_at"`
	LastWeeklySummaryAt            *time.Time `gorm:"column:last_weekly_summary_at"`
	LastResourceExpiringNotifyDate *time.Time `gorm:"column:last_resource_expiring_notify_date"`
	CreatedAt                      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for GORM
func (AdminTelegramBindingModel) TableName() string {
	return "admin_telegram_bindings"
}
