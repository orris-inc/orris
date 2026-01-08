package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// GetTotalTrafficBySubscriptionIDs returns total traffic for subscription IDs within a time range.
// Aggregates hourly data from Redis for the given subscription IDs and resource type.
// If resourceType is empty, aggregates all resource types.
// Only returns data within the last 24 hours (Redis hourly data TTL).
func (c *RedisHourlyTrafficCache) GetTotalTrafficBySubscriptionIDs(ctx context.Context, subscriptionIDs []uint, resourceType string, from, to time.Time) (map[uint]*TrafficSummary, error) {
	// Handle empty subscriptionIDs
	if len(subscriptionIDs) == 0 {
		return make(map[uint]*TrafficSummary), nil
	}

	// Validate resourceType if provided
	if resourceType != "" {
		if err := validateResourceType(resourceType); err != nil {
			return nil, err
		}
	}

	// Cap time range to last 48 hours (Redis TTL constraint)
	now := biztime.NowUTC()
	maxFrom := now.Add(-48 * time.Hour)
	if from.Before(maxFrom) {
		from = maxFrom
	}
	if to.After(now) {
		to = now
	}

	// Truncate to hour boundaries in business timezone
	fromHour := biztime.TruncateToHourInBiz(from)
	toHour := biztime.TruncateToHourInBiz(to)

	// Validate time range
	if fromHour.After(toHour) {
		return make(map[uint]*TrafficSummary), nil
	}

	// Build set of subscription IDs for quick lookup
	subIDSet := make(map[uint]struct{}, len(subscriptionIDs))
	for _, id := range subscriptionIDs {
		subIDSet[id] = struct{}{}
	}

	// Initialize result map
	result := make(map[uint]*TrafficSummary, len(subscriptionIDs))

	// Collect all active keys from each hour
	var allKeys []string
	current := fromHour
	for !current.After(toHour) {
		hourKey := formatHourKey(current)
		activeKey := hourlyActiveSetKey(hourKey)

		keys, err := c.client.SMembers(ctx, activeKey).Result()
		if err != nil && err != redis.Nil {
			c.logger.Warnw("failed to get active set for hour",
				"hour_key", hourKey,
				"error", err,
			)
			current = current.Add(time.Hour)
			continue
		}

		// Filter keys by subscription ID and resource type
		for _, key := range keys {
			_, subID, resType, _, err := parseHourlyTrafficKey(key)
			if err != nil {
				continue
			}

			// Check if subscription ID matches
			if _, ok := subIDSet[subID]; !ok {
				continue
			}

			// Check if resource type matches (if specified)
			if resourceType != "" && resType != resourceType {
				continue
			}

			allKeys = append(allKeys, key)
		}

		current = current.Add(time.Hour)
	}

	if len(allKeys) == 0 {
		c.logger.Debugw("no traffic data found for subscription IDs",
			"subscription_ids_count", len(subscriptionIDs),
			"resource_type", resourceType,
			"from", from,
			"to", to,
		)
		return result, nil
	}

	// Use pipeline to get all traffic data
	pipe := c.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(allKeys))

	for i, key := range allKeys {
		cmds[i] = pipe.HGetAll(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to execute pipeline for subscription traffic aggregation",
			"keys_count", len(allKeys),
			"error", err,
		)
		return nil, fmt.Errorf("failed to get subscription traffic: %w", err)
	}

	// Process results and aggregate by subscription ID
	for i, cmd := range cmds {
		values, err := cmd.Result()
		if err != nil && err != redis.Nil {
			c.logger.Warnw("failed to get traffic from pipeline result",
				"key", allKeys[i],
				"error", err,
			)
			continue
		}

		if len(values) == 0 {
			continue
		}

		// Parse key to get subscription ID
		_, subID, _, _, err := parseHourlyTrafficKey(allKeys[i])
		if err != nil {
			continue
		}

		upload, _ := strconv.ParseInt(values[hourlyFieldUpload], 10, 64)
		download, _ := strconv.ParseInt(values[hourlyFieldDownload], 10, 64)

		// Aggregate traffic for this subscription
		if upload > 0 || download > 0 {
			if result[subID] == nil {
				result[subID] = &TrafficSummary{}
			}
			result[subID].Upload += uint64(upload)
			result[subID].Download += uint64(download)
			result[subID].Total += uint64(upload + download)
		}
	}

	c.logger.Debugw("got total traffic by subscription IDs",
		"subscription_ids_count", len(subscriptionIDs),
		"resource_type", resourceType,
		"from", from,
		"to", to,
		"keys_processed", len(allKeys),
		"subscriptions_with_data", len(result),
	)

	return result, nil
}

