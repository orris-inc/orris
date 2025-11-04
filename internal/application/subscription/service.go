package subscription

import (
	"context"

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

func (s *Service) SyncPermissions(ctx context.Context, subscriptionID uint) error {
	s.logger.Infow("permission sync is no longer needed in simplified permission system",
		"subscription_id", subscriptionID)
	return nil
}
