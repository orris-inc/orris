package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/utils/jsonutil"
)

// List retrieves a paginated list of forward rules with filtering.
func (r *ForwardRuleRepositoryImpl) List(ctx context.Context, filter forward.ListFilter) ([]*forward.ForwardRule, int64, error) {
	tx := db.GetTxFromContext(ctx, r.db)
	query := tx.Model(&models.ForwardRuleModel{})

	// Apply filters
	if filter.AgentID != 0 {
		query = query.Where("agent_id = ?", filter.AgentID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	} else if !filter.IncludeUserRules {
		// When IncludeUserRules is false (default), exclude rules created by users
		query = query.Where("user_id IS NULL OR user_id = 0")
	}
	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Protocol != "" {
		query = query.Where("protocol = ?", filter.Protocol)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.RuleType != "" {
		query = query.Where("rule_type = ?", filter.RuleType)
	}
	if filter.ExternalSource != "" {
		query = query.Where("external_source = ?", filter.ExternalSource)
	}
	if len(filter.GroupIDs) > 0 {
		// Use JSON_OVERLAPS to check if group_ids array contains any of the filter group IDs
		// JSON_OVERLAPS returns true if two JSON arrays have at least one element in common
		// Use json.Marshal for safe JSON array construction instead of string formatting
		groupIDsJSON, _ := json.Marshal(filter.GroupIDs)
		query = query.Where("JSON_OVERLAPS(group_ids, ?)", string(groupIDsJSON))
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count forward rules", "error", err)
		return nil, 0, fmt.Errorf("failed to count forward rules: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	orderBy := strings.ToLower(filter.OrderBy)
	if orderBy == "" || !allowedRuleOrderByFields[orderBy] {
		// Default: sort by sort_order ASC, then created_at DESC
		query = query.Order("sort_order ASC, created_at DESC")
	} else {
		order := strings.ToUpper(filter.Order)
		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", orderBy, order))
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Execute query
	var ruleModels []*models.ForwardRuleModel
	if err := query.Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list forward rules", "error", err)
		return nil, 0, fmt.Errorf("failed to list forward rules: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, total, nil
}

// ListEnabled returns all enabled forward rules.
func (r *ForwardRuleRepositoryImpl) ListEnabled(ctx context.Context) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("status = ?", "enabled").Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list enabled forward rules", "error", err)
		return nil, fmt.Errorf("failed to list enabled forward rules: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListByAgentID returns all forward rules for a specific agent.
func (r *ForwardRuleRepositoryImpl) ListByAgentID(ctx context.Context, agentID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("agent_id = ?", agentID).Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list forward rules by agent ID", "agent_id", agentID, "error", err)
		return nil, fmt.Errorf("failed to list forward rules by agent ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListEnabledByAgentID returns all enabled forward rules for a specific agent.
func (r *ForwardRuleRepositoryImpl) ListEnabledByAgentID(ctx context.Context, agentID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("agent_id = ? AND status = ?", agentID, "enabled").Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list enabled forward rules by agent ID", "agent_id", agentID, "error", err)
		return nil, fmt.Errorf("failed to list enabled forward rules by agent ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListByExitAgentID returns all entrance rules for a specific exit agent.
func (r *ForwardRuleRepositoryImpl) ListByExitAgentID(ctx context.Context, exitAgentID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("exit_agent_id = ?", exitAgentID).Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list forward rules by exit agent ID", "exit_agent_id", exitAgentID, "error", err)
		return nil, fmt.Errorf("failed to list forward rules by exit agent ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListEnabledByExitAgentID returns all enabled entry rules for a specific exit agent.
// This includes rules where exit_agent_id matches OR exit_agents JSON contains the agent.
func (r *ForwardRuleRepositoryImpl) ListEnabledByExitAgentID(ctx context.Context, exitAgentID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	// Query rules where:
	// 1. exit_agent_id matches (single exit agent), OR
	// 2. exit_agents JSON array contains an object with matching agent_id
	// Note: JSON_CONTAINS returns NULL when exit_agents is NULL, so we need explicit NULL check
	if err := tx.Where(
		"status = ? AND rule_type = ? AND (exit_agent_id = ? OR (exit_agents IS NOT NULL AND JSON_CONTAINS(exit_agents, JSON_OBJECT('agent_id', ?))))",
		"enabled", "entry", exitAgentID, exitAgentID,
	).Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list enabled entry rules by exit agent ID", "exit_agent_id", exitAgentID, "error", err)
		return nil, fmt.Errorf("failed to list enabled entry rules by exit agent ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListEnabledByExitAgentIDs returns all enabled entry rules for multiple exit agents.
// This includes rules where exit_agent_id is in the list OR exit_agents JSON contains any of the agents.
func (r *ForwardRuleRepositoryImpl) ListEnabledByExitAgentIDs(ctx context.Context, exitAgentIDs []uint) ([]*forward.ForwardRule, error) {
	if len(exitAgentIDs) == 0 {
		return []*forward.ForwardRule{}, nil
	}

	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)

	// Build OR conditions for exit_agents JSON check
	// We need to check if any of the exit agent IDs exist in the exit_agents array
	// Note: JSON_CONTAINS returns NULL when exit_agents is NULL, so we need explicit NULL check
	jsonConditions := make([]string, len(exitAgentIDs))
	jsonArgs := make([]interface{}, len(exitAgentIDs))
	for i, id := range exitAgentIDs {
		jsonConditions[i] = "JSON_CONTAINS(exit_agents, JSON_OBJECT('agent_id', ?))"
		jsonArgs[i] = id
	}
	jsonOrCondition := strings.Join(jsonConditions, " OR ")

	// Build query with proper parameter ordering
	// GORM requires the slice to be passed directly for IN clause expansion
	query := fmt.Sprintf(
		"status = ? AND rule_type = ? AND (exit_agent_id IN ? OR (exit_agents IS NOT NULL AND (%s)))",
		jsonOrCondition,
	)

	// Build args: status, rule_type, exitAgentIDs (for IN), jsonArgs (for JSON_CONTAINS)
	args := make([]interface{}, 0, 2+1+len(jsonArgs))
	args = append(args, "enabled", "entry")
	args = append(args, exitAgentIDs) // GORM will expand this for IN clause
	args = append(args, jsonArgs...)

	if err := tx.Where(query, args...).Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list enabled entry rules by exit agent IDs", "exit_agent_ids", exitAgentIDs, "error", err)
		return nil, fmt.Errorf("failed to list enabled entry rules by exit agent IDs: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListEnabledByChainAgentID returns all enabled chain rules where the agent participates.
// This includes both 'chain' and 'direct_chain' rule types.
func (r *ForwardRuleRepositoryImpl) ListEnabledByChainAgentID(ctx context.Context, agentID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	// Query chain and direct_chain rules where agent is in chain_agent_ids using MySQL JSON_CONTAINS
	if err := tx.Where(
		"status = ? AND rule_type IN (?, ?) AND JSON_CONTAINS(chain_agent_ids, ?)",
		"enabled",
		"chain",
		"direct_chain",
		fmt.Sprintf("%d", agentID),
	).Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list enabled chain rules by agent ID", "agent_id", agentID, "error", err)
		return nil, fmt.Errorf("failed to list enabled chain rules: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map chain rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map chain rules: %w", err)
	}

	return entities, nil
}

// ListByUserID returns forward rules for a specific user with filtering and pagination.
func (r *ForwardRuleRepositoryImpl) ListByUserID(ctx context.Context, userID uint, filter forward.ListFilter) ([]*forward.ForwardRule, int64, error) {
	tx := db.GetTxFromContext(ctx, r.db)
	query := tx.Model(&models.ForwardRuleModel{}).Where("user_id = ?", userID)

	// Apply additional filters
	if filter.AgentID != 0 {
		query = query.Where("agent_id = ?", filter.AgentID)
	}
	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Protocol != "" {
		query = query.Where("protocol = ?", filter.Protocol)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count forward rules by user ID", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to count forward rules by user ID: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	orderBy := strings.ToLower(filter.OrderBy)
	if orderBy == "" || !allowedRuleOrderByFields[orderBy] {
		// Default: sort by sort_order ASC, then created_at DESC
		query = query.Order("sort_order ASC, created_at DESC")
	} else {
		order := strings.ToUpper(filter.Order)
		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", orderBy, order))
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Execute query
	var ruleModels []*models.ForwardRuleModel
	if err := query.Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list forward rules by user ID", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to list forward rules by user ID: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, total, nil
}

// ListBySubscriptionID returns all forward rules for a specific subscription (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) ListBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.
		Scopes(db.NotDeleted()).
		Where("subscription_id = ?", subscriptionID).
		Order("sort_order ASC, created_at DESC").
		Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list forward rules by subscription ID", "subscription_id", subscriptionID, "error", err)
		return nil, fmt.Errorf("failed to list forward rules by subscription ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListSystemRulesByTargetNodes returns enabled system rules targeting the specified nodes.
// Only includes rules with system scope (user_id IS NULL or 0).
// If groupIDs is not empty, only returns rules that belong to at least one of the specified resource groups.
// This is used for Node Plan subscription delivery where user rules should be excluded.
func (r *ForwardRuleRepositoryImpl) ListSystemRulesByTargetNodes(ctx context.Context, nodeIDs []uint, groupIDs []uint) ([]*forward.ForwardRule, error) {
	if len(nodeIDs) == 0 {
		return []*forward.ForwardRule{}, nil
	}

	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	// Query enabled system rules (user_id IS NULL or 0) targeting the specified nodes
	// This encapsulates the isolation logic that was previously scattered in SQL queries
	query := tx.
		Where("target_node_id IN ?", nodeIDs).
		Where("status = ?", "enabled").
		Where("rule_type IN ?", []string{"direct", "entry", "chain", "direct_chain", "external"}).
		Where("user_id IS NULL OR user_id = 0")

	// If groupIDs is specified, filter by resource group membership
	if len(groupIDs) > 0 {
		groupIDsJSON := jsonutil.UintSliceToJSONArray(groupIDs)
		query = query.Where("JSON_OVERLAPS(group_ids, ?)", groupIDsJSON)
	}

	if err := query.Order("sort_order ASC").Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list system rules by target nodes", "node_count", len(nodeIDs), "group_count", len(groupIDs), "error", err)
		return nil, fmt.Errorf("failed to list system rules by target nodes: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListSystemRulesByGroupIDs returns enabled system rules that belong to any of the specified groups.
// Unlike ListSystemRulesByTargetNodes, this does not require target nodes to be in the same resource groups.
// This allows rules to be delivered even when their target nodes are outside the resource groups.
// Only includes rules with system scope (user_id IS NULL or 0) and target_node_id set.
func (r *ForwardRuleRepositoryImpl) ListSystemRulesByGroupIDs(ctx context.Context, groupIDs []uint) ([]*forward.ForwardRule, error) {
	if len(groupIDs) == 0 {
		return []*forward.ForwardRule{}, nil
	}

	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	groupIDsJSON := jsonutil.UintSliceToJSONArray(groupIDs)

	// Query enabled system rules (user_id IS NULL or 0) that belong to the specified groups
	// Requires target_node_id to be set since rules without target nodes cannot generate subscription entries
	if err := tx.
		Where("status = ?", "enabled").
		Where("rule_type IN ?", []string{"direct", "entry", "chain", "direct_chain", "external"}).
		Where("user_id IS NULL OR user_id = 0").
		Where("target_node_id IS NOT NULL").
		Where("JSON_OVERLAPS(group_ids, ?)", groupIDsJSON).
		Order("sort_order ASC").
		Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list system rules by group IDs", "group_count", len(groupIDs), "error", err)
		return nil, fmt.Errorf("failed to list system rules by group IDs: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListUserRulesForDelivery returns enabled user rules for subscription delivery.
// Only includes rules with user scope (user_id = userID) and target_node_id set.
// This is used for Forward Plan subscription delivery.
func (r *ForwardRuleRepositoryImpl) ListUserRulesForDelivery(ctx context.Context, userID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	// Query enabled user rules with target_node_id set
	// This encapsulates the isolation logic for user-specific rule delivery
	// Includes external rules which also require target_node_id for protocol information
	if err := tx.
		Where("user_id = ?", userID).
		Where("status = ?", "enabled").
		Where("target_node_id IS NOT NULL").
		Where("rule_type IN ?", []string{"direct", "entry", "chain", "direct_chain", "external"}).
		Order("sort_order ASC").
		Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list user rules for delivery", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to list user rules for delivery: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListEnabledByTargetNodeID returns all enabled forward rules targeting a specific node.
// This is used for notifying agents when a node's address changes.
func (r *ForwardRuleRepositoryImpl) ListEnabledByTargetNodeID(ctx context.Context, nodeID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.
		Where("target_node_id = ?", nodeID).
		Where("status = ?", "enabled").
		Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list enabled rules by target node ID", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to list enabled rules by target node ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}

// ListByGroupID returns all forward rules that belong to the specified resource group.
// Uses JSON_CONTAINS to check if group_ids array contains the given group ID.
// Supports pagination when page > 0 and pageSize > 0.
func (r *ForwardRuleRepositoryImpl) ListByGroupID(ctx context.Context, groupID uint, page, pageSize int) ([]*forward.ForwardRule, int64, error) {
	tx := db.GetTxFromContext(ctx, r.db)

	// Build base query using CAST(? AS JSON) for proper numeric comparison
	baseQuery := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("JSON_CONTAINS(group_ids, CAST(? AS JSON))", groupID)

	// Count total records
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count forward rules by group ID", "group_id", groupID, "error", err)
		return nil, 0, fmt.Errorf("failed to count forward rules by group ID: %w", err)
	}

	// Build paginated query with same sorting as List: sort_order ASC, created_at DESC
	var ruleModels []*models.ForwardRuleModel
	query := tx.
		Scopes(db.NotDeleted()).
		Where("JSON_CONTAINS(group_ids, CAST(? AS JSON))", groupID).
		Order("sort_order ASC, created_at DESC")

	// Apply pagination if specified
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	if err := query.Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list forward rules by group ID", "group_id", groupID, "error", err)
		return nil, 0, fmt.Errorf("failed to list forward rules by group ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, total, nil
}

// ListByExternalSource returns all forward rules with the given external source.
func (r *ForwardRuleRepositoryImpl) ListByExternalSource(ctx context.Context, source string) ([]*forward.ForwardRule, error) {
	if source == "" {
		return []*forward.ForwardRule{}, nil
	}

	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("external_source = ?", source).
		Order("sort_order ASC, created_at DESC").
		Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to list forward rules by external source", "source", source, "error", err)
		return nil, fmt.Errorf("failed to list forward rules by external source: %w", err)
	}

	entities, err := r.mapper.ToEntities(ruleModels)
	if err != nil {
		r.logger.Errorw("failed to map forward rule models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward rules: %w", err)
	}

	return entities, nil
}
