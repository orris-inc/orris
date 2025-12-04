package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// TrojanConfigModel represents the database persistence model for Trojan protocol configuration
// Separated from NodeModel to follow protocol-specific table pattern
type TrojanConfigModel struct {
	ID                uint   `gorm:"primarykey"`
	NodeID            uint   `gorm:"uniqueIndex;not null"`         // Logical foreign key to nodes table
	TransportProtocol string `gorm:"not null;size:10;default:tcp"` // tcp, ws, grpc
	Host              string `gorm:"size:255"`                     // WebSocket host header or gRPC service name
	Path              string `gorm:"size:255"`                     // WebSocket path
	SNI               string `gorm:"size:255"`                     // TLS Server Name Indication
	AllowInsecure     bool   `gorm:"not null;default:true"`        // Allow insecure TLS connection (default true for self-signed certs)
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (TrojanConfigModel) TableName() string {
	return constants.TableTrojanConfigs
}

// BeforeCreate hook for GORM
func (t *TrojanConfigModel) BeforeCreate(tx *gorm.DB) error {
	if t.TransportProtocol == "" {
		t.TransportProtocol = "tcp"
	}
	return nil
}
