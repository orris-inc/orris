package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type PlanRepositoryImpl struct {
	db     *gorm.DB
	logger logger.Interface
}

func NewPlanRepository(db *gorm.DB, logger logger.Interface) subscription.PlanRepository {
	return &PlanRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *PlanRepositoryImpl) Create(ctx context.Context, plan *subscription.Plan) error {
	model, err := r.toModel(plan)
	if err != nil {
		r.logger.Errorw("failed to convert plan to model", "error", err)
		return fmt.Errorf("failed to convert plan to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create subscription plan", "error", err, "slug", plan.Slug())
		return fmt.Errorf("failed to create subscription plan: %w", err)
	}

	if err := plan.SetID(model.ID); err != nil {
		return err
	}

	r.logger.Infow("subscription plan created successfully", "plan_id", model.ID, "slug", plan.Slug())
	return nil
}

func (r *PlanRepositoryImpl) GetByID(ctx context.Context, id uint) (*subscription.Plan, error) {
	var model models.PlanModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get subscription plan by ID", "error", err, "plan_id", id)
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	return r.toEntity(&model)
}

func (r *PlanRepositoryImpl) GetBySlug(ctx context.Context, slug string) (*subscription.Plan, error) {
	var model models.PlanModel
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get subscription plan by slug", "error", err, "slug", slug)
		return nil, fmt.Errorf("failed to get subscription plan by slug: %w", err)
	}

	return r.toEntity(&model)
}

