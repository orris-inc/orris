package subscription

import (
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ServiceDDD provides domain-driven service operations for the subscription domain.
// It orchestrates use cases and coordinates cross-cutting concerns.
type ServiceDDD struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

// NewServiceDDD creates a new subscription domain service.
func NewServiceDDD(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ServiceDDD {
	return &ServiceDDD{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}
