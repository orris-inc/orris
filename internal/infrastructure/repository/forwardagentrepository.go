package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// allowedAgentOrderByFields defines the whitelist of allowed ORDER BY fields
// to prevent SQL injection attacks.
var allowedAgentOrderByFields = map[string]bool{
	"id":           true,
	"sid":          true,
	"name":         true,
	"status":       true,
	"sort_order":   true,
	"last_seen_at": true,
	"created_at":   true,
	"updated_at":   true,
}

// ForwardAgentRepositoryImpl implements the forward.AgentRepository interface.
type ForwardAgentRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.ForwardAgentMapper
	logger logger.Interface
}

// NewForwardAgentRepository creates a new forward agent repository instance.
func NewForwardAgentRepository(db *gorm.DB, logger logger.Interface) forward.AgentRepository {
	return &ForwardAgentRepositoryImpl{
		db:     db,
		mapper: mappers.NewForwardAgentMapper(),
		logger: logger,
	}
}

// Create creates a new forward agent in the database.
func (r *ForwardAgentRepositoryImpl) Create(ctx context.Context, agent *forward.ForwardAgent) error {
	model, err := r.mapper.ToModel(agent)
	if err != nil {
		r.logger.Errorw("failed to map forward agent entity to model", "error", err)
		return fmt.Errorf("failed to map forward agent entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
			return errors.NewConflictError("forward agent already exists")
		}
		r.logger.Errorw("failed to create forward agent in database", "error", err)
		return fmt.Errorf("failed to create forward agent: %w", err)
	}

	if err := agent.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set forward agent ID", "error", err)
		return fmt.Errorf("failed to set forward agent ID: %w", err)
	}

	r.logger.Infow("forward agent created successfully", "id", model.ID, "name", model.Name)
	return nil
}

// GetByID retrieves a forward agent by its ID.
func (r *ForwardAgentRepositoryImpl) GetByID(ctx context.Context, id uint) (*forward.ForwardAgent, error) {
	var model models.ForwardAgentModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get forward agent by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map forward agent model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map forward agent: %w", err)
	}

	return entity, nil
}

// GetBySID retrieves a forward agent by its SID.
func (r *ForwardAgentRepositoryImpl) GetBySID(ctx context.Context, sid string) (*forward.ForwardAgent, error) {
	var model models.ForwardAgentModel

	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get forward agent by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map forward agent model to entity", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to map forward agent: %w", err)
	}

	return entity, nil
}

// GetBySIDs retrieves multiple forward agents by their SIDs.
func (r *ForwardAgentRepositoryImpl) GetBySIDs(ctx context.Context, sids []string) ([]*forward.ForwardAgent, error) {
	if len(sids) == 0 {
		return []*forward.ForwardAgent{}, nil
	}

	var agentModels []*models.ForwardAgentModel
	if err := r.db.WithContext(ctx).Where("sid IN ?", sids).Find(&agentModels).Error; err != nil {
		r.logger.Errorw("failed to get forward agents by SIDs", "sids", sids, "error", err)
		return nil, fmt.Errorf("failed to get forward agents: %w", err)
	}

	entities, err := r.mapper.ToEntities(agentModels)
	if err != nil {
		r.logger.Errorw("failed to map forward agent models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward agents: %w", err)
	}

	return entities, nil
}

// GetSIDsByIDs retrieves SIDs for multiple agents by their internal IDs.
func (r *ForwardAgentRepositoryImpl) GetSIDsByIDs(ctx context.Context, ids []uint) (map[uint]string, error) {
	if len(ids) == 0 {
		return make(map[uint]string), nil
	}

	var results []struct {
		ID  uint   `gorm:"column:id"`
		SID string `gorm:"column:sid"`
	}

	if err := r.db.WithContext(ctx).
		Model(&models.ForwardAgentModel{}).
		Select("id, sid").
		Where("id IN ?", ids).
		Find(&results).Error; err != nil {
		r.logger.Errorw("failed to get forward agent SIDs", "ids", ids, "error", err)
		return nil, fmt.Errorf("failed to get forward agent SIDs: %w", err)
	}

	sidMap := make(map[uint]string, len(results))
	for _, r := range results {
		sidMap[r.ID] = r.SID
	}

	return sidMap, nil
}

// GetByIDs retrieves multiple forward agents by their internal IDs.
func (r *ForwardAgentRepositoryImpl) GetByIDs(ctx context.Context, ids []uint) (map[uint]*forward.ForwardAgent, error) {
	if len(ids) == 0 {
		return make(map[uint]*forward.ForwardAgent), nil
	}

	var agentModels []*models.ForwardAgentModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&agentModels).Error; err != nil {
		r.logger.Errorw("failed to get forward agents by IDs", "ids", ids, "error", err)
		return nil, fmt.Errorf("failed to get forward agents: %w", err)
	}

	result := make(map[uint]*forward.ForwardAgent, len(agentModels))
	for _, model := range agentModels {
		entity, err := r.mapper.ToEntity(model)
		if err != nil {
			r.logger.Warnw("failed to map forward agent model to entity", "id", model.ID, "error", err)
			continue
		}
		result[model.ID] = entity
	}

	return result, nil
}

