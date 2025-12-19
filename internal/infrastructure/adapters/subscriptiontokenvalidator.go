package adapters

import (
	"context"
	"time"

	"gorm.io/gorm"

	nodeusecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type SubscriptionTokenValidatorAdapter struct {
	db     *gorm.DB
	logger logger.Interface
}

func NewSubscriptionTokenValidatorAdapter(db *gorm.DB, logger logger.Interface) *SubscriptionTokenValidatorAdapter {
	return &SubscriptionTokenValidatorAdapter{
		db:     db,
		logger: logger,
	}
}

func (v *SubscriptionTokenValidatorAdapter) Validate(ctx context.Context, linkToken string) error {
	var subscriptionModel models.SubscriptionModel
	if err := v.db.WithContext(ctx).
		Where("link_token = ?", linkToken).
		First(&subscriptionModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			v.logger.Warnw("subscription not found", "link_token_prefix", linkToken[:8]+"...")
			return errors.NewNotFoundError("subscription not found")
		}
		v.logger.Errorw("failed to query subscription", "error", err)
		return errors.NewInternalError("failed to validate subscription")
	}

	if subscriptionModel.Status != string(valueobjects.StatusActive) {
		v.logger.Warnw("subscription is not active", "subscription_id", subscriptionModel.ID, "status", subscriptionModel.Status)
		return errors.NewValidationError("subscription is not active")
	}

	if subscriptionModel.EndDate.Before(time.Now()) {
		v.logger.Warnw("subscription expired", "subscription_id", subscriptionModel.ID, "end_date", subscriptionModel.EndDate)
		return errors.NewValidationError("subscription expired")
	}

	return nil
}

func (v *SubscriptionTokenValidatorAdapter) ValidateAndGetSubscription(ctx context.Context, linkToken string) (*nodeusecases.SubscriptionValidationResult, error) {
	var subscriptionModel models.SubscriptionModel
	if err := v.db.WithContext(ctx).
		Where("link_token = ?", linkToken).
		First(&subscriptionModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			v.logger.Warnw("subscription not found", "link_token_prefix", linkToken[:8]+"...")
			return nil, errors.NewNotFoundError("subscription not found")
		}
		v.logger.Errorw("failed to query subscription", "error", err)
		return nil, errors.NewInternalError("failed to validate subscription")
	}

	if subscriptionModel.Status != string(valueobjects.StatusActive) {
		v.logger.Warnw("subscription is not active", "subscription_id", subscriptionModel.ID, "status", subscriptionModel.Status)
		return nil, errors.NewValidationError("subscription is not active")
	}

	if subscriptionModel.EndDate.Before(time.Now()) {
		v.logger.Warnw("subscription expired", "subscription_id", subscriptionModel.ID, "end_date", subscriptionModel.EndDate)
		return nil, errors.NewValidationError("subscription expired")
	}

	return &nodeusecases.SubscriptionValidationResult{
		SubscriptionUUID: subscriptionModel.UUID,
	}, nil
}
