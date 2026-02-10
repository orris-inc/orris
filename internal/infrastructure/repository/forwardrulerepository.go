package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils/jsonutil"
)

// Compile-time interface assertion.
var _ forward.Repository = (*ForwardRuleRepositoryImpl)(nil)

// allowedRuleOrderByFields defines the whitelist of allowed ORDER BY fields
// to prevent SQL injection attacks.
var allowedRuleOrderByFields = map[string]bool{
	"id":             true,
	"sid":            true,
	"agent_id":       true,
	"user_id":        true,
	"rule_type":      true,
	"name":           true,
	"listen_port":    true,
	"target_port":    true,
	"protocol":       true,
	"status":         true,
	"sort_order":     true,
	"upload_bytes":   true,
	"download_bytes": true,
	"created_at":     true,
	"updated_at":     true,
}

// ForwardRuleRepositoryImpl implements the forward.Repository interface.
type ForwardRuleRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.ForwardRuleMapper
	logger logger.Interface
}

// NewForwardRuleRepository creates a new forward rule repository instance.
func NewForwardRuleRepository(db *gorm.DB, logger logger.Interface) forward.Repository {
	return &ForwardRuleRepositoryImpl{
		db:     db,
		mapper: mappers.NewForwardRuleMapper(),
		logger: logger,
	}
}

// Create creates a new forward rule in the database.
func (r *ForwardRuleRepositoryImpl) Create(ctx context.Context, rule *forward.ForwardRule) error {
	model, err := r.mapper.ToModel(rule)
	if err != nil {
		r.logger.Errorw("failed to map forward rule entity to model", "error", err)
		return fmt.Errorf("failed to map forward rule entity: %w", err)
	}

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Create(model).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "listen_port") {
				return errors.NewConflictError("listen port is already in use")
			}
			return errors.NewConflictError("forward rule already exists")
		}
		r.logger.Errorw("failed to create forward rule in database", "error", err)
		return fmt.Errorf("failed to create forward rule: %w", err)
	}

	if err := rule.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set forward rule ID", "error", err)
		return fmt.Errorf("failed to set forward rule ID: %w", err)
	}

	r.logger.Infow("forward rule created successfully", "id", model.ID, "name", model.Name)
	return nil
}

// GetByID retrieves a forward rule by its ID.
func (r *ForwardRuleRepositoryImpl) GetByID(ctx context.Context, id uint) (*forward.ForwardRule, error) {
	var model models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get forward rule by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get forward rule: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map forward rule model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map forward rule: %w", err)
	}

	return entity, nil
}

// GetBySID retrieves a forward rule by its SID.
func (r *ForwardRuleRepositoryImpl) GetBySID(ctx context.Context, sid string) (*forward.ForwardRule, error) {
	var model models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get forward rule by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get forward rule: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map forward rule model to entity", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to map forward rule: %w", err)
	}

	return entity, nil
}

// GetBySIDs retrieves multiple forward rules by their SIDs.
func (r *ForwardRuleRepositoryImpl) GetBySIDs(ctx context.Context, sids []string) (map[string]*forward.ForwardRule, error) {
	if len(sids) == 0 {
		return make(map[string]*forward.ForwardRule), nil
	}

	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("sid IN ?", sids).Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to get forward rules by SIDs", "count", len(sids), "error", err)
		return nil, fmt.Errorf("failed to get forward rules by SIDs: %w", err)
	}

	result := make(map[string]*forward.ForwardRule, len(ruleModels))
	for _, model := range ruleModels {
		entity, err := r.mapper.ToEntity(model)
		if err != nil {
			r.logger.Errorw("failed to map forward rule model to entity", "sid", model.SID, "error", err)
			return nil, fmt.Errorf("failed to map forward rule: %w", err)
		}
		result[model.SID] = entity
	}

	return result, nil
}

// GetByIDs retrieves multiple forward rules by their internal IDs.
func (r *ForwardRuleRepositoryImpl) GetByIDs(ctx context.Context, ids []uint) (map[uint]*forward.ForwardRule, error) {
	if len(ids) == 0 {
		return make(map[uint]*forward.ForwardRule), nil
	}

	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("id IN ?", ids).Find(&ruleModels).Error; err != nil {
		r.logger.Errorw("failed to get forward rules by IDs", "count", len(ids), "error", err)
		return nil, fmt.Errorf("failed to get forward rules by IDs: %w", err)
	}

	result := make(map[uint]*forward.ForwardRule, len(ruleModels))
	for _, model := range ruleModels {
		entity, err := r.mapper.ToEntity(model)
		if err != nil {
			r.logger.Errorw("failed to map forward rule model to entity", "id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to map forward rule: %w", err)
		}
		result[model.ID] = entity
	}

	return result, nil
}

