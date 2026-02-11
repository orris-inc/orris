package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// AnyTLSConfigModel represents the database persistence model for AnyTLS protocol configuration
type AnyTLSConfigModel struct {
	ID                       uint   `gorm:"primarykey"`
	NodeID                   uint   `gorm:"uniqueIndex;not null"`
	SNI                      string `gorm:"size:255"`
	AllowInsecure            bool   `gorm:"not null;default:true"`
	Fingerprint              string `gorm:"size:100"`
	IdleSessionCheckInterval string `gorm:"size:20"`
	IdleSessionTimeout       string `gorm:"size:20"`
	MinIdleSession           int    `gorm:"not null;default:0"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (AnyTLSConfigModel) TableName() string {
	return constants.TableNodeAnyTLSConfigs
}