// GetPlatformTotalTraffic returns total platform-wide traffic within a time range.
func (c *RedisHourlyTrafficCache) GetPlatformTotalTraffic(ctx context.Context, resourceType string, from, to time.Time) (*TrafficSummary, error) {
	// Validate resourceType if provided
	if resourceType != "" {
		if err := validateResourceType(resourceType); err != nil {
			return nil, err
		}
	}

	// Use batch method to get all data at once
	allData, err := c.GetAllHourlyTrafficBatch(ctx, from, to)
	if err != nil {
		return nil, err
	}

	result := &TrafficSummary{}

	// Filter by resource type and aggregate
	for _, data := range allData {
		if resourceType != "" && data.ResourceType != resourceType {
			continue
		}
		result.Upload += uint64(data.Upload)
		result.Download += uint64(data.Download)
	}

	result.Total = result.Upload + result.Download

	c.logger.Debugw("got platform total traffic",
		"resource_type", resourceType,
		"from", from,
		"to", to,
		"upload", result.Upload,
		"download", result.Download,
		"total", result.Total,
	)

	return result, nil
}

// GetTrafficGroupedBySubscription returns traffic grouped by subscription within a time range.
func (c *RedisHourlyTrafficCache) GetTrafficGroupedBySubscription(ctx context.Context, resourceType string, from, to time.Time) (map[uint]*TrafficSummary, error) {
	// Validate resourceType if provided
	if resourceType != "" {
		if err := validateResourceType(resourceType); err != nil {
			return nil, err
		}
	}

	// Use batch method to get all data at once
	allData, err := c.GetAllHourlyTrafficBatch(ctx, from, to)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]*TrafficSummary)

	// Filter by resource type and aggregate by subscription
	for _, data := range allData {
		if resourceType != "" && data.ResourceType != resourceType {
			continue
		}
		if result[data.SubscriptionID] == nil {
			result[data.SubscriptionID] = &TrafficSummary{}
		}
		result[data.SubscriptionID].Upload += uint64(data.Upload)
		result[data.SubscriptionID].Download += uint64(data.Download)
		result[data.SubscriptionID].Total += uint64(data.Upload + data.Download)
	}

	c.logger.Debugw("got traffic grouped by subscription",
		"resource_type", resourceType,
		"from", from,
		"to", to,
		"subscriptions_count", len(result),
	)

	return result, nil
}

// GetTrafficGroupedByResourceID returns traffic grouped by resource ID within a time range.
func (c *RedisHourlyTrafficCache) GetTrafficGroupedByResourceID(ctx context.Context, resourceType string, from, to time.Time) (map[uint]*TrafficSummary, error) {
	// ResourceType is required for this method
	if resourceType == "" {
		return nil, fmt.Errorf("resource type is required")
	}
	if err := validateResourceType(resourceType); err != nil {
		return nil, err
	}

	// Use batch method to get all data at once
	allData, err := c.GetAllHourlyTrafficBatch(ctx, from, to)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]*TrafficSummary)

	// Filter by resource type and aggregate by resource ID
	for _, data := range allData {
		if data.ResourceType != resourceType {
			continue
		}
		if result[data.ResourceID] == nil {
			result[data.ResourceID] = &TrafficSummary{}
		}
		result[data.ResourceID].Upload += uint64(data.Upload)
		result[data.ResourceID].Download += uint64(data.Download)
		result[data.ResourceID].Total += uint64(data.Upload + data.Download)
	}

	c.logger.Debugw("got traffic grouped by resource ID",
		"resource_type", resourceType,
		"from", from,
		"to", to,
		"resources_count", len(result),
	)

	return result, nil
}

