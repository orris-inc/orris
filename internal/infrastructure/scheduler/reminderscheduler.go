package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// ReminderProcessor defines the interface for processing reminders
type ReminderProcessor interface {
	ProcessReminders(ctx context.Context) error
}

// ReminderScheduler runs periodic reminder checks
type ReminderScheduler struct {
	processor    ReminderProcessor
	logger       logger.Interface
	stopChan     chan struct{}
	interval     time.Duration
	cancelFn     context.CancelFunc
	wg           sync.WaitGroup
	stopTimeout  time.Duration
}

// NewReminderScheduler creates a new reminder scheduler
func NewReminderScheduler(
	processor ReminderProcessor,
	logger logger.Interface,
) *ReminderScheduler {
	return &ReminderScheduler{
		processor:   processor,
		logger:      logger,
		stopChan:    make(chan struct{}),
		interval:    6 * time.Hour,     // Run every 6 hours
		stopTimeout: 3 * time.Second,   // Wait up to 3 seconds for graceful stop
	}
}

// Start starts the scheduler
func (s *ReminderScheduler) Start(ctx context.Context) {
	s.logger.Infow("starting reminder scheduler", "interval", s.interval)

	// Create a cancellable context for the scheduler
	schedulerCtx, cancel := context.WithCancel(ctx)
	s.cancelFn = cancel

	// Run immediately on start
	s.runTask(schedulerCtx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-schedulerCtx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.runTask(schedulerCtx)
		}
	}
}

// runTask runs processReminders in a goroutine with WaitGroup tracking
func (s *ReminderScheduler) runTask(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processReminders(ctx)
	}()
}

// Stop stops the scheduler with graceful shutdown
// Waits up to stopTimeout for running tasks, then forces cancellation
func (s *ReminderScheduler) Stop() {
	close(s.stopChan)

	// Wait for running tasks with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Tasks completed gracefully
	case <-time.After(s.stopTimeout):
		// Timeout: cancel running tasks
		if s.cancelFn != nil {
			s.cancelFn()
		}
	}
}

func (s *ReminderScheduler) processReminders(ctx context.Context) {
	s.logger.Debugw("processing reminders task started")

	if err := s.processor.ProcessReminders(ctx); err != nil {
		// Don't log error if context was cancelled (graceful shutdown)
		if ctx.Err() != nil {
			return
		}
		s.logger.Errorw("failed to process reminders", "error", err)
		return
	}

	s.logger.Debugw("reminders processed successfully")
}
