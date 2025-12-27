package adapters

import (
	"context"

	"gorm.io/gorm"

	nodeusecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
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

// safeTokenPrefix returns a safe prefix of the token for logging.
// Returns at most 8 characters followed by "..." to avoid exposing the full token.
func safeTokenPrefix(token string) string {
	if len(token) <= 8 {
		return token + "..."
	}
	return token[:8] + "..."
}

func (v *SubscriptionTokenValidatorAdapter) Validate(ctx context.Context, linkToken string) error {
	var subscriptionModel models.SubscriptionModel
	if err := v.db.WithContext(ctx).
		Where("link_token = ?", linkToken).
		First(&subscriptionModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			v.logger.Warnw("subscription not found", "link_token_prefix", safeTokenPrefix(linkToken))
			return errors.NewNotFoundError("subscription not found")
		}
		v.logger.Errorw("failed to query subscription", "error", err)
		return errors.NewInternalError("failed to validate subscription")
	}

	if subscriptionModel.Status != string(valueobjects.StatusActive) {
		v.logger.Warnw("subscription is not active", "subscription_id", subscriptionModel.ID, "status", subscriptionModel.Status)
		return errors.NewValidationError("subscription is not active")
	}

	if subscriptionModel.EndDate.Before(biztime.NowUTC()) {
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
			v.logger.Warnw("subscription not found", "link_token_prefix", safeTokenPrefix(linkToken))
			return nil, errors.NewNotFoundError("subscription not found")
		}
		v.logger.Errorw("failed to query subscription", "error", err)
		return nil, errors.NewInternalError("failed to validate subscription")
	}

	if subscriptionModel.Status != string(valueobjects.StatusActive) {
		v.logger.Warnw("subscription is not active", "subscription_id", subscriptionModel.ID, "status", subscriptionModel.Status)
		return nil, errors.NewValidationError("subscription is not active")
	}

	if subscriptionModel.EndDate.Before(biztime.NowUTC()) {
		v.logger.Warnw("subscription expired", "subscription_id", subscriptionModel.ID, "end_date", subscriptionModel.EndDate)
		return nil, errors.NewValidationError("subscription expired")
	}

	return &nodeusecases.SubscriptionValidationResult{
		SubscriptionUUID: subscriptionModel.UUID,
	}, nil
}
