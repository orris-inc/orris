package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// Hysteria2ConfigModel represents the database persistence model for Hysteria2 protocol configuration
// Separated from NodeModel to follow protocol-specific table pattern
type Hysteria2ConfigModel struct {
	ID                uint    `gorm:"primarykey"`
	NodeID            uint    `gorm:"uniqueIndex;not null"`              // Logical foreign key to nodes table
	CongestionControl string  `gorm:"not null;size:20;default:bbr"`      // cubic, bbr, new_reno
	Obfs              string  `gorm:"size:20"`                           // salamander or empty
	ObfsPassword      string  `gorm:"size:255"`                          // Obfuscation password (required if obfs=salamander)
	UpMbps            *uint   `gorm:"type:int unsigned"`                 // Upstream bandwidth limit (nullable)
	DownMbps          *uint   `gorm:"type:int unsigned"`                 // Downstream bandwidth limit (nullable)
	SNI               string  `gorm:"size:255"`                          // TLS Server Name Indication
	AllowInsecure     bool    `gorm:"not null;default:true"`             // Allow insecure TLS connection
	Fingerprint       string  `gorm:"size:100"`                          // TLS fingerprint
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (Hysteria2ConfigModel) TableName() string {
	return constants.TableNodeHysteria2Configs
}

// BeforeCreate hook for GORM
func (h *Hysteria2ConfigModel) BeforeCreate(tx *gorm.DB) error {
	if h.CongestionControl == "" {
		h.CongestionControl = "bbr"
	}
	return nil
}
