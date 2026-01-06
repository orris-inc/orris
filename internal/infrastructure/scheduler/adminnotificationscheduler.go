package scheduler

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminNotificationProcessor defines the interface for processing admin notifications
type AdminNotificationProcessor interface {
	// CheckOffline checks for offline nodes and agents, sends alerts
	CheckOffline(ctx context.Context) error
	// SendDailySummary sends daily business summary
	SendDailySummary(ctx context.Context) error
	// SendWeeklySummary sends weekly business summary
	SendWeeklySummary(ctx context.Context) error
}

// AdminNotificationScheduler runs periodic admin notification tasks
type AdminNotificationScheduler struct {
	processor            AdminNotificationProcessor
	logger               logger.Interface
	stopChan             chan struct{}
	offlineCheckInterval time.Duration
	dailySummaryHour     int // Hour to send daily summary (in business timezone)
	weeklySummaryHour    int // Hour to send weekly summary (in business timezone)
	weeklySummaryWeekday time.Weekday
}

// NewAdminNotificationScheduler creates a new admin notification scheduler
func NewAdminNotificationScheduler(
	processor AdminNotificationProcessor,
	logger logger.Interface,
) *AdminNotificationScheduler {
	return &AdminNotificationScheduler{
		processor:            processor,
		logger:               logger,
		stopChan:             make(chan struct{}),
		offlineCheckInterval: 2 * time.Minute, // Check every 2 minutes
		dailySummaryHour:     9,               // 09:00 business timezone
		weeklySummaryHour:    9,               // 09:00 business timezone
		weeklySummaryWeekday: time.Monday,     // Monday
	}
}

// Start starts the scheduler
func (s *AdminNotificationScheduler) Start(ctx context.Context) {
	s.logger.Infow("starting admin notification scheduler",
		"offline_check_interval", s.offlineCheckInterval,
		"daily_summary_hour", s.dailySummaryHour,
		"weekly_summary_hour", s.weeklySummaryHour,
		"weekly_summary_weekday", s.weeklySummaryWeekday,
	)

	// Start offline checker
	go s.runOfflineChecker(ctx)

	// Start daily summary scheduler
	go s.runDailySummaryScheduler(ctx)

	// Start weekly summary scheduler
	go s.runWeeklySummaryScheduler(ctx)
}

// Stop stops the scheduler
func (s *AdminNotificationScheduler) Stop() {
	close(s.stopChan)
}

// runOfflineChecker runs the offline check loop
func (s *AdminNotificationScheduler) runOfflineChecker(ctx context.Context) {
	// Run immediately on start
	s.checkOffline(ctx)

	ticker := time.NewTicker(s.offlineCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("offline checker stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("offline checker stopped")
			return
		case <-ticker.C:
			s.checkOffline(ctx)
		}
	}
}

// runDailySummaryScheduler runs the daily summary scheduler
func (s *AdminNotificationScheduler) runDailySummaryScheduler(ctx context.Context) {
	for {
		// Calculate next run time
		nextRun := s.nextDailySummaryTime()
		waitDuration := time.Until(nextRun)

		s.logger.Debugw("daily summary scheduled",
			"next_run", nextRun,
			"wait_duration", waitDuration,
		)

		timer := time.NewTimer(waitDuration)

		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Infow("daily summary scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			timer.Stop()
			s.logger.Infow("daily summary scheduler stopped")
			return
		case <-timer.C:
			s.sendDailySummary(ctx)
		}
	}
}

// runWeeklySummaryScheduler runs the weekly summary scheduler
func (s *AdminNotificationScheduler) runWeeklySummaryScheduler(ctx context.Context) {
	for {
		// Calculate next run time
		nextRun := s.nextWeeklySummaryTime()
		waitDuration := time.Until(nextRun)

		s.logger.Debugw("weekly summary scheduled",
			"next_run", nextRun,
			"wait_duration", waitDuration,
		)

		timer := time.NewTimer(waitDuration)

		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Infow("weekly summary scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			timer.Stop()
			s.logger.Infow("weekly summary scheduler stopped")
			return
		case <-timer.C:
			s.sendWeeklySummary(ctx)
		}
	}
}

// checkOffline performs the offline check
func (s *AdminNotificationScheduler) checkOffline(ctx context.Context) {
	s.logger.Debugw("starting offline check")

	if err := s.processor.CheckOffline(ctx); err != nil {
		s.logger.Errorw("failed to check offline", "error", err)
		return
	}

	s.logger.Debugw("offline check completed")
}

// sendDailySummary sends the daily summary
func (s *AdminNotificationScheduler) sendDailySummary(ctx context.Context) {
	s.logger.Infow("starting daily summary")

	if err := s.processor.SendDailySummary(ctx); err != nil {
		s.logger.Errorw("failed to send daily summary", "error", err)
		return
	}

	s.logger.Infow("daily summary sent successfully")
}

// sendWeeklySummary sends the weekly summary
func (s *AdminNotificationScheduler) sendWeeklySummary(ctx context.Context) {
	s.logger.Infow("starting weekly summary")

	if err := s.processor.SendWeeklySummary(ctx); err != nil {
		s.logger.Errorw("failed to send weekly summary", "error", err)
		return
	}

	s.logger.Infow("weekly summary sent successfully")
}

// nextDailySummaryTime calculates the next daily summary time
func (s *AdminNotificationScheduler) nextDailySummaryTime() time.Time {
	now := biztime.ToBizTimezone(biztime.NowUTC())
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), s.dailySummaryHour, 0, 0, 0, now.Location())

	// If target time has passed today, schedule for tomorrow
	if now.After(targetTime) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	return targetTime
}

// nextWeeklySummaryTime calculates the next weekly summary time
func (s *AdminNotificationScheduler) nextWeeklySummaryTime() time.Time {
	now := biztime.ToBizTimezone(biztime.NowUTC())

	// Find next target weekday
	daysUntilTarget := int(s.weeklySummaryWeekday - now.Weekday())
	if daysUntilTarget < 0 {
		daysUntilTarget += 7
	}

	targetTime := time.Date(now.Year(), now.Month(), now.Day()+daysUntilTarget, s.weeklySummaryHour, 0, 0, 0, now.Location())

	// If it's the target weekday but the time has passed, schedule for next week
	if daysUntilTarget == 0 && now.After(targetTime) {
		targetTime = targetTime.Add(7 * 24 * time.Hour)
	}

	return targetTime
}
