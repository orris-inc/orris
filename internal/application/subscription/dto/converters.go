package dto

import (
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
)

// ToPricingOptionDTO converts a PlanPricing value object to PricingOptionDTO
// This function transforms domain layer pricing information to presentation layer
func ToPricingOptionDTO(pricing *vo.PlanPricing) *PricingOptionDTO {
	if pricing == nil {
		return nil
	}

	return &PricingOptionDTO{
		BillingCycle: pricing.BillingCycle().String(),
		Price:        pricing.Price(),
		Currency:     pricing.Currency(),
		IsActive:     pricing.IsActive(),
	}
}

// ToPricingOptionDTOList converts a list of PlanPricing value objects to PricingOptionDTO slice
// This function batch converts domain pricing information to presentation layer DTOs
// Returns an empty slice if the input slice is nil or empty
func ToPricingOptionDTOList(pricings []*vo.PlanPricing) []*PricingOptionDTO {
	if pricings == nil || len(pricings) == 0 {
		return []*PricingOptionDTO{}
	}

	dtos := make([]*PricingOptionDTO, 0, len(pricings))
	for _, pricing := range pricings {
		if pricing != nil {
			dtos = append(dtos, ToPricingOptionDTO(pricing))
		}
	}

	return dtos
}

// ToPlanDTOWithPricings converts a Plan and its pricing options to PlanDTO
// This function enriches the basic plan information with flexible pricing options
// The Pricings field will contain all available pricing options for different billing cycles
func ToPlanDTOWithPricings(plan *subscription.Plan, pricings []*vo.PlanPricing) *PlanDTO {
	if plan == nil {
		return nil
	}

	// Start with the basic plan DTO
	planDTO := ToPlanDTO(plan)

	// Add pricing options
	planDTO.Pricings = ToPricingOptionDTOList(pricings)

	return planDTO
}
