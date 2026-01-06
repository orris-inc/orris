package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UsageAggregator defines the interface for aggregating usage data.
type UsageAggregator interface {
	// AggregateDailyUsage aggregates hourly data from yesterday into daily stats.
	AggregateDailyUsage(ctx context.Context) error
	// AggregateMonthlyUsage aggregates daily data from last month into monthly stats.
	AggregateMonthlyUsage(ctx context.Context) error
	// CleanupOldUsageData deletes raw usage records older than the specified retention days.
	CleanupOldUsageData(ctx context.Context, retentionDays int) error
}

// UsageAggregationScheduler runs periodic usage data aggregation tasks.
// - Daily aggregation: runs at 03:00 business timezone every day
// - Monthly aggregation: runs at 04:00 business timezone on the 1st of each month
// - Data cleanup: runs at 05:00 business timezone every day (after daily aggregation)
type UsageAggregationScheduler struct {
	aggregator    UsageAggregator
	retentionDays int // Raw data retention period in days, default 90
	logger        logger.Interface
	stopChan      chan struct{}
	stopOnce      sync.Once      // Ensures Stop() is only called once
	wg            sync.WaitGroup // Tracks running goroutines for graceful shutdown
}

// DefaultRetentionDays is the default number of days to retain raw usage data.
const DefaultRetentionDays = 90

// NewUsageAggregationScheduler creates a new usage aggregation scheduler.
// retentionDays specifies how many days to retain raw usage data before cleanup.
// If retentionDays <= 0, it defaults to DefaultRetentionDays (90 days).
func NewUsageAggregationScheduler(
	aggregator UsageAggregator,
	retentionDays int,
	logger logger.Interface,
) *UsageAggregationScheduler {
	if retentionDays <= 0 {
		retentionDays = DefaultRetentionDays
	}
	return &UsageAggregationScheduler{
		aggregator:    aggregator,
		retentionDays: retentionDays,
		logger:        logger,
		stopChan:      make(chan struct{}),
	}
}

// Start starts the scheduler and blocks until stopped or context is cancelled.
func (s *UsageAggregationScheduler) Start(ctx context.Context) {
	s.logger.Infow("starting usage aggregation scheduler",
		"retention_days", s.retentionDays,
	)

	// Start goroutine for daily aggregation
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runDailyAggregation(ctx)
	}()

	// Start goroutine for monthly aggregation
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runMonthlyAggregation(ctx)
	}()

	// Start goroutine for data cleanup
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runDataCleanup(ctx)
	}()
}

// Stop stops the scheduler gracefully and waits for all goroutines to complete.
// Safe to call multiple times - only the first call will actually stop the scheduler.
func (s *UsageAggregationScheduler) Stop() {
	s.stopOnce.Do(func() {
		s.logger.Infow("stopping usage aggregation scheduler")
		close(s.stopChan)
		// Wait for all goroutines to complete
		s.wg.Wait()
		s.logger.Infow("usage aggregation scheduler stopped")
	})
}

// runDailyAggregation runs the daily aggregation task at 03:00 business timezone.
func (s *UsageAggregationScheduler) runDailyAggregation(ctx context.Context) {
	for {
		// Calculate duration until next 03:00 in business timezone
		nextRun := s.nextDailyRunTime()
		duration := time.Until(nextRun)

		s.logger.Infow("scheduled next daily aggregation",
			"next_run", nextRun.Format(time.RFC3339),
			"duration", duration,
		)

		timer := time.NewTimer(duration)

		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Infow("daily aggregation scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			timer.Stop()
			s.logger.Infow("daily aggregation scheduler stopped")
			return
		case <-timer.C:
			s.executeDailyAggregation(ctx)
		}
	}
}

// runMonthlyAggregation runs the monthly aggregation task at 04:00 on the 1st of each month.
func (s *UsageAggregationScheduler) runMonthlyAggregation(ctx context.Context) {
	for {
		// Calculate duration until next 1st day 04:00 in business timezone
		nextRun := s.nextMonthlyRunTime()
		duration := time.Until(nextRun)

		s.logger.Infow("scheduled next monthly aggregation",
			"next_run", nextRun.Format(time.RFC3339),
			"duration", duration,
		)

		timer := time.NewTimer(duration)

		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Infow("monthly aggregation scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			timer.Stop()
			s.logger.Infow("monthly aggregation scheduler stopped")
			return
		case <-timer.C:
			s.executeMonthlyAggregation(ctx)
		}
	}
}

