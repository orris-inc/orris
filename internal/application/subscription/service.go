package subscription

import (
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type Service struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	logger           logger.Interface
}

func NewService(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *Service {
	return &Service{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}
