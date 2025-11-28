package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"orris/internal/domain/subscription"
	vo "orris/internal/domain/subscription/value_objects"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/logger"
)

// PlanPricingRepositoryImpl implements PlanPricingRepository interface
type PlanPricingRepositoryImpl struct {
	db     *gorm.DB
	mapper *mappers.PlanPricingMapper
	logger logger.Interface
}

// NewPlanPricingRepository creates a new PlanPricingRepository
func NewPlanPricingRepository(db *gorm.DB, logger logger.Interface) subscription.PlanPricingRepository {
	return &PlanPricingRepositoryImpl{
		db:     db,
		mapper: mappers.NewPlanPricingMapper(),
		logger: logger,
	}
}

// Create creates a new plan pricing record
func (r *PlanPricingRepositoryImpl) Create(ctx context.Context, pricing *vo.PlanPricing) error {
	model, err := r.mapper.ToModel(pricing)
	if err != nil {
		r.logger.Errorw("failed to map pricing to model", "error", err)
		return fmt.Errorf("failed to map pricing: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create pricing in database",
			"plan_id", model.PlanID,
			"billing_cycle", model.BillingCycle,
			"error", err)
		return fmt.Errorf("failed to create pricing: %w", err)
	}

	r.logger.Infow("pricing created successfully",
		"id", model.ID,
		"plan_id", model.PlanID,
		"billing_cycle", model.BillingCycle,
		"price", model.Price)

	return nil
}

// GetByID retrieves a pricing record by ID
func (r *PlanPricingRepositoryImpl) GetByID(ctx context.Context, id uint) (*vo.PlanPricing, error) {
	var model models.SubscriptionPlanPricingModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get pricing by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}

	pricing, err := r.mapper.ToDomain(&model)
	if err != nil {
		r.logger.Errorw("failed to map model to domain", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map model: %w", err)
	}

	return pricing, nil
}

// GetByPlanAndCycle retrieves pricing for a specific plan and billing cycle
func (r *PlanPricingRepositoryImpl) GetByPlanAndCycle(ctx context.Context, planID uint, cycle vo.BillingCycle) (*vo.PlanPricing, error) {
	var model models.SubscriptionPlanPricingModel

	err := r.db.WithContext(ctx).
		Where("plan_id = ? AND billing_cycle = ? AND is_active = ?", planID, cycle.String(), true).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debugw("pricing not found",
				"plan_id", planID,
				"billing_cycle", cycle.String())
			return nil, nil
		}
		r.logger.Errorw("failed to get pricing by plan and cycle",
			"plan_id", planID,
			"billing_cycle", cycle.String(),
			"error", err)
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}

	pricing, err := r.mapper.ToDomain(&model)
	if err != nil {
		r.logger.Errorw("failed to map model to domain", "error", err)
		return nil, fmt.Errorf("failed to map model: %w", err)
	}

	return pricing, nil
}

// GetByPlanID retrieves all pricing records for a plan
func (r *PlanPricingRepositoryImpl) GetByPlanID(ctx context.Context, planID uint) ([]*vo.PlanPricing, error) {
	var modelList []*models.SubscriptionPlanPricingModel

	err := r.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Order("FIELD(billing_cycle, 'weekly', 'monthly', 'quarterly', 'semi_annual', 'yearly', 'lifetime')").
		Find(&modelList).Error

	if err != nil {
		r.logger.Errorw("failed to get pricings by plan ID", "plan_id", planID, "error", err)
		return nil, fmt.Errorf("failed to get pricings: %w", err)
	}

	pricings, err := r.mapper.ToDomainList(modelList)
	if err != nil {
		r.logger.Errorw("failed to map models to domain", "plan_id", planID, "error", err)
		return nil, fmt.Errorf("failed to map models: %w", err)
	}

	r.logger.Debugw("pricings retrieved", "plan_id", planID, "count", len(pricings))
	return pricings, nil
}

