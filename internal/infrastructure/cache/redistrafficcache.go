package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type RedisTrafficCache struct {
	client   *redis.Client
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewRedisTrafficCache(
	client *redis.Client,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) TrafficCache {
	return &RedisTrafficCache{
		client:   client,
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

// IncrementTraffic atomically increments traffic in Redis
func (c *RedisTrafficCache) IncrementTraffic(ctx context.Context, nodeID uint, upload, download uint64) error {
	key := fmt.Sprintf("node:%d:traffic", nodeID)

	pipe := c.client.Pipeline()

	// Atomic increment with safe conversion to prevent overflow
	if upload > 0 {
		pipe.HIncrBy(ctx, key, "upload", utils.SafeUint64ToInt64(upload))
	}
	if download > 0 {
		pipe.HIncrBy(ctx, key, "download", utils.SafeUint64ToInt64(download))
	}

	// Set expiration to prevent memory leak (24 hours)
	pipe.Expire(ctx, key, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Errorw("failed to increment traffic in redis",
			"node_id", nodeID,
			"upload", upload,
			"download", download,
			"error", err,
		)
		return fmt.Errorf("failed to increment traffic: %w", err)
	}

	return nil
}

// GetNodeTraffic returns total traffic from Redis only
// Note: Base traffic has been removed from Node entity, all traffic is now tracked via Redis cache
func (c *RedisTrafficCache) GetNodeTraffic(ctx context.Context, nodeID uint) (uint64, error) {
	// Get traffic from Redis cache
	key := fmt.Sprintf("node:%d:traffic", nodeID)
	values, err := c.client.HGetAll(ctx, key).Result()
	if err != nil && err != redis.Nil {
		c.logger.Warnw("failed to get traffic from redis",
			"node_id", nodeID,
			"error", err,
		)
		return 0, fmt.Errorf("failed to get traffic from redis: %w", err)
	}

	// Calculate total traffic (upload + download)
	upload, _ := strconv.ParseUint(values["upload"], 10, 64)
	download, _ := strconv.ParseUint(values["download"], 10, 64)

	totalTraffic := upload + download

	return totalTraffic, nil
}

// FlushToDatabase synchronizes all Redis traffic to MySQL
func (c *RedisTrafficCache) FlushToDatabase(ctx context.Context) error {
	c.logger.Debugw("starting traffic flush to database")

	flushedCount := 0
	errorCount := 0

	// Scan all traffic keys
	iter := c.client.Scan(ctx, 0, "node:*:traffic", 100).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()

		// Parse node ID from key
		nodeID, err := parseNodeIDFromKey(key)
		if err != nil {
			c.logger.Warnw("failed to parse node id from key", "key", key, "error", err)
			continue
		}

		// Get traffic from Redis
		values, err := c.client.HGetAll(ctx, key).Result()
		if err != nil {
			c.logger.Errorw("failed to get traffic from redis", "key", key, "error", err)
			errorCount++
			continue
		}

		upload, _ := strconv.ParseUint(values["upload"], 10, 64)
		download, _ := strconv.ParseUint(values["download"], 10, 64)
		totalAmount := upload + download

		if totalAmount == 0 {
			// No traffic to flush, delete the key
			c.client.Del(ctx, key)
			continue
		}

		// Flush to MySQL using atomic increment
		err = c.nodeRepo.IncrementTraffic(ctx, nodeID, totalAmount)
		if err != nil {
			c.logger.Errorw("failed to flush traffic to database",
				"node_id", nodeID,
				"amount", totalAmount,
				"error", err,
			)
			errorCount++
			continue
		}

		// Delete Redis key after successful flush
		err = c.client.Del(ctx, key).Err()
		if err != nil {
			c.logger.Warnw("failed to delete redis key after flush",
				"key", key,
				"error", err,
			)
		}

		flushedCount++
		c.logger.Debugw("flushed traffic to database",
			"node_id", nodeID,
			"amount", totalAmount,
		)
	}

	if err := iter.Err(); err != nil {
		c.logger.Errorw("error during redis scan", "error", err)
		return fmt.Errorf("scan error: %w", err)
	}

	c.logger.Infow("traffic flush completed",
		"flushed_count", flushedCount,
		"error_count", errorCount,
	)

	return nil
}

// GetAllPendingNodeIDs returns all node IDs with pending traffic
func (c *RedisTrafficCache) GetAllPendingNodeIDs(ctx context.Context) ([]uint, error) {
	var nodeIDs []uint

	iter := c.client.Scan(ctx, 0, "node:*:traffic", 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		nodeID, err := parseNodeIDFromKey(key)
		if err != nil {
			continue
		}
		nodeIDs = append(nodeIDs, nodeID)
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return nodeIDs, nil
}

// parseNodeIDFromKey extracts node ID from Redis key
// Example: "node:123:traffic" -> 123
func parseNodeIDFromKey(key string) (uint, error) {
	var nodeID uint
	_, err := fmt.Sscanf(key, "node:%d:traffic", &nodeID)
	if err != nil {
		return 0, fmt.Errorf("invalid key format: %s", key)
	}
	return nodeID, nil
}
