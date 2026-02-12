package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// defaultWorkerCount is the number of concurrent workers for processing updates.
	// Updates are dispatched to workers by user affinity (userID % workerCount)
	// to ensure same-user ordering while allowing cross-user concurrency.
	defaultWorkerCount = 4
)

var errMuteServiceNotConfigured = errors.New("mute service not configured")

// OffsetStore persists polling offset across restarts.
type OffsetStore interface {
	GetOffset(ctx context.Context) (int64, error)
	SaveOffset(ctx context.Context, offset int64) error
}

// UpdateHandler defines the interface for handling Telegram updates
type UpdateHandler interface {
	HandleUpdate(ctx context.Context, update *Update) error
}

// PollingService handles long polling for Telegram updates
type PollingService struct {
	botService         *BotService
	handler            UpdateHandler
	logger             logger.Interface
	offsetStore        OffsetStore // nil = in-memory only
	pollTimeout        int
	stopChan           chan struct{}
	cancelFunc         context.CancelFunc // Used to cancel ongoing HTTP requests during shutdown
	wg                 sync.WaitGroup
	lastUpdateID       int64
	processedWatermark int64 // highest update_id processed in this session (dedup safety net)
	workerCount        int
	isRunning          bool
	runningMu          sync.Mutex
}

// NewPollingService creates a new polling service.
// offsetStore is optional — pass nil for in-memory only (backward compatible).
func NewPollingService(
	botService *BotService,
	handler UpdateHandler,
	logger logger.Interface,
	offsetStore OffsetStore,
) *PollingService {
	return &PollingService{
		botService:  botService,
		handler:     handler,
		logger:      logger,
		offsetStore: offsetStore,
		pollTimeout: 30, // 30 seconds long polling timeout
		stopChan:    make(chan struct{}),
		workerCount: defaultWorkerCount,
	}
}

// Start begins polling for updates
func (s *PollingService) Start(ctx context.Context) error {
	s.runningMu.Lock()
	if s.isRunning {
		s.runningMu.Unlock()
		return nil
	}
	s.isRunning = true
	// Recreate stopChan for restart capability
	s.stopChan = make(chan struct{})
	// Create a cancellable context for HTTP requests
	pollCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.runningMu.Unlock()

	// Load persisted offset from store
	if s.offsetStore != nil {
		saved, err := s.offsetStore.GetOffset(ctx)
		if err != nil {
			s.logger.Warnw("failed to load polling offset, starting from 0", "error", err)
		} else if saved > 0 {
			s.lastUpdateID = saved
			s.processedWatermark = saved
			s.logger.Infow("loaded polling offset from store", "offset", saved)
		}
	}

	// Delete any existing webhook before starting polling
	if err := s.botService.DeleteWebhook(); err != nil {
		s.logger.Warnw("failed to delete webhook before polling", "error", err)
	}

	s.logger.Infow("starting telegram polling service",
		"timeout", s.pollTimeout,
		"workers", s.workerCount,
	)

	s.wg.Add(1)
	goroutine.SafeGo(s.logger, "telegram-poll-loop", func() {
		s.pollLoop(pollCtx)
	})

	return nil
}

// Stop stops the polling service
func (s *PollingService) Stop() {
	s.runningMu.Lock()
	if !s.isRunning {
		s.runningMu.Unlock()
		return
	}
	s.isRunning = false
	// Cancel ongoing HTTP requests first to unblock poll()
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.runningMu.Unlock()

	close(s.stopChan)
	s.wg.Wait()
	s.logger.Infow("telegram polling service stopped")
}

func (s *PollingService) pollLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("polling stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("polling stopped by stop signal")
			return
		default:
			s.poll(ctx)
		}
	}
}