// GetActivePricings retrieves all active pricing records for a plan
func (r *PlanPricingRepositoryImpl) GetActivePricings(ctx context.Context, planID uint) ([]*vo.PlanPricing, error) {
	var modelList []*models.SubscriptionPlanPricingModel

	err := r.db.WithContext(ctx).
		Where("plan_id = ? AND is_active = ?", planID, true).
		Order("FIELD(billing_cycle, 'weekly', 'monthly', 'quarterly', 'semi_annual', 'yearly', 'lifetime')").
		Find(&modelList).Error

	if err != nil {
		r.logger.Errorw("failed to get active pricings", "plan_id", planID, "error", err)
		return nil, fmt.Errorf("failed to get active pricings: %w", err)
	}

	pricings, err := r.mapper.ToDomainList(modelList)
	if err != nil {
		r.logger.Errorw("failed to map models to domain", "error", err)
		return nil, fmt.Errorf("failed to map models: %w", err)
	}

	r.logger.Debugw("active pricings retrieved", "plan_id", planID, "count", len(pricings))
	return pricings, nil
}

// GetActivePricingsByPlanIDs retrieves active pricings for multiple plans in a single query
// This method solves the N+1 query problem when fetching pricings for multiple plans
func (r *PlanPricingRepositoryImpl) GetActivePricingsByPlanIDs(ctx context.Context, planIDs []uint) (map[uint][]*vo.PlanPricing, error) {
	if len(planIDs) == 0 {
		return make(map[uint][]*vo.PlanPricing), nil
	}

	var modelList []*models.SubscriptionPlanPricingModel

	err := r.db.WithContext(ctx).
		Where("plan_id IN ? AND is_active = ?", planIDs, true).
		Order("plan_id ASC, FIELD(billing_cycle, 'weekly', 'monthly', 'quarterly', 'semi_annual', 'yearly', 'lifetime')").
		Find(&modelList).Error

	if err != nil {
		r.logger.Errorw("failed to get active pricings by plan IDs", "plan_count", len(planIDs), "error", err)
		return nil, fmt.Errorf("failed to get active pricings: %w", err)
	}

	// Group pricings by plan ID
	pricingsByPlanID := make(map[uint][]*vo.PlanPricing)

	for _, model := range modelList {
		pricing, err := r.mapper.ToDomain(model)
		if err != nil {
			r.logger.Errorw("failed to map model to domain",
				"plan_id", model.PlanID,
				"pricing_id", model.ID,
				"error", err)
			return nil, fmt.Errorf("failed to map model: %w", err)
		}

		pricingsByPlanID[model.PlanID] = append(pricingsByPlanID[model.PlanID], pricing)
	}

	r.logger.Debugw("active pricings retrieved in batch",
		"plan_count", len(planIDs),
		"total_pricings", len(modelList),
		"plans_with_pricings", len(pricingsByPlanID))

	return pricingsByPlanID, nil
}

// Update updates an existing pricing record
func (r *PlanPricingRepositoryImpl) Update(ctx context.Context, pricing *vo.PlanPricing) error {
	model, err := r.mapper.ToModel(pricing)
	if err != nil {
		r.logger.Errorw("failed to map pricing to model", "error", err)
		return fmt.Errorf("failed to map pricing: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&models.SubscriptionPlanPricingModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]interface{}{
			"price":      model.Price,
			"is_active":  model.IsActive,
			"updated_at": model.UpdatedAt,
		})

	if result.Error != nil {
		r.logger.Errorw("failed to update pricing",
			"id", model.ID,
			"error", result.Error)
		return fmt.Errorf("failed to update pricing: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.logger.Warnw("no rows affected when updating pricing", "id", model.ID)
		return fmt.Errorf("pricing not found: id=%d", model.ID)
	}

	r.logger.Infow("pricing updated successfully",
		"id", model.ID,
		"plan_id", model.PlanID,
		"billing_cycle", model.BillingCycle)

	return nil
}

// Delete soft deletes a pricing record
func (r *PlanPricingRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.SubscriptionPlanPricingModel{}, id)

	if result.Error != nil {
		r.logger.Errorw("failed to delete pricing", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete pricing: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.logger.Warnw("no rows affected when deleting pricing", "id", id)
		return fmt.Errorf("pricing not found: id=%d", id)
	}

	r.logger.Infow("pricing deleted successfully", "id", id)
	return nil
}
