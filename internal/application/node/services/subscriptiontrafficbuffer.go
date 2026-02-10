package services

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// SubscriptionTrafficNumShards is the number of shards for subscription traffic buffer partitioning.
	SubscriptionTrafficNumShards = 16

	// SubscriptionTrafficFlushInterval is the interval for flushing subscription traffic data to Redis.
	SubscriptionTrafficFlushInterval = 5 * time.Second

	// SubscriptionTrafficMaxRetryCount is the maximum number of flush retry attempts before dropping data.
	SubscriptionTrafficMaxRetryCount = 10
)

// SubscriptionTrafficEntry represents a single subscription traffic record with retry tracking.
type SubscriptionTrafficEntry struct {
	NodeID         uint
	SubscriptionID uint
	Upload         int64
	Download       int64
	RetryCount     int // Number of failed flush attempts
}

// subscriptionTrafficKey is used as map key for aggregation.
type subscriptionTrafficKey struct {
	NodeID         uint
	SubscriptionID uint
}

// SubscriptionTrafficCacheWriter defines the interface for writing traffic data to Redis.
type SubscriptionTrafficCacheWriter interface {
	IncrementSubscriptionTraffic(ctx context.Context, nodeID, subscriptionID uint, upload, download int64) error
	BatchIncrementSubscriptionTraffic(ctx context.Context, entries []cache.SubscriptionTrafficBatchEntry) error
}

// subscriptionBufferShard is a single shard containing traffic entries with its own mutex.
type subscriptionBufferShard struct {
	mu      sync.Mutex
	entries map[subscriptionTrafficKey]*SubscriptionTrafficEntry
}

// SubscriptionTrafficBuffer is a sharded in-memory buffer for subscription traffic data.
type SubscriptionTrafficBuffer struct {
	shards      [SubscriptionTrafficNumShards]*subscriptionBufferShard
	cache       SubscriptionTrafficCacheWriter
	logger      logger.Interface
	flushTicker *time.Ticker
	done        chan struct{}
	wg          sync.WaitGroup
}

// NewSubscriptionTrafficBuffer creates a new SubscriptionTrafficBuffer instance.
func NewSubscriptionTrafficBuffer(cache SubscriptionTrafficCacheWriter, log logger.Interface) *SubscriptionTrafficBuffer {
	b := &SubscriptionTrafficBuffer{
		cache:       cache,
		logger:      log,
		flushTicker: time.NewTicker(SubscriptionTrafficFlushInterval),
		done:        make(chan struct{}),
	}

	// Initialize all shards
	for i := 0; i < SubscriptionTrafficNumShards; i++ {
		b.shards[i] = &subscriptionBufferShard{
			entries: make(map[subscriptionTrafficKey]*SubscriptionTrafficEntry),
		}
	}

	return b
}

// getShard returns the shard for a given subscriptionID using modulo sharding.
func (b *SubscriptionTrafficBuffer) getShard(subscriptionID uint) *subscriptionBufferShard {
	return b.shards[subscriptionID%SubscriptionTrafficNumShards]
}

// AddTraffic adds traffic data to the buffer (thread-safe).
func (b *SubscriptionTrafficBuffer) AddTraffic(nodeID, subscriptionID uint, upload, download int64) {
	// Skip zero traffic entries
	if upload == 0 && download == 0 {
		return
	}

	key := subscriptionTrafficKey{
		NodeID:         nodeID,
		SubscriptionID: subscriptionID,
	}

	shard := b.getShard(subscriptionID)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if existing, ok := shard.entries[key]; ok {
		existing.Upload += upload
		existing.Download += download
	} else {
		shard.entries[key] = &SubscriptionTrafficEntry{
			NodeID:         nodeID,
			SubscriptionID: subscriptionID,
			Upload:         upload,
			Download:       download,
		}
	}
}

// Start starts the background flush goroutine.
func (b *SubscriptionTrafficBuffer) Start() {
	b.wg.Add(1)
	goroutine.SafeGo(b.logger, "subscription-traffic-buffer-flush-loop", b.flushLoop)
	b.logger.Infow("subscription traffic buffer started",
		"shards", SubscriptionTrafficNumShards,
		"flush_interval", SubscriptionTrafficFlushInterval.String(),
	)
}

