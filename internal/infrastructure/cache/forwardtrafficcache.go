package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	forwardTrafficKeyPrefix = "forward:traffic:"
	forwardTrafficTTL       = 24 * time.Hour

	// Active rules set key - tracks which rules have pending traffic updates
	activeRulesSetKey = "forward:traffic:active_rules"

	// Hash field names
	fieldUpload              = "upload"
	fieldDownload            = "download"
	fieldLastFlushedUpload   = "last_flushed_upload"
	fieldLastFlushedDownload = "last_flushed_download"
)

// safeRemoveFromActiveSetScript atomically removes a key from active set
// only if current values equal last_flushed values (no new data since last check).
// KEYS[1] = traffic hash key, KEYS[2] = active set key
// ARGV[1] = member to remove from set
// Returns 1 if removed, 0 if skipped (new data exists)
var safeRemoveFromActiveSetScript = redis.NewScript(`
local current_upload = redis.call('HGET', KEYS[1], 'upload') or '0'
local current_download = redis.call('HGET', KEYS[1], 'download') or '0'
local last_flushed_upload = redis.call('HGET', KEYS[1], 'last_flushed_upload') or '0'
local last_flushed_download = redis.call('HGET', KEYS[1], 'last_flushed_download') or '0'

if current_upload == last_flushed_upload and current_download == last_flushed_download then
    redis.call('SREM', KEYS[2], ARGV[1])
    return 1
end
return 0
`)

// TrafficData represents traffic statistics for a forward rule.
type TrafficData struct {
	Upload   int64
	Download int64
}

// ForwardTrafficCache defines the interface for forward rule traffic caching.
type ForwardTrafficCache interface {
	// IncrementRuleTraffic atomically increments rule traffic in Redis.
	IncrementRuleTraffic(ctx context.Context, ruleID uint, upload, download int64) error

	// BatchIncrementRuleTraffic atomically increments traffic for multiple rules in a single Redis pipeline.
	BatchIncrementRuleTraffic(ctx context.Context, entries []RuleTrafficBatchEntry) error

	// GetRuleTraffic returns the real-time traffic (accumulated value in Redis).
	// If the key does not exist in Redis, returns (0, 0, false).
	GetRuleTraffic(ctx context.Context, ruleID uint) (upload, download int64, exists bool)

	// BatchGetRuleTraffic returns traffic data for multiple rules in batch.
	BatchGetRuleTraffic(ctx context.Context, ruleIDs []uint) (map[uint]TrafficData, error)

	// FlushToDatabase flushes all pending traffic to MySQL in batch.
	// Calculates increments, writes to forward_rules table.
	FlushToDatabase(ctx context.Context) error

	// InitRuleTraffic initializes rule traffic (loads base value from MySQL).
	// Uses HSETNX to prevent overwriting concurrent increments.
	InitRuleTraffic(ctx context.Context, ruleID uint, upload, download int64) error

	// CleanupRuleCache removes traffic cache for a rule.
	// Should be called when a rule is deleted.
	CleanupRuleCache(ctx context.Context, ruleID uint) error
}

// RuleTrafficBatchEntry represents a single entry for batch traffic increment.
type RuleTrafficBatchEntry struct {
	RuleID   uint
	Upload   int64
	Download int64
}

// RedisForwardTrafficCache implements ForwardTrafficCache using Redis.
type RedisForwardTrafficCache struct {
	client   *redis.Client
	ruleRepo forward.Repository
	logger   logger.Interface
}

// NewRedisForwardTrafficCache creates a new RedisForwardTrafficCache instance.
func NewRedisForwardTrafficCache(
	client *redis.Client,
	ruleRepo forward.Repository,
	logger logger.Interface,
) ForwardTrafficCache {
	return &RedisForwardTrafficCache{
		client:   client,
		ruleRepo: ruleRepo,
		logger:   logger,
	}
}

// ruleTrafficKey generates the Redis key for a forward rule's traffic.
func ruleTrafficKey(ruleID uint) string {
	return fmt.Sprintf("%s%d", forwardTrafficKeyPrefix, ruleID)
}

