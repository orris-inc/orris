package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// TUICConfigModel represents the database persistence model for TUIC protocol configuration
// Separated from NodeModel to follow protocol-specific table pattern
type TUICConfigModel struct {
	ID                uint   `gorm:"primarykey"`
	NodeID            uint   `gorm:"uniqueIndex;not null"`            // Logical foreign key to nodes table
	CongestionControl string `gorm:"not null;size:20;default:bbr"`    // cubic, bbr, new_reno
	UDPRelayMode      string `gorm:"not null;size:10;default:native"` // native, quic
	ALPN              string `gorm:"size:50"`                         // h3, h3-29, etc.
	SNI               string `gorm:"size:255"`                        // TLS Server Name Indication
	AllowInsecure     bool   `gorm:"not null;default:false"`          // Allow insecure TLS connection
	DisableSNI        bool   `gorm:"not null;default:false"`          // Disable SNI
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (TUICConfigModel) TableName() string {
	return constants.TableNodeTUICConfigs
}

// BeforeCreate hook for GORM
func (t *TUICConfigModel) BeforeCreate(tx *gorm.DB) error {
	if t.CongestionControl == "" {
		t.CongestionControl = "bbr"
	}
	if t.UDPRelayMode == "" {
		t.UDPRelayMode = "native"
	}
	return nil
}