// GetByTokenHash retrieves a forward agent by token hash.
func (r *ForwardAgentRepositoryImpl) GetByTokenHash(ctx context.Context, tokenHash string) (*forward.ForwardAgent, error) {
	var model models.ForwardAgentModel

	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get forward agent by token hash", "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map forward agent model to entity", "error", err)
		return nil, fmt.Errorf("failed to map forward agent: %w", err)
	}

	return entity, nil
}

// Update updates an existing forward agent.
func (r *ForwardAgentRepositoryImpl) Update(ctx context.Context, agent *forward.ForwardAgent) error {
	model, err := r.mapper.ToModel(agent)
	if err != nil {
		r.logger.Errorw("failed to map forward agent entity to model", "error", err)
		return fmt.Errorf("failed to map forward agent entity: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&models.ForwardAgentModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"name":               model.Name,
			"token_hash":         model.TokenHash,
			"api_token":          model.APIToken,
			"status":             model.Status,
			"public_address":     model.PublicAddress,
			"tunnel_address":     model.TunnelAddress,
			"remark":             model.Remark,
			"group_ids":          model.GroupIDs,
			"allowed_port_range": model.AllowedPortRange,
			"blocked_protocols":  model.BlockedProtocols,
			"sort_order":         model.SortOrder,
			"mute_notification":  model.MuteNotification,
			"expires_at":         model.ExpiresAt,
			"cost_label":         model.CostLabel,
			"updated_at":         model.UpdatedAt,
		})

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "duplicate key") {
			return errors.NewConflictError("forward agent already exists")
		}
		r.logger.Errorw("failed to update forward agent", "id", model.ID, "error", result.Error)
		return fmt.Errorf("failed to update forward agent: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.

	r.logger.Infow("forward agent updated successfully", "id", model.ID, "name", model.Name)
	return nil
}

