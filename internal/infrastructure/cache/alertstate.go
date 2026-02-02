package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// alertStateKeyPrefix is the prefix for all alert state keys
	alertStateKeyPrefix = "alert_state:"

	// alertStateTTL is the maximum time an alert state can exist without being cleaned up.
	// This prevents resource leaks if a resource is deleted without calling ClearState.
	// 7 days is long enough for any reasonable alert lifecycle.
	alertStateTTL = 7 * 24 * time.Hour
)

// AlertState represents the state of an alert
type AlertState string

const (
	// AlertStateNormal indicates the resource is operating normally
	AlertStateNormal AlertState = "normal"
	// AlertStateFiring indicates an alert has been triggered for the resource
	AlertStateFiring AlertState = "firing"
)

// AlertResourceType represents different resource types for alerts
type AlertResourceType string

const (
	AlertResourceTypeNode  AlertResourceType = "node"
	AlertResourceTypeAgent AlertResourceType = "agent"
)

// AlertStateData holds the state information for an alert
type AlertStateData struct {
	State          AlertState `json:"state"`
	FiredAt        *time.Time `json:"fired_at,omitempty"`
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty"`
	NotifyCount    int        `json:"notify_count"`
}

// AlertStateManager manages alert states using Redis
// It implements a state machine model for alert lifecycle:
//
//	Normal -> Firing (when resource goes offline beyond threshold)
//	Firing -> Normal (when resource comes back online)
type AlertStateManager struct {
	client *redis.Client
}

// NewAlertStateManager creates a new AlertStateManager instance
func NewAlertStateManager(client *redis.Client) *AlertStateManager {
	return &AlertStateManager{client: client}
}

// buildKey builds the Redis key for alert state
// Format: alert_state:{resource_type}:{resource_id}
func (m *AlertStateManager) buildKey(resourceType AlertResourceType, resourceID uint) string {
	return fmt.Sprintf("%s%s:%d", alertStateKeyPrefix, resourceType, resourceID)
}

// GetState retrieves the current alert state for a resource
// Returns nil if no state exists (resource is in implicit normal state)
func (m *AlertStateManager) GetState(ctx context.Context, resourceType AlertResourceType, resourceID uint) (*AlertStateData, error) {
	key := m.buildKey(resourceType, resourceID)

	data, err := m.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // No state = implicit normal
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert state: %w", err)
	}

	var state AlertStateData
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert state: %w", err)
	}

	return &state, nil
}

// TransitionToFiring atomically transitions an alert from Normal to Firing state.
// Returns (true, nil) if this is a new firing (state changed from Normal/nil to Firing)
// Returns (false, nil) if already in Firing state (no notification needed)
// The state is stored with alertStateTTL to prevent resource leaks.
func (m *AlertStateManager) TransitionToFiring(ctx context.Context, resourceType AlertResourceType, resourceID uint, now time.Time) (isNewFiring bool, err error) {
	key := m.buildKey(resourceType, resourceID)

	// Use Lua script for atomic read-check-write
	script := redis.NewScript(`
		local key = KEYS[1]
		local newState = ARGV[1]
		local ttlSeconds = tonumber(ARGV[2])

		local existing = redis.call('GET', key)
		if existing then
			local data = cjson.decode(existing)
			if data.state == 'firing' then
				return 0  -- Already firing, no change
			end
		end

		-- Transition to firing with TTL to prevent resource leaks
		redis.call('SET', key, newState, 'EX', ttlSeconds)
		return 1  -- New firing
	`)

	state := AlertStateData{
		State:          AlertStateFiring,
		FiredAt:        &now,
		LastNotifiedAt: &now,
		NotifyCount:    1,
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return false, fmt.Errorf("failed to marshal alert state: %w", err)
	}

	ttlSeconds := int(alertStateTTL.Seconds())
	result, err := script.Run(ctx, m.client, []string{key}, string(stateJSON), ttlSeconds).Int()
	if err != nil {
		return false, fmt.Errorf("failed to transition to firing: %w", err)
	}

	return result == 1, nil
}