// GetByListenPort retrieves a forward rule by listen port.
func (r *ForwardRuleRepositoryImpl) GetByListenPort(ctx context.Context, port uint16) (*forward.ForwardRule, error) {
	var model models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("listen_port = ?", port).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get forward rule by listen port", "port", port, "error", err)
		return nil, fmt.Errorf("failed to get forward rule: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map forward rule model to entity", "port", port, "error", err)
		return nil, fmt.Errorf("failed to map forward rule: %w", err)
	}

	return entity, nil
}

// Update updates an existing forward rule.
func (r *ForwardRuleRepositoryImpl) Update(ctx context.Context, rule *forward.ForwardRule) error {
	model, err := r.mapper.ToModel(rule)
	if err != nil {
		r.logger.Errorw("failed to map forward rule entity to model", "error", err)
		return fmt.Errorf("failed to map forward rule entity: %w", err)
	}

	tx := db.GetTxFromContext(ctx, r.db)
	result := tx.Model(&models.ForwardRuleModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"name":               model.Name,
			"agent_id":           model.AgentID,
			"subscription_id":    model.SubscriptionID,
			"listen_port":        model.ListenPort,
			"target_address":     model.TargetAddress,
			"target_port":        model.TargetPort,
			"target_node_id":     model.TargetNodeID,
			"bind_ip":            model.BindIP,
			"ip_version":         model.IPVersion,
			"protocol":           model.Protocol,
			"status":             model.Status,
			"remark":             model.Remark,
			"upload_bytes":       model.UploadBytes,
			"download_bytes":     model.DownloadBytes,
			"rule_type":          model.RuleType,
			"exit_agent_id":      model.ExitAgentID,
			"exit_agents":        model.ExitAgents,
			"chain_agent_ids":    model.ChainAgentIDs,
			"chain_port_config":  model.ChainPortConfig,
			"tunnel_type":        model.TunnelType,
			"tunnel_hops":        model.TunnelHops,
			"traffic_multiplier": model.TrafficMultiplier,
			"sort_order":         model.SortOrder,
			"group_ids":          model.GroupIDs,
			"updated_at":         model.UpdatedAt,
		})

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "duplicate key") {
			if strings.Contains(result.Error.Error(), "listen_port") {
				return errors.NewConflictError("listen port is already in use")
			}
			return errors.NewConflictError("forward rule already exists")
		}
		r.logger.Errorw("failed to update forward rule", "id", model.ID, "error", result.Error)
		return fmt.Errorf("failed to update forward rule: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.

	r.logger.Infow("forward rule updated successfully", "id", model.ID, "name", model.Name)
	return nil
}

// Delete soft deletes a forward rule and sets status to disabled.
func (r *ForwardRuleRepositoryImpl) Delete(ctx context.Context, id uint) error {
	tx := db.GetTxFromContext(ctx, r.db)
	// Set status to disabled before soft delete for defensive programming
	result := tx.Model(&models.ForwardRuleModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"status":     "disabled",
			"deleted_at": biztime.NowUTC(),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to delete forward rule", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete forward rule: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", id))
	}

	r.logger.Infow("forward rule deleted successfully", "id", id)
	return nil
}

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

// ExistsByListenPort checks if a rule with the given listen port exists (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) ExistsByListenPort(ctx context.Context, port uint16) (bool, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("listen_port = ?", port).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check forward rule existence by listen port", "port", port, "error", err)
		return false, fmt.Errorf("failed to check forward rule existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAgentIDAndListenPort checks if a rule with the given agent ID and listen port exists.
// This is used for auto-assigning ports within an agent's scope.
func (r *ForwardRuleRepositoryImpl) ExistsByAgentIDAndListenPort(ctx context.Context, agentID uint, port uint16) (bool, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("agent_id = ? AND listen_port = ?", agentID, port).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check forward rule existence by agent and port", "agent_id", agentID, "port", port, "error", err)
		return false, fmt.Errorf("failed to check forward rule existence: %w", err)
	}
	return count > 0, nil
}

