package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// Redis key patterns for hourly traffic
	// Key format: sub_hourly:{hour}:{subscriptionID}:{resourceType}:{resourceID}
	// hour format: 2025010712 (YYYYMMDDHH in business timezone)
	hourlyTrafficKeyPrefix = "sub_hourly:"

	// Active set key format: sub_hourly:active:{hour}
	// Tracks which keys have data for a given hour (for efficient aggregation)
	hourlyActiveSetPrefix = "sub_hourly:active:"

	// TTL: 49 hours (48 hours + 1 hour buffer)
	hourlyTrafficTTL = 49 * time.Hour

	// Hash field names for hourly traffic
	hourlyFieldUpload   = "upload"
	hourlyFieldDownload = "download"

	// Hour key format layout (business timezone)
	hourKeyLayout = "2006010215"
)

// ErrInvalidResourceType is returned when resource type contains invalid characters.
var ErrInvalidResourceType = fmt.Errorf("invalid resource type: must not contain colon")

// validateResourceType checks if the resource type is valid for use in Redis keys.
// Resource type must not contain colon (:) as it's used as a delimiter in key format.
func validateResourceType(resourceType string) error {
	if strings.Contains(resourceType, ":") {
		return ErrInvalidResourceType
	}
	return nil
}

// HourlyTrafficPoint represents traffic data for a specific hour.
type HourlyTrafficPoint struct {
	Hour     time.Time
	Upload   int64
	Download int64
}

// HourlyTrafficData represents traffic data for a subscription resource at a specific hour.
type HourlyTrafficData struct {
	SubscriptionID uint
	ResourceType   string
	ResourceID     uint
	Upload         int64
	Download       int64
}

// TrafficSummary represents aggregated traffic with upload/download breakdown.
type TrafficSummary struct {
	Upload   uint64
	Download uint64
	Total    uint64
}

// HourlyTrafficCache defines the interface for hourly traffic caching.
type HourlyTrafficCache interface {
	// IncrementHourlyTraffic increments traffic for the current hour.
	IncrementHourlyTraffic(ctx context.Context, subscriptionID uint, resourceType string, resourceID uint, upload, download int64) error

	// GetHourlyTraffic returns traffic for a specific hour.
	GetHourlyTraffic(ctx context.Context, hour time.Time, subscriptionID uint, resourceType string, resourceID uint) (upload, download int64, err error)

	// GetHourlyTrafficRange returns traffic data for a time range (used for trend queries).
	GetHourlyTrafficRange(ctx context.Context, subscriptionID uint, resourceType string, resourceID uint, from, to time.Time) ([]HourlyTrafficPoint, error)

	// GetAllHourlyTraffic returns all active traffic data for a specific hour (used for daily aggregation).
	// IMPORTANT: Only use this for hours that are safely in the past (at least 1 hour ago)
	// to avoid race conditions with IncrementHourlyTraffic.
	GetAllHourlyTraffic(ctx context.Context, hour time.Time) ([]HourlyTrafficData, error)

	// GetAndCleanupHour atomically retrieves and removes all data for a specific hour.
	// This is the preferred method for daily aggregation as it prevents data loss from
	// race conditions between GetAllHourlyTraffic and CleanupHour.
	// IMPORTANT: Only use this for hours that are safely in the past (at least 1 hour ago).
	GetAndCleanupHour(ctx context.Context, hour time.Time) ([]HourlyTrafficData, error)

	// CleanupHour removes all data for a specific hour atomically.
	// Prefer GetAndCleanupHour for daily aggregation to prevent race conditions.
	CleanupHour(ctx context.Context, hour time.Time) error

	// GetTotalTrafficBySubscriptionIDs returns total traffic for subscription IDs within a time range.
	// Aggregates hourly data from Redis for the given subscription IDs and resource type.
	// If resourceType is empty, aggregates all resource types.
	// Only returns data within the last 48 hours (Redis hourly data TTL).
	GetTotalTrafficBySubscriptionIDs(ctx context.Context, subscriptionIDs []uint, resourceType string, from, to time.Time) (map[uint]*TrafficSummary, error)

	// ========== Admin Analytics Methods ==========

	// GetPlatformTotalTraffic returns total platform-wide traffic within a time range.
	// If resourceType is empty, aggregates all resource types.
	// Only returns data within the last 48 hours (Redis hourly data TTL).
	GetPlatformTotalTraffic(ctx context.Context, resourceType string, from, to time.Time) (*TrafficSummary, error)

	// GetTrafficGroupedBySubscription returns traffic grouped by subscription within a time range.
	// If resourceType is empty, aggregates all resource types.
	// Only returns data within the last 48 hours (Redis hourly data TTL).
	GetTrafficGroupedBySubscription(ctx context.Context, resourceType string, from, to time.Time) (map[uint]*TrafficSummary, error)

	// GetTrafficGroupedByResourceID returns traffic grouped by resource ID within a time range.
	// Only returns data within the last 48 hours (Redis hourly data TTL).
	GetTrafficGroupedByResourceID(ctx context.Context, resourceType string, from, to time.Time) (map[uint]*TrafficSummary, error)

	// GetTopSubscriptionsByTraffic returns top N subscriptions by total traffic within a time range.
	// If resourceType is empty, aggregates all resource types.
	// Only returns data within the last 48 hours (Redis hourly data TTL).
	GetTopSubscriptionsByTraffic(ctx context.Context, resourceType string, from, to time.Time, limit int) ([]SubscriptionTrafficSummary, error)

	// GetAllHourlyTrafficBatch returns all traffic data for a time range in a single batch operation.
	// This is more efficient than calling GetAllHourlyTraffic for each hour.
	// Only returns data within the last 48 hours (Redis hourly data TTL).
	GetAllHourlyTrafficBatch(ctx context.Context, from, to time.Time) ([]HourlyTrafficData, error)
}

