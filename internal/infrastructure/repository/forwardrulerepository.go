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

// GetByShortID retrieves a forward rule by its short ID.
func (r *ForwardRuleRepositoryImpl) GetByShortID(ctx context.Context, shortID string) (*forward.ForwardRule, error) {
	var model models.ForwardRuleModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Where("short_id = ?", shortID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get forward rule by short ID", "short_id", shortID, "error", err)
		return nil, fmt.Errorf("failed to get forward rule: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map forward rule model to entity", "short_id", shortID, "error", err)
		return nil, fmt.Errorf("failed to map forward rule: %w", err)
	}

	return entity, nil
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
		Updates(map[string]interface{}{
			"name":           model.Name,
			"listen_port":    model.ListenPort,
			"target_address": model.TargetAddress,
			"target_port":    model.TargetPort,
			"target_node_id": model.TargetNodeID,
			"protocol":       model.Protocol,
			"status":         model.Status,
			"remark":         model.Remark,
			"upload_bytes":   model.UploadBytes,
			"download_bytes": model.DownloadBytes,
			"rule_type":      model.RuleType,
			"exit_agent_id":  model.ExitAgentID,
			"ws_listen_port": model.WsListenPort,
			"updated_at":     model.UpdatedAt,
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

// Delete soft deletes a forward rule.
func (r *ForwardRuleRepositoryImpl) Delete(ctx context.Context, id uint) error {
	tx := db.GetTxFromContext(ctx, r.db)
	result := tx.Delete(&models.ForwardRuleModel{}, id)
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

	// Apply sorting
	orderBy := filter.OrderBy
	order := filter.Order
	if orderBy == "" {
		orderBy = "created_at"
	}
	if order == "" {
		order = "desc"
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

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

// ExistsByListenPort checks if a rule with the given listen port exists.
func (r *ForwardRuleRepositoryImpl) ExistsByListenPort(ctx context.Context, port uint16) (bool, error) {
	var count int64
	tx := db.GetTxFromContext(ctx, r.db)
	err := tx.Model(&models.ForwardRuleModel{}).Where("listen_port = ?", port).Count(&count).Error
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
		Updates(map[string]interface{}{
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

// GetExitRuleByAgentID is deprecated (exit type has been removed).
func (r *ForwardRuleRepositoryImpl) GetExitRuleByAgentID(ctx context.Context, agentID uint) (*forward.ForwardRule, error) {
	r.logger.Warnw("GetExitRuleByAgentID called but exit type has been removed", "agent_id", agentID)
	return nil, nil
}
