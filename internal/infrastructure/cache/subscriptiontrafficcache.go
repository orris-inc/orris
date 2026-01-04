package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// Redis key patterns for subscription traffic
	// Key format: sub_traffic:{nodeID}:{subscriptionID}
	subscriptionTrafficKeyPrefix = "sub_traffic:"
	subscriptionTrafficTTL       = 24 * time.Hour

	// Active subscriptions set key - tracks which node:subscription pairs have pending traffic
	activeSubscriptionsSetKey = "sub_traffic:active"

	// Hash field names for subscription traffic
	subFieldUpload              = "upload"
	subFieldDownload            = "download"
	subFieldLastFlushedUpload   = "last_flushed_upload"
	subFieldLastFlushedDownload = "last_flushed_download"
)

// subSafeRemoveFromActiveSetScript atomically removes a key from active set
// only if current values equal last_flushed values (no new data since last check).
// KEYS[1] = traffic hash key, KEYS[2] = active set key
// ARGV[1] = member to remove from set (the key itself)
// Returns 1 if removed, 0 if skipped (new data exists)
var subSafeRemoveFromActiveSetScript = redis.NewScript(`
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

// subAtomicUpdateLastFlushedScript updates last_flushed values and removes from active set
// only if current values match the expected values (no new data since we read).
// KEYS[1] = traffic hash key, KEYS[2] = active set key
// ARGV[1] = expected current upload, ARGV[2] = expected current download
// ARGV[3] = member to remove (the key itself), ARGV[4] = TTL in seconds
// Returns 1 if removed from active set, 0 if kept (new data arrived)
var subAtomicUpdateLastFlushedScript = redis.NewScript(`
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

// SubscriptionTrafficData represents traffic statistics for a subscription.
type SubscriptionTrafficData struct {
	Upload   int64
	Download int64
}

// SubscriptionTrafficCache defines the interface for subscription traffic caching.
type SubscriptionTrafficCache interface {
	// IncrementSubscriptionTraffic atomically increments subscription traffic in Redis.
	IncrementSubscriptionTraffic(ctx context.Context, nodeID, subscriptionID uint, upload, download int64) error

	// GetSubscriptionTraffic returns the real-time traffic for a node:subscription pair.
	GetSubscriptionTraffic(ctx context.Context, nodeID, subscriptionID uint) (upload, download int64, exists bool)

	// FlushToDatabase flushes all pending traffic to MySQL subscription_usages table.
	FlushToDatabase(ctx context.Context) error
}

// RedisSubscriptionTrafficCache implements SubscriptionTrafficCache using Redis.
type RedisSubscriptionTrafficCache struct {
	client                *redis.Client
	subscriptionUsageRepo subscription.SubscriptionUsageRepository
	logger                logger.Interface
}

// NewRedisSubscriptionTrafficCache creates a new RedisSubscriptionTrafficCache instance.
func NewRedisSubscriptionTrafficCache(
	client *redis.Client,
	subscriptionUsageRepo subscription.SubscriptionUsageRepository,
	logger logger.Interface,
) SubscriptionTrafficCache {
	return &RedisSubscriptionTrafficCache{
		client:                client,
		subscriptionUsageRepo: subscriptionUsageRepo,
		logger:                logger,
	}
}

// subscriptionTrafficKey generates the Redis key for a node:subscription traffic.
// Format: sub_traffic:{nodeID}:{subscriptionID}
func subscriptionTrafficKey(nodeID, subscriptionID uint) string {
	return fmt.Sprintf("%s%d:%d", subscriptionTrafficKeyPrefix, nodeID, subscriptionID)
}

// parseSubscriptionTrafficKey extracts nodeID and subscriptionID from Redis key.
// Example: "sub_traffic:123:456" -> nodeID=123, subscriptionID=456
func parseSubscriptionTrafficKey(key string) (nodeID, subscriptionID uint, err error) {
	var nid, sid uint64
	_, err = fmt.Sscanf(key, subscriptionTrafficKeyPrefix+"%d:%d", &nid, &sid)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid key format: %s", key)
	}
	return uint(nid), uint(sid), nil
}

