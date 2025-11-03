package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type PaymentModel struct {
	ID             uint    `gorm:"primaryKey"`
	OrderNo        string  `gorm:"uniqueIndex;size:64;not null"`
	SubscriptionID uint    `gorm:"index;not null"`
	UserID         uint    `gorm:"index;not null"`
	Amount         int64   `gorm:"not null"`
	Currency       string  `gorm:"size:10;not null;default:'CNY'"`
	PaymentMethod  string  `gorm:"size:20;not null"`
	PaymentStatus  string  `gorm:"size:20;not null;index"`
	GatewayOrderNo *string `gorm:"size:128;index"`
	TransactionID  *string `gorm:"size:128"`
	PaymentURL     *string `gorm:"type:text"`
	QRCode         *string `gorm:"type:text"`
	PaidAt         *time.Time
	ExpiredAt      time.Time `gorm:"not null"`
	Metadata       JSONB     `gorm:"type:jsonb"`
	Version        int       `gorm:"default:0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (PaymentModel) TableName() string {
	return "payments"
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
