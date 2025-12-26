package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// allowedSubscriptionSortByFields defines the whitelist of allowed ORDER BY fields
// to prevent SQL injection attacks.
var allowedSubscriptionSortByFields = map[string]bool{
	"id":         true,
	"sid":        true,
	"user_id":    true,
	"plan_id":    true,
	"status":     true,
	"start_date": true,
	"end_date":   true,
	"created_at": true,
	"updated_at": true,
}

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

func (r *SubscriptionRepositoryImpl) GetBySID(ctx context.Context, sid string) (*subscription.Subscription, error) {
	var model models.SubscriptionModel

	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get subscription by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map subscription model to entity", "sid", sid, "error", err)
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

	// Query both active and trialing status for compatibility
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []string{string(valueobjects.StatusActive), string(valueobjects.StatusTrialing)}).
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
	// Get node's group_ids from nodes table
	var nodeModel models.NodeModel
	if err := r.db.WithContext(ctx).
		Select("group_ids").
		Where("id = ?", nodeID).
		First(&nodeModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Infow("node not found", "node_id", nodeID)
			return []*subscription.Subscription{}, nil
		}
		r.logger.Errorw("failed to query node", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to query node: %w", err)
	}

	// Parse group_ids from JSON
	var groupIDs []uint
	if len(nodeModel.GroupIDs) > 0 {
		if err := json.Unmarshal(nodeModel.GroupIDs, &groupIDs); err != nil {
			r.logger.Errorw("failed to unmarshal group_ids", "node_id", nodeID, "error", err)
			return nil, fmt.Errorf("failed to parse group_ids: %w", err)
		}
	}

	// If node has no group_ids, no subscriptions available
	if len(groupIDs) == 0 {
		r.logger.Infow("node has no resource groups", "node_id", nodeID)
		return []*subscription.Subscription{}, nil
	}

	// Get the plan_ids from resource_groups table
	var planIDs []uint
	if err := r.db.WithContext(ctx).
		Table("resource_groups").
		Select("plan_id").
		Scopes(db.NotDeleted()).
		Where("id IN ? AND status = ?", groupIDs, "active").
		Pluck("plan_id", &planIDs).Error; err != nil {
		r.logger.Errorw("failed to query resource groups", "group_ids", groupIDs, "error", err)
		return nil, fmt.Errorf("failed to query resource groups: %w", err)
	}

	if len(planIDs) == 0 {
		r.logger.Infow("no active resource groups found", "group_ids", groupIDs)
		return []*subscription.Subscription{}, nil
	}

	// Query active subscriptions for these plans (including trialing status for compatibility)
	var subscriptionModels []*models.SubscriptionModel
	if err := r.db.WithContext(ctx).
		Where("plan_id IN ? AND status IN ?", planIDs, []string{string(valueobjects.StatusActive), string(valueobjects.StatusTrialing)}).
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

	r.logger.Debugw("retrieved active subscriptions for node",
		"node_id", nodeID,
		"group_ids", groupIDs,
		"plan_ids", planIDs,
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

	result := r.db.WithContext(ctx).Model(model).
		Where("id = ?", model.ID).
		Updates(map[string]interface{}{
			"user_id":              model.UserID,
			"plan_id":              model.PlanID,
			"status":               model.Status,
			"start_date":           model.StartDate,
			"end_date":             model.EndDate,
			"auto_renew":           model.AutoRenew,
			"current_period_start": model.CurrentPeriodStart,
			"current_period_end":   model.CurrentPeriodEnd,
			"link_token":           model.LinkToken,
			"cancelled_at":         model.CancelledAt,
			"cancel_reason":        model.CancelReason,
			"metadata":             model.Metadata,
			"version":              model.Version,
			"updated_at":           model.UpdatedAt,
		})

	if result.Error != nil {
		r.logger.Errorw("failed to update subscription", "id", model.ID, "error", result.Error)
		return fmt.Errorf("failed to update subscription: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.

	r.logger.Infow("subscription updated successfully", "id", model.ID)
	return nil
}

func (r *SubscriptionRepositoryImpl) Delete(ctx context.Context, id uint) error {
	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Delete(&models.SubscriptionModel{}, id).Error; err != nil {
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
		Where("status IN ?", []string{string(valueobjects.StatusActive), string(valueobjects.StatusTrialing)}).
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
		Where("status IN ?", []string{string(valueobjects.StatusActive), string(valueobjects.StatusTrialing), string(valueobjects.StatusPastDue)}).
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

	query := r.db.WithContext(ctx).Table("subscriptions").Scopes(db.NotDeleted())

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

	// Apply sorting with whitelist validation to prevent SQL injection
	sortBy := filter.SortBy
	if sortBy == "" || !allowedSubscriptionSortByFields[sortBy] {
		sortBy = "created_at"
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

// CountByPlanID counts subscriptions by plan ID (excluding soft-deleted records).
func (r *SubscriptionRepositoryImpl) CountByPlanID(ctx context.Context, planID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SubscriptionModel{}).
		Scopes(db.NotDeleted()).
		Where("plan_id = ?", planID).
		Count(&count).Error; err != nil {
		r.logger.Errorw("failed to count subscriptions by plan ID", "plan_id", planID, "error", err)
		return 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}
	return count, nil
}

// CountByStatus counts subscriptions by status (excluding soft-deleted records).
func (r *SubscriptionRepositoryImpl) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SubscriptionModel{}).
		Scopes(db.NotDeleted()).
		Where("status = ?", status).
		Count(&count).Error; err != nil {
		r.logger.Errorw("failed to count subscriptions by status", "status", status, "error", err)
		return 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}
	return count, nil
}
