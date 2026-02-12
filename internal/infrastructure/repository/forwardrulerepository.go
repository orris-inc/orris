package repository

import (
	"context"
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
