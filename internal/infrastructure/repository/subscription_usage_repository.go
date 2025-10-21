package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"orris/internal/domain/subscription"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/logger"
)

type SubscriptionUsageRepositoryImpl struct {
	db     *gorm.DB
	logger logger.Interface
}

func NewSubscriptionUsageRepository(db *gorm.DB, logger logger.Interface) subscription.SubscriptionUsageRepository {
	return &SubscriptionUsageRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *SubscriptionUsageRepositoryImpl) GetCurrentUsage(ctx context.Context, subscriptionID uint) (*subscription.SubscriptionUsage, error) {
	period := getCurrentBillingPeriod()

	var model models.SubscriptionUsageModel
	err := r.db.WithContext(ctx).
		Where("subscription_id = ? AND period_start <= ? AND period_end >= ?", subscriptionID, period, period).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get current usage", "error", err, "subscription_id", subscriptionID)
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}

	return r.toEntity(&model)
}

func (r *SubscriptionUsageRepositoryImpl) Upsert(ctx context.Context, usage *subscription.SubscriptionUsage) error {
	model, err := r.toModel(usage)
	if err != nil {
		r.logger.Errorw("failed to convert usage to model", "error", err)
		return fmt.Errorf("failed to convert usage to model: %w", err)
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "subscription_id"}, {Name: "period_start"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"api_requests",
			"api_data_out",
			"api_data_in",
			"storage_used",
			"users_count",
			"projects_count",
			"webhook_calls",
			"emails_sent",
			"reports_generated",
			"updated_at",
		}),
	}).Create(model)

	if result.Error != nil {
		r.logger.Errorw("failed to upsert subscription usage", "error", result.Error, "subscription_id", usage.SubscriptionID())
		return fmt.Errorf("failed to upsert subscription usage: %w", result.Error)
	}

	if usage.ID() == 0 && model.ID > 0 {
		if reconErr := usage.SetID(model.ID); reconErr != nil {
			return reconErr
		}
	}

	r.logger.Infow("subscription usage upserted successfully", "usage_id", model.ID, "subscription_id", usage.SubscriptionID())
	return nil
}

