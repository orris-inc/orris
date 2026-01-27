package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/orris-inc/orris/internal/shared/constants"
)

type PaymentModel struct {
	ID             uint    `gorm:"primaryKey"`
	OrderNo        string  `gorm:"uniqueIndex;size:64;not null"`
	SubscriptionID uint    `gorm:"index;not null"`
	UserID         uint    `gorm:"index;not null"`
	Amount         int64   `gorm:"not null"`
	Currency       string  `gorm:"size:10;not null"`
	PaymentMethod  string  `gorm:"size:20;not null"`
	PaymentStatus  string  `gorm:"size:20;not null;index"`
	GatewayOrderNo *string `gorm:"size:128;index"`
	TransactionID  *string `gorm:"size:128"`
	PaymentURL     *string `gorm:"type:text"`
	QRCode         *string `gorm:"type:text"`
	PaidAt         *time.Time
	ExpiredAt      time.Time `gorm:"not null"`

	// USDT-specific fields
	ChainType        *string  `gorm:"size:10;uniqueIndex:uk_payments_chain_tx_hash,priority:1"`
	USDTAmountRaw    *uint64  `gorm:"column:usdt_amount_raw"` // USDT amount in smallest unit (1 USDT = 1000000)
	ReceivingAddress *string  `gorm:"size:64"`
	ExchangeRate     *float64 `gorm:"type:decimal(20,8)"` // Exchange rate at time of payment (for display only)
	TxHash           *string  `gorm:"size:128;uniqueIndex:uk_payments_chain_tx_hash,priority:2"`
	BlockNumber      *uint64
	ConfirmedAt      *time.Time

	Metadata  JSONB `gorm:"type:jsonb"`
	Version   int   `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (PaymentModel) TableName() string {
	return constants.TablePayments
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}
