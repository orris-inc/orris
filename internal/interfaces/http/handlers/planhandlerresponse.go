package handlers

import (
	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
)

// PlanResponse is a type alias for the application DTO.
// Use explicit struct if API contract diverges from internal DTO.
type PlanResponse = subdto.PlanDTO

// PricingOptionResponse is a type alias for the application DTO.
type PricingOptionResponse = subdto.PricingOptionDTO