func (r *SubscriptionUsageRepositoryImpl) IncrementAPIRequests(ctx context.Context, subscriptionID uint, count uint64) error {
	period := getCurrentBillingPeriod()
	periodEnd := period.AddDate(0, 1, 0)

	result := r.db.WithContext(ctx).
		Model(&models.SubscriptionUsageModel{}).
		Where("subscription_id = ? AND period_start = ?", subscriptionID, period).
		Updates(map[string]interface{}{
			"api_requests": gorm.Expr("api_requests + ?", count),
			"updated_at":   time.Now(),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to increment API requests", "error", result.Error, "subscription_id", subscriptionID)
		return fmt.Errorf("failed to increment API requests: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		newUsage := &models.SubscriptionUsageModel{
			SubscriptionID: subscriptionID,
			PeriodStart:    period,
			PeriodEnd:      periodEnd,
			APIRequests:    count,
			UpdatedAt:      time.Now(),
		}

		if err := r.db.WithContext(ctx).Create(newUsage).Error; err != nil {
			r.logger.Errorw("failed to create usage record for API requests", "error", err, "subscription_id", subscriptionID)
			return fmt.Errorf("failed to create usage record: %w", err)
		}

		r.logger.Infow("created new usage record with API requests", "subscription_id", subscriptionID, "count", count)
		return nil
	}

	r.logger.Infow("API requests incremented", "subscription_id", subscriptionID, "count", count)
	return nil
}

func (r *SubscriptionUsageRepositoryImpl) IncrementStorageUsed(ctx context.Context, subscriptionID uint, bytes uint64) error {
	period := getCurrentBillingPeriod()
	periodEnd := period.AddDate(0, 1, 0)

	result := r.db.WithContext(ctx).
		Model(&models.SubscriptionUsageModel{}).
		Where("subscription_id = ? AND period_start = ?", subscriptionID, period).
		Updates(map[string]interface{}{
			"storage_used": gorm.Expr("storage_used + ?", bytes),
			"updated_at":   time.Now(),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to increment storage used", "error", result.Error, "subscription_id", subscriptionID)
		return fmt.Errorf("failed to increment storage used: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		newUsage := &models.SubscriptionUsageModel{
			SubscriptionID: subscriptionID,
			PeriodStart:    period,
			PeriodEnd:      periodEnd,
			StorageUsed:    bytes,
			UpdatedAt:      time.Now(),
		}

		if err := r.db.WithContext(ctx).Create(newUsage).Error; err != nil {
			r.logger.Errorw("failed to create usage record for storage", "error", err, "subscription_id", subscriptionID)
			return fmt.Errorf("failed to create usage record: %w", err)
		}

		r.logger.Infow("created new usage record with storage", "subscription_id", subscriptionID, "bytes", bytes)
		return nil
	}

	r.logger.Infow("storage used incremented", "subscription_id", subscriptionID, "bytes", bytes)
	return nil
}

func (r *SubscriptionUsageRepositoryImpl) GetUsageHistory(ctx context.Context, subscriptionID uint, from, to time.Time) ([]*subscription.SubscriptionUsage, error) {
	var usageModels []*models.SubscriptionUsageModel
	err := r.db.WithContext(ctx).
		Where("subscription_id = ? AND period_start BETWEEN ? AND ?", subscriptionID, from, to).
		Order("period_start DESC").
		Find(&usageModels).Error

	if err != nil {
		r.logger.Errorw("failed to get usage history", "error", err, "subscription_id", subscriptionID)
		return nil, fmt.Errorf("failed to get usage history: %w", err)
	}

	return r.toEntities(usageModels)
}

func (r *SubscriptionUsageRepositoryImpl) ResetUsage(ctx context.Context, subscriptionID uint, period time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&models.SubscriptionUsageModel{}).
		Where("subscription_id = ? AND period_start = ?", subscriptionID, period).
		Updates(map[string]interface{}{
			"api_requests":      0,
			"api_data_out":      0,
			"api_data_in":       0,
			"storage_used":      0,
			"users_count":       0,
			"projects_count":    0,
			"webhook_calls":     0,
			"emails_sent":       0,
			"reports_generated": 0,
			"last_reset_at":     time.Now(),
			"updated_at":        time.Now(),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to reset usage", "error", result.Error, "subscription_id", subscriptionID)
		return fmt.Errorf("failed to reset usage: %w", result.Error)
	}

	r.logger.Infow("usage reset successfully", "subscription_id", subscriptionID, "period", period)
	return nil
}

func (r *SubscriptionUsageRepositoryImpl) toEntity(model *models.SubscriptionUsageModel) (*subscription.SubscriptionUsage, error) {
	if model == nil {
		return nil, nil
	}

	return subscription.ReconstructSubscriptionUsage(
		model.ID,
		model.SubscriptionID,
		model.PeriodStart,
		model.APIRequests,
		model.APIDataOut,
		model.APIDataIn,
		model.StorageUsed,
		model.UsersCount,
		model.ProjectsCount,
		model.WebhookCalls,
		model.EmailsSent,
		model.ReportsGenerated,
		model.UpdatedAt,
	)
}

func (r *SubscriptionUsageRepositoryImpl) toModel(usage *subscription.SubscriptionUsage) (*models.SubscriptionUsageModel, error) {
	if usage == nil {
		return nil, nil
	}

	period := usage.Period()
	periodEnd := period.AddDate(0, 1, 0)

	return &models.SubscriptionUsageModel{
		ID:               usage.ID(),
		SubscriptionID:   usage.SubscriptionID(),
		PeriodStart:      usage.Period(),
		PeriodEnd:        periodEnd,
		APIRequests:      usage.APIRequests(),
		APIDataOut:       usage.APIDataOut(),
		APIDataIn:        usage.APIDataIn(),
		StorageUsed:      usage.StorageUsed(),
		UsersCount:       usage.UsersCount(),
		ProjectsCount:    usage.ProjectsCount(),
		WebhookCalls:     usage.WebhookCalls(),
		EmailsSent:       usage.EmailsSent(),
		ReportsGenerated: usage.ReportsGenerated(),
		UpdatedAt:        usage.UpdatedAt(),
	}, nil
}

func (r *SubscriptionUsageRepositoryImpl) toEntities(models []*models.SubscriptionUsageModel) ([]*subscription.SubscriptionUsage, error) {
	usages := make([]*subscription.SubscriptionUsage, 0, len(models))

	for _, model := range models {
		usage, err := r.toEntity(model)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model ID %d: %w", model.ID, err)
		}
		if usage != nil {
			usages = append(usages, usage)
		}
	}

	return usages, nil
}

func getCurrentBillingPeriod() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}
