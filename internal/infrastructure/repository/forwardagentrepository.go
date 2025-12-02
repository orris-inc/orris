package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"orris/internal/domain/forward"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

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
		Updates(map[string]interface{}{
			"name":       model.Name,
			"token_hash": model.TokenHash,
			"status":     model.Status,
			"remark":     model.Remark,
			"updated_at": model.UpdatedAt,
		})

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "duplicate key") {
			return errors.NewConflictError("forward agent already exists")
		}
		r.logger.Errorw("failed to update forward agent", "id", model.ID, "error", result.Error)
		return fmt.Errorf("failed to update forward agent: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", model.ID))
	}

	r.logger.Infow("forward agent updated successfully", "id", model.ID, "name", model.Name)
	return nil
}

// Delete soft deletes a forward agent.
func (r *ForwardAgentRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.ForwardAgentModel{}, id)
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

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count forward agents", "error", err)
		return nil, 0, fmt.Errorf("failed to count forward agents: %w", err)
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
	err := r.db.WithContext(ctx).Model(&models.ForwardAgentModel{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check forward agent existence by name", "name", name, "error", err)
		return false, fmt.Errorf("failed to check forward agent existence: %w", err)
	}
	return count > 0, nil
}