// runDataCleanup runs the data cleanup task at 05:00 business timezone every day.
func (s *UsageAggregationScheduler) runDataCleanup(ctx context.Context) {
	for {
		// Calculate duration until next 05:00 in business timezone
		nextRun := s.nextCleanupRunTime()
		duration := time.Until(nextRun)

		s.logger.Infow("scheduled next data cleanup",
			"next_run", nextRun.Format(time.RFC3339),
			"duration", duration,
			"retention_days", s.retentionDays,
		)

		timer := time.NewTimer(duration)

		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Infow("data cleanup scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			timer.Stop()
			s.logger.Infow("data cleanup scheduler stopped")
			return
		case <-timer.C:
			s.executeDataCleanup(ctx)
		}
	}
}

// nextDailyRunTime calculates the next 03:00 in business timezone.
func (s *UsageAggregationScheduler) nextDailyRunTime() time.Time {
	loc := biztime.Location()
	now := time.Now().In(loc)

	// Target time: 03:00 today
	target := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, loc)

	// If we've already passed 03:00 today, schedule for tomorrow
	if now.After(target) {
		target = target.AddDate(0, 0, 1)
	}

	return target
}

// nextMonthlyRunTime calculates the next 1st day 04:00 in business timezone.
func (s *UsageAggregationScheduler) nextMonthlyRunTime() time.Time {
	loc := biztime.Location()
	now := time.Now().In(loc)

	// Target time: 04:00 on the 1st of current month
	target := time.Date(now.Year(), now.Month(), 1, 4, 0, 0, 0, loc)

	// If we've already passed this month's 1st 04:00, schedule for next month
	if now.After(target) {
		target = target.AddDate(0, 1, 0)
	}

	return target
}

// nextCleanupRunTime calculates the next 05:00 in business timezone.
func (s *UsageAggregationScheduler) nextCleanupRunTime() time.Time {
	loc := biztime.Location()
	now := time.Now().In(loc)

	// Target time: 05:00 today (after daily aggregation at 03:00)
	target := time.Date(now.Year(), now.Month(), now.Day(), 5, 0, 0, 0, loc)

	// If we've already passed 05:00 today, schedule for tomorrow
	if now.After(target) {
		target = target.AddDate(0, 0, 1)
	}

	return target
}

// executeDailyAggregation performs the daily aggregation task.
func (s *UsageAggregationScheduler) executeDailyAggregation(ctx context.Context) {
	s.logger.Infow("executing daily usage aggregation")

	startTime := time.Now()
	if err := s.aggregator.AggregateDailyUsage(ctx); err != nil {
		s.logger.Errorw("daily usage aggregation failed",
			"error", err,
			"duration", time.Since(startTime),
		)
		return
	}

	s.logger.Infow("daily usage aggregation completed successfully",
		"duration", time.Since(startTime),
	)
}

// executeMonthlyAggregation performs the monthly aggregation task.
func (s *UsageAggregationScheduler) executeMonthlyAggregation(ctx context.Context) {
	s.logger.Infow("executing monthly usage aggregation")

	startTime := time.Now()
	if err := s.aggregator.AggregateMonthlyUsage(ctx); err != nil {
		s.logger.Errorw("monthly usage aggregation failed",
			"error", err,
			"duration", time.Since(startTime),
		)
		return
	}

	s.logger.Infow("monthly usage aggregation completed successfully",
		"duration", time.Since(startTime),
	)
}

// executeDataCleanup performs the data cleanup task.
func (s *UsageAggregationScheduler) executeDataCleanup(ctx context.Context) {
	s.logger.Infow("executing data cleanup",
		"retention_days", s.retentionDays,
	)

	startTime := time.Now()
	if err := s.aggregator.CleanupOldUsageData(ctx, s.retentionDays); err != nil {
		s.logger.Errorw("data cleanup failed",
			"error", err,
			"duration", time.Since(startTime),
			"retention_days", s.retentionDays,
		)
		return
	}

	s.logger.Infow("data cleanup completed successfully",
		"duration", time.Since(startTime),
		"retention_days", s.retentionDays,
	)
}