// IncrementSubscriptionTraffic atomically increments subscription traffic in Redis.
func (c *RedisSubscriptionTrafficCache) IncrementSubscriptionTraffic(ctx context.Context, nodeID, subscriptionID uint, upload, download int64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	key := subscriptionTrafficKey(nodeID, subscriptionID)
	pipe := c.client.Pipeline()

	if upload > 0 {
		pipe.HIncrBy(ctx, key, subFieldUpload, upload)
	}
	if download > 0 {
		pipe.HIncrBy(ctx, key, subFieldDownload, download)
	}

	// Set expiration to prevent memory leak
	pipe.Expire(ctx, key, subscriptionTrafficTTL)

	// Add to active set for efficient flush lookup
	pipe.SAdd(ctx, activeSubscriptionsSetKey, key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Errorw("failed to increment subscription traffic in redis",
			"node_id", nodeID,
			"subscription_id", subscriptionID,
			"upload", upload,
			"download", download,
			"error", err,
		)
		return fmt.Errorf("failed to increment subscription traffic: %w", err)
	}

	c.logger.Debugw("subscription traffic incremented in redis",
		"node_id", nodeID,
		"subscription_id", subscriptionID,
		"upload", upload,
		"download", download,
	)

	return nil
}

// GetSubscriptionTraffic returns the real-time traffic for a node:subscription pair.
func (c *RedisSubscriptionTrafficCache) GetSubscriptionTraffic(ctx context.Context, nodeID, subscriptionID uint) (upload, download int64, exists bool) {
	key := subscriptionTrafficKey(nodeID, subscriptionID)

	values, err := c.client.HGetAll(ctx, key).Result()
	if err != nil && err != redis.Nil {
		c.logger.Warnw("failed to get subscription traffic from redis",
			"node_id", nodeID,
			"subscription_id", subscriptionID,
			"error", err,
		)
		return 0, 0, false
	}

	if len(values) == 0 {
		return 0, 0, false
	}

	upload, _ = strconv.ParseInt(values[subFieldUpload], 10, 64)
	download, _ = strconv.ParseInt(values[subFieldDownload], 10, 64)

	return upload, download, true
}

