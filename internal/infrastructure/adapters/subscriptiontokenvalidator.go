package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"

	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
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

func (v *SubscriptionTokenValidatorAdapter) Validate(ctx context.Context, token string) error {
	tokenHash := hashToken(token)

	var tokenModel models.SubscriptionTokenModel
	if err := v.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&tokenModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			v.logger.Warnw("subscription token not found", "token_hash", tokenHash)
			return errors.NewNotFoundError("subscription token not found")
		}
		v.logger.Errorw("failed to query subscription token", "error", err)
		return errors.NewInternalError("failed to validate token")
	}

	if !tokenModel.IsActive {
		v.logger.Warnw("subscription token is inactive", "token_id", tokenModel.ID)
		return errors.NewValidationError("subscription token is inactive")
	}

	if tokenModel.ExpiresAt != nil && tokenModel.ExpiresAt.Before(time.Now()) {
		v.logger.Warnw("subscription token expired", "token_id", tokenModel.ID, "expired_at", tokenModel.ExpiresAt)
		return errors.NewValidationError("subscription token expired")
	}

	var subscriptionModel models.SubscriptionModel
	if err := v.db.WithContext(ctx).
		Where("id = ?", tokenModel.SubscriptionID).
		First(&subscriptionModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			v.logger.Warnw("subscription not found", "subscription_id", tokenModel.SubscriptionID)
			return errors.NewNotFoundError("subscription not found")
		}
		v.logger.Errorw("failed to query subscription", "error", err)
		return errors.NewInternalError("failed to validate token")
	}

	if subscriptionModel.Status != "active" {
		v.logger.Warnw("subscription is not active", "subscription_id", subscriptionModel.ID, "status", subscriptionModel.Status)
		return errors.NewValidationError("subscription is not active")
	}

	if subscriptionModel.EndDate.Before(time.Now()) {
		v.logger.Warnw("subscription expired", "subscription_id", subscriptionModel.ID, "end_date", subscriptionModel.EndDate)
		return errors.NewValidationError("subscription expired")
	}

	v.db.WithContext(ctx).
		Model(&tokenModel).
		Updates(map[string]interface{}{
			"last_used_at": time.Now(),
			"usage_count":  gorm.Expr("usage_count + 1"),
		})

	return nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
