package blockchain

import (
	"context"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
)

const (
	// USDTDecimals is the number of decimal places for USDT (6 decimals)
	USDTDecimals = 6
	// USDTUnit is the multiplier to convert USDT to smallest unit (1 USDT = 1000000 units)
	USDTUnit = 1000000
)

// Transaction represents a blockchain transaction
type Transaction struct {
	TxHash        string
	FromAddress   string
	ToAddress     string
	AmountRaw     uint64 // Amount in smallest unit (1 USDT = 1000000)
	BlockNumber   uint64
	Confirmations int
	Timestamp     time.Time
}

// AmountUSDT returns the amount as a float64 for display purposes only
func (t *Transaction) AmountUSDT() float64 {
	return float64(t.AmountRaw) / float64(USDTUnit)
}

// TransactionMonitor provides blockchain transaction monitoring
type TransactionMonitor interface {
	// FindTransaction searches for a transaction matching the given criteria
	// toAddress is the receiving address
	// amountRaw is the expected USDT amount in smallest unit (1 USDT = 1000000)
	// createdAfter filters transactions to only include those after the payment creation time
	// Returns nil if no matching transaction is found
	FindTransaction(ctx context.Context, chainType vo.ChainType, toAddress string, amountRaw uint64, createdAfter time.Time) (*Transaction, error)

	// GetConfirmations returns the current number of confirmations for a transaction
	GetConfirmations(ctx context.Context, chainType vo.ChainType, txHash string) (int, error)
}

// FloatToRawAmount converts a float64 USDT amount to raw uint64 (smallest unit)
// Uses rounding to handle floating point precision issues
func FloatToRawAmount(amount float64) uint64 {
	// Add 0.5 for rounding before truncating
	return uint64(amount*float64(USDTUnit) + 0.5)
}

// RawAmountToFloat converts raw uint64 (smallest unit) to float64 USDT amount
func RawAmountToFloat(raw uint64) float64 {
	return float64(raw) / float64(USDTUnit)
}
