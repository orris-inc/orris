package scheduler

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// ReminderProcessor defines the interface for processing reminders
type ReminderProcessor interface {
	ProcessReminders(ctx context.Context) error
}

// ReminderScheduler runs periodic reminder checks
type ReminderScheduler struct {
	processor ReminderProcessor
	logger    logger.Interface
	stopChan  chan struct{}
	interval  time.Duration
}

// NewReminderScheduler creates a new reminder scheduler
func NewReminderScheduler(
	processor ReminderProcessor,
	logger logger.Interface,
) *ReminderScheduler {
	return &ReminderScheduler{
		processor: processor,
		logger:    logger,
		stopChan:  make(chan struct{}),
		interval:  6 * time.Hour, // Run every 6 hours
	}
}

// Start starts the scheduler
func (s *ReminderScheduler) Start(ctx context.Context) {
	s.logger.Infow("starting reminder scheduler", "interval", s.interval)

	// Run immediately on start
	s.processReminders(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("reminder scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("reminder scheduler stopped")
			return
		case <-ticker.C:
			s.processReminders(ctx)
		}
	}
}

// Stop stops the scheduler
func (s *ReminderScheduler) Stop() {
	close(s.stopChan)
}

func (s *ReminderScheduler) processReminders(ctx context.Context) {
	s.logger.Debugw("processing reminders task started")

	if err := s.processor.ProcessReminders(ctx); err != nil {
		s.logger.Errorw("failed to process reminders", "error", err)
		return
	}

	s.logger.Debugw("reminders processed successfully")
}
