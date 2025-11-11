package mappers

import (
	"fmt"

	"orris/internal/domain/subscription/value_objects"
	"orris/internal/infrastructure/persistence/models"
)

// PlanPricingMapper handles mapping between PlanPricing domain object and database model
type PlanPricingMapper struct{}

// NewPlanPricingMapper creates a new PlanPricingMapper
func NewPlanPricingMapper() *PlanPricingMapper {
	return &PlanPricingMapper{}
}

// ToDomain converts database model to domain value object
func (m *PlanPricingMapper) ToDomain(model *models.SubscriptionPlanPricingModel) (*value_objects.PlanPricing, error) {
	if model == nil {
		return nil, fmt.Errorf("pricing model cannot be nil")
	}

	// Parse billing cycle
	cycle, err := value_objects.ParseBillingCycle(model.BillingCycle)
	if err != nil {
		return nil, fmt.Errorf("failed to parse billing cycle: %w", err)
	}

	// Reconstruct domain object
	pricing := value_objects.ReconstructPlanPricing(
		model.ID,
		model.PlanID,
		cycle,
		model.Price,
		model.Currency,
		model.IsActive,
		model.CreatedAt,
		model.UpdatedAt,
	)

	return pricing, nil
}

// ToModel converts domain value object to database model
func (m *PlanPricingMapper) ToModel(pricing *value_objects.PlanPricing) (*models.SubscriptionPlanPricingModel, error) {
	if pricing == nil {
		return nil, fmt.Errorf("pricing cannot be nil")
	}

	model := &models.SubscriptionPlanPricingModel{
		ID:           pricing.ID(),
		PlanID:       pricing.PlanID(),
		BillingCycle: pricing.BillingCycle().String(),
		Price:        pricing.Price(),
		Currency:     pricing.Currency(),
		IsActive:     pricing.IsActive(),
		CreatedAt:    pricing.CreatedAt(),
		UpdatedAt:    pricing.UpdatedAt(),
	}

	return model, nil
}

// ToDomainList converts a list of database models to domain value objects
func (m *PlanPricingMapper) ToDomainList(models []*models.SubscriptionPlanPricingModel) ([]*value_objects.PlanPricing, error) {
	if models == nil {
		return []*value_objects.PlanPricing{}, nil
	}

	pricings := make([]*value_objects.PlanPricing, 0, len(models))
	for _, model := range models {
		pricing, err := m.ToDomain(model)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model at index: %w", err)
		}
		pricings = append(pricings, pricing)
	}

	return pricings, nil
}

// ToModelList converts a list of domain value objects to database models
func (m *PlanPricingMapper) ToModelList(pricings []*value_objects.PlanPricing) ([]*models.SubscriptionPlanPricingModel, error) {
	if pricings == nil {
		return []*models.SubscriptionPlanPricingModel{}, nil
	}

	modelList := make([]*models.SubscriptionPlanPricingModel, 0, len(pricings))
	for _, pricing := range pricings {
		model, err := m.ToModel(pricing)
		if err != nil {
			return nil, fmt.Errorf("failed to convert pricing: %w", err)
		}
		modelList = append(modelList, model)
	}

	return modelList, nil
}