// SubscriptionTrafficSummary represents aggregated traffic for a subscription.
type SubscriptionTrafficSummary struct {
	SubscriptionID uint
	Upload         uint64
	Download       uint64
	Total          uint64
}

// RedisHourlyTrafficCache implements HourlyTrafficCache using Redis.
type RedisHourlyTrafficCache struct {
	client *redis.Client
	logger logger.Interface
}

// NewRedisHourlyTrafficCache creates a new RedisHourlyTrafficCache instance.
func NewRedisHourlyTrafficCache(
	client *redis.Client,
	logger logger.Interface,
) HourlyTrafficCache {
	return &RedisHourlyTrafficCache{
		client: client,
		logger: logger,
	}
}

// formatHourKey formats a time to hour key string in business timezone.
// Format: YYYYMMDDHH (e.g., 2025010712)
func formatHourKey(t time.Time) string {
	return t.In(biztime.Location()).Format(hourKeyLayout)
}

// parseHourKey parses an hour key string back to time.Time (UTC).
func parseHourKey(hourKey string) (time.Time, error) {
	t, err := time.ParseInLocation(hourKeyLayout, hourKey, biztime.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour key format %q: %w", hourKey, err)
	}
	return t.UTC(), nil
}

// hourlyTrafficKey generates the Redis key for hourly traffic data.
// Format: sub_hourly:{hour}:{subscriptionID}:{resourceType}:{resourceID}
func hourlyTrafficKey(hourKey string, subscriptionID uint, resourceType string, resourceID uint) string {
	return fmt.Sprintf("%s%s:%d:%s:%d", hourlyTrafficKeyPrefix, hourKey, subscriptionID, resourceType, resourceID)
}

// hourlyActiveSetKey generates the Redis key for the active set of an hour.
// Format: sub_hourly:active:{hour}
func hourlyActiveSetKey(hourKey string) string {
	return fmt.Sprintf("%s%s", hourlyActiveSetPrefix, hourKey)
}

