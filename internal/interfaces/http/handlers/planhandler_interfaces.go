package handlers

import (
	"context"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
)

// Use case interfaces for PlanHandler

type createPlanUseCase interface {
	Execute(ctx context.Context, cmd usecases.CreatePlanCommand) (*subdto.PlanDTO, error)
}

type updatePlanUseCase interface {
	Execute(ctx context.Context, cmd usecases.UpdatePlanCommand) (*subdto.PlanDTO, error)
}

type getPlanUseCase interface {
	ExecuteBySID(ctx context.Context, sid string) (*subdto.PlanDTO, error)
}

type listPlansUseCase interface {
	Execute(ctx context.Context, query usecases.ListPlansQuery) (*usecases.ListPlansResult, error)
}

type getPublicPlansUseCase interface {
	Execute(ctx context.Context) ([]*subdto.PlanDTO, error)
}

type activatePlanUseCase interface {
	Execute(ctx context.Context, planSID string) error
}

type deactivatePlanUseCase interface {
	Execute(ctx context.Context, planSID string) error
}

type deletePlanUseCase interface {
	Execute(ctx context.Context, planSID string) error
}

type getPlanPricingsUseCase interface {
	Execute(ctx context.Context, query usecases.GetPlanPricingsQuery) ([]*subdto.PricingOptionDTO, error)
}
