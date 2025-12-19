package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// SubscriptionTokenMapper handles the conversion between domain entities and persistence models
type SubscriptionTokenMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.SubscriptionTokenModel) (*subscription.SubscriptionToken, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *subscription.SubscriptionToken) (*models.SubscriptionTokenModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.SubscriptionTokenModel) ([]*subscription.SubscriptionToken, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*subscription.SubscriptionToken) ([]*models.SubscriptionTokenModel, error)
}

// subscriptionTokenMapper is the concrete implementation of SubscriptionTokenMapper
type subscriptionTokenMapper struct{}

// NewSubscriptionTokenMapper creates a new subscription token mapper
func NewSubscriptionTokenMapper() SubscriptionTokenMapper {
	return &subscriptionTokenMapper{}
}

// ToEntity converts a persistence model to a domain entity
func (m *subscriptionTokenMapper) ToEntity(model *models.SubscriptionTokenModel) (*subscription.SubscriptionToken, error) {
	if model == nil {
		return nil, nil
	}

	// Parse token scope
	scope, err := vo.ParseTokenScope(model.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token scope: %w", err)
	}

	// Reconstruct subscription token using domain factory method
	entity, err := subscription.ReconstructSubscriptionToken(
		model.ID,
		model.SID,
		model.SubscriptionID,
		model.Name,
		model.TokenHash,
		model.Prefix,
		scope,
		model.ExpiresAt,
		model.LastUsedAt,
		model.LastUsedIP,
		model.UsageCount,
		model.IsActive,
		model.CreatedAt,
		model.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct subscription token entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *subscriptionTokenMapper) ToModel(entity *subscription.SubscriptionToken) (*models.SubscriptionTokenModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.SubscriptionTokenModel{
		ID:             entity.ID(),
		SID:            entity.SID(),
		SubscriptionID: entity.SubscriptionID(),
		Name:           entity.Name(),
		TokenHash:      entity.TokenHash(),
		Prefix:         entity.Prefix(),
		Scope:          entity.Scope().String(),
		ExpiresAt:      entity.ExpiresAt(),
		LastUsedAt:     entity.LastUsedAt(),
		LastUsedIP:     entity.LastUsedIP(),
		UsageCount:     entity.UsageCount(),
		IsActive:       entity.IsActive(),
		CreatedAt:      entity.CreatedAt(),
		RevokedAt:      entity.RevokedAt(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *subscriptionTokenMapper) ToEntities(models []*models.SubscriptionTokenModel) ([]*subscription.SubscriptionToken, error) {
	entities := make([]*subscription.SubscriptionToken, 0, len(models))

	for i, model := range models {
		entity, err := m.ToEntity(model)
		if err != nil {
			return nil, fmt.Errorf("failed to map model at index %d (ID %d): %w", i, model.ID, err)
		}
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// ToModels converts multiple domain entities to persistence models
func (m *subscriptionTokenMapper) ToModels(entities []*subscription.SubscriptionToken) ([]*models.SubscriptionTokenModel, error) {
	models := make([]*models.SubscriptionTokenModel, 0, len(entities))

	for i, entity := range entities {
		model, err := m.ToModel(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map entity at index %d (ID %d): %w", i, entity.ID(), err)
		}
		if model != nil {
			models = append(models, model)
		}
	}

	return models, nil
}
