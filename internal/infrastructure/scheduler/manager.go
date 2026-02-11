// Package scheduler provides unified scheduler management using gocron v2.
package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// BatchJob defines the interface for a scheduled batch processing job.
// Each Execute call processes a batch and returns the number of items processed.
type BatchJob interface {
	Execute(ctx context.Context) (int, error)
}

// SchedulerManager manages all scheduled jobs using gocron v2.
// It unifies the previously separate scheduler implementations into a single
// scheduler instance with better management and reduced boilerplate.
//
// Note: USDT payment monitoring is managed separately by USDTServiceManager
// to support hot-reload of configuration.
type SchedulerManager struct {
	scheduler gocron.Scheduler
	logger    logger.Interface

	// Track whether the scheduler has been started
	started   bool
	startedMu sync.RWMutex
}

// NewSchedulerManager creates a new SchedulerManager instance.
// It initializes gocron with the business timezone for cron expressions.
func NewSchedulerManager(log logger.Interface) (*SchedulerManager, error) {
	scheduler, err := gocron.NewScheduler(
		gocron.WithLocation(biztime.Location()),
	)
	if err != nil {
		return nil, err
	}

	return &SchedulerManager{
		scheduler: scheduler,
		logger:    log,
	}, nil
}

// ========================================
// Payment Jobs (5 min interval, start immediately)
// ========================================

// RegisterPaymentJobs registers payment-related scheduled jobs:
// - Expire pending payments that have passed their expiration time
// - Cancel subscriptions with unpaid payments after 24-hour grace period
// - Retry failed subscription activations for paid non-USDT payments
func (m *SchedulerManager) RegisterPaymentJobs(
	expirePaymentsJob BatchJob,
	cancelUnpaidSubsJob BatchJob,
	retryActivationJob BatchJob,
) error {
	_, err := m.scheduler.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			m.processPaymentTasks(ctx, expirePaymentsJob, cancelUnpaidSubsJob, retryActivationJob)
		}),
		gocron.WithStartAt(gocron.WithStartImmediately()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("payment", "expire", "cancel-unpaid", "retry-activation"),
		gocron.WithName("payment-processor"),
	)
	if err != nil {
		return err
	}

	m.logger.Infow("registered payment jobs", "interval", "5m")
	return nil
}

func (m *SchedulerManager) processPaymentTasks(
	ctx context.Context,
	expirePaymentsJob BatchJob,
	cancelUnpaidSubsJob BatchJob,
	retryActivationJob BatchJob,
) {
	m.logger.Debugw("processing payment tasks started")

	startTime := biztime.NowUTC()

	// Step 1: Expire pending payments that have passed their expiration time
	expiredCount, err := expirePaymentsJob.Execute(ctx)
	if err != nil {
		m.logger.Errorw("failed to process expired payments",
			"error", err,
			"duration", time.Since(startTime),
		)
	} else if expiredCount > 0 {
		m.logger.Infow("expired payments processed",
			"count", expiredCount,
			"duration", time.Since(startTime),
		)
	}

	// Step 2: Cancel subscriptions with unpaid payments beyond grace period
	cancelledCount, err := cancelUnpaidSubsJob.Execute(ctx)
	if err != nil {
		m.logger.Errorw("failed to cancel unpaid subscriptions",
			"error", err,
		)
	} else if cancelledCount > 0 {
		m.logger.Infow("unpaid subscriptions cancelled",
			"count", cancelledCount,
		)
	}

	// Step 3: Retry failed subscription activations for paid non-USDT payments
	if retryActivationJob != nil {
		retryCount, err := retryActivationJob.Execute(ctx)
		if err != nil {
			m.logger.Errorw("failed to retry subscription activations",
				"error", err,
			)
		} else if retryCount > 0 {
			m.logger.Infow("subscription activations retried",
				"count", retryCount,
			)
		}
	}
}

// ========================================
// Subscription Jobs (24h interval, start immediately)
// ========================================