func (s *PollingService) poll(ctx context.Context) {
	// Calculate offset: 0 for first poll (to get all pending updates), lastUpdateID+1 for subsequent polls
	offset := int64(0)
	if s.lastUpdateID > 0 {
		offset = s.lastUpdateID + 1
	}
	// Use context-aware GetUpdates for graceful shutdown support
	updates, err := s.botService.GetUpdatesWithContext(ctx, offset, s.pollTimeout)
	if err != nil {
		// Check if the error is due to context cancellation (graceful shutdown)
		if ctx.Err() != nil {
			return
		}
		s.logger.Errorw("failed to get updates", "error", err)
		// Wait a bit before retrying to avoid hammering the API on errors
		// Use select to respond to stop signals during wait
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-time.After(5 * time.Second):
			return
		}
	}

	if len(updates) == 0 {
		return
	}

	// Dedup: skip updates already processed (watermark safety net for restart overlap)
	filtered := updates[:0]
	for _, u := range updates {
		if u.UpdateID > s.processedWatermark {
			filtered = append(filtered, u)
		}
	}
	if len(filtered) == 0 {
		// Still advance lastUpdateID so Telegram won't resend these
		for _, u := range updates {
			if u.UpdateID > s.lastUpdateID {
				s.lastUpdateID = u.UpdateID
			}
		}
		return
	}

	// Dispatch updates to worker buckets by user affinity
	buckets := make([][]Update, s.workerCount)
	for i := range buckets {
		buckets[i] = make([]Update, 0)
	}
	var maxUpdateID int64
	for _, u := range filtered {
		idx := s.getUserAffinity(&u)
		buckets[idx] = append(buckets[idx], u)
		// Track max update ID (local var; commit to s.lastUpdateID after workers finish)
		if u.UpdateID > maxUpdateID {
			maxUpdateID = u.UpdateID
		}
	}

	// Process buckets concurrently
	var batchWg sync.WaitGroup
	for i, bucket := range buckets {
		if len(bucket) == 0 {
			continue
		}
		batchWg.Add(1)
		workerIdx := i
		workerBucket := bucket
		goroutine.SafeGo(s.logger, "telegram-worker-batch", func() {
			s.processWorkerBatch(ctx, &batchWg, workerIdx, workerBucket)
		})
	}
	batchWg.Wait()

	// Advance lastUpdateID and watermark only after all workers finished,
	// so a crash during processing won't skip unprocessed updates.
	s.lastUpdateID = maxUpdateID
	s.processedWatermark = maxUpdateID

	// Persist offset after processing batch.
	// Use a fresh context because the poll context may already be cancelled during shutdown.
	if s.offsetStore != nil && s.lastUpdateID > 0 {
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer saveCancel()
		if err := s.offsetStore.SaveOffset(saveCtx, s.lastUpdateID); err != nil {
			s.logger.Warnw("failed to save polling offset", "error", err)
		}
	}
}

// processWorkerBatch processes a slice of updates sequentially within one worker goroutine.
// Each goroutine has panic recovery to prevent a single update from crashing the entire service.
func (s *PollingService) processWorkerBatch(ctx context.Context, wg *sync.WaitGroup, workerIdx int, updates []Update) {
	defer wg.Done()

	for i := range updates {
		// Short-circuit remaining updates on shutdown to improve stop responsiveness
		if ctx.Err() != nil {
			return
		}

		func(u *Update) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Errorw("panic recovered in update handler",
						"worker", workerIdx,
						"update_id", u.UpdateID,
						"panic", fmt.Sprintf("%v", r),
					)
				}
			}()

			if err := s.handler.HandleUpdate(ctx, u); err != nil {
				s.logger.Errorw("failed to handle update",
					"worker", workerIdx,
					"update_id", u.UpdateID,
					"error", err,
				)
			}
		}(&updates[i])
	}
}

// getUserAffinity maps an update to a worker index by user ID.
// Same user always goes to the same worker, preserving per-user ordering.
func (s *PollingService) getUserAffinity(u *Update) int {
	var userID int64
	switch {
	case u.CallbackQuery != nil && u.CallbackQuery.From != nil:
		userID = u.CallbackQuery.From.ID
	case u.Message != nil && u.Message.From != nil:
		userID = u.Message.From.ID
	default:
		// Fallback: spread by update ID
		userID = u.UpdateID
	}
	// Ensure non-negative modulo
	idx := int(userID % int64(s.workerCount))
	if idx < 0 {
		idx += s.workerCount
	}
	return idx
}
