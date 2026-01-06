package models

import (
	"time"
)

// SystemSettingModel is the GORM model for system_settings table
type SystemSettingModel struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	SID         string    `gorm:"column:sid;type:varchar(50);not null;uniqueIndex"`
	Category    string    `gorm:"column:category;type:varchar(100);not null;index:idx_category_key"`
	SettingKey  string    `gorm:"column:setting_key;type:varchar(100);not null;index:idx_category_key"`
	Value       string    `gorm:"column:value;type:text"`
	ValueType   string    `gorm:"column:value_type;type:varchar(20);not null;default:'string'"`
	Description string    `gorm:"column:description;type:varchar(500)"`
	UpdatedBy   uint      `gorm:"column:updated_by"`
	Version     int       `gorm:"column:version;default:1"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for GORM
func (SystemSettingModel) TableName() string {
	return "system_settings"
}