// FlushToDatabase synchronizes all Redis traffic to MySQL subscription_usages table.
func (c *RedisSubscriptionTrafficCache) FlushToDatabase(ctx context.Context) error {
	c.logger.Infow("starting subscription traffic flush to database")

	flushedCount := 0
	errorCount := 0
	skippedCount := 0

	// Get all active keys from the set
	keys, err := c.client.SMembers(ctx, activeSubscriptionsSetKey).Result()
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to get active subscriptions set", "error", err)
		return fmt.Errorf("failed to get active subscriptions: %w", err)
	}

	if len(keys) == 0 {
		c.logger.Infow("subscription traffic flush completed, no active entries")
		return nil
	}

	// Use current hour as period for aggregation
	period := biztime.TruncateToHourInBiz(biztime.NowUTC())

	for _, key := range keys {
		nodeID, subscriptionID, err := parseSubscriptionTrafficKey(key)
		if err != nil {
			c.logger.Warnw("failed to parse subscription traffic key", "key", key, "error", err)
			// Remove invalid entry from active set
			c.client.SRem(ctx, activeSubscriptionsSetKey, key)
			continue
		}

		// Get all traffic values from Redis
		values, err := c.client.HGetAll(ctx, key).Result()
		if err != nil {
			c.logger.Errorw("failed to get subscription traffic from redis", "key", key, "error", err)
			errorCount++
			continue
		}

		if len(values) == 0 {
			// Key expired or doesn't exist, remove from active set
			c.client.SRem(ctx, activeSubscriptionsSetKey, key)
			continue
		}

		// Parse current values
		currentUpload, _ := strconv.ParseInt(values[subFieldUpload], 10, 64)
		currentDownload, _ := strconv.ParseInt(values[subFieldDownload], 10, 64)
		lastFlushedUpload, _ := strconv.ParseInt(values[subFieldLastFlushedUpload], 10, 64)
		lastFlushedDownload, _ := strconv.ParseInt(values[subFieldLastFlushedDownload], 10, 64)

		// Calculate increments
		uploadDelta := currentUpload - lastFlushedUpload
		downloadDelta := currentDownload - lastFlushedDownload

		if uploadDelta <= 0 && downloadDelta <= 0 {
			skippedCount++
			// Use Lua script to atomically check and remove from active set
			// This prevents race condition where new data arrives between check and remove
			c.safeRemoveFromActiveSet(ctx, key)
			continue
		}

		// Ensure non-negative deltas
		if uploadDelta < 0 {
			uploadDelta = 0
		}
		if downloadDelta < 0 {
			downloadDelta = 0
		}

		// Create subscription usage entity and record to MySQL
		usage, err := subscription.NewSubscriptionUsage(
			subscription.ResourceTypeNode.String(),
			nodeID,
			&subscriptionID,
			period,
		)
		if err != nil {
			c.logger.Errorw("failed to create subscription usage entity",
				"node_id", nodeID,
				"subscription_id", subscriptionID,
				"error", err,
			)
			errorCount++
			continue
		}

		if err := usage.Accumulate(uint64(uploadDelta), uint64(downloadDelta)); err != nil {
			c.logger.Errorw("failed to accumulate usage",
				"node_id", nodeID,
				"subscription_id", subscriptionID,
				"error", err,
			)
			errorCount++
			continue
		}

		if err := c.subscriptionUsageRepo.RecordUsage(ctx, usage); err != nil {
			c.logger.Errorw("failed to flush subscription traffic to database",
				"node_id", nodeID,
				"subscription_id", subscriptionID,
				"upload_delta", uploadDelta,
				"download_delta", downloadDelta,
				"error", err,
			)
			errorCount++
			continue
		}

		// Atomically update last_flushed values and remove from active set if no new data
		// Use Lua script to prevent data loss from race condition
		removed, err := c.atomicUpdateLastFlushed(ctx, key, currentUpload, currentDownload)
		if err != nil {
			c.logger.Warnw("failed to update last_flushed values in redis",
				"key", key,
				"error", err,
			)
			// Don't remove from active set - will retry next flush
		}

		flushedCount++

		c.logger.Debugw("flushed subscription traffic to database",
			"node_id", nodeID,
			"subscription_id", subscriptionID,
			"upload_delta", uploadDelta,
			"download_delta", downloadDelta,
			"removed_from_active", removed,
		)
	}

	c.logger.Infow("subscription traffic flush completed",
		"flushed_count", flushedCount,
		"skipped_count", skippedCount,
		"error_count", errorCount,
	)

	return nil
}

// safeRemoveFromActiveSet atomically removes a key from active set only if no new data exists.
func (c *RedisSubscriptionTrafficCache) safeRemoveFromActiveSet(ctx context.Context, key string) {
	_, err := subSafeRemoveFromActiveSetScript.Run(ctx, c.client,
		[]string{key, activeSubscriptionsSetKey},
		key,
	).Result()
	if err != nil && err != redis.Nil {
		c.logger.Warnw("failed to safely remove from active set",
			"key", key,
			"error", err,
		)
	}
}

// atomicUpdateLastFlushed updates last_flushed values and conditionally removes from active set.
func (c *RedisSubscriptionTrafficCache) atomicUpdateLastFlushed(ctx context.Context, key string, currentUpload, currentDownload int64) (bool, error) {
	result, err := subAtomicUpdateLastFlushedScript.Run(ctx, c.client,
		[]string{key, activeSubscriptionsSetKey},
		currentUpload, currentDownload, key, int(subscriptionTrafficTTL.Seconds()),
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
