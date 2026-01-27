package payment

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/orris-inc/orris/internal/application/payment/suffixalloc"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// Suffix range: 0001-9999 (4 decimal places)
	// Using sequential allocation, most orders will use 0001-0100
	minSuffix = 1
	maxSuffix = 9999
	// Maximum allocation attempts (for sequential search)
	maxAttempts = 100
	// Cooldown period after suffix expiration before it can be reused
	// This prevents race conditions where a late payment from user A
	// gets confirmed for user B who received the same suffix
	suffixCooldownPeriod = 1 * time.Hour
)

// SuffixAllocator manages unique USDT amount suffixes
type SuffixAllocator struct {
	db     *gorm.DB
	logger logger.Interface
}

// NewSuffixAllocator creates a new suffix allocator
func NewSuffixAllocator(db *gorm.DB, logger logger.Interface) *SuffixAllocator {
	return &SuffixAllocator{
		db:     db,
		logger: logger,
	}
}

// Ensure SuffixAllocator implements the interface
var _ suffixalloc.SuffixAllocator = (*SuffixAllocator)(nil)

// Allocate allocates a unique suffix for a payment using sequential allocation strategy
// with multi-wallet support. It iterates through suffixes first, then addresses.
// addresses is the list of receiving addresses to try
// baseAmountRaw is the base USDT amount in smallest unit (e.g., 10000000 for 10.00 USDT)
func (a *SuffixAllocator) Allocate(ctx context.Context, chainType vo.ChainType, addresses []string, baseAmountRaw uint64, paymentID uint, ttl time.Duration) (*suffixalloc.AllocationResult, error) {
	if len(addresses) == 0 {
		return nil, fmt.Errorf("no receiving addresses configured for %s", chainType)
	}

	now := biztime.NowUTC()
	expiresAt := now.Add(ttl)
	cooldownThreshold := now.Add(-suffixCooldownPeriod)

	// New allocation strategy: iterate suffixes first, then addresses
	// This distributes load across multiple wallets for the same suffix
	for suffix := uint(minSuffix); suffix <= maxSuffix && suffix < uint(minSuffix+maxAttempts); suffix++ {
		for _, address := range addresses {
			if allocated := a.tryAllocate(ctx, chainType, address, baseAmountRaw, suffix, paymentID, now, expiresAt, cooldownThreshold); allocated != nil {
				return allocated, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to allocate suffix after trying %d suffixes across %d addresses, all slots may be in use", maxAttempts, len(addresses))
}

// tryAllocate attempts to allocate a specific suffix for a specific address
// Uses SELECT FOR UPDATE + conditional logic since MySQL doesn't support
// ON CONFLICT ... WHERE clause
func (a *SuffixAllocator) tryAllocate(ctx context.Context, chainType vo.ChainType, address string, baseAmountRaw uint64, suffix uint, paymentID uint, now, expiresAt, cooldownThreshold time.Time) *suffixalloc.AllocationResult {
	txDB := db.GetTxFromContext(ctx, a.db)

	// First, try to find an existing record that can be reused
	var existing models.USDTSuffixModel
	err := txDB.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("chain_type = ? AND receiving_address = ? AND base_amount_raw = ? AND suffix = ?",
			chainType.String(), address, baseAmountRaw, suffix).
		First(&existing).Error

	if err == nil {
		// Record exists, check if it can be reused
		if existing.PaymentID == nil || (existing.ExpiresAt != nil && existing.ExpiresAt.Before(cooldownThreshold)) {
			// Suffix is available (not allocated or expired beyond cooldown)
			result := txDB.Model(&existing).Updates(map[string]interface{}{
				"payment_id":   paymentID,
				"allocated_at": now,
				"expires_at":   expiresAt,
			})
			if result.Error != nil {
				a.logger.Debugw("failed to update suffix allocation",
					"error", result.Error,
					"suffix", suffix,
					"address", address,
				)
				return nil
			}
			return a.buildAllocationResult(chainType, address, baseAmountRaw, suffix, paymentID)
		}
		// Suffix is in use by another payment
		return nil
	}

	// Record doesn't exist, try to insert
	if err == gorm.ErrRecordNotFound {
		model := &models.USDTSuffixModel{
			ChainType:        chainType.String(),
			ReceivingAddress: address,
			BaseAmountRaw:    baseAmountRaw,
			Suffix:           suffix,
			PaymentID:        &paymentID,
			AllocatedAt:      &now,
			ExpiresAt:        &expiresAt,
			CreatedAt:        now,
		}
		result := txDB.Create(model)
		if result.Error != nil {
			// Could be duplicate key error from concurrent insert, that's OK
			a.logger.Debugw("failed to insert suffix allocation, may be concurrent insert",
				"error", result.Error,
				"suffix", suffix,
				"address", address,
			)
			return nil
		}
		return a.buildAllocationResult(chainType, address, baseAmountRaw, suffix, paymentID)
	}

	// Other error
	a.logger.Debugw("failed to query suffix allocation",
		"error", err,
		"suffix", suffix,
		"address", address,
	)
	return nil
}

// buildAllocationResult creates an AllocationResult and logs the allocation
func (a *SuffixAllocator) buildAllocationResult(chainType vo.ChainType, address string, baseAmountRaw uint64, suffix uint, paymentID uint) *suffixalloc.AllocationResult {
	// Calculate full amount: base + suffix * SuffixUnit
	// Suffix is 0001-9999 representing 0.0001-0.9999 USDT
	// SuffixUnit = 100 (0.0001 USDT in smallest units)
	fullAmountRaw := baseAmountRaw + uint64(suffix)*suffixalloc.SuffixUnit
	a.logger.Infow("allocated USDT suffix (multi-wallet)",
		"chain_type", chainType,
		"receiving_address", address,
		"base_amount_raw", baseAmountRaw,
		"suffix", suffix,
		"full_amount_raw", fullAmountRaw,
		"payment_id", paymentID,
	)
	return &suffixalloc.AllocationResult{
		Suffix:           suffix,
		FullAmountRaw:    fullAmountRaw,
		ReceivingAddress: address,
	}
}

// Release releases a previously allocated suffix
func (a *SuffixAllocator) Release(ctx context.Context, chainType vo.ChainType, receivingAddress string, baseAmountRaw uint64, suffix uint) error {
	result := db.GetTxFromContext(ctx, a.db).
		Model(&models.USDTSuffixModel{}).
		Where("chain_type = ? AND receiving_address = ? AND base_amount_raw = ? AND suffix = ?",
			chainType.String(), receivingAddress, baseAmountRaw, suffix).
		Updates(map[string]interface{}{
			"payment_id":   nil,
			"allocated_at": nil,
			"expires_at":   nil,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to release suffix: %w", result.Error)
	}

	a.logger.Infow("released USDT suffix",
		"chain_type", chainType,
		"receiving_address", receivingAddress,
		"base_amount_raw", baseAmountRaw,
		"suffix", suffix,
	)

	return nil
}

// CleanupExpired removes expired suffix allocations after cooldown period
// The cooldown period prevents race conditions where a late payment gets
// confirmed for the wrong user after suffix reallocation
func (a *SuffixAllocator) CleanupExpired(ctx context.Context) error {
	now := biztime.NowUTC()
	// Only cleanup suffixes that expired more than cooldownPeriod ago
	cleanupThreshold := now.Add(-suffixCooldownPeriod)

	result := db.GetTxFromContext(ctx, a.db).
		Model(&models.USDTSuffixModel{}).
		Where("expires_at < ? AND payment_id IS NOT NULL", cleanupThreshold).
		Updates(map[string]interface{}{
			"payment_id":   nil,
			"allocated_at": nil,
			"expires_at":   nil,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired suffixes: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		a.logger.Infow("cleaned up expired suffixes",
			"count", result.RowsAffected,
			"cooldown_period", suffixCooldownPeriod,
		)
	}

	return nil
}
