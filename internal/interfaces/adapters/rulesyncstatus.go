package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	ruleSyncStatusKeyPrefix = "forward_agent:%d:rule_status"
	ruleSyncStatusTTL       = 5 * time.Minute
)

// RuleSyncStatusUpdater updates rule sync status in Redis.
type RuleSyncStatusUpdater interface {
	UpdateRuleStatus(ctx context.Context, agentID uint, rules []dto.RuleSyncStatusItem) error
}

// RuleSyncStatusQuerier queries rule sync status from Redis.
type RuleSyncStatusQuerier interface {
	GetRuleStatus(ctx context.Context, agentID uint) (*dto.RuleSyncStatusQueryResult, error)
}

// RuleSyncStatusAdapter implements RuleSyncStatusUpdater, RuleSyncStatusQuerier,
// and usecases.RuleSyncStatusBatchQuerier interfaces.
type RuleSyncStatusAdapter struct {
	redisClient *redis.Client
	logger      logger.Interface
}

// NewRuleSyncStatusAdapter creates a new rule sync status adapter.
func NewRuleSyncStatusAdapter(
	redisClient *redis.Client,
	logger logger.Interface,
) *RuleSyncStatusAdapter {
	return &RuleSyncStatusAdapter{
		redisClient: redisClient,
		logger:      logger,
	}
}

// UpdateRuleStatus updates rule sync status in Redis.
func (a *RuleSyncStatusAdapter) UpdateRuleStatus(ctx context.Context, agentID uint, rules []dto.RuleSyncStatusItem) error {
	key := fmt.Sprintf(ruleSyncStatusKeyPrefix, agentID)

	// Build hash map for all rules
	hashData := make(map[string]interface{})
	for _, rule := range rules {
		ruleJSON, err := json.Marshal(rule)
		if err != nil {
			a.logger.Errorw("failed to marshal rule sync status",
				"error", err,
				"agent_id", agentID,
				"rule_id", rule.RuleID,
			)
			continue
		}
		hashData[rule.RuleID] = string(ruleJSON)
	}

	// Store updated_at timestamp
	hashData["updated_at"] = biztime.NowUTC().Unix()

	// Store all rules in Redis hash with TTL
	pipe := a.redisClient.Pipeline()

	// Clear existing data first
	pipe.Del(ctx, key)

	// Set new data
	if len(hashData) > 0 {
		pipe.HSet(ctx, key, hashData)
	}

	pipe.Expire(ctx, key, ruleSyncStatusTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		a.logger.Errorw("failed to store rule sync status in redis",
			"error", err,
			"agent_id", agentID,
			"rule_count", len(rules),
		)
		return fmt.Errorf("failed to store rule sync status: %w", err)
	}

	a.logger.Debugw("rule sync status updated in redis",
		"agent_id", agentID,
		"rule_count", len(rules),
	)

	return nil
}

// GetRuleStatus retrieves rule sync status from Redis.
func (a *RuleSyncStatusAdapter) GetRuleStatus(ctx context.Context, agentID uint) (*dto.RuleSyncStatusQueryResult, error) {
	key := fmt.Sprintf(ruleSyncStatusKeyPrefix, agentID)

	values, err := a.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		a.logger.Errorw("failed to get rule sync status from redis",
			"error", err,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("failed to get rule sync status: %w", err)
	}

	result := &dto.RuleSyncStatusQueryResult{
		Rules:     []dto.RuleSyncStatusItem{},
		UpdatedAt: 0,
	}

	if len(values) == 0 {
		return result, nil
	}

	// Parse updated_at timestamp
	if updatedAtStr, ok := values["updated_at"]; ok {
		fmt.Sscanf(updatedAtStr, "%d", &result.UpdatedAt)
	}

	// Parse all rule statuses
	// Pre-allocate with estimated capacity (total fields minus updated_at)
	result.Rules = make([]dto.RuleSyncStatusItem, 0, len(values)-1)
	for fieldName, ruleJSON := range values {
		// Skip the updated_at field
		if fieldName == "updated_at" {
			continue
		}

		var rule dto.RuleSyncStatusItem
		if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
			a.logger.Warnw("failed to unmarshal rule sync status",
				"error", err,
				"agent_id", agentID,
				"field_name", fieldName,
			)
			continue
		}

		result.Rules = append(result.Rules, rule)
	}

	return result, nil
}

// GetMultipleRuleStatus retrieves rule sync status for multiple agents in batch.
func (a *RuleSyncStatusAdapter) GetMultipleRuleStatus(ctx context.Context, agentIDs []uint) (map[uint]*dto.RuleSyncStatusQueryResult, error) {
	result := make(map[uint]*dto.RuleSyncStatusQueryResult)

	if len(agentIDs) == 0 {
		return result, nil
	}

	// Use pipeline for efficient batch querying
	pipe := a.redisClient.Pipeline()
	cmds := make(map[uint]*redis.MapStringStringCmd)

	for _, agentID := range agentIDs {
		key := fmt.Sprintf(ruleSyncStatusKeyPrefix, agentID)
		cmds[agentID] = pipe.HGetAll(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		a.logger.Errorw("failed to get multiple rule sync statuses from redis",
			"error", err,
			"agent_count", len(agentIDs),
		)
		return result, fmt.Errorf("failed to get rule sync statuses: %w", err)
	}

	// Parse results for each agent
	for agentID, cmd := range cmds {
		values, err := cmd.Result()
		if err != nil || len(values) == 0 {
			continue
		}

		queryResult := &dto.RuleSyncStatusQueryResult{
			Rules:     []dto.RuleSyncStatusItem{},
			UpdatedAt: 0,
		}

		// Parse updated_at timestamp
		if updatedAtStr, ok := values["updated_at"]; ok {
			fmt.Sscanf(updatedAtStr, "%d", &queryResult.UpdatedAt)
		}

		// Parse all rule statuses
		queryResult.Rules = make([]dto.RuleSyncStatusItem, 0, len(values)-1)
		for fieldName, ruleJSON := range values {
			if fieldName == "updated_at" {
				continue
			}

			var rule dto.RuleSyncStatusItem
			if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
				a.logger.Warnw("failed to unmarshal rule sync status",
					"error", err,
					"agent_id", agentID,
					"field_name", fieldName,
				)
				continue
			}

			queryResult.Rules = append(queryResult.Rules, rule)
		}

		result[agentID] = queryResult
	}

	return result, nil
}
