package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type SubscriptionTokenRepositoryImpl struct {
	db     *gorm.DB
	logger logger.Interface
}

func NewSubscriptionTokenRepository(db *gorm.DB, logger logger.Interface) subscription.SubscriptionTokenRepository {
	return &SubscriptionTokenRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *SubscriptionTokenRepositoryImpl) Create(ctx context.Context, token *subscription.SubscriptionToken) error {
	model, err := r.toModel(token)
	if err != nil {
		r.logger.Errorw("failed to convert token to model", "error", err)
		return fmt.Errorf("failed to convert token to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create subscription token", "error", err, "subscription_id", token.SubscriptionID())
		return fmt.Errorf("failed to create subscription token: %w", err)
	}

	if err := token.SetID(model.ID); err != nil {
		return err
	}

	r.logger.Infow("subscription token created successfully", "token_id", model.ID, "subscription_id", token.SubscriptionID())
	return nil
}

func (r *SubscriptionTokenRepositoryImpl) GetByID(ctx context.Context, id uint) (*subscription.SubscriptionToken, error) {
	var model models.SubscriptionTokenModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get subscription token by ID", "error", err, "token_id", id)
		return nil, fmt.Errorf("failed to get subscription token: %w", err)
	}

	return r.toEntity(&model)
}

func (r *SubscriptionTokenRepositoryImpl) GetByTokenHash(ctx context.Context, tokenHash string) (*subscription.SubscriptionToken, error) {
	var model models.SubscriptionTokenModel
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get subscription token by hash", "error", err)
		return nil, fmt.Errorf("failed to get subscription token by hash: %w", err)
	}

	return r.toEntity(&model)
}

func (r *SubscriptionTokenRepositoryImpl) Update(ctx context.Context, token *subscription.SubscriptionToken) error {
	model, err := r.toModel(token)
	if err != nil {
		r.logger.Errorw("failed to convert token to model", "error", err)
		return fmt.Errorf("failed to convert token to model: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&models.SubscriptionTokenModel{}).
		Where("id = ?", token.ID()).
		Updates(map[string]interface{}{
			"name":         model.Name,
			"scope":        model.Scope,
			"expires_at":   model.ExpiresAt,
			"last_used_at": model.LastUsedAt,
			"last_used_ip": model.LastUsedIP,
			"usage_count":  model.UsageCount,
			"is_active":    model.IsActive,
			"revoked_at":   model.RevokedAt,
		})

	if result.Error != nil {
		r.logger.Errorw("failed to update subscription token", "error", result.Error, "token_id", token.ID())
		return fmt.Errorf("failed to update subscription token: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.

	r.logger.Infow("subscription token updated successfully", "token_id", token.ID())
	return nil
}

func (r *SubscriptionTokenRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.SubscriptionTokenModel{}, id)
	if result.Error != nil {
		r.logger.Errorw("failed to delete subscription token", "error", result.Error, "token_id", id)
		return fmt.Errorf("failed to delete subscription token: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("subscription token not found")
	}

	r.logger.Infow("subscription token deleted successfully", "token_id", id)
	return nil
}

func (r *SubscriptionTokenRepositoryImpl) GetBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*subscription.SubscriptionToken, error) {
	var tokenModels []*models.SubscriptionTokenModel
	err := r.db.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Order("created_at DESC").
		Find(&tokenModels).Error

	if err != nil {
		r.logger.Errorw("failed to get tokens by subscription ID", "error", err, "subscription_id", subscriptionID)
		return nil, fmt.Errorf("failed to get tokens by subscription ID: %w", err)
	}

	return r.toEntities(tokenModels)
}

func (r *SubscriptionTokenRepositoryImpl) GetActiveBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*subscription.SubscriptionToken, error) {
	var tokenModels []*models.SubscriptionTokenModel
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("subscription_id = ? AND is_active = ?", subscriptionID, true).
		Where("expires_at IS NULL OR expires_at > ?", now).
		Order("created_at DESC").
		Find(&tokenModels).Error

	if err != nil {
		r.logger.Errorw("failed to get active tokens by subscription ID", "error", err, "subscription_id", subscriptionID)
		return nil, fmt.Errorf("failed to get active tokens by subscription ID: %w", err)
	}

	return r.toEntities(tokenModels)
}

func (r *SubscriptionTokenRepositoryImpl) RevokeAllBySubscriptionID(ctx context.Context, subscriptionID uint) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.SubscriptionTokenModel{}).
		Where("subscription_id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"is_active":  false,
			"revoked_at": now,
		})

	if result.Error != nil {
		r.logger.Errorw("failed to revoke all tokens", "error", result.Error, "subscription_id", subscriptionID)
		return fmt.Errorf("failed to revoke all tokens: %w", result.Error)
	}

	r.logger.Infow("all subscription tokens revoked", "subscription_id", subscriptionID, "count", result.RowsAffected)
	return nil
}

func (r *SubscriptionTokenRepositoryImpl) DeleteExpiredTokens(ctx context.Context) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Where("expires_at < ? AND expires_at IS NOT NULL", now).
		Delete(&models.SubscriptionTokenModel{})

	if result.Error != nil {
		r.logger.Errorw("failed to delete expired tokens", "error", result.Error)
		return fmt.Errorf("failed to delete expired tokens: %w", result.Error)
	}

	r.logger.Infow("expired tokens deleted", "count", result.RowsAffected)
	return nil
}

func (r *SubscriptionTokenRepositoryImpl) toEntity(model *models.SubscriptionTokenModel) (*subscription.SubscriptionToken, error) {
	if model == nil {
		return nil, nil
	}

	scope, err := vo.NewTokenScope(model.Scope)
	if err != nil {
		r.logger.Errorw("invalid token scope", "error", err, "value", model.Scope)
		return nil, fmt.Errorf("invalid token scope: %w", err)
	}

	return subscription.ReconstructSubscriptionToken(
		model.ID,
		model.SID,
		model.SubscriptionID,
		model.Name,
		model.TokenHash,
		model.Prefix,
		*scope,
		model.ExpiresAt,
		model.LastUsedAt,
		model.LastUsedIP,
		model.UsageCount,
		model.IsActive,
		model.CreatedAt,
		model.RevokedAt,
	)
}

func (r *SubscriptionTokenRepositoryImpl) toModel(token *subscription.SubscriptionToken) (*models.SubscriptionTokenModel, error) {
	if token == nil {
		return nil, nil
	}

	return &models.SubscriptionTokenModel{
		ID:             token.ID(),
		SubscriptionID: token.SubscriptionID(),
		Name:           token.Name(),
		TokenHash:      token.TokenHash(),
		Prefix:         token.Prefix(),
		Scope:          token.Scope().String(),
		ExpiresAt:      token.ExpiresAt(),
		LastUsedAt:     token.LastUsedAt(),
		LastUsedIP:     token.LastUsedIP(),
		UsageCount:     token.UsageCount(),
		IsActive:       token.IsActive(),
		CreatedAt:      token.CreatedAt(),
		RevokedAt:      token.RevokedAt(),
	}, nil
}

func (r *SubscriptionTokenRepositoryImpl) toEntities(models []*models.SubscriptionTokenModel) ([]*subscription.SubscriptionToken, error) {
	tokens := make([]*subscription.SubscriptionToken, 0, len(models))

	for _, model := range models {
		token, err := r.toEntity(model)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model ID %d: %w", model.ID, err)
		}
		if token != nil {
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}