// IncrementRuleTraffic atomically increments rule traffic in Redis.
func (c *RedisForwardTrafficCache) IncrementRuleTraffic(ctx context.Context, ruleID uint, upload, download int64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	key := ruleTrafficKey(ruleID)
	ruleIDStr := strconv.FormatUint(uint64(ruleID), 10)
	pipe := c.client.Pipeline()

	if upload > 0 {
		pipe.HIncrBy(ctx, key, fieldUpload, upload)
	}
	if download > 0 {
		pipe.HIncrBy(ctx, key, fieldDownload, download)
	}

	// Set expiration to prevent memory leak
	pipe.Expire(ctx, key, forwardTrafficTTL)

	// Add rule ID to active rules set for efficient flush lookup
	pipe.SAdd(ctx, activeRulesSetKey, ruleIDStr)

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Errorw("failed to increment forward rule traffic in redis",
			"rule_id", ruleID,
			"upload", upload,
			"download", download,
			"error", err,
		)
		return fmt.Errorf("failed to increment rule traffic: %w", err)
	}

	c.logger.Debugw("forward rule traffic incremented in redis",
		"rule_id", ruleID,
		"upload", upload,
		"download", download,
	)

	return nil
}

// BatchIncrementRuleTraffic atomically increments traffic for multiple rules in a single Redis pipeline.
// All entries are written in one pipeline round-trip to minimize Redis network overhead.
func (c *RedisForwardTrafficCache) BatchIncrementRuleTraffic(ctx context.Context, entries []RuleTrafficBatchEntry) error {
	if len(entries) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	for _, entry := range entries {
		if entry.Upload == 0 && entry.Download == 0 {
			continue
		}

		key := ruleTrafficKey(entry.RuleID)
		ruleIDStr := strconv.FormatUint(uint64(entry.RuleID), 10)

		if entry.Upload > 0 {
			pipe.HIncrBy(ctx, key, fieldUpload, entry.Upload)
		}
		if entry.Download > 0 {
			pipe.HIncrBy(ctx, key, fieldDownload, entry.Download)
		}

		// Set expiration to prevent memory leak
		pipe.Expire(ctx, key, forwardTrafficTTL)

		// Add rule ID to active rules set for efficient flush lookup
		pipe.SAdd(ctx, activeRulesSetKey, ruleIDStr)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Errorw("failed to batch increment forward rule traffic in redis",
			"entry_count", len(entries),
			"error", err,
		)
		return fmt.Errorf("failed to batch increment rule traffic: %w", err)
	}

	c.logger.Debugw("forward rule traffic batch incremented in redis",
		"entry_count", len(entries),
	)

	return nil
}

// GetRuleTraffic returns the real-time traffic for a rule from Redis.
func (c *RedisForwardTrafficCache) GetRuleTraffic(ctx context.Context, ruleID uint) (upload, download int64, exists bool) {
	key := ruleTrafficKey(ruleID)

	values, err := c.client.HGetAll(ctx, key).Result()
	if err != nil && err != redis.Nil {
		c.logger.Warnw("failed to get forward rule traffic from redis",
			"rule_id", ruleID,
			"error", err,
		)
		return 0, 0, false
	}

	if len(values) == 0 {
		return 0, 0, false
	}

	upload, _ = strconv.ParseInt(values[fieldUpload], 10, 64)
	download, _ = strconv.ParseInt(values[fieldDownload], 10, 64)

	c.logger.Debugw("got forward rule traffic from redis",
		"rule_id", ruleID,
		"upload", upload,
		"download", download,
	)

	return upload, download, true
}

// BatchGetRuleTraffic returns traffic data for multiple rules using pipeline.
func (c *RedisForwardTrafficCache) BatchGetRuleTraffic(ctx context.Context, ruleIDs []uint) (map[uint]TrafficData, error) {
	if len(ruleIDs) == 0 {
		return make(map[uint]TrafficData), nil
	}

	pipe := c.client.Pipeline()
	cmds := make(map[uint]*redis.MapStringStringCmd)

	for _, ruleID := range ruleIDs {
		key := ruleTrafficKey(ruleID)
		cmds[ruleID] = pipe.HGetAll(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to batch get forward rule traffic from redis",
			"rule_count", len(ruleIDs),
			"error", err,
		)
		return nil, fmt.Errorf("failed to batch get rule traffic: %w", err)
	}

	result := make(map[uint]TrafficData, len(ruleIDs))
	for ruleID, cmd := range cmds {
		values, err := cmd.Result()
		if err != nil && err != redis.Nil {
			c.logger.Warnw("failed to get forward rule traffic from pipeline result",
				"rule_id", ruleID,
				"error", err,
			)
			continue
		}

		if len(values) == 0 {
			continue
		}

		upload, _ := strconv.ParseInt(values[fieldUpload], 10, 64)
		download, _ := strconv.ParseInt(values[fieldDownload], 10, 64)

		result[ruleID] = TrafficData{
			Upload:   upload,
			Download: download,
		}
	}

	c.logger.Debugw("batch got forward rule traffic from redis",
		"requested", len(ruleIDs),
		"found", len(result),
	)

	return result, nil
}

