package services

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// RuleTrafficNumShards is the number of shards for rule traffic buffer partitioning.
	RuleTrafficNumShards = 16

	// RuleTrafficFlushInterval is the interval for flushing rule traffic data to Redis.
	RuleTrafficFlushInterval = 5 * time.Second

	// RuleTrafficMaxRetryCount is the maximum number of flush retry attempts before dropping data.
	// After this many failed attempts, data will be dropped with a warning log.
	RuleTrafficMaxRetryCount = 10
)

// RuleTrafficEntry represents a single rule traffic record with retry tracking.
type RuleTrafficEntry struct {
	RuleID     uint
	Upload     int64
	Download   int64
	RetryCount int // Number of failed flush attempts
}

// RuleTrafficCacheWriter defines the interface for writing rule traffic data to Redis.
type RuleTrafficCacheWriter interface {
	IncrementRuleTraffic(ctx context.Context, ruleID uint, upload, download int64) error
}

// ruleBufferShard is a single shard containing rule traffic entries with its own mutex.
type ruleBufferShard struct {
	mu      sync.Mutex
	entries map[uint]*RuleTrafficEntry // ruleID -> accumulated traffic
}

// RuleTrafficBuffer is a sharded in-memory buffer for rule traffic data.
type RuleTrafficBuffer struct {
	shards      [RuleTrafficNumShards]*ruleBufferShard
	cache       RuleTrafficCacheWriter
	logger      logger.Interface
	flushTicker *time.Ticker
	done        chan struct{}
	wg          sync.WaitGroup
}

// NewRuleTrafficBuffer creates a new RuleTrafficBuffer instance.
func NewRuleTrafficBuffer(cache RuleTrafficCacheWriter, log logger.Interface) *RuleTrafficBuffer {
	b := &RuleTrafficBuffer{
		cache:       cache,
		logger:      log,
		flushTicker: time.NewTicker(RuleTrafficFlushInterval),
		done:        make(chan struct{}),
	}

	// Initialize all shards
	for i := 0; i < RuleTrafficNumShards; i++ {
		b.shards[i] = &ruleBufferShard{
			entries: make(map[uint]*RuleTrafficEntry),
		}
	}

	return b
}

// getShard returns the shard for a given ruleID using modulo sharding.
func (b *RuleTrafficBuffer) getShard(ruleID uint) *ruleBufferShard {
	return b.shards[ruleID%RuleTrafficNumShards]
}

// Add adds a traffic entry to the buffer (thread-safe).
func (b *RuleTrafficBuffer) Add(entry *RuleTrafficEntry) {
	if entry == nil {
		return
	}
	b.AddTraffic(entry.RuleID, entry.Upload, entry.Download)
}

// AddTraffic adds traffic data to the buffer (thread-safe).
// This method is used by TrafficMessageHandler via the RuleTrafficBufferWriter interface.
func (b *RuleTrafficBuffer) AddTraffic(ruleID uint, upload, download int64) {
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
		shard.entries[ruleID] = &RuleTrafficEntry{
			RuleID:   ruleID,
			Upload:   upload,
			Download: download,
		}
	}
}

// Start starts the background flush goroutine.
func (b *RuleTrafficBuffer) Start() {
	b.wg.Add(1)
	go b.flushLoop()
	b.logger.Infow("rule traffic buffer started",
		"shards", RuleTrafficNumShards,
		"flush_interval", RuleTrafficFlushInterval.String(),
	)
}

// Stop stops the buffer and flushes remaining data.
func (b *RuleTrafficBuffer) Stop() {
	close(b.done)
	b.wg.Wait()
	b.flushTicker.Stop()

	// Final flush to ensure no data is lost
	b.flush()

	b.logger.Infow("rule traffic buffer stopped")
}

// flushLoop is the background loop that periodically flushes traffic data.
func (b *RuleTrafficBuffer) flushLoop() {
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
func (b *RuleTrafficBuffer) flush() {
	ctx := context.Background()
	flushedCount := 0
	failedCount := 0
	droppedCount := 0

	for i := 0; i < RuleTrafficNumShards; i++ {
		shard := b.shards[i]

		// Fast swap to minimize lock hold time
		shard.mu.Lock()
		entries := shard.entries
		shard.entries = make(map[uint]*RuleTrafficEntry)
		shard.mu.Unlock()

		for _, entry := range entries {
			if entry.Upload > 0 || entry.Download > 0 {
				if err := b.cache.IncrementRuleTraffic(ctx, entry.RuleID, entry.Upload, entry.Download); err != nil {
					entry.RetryCount++
					if entry.RetryCount >= RuleTrafficMaxRetryCount {
						// Drop data after max retries to prevent memory accumulation
						b.logger.Errorw("rule traffic data dropped after max retries",
							"rule_id", entry.RuleID,
							"upload", entry.Upload,
							"download", entry.Download,
							"retry_count", entry.RetryCount,
							"error", err,
						)
						droppedCount++
						continue
					}
					b.logger.Warnw("failed to flush rule traffic to redis, will retry",
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
		b.logger.Debugw("rule traffic buffer flushed to redis",
			"flushed_count", flushedCount,
			"failed_count", failedCount,
			"dropped_count", droppedCount,
		)
	}
}

// reAddEntry re-adds a failed entry back to its shard for retry.
func (b *RuleTrafficBuffer) reAddEntry(entry *RuleTrafficEntry) {
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
		shard.entries[entry.RuleID] = &RuleTrafficEntry{
			RuleID:     entry.RuleID,
			Upload:     entry.Upload,
			Download:   entry.Download,
			RetryCount: entry.RetryCount,
		}
	}
}
