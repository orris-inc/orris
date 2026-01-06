package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// alertKeyPrefix is the prefix for all alert deduplication keys
	alertKeyPrefix = "admin_alert:"
	// DefaultAlertCooldownMinutes is the default cooldown period in minutes
	DefaultAlertCooldownMinutes = 30
)

// AlertType represents different alert types for deduplication
type AlertType string

const (
	AlertTypeNodeOffline  AlertType = "node_offline"
	AlertTypeAgentOffline AlertType = "agent_offline"
)

// AlertDeduplicator provides Redis-based alert deduplication
type AlertDeduplicator struct {
	client *redis.Client
}

// NewAlertDeduplicator creates a new AlertDeduplicator instance
func NewAlertDeduplicator(client *redis.Client) *AlertDeduplicator {
	return &AlertDeduplicator{client: client}
}

// buildKey builds the Redis key for alert deduplication
// Format: admin_alert:{type}:{resource_id}
func (d *AlertDeduplicator) buildKey(alertType AlertType, resourceID uint) string {
	return fmt.Sprintf("%s%s:%d", alertKeyPrefix, alertType, resourceID)
}

// ShouldAlert checks if an alert should be sent for the given resource
// Returns true if the alert should be sent (not in cooldown), false otherwise
func (d *AlertDeduplicator) ShouldAlert(ctx context.Context, alertType AlertType, resourceID uint) (bool, error) {
	key := d.buildKey(alertType, resourceID)

	// Check if key exists
	exists, err := d.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check alert key: %w", err)
	}

	// If key exists, we're in cooldown period
	return exists == 0, nil
}

// MarkAlerted marks that an alert has been sent for the given resource
// The alert will be in cooldown for the specified TTL
func (d *AlertDeduplicator) MarkAlerted(ctx context.Context, alertType AlertType, resourceID uint, ttl time.Duration) error {
	key := d.buildKey(alertType, resourceID)

	// Set key with TTL
	err := d.client.Set(ctx, key, "1", ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to mark alert: %w", err)
	}

	return nil
}

// TryAcquireAlertLock atomically checks and acquires an alert lock using SetNX.
// Returns true if the lock was acquired (alert should be sent), false if already in cooldown.
// This prevents TOCTOU race conditions in multi-instance deployments.
func (d *AlertDeduplicator) TryAcquireAlertLock(ctx context.Context, alertType AlertType, resourceID uint, ttl time.Duration) (bool, error) {
	key := d.buildKey(alertType, resourceID)

	// SetNX is atomic: only sets if key doesn't exist
	acquired, err := d.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire alert lock: %w", err)
	}

	return acquired, nil
}

// ClearAlert clears the alert cooldown for the given resource
// This can be used when a resource comes back online
func (d *AlertDeduplicator) ClearAlert(ctx context.Context, alertType AlertType, resourceID uint) error {
	key := d.buildKey(alertType, resourceID)

	err := d.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to clear alert: %w", err)
	}

	return nil
}

// GetRemainingCooldown returns the remaining cooldown time for an alert
// Returns 0 if not in cooldown
func (d *AlertDeduplicator) GetRemainingCooldown(ctx context.Context, alertType AlertType, resourceID uint) (time.Duration, error) {
	key := d.buildKey(alertType, resourceID)

	ttl, err := d.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get cooldown: %w", err)
	}

	// TTL returns -2 if key doesn't exist, -1 if no TTL set
	if ttl < 0 {
		return 0, nil
	}

	return ttl, nil
}