// TransitionToNormal atomically transitions an alert from Firing to Normal state.
// Returns (true, firedAt, nil) if the alert was in Firing state (recovery notification may be needed)
// Returns (false, nil, nil) if the alert was not in Firing state
// The state key is deleted on transition to Normal.
// This operation is atomic to prevent duplicate recovery notifications in multi-instance deployments.
func (m *AlertStateManager) TransitionToNormal(ctx context.Context, resourceType AlertResourceType, resourceID uint) (wasFiring bool, firedAt *time.Time, err error) {
	key := m.buildKey(resourceType, resourceID)

	// Use Lua script for atomic get-and-delete to prevent race conditions
	// in multi-instance deployments (TOCTOU-safe)
	script := redis.NewScript(`
		local key = KEYS[1]
		local existing = redis.call('GET', key)
		if not existing then
			return nil  -- No state = was normal
		end
		redis.call('DEL', key)
		return existing
	`)

	result, err := script.Run(ctx, m.client, []string{key}).Result()
	if err == redis.Nil || result == nil {
		return false, nil, nil // No state = was normal
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to transition to normal: %w", err)
	}

	var state AlertStateData
	if err := json.Unmarshal([]byte(result.(string)), &state); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal alert state: %w", err)
	}

	if state.State == AlertStateFiring {
		return true, state.FiredAt, nil
	}

	return false, nil, nil
}

// ShouldRepeatNotify checks if a repeat notification should be sent for a firing alert.
// Returns true if:
// 1. The alert is in Firing state
// 2. repeatInterval > 0
// 3. Time since last notification >= repeatInterval
func (m *AlertStateManager) ShouldRepeatNotify(ctx context.Context, resourceType AlertResourceType, resourceID uint, repeatInterval time.Duration) (bool, error) {
	if repeatInterval <= 0 {
		return false, nil // Repeat notifications disabled
	}

	state, err := m.GetState(ctx, resourceType, resourceID)
	if err != nil {
		return false, err
	}

	if state == nil || state.State != AlertStateFiring {
		return false, nil // Not firing
	}

	if state.LastNotifiedAt == nil {
		return true, nil // No previous notification recorded
	}

	return time.Since(*state.LastNotifiedAt) >= repeatInterval, nil
}

// MarkNotified atomically updates the last notification time for a firing alert.
// This should be called after successfully sending a repeat notification.
// This operation is atomic to prevent lost updates in multi-instance deployments.
func (m *AlertStateManager) MarkNotified(ctx context.Context, resourceType AlertResourceType, resourceID uint, now time.Time) error {
	key := m.buildKey(resourceType, resourceID)

	// Use Lua script for atomic read-modify-write to prevent lost updates
	script := redis.NewScript(`
		local key = KEYS[1]
		local nowStr = ARGV[1]
		local ttlSeconds = tonumber(ARGV[2])

		local existing = redis.call('GET', key)
		if not existing then
			return 0  -- No state, nothing to update
		end

		local data = cjson.decode(existing)
		data.last_notified_at = nowStr
		data.notify_count = (data.notify_count or 0) + 1

		-- Preserve TTL when updating
		redis.call('SET', key, cjson.encode(data), 'EX', ttlSeconds)
		return 1
	`)

	// Format time as RFC3339 for JSON compatibility
	nowStr := now.Format(time.RFC3339Nano)
	ttlSeconds := int(alertStateTTL.Seconds())

	_, err := script.Run(ctx, m.client, []string{key}, nowStr, ttlSeconds).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to mark notified: %w", err)
	}

	return nil
}

// ClearState forcefully clears the alert state for a resource.
// Use this for cleanup or manual intervention scenarios.
// IMPORTANT: This should be called when a resource (node/agent) is deleted
// to prevent stale alert states from accumulating in Redis.
// Note: States also have a TTL (alertStateTTL) as a safety net against leaks.
func (m *AlertStateManager) ClearState(ctx context.Context, resourceType AlertResourceType, resourceID uint) error {
	key := m.buildKey(resourceType, resourceID)

	if err := m.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to clear alert state: %w", err)
	}

	return nil
}