// RegisterSubscriptionJobs registers subscription maintenance jobs:
// - Mark expired subscriptions (data consistency for reports/statistics)
func (m *SchedulerManager) RegisterSubscriptionJobs(
	expireSubscriptionsJob BatchJob,
) error {
	_, err := m.scheduler.NewJob(
		gocron.DurationJob(24*time.Hour),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			m.processExpiredSubscriptions(ctx, expireSubscriptionsJob)
		}),
		gocron.WithStartAt(gocron.WithStartImmediately()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("subscription", "expire"),
		gocron.WithName("subscription-expire"),
	)
	if err != nil {
		return err
	}

	m.logger.Infow("registered subscription jobs", "interval", "24h")
	return nil
}

func (m *SchedulerManager) processExpiredSubscriptions(
	ctx context.Context,
	expireSubscriptionsJob BatchJob,
) {
	m.logger.Debugw("processing expired subscriptions task started")

	startTime := biztime.NowUTC()

	expiredCount, err := expireSubscriptionsJob.Execute(ctx)
	if err != nil {
		m.logger.Errorw("failed to process expired subscriptions",
			"error", err,
			"duration", time.Since(startTime),
		)
		return
	}

	if expiredCount > 0 {
		m.logger.Infow("expired subscriptions processed",
			"count", expiredCount,
			"duration", time.Since(startTime),
		)
	} else {
		m.logger.Debugw("no expired subscriptions to process",
			"duration", time.Since(startTime),
		)
	}
}

// ========================================
// Usage Aggregation Jobs (cron-based)
// ========================================

// UsageAggregator defines the interface for aggregating usage data.
type UsageAggregator interface {
	// AggregateDailyUsage aggregates hourly data from yesterday into daily stats.
	AggregateDailyUsage(ctx context.Context) error
	// AggregateMonthlyUsage aggregates daily data from last month into monthly stats.
	AggregateMonthlyUsage(ctx context.Context) error
	// CleanupOldUsageData deletes raw usage records older than the specified retention days.
	CleanupOldUsageData(ctx context.Context, retentionDays int) error
}

// DefaultRetentionDays is the default number of days to retain raw usage data.
const DefaultRetentionDays = 90

// RegisterUsageAggregationJobs registers usage aggregation jobs:
// - Daily aggregation: runs at 03:00 business timezone every day
// - Monthly aggregation: runs at 04:00 business timezone on the 1st of each month
// - Data cleanup: runs at 05:00 business timezone every day
func (m *SchedulerManager) RegisterUsageAggregationJobs(
	aggregator UsageAggregator,
	retentionDays int,
) error {
	if retentionDays <= 0 {
		retentionDays = DefaultRetentionDays
	}

	// Daily aggregation at 03:00 business timezone
	_, err := m.scheduler.NewJob(
		gocron.CronJob("0 3 * * *", false),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			m.executeDailyAggregation(ctx, aggregator)
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("usage", "daily-aggregation"),
		gocron.WithName("usage-daily-aggregation"),
	)
	if err != nil {
		return err
	}

	// Monthly aggregation at 04:00 on the 1st of each month
	_, err = m.scheduler.NewJob(
		gocron.CronJob("0 4 1 * *", false),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			m.executeMonthlyAggregation(ctx, aggregator)
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("usage", "monthly-aggregation"),
		gocron.WithName("usage-monthly-aggregation"),
	)
	if err != nil {
		return err
	}

	// Data cleanup at 05:00 every day
	_, err = m.scheduler.NewJob(
		gocron.CronJob("0 5 * * *", false),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			m.executeDataCleanup(ctx, aggregator, retentionDays)
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("usage", "cleanup"),
		gocron.WithName("usage-data-cleanup"),
	)
	if err != nil {
		return err
	}

	m.logger.Infow("registered usage aggregation jobs",
		"daily_aggregation", "03:00",
		"monthly_aggregation", "04:00 on 1st",
		"cleanup", "05:00",
		"retention_days", retentionDays,
	)
	return nil
}