// IsPortInUseByAgent checks if a port is in use by the specified agent across all rules.
// This includes both main rule ports (agent_id + listen_port) and chain_port_config entries.
func (r *ForwardRuleRepositoryImpl) IsPortInUseByAgent(ctx context.Context, agentID uint, port uint16, excludeRuleID uint) (bool, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)

	// Build query to check both:
	// 1. Main rule: agent_id = ? AND listen_port = ?
	// 2. Chain port config: JSON_EXTRACT(chain_port_config, '$."<agent_id>"') = port
	// Note: MySQL JSON keys are strings, so we use the agent ID as a string key
	query := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted())

	// Exclude specific rule if provided (for update scenarios)
	if excludeRuleID > 0 {
		query = query.Where("id != ?", excludeRuleID)
	}

	// Check main rule ports OR chain_port_config entries
	// Use CAST for explicit type conversion to ensure correct comparison
	// JSON_EXTRACT returns JSON value, CAST converts it to unsigned integer for comparison
	err := query.Where(
		"(agent_id = ? AND listen_port = ?) OR (chain_port_config IS NOT NULL AND CAST(JSON_EXTRACT(chain_port_config, CONCAT('$.\"', ?, '\"')) AS UNSIGNED) = ?)",
		agentID, port, agentID, port,
	).Count(&count).Error

	if err != nil {
		r.logger.Errorw("failed to check port in use by agent",
			"agent_id", agentID,
			"port", port,
			"exclude_rule_id", excludeRuleID,
			"error", err,
		)
		return false, fmt.Errorf("failed to check port in use: %w", err)
	}

	return count > 0, nil
}

// UpdateTraffic updates the traffic counters for a rule.
func (r *ForwardRuleRepositoryImpl) UpdateTraffic(ctx context.Context, id uint, upload, download int64) error {
	tx := db.GetTxFromContext(ctx, r.db)
	result := tx.Model(&models.ForwardRuleModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"upload_bytes":   gorm.Expr("upload_bytes + ?", upload),
			"download_bytes": gorm.Expr("download_bytes + ?", download),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to update forward rule traffic", "id", id, "error", result.Error)
		return fmt.Errorf("failed to update traffic: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", id))
	}

	return nil
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

// CountByUserID returns the total count of forward rules for a specific user (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to count forward rules by user ID", "user_id", userID, "error", err)
		return 0, fmt.Errorf("failed to count forward rules by user ID: %w", err)
	}
	return count, nil
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

// CountBySubscriptionID returns the total count of forward rules for a specific subscription (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) CountBySubscriptionID(ctx context.Context, subscriptionID uint) (int64, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Where("subscription_id = ?", subscriptionID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to count forward rules by subscription ID", "subscription_id", subscriptionID, "error", err)
		return 0, fmt.Errorf("failed to count forward rules by subscription ID: %w", err)
	}
	return count, nil
}

// GetTotalTrafficByUserID returns the total traffic (upload + download) for all rules owned by a user (excluding soft-deleted records).
func (r *ForwardRuleRepositoryImpl) GetTotalTrafficByUserID(ctx context.Context, userID uint) (int64, error) {
	var result struct {
		TotalTraffic int64
	}

	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).
		Scopes(db.NotDeleted()).
		Select("COALESCE(SUM(upload_bytes + download_bytes), 0) as total_traffic").
		Where("user_id = ?", userID).
		Scan(&result).Error

	if err != nil {
		r.logger.Errorw("failed to get total traffic by user ID", "user_id", userID, "error", err)
		return 0, fmt.Errorf("failed to get total traffic by user ID: %w", err)
	}

	return result.TotalTraffic, nil
}

