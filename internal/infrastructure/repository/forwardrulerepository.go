package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

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
	"upload_bytes":   true,
	"download_bytes": true,
	"sort_order":     true,
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
			"chain_agent_ids":    model.ChainAgentIDs,
			"chain_port_config":  model.ChainPortConfig,
			"tunnel_type":        model.TunnelType,
			"tunnel_hops":        model.TunnelHops,
			"traffic_multiplier": model.TrafficMultiplier,
			"sort_order":         model.SortOrder,
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
			"deleted_at": gorm.Expr("NOW()"),
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
func (r *ForwardRuleRepositoryImpl) ListEnabledByExitAgentID(ctx context.Context, exitAgentID uint) ([]*forward.ForwardRule, error) {
	var ruleModels []*models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("exit_agent_id = ? AND status = ? AND rule_type = ?", exitAgentID, "enabled", "entry").Find(&ruleModels).Error; err != nil {
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

// UpdateSortOrders batch updates sort_order for multiple rules.
func (r *ForwardRuleRepositoryImpl) UpdateSortOrders(ctx context.Context, ruleOrders map[uint]int) error {
	if len(ruleOrders) == 0 {
		return nil
	}

	tx := db.GetTxFromContext(ctx, r.db)
	for id, sortOrder := range ruleOrders {
		result := tx.Model(&models.ForwardRuleModel{}).
			Where("id = ?", id).
			Update("sort_order", sortOrder)

		if result.Error != nil {
			r.logger.Errorw("failed to update sort order", "id", id, "sort_order", sortOrder, "error", result.Error)
			return fmt.Errorf("failed to update sort order for rule %d: %w", id, result.Error)
		}

		if result.RowsAffected == 0 {
			r.logger.Warnw("rule not found when updating sort order", "id", id)
		}
	}

	r.logger.Infow("sort orders updated successfully", "count", len(ruleOrders))
	return nil
}