func (m *SchedulerManager) executeDailyAggregation(ctx context.Context, aggregator UsageAggregator) {
	m.logger.Debugw("executing daily usage aggregation")

	startTime := biztime.NowUTC()
	if err := aggregator.AggregateDailyUsage(ctx); err != nil {
		m.logger.Errorw("daily usage aggregation failed",
			"error", err,
			"duration", time.Since(startTime),
		)
		return
	}

	m.logger.Infow("daily usage aggregation completed successfully",
		"duration", time.Since(startTime),
	)
}

func (m *SchedulerManager) executeMonthlyAggregation(ctx context.Context, aggregator UsageAggregator) {
	m.logger.Debugw("executing monthly usage aggregation")

	startTime := biztime.NowUTC()
	if err := aggregator.AggregateMonthlyUsage(ctx); err != nil {
		m.logger.Errorw("monthly usage aggregation failed",
			"error", err,
			"duration", time.Since(startTime),
		)
		return
	}

	m.logger.Infow("monthly usage aggregation completed successfully",
		"duration", time.Since(startTime),
	)
}

func (m *SchedulerManager) executeDataCleanup(ctx context.Context, aggregator UsageAggregator, retentionDays int) {
	m.logger.Debugw("executing data cleanup",
		"retention_days", retentionDays,
	)

	startTime := biztime.NowUTC()
	if err := aggregator.CleanupOldUsageData(ctx, retentionDays); err != nil {
		m.logger.Errorw("data cleanup failed",
			"error", err,
			"duration", time.Since(startTime),
			"retention_days", retentionDays,
		)
		return
	}

	m.logger.Infow("data cleanup completed successfully",
		"duration", time.Since(startTime),
		"retention_days", retentionDays,
	)
}

// ========================================
// Reminder Jobs (6h interval, start immediately)
// ========================================

// ReminderProcessor defines the interface for processing reminders.
type ReminderProcessor interface {
	ProcessReminders(ctx context.Context) error
}

// RegisterReminderJobs registers reminder processing jobs:
// - Process reminders every 6 hours
func (m *SchedulerManager) RegisterReminderJobs(
	processor ReminderProcessor,
) error {
	_, err := m.scheduler.NewJob(
		gocron.DurationJob(6*time.Hour),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			m.processReminders(ctx, processor)
		}),
		gocron.WithStartAt(gocron.WithStartImmediately()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("reminder", "telegram"),
		gocron.WithName("reminder-processor"),
	)
	if err != nil {
		return err
	}

	m.logger.Infow("registered reminder jobs", "interval", "6h")
	return nil
}

func (m *SchedulerManager) processReminders(ctx context.Context, processor ReminderProcessor) {
	m.logger.Debugw("processing reminders task started")

	if err := processor.ProcessReminders(ctx); err != nil {
		// Don't log error if context was cancelled (graceful shutdown)
		if ctx.Err() != nil {
			return
		}
		m.logger.Errorw("failed to process reminders", "error", err)
		return
	}

	m.logger.Debugw("reminders processed successfully")
}

// ========================================
// Admin Notification Jobs (mixed intervals)
// ========================================

// AdminNotificationProcessor defines the interface for processing admin notifications.
type AdminNotificationProcessor interface {
	// CheckOffline checks for offline nodes and agents, sends alerts
	CheckOffline(ctx context.Context) error
	// CheckExpiring checks for expiring resources, sends alerts
	CheckExpiring(ctx context.Context) error
	// SendDailySummary sends daily business summary
	SendDailySummary(ctx context.Context) error
	// SendWeeklySummary sends weekly business summary
	SendWeeklySummary(ctx context.Context) error
}