// parseHourlyTrafficKey extracts components from a Redis key.
// Example: "sub_hourly:2025010712:123:node:456" -> hourKey="2025010712", subscriptionID=123, resourceType="node", resourceID=456
func parseHourlyTrafficKey(key string) (hourKey string, subscriptionID uint, resourceType string, resourceID uint, err error) {
	// Remove prefix
	if !strings.HasPrefix(key, hourlyTrafficKeyPrefix) {
		return "", 0, "", 0, fmt.Errorf("invalid key prefix: %s", key)
	}

	remainder := strings.TrimPrefix(key, hourlyTrafficKeyPrefix)
	parts := strings.Split(remainder, ":")
	if len(parts) != 4 {
		return "", 0, "", 0, fmt.Errorf("invalid key format: %s", key)
	}

	hourKey = parts[0]

	sid, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return "", 0, "", 0, fmt.Errorf("invalid subscription ID in key %s: %w", key, err)
	}

	resourceType = parts[2]

	rid, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return "", 0, "", 0, fmt.Errorf("invalid resource ID in key %s: %w", key, err)
	}

	return hourKey, uint(sid), resourceType, uint(rid), nil
}

// IncrementHourlyTraffic increments traffic for the current hour.
func (c *RedisHourlyTrafficCache) IncrementHourlyTraffic(ctx context.Context, subscriptionID uint, resourceType string, resourceID uint, upload, download int64) error {
	if err := validateResourceType(resourceType); err != nil {
		return err
	}

	if upload == 0 && download == 0 {
		return nil
	}

	// Get current hour key in business timezone
	hourKey := formatHourKey(biztime.NowUTC())
	trafficKey := hourlyTrafficKey(hourKey, subscriptionID, resourceType, resourceID)
	activeKey := hourlyActiveSetKey(hourKey)

	pipe := c.client.Pipeline()

	// Increment traffic values
	if upload > 0 {
		pipe.HIncrBy(ctx, trafficKey, hourlyFieldUpload, upload)
	}
	if download > 0 {
		pipe.HIncrBy(ctx, trafficKey, hourlyFieldDownload, download)
	}

	// Set expiration to prevent memory leak
	pipe.Expire(ctx, trafficKey, hourlyTrafficTTL)

	// Add to active set for efficient lookup during aggregation
	pipe.SAdd(ctx, activeKey, trafficKey)
	pipe.Expire(ctx, activeKey, hourlyTrafficTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Errorw("failed to increment hourly traffic in redis",
			"subscription_id", subscriptionID,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"hour_key", hourKey,
			"upload", upload,
			"download", download,
			"error", err,
		)
		return fmt.Errorf("failed to increment hourly traffic: %w", err)
	}

	c.logger.Debugw("hourly traffic incremented in redis",
		"subscription_id", subscriptionID,
		"resource_type", resourceType,
		"resource_id", resourceID,
		"hour_key", hourKey,
		"upload", upload,
		"download", download,
	)

	return nil
}

// GetHourlyTraffic returns traffic for a specific hour.
func (c *RedisHourlyTrafficCache) GetHourlyTraffic(ctx context.Context, hour time.Time, subscriptionID uint, resourceType string, resourceID uint) (upload, download int64, err error) {
	if err := validateResourceType(resourceType); err != nil {
		return 0, 0, err
	}

	hourKey := formatHourKey(hour)
	trafficKey := hourlyTrafficKey(hourKey, subscriptionID, resourceType, resourceID)

	values, err := c.client.HGetAll(ctx, trafficKey).Result()
	if err != nil && err != redis.Nil {
		c.logger.Warnw("failed to get hourly traffic from redis",
			"subscription_id", subscriptionID,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"hour_key", hourKey,
			"error", err,
		)
		return 0, 0, fmt.Errorf("failed to get hourly traffic: %w", err)
	}

	if len(values) == 0 {
		return 0, 0, nil
	}

	upload, _ = strconv.ParseInt(values[hourlyFieldUpload], 10, 64)
	download, _ = strconv.ParseInt(values[hourlyFieldDownload], 10, 64)

	return upload, download, nil
}

