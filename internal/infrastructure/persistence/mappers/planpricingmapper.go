package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// PlanPricingMapper handles mapping between PlanPricing domain object and database model
type PlanPricingMapper struct{}

// NewPlanPricingMapper creates a new PlanPricingMapper
func NewPlanPricingMapper() *PlanPricingMapper {
	return &PlanPricingMapper{}
}

// ToDomain converts database model to domain value object
func (m *PlanPricingMapper) ToDomain(model *models.PlanPricingModel) (*valueobjects.PlanPricing, error) {
	if model == nil {
		return nil, fmt.Errorf("pricing model cannot be nil")
	}

	// Parse billing cycle
	cycle, err := valueobjects.ParseBillingCycle(model.BillingCycle)
	if err != nil {
		return nil, fmt.Errorf("failed to parse billing cycle: %w", err)
	}

	// Reconstruct domain object
	pricing := valueobjects.ReconstructPlanPricing(
		model.ID,
		model.SID,
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
func (m *PlanPricingMapper) ToModel(pricing *valueobjects.PlanPricing) (*models.PlanPricingModel, error) {
	if pricing == nil {
		return nil, fmt.Errorf("pricing cannot be nil")
	}

	model := &models.PlanPricingModel{
		ID:           pricing.ID(),
		SID:          pricing.SID(),
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
func (m *PlanPricingMapper) ToDomainList(models []*models.PlanPricingModel) ([]*valueobjects.PlanPricing, error) {
	if models == nil {
		return []*valueobjects.PlanPricing{}, nil
	}

	pricings := make([]*valueobjects.PlanPricing, 0, len(models))
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
func (m *PlanPricingMapper) ToModelList(pricings []*valueobjects.PlanPricing) ([]*models.PlanPricingModel, error) {
	if pricings == nil {
		return []*models.PlanPricingModel{}, nil
	}

	modelList := make([]*models.PlanPricingModel, 0, len(pricings))
	for _, pricing := range pricings {
		model, err := m.ToModel(pricing)
		if err != nil {
			return nil, fmt.Errorf("failed to convert pricing: %w", err)
		}
		modelList = append(modelList, model)
	}

	return modelList, nil
}
