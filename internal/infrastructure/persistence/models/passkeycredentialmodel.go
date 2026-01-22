package models

import (
	"time"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// PasskeyCredentialModel represents the database persistence model for passkey credentials
type PasskeyCredentialModel struct {
	ID              uint   `gorm:"primarykey"`
	SID             string `gorm:"uniqueIndex;not null;size:50;column:sid"`
	UserID          uint   `gorm:"not null;index"`
	CredentialID    []byte `gorm:"type:varbinary(1024);not null;uniqueIndex:idx_passkey_credentials_credential_id,length:255"`
	PublicKey       []byte `gorm:"type:blob;not null"`
	AttestationType string `gorm:"size:50;default:none"`
	AAGUID          []byte `gorm:"type:varbinary(16);column:aaguid"`
	SignCount       uint32 `gorm:"default:0"`
	BackupEligible  bool   `gorm:"default:false"` // WebAuthn BE flag
	BackupState     bool   `gorm:"default:false"` // WebAuthn BS flag
	Transports      []byte `gorm:"type:json"`     // JSON array of transport hints
	DeviceName      string `gorm:"size:100;default:''"`
	LastUsedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TableName specifies the table name for GORM
func (PasskeyCredentialModel) TableName() string {
	return constants.TablePasskeyCredentials
}
