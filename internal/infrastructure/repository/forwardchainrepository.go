package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"orris/internal/domain/forward"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// ForwardChainRepositoryImpl implements the forward.ChainRepository interface.
type ForwardChainRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.ForwardChainMapper
	logger logger.Interface
}

// NewForwardChainRepository creates a new forward chain repository instance.
func NewForwardChainRepository(db *gorm.DB, logger logger.Interface) forward.ChainRepository {
	return &ForwardChainRepositoryImpl{
		db:     db,
		mapper: mappers.NewForwardChainMapper(),
		logger: logger,
	}
}

// Create creates a new forward chain in the database.
func (r *ForwardChainRepositoryImpl) Create(ctx context.Context, chain *forward.ForwardChain) error {
	model, err := r.mapper.ToModel(chain)
	if err != nil {
		r.logger.Errorw("failed to map forward chain entity to model", "error", err)
		return fmt.Errorf("failed to map forward chain entity: %w", err)
	}

	// Use transaction to create chain and nodes
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create chain first (without nodes)
		chainModel := &models.ForwardChainModel{
			Name:          model.Name,
			Protocol:      model.Protocol,
			Status:        model.Status,
			TargetAddress: model.TargetAddress,
			TargetPort:    model.TargetPort,
			Remark:        model.Remark,
		}

		if err := tx.Create(chainModel).Error; err != nil {
			return fmt.Errorf("failed to create chain: %w", err)
		}

		// Set chain ID for nodes
		for i := range model.Nodes {
			model.Nodes[i].ChainID = chainModel.ID
		}

		// Create nodes
		if len(model.Nodes) > 0 {
			if err := tx.Create(&model.Nodes).Error; err != nil {
				return fmt.Errorf("failed to create chain nodes: %w", err)
			}
		}

		// Set ID back to domain entity
		if err := chain.SetID(chainModel.ID); err != nil {
			return fmt.Errorf("failed to set chain ID: %w", err)
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to create forward chain", "error", err)
		return err
	}

	r.logger.Infow("forward chain created successfully", "id", chain.ID(), "name", chain.Name())
	return nil
}

// GetByID retrieves a forward chain by its ID.
func (r *ForwardChainRepositoryImpl) GetByID(ctx context.Context, id uint) (*forward.ForwardChain, error) {
	var model models.ForwardChainModel

	if err := r.db.WithContext(ctx).Preload("Nodes", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence ASC")
	}).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get forward chain by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get forward chain: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map forward chain model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map forward chain: %w", err)
	}

	return entity, nil
}

// Update updates an existing forward chain.
func (r *ForwardChainRepositoryImpl) Update(ctx context.Context, chain *forward.ForwardChain) error {
	model, err := r.mapper.ToModel(chain)
	if err != nil {
		r.logger.Errorw("failed to map forward chain entity to model", "error", err)
		return fmt.Errorf("failed to map forward chain entity: %w", err)
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update chain
		result := tx.Model(&models.ForwardChainModel{}).
			Where("id = ?", model.ID).
			Updates(map[string]interface{}{
				"name":           model.Name,
				"protocol":       model.Protocol,
				"status":         model.Status,
				"target_address": model.TargetAddress,
				"target_port":    model.TargetPort,
				"remark":         model.Remark,
				"updated_at":     model.UpdatedAt,
			})

		if result.Error != nil {
			return fmt.Errorf("failed to update chain: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return errors.NewNotFoundError("forward chain", fmt.Sprintf("%d", model.ID))
		}

		// Delete old nodes and create new ones
		if err := tx.Where("chain_id = ?", model.ID).Delete(&models.ForwardChainNodeModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete old nodes: %w", err)
		}

		if len(model.Nodes) > 0 {
			for i := range model.Nodes {
				model.Nodes[i].ChainID = model.ID
				model.Nodes[i].ID = 0 // Reset ID for new creation
			}
			if err := tx.Create(&model.Nodes).Error; err != nil {
				return fmt.Errorf("failed to create new nodes: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to update forward chain", "id", model.ID, "error", err)
		return err
	}

	r.logger.Infow("forward chain updated successfully", "id", model.ID, "name", model.Name)
	return nil
}

// Delete soft deletes a forward chain.
func (r *ForwardChainRepositoryImpl) Delete(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete chain rules association
		if err := tx.Where("chain_id = ?", id).Delete(&models.ForwardChainRuleModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete chain rules: %w", err)
		}

		// Delete nodes (hard delete since chain is soft deleted)
		if err := tx.Where("chain_id = ?", id).Delete(&models.ForwardChainNodeModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete chain nodes: %w", err)
		}

		// Soft delete chain
		result := tx.Delete(&models.ForwardChainModel{}, id)
		if result.Error != nil {
			return fmt.Errorf("failed to delete chain: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return errors.NewNotFoundError("forward chain", fmt.Sprintf("%d", id))
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to delete forward chain", "id", id, "error", err)
		return err
	}

	r.logger.Infow("forward chain deleted successfully", "id", id)
	return nil
}

// List retrieves a paginated list of forward chains with filtering.
func (r *ForwardChainRepositoryImpl) List(ctx context.Context, filter forward.ChainListFilter) ([]*forward.ForwardChain, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ForwardChainModel{})

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
		r.logger.Errorw("failed to count forward chains", "error", err)
		return nil, 0, fmt.Errorf("failed to count forward chains: %w", err)
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

	// Execute query with preloaded nodes
	var chainModels []*models.ForwardChainModel
	if err := query.Preload("Nodes", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence ASC")
	}).Find(&chainModels).Error; err != nil {
		r.logger.Errorw("failed to list forward chains", "error", err)
		return nil, 0, fmt.Errorf("failed to list forward chains: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(chainModels)
	if err != nil {
		r.logger.Errorw("failed to map forward chain models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map forward chains: %w", err)
	}

	return entities, total, nil
}

// GetRuleIDsByChainID returns all rule IDs associated with a chain.
func (r *ForwardChainRepositoryImpl) GetRuleIDsByChainID(ctx context.Context, chainID uint) ([]uint, error) {
	var ruleAssocs []models.ForwardChainRuleModel

	if err := r.db.WithContext(ctx).Where("chain_id = ?", chainID).Find(&ruleAssocs).Error; err != nil {
		r.logger.Errorw("failed to get rule IDs by chain ID", "chain_id", chainID, "error", err)
		return nil, fmt.Errorf("failed to get rule IDs: %w", err)
	}

	ruleIDs := make([]uint, len(ruleAssocs))
	for i, assoc := range ruleAssocs {
		ruleIDs[i] = assoc.RuleID
	}

	return ruleIDs, nil
}

// AssociateRules associates rules with a chain.
func (r *ForwardChainRepositoryImpl) AssociateRules(ctx context.Context, chainID uint, ruleIDs []uint) error {
	if len(ruleIDs) == 0 {
		return nil
	}

	assocs := make([]models.ForwardChainRuleModel, len(ruleIDs))
	for i, ruleID := range ruleIDs {
		assocs[i] = models.ForwardChainRuleModel{
			ChainID: chainID,
			RuleID:  ruleID,
		}
	}

	if err := r.db.WithContext(ctx).Create(&assocs).Error; err != nil {
		r.logger.Errorw("failed to associate rules with chain", "chain_id", chainID, "error", err)
		return fmt.Errorf("failed to associate rules: %w", err)
	}

	return nil
}
