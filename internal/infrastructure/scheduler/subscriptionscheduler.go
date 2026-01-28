package scheduler

import (
	"context"
	"sync"
	"time"

	subscriptionUsecases "github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionScheduler handles periodic subscription maintenance tasks.
// - Runs daily to mark expired subscriptions (data consistency for reports/statistics)
// - Note: Display layer uses EffectiveStatus() for real-time accuracy
type SubscriptionScheduler struct {
	expireSubscriptionsUC *subscriptionUsecases.ExpireSubscriptionsUseCase
	logger                logger.Interface
	stopChan              chan struct{}
	stopOnce              sync.Once
	wg                    sync.WaitGroup
	interval              time.Duration
}

// NewSubscriptionScheduler creates a new SubscriptionScheduler
func NewSubscriptionScheduler(
	expireSubscriptionsUC *subscriptionUsecases.ExpireSubscriptionsUseCase,
	logger logger.Interface,
) *SubscriptionScheduler {
	return &SubscriptionScheduler{
		expireSubscriptionsUC: expireSubscriptionsUC,
		logger:                logger,
		stopChan:              make(chan struct{}),
		interval:              24 * time.Hour, // Run once per day
	}
}

// Start starts the scheduler
func (s *SubscriptionScheduler) Start(ctx context.Context) {
	s.logger.Infow("starting subscription scheduler", "interval", s.interval)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runLoop(ctx)
	}()
}

// Stop stops the scheduler gracefully
func (s *SubscriptionScheduler) Stop() {
	s.stopOnce.Do(func() {
		s.logger.Infow("stopping subscription scheduler")
		close(s.stopChan)
		s.wg.Wait()
		s.logger.Infow("subscription scheduler stopped")
	})
}

func (s *SubscriptionScheduler) runLoop(ctx context.Context) {
	// Run immediately on startup to clear any pending expired subscriptions
	s.processExpiredSubscriptions(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("subscription scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("subscription scheduler stopped")
			return
		case <-ticker.C:
			s.processExpiredSubscriptions(ctx)
		}
	}
}

func (s *SubscriptionScheduler) processExpiredSubscriptions(ctx context.Context) {
	s.logger.Debugw("processing expired subscriptions task started")

	startTime := time.Now()

	expiredCount, err := s.expireSubscriptionsUC.Execute(ctx)
	if err != nil {
		s.logger.Errorw("failed to process expired subscriptions",
			"error", err,
			"duration", time.Since(startTime),
		)
		return
	}

	if expiredCount > 0 {
		s.logger.Infow("expired subscriptions processed",
			"count", expiredCount,
			"duration", time.Since(startTime),
		)
	} else {
		s.logger.Debugw("no expired subscriptions to process",
			"duration", time.Since(startTime),
		)
	}
}