// UpdateSortOrders batch updates sort_order for multiple rules using a single CASE WHEN SQL.
func (r *ForwardRuleRepositoryImpl) UpdateSortOrders(ctx context.Context, ruleOrders map[uint]int) error {
	if len(ruleOrders) == 0 {
		return nil
	}

	// Build CASE WHEN SQL: UPDATE forward_rules SET sort_order = CASE id WHEN ? THEN ? ... END WHERE id IN (?,...)
	var sb strings.Builder
	sb.WriteString("UPDATE forward_rules SET sort_order = CASE id ")

	args := make([]interface{}, 0, len(ruleOrders)*2+len(ruleOrders))
	ids := make([]interface{}, 0, len(ruleOrders))

	for id, sortOrder := range ruleOrders {
		sb.WriteString("WHEN ? THEN ? ")
		args = append(args, id, sortOrder)
		ids = append(ids, id)
	}

	sb.WriteString("END WHERE id IN (")
	for i := range ids {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("?")
	}
	sb.WriteString(")")
	args = append(args, ids...)

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Exec(sb.String(), args...).Error; err != nil {
		r.logger.Errorw("failed to batch update sort orders", "error", err, "count", len(ruleOrders))
		return fmt.Errorf("failed to batch update sort orders: %w", err)
	}

	r.logger.Infow("sort orders updated successfully", "count", len(ruleOrders))
	return nil
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

// AddGroupIDAtomically adds a group ID to a rule's group_ids array atomically.
// Returns true if the group ID was added, false if it already exists.
// Uses a single UPDATE statement with conditional logic to avoid TOCTOU race conditions.
func (r *ForwardRuleRepositoryImpl) AddGroupIDAtomically(ctx context.Context, ruleID uint, groupID uint) (bool, error) {
	tx := db.GetTxFromContext(ctx, r.db)

	// Single atomic UPDATE that:
	// 1. Only updates if the group ID doesn't already exist (via WHERE clause)
	// 2. Creates new array if NULL, otherwise appends
	// This avoids TOCTOU race conditions by combining check and update in one statement
	updateQuery := `
		UPDATE forward_rules
		SET group_ids = CASE
			WHEN group_ids IS NULL THEN JSON_ARRAY(?)
			ELSE JSON_ARRAY_APPEND(group_ids, '$', CAST(? AS UNSIGNED))
		END,
		updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL
		AND (group_ids IS NULL OR NOT JSON_CONTAINS(group_ids, CAST(? AS JSON)))
	`
	result := tx.Exec(updateQuery, groupID, groupID, ruleID, groupID)
	if result.Error != nil {
		r.logger.Errorw("failed to add group ID to rule atomically", "rule_id", ruleID, "group_id", groupID, "error", result.Error)
		return false, fmt.Errorf("failed to add group ID atomically: %w", result.Error)
	}

	// RowsAffected == 0 means either:
	// 1. Rule not found / deleted
	// 2. Group ID already exists in the array
	// We need to distinguish these cases
	if result.RowsAffected == 0 {
		// Check if rule exists
		var exists bool
		if err := tx.Raw("SELECT EXISTS(SELECT 1 FROM forward_rules WHERE id = ? AND deleted_at IS NULL)", ruleID).Scan(&exists).Error; err != nil {
			return false, fmt.Errorf("failed to check rule existence: %w", err)
		}
		if !exists {
			return false, fmt.Errorf("rule not found or already deleted")
		}
		// Rule exists but group ID already in array
		return false, nil
	}

	return true, nil
}

// RemoveGroupIDAtomically removes a group ID from a rule's group_ids array atomically.
// Returns true if the group ID was removed, false if it was not found.
// Uses JSON_TABLE (MySQL 8.0+) to rebuild the array excluding the target element,
// which correctly handles numeric values unlike JSON_SEARCH.
func (r *ForwardRuleRepositoryImpl) RemoveGroupIDAtomically(ctx context.Context, ruleID uint, groupID uint) (bool, error) {
	tx := db.GetTxFromContext(ctx, r.db)

	// Single atomic UPDATE that rebuilds the array excluding the target group ID
	// JSON_TABLE extracts array elements, we filter out the target and rebuild with JSON_ARRAYAGG
	// The WHERE clause ensures we only update if the group ID exists
	updateQuery := `
		UPDATE forward_rules fr
		SET fr.group_ids = (
			SELECT JSON_ARRAYAGG(jt.gid)
			FROM JSON_TABLE(fr.group_ids, '$[*]' COLUMNS(gid INT PATH '$')) AS jt
			WHERE jt.gid != ?
		),
		fr.updated_at = NOW()
		WHERE fr.id = ? AND fr.deleted_at IS NULL
		AND fr.group_ids IS NOT NULL
		AND JSON_CONTAINS(fr.group_ids, CAST(? AS JSON))
	`
	result := tx.Exec(updateQuery, groupID, ruleID, groupID)
	if result.Error != nil {
		r.logger.Errorw("failed to remove group ID from rule atomically", "rule_id", ruleID, "group_id", groupID, "error", result.Error)
		return false, fmt.Errorf("failed to remove group ID atomically: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// RemoveGroupIDFromAllRules removes a group ID from all rules that contain it.
// This is used when deleting a resource group to clean up orphaned references.
// Uses JSON_TABLE (MySQL 8.0+) to correctly handle numeric array values.
func (r *ForwardRuleRepositoryImpl) RemoveGroupIDFromAllRules(ctx context.Context, groupID uint) (int64, error) {
	tx := db.GetTxFromContext(ctx, r.db)

	// Use a subquery with JSON_TABLE to rebuild arrays excluding the target group ID
	// This correctly handles numeric values in JSON arrays
	updateQuery := `
		UPDATE forward_rules fr
		SET fr.group_ids = (
			SELECT JSON_ARRAYAGG(jt.gid)
			FROM JSON_TABLE(fr.group_ids, '$[*]' COLUMNS(gid INT PATH '$')) AS jt
			WHERE jt.gid != ?
		),
		fr.updated_at = NOW()
		WHERE fr.deleted_at IS NULL
		AND fr.group_ids IS NOT NULL
		AND JSON_CONTAINS(fr.group_ids, CAST(? AS JSON))
	`
	result := tx.Exec(updateQuery, groupID, groupID)
	if result.Error != nil {
		r.logger.Errorw("failed to remove group ID from all rules", "group_id", groupID, "error", result.Error)
		return 0, fmt.Errorf("failed to remove group ID from all rules: %w", result.Error)
	}

	r.logger.Infow("removed group ID from rules", "group_id", groupID, "affected_rows", result.RowsAffected)
	return result.RowsAffected, nil
}

// BatchAddGroupID adds a group ID to multiple rules atomically in a single transaction.
// Returns the number of rules that were updated (excludes rules that already had the group ID).
func (r *ForwardRuleRepositoryImpl) BatchAddGroupID(ctx context.Context, ruleIDs []uint, groupID uint) (int, error) {
	if len(ruleIDs) == 0 {
		return 0, nil
	}

	updated := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Single UPDATE statement that affects all matching rules
		// Uses IN clause for efficiency
		updateQuery := `
			UPDATE forward_rules
			SET group_ids = CASE
				WHEN group_ids IS NULL THEN JSON_ARRAY(?)
				ELSE JSON_ARRAY_APPEND(group_ids, '$', CAST(? AS UNSIGNED))
			END,
			updated_at = NOW()
			WHERE id IN ? AND deleted_at IS NULL
			AND (group_ids IS NULL OR NOT JSON_CONTAINS(group_ids, CAST(? AS JSON)))
		`
		result := tx.Exec(updateQuery, groupID, groupID, ruleIDs, groupID)
		if result.Error != nil {
			return fmt.Errorf("failed to batch add group ID: %w", result.Error)
		}
		updated = int(result.RowsAffected)
		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to batch add group ID to rules", "group_id", groupID, "rule_count", len(ruleIDs), "error", err)
		return 0, err
	}

	r.logger.Infow("batch added group ID to rules", "group_id", groupID, "updated_count", updated, "total_count", len(ruleIDs))
	return updated, nil
}

// BatchRemoveGroupID removes a group ID from multiple rules atomically in a single transaction.
// Returns the number of rules that were updated.
func (r *ForwardRuleRepositoryImpl) BatchRemoveGroupID(ctx context.Context, ruleIDs []uint, groupID uint) (int, error) {
	if len(ruleIDs) == 0 {
		return 0, nil
	}

	updated := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Single UPDATE statement that affects all matching rules
		updateQuery := `
			UPDATE forward_rules fr
			SET fr.group_ids = (
				SELECT JSON_ARRAYAGG(jt.gid)
				FROM JSON_TABLE(fr.group_ids, '$[*]' COLUMNS(gid INT PATH '$')) AS jt
				WHERE jt.gid != ?
			),
			fr.updated_at = NOW()
			WHERE fr.id IN ? AND fr.deleted_at IS NULL
			AND fr.group_ids IS NOT NULL
			AND JSON_CONTAINS(fr.group_ids, CAST(? AS JSON))
		`
		result := tx.Exec(updateQuery, groupID, ruleIDs, groupID)
		if result.Error != nil {
			return fmt.Errorf("failed to batch remove group ID: %w", result.Error)
		}
		updated = int(result.RowsAffected)
		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to batch remove group ID from rules", "group_id", groupID, "rule_count", len(ruleIDs), "error", err)
		return 0, err
	}

	r.logger.Infow("batch removed group ID from rules", "group_id", groupID, "updated_count", updated, "total_count", len(ruleIDs))
	return updated, nil
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