// GetHourlyTrafficRange returns traffic data for a time range.
func (c *RedisHourlyTrafficCache) GetHourlyTrafficRange(ctx context.Context, subscriptionID uint, resourceType string, resourceID uint, from, to time.Time) ([]HourlyTrafficPoint, error) {
	if err := validateResourceType(resourceType); err != nil {
		return nil, err
	}

	// Truncate to hour boundaries
	fromHour := biztime.TruncateToHourInBiz(from)
	toHour := biztime.TruncateToHourInBiz(to)

	// Build list of hour keys
	var hourKeys []string
	current := fromHour
	for !current.After(toHour) {
		hourKeys = append(hourKeys, formatHourKey(current))
		current = current.Add(time.Hour)
	}

	if len(hourKeys) == 0 {
		return nil, nil
	}

	// Use pipeline for batch queries
	pipe := c.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(hourKeys))

	for i, hourKey := range hourKeys {
		trafficKey := hourlyTrafficKey(hourKey, subscriptionID, resourceType, resourceID)
		cmds[i] = pipe.HGetAll(ctx, trafficKey)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		c.logger.Warnw("failed to execute pipeline for hourly traffic range",
			"subscription_id", subscriptionID,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"from", from,
			"to", to,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get hourly traffic range: %w", err)
	}

	// Process results
	var result []HourlyTrafficPoint
	current = fromHour
	for i, cmd := range cmds {
		values, err := cmd.Result()
		if err != nil && err != redis.Nil {
			c.logger.Warnw("failed to get hourly traffic from pipeline result",
				"hour_key", hourKeys[i],
				"error", err,
			)
			current = current.Add(time.Hour)
			continue
		}

		if len(values) > 0 {
			upload, _ := strconv.ParseInt(values[hourlyFieldUpload], 10, 64)
			download, _ := strconv.ParseInt(values[hourlyFieldDownload], 10, 64)

			if upload > 0 || download > 0 {
				result = append(result, HourlyTrafficPoint{
					Hour:     current,
					Upload:   upload,
					Download: download,
				})
			}
		}

		current = current.Add(time.Hour)
	}

	c.logger.Debugw("got hourly traffic range from redis",
		"subscription_id", subscriptionID,
		"resource_type", resourceType,
		"resource_id", resourceID,
		"from", from,
		"to", to,
		"points_count", len(result),
	)

	return result, nil
}

// GetAllHourlyTraffic returns all active traffic data for a specific hour.
func (c *RedisHourlyTrafficCache) GetAllHourlyTraffic(ctx context.Context, hour time.Time) ([]HourlyTrafficData, error) {
	hourKey := formatHourKey(hour)
	activeKey := hourlyActiveSetKey(hourKey)

	// Get all active keys for this hour
	keys, err := c.client.SMembers(ctx, activeKey).Result()
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to get active set for hour", "hour_key", hourKey, "error", err)
		return nil, fmt.Errorf("failed to get active set: %w", err)
	}

	if len(keys) == 0 {
		c.logger.Debugw("no active traffic data for hour", "hour_key", hourKey)
		return nil, nil
	}

	// Use pipeline to get all traffic data
	pipe := c.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.HGetAll(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to execute pipeline for all hourly traffic",
			"hour_key", hourKey,
			"keys_count", len(keys),
			"error", err,
		)
		return nil, fmt.Errorf("failed to get all hourly traffic: %w", err)
	}

	// Process results
	var result []HourlyTrafficData
	for i, cmd := range cmds {
		values, err := cmd.Result()
		if err != nil && err != redis.Nil {
			c.logger.Warnw("failed to get traffic from pipeline result",
				"key", keys[i],
				"error", err,
			)
			continue
		}

		if len(values) == 0 {
			continue
		}

		// Parse key to extract subscription/resource info
		_, subscriptionID, resourceType, resourceID, err := parseHourlyTrafficKey(keys[i])
		if err != nil {
			c.logger.Warnw("failed to parse hourly traffic key",
				"key", keys[i],
				"error", err,
			)
			continue
		}

		upload, _ := strconv.ParseInt(values[hourlyFieldUpload], 10, 64)
		download, _ := strconv.ParseInt(values[hourlyFieldDownload], 10, 64)

		if upload > 0 || download > 0 {
			result = append(result, HourlyTrafficData{
				SubscriptionID: subscriptionID,
				ResourceType:   resourceType,
				ResourceID:     resourceID,
				Upload:         upload,
				Download:       download,
			})
		}
	}

	c.logger.Debugw("got all hourly traffic data",
		"hour_key", hourKey,
		"active_keys_count", len(keys),
		"data_count", len(result),
	)

	return result, nil
}