// FlushToDatabase synchronizes all Redis traffic to MySQL.
func (c *RedisForwardTrafficCache) FlushToDatabase(ctx context.Context) error {
	c.logger.Infow("starting forward traffic flush to database")

	flushedCount := 0
	errorCount := 0
	skippedCount := 0

	// Get all active rule IDs from the set (O(N) where N is set size, much faster than SCAN)
	ruleIDStrs, err := c.client.SMembers(ctx, activeRulesSetKey).Result()
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to get active rules set", "error", err)
		return fmt.Errorf("failed to get active rules: %w", err)
	}

	if len(ruleIDStrs) == 0 {
		c.logger.Infow("forward traffic flush completed, no active rules")
		return nil
	}

	for _, ruleIDStr := range ruleIDStrs {
		ruleID64, err := strconv.ParseUint(ruleIDStr, 10, 64)
		if err != nil {
			c.logger.Warnw("failed to parse rule id from set", "rule_id_str", ruleIDStr, "error", err)
			// Remove invalid entry from active set
			c.client.SRem(ctx, activeRulesSetKey, ruleIDStr)
			continue
		}
		ruleID := uint(ruleID64)
		key := ruleTrafficKey(ruleID)

		// Get all traffic values from Redis
		values, err := c.client.HGetAll(ctx, key).Result()
		if err != nil {
			c.logger.Errorw("failed to get forward rule traffic from redis", "rule_id", ruleID, "error", err)
			errorCount++
			continue
		}

		if len(values) == 0 {
			// Key expired or doesn't exist, remove from active set
			c.client.SRem(ctx, activeRulesSetKey, ruleIDStr)
			continue
		}

		// Parse current values
		currentUpload, _ := strconv.ParseInt(values[fieldUpload], 10, 64)
		currentDownload, _ := strconv.ParseInt(values[fieldDownload], 10, 64)
		lastFlushedUpload, _ := strconv.ParseInt(values[fieldLastFlushedUpload], 10, 64)
		lastFlushedDownload, _ := strconv.ParseInt(values[fieldLastFlushedDownload], 10, 64)

		// Calculate increments
		uploadDelta := currentUpload - lastFlushedUpload
		downloadDelta := currentDownload - lastFlushedDownload

		if uploadDelta <= 0 && downloadDelta <= 0 {
			skippedCount++
			// Use Lua script to atomically check and remove from active set
			// This prevents race condition where new data arrives between check and remove
			c.safeRemoveFromActiveSet(ctx, key, ruleIDStr)
			continue
		}

		// Ensure non-negative deltas
		if uploadDelta < 0 {
			uploadDelta = 0
		}
		if downloadDelta < 0 {
			downloadDelta = 0
		}

		// Update MySQL
		err = c.ruleRepo.UpdateTraffic(ctx, ruleID, uploadDelta, downloadDelta)
		if err != nil {
			// If rule no longer exists, clean up Redis cache to prevent repeated errors
			if errors.IsNotFoundError(err) {
				c.logger.Warnw("forward rule no longer exists, cleaning up cache",
					"rule_id", ruleID,
					"upload_delta", uploadDelta,
					"download_delta", downloadDelta,
				)
				c.cleanupDeletedRuleCache(ctx, key, ruleIDStr)
				skippedCount++
				continue
			}
			c.logger.Errorw("failed to flush forward rule traffic to database",
				"rule_id", ruleID,
				"upload_delta", uploadDelta,
				"download_delta", downloadDelta,
				"error", err,
			)
			errorCount++
			continue
		}

		// Atomically update last_flushed values and remove from active set if no new data
		// Use Lua script to prevent data loss from race condition
		removed, err := c.atomicUpdateLastFlushed(ctx, key, ruleIDStr, currentUpload, currentDownload)
		if err != nil {
			c.logger.Warnw("failed to update last_flushed values in redis",
				"rule_id", ruleID,
				"error", err,
			)
			// Don't remove from active set - will retry next flush
		}

		flushedCount++

		c.logger.Debugw("flushed forward rule traffic to database",
			"rule_id", ruleID,
			"upload_delta", uploadDelta,
			"download_delta", downloadDelta,
			"removed_from_active", removed,
		)
	}

	c.logger.Infow("forward traffic flush completed",
		"flushed_count", flushedCount,
		"skipped_count", skippedCount,
		"error_count", errorCount,
	)

	return nil
}