// Stop stops the buffer and flushes remaining data.
func (b *SubscriptionTrafficBuffer) Stop() {
	close(b.done)
	b.wg.Wait()
	b.flushTicker.Stop()

	// Final flush to ensure no data is lost
	b.flush()

	b.logger.Infow("subscription traffic buffer stopped")
}

// flushLoop is the background loop that periodically flushes traffic data.
func (b *SubscriptionTrafficBuffer) flushLoop() {
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

// subscriptionTrafficBatchSize is the maximum number of entries per batch pipeline call.
const subscriptionTrafficBatchSize = 500

// flush flushes all accumulated traffic data to Redis using batch pipeline.
func (b *SubscriptionTrafficBuffer) flush() {
	ctx := context.Background()
	droppedCount := 0

	// Phase 1: Collect all entries from all shards
	var allEntries []*SubscriptionTrafficEntry
	for i := 0; i < SubscriptionTrafficNumShards; i++ {
		shard := b.shards[i]

		// Fast swap to minimize lock hold time
		shard.mu.Lock()
		entries := shard.entries
		shard.entries = make(map[subscriptionTrafficKey]*SubscriptionTrafficEntry)
		shard.mu.Unlock()

		for _, entry := range entries {
			if entry.Upload > 0 || entry.Download > 0 {
				allEntries = append(allEntries, entry)
			}
		}
	}

	if len(allEntries) == 0 {
		return
	}

	// Phase 2: Batch write in chunks to avoid oversized pipelines
	flushedCount := 0
	failedCount := 0

	for start := 0; start < len(allEntries); start += subscriptionTrafficBatchSize {
		end := start + subscriptionTrafficBatchSize
		if end > len(allEntries) {
			end = len(allEntries)
		}
		batch := allEntries[start:end]

		// Convert to cache batch entry slice
		batchValues := make([]cache.SubscriptionTrafficBatchEntry, len(batch))
		for i, e := range batch {
			batchValues[i] = cache.SubscriptionTrafficBatchEntry{
				NodeID:         e.NodeID,
				SubscriptionID: e.SubscriptionID,
				Upload:         e.Upload,
				Download:       e.Download,
			}
		}

		if err := b.cache.BatchIncrementSubscriptionTraffic(ctx, batchValues); err != nil {
			// Entire batch failed — re-add or drop each entry based on retry count
			for _, entry := range batch {
				entry.RetryCount++
				if entry.RetryCount >= SubscriptionTrafficMaxRetryCount {
					b.logger.Errorw("subscription traffic data dropped after max retries",
						"node_id", entry.NodeID,
						"subscription_id", entry.SubscriptionID,
						"upload", entry.Upload,
						"download", entry.Download,
						"retry_count", entry.RetryCount,
						"error", err,
					)
					droppedCount++
					continue
				}
				b.reAddEntry(entry)
				failedCount++
			}
			b.logger.Warnw("failed to batch flush subscription traffic to redis, will retry",
				"batch_size", len(batch),
				"error", err,
			)
			continue
		}
		flushedCount += len(batch)
	}

	if flushedCount > 0 || failedCount > 0 || droppedCount > 0 {
		b.logger.Debugw("subscription traffic buffer flushed to redis",
			"flushed_count", flushedCount,
			"failed_count", failedCount,
			"dropped_count", droppedCount,
		)
	}
}

// reAddEntry re-adds a failed entry back to its shard for retry.
func (b *SubscriptionTrafficBuffer) reAddEntry(entry *SubscriptionTrafficEntry) {
	key := subscriptionTrafficKey{
		NodeID:         entry.NodeID,
		SubscriptionID: entry.SubscriptionID,
	}

	shard := b.getShard(entry.SubscriptionID)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if existing, ok := shard.entries[key]; ok {
		// Merge with any new traffic that arrived during flush
		existing.Upload += entry.Upload
		existing.Download += entry.Download
		// Keep the higher retry count to track total failures
		if entry.RetryCount > existing.RetryCount {
			existing.RetryCount = entry.RetryCount
		}
	} else {
		shard.entries[key] = &SubscriptionTrafficEntry{
			NodeID:         entry.NodeID,
			SubscriptionID: entry.SubscriptionID,
			Upload:         entry.Upload,
			Download:       entry.Download,
			RetryCount:     entry.RetryCount,
		}
	}
}