// GetTopSubscriptionsByTraffic returns top N subscriptions by total traffic within a time range.
func (c *RedisHourlyTrafficCache) GetTopSubscriptionsByTraffic(ctx context.Context, resourceType string, from, to time.Time, limit int) ([]SubscriptionTrafficSummary, error) {
	// Get all traffic grouped by subscription
	trafficMap, err := c.GetTrafficGroupedBySubscription(ctx, resourceType, from, to)
	if err != nil {
		return nil, err
	}

	// Convert to slice for sorting
	summaries := make([]SubscriptionTrafficSummary, 0, len(trafficMap))
	for subID, traffic := range trafficMap {
		summaries = append(summaries, SubscriptionTrafficSummary{
			SubscriptionID: subID,
			Upload:         traffic.Upload,
			Download:       traffic.Download,
			Total:          traffic.Total,
		})
	}

	// Sort by total descending (simple bubble sort for small datasets)
	for i := 0; i < len(summaries)-1; i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[j].Total > summaries[i].Total {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(summaries) > limit {
		summaries = summaries[:limit]
	}

	c.logger.Debugw("got top subscriptions by traffic",
		"resource_type", resourceType,
		"from", from,
		"to", to,
		"limit", limit,
		"result_count", len(summaries),
	)

	return summaries, nil
}

// GetAllHourlyTrafficBatch returns all traffic data for a time range in a single batch operation.
// Uses Redis Pipeline to fetch all hours' data efficiently.
func (c *RedisHourlyTrafficCache) GetAllHourlyTrafficBatch(ctx context.Context, from, to time.Time) ([]HourlyTrafficData, error) {
	// Cap time range to last 48 hours (Redis TTL constraint)
	now := biztime.NowUTC()
	maxFrom := now.Add(-48 * time.Hour)
	if from.Before(maxFrom) {
		from = maxFrom
	}
	if to.After(now) {
		to = now
	}

	// Truncate to hour boundaries in business timezone
	fromHour := biztime.TruncateToHourInBiz(from)
	toHour := biztime.TruncateToHourInBiz(to)

	// Validate time range
	if fromHour.After(toHour) {
		return nil, nil
	}

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

	// Step 1: Use pipeline to get all active sets for all hours
	pipe := c.client.Pipeline()
	activeSetCmds := make([]*redis.StringSliceCmd, len(hourKeys))
	for i, hourKey := range hourKeys {
		activeKey := hourlyActiveSetKey(hourKey)
		activeSetCmds[i] = pipe.SMembers(ctx, activeKey)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to get active sets batch",
			"hours_count", len(hourKeys),
			"error", err,
		)
		return nil, fmt.Errorf("failed to get active sets batch: %w", err)
	}

	// Collect all traffic keys from all hours
	var allTrafficKeys []string
	for i, cmd := range activeSetCmds {
		keys, err := cmd.Result()
		if err != nil && err != redis.Nil {
			c.logger.Warnw("failed to get active set for hour",
				"hour_key", hourKeys[i],
				"error", err,
			)
			continue
		}
		allTrafficKeys = append(allTrafficKeys, keys...)
	}

	if len(allTrafficKeys) == 0 {
		c.logger.Debugw("no traffic data found in batch",
			"from", from,
			"to", to,
			"hours_count", len(hourKeys),
		)
		return nil, nil
	}

	// Step 2: Use pipeline to get all traffic data
	pipe = c.client.Pipeline()
	trafficCmds := make([]*redis.MapStringStringCmd, len(allTrafficKeys))
	for i, key := range allTrafficKeys {
		trafficCmds[i] = pipe.HGetAll(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		c.logger.Errorw("failed to get traffic data batch",
			"keys_count", len(allTrafficKeys),
			"error", err,
		)
		return nil, fmt.Errorf("failed to get traffic data batch: %w", err)
	}

	// Process results
	var result []HourlyTrafficData
	for i, cmd := range trafficCmds {
		values, err := cmd.Result()
		if err != nil && err != redis.Nil {
			c.logger.Warnw("failed to get traffic from pipeline result",
				"key", allTrafficKeys[i],
				"error", err,
			)
			continue
		}

		if len(values) == 0 {
			continue
		}

		// Parse key to extract subscription/resource info
		_, subscriptionID, resourceType, resourceID, err := parseHourlyTrafficKey(allTrafficKeys[i])
		if err != nil {
			c.logger.Warnw("failed to parse hourly traffic key",
				"key", allTrafficKeys[i],
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

	c.logger.Debugw("got all hourly traffic data batch",
		"from", from,
		"to", to,
		"hours_count", len(hourKeys),
		"keys_count", len(allTrafficKeys),
		"data_count", len(result),
	)

	return result, nil
}
