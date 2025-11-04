package subscription

import (
	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type Service struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	logger           logger.Interface
}

func NewService(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
	permissionService interface{},
	logger logger.Interface,
) *Service {
	return &Service{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

