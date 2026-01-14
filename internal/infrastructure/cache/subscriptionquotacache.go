package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// CachedQuota represents cached subscription quota information
type CachedQuota struct {
	Limit       int64     // Traffic limit in bytes
	PeriodStart time.Time // Billing period start
	PeriodEnd   time.Time // Billing period end
	PlanType    string    // Plan type: node/forward/hybrid
	Suspended   bool      // Whether the subscription is suspended
}

// SubscriptionQuotaCache defines the interface for subscription quota caching
type SubscriptionQuotaCache interface {
	GetQuota(ctx context.Context, subscriptionID uint) (*CachedQuota, error)
	SetQuota(ctx context.Context, subscriptionID uint, quota *CachedQuota) error
	InvalidateQuota(ctx context.Context, subscriptionID uint) error
	SetSuspended(ctx context.Context, subscriptionID uint, suspended bool) error
}

const (
	quotaKeyPrefix   = "subscription:quota:"
	defaultQuotaTTL  = 1 * time.Hour
	fieldLimit       = "limit"
	fieldPeriodStart = "period_start"
	fieldPeriodEnd   = "period_end"
	fieldPlanType    = "plan_type"
	fieldSuspended   = "suspended"
)

// RedisSubscriptionQuotaCache implements SubscriptionQuotaCache using Redis Hash
type RedisSubscriptionQuotaCache struct {
	client *redis.Client
	logger logger.Interface
}

// NewRedisSubscriptionQuotaCache creates a new Redis-based subscription quota cache
func NewRedisSubscriptionQuotaCache(client *redis.Client, logger logger.Interface) *RedisSubscriptionQuotaCache {
	return &RedisSubscriptionQuotaCache{
		client: client,
		logger: logger,
	}
}

func (c *RedisSubscriptionQuotaCache) key(subscriptionID uint) string {
	return fmt.Sprintf("%s%d", quotaKeyPrefix, subscriptionID)
}

// GetQuota retrieves quota information from cache
func (c *RedisSubscriptionQuotaCache) GetQuota(ctx context.Context, subscriptionID uint) (*CachedQuota, error) {
	key := c.key(subscriptionID)

	result, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get quota from cache: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // Cache miss
	}

	quota := &CachedQuota{}

	if limitStr, ok := result[fieldLimit]; ok {
		quota.Limit, _ = strconv.ParseInt(limitStr, 10, 64)
	}

	if periodStartStr, ok := result[fieldPeriodStart]; ok {
		periodStartUnix, _ := strconv.ParseInt(periodStartStr, 10, 64)
		quota.PeriodStart = time.Unix(periodStartUnix, 0)
	}

	if periodEndStr, ok := result[fieldPeriodEnd]; ok {
		periodEndUnix, _ := strconv.ParseInt(periodEndStr, 10, 64)
		quota.PeriodEnd = time.Unix(periodEndUnix, 0)
	}

	if planType, ok := result[fieldPlanType]; ok {
		quota.PlanType = planType
	}

	if suspendedStr, ok := result[fieldSuspended]; ok {
		quota.Suspended = suspendedStr == "1"
	}

	return quota, nil
}

// SetQuota stores quota information in cache
func (c *RedisSubscriptionQuotaCache) SetQuota(ctx context.Context, subscriptionID uint, quota *CachedQuota) error {
	key := c.key(subscriptionID)

	fields := map[string]interface{}{
		fieldLimit:       quota.Limit,
		fieldPeriodStart: quota.PeriodStart.Unix(),
		fieldPeriodEnd:   quota.PeriodEnd.Unix(),
		fieldPlanType:    quota.PlanType,
		fieldSuspended:   boolToInt(quota.Suspended),
	}

	pipe := c.client.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, defaultQuotaTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set quota in cache: %w", err)
	}

	c.logger.Debugw("subscription quota cached",
		"subscription_id", subscriptionID,
		"limit", quota.Limit,
		"plan_type", quota.PlanType,
	)

	return nil
}

// InvalidateQuota removes quota information from cache
func (c *RedisSubscriptionQuotaCache) InvalidateQuota(ctx context.Context, subscriptionID uint) error {
	key := c.key(subscriptionID)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to invalidate quota cache: %w", err)
	}

	c.logger.Debugw("subscription quota cache invalidated",
		"subscription_id", subscriptionID,
	)

	return nil
}

// SetSuspended updates only the suspended field in the cache
func (c *RedisSubscriptionQuotaCache) SetSuspended(ctx context.Context, subscriptionID uint, suspended bool) error {
	key := c.key(subscriptionID)

	// Check if key exists
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check quota cache existence: %w", err)
	}

	if exists == 0 {
		// Cache miss, nothing to update
		return nil
	}

	if err := c.client.HSet(ctx, key, fieldSuspended, boolToInt(suspended)).Err(); err != nil {
		return fmt.Errorf("failed to update suspended status in cache: %w", err)
	}

	c.logger.Debugw("subscription suspended status updated in cache",
		"subscription_id", subscriptionID,
		"suspended", suspended,
	)

	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