// cleanupHourScript is a Lua script for atomic cleanup of hourly traffic data.
// It atomically gets all keys from active set, deletes them, and deletes the active set.
// This prevents race conditions where new keys could be added between SMEMBERS and DEL.
var cleanupHourScript = redis.NewScript(`
local activeKey = KEYS[1]
local keys = redis.call('SMEMBERS', activeKey)
local deleted = 0
for _, key in ipairs(keys) do
    redis.call('DEL', key)
    deleted = deleted + 1
end
redis.call('DEL', activeKey)
return deleted
`)

// getAndCleanupHourScript atomically retrieves all traffic data and cleans up.
// Returns array of [key, upload, download, key, upload, download, ...]
var getAndCleanupHourScript = redis.NewScript(`
local activeKey = KEYS[1]
local keys = redis.call('SMEMBERS', activeKey)
local result = {}
for _, key in ipairs(keys) do
    local data = redis.call('HGETALL', key)
    if #data > 0 then
        table.insert(result, key)
        local upload = '0'
        local download = '0'
        for i = 1, #data, 2 do
            if data[i] == 'upload' then
                upload = data[i+1]
            elseif data[i] == 'download' then
                download = data[i+1]
            end
        end
        table.insert(result, upload)
        table.insert(result, download)
    end
    redis.call('DEL', key)
end
redis.call('DEL', activeKey)
return result
`)

// GetAndCleanupHour atomically retrieves and removes all data for a specific hour.
// This prevents race conditions between GetAllHourlyTraffic and CleanupHour.
func (c *RedisHourlyTrafficCache) GetAndCleanupHour(ctx context.Context, hour time.Time) ([]HourlyTrafficData, error) {
	hourKey := formatHourKey(hour)
	activeKey := hourlyActiveSetKey(hourKey)

	rawResult, err := getAndCleanupHourScript.Run(ctx, c.client, []string{activeKey}).Result()
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to get and cleanup hour data",
			"hour_key", hourKey,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get and cleanup hour data: %w", err)
	}

	// Parse result: [key, upload, download, key, upload, download, ...]
	items, ok := rawResult.([]any)
	if !ok || len(items) == 0 {
		c.logger.Debugw("no traffic data for hour", "hour_key", hourKey)
		return nil, nil
	}

	var result []HourlyTrafficData
	for i := 0; i+2 < len(items); i += 3 {
		key, ok := items[i].(string)
		if !ok {
			continue
		}

		_, subscriptionID, resourceType, resourceID, err := parseHourlyTrafficKey(key)
		if err != nil {
			c.logger.Warnw("failed to parse hourly traffic key",
				"key", key,
				"error", err,
			)
			continue
		}

		uploadStr, _ := items[i+1].(string)
		downloadStr, _ := items[i+2].(string)
		upload, _ := strconv.ParseInt(uploadStr, 10, 64)
		download, _ := strconv.ParseInt(downloadStr, 10, 64)

		if upload > 0 || download > 0 {
			result = append(result, HourlyTrafficData{
				SubscriptionID: subscriptionID,
				ResourceType:   resourceType,
				ResourceID:     resourceID,
				Upload:         upload,
				Download:       download,
			})
		}
	}

	c.logger.Infow("got and cleaned up hourly traffic data",
		"hour_key", hourKey,
		"data_count", len(result),
	)

	return result, nil
}

// CleanupHour removes all data for a specific hour atomically.
// Uses Lua script to prevent race conditions with concurrent IncrementHourlyTraffic calls.
func (c *RedisHourlyTrafficCache) CleanupHour(ctx context.Context, hour time.Time) error {
	hourKey := formatHourKey(hour)
	activeKey := hourlyActiveSetKey(hourKey)

	deleted, err := cleanupHourScript.Run(ctx, c.client, []string{activeKey}).Int()
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to cleanup hour data",
			"hour_key", hourKey,
			"error", err,
		)
		return fmt.Errorf("failed to cleanup hour data: %w", err)
	}

	if deleted == 0 {
		c.logger.Debugw("no keys to cleanup for hour", "hour_key", hourKey)
	} else {
		c.logger.Infow("cleaned up hourly traffic data",
			"hour_key", hourKey,
			"deleted_keys_count", deleted+1, // +1 for active set
		)
	}

	return nil
}
