package services

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// NumShards is the number of shards for traffic buffer partitioning.
	NumShards = 16

	// FlushInterval is the interval for flushing traffic data to Redis.
	FlushInterval = 5 * time.Second

	// MaxRetryCount is the maximum number of flush retry attempts before dropping data.
	// After this many failed attempts, data will be dropped with a warning log.
	MaxRetryCount = 10
)

// TrafficEntry represents a single traffic record with retry tracking.
type TrafficEntry struct {
	RuleID     uint
	Upload     int64
	Download   int64
	RetryCount int // Number of failed flush attempts
}

// ForwardTrafficCacheWriter defines the interface for writing traffic data to Redis (decoupled).
type ForwardTrafficCacheWriter interface {
	IncrementRuleTraffic(ctx context.Context, ruleID uint, upload, download int64) error
}

// bufferShard is a single shard containing traffic entries with its own mutex.
type bufferShard struct {
	mu      sync.Mutex
	entries map[uint]*TrafficEntry // ruleID -> accumulated traffic
}

// TrafficBuffer is a sharded in-memory buffer for traffic data.
type TrafficBuffer struct {
	shards      [NumShards]*bufferShard
	cache       ForwardTrafficCacheWriter
	logger      logger.Interface
	flushTicker *time.Ticker
	done        chan struct{}
	wg          sync.WaitGroup
}

// NewTrafficBuffer creates a new TrafficBuffer instance.
func NewTrafficBuffer(cache ForwardTrafficCacheWriter, log logger.Interface) *TrafficBuffer {
	b := &TrafficBuffer{
		cache:       cache,
		logger:      log,
		flushTicker: time.NewTicker(FlushInterval),
		done:        make(chan struct{}),
	}

	// Initialize all shards
	for i := 0; i < NumShards; i++ {
		b.shards[i] = &bufferShard{
			entries: make(map[uint]*TrafficEntry),
		}
	}

	return b
}

// getShard returns the shard for a given ruleID using modulo sharding.
func (b *TrafficBuffer) getShard(ruleID uint) *bufferShard {
	return b.shards[ruleID%NumShards]
}

// Add adds a traffic entry to the buffer (thread-safe).
func (b *TrafficBuffer) Add(entry *TrafficEntry) {
	if entry == nil {
		return
	}
	b.AddTraffic(entry.RuleID, entry.Upload, entry.Download)
}

// AddTraffic adds traffic data to the buffer (thread-safe).
// This method is used by TrafficMessageHandler via the TrafficBufferWriter interface.
func (b *TrafficBuffer) AddTraffic(ruleID uint, upload, download int64) {
	// Skip zero traffic entries
	if upload == 0 && download == 0 {
		return
	}

	shard := b.getShard(ruleID)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if existing, ok := shard.entries[ruleID]; ok {
		existing.Upload += upload
		existing.Download += download
	} else {
		shard.entries[ruleID] = &TrafficEntry{
			RuleID:   ruleID,
			Upload:   upload,
			Download: download,
		}
	}
}

// Start starts the background flush goroutine.
func (b *TrafficBuffer) Start() {
	b.wg.Add(1)
	go b.flushLoop()
	b.logger.Infow("traffic buffer started",
		"shards", NumShards,
		"flush_interval", FlushInterval.String(),
	)
}

// Stop stops the buffer and flushes remaining data.
func (b *TrafficBuffer) Stop() {
	close(b.done)
	b.wg.Wait()
	b.flushTicker.Stop()

	// Final flush to ensure no data is lost
	b.flush()

	b.logger.Infow("traffic buffer stopped")
}

// flushLoop is the background loop that periodically flushes traffic data.
func (b *TrafficBuffer) flushLoop() {
	defer b.wg.Done()
	for {
		select {
		case <-b.flushTicker.C:
			b.flush()
		case <-b.done:
			return
		}
	}
}

// flush flushes all accumulated traffic data to Redis.
func (b *TrafficBuffer) flush() {
	ctx := context.Background()
	flushedCount := 0
	failedCount := 0
	droppedCount := 0

	for i := 0; i < NumShards; i++ {
		shard := b.shards[i]

		// Fast swap to minimize lock hold time
		shard.mu.Lock()
		entries := shard.entries
		shard.entries = make(map[uint]*TrafficEntry)
		shard.mu.Unlock()

		for _, entry := range entries {
			if entry.Upload > 0 || entry.Download > 0 {
				if err := b.cache.IncrementRuleTraffic(ctx, entry.RuleID, entry.Upload, entry.Download); err != nil {
					entry.RetryCount++
					if entry.RetryCount >= MaxRetryCount {
						// Drop data after max retries to prevent memory accumulation
						b.logger.Errorw("traffic data dropped after max retries",
							"rule_id", entry.RuleID,
							"upload", entry.Upload,
							"download", entry.Download,
							"retry_count", entry.RetryCount,
							"error", err,
						)
						droppedCount++
						continue
					}
					b.logger.Warnw("failed to flush traffic to redis, will retry",
						"rule_id", entry.RuleID,
						"upload", entry.Upload,
						"download", entry.Download,
						"retry_count", entry.RetryCount,
						"error", err,
					)
					// Re-add failed entry to shard for retry on next flush
					b.reAddEntry(entry)
					failedCount++
					continue
				}
				flushedCount++
			}
		}
	}

	if flushedCount > 0 || failedCount > 0 || droppedCount > 0 {
		b.logger.Debugw("traffic buffer flushed to redis",
			"flushed_count", flushedCount,
			"failed_count", failedCount,
			"dropped_count", droppedCount,
		)
	}
}

// reAddEntry re-adds a failed entry back to its shard for retry.
func (b *TrafficBuffer) reAddEntry(entry *TrafficEntry) {
	shard := b.getShard(entry.RuleID)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if existing, ok := shard.entries[entry.RuleID]; ok {
		// Merge with any new traffic that arrived during flush
		existing.Upload += entry.Upload
		existing.Download += entry.Download
		// Keep the higher retry count to track total failures
		if entry.RetryCount > existing.RetryCount {
			existing.RetryCount = entry.RetryCount
		}
	} else {
		shard.entries[entry.RuleID] = &TrafficEntry{
			RuleID:     entry.RuleID,
			Upload:     entry.Upload,
			Download:   entry.Download,
			RetryCount: entry.RetryCount,
		}
	}
}
