package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"orris/internal/domain/subscription"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/logger"
)

type SubscriptionRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.SubscriptionMapper
	logger logger.Interface
}

func NewSubscriptionRepository(
	db *gorm.DB,
	logger logger.Interface,
) subscription.SubscriptionRepository {
	return &SubscriptionRepositoryImpl{
		db:     db,
		mapper: mappers.NewSubscriptionMapper(),
		logger: logger,
	}
}

func (r *SubscriptionRepositoryImpl) Create(ctx context.Context, subscriptionEntity *subscription.Subscription) error {
	model, err := r.mapper.ToModel(subscriptionEntity)
	if err != nil {
		r.logger.Errorw("failed to map subscription entity to model", "error", err)
		return fmt.Errorf("failed to map subscription entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create subscription in database", "error", err)
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	if err := subscriptionEntity.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set subscription ID", "error", err)
		return fmt.Errorf("failed to set subscription ID: %w", err)
	}

	r.logger.Infow("subscription created successfully", "id", model.ID, "user_id", model.UserID, "plan_id", model.PlanID)
	return nil
}

func (r *SubscriptionRepositoryImpl) GetByID(ctx context.Context, id uint) (*subscription.Subscription, error) {
	var model models.SubscriptionModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get subscription by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map subscription model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map subscription: %w", err)
	}

	return entity, nil
}

