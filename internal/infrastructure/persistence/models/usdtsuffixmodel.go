package models

import (
	"time"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// USDTSuffixModel represents an allocated USDT amount suffix for payment matching
type USDTSuffixModel struct {
	ID               uint   `gorm:"primaryKey"`
	ChainType        string `gorm:"size:10;not null;uniqueIndex:uk_chain_address_base_suffix"`
	ReceivingAddress string `gorm:"size:64;not null;uniqueIndex:uk_chain_address_base_suffix"`
	BaseAmountRaw    uint64 `gorm:"column:base_amount_raw;not null;uniqueIndex:uk_chain_address_base_suffix"` // Base amount in smallest unit
	Suffix           uint   `gorm:"not null;uniqueIndex:uk_chain_address_base_suffix"`
	PaymentID        *uint  `gorm:"index"`
	AllocatedAt      *time.Time
	ExpiresAt        *time.Time
	CreatedAt        time.Time
}

func (USDTSuffixModel) TableName() string {
	return constants.TableUSDTAmountSuffixes
}
