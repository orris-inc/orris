package handlers

import (
	"context"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
)

// Use case interfaces for SubscriptionHandler

type createSubscriptionUseCase interface {
	Execute(ctx context.Context, cmd usecases.CreateSubscriptionCommand) (*usecases.CreateSubscriptionResult, error)
}

type getSubscriptionUseCase interface {
	Execute(ctx context.Context, query usecases.GetSubscriptionQuery) (*subdto.SubscriptionDTO, error)
	ExecuteBySID(ctx context.Context, sid string) (*subdto.SubscriptionDTO, error)
}

type listUserSubscriptionsUseCase interface {
	Execute(ctx context.Context, query usecases.ListUserSubscriptionsQuery) (*usecases.ListUserSubscriptionsResult, error)
}

type cancelSubscriptionUseCase interface {
	Execute(ctx context.Context, cmd usecases.CancelSubscriptionCommand) error
}

type deleteSubscriptionUseCase interface {
	Execute(ctx context.Context, subscriptionID uint) error
}

type changePlanUseCase interface {
	Execute(ctx context.Context, cmd usecases.ChangePlanCommand) error
}

type getSubscriptionUsageStatsUseCase interface {
	Execute(ctx context.Context, query usecases.GetSubscriptionUsageStatsQuery) (*usecases.GetSubscriptionUsageStatsResponse, error)
}

type resetSubscriptionLinkUseCase interface {
	Execute(ctx context.Context, cmd usecases.ResetSubscriptionLinkCommand) (*subdto.SubscriptionDTO, error)
}