// safeRemoveFromActiveSet atomically removes a rule from active set only if no new data exists.
func (c *RedisForwardTrafficCache) safeRemoveFromActiveSet(ctx context.Context, hashKey, ruleIDStr string) {
	_, err := safeRemoveFromActiveSetScript.Run(ctx, c.client,
		[]string{hashKey, activeRulesSetKey},
		ruleIDStr,
	).Result()
	if err != nil && err != redis.Nil {
		c.logger.Warnw("failed to safely remove from active set",
			"hash_key", hashKey,
			"error", err,
		)
	}
}

// CleanupRuleCache removes traffic cache for a rule.
// Should be called when a rule is deleted.
func (c *RedisForwardTrafficCache) CleanupRuleCache(ctx context.Context, ruleID uint) error {
	key := ruleTrafficKey(ruleID)
	ruleIDStr := strconv.FormatUint(uint64(ruleID), 10)

	pipe := c.client.Pipeline()
	pipe.SRem(ctx, activeRulesSetKey, ruleIDStr)
	pipe.Del(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Warnw("failed to cleanup rule cache",
			"rule_id", ruleID,
			"error", err,
		)
		return fmt.Errorf("failed to cleanup rule cache: %w", err)
	}

	c.logger.Debugw("cleaned up rule traffic cache", "rule_id", ruleID)
	return nil
}

// cleanupDeletedRuleCache removes traffic cache for a deleted rule.
// This is called internally when we detect a rule no longer exists in MySQL during flush.
func (c *RedisForwardTrafficCache) cleanupDeletedRuleCache(ctx context.Context, hashKey, ruleIDStr string) {
	pipe := c.client.Pipeline()
	pipe.SRem(ctx, activeRulesSetKey, ruleIDStr)
	pipe.Del(ctx, hashKey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Warnw("failed to cleanup deleted rule cache",
			"hash_key", hashKey,
			"error", err,
		)
	}
}

// atomicUpdateLastFlushedScript updates last_flushed values and removes from active set
// only if current values match the expected values (no new data since we read).
// KEYS[1] = traffic hash key, KEYS[2] = active set key
// ARGV[1] = expected current upload, ARGV[2] = expected current download
// ARGV[3] = member to remove, ARGV[4] = TTL in seconds
// Returns 1 if removed from active set, 0 if kept (new data arrived)
var atomicUpdateLastFlushedScript = redis.NewScript(`
local current_upload = redis.call('HGET', KEYS[1], 'upload') or '0'
local current_download = redis.call('HGET', KEYS[1], 'download') or '0'
local expected_upload = ARGV[1]
local expected_download = ARGV[2]

-- Always update last_flushed to the values we successfully wrote to MySQL
redis.call('HSET', KEYS[1], 'last_flushed_upload', expected_upload)
redis.call('HSET', KEYS[1], 'last_flushed_download', expected_download)
redis.call('EXPIRE', KEYS[1], ARGV[4])

-- Only remove from active set if no new data has arrived
if current_upload == expected_upload and current_download == expected_download then
    redis.call('SREM', KEYS[2], ARGV[3])
    return 1
end
return 0
`)

// atomicUpdateLastFlushed updates last_flushed values and conditionally removes from active set.
func (c *RedisForwardTrafficCache) atomicUpdateLastFlushed(ctx context.Context, hashKey, ruleIDStr string, currentUpload, currentDownload int64) (bool, error) {
	result, err := atomicUpdateLastFlushedScript.Run(ctx, c.client,
		[]string{hashKey, activeRulesSetKey},
		currentUpload, currentDownload, ruleIDStr, int(forwardTrafficTTL.Seconds()),
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// InitRuleTraffic initializes rule traffic in Redis from MySQL base values.
// Uses HSETNX to prevent overwriting concurrent increments.
func (c *RedisForwardTrafficCache) InitRuleTraffic(ctx context.Context, ruleID uint, upload, download int64) error {
	key := ruleTrafficKey(ruleID)

	pipe := c.client.Pipeline()

	// Use HSETNX to only set if field doesn't exist
	pipe.HSetNX(ctx, key, fieldUpload, upload)
	pipe.HSetNX(ctx, key, fieldDownload, download)
	pipe.HSetNX(ctx, key, fieldLastFlushedUpload, upload)
	pipe.HSetNX(ctx, key, fieldLastFlushedDownload, download)

	// Set expiration
	pipe.Expire(ctx, key, forwardTrafficTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Errorw("failed to initialize forward rule traffic in redis",
			"rule_id", ruleID,
			"upload", upload,
			"download", download,
			"error", err,
		)
		return fmt.Errorf("failed to init rule traffic: %w", err)
	}

	c.logger.Debugw("forward rule traffic initialized in redis",
		"rule_id", ruleID,
		"upload", upload,
		"download", download,
	)

	return nil
}