// RegisterAdminNotificationJobs registers admin notification jobs:
// - Offline check: every 2 minutes
// - Expiring check: at 08:00 business timezone daily
// - Daily summary: at 09:00 business timezone
// - Weekly summary: Monday 09:00 business timezone
func (m *SchedulerManager) RegisterAdminNotificationJobs(
	processor AdminNotificationProcessor,
) error {
	// Offline check every 2 minutes, start immediately
	_, err := m.scheduler.NewJob(
		gocron.DurationJob(2*time.Minute),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()
			m.checkOffline(ctx, processor)
		}),
		gocron.WithStartAt(gocron.WithStartImmediately()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("admin", "offline-check"),
		gocron.WithName("admin-offline-check"),
	)
	if err != nil {
		return err
	}

	// Expiring check at 08:00 daily (before daily summary)
	_, err = m.scheduler.NewJob(
		gocron.CronJob("0 8 * * *", false),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			m.checkExpiring(ctx, processor)
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("admin", "expiring-check"),
		gocron.WithName("admin-expiring-check"),
	)
	if err != nil {
		return err
	}

	// Daily summary: hourly trigger, UseCase filters by per-binding configured hour
	_, err = m.scheduler.NewJob(
		gocron.CronJob("0 * * * *", false),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			m.sendDailySummary(ctx, processor)
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("admin", "daily-summary"),
		gocron.WithName("admin-daily-summary"),
	)
	if err != nil {
		return err
	}

	// Weekly summary: hourly trigger, UseCase filters by per-binding configured hour and weekday
	_, err = m.scheduler.NewJob(
		gocron.CronJob("0 * * * *", false),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			m.sendWeeklySummary(ctx, processor)
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithTags("admin", "weekly-summary"),
		gocron.WithName("admin-weekly-summary"),
	)
	if err != nil {
		return err
	}

	m.logger.Infow("registered admin notification jobs",
		"offline_check", "2m",
		"expiring_check", "08:00",
		"daily_summary", "hourly (per-binding hour config)",
		"weekly_summary", "hourly (per-binding hour/weekday config)",
	)
	return nil
}

func (m *SchedulerManager) checkOffline(ctx context.Context, processor AdminNotificationProcessor) {
	m.logger.Debugw("starting offline check")

	if err := processor.CheckOffline(ctx); err != nil {
		m.logger.Errorw("failed to check offline", "error", err)
		return
	}

	m.logger.Debugw("offline check completed")
}

func (m *SchedulerManager) checkExpiring(ctx context.Context, processor AdminNotificationProcessor) {
	m.logger.Debugw("starting expiring check")

	if err := processor.CheckExpiring(ctx); err != nil {
		m.logger.Errorw("failed to check expiring", "error", err)
		return
	}

	m.logger.Infow("expiring check completed")
}

func (m *SchedulerManager) sendDailySummary(ctx context.Context, processor AdminNotificationProcessor) {
	if err := processor.SendDailySummary(ctx); err != nil {
		m.logger.Errorw("failed to send daily summary", "error", err)
		return
	}

	m.logger.Infow("daily summary sent successfully")
}

func (m *SchedulerManager) sendWeeklySummary(ctx context.Context, processor AdminNotificationProcessor) {
	if err := processor.SendWeeklySummary(ctx); err != nil {
		m.logger.Errorw("failed to send weekly summary", "error", err)
		return
	}

	m.logger.Infow("weekly summary sent successfully")
}

// ========================================
// Scheduler Lifecycle Methods
// ========================================

// Start starts the scheduler and all registered jobs.
func (m *SchedulerManager) Start() {
	m.startedMu.Lock()
	defer m.startedMu.Unlock()

	if m.started {
		return
	}

	m.scheduler.Start()
	m.started = true
	m.logger.Infow("scheduler manager started", "job_count", len(m.scheduler.Jobs()))
}

// Stop gracefully stops the scheduler.
// It waits for all running jobs to complete before returning.
func (m *SchedulerManager) Stop() error {
	m.startedMu.Lock()
	defer m.startedMu.Unlock()

	if !m.started {
		return nil
	}

	m.logger.Infow("stopping scheduler manager")

	// Shutdown scheduler and wait for running jobs
	err := m.scheduler.Shutdown()
	m.started = false

	if err != nil {
		m.logger.Errorw("scheduler manager shutdown with error", "error", err)
		return err
	}

	m.logger.Infow("scheduler manager stopped")
	return nil
}

// IsStarted returns whether the scheduler is running.
func (m *SchedulerManager) IsStarted() bool {
	m.startedMu.RLock()
	defer m.startedMu.RUnlock()
	return m.started
}

// Jobs returns all registered jobs for inspection.
func (m *SchedulerManager) Jobs() []gocron.Job {
	return m.scheduler.Jobs()
}