func (r *PlanRepositoryImpl) Update(ctx context.Context, plan *subscription.Plan) error {
	model, err := r.toModel(plan)
	if err != nil {
		r.logger.Errorw("failed to convert plan to model", "error", err)
		return fmt.Errorf("failed to convert plan to model: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&models.PlanModel{}).
		Where("id = ?", plan.ID()).
		Updates(map[string]interface{}{
			"name":           model.Name,
			"description":    model.Description,
			"price":          model.Price,
			"currency":       model.Currency,
			"billing_cycle":  model.BillingCycle,
			"trial_days":     model.TrialDays,
			"status":         model.Status,
			"features":       model.Features,
			"limits":         model.Limits,
			"api_rate_limit": model.APIRateLimit,
			"max_users":      model.MaxUsers,
			"max_projects":   model.MaxProjects,
			"is_public":      model.IsPublic,
			"sort_order":     model.SortOrder,
			"metadata":       model.Metadata,
			"version":        model.Version,
			"updated_at":     model.UpdatedAt,
		})

	if result.Error != nil {
		r.logger.Errorw("failed to update subscription plan", "error", result.Error, "plan_id", plan.ID())
		return fmt.Errorf("failed to update subscription plan: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.

	r.logger.Infow("subscription plan updated successfully", "plan_id", plan.ID())
	return nil
}

func (r *PlanRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.PlanModel{}, id)
	if result.Error != nil {
		r.logger.Errorw("failed to delete subscription plan", "error", result.Error, "plan_id", id)
		return fmt.Errorf("failed to delete subscription plan: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("subscription plan not found")
	}

	r.logger.Infow("subscription plan deleted successfully", "plan_id", id)
	return nil
}

func (r *PlanRepositoryImpl) GetActivePublicPlans(ctx context.Context) ([]*subscription.Plan, error) {
	var planModels []*models.PlanModel
	err := r.db.WithContext(ctx).
		Where("status = ? AND is_public = ?", "active", true).
		Order("sort_order ASC, created_at DESC").
		Find(&planModels).Error

	if err != nil {
		r.logger.Errorw("failed to get active public plans", "error", err)
		return nil, fmt.Errorf("failed to get active public plans: %w", err)
	}

	return r.toEntities(planModels)
}

func (r *PlanRepositoryImpl) GetAllActive(ctx context.Context) ([]*subscription.Plan, error) {
	var planModels []*models.PlanModel
	err := r.db.WithContext(ctx).
		Where("status = ?", "active").
		Order("sort_order ASC, created_at DESC").
		Find(&planModels).Error

	if err != nil {
		r.logger.Errorw("failed to get all active plans", "error", err)
		return nil, fmt.Errorf("failed to get all active plans: %w", err)
	}

	return r.toEntities(planModels)
}

func (r *PlanRepositoryImpl) List(ctx context.Context, filter subscription.PlanFilter) ([]*subscription.Plan, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.PlanModel{})

	if filter.Status != nil && *filter.Status != "" {
		query = query.Where("status = ?", *filter.Status)
	}

	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}

	if filter.BillingCycle != nil && *filter.BillingCycle != "" {
		query = query.Where("billing_cycle = ?", *filter.BillingCycle)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count subscription plans", "error", err)
		return nil, 0, fmt.Errorf("failed to count subscription plans: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = constants.DefaultPage
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	offset := (page - 1) * pageSize

	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "sort_order ASC, created_at DESC"
	}

	query = query.Offset(offset).Limit(pageSize).Order(sortBy)

	var planModels []*models.PlanModel
	if err := query.Find(&planModels).Error; err != nil {
		r.logger.Errorw("failed to list subscription plans", "error", err)
		return nil, 0, fmt.Errorf("failed to list subscription plans: %w", err)
	}

	plans, err := r.toEntities(planModels)
	if err != nil {
		return nil, 0, err
	}

	return plans, total, nil
}

func (r *PlanRepositoryImpl) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PlanModel{}).
		Where("slug = ?", slug).
		Count(&count).Error

	if err != nil {
		r.logger.Errorw("failed to check plan slug existence", "error", err, "slug", slug)
		return false, fmt.Errorf("failed to check plan slug existence: %w", err)
	}

	return count > 0, nil
}

func (r *PlanRepositoryImpl) toEntity(model *models.PlanModel) (*subscription.Plan, error) {
	if model == nil {
		return nil, nil
	}

	billingCycle, err := vo.NewBillingCycle(model.BillingCycle)
	if err != nil {
		r.logger.Errorw("invalid billing cycle", "error", err, "value", model.BillingCycle)
		return nil, fmt.Errorf("invalid billing cycle: %w", err)
	}

	var features *vo.PlanFeatures
	if model.Features != nil {
		var featuresData map[string]interface{}
		if err := json.Unmarshal(model.Features, &featuresData); err != nil {
			r.logger.Errorw("failed to unmarshal features", "error", err)
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}

		var featuresList []string
		if featuresRaw, ok := featuresData["features"]; ok {
			if featuresArray, ok := featuresRaw.([]interface{}); ok {
				for _, f := range featuresArray {
					if str, ok := f.(string); ok {
						featuresList = append(featuresList, str)
					}
				}
			}
		}

		var limits map[string]interface{}
		if model.Limits != nil {
			if err := json.Unmarshal(model.Limits, &limits); err != nil {
				r.logger.Errorw("failed to unmarshal limits", "error", err)
				return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
			}
		}

		features = vo.NewPlanFeatures(featuresList, limits)
	}

	var metadata map[string]interface{}
	if model.Metadata != nil {
		if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
			r.logger.Errorw("failed to unmarshal metadata", "error", err)
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return subscription.ReconstructPlan(
		model.ID,
		model.Name,
		model.Slug,
		model.Description,
		model.Price,
		model.Currency,
		*billingCycle,
		model.TrialDays,
		model.Status,
		features,
		model.APIRateLimit,
		model.MaxUsers,
		model.MaxProjects,
		model.IsPublic,
		model.SortOrder,
		metadata,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

func (r *PlanRepositoryImpl) toModel(plan *subscription.Plan) (*models.PlanModel, error) {
	if plan == nil {
		return nil, nil
	}

	var featuresJSON []byte
	var limitsJSON []byte
	if plan.Features() != nil {
		var err error
		featuresData := map[string]interface{}{
			"features": plan.Features().Features,
		}
		featuresJSON, err = json.Marshal(featuresData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal features: %w", err)
		}

		if plan.Features().Limits != nil {
			limitsJSON, err = json.Marshal(plan.Features().Limits)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal limits: %w", err)
			}
		}
	}

	var metadataJSON []byte
	if plan.Metadata() != nil {
		var err error
		metadataJSON, err = json.Marshal(plan.Metadata())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	return &models.PlanModel{
		ID:           plan.ID(),
		Name:         plan.Name(),
		Slug:         plan.Slug(),
		Description:  plan.Description(),
		Price:        plan.Price(),
		Currency:     plan.Currency(),
		BillingCycle: plan.BillingCycle().String(),
		TrialDays:    plan.TrialDays(),
		Status:       string(plan.Status()),
		Features:     featuresJSON,
		Limits:       limitsJSON,
		APIRateLimit: plan.APIRateLimit(),
		MaxUsers:     plan.MaxUsers(),
		MaxProjects:  plan.MaxProjects(),
		IsPublic:     plan.IsPublic(),
		SortOrder:    plan.SortOrder(),
		Metadata:     metadataJSON,
		Version:      plan.Version(),
		CreatedAt:    plan.CreatedAt(),
		UpdatedAt:    plan.UpdatedAt(),
	}, nil
}

func (r *PlanRepositoryImpl) toEntities(models []*models.PlanModel) ([]*subscription.Plan, error) {
	plans := make([]*subscription.Plan, 0, len(models))

	for _, model := range models {
		plan, err := r.toEntity(model)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model ID %d: %w", model.ID, err)
		}
		if plan != nil {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}