func (r *SubscriptionRepositoryImpl) GetByUserID(ctx context.Context, userID uint) ([]*subscription.Subscription, error) {
	var models []*models.SubscriptionModel

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&models).Error; err != nil {
		r.logger.Errorw("failed to get subscriptions by user ID", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	entities, err := r.mapper.ToEntities(models)
	if err != nil {
		r.logger.Errorw("failed to map subscription models to entities", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to map subscriptions: %w", err)
	}

	return entities, nil
}

func (r *SubscriptionRepositoryImpl) GetActiveByUserID(ctx context.Context, userID uint) ([]*subscription.Subscription, error) {
	var models []*models.SubscriptionModel

	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, "active").
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		r.logger.Errorw("failed to get active subscriptions by user ID", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get active subscriptions: %w", err)
	}

	entities, err := r.mapper.ToEntities(models)
	if err != nil {
		r.logger.Errorw("failed to map subscription models to entities", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to map subscriptions: %w", err)
	}

	return entities, nil
}

func (r *SubscriptionRepositoryImpl) GetActiveSubscriptionsByNodeID(ctx context.Context, nodeID uint) ([]*subscription.Subscription, error) {
	// Query node groups that contain this node
	var nodeGroupNodeModels []models.NodeGroupNodeModel
	if err := r.db.WithContext(ctx).
		Where("node_id = ?", nodeID).
		Find(&nodeGroupNodeModels).Error; err != nil {
		r.logger.Errorw("failed to query node groups for node", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to query node groups: %w", err)
	}

	if len(nodeGroupNodeModels) == 0 {
		r.logger.Infow("no node groups found for node", "node_id", nodeID)
		return []*subscription.Subscription{}, nil
	}

	// Extract node group IDs
	nodeGroupIDs := make([]uint, len(nodeGroupNodeModels))
	for i, ngn := range nodeGroupNodeModels {
		nodeGroupIDs[i] = ngn.NodeGroupID
	}

	// Query subscription plans associated with these node groups
	var nodeGroupPlanModels []models.NodeGroupPlanModel
	if err := r.db.WithContext(ctx).
		Where("node_group_id IN ?", nodeGroupIDs).
		Find(&nodeGroupPlanModels).Error; err != nil {
		r.logger.Errorw("failed to query plans for node groups", "node_group_ids", nodeGroupIDs, "error", err)
		return nil, fmt.Errorf("failed to query subscription plans: %w", err)
	}

	if len(nodeGroupPlanModels) == 0 {
		r.logger.Infow("no subscription plans found for node groups", "node_group_ids", nodeGroupIDs)
		return []*subscription.Subscription{}, nil
	}

	// Extract unique plan IDs
	planIDsMap := make(map[uint]bool)
	for _, ngp := range nodeGroupPlanModels {
		planIDsMap[ngp.SubscriptionPlanID] = true
	}
	planIDs := make([]uint, 0, len(planIDsMap))
	for planID := range planIDsMap {
		planIDs = append(planIDs, planID)
	}

	// Query active subscriptions for these plans
	var subscriptionModels []*models.SubscriptionModel
	if err := r.db.WithContext(ctx).
		Where("plan_id IN ? AND status = ?", planIDs, "active").
		Order("created_at DESC").
		Find(&subscriptionModels).Error; err != nil {
		r.logger.Errorw("failed to query active subscriptions", "plan_ids", planIDs, "error", err)
		return nil, fmt.Errorf("failed to query subscriptions: %w", err)
	}

	entities, err := r.mapper.ToEntities(subscriptionModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscriptions: %w", err)
	}

	r.logger.Infow("retrieved active subscriptions for node",
		"node_id", nodeID,
		"node_group_count", len(nodeGroupIDs),
		"plan_count", len(planIDs),
		"subscription_count", len(entities),
	)

	return entities, nil
}

func (r *SubscriptionRepositoryImpl) Update(ctx context.Context, subscriptionEntity *subscription.Subscription) error {
	model, err := r.mapper.ToModel(subscriptionEntity)
	if err != nil {
		r.logger.Errorw("failed to map subscription entity to model", "id", subscriptionEntity.ID(), "error", err)
		return fmt.Errorf("failed to map subscription entity: %w", err)
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(model).
			Where("id = ? AND version = ?", model.ID, model.Version).
			Updates(map[string]interface{}{
				"user_id":              model.UserID,
				"plan_id":              model.PlanID,
				"status":               model.Status,
				"start_date":           model.StartDate,
				"end_date":             model.EndDate,
				"auto_renew":           model.AutoRenew,
				"current_period_start": model.CurrentPeriodStart,
				"current_period_end":   model.CurrentPeriodEnd,
				"cancelled_at":         model.CancelledAt,
				"cancel_reason":        model.CancelReason,
				"metadata":             model.Metadata,
				"version":              model.Version + 1,
				"updated_at":           model.UpdatedAt,
			})

		if result.Error != nil {
			r.logger.Errorw("failed to update subscription", "id", model.ID, "error", result.Error)
			return fmt.Errorf("failed to update subscription: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("subscription not found or version mismatch (optimistic lock failed)")
		}

		return nil
	})

	if err != nil {
		return err
	}

	r.logger.Infow("subscription updated successfully", "id", model.ID)
	return nil
}

func (r *SubscriptionRepositoryImpl) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.SubscriptionModel{}, id).Error; err != nil {
		r.logger.Errorw("failed to delete subscription", "id", id, "error", err)
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	r.logger.Infow("subscription deleted successfully", "id", id)
	return nil
}

func (r *SubscriptionRepositoryImpl) FindExpiringSubscriptions(ctx context.Context, days int) ([]*subscription.Subscription, error) {
	var models []*models.SubscriptionModel

	now := time.Now()
	expiryThreshold := now.AddDate(0, 0, days)

	if err := r.db.WithContext(ctx).
		Where("auto_renew = ?", true).
		Where("end_date BETWEEN ? AND ?", now, expiryThreshold).
		Where("status IN ?", []string{"active", "trialing"}).
		Order("end_date ASC").
		Find(&models).Error; err != nil {
		r.logger.Errorw("failed to find expiring subscriptions", "days", days, "error", err)
		return nil, fmt.Errorf("failed to find expiring subscriptions: %w", err)
	}

	entities, err := r.mapper.ToEntities(models)
	if err != nil {
		r.logger.Errorw("failed to map subscription models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscriptions: %w", err)
	}

	return entities, nil
}

func (r *SubscriptionRepositoryImpl) FindExpiredSubscriptions(ctx context.Context) ([]*subscription.Subscription, error) {
	var models []*models.SubscriptionModel

	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("end_date < ?", now).
		Where("status IN ?", []string{"active", "trialing", "past_due"}).
		Order("end_date ASC").
		Find(&models).Error; err != nil {
		r.logger.Errorw("failed to find expired subscriptions", "error", err)
		return nil, fmt.Errorf("failed to find expired subscriptions: %w", err)
	}

	entities, err := r.mapper.ToEntities(models)
	if err != nil {
		r.logger.Errorw("failed to map subscription models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscriptions: %w", err)
	}

	return entities, nil
}

func (r *SubscriptionRepositoryImpl) List(ctx context.Context, filter subscription.SubscriptionFilter) ([]*subscription.Subscription, int64, error) {
	var models []*models.SubscriptionModel
	var total int64

	query := r.db.WithContext(ctx).Table("subscriptions")

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.PlanID != nil {
		query = query.Where("plan_id = ?", *filter.PlanID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count subscriptions", "error", err)
		return nil, 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}

	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}

	order := "DESC"
	if !filter.SortDesc {
		order = "ASC"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, order))

	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Find(&models).Error; err != nil {
		r.logger.Errorw("failed to list subscriptions", "error", err)
		return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	entities, err := r.mapper.ToEntities(models)
	if err != nil {
		r.logger.Errorw("failed to map subscription models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map subscriptions: %w", err)
	}

	return entities, total, nil
}

func (r *SubscriptionRepositoryImpl) CountByPlanID(ctx context.Context, planID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SubscriptionModel{}).Where("plan_id = ?", planID).Count(&count).Error; err != nil {
		r.logger.Errorw("failed to count subscriptions by plan ID", "plan_id", planID, "error", err)
		return 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}
	return count, nil
}

func (r *SubscriptionRepositoryImpl) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SubscriptionModel{}).Where("status = ?", status).Count(&count).Error; err != nil {
		r.logger.Errorw("failed to count subscriptions by status", "status", status, "error", err)
		return 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}
	return count, nil
}
