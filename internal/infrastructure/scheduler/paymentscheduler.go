package scheduler

import (
	"context"
	"sync"
	"time"

	paymentUsecases "github.com/orris-inc/orris/internal/application/payment/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// PaymentScheduler handles periodic payment expiration tasks.
// - Runs every 5 minutes to check and expire pending payments that have passed their expiration time
// - Payment orders expire 30 minutes after creation if not completed
// - Auto-cancels subscriptions with unpaid payments after 24-hour grace period
// - Retries failed subscription activations for paid non-USDT payments
type PaymentScheduler struct {
	expirePaymentsUC   *paymentUsecases.ExpirePaymentsUseCase
	cancelUnpaidSubsUC *paymentUsecases.CancelUnpaidSubscriptionsUseCase
	retryActivationUC  *paymentUsecases.RetrySubscriptionActivationUseCase
	logger             logger.Interface
	stopChan           chan struct{}
	stopOnce           sync.Once      // Ensures Stop() is only called once
	wg                 sync.WaitGroup // Tracks running goroutines for graceful shutdown
	interval           time.Duration
}

func NewPaymentScheduler(
	expirePaymentsUC *paymentUsecases.ExpirePaymentsUseCase,
	cancelUnpaidSubsUC *paymentUsecases.CancelUnpaidSubscriptionsUseCase,
	retryActivationUC *paymentUsecases.RetrySubscriptionActivationUseCase,
	logger logger.Interface,
) *PaymentScheduler {
	return &PaymentScheduler{
		expirePaymentsUC:   expirePaymentsUC,
		cancelUnpaidSubsUC: cancelUnpaidSubsUC,
		retryActivationUC:  retryActivationUC,
		logger:             logger,
		stopChan:           make(chan struct{}),
		interval:           5 * time.Minute,
	}
}

// Start starts the scheduler and blocks until stopped or context is cancelled.
func (s *PaymentScheduler) Start(ctx context.Context) {
	s.logger.Infow("starting payment scheduler", "interval", s.interval)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runExpirePaymentsLoop(ctx)
	}()
}

// Stop stops the scheduler gracefully and waits for all goroutines to complete.
// Safe to call multiple times - only the first call will actually stop the scheduler.
func (s *PaymentScheduler) Stop() {
	s.stopOnce.Do(func() {
		s.logger.Infow("stopping payment scheduler")
		close(s.stopChan)
		// Wait for all goroutines to complete
		s.wg.Wait()
		s.logger.Infow("payment scheduler stopped")
	})
}

func (s *PaymentScheduler) runExpirePaymentsLoop(ctx context.Context) {
	// Run immediately on startup to clear any pending expired payments
	s.processExpiredPayments(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("payment scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("payment scheduler stopped")
			return
		case <-ticker.C:
			s.processExpiredPayments(ctx)
		}
	}
}

func (s *PaymentScheduler) processExpiredPayments(ctx context.Context) {
	s.logger.Debugw("processing expired payments task started")

	startTime := time.Now()

	// Step 1: Expire pending payments that have passed their expiration time
	expiredCount, err := s.expirePaymentsUC.Execute(ctx)
	if err != nil {
		s.logger.Errorw("failed to process expired payments",
			"error", err,
			"duration", time.Since(startTime),
		)
	} else if expiredCount > 0 {
		s.logger.Infow("expired payments processed",
			"count", expiredCount,
			"duration", time.Since(startTime),
		)
	}

	// Step 2: Cancel subscriptions with unpaid payments beyond grace period
	cancelledCount, err := s.cancelUnpaidSubsUC.Execute(ctx)
	if err != nil {
		s.logger.Errorw("failed to cancel unpaid subscriptions",
			"error", err,
		)
	} else if cancelledCount > 0 {
		s.logger.Infow("unpaid subscriptions cancelled",
			"count", cancelledCount,
		)
	}

	// Step 3: Retry failed subscription activations for paid non-USDT payments
	if s.retryActivationUC != nil {
		retryCount, err := s.retryActivationUC.Execute(ctx)
		if err != nil {
			s.logger.Errorw("failed to retry subscription activations",
				"error", err,
			)
		} else if retryCount > 0 {
			s.logger.Infow("subscription activations retried",
				"count", retryCount,
			)
		}
	}
}
