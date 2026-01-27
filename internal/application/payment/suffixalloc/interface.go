package suffixalloc

import (
	"context"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
)

const (
	// USDTUnit is 1 USDT in smallest unit (6 decimals)
	USDTUnit = 1000000
	// SuffixUnit is the value of suffix 1 in smallest unit (0.0001 USDT = 100 units)
	SuffixUnit = 100
)

// AllocationResult contains the result of a suffix allocation
type AllocationResult struct {
	Suffix           uint   // The allocated suffix (e.g., 1234 for 0.1234 USDT suffix)
	FullAmountRaw    uint64 // The full amount in smallest unit (base + suffix)
	ReceivingAddress string // The allocated receiving address from the address pool
}

// SuffixAllocator manages unique USDT amount suffixes for payment matching
type SuffixAllocator interface {
	// Allocate allocates a unique suffix for a payment from the address pool
	// addresses is the list of receiving addresses to try (multi-wallet support)
	// baseAmountRaw is the base USDT amount in smallest unit (e.g., 10000000 for 10.00 USDT)
	// paymentID is the payment ID to associate with this suffix
	// ttl is the duration the suffix should be reserved
	// Returns the allocated suffix, full amount, and receiving address
	Allocate(ctx context.Context, chainType vo.ChainType, addresses []string, baseAmountRaw uint64, paymentID uint, ttl time.Duration) (*AllocationResult, error)

	// Release releases a previously allocated suffix
	Release(ctx context.Context, chainType vo.ChainType, receivingAddress string, baseAmountRaw uint64, suffix uint) error

	// CleanupExpired removes all expired suffix allocations
	CleanupExpired(ctx context.Context) error
}

// RoundToBase rounds a raw USDT amount down to the nearest 0.01 USDT (base amount)
// For example: 10123456 -> 10120000 (10.123456 -> 10.12)
func RoundToBase(amountRaw uint64) uint64 {
	// 0.01 USDT = 10000 smallest units
	const baseUnit = 10000
	return (amountRaw / baseUnit) * baseUnit
}