// Delete permanently deletes a forward agent from the database.
func (r *ForwardAgentRepositoryImpl) Delete(ctx context.Context, id uint) error {
	// Use Unscoped() to perform hard delete instead of soft delete
	result := r.db.WithContext(ctx).Unscoped().Delete(&models.ForwardAgentModel{}, id)

	if result.Error != nil {
		r.logger.Errorw("failed to delete forward agent", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete forward agent: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", id))
	}

	r.logger.Infow("forward agent deleted successfully", "id", id)
	return nil
}

// List retrieves a paginated list of forward agents with filtering.
func (r *ForwardAgentRepositoryImpl) List(ctx context.Context, filter forward.AgentListFilter) ([]*forward.ForwardAgent, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ForwardAgentModel{})

	// Apply filters
	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if len(filter.GroupIDs) > 0 {
		// Use JSON_OVERLAPS to check if group_ids array contains any of the filter group IDs
		groupIDsJSON, _ := json.Marshal(filter.GroupIDs)
		query = query.Where("JSON_OVERLAPS(group_ids, ?)", string(groupIDsJSON))
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count forward agents", "error", err)
		return nil, 0, fmt.Errorf("failed to count forward agents: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	orderBy := strings.ToLower(filter.OrderBy)
	if orderBy == "" || !allowedAgentOrderByFields[orderBy] {
		orderBy = "created_at"
	}
	order := strings.ToUpper(filter.Order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

	// Apply pagination (only when PageSize > 0)
	if filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Execute query
	var agentModels []*models.ForwardAgentModel
	if err := query.Find(&agentModels).Error; err != nil {
		r.logger.Errorw("failed to list forward agents", "error", err)
		return nil, 0, fmt.Errorf("failed to list forward agents: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(agentModels)
	if err != nil {
		r.logger.Errorw("failed to map forward agent models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map forward agents: %w", err)
	}

	return entities, total, nil
}

// ListEnabled returns all enabled forward agents.
func (r *ForwardAgentRepositoryImpl) ListEnabled(ctx context.Context) ([]*forward.ForwardAgent, error) {
	var agentModels []*models.ForwardAgentModel

	if err := r.db.WithContext(ctx).Where("status = ?", "enabled").Find(&agentModels).Error; err != nil {
		r.logger.Errorw("failed to list enabled forward agents", "error", err)
		return nil, fmt.Errorf("failed to list enabled forward agents: %w", err)
	}

	entities, err := r.mapper.ToEntities(agentModels)
	if err != nil {
		r.logger.Errorw("failed to map forward agent models to entities", "error", err)
		return nil, fmt.Errorf("failed to map forward agents: %w", err)
	}

	return entities, nil
}

// ExistsByName checks if an agent with the given name exists.
func (r *ForwardAgentRepositoryImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ForwardAgentModel{}).
		Where("name = ?", name).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check forward agent existence by name", "name", name, "error", err)
		return false, fmt.Errorf("failed to check forward agent existence: %w", err)
	}
	return count > 0, nil
}

// UpdateLastSeen updates the last_seen_at timestamp and agent info for an agent.
func (r *ForwardAgentRepositoryImpl) UpdateLastSeen(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Model(&models.ForwardAgentModel{}).
		Where("id = ?", id).
		Update("last_seen_at", biztime.NowUTC())

	if result.Error != nil {
		r.logger.Errorw("failed to update forward agent last_seen_at", "id", id, "error", result.Error)
		return fmt.Errorf("failed to update last_seen_at: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", id))
	}

	r.logger.Debugw("forward agent last_seen_at updated", "id", id)
	return nil
}

// UpdateAgentInfo updates the agent info (version, platform, arch) for an agent.
func (r *ForwardAgentRepositoryImpl) UpdateAgentInfo(ctx context.Context, id uint, agentVersion, platform, arch string) error {
	updates := map[string]interface{}{}

	if agentVersion != "" {
		updates["agent_version"] = agentVersion
	}
	if platform != "" {
		updates["platform"] = platform
	}
	if arch != "" {
		updates["arch"] = arch
	}

	if len(updates) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).Model(&models.ForwardAgentModel{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		r.logger.Errorw("failed to update forward agent info", "id", id, "error", result.Error)
		return fmt.Errorf("failed to update agent info: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", id))
	}

	r.logger.Infow("forward agent info updated", "id", id, "version", agentVersion, "platform", platform, "arch", arch)
	return nil
}

// GetAllEnabledMetadata returns lightweight metadata for all enabled agents.
// Only queries id, sid, name fields.
func (r *ForwardAgentRepositoryImpl) GetAllEnabledMetadata(ctx context.Context) ([]*forward.AgentMetadata, error) {
	var results []struct {
		ID   uint   `gorm:"column:id"`
		SID  string `gorm:"column:sid"`
		Name string `gorm:"column:name"`
	}

	if err := r.db.WithContext(ctx).
		Model(&models.ForwardAgentModel{}).
		Select("id, sid, name").
		Where("status = ?", "enabled").
		Find(&results).Error; err != nil {
		r.logger.Errorw("failed to get all enabled agent metadata", "error", err)
		return nil, fmt.Errorf("failed to get agent metadata: %w", err)
	}

	metadata := make([]*forward.AgentMetadata, len(results))
	for i, res := range results {
		metadata[i] = &forward.AgentMetadata{
			ID:   res.ID,
			SID:  res.SID,
			Name: res.Name,
		}
	}

	return metadata, nil
}

// GetMetadataBySIDs returns lightweight metadata for agents by SIDs.
// Only queries id, sid, name fields.
func (r *ForwardAgentRepositoryImpl) GetMetadataBySIDs(ctx context.Context, sids []string) ([]*forward.AgentMetadata, error) {
	if len(sids) == 0 {
		return []*forward.AgentMetadata{}, nil
	}

	var results []struct {
		ID   uint   `gorm:"column:id"`
		SID  string `gorm:"column:sid"`
		Name string `gorm:"column:name"`
	}

	if err := r.db.WithContext(ctx).
		Model(&models.ForwardAgentModel{}).
		Select("id, sid, name").
		Where("sid IN ?", sids).
		Find(&results).Error; err != nil {
		r.logger.Errorw("failed to get agent metadata by SIDs", "sids", sids, "error", err)
		return nil, fmt.Errorf("failed to get agent metadata: %w", err)
	}

	metadata := make([]*forward.AgentMetadata, len(results))
	for i, res := range results {
		metadata[i] = &forward.AgentMetadata{
			ID:   res.ID,
			SID:  res.SID,
			Name: res.Name,
		}
	}

	return metadata, nil
}

// FindExpiringAgents returns enabled agents that will expire within the specified days.
// Only returns agents that have expires_at set and are not already expired.
func (r *ForwardAgentRepositoryImpl) FindExpiringAgents(ctx context.Context, withinDays int) ([]*forward.ExpiringAgentInfo, error) {
	now := biztime.NowUTC()
	threshold := now.AddDate(0, 0, withinDays)

	var results []struct {
		ID        uint       `gorm:"column:id"`
		SID       string     `gorm:"column:sid"`
		Name      string     `gorm:"column:name"`
		ExpiresAt *time.Time `gorm:"column:expires_at"`
		CostLabel *string    `gorm:"column:cost_label"`
	}

	if err := r.db.WithContext(ctx).
		Model(&models.ForwardAgentModel{}).
		Select("id, sid, name, expires_at, cost_label").
		Where("status = ?", "enabled").
		Where("expires_at IS NOT NULL").
		Where("expires_at > ?", now).        // Not already expired
		Where("expires_at <= ?", threshold). // Within threshold
		Order("expires_at ASC").
		Find(&results).Error; err != nil {
		r.logger.Errorw("failed to find expiring agents", "within_days", withinDays, "error", err)
		return nil, fmt.Errorf("failed to find expiring agents: %w", err)
	}

	agents := make([]*forward.ExpiringAgentInfo, 0, len(results))
	for _, res := range results {
		if res.ExpiresAt == nil {
			continue
		}
		agents = append(agents, &forward.ExpiringAgentInfo{
			ID:        res.ID,
			SID:       res.SID,
			Name:      res.Name,
			ExpiresAt: res.ExpiresAt.UTC(),
			CostLabel: res.CostLabel,
		})
	}

	return agents, nil
}
