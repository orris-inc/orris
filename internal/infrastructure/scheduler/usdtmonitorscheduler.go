package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/application/payment/suffixalloc"
	paymentUsecases "github.com/orris-inc/orris/internal/application/payment/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// USDTMonitorScheduler handles periodic USDT payment confirmation monitoring.
// - Runs every 30 seconds to check pending USDT payments for blockchain confirmations
// - When a matching transaction is found with sufficient confirmations, the payment is marked as paid
// - Also periodically cleans up expired suffix allocations
type USDTMonitorScheduler struct {
	confirmUSDTPaymentUC *paymentUsecases.ConfirmUSDTPaymentUseCase
	suffixAllocator      suffixalloc.SuffixAllocator
	logger               logger.Interface
	stopChan             chan struct{}
	stopOnce             sync.Once
	wg                   sync.WaitGroup
	interval             time.Duration
	cleanupInterval      time.Duration
	lastCleanup          time.Time
	cleanupRunning       bool // Prevents concurrent cleanup executions
	running              bool
	mu                   sync.RWMutex
}

// NewUSDTMonitorScheduler creates a new USDT monitor scheduler
func NewUSDTMonitorScheduler(
	confirmUSDTPaymentUC *paymentUsecases.ConfirmUSDTPaymentUseCase,
	logger logger.Interface,
) *USDTMonitorScheduler {
	return &USDTMonitorScheduler{
		confirmUSDTPaymentUC: confirmUSDTPaymentUC,
		logger:               logger,
		stopChan:             make(chan struct{}),
		interval:             30 * time.Second,
		cleanupInterval:      5 * time.Minute, // Cleanup expired suffixes every 5 minutes
	}
}

// SetSuffixAllocator sets the suffix allocator for cleanup operations
func (s *USDTMonitorScheduler) SetSuffixAllocator(allocator suffixalloc.SuffixAllocator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suffixAllocator = allocator
}

// Start starts the scheduler
func (s *USDTMonitorScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Infow("starting USDT monitor scheduler", "interval", s.interval)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runMonitorLoop(ctx)
	}()
}

// Stop stops the scheduler gracefully
func (s *USDTMonitorScheduler) Stop() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()

		s.logger.Infow("stopping USDT monitor scheduler")
		close(s.stopChan)
		s.wg.Wait()
		s.logger.Infow("USDT monitor scheduler stopped")
	})
}

// IsRunning returns whether the scheduler is running
func (s *USDTMonitorScheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *USDTMonitorScheduler) runMonitorLoop(ctx context.Context) {
	// Run immediately on startup
	s.processUSDTPayments(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("USDT monitor scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("USDT monitor scheduler stopped")
			return
		case <-ticker.C:
			s.processUSDTPayments(ctx)
		}
	}
}

func (s *USDTMonitorScheduler) processUSDTPayments(ctx context.Context) {
	s.logger.Debugw("checking USDT payments")

	startTime := time.Now()
	results, err := s.confirmUSDTPaymentUC.Execute(ctx)
	if err != nil {
		s.logger.Errorw("failed to check USDT payments",
			"error", err,
			"duration", time.Since(startTime),
		)
		return
	}

	if len(results) > 0 {
		confirmedCount := 0
		pendingCount := 0
		activationFailedCount := 0
		for _, r := range results {
			if r.Confirmed {
				confirmedCount++
				if r.SubscriptionActivation == "failed" {
					activationFailedCount++
				}
				s.logger.Infow("USDT payment confirmed",
					"payment_id", r.PaymentID,
					"tx_hash", r.TxHash,
					"confirmations", r.Confirmations,
					"subscription_activated", r.SubscriptionActivated,
				)
			} else {
				pendingCount++
			}
		}

		if confirmedCount > 0 || pendingCount > 0 {
			s.logger.Infow("USDT payment check completed",
				"confirmed", confirmedCount,
				"pending_confirmations", pendingCount,
				"activation_failed", activationFailedCount,
				"duration", time.Since(startTime),
			)
		}
	}

	// Retry pending subscription activations
	s.retryPendingActivations(ctx)

	// Cleanup expired suffix allocations periodically
	s.cleanupExpiredSuffixes(ctx)
}

// retryPendingActivations retries subscription activation for confirmed payments
func (s *USDTMonitorScheduler) retryPendingActivations(ctx context.Context) {
	successCount, err := s.confirmUSDTPaymentUC.RetryPendingSubscriptionActivations(ctx)
	if err != nil {
		s.logger.Warnw("failed to retry pending activations", "error", err)
		return
	}
	if successCount > 0 {
		s.logger.Infow("retried pending subscription activations", "success_count", successCount)
	}
}

// cleanupExpiredSuffixes cleans up expired suffix allocations periodically
func (s *USDTMonitorScheduler) cleanupExpiredSuffixes(ctx context.Context) {
	s.mu.Lock()
	allocator := s.suffixAllocator
	if allocator == nil {
		s.mu.Unlock()
		return
	}

	// Only cleanup every cleanupInterval
	if time.Since(s.lastCleanup) < s.cleanupInterval {
		s.mu.Unlock()
		return
	}

	// Check if cleanup is already running to prevent concurrent executions
	if s.cleanupRunning {
		s.mu.Unlock()
		return
	}

	// Mark cleanup as started
	s.cleanupRunning = true
	s.lastCleanup = time.Now()
	s.mu.Unlock()

	// Perform cleanup outside of lock to avoid blocking other operations
	if err := allocator.CleanupExpired(ctx); err != nil {
		s.logger.Warnw("failed to cleanup expired suffixes", "error", err)
	}

	// Mark cleanup as finished
	s.mu.Lock()
	s.cleanupRunning = false
	s.mu.Unlock()
}
