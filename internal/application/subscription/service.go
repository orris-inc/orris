package subscription

import (
	"context"
	"fmt"

	"orris/internal/domain/subscription"
	permissionApp "orris/internal/application/permission"
	"orris/internal/shared/logger"
)

type Service struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	permissionService *permissionApp.Service
	logger           logger.Interface
}

func NewService(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
	permissionService *permissionApp.Service,
	logger logger.Interface,
) *Service {
	return &Service{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		permissionService: permissionService,
		logger:           logger,
	}
}

func (s *Service) SyncPermissions(ctx context.Context, subscriptionID uint) error {
	sub, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		s.logger.Errorw("failed to get subscription", "error", err, "subscription_id", subscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("subscription not found: %d", subscriptionID)
	}

	plan, err := s.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		s.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", sub.PlanID())
		return fmt.Errorf("failed to get subscription plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("subscription plan not found: %d", sub.PlanID())
	}

	if !sub.IsActive() {
		s.logger.Infow("subscription is not active, skipping permission sync",
			"subscription_id", subscriptionID,
			"user_id", sub.UserID(),
			"status", sub.Status())
		return nil
	}

	var features []string
	if plan.Features() != nil {
		features = plan.Features().Features
	}

	permissions := GetPermissionsForFeatures(features)
	if len(permissions) == 0 {
		s.logger.Infow("no permissions to sync for plan",
			"plan_id", plan.ID(),
			"plan_name", plan.Name())
		return nil
	}

	s.logger.Infow("syncing subscription permissions",
		"subscription_id", subscriptionID,
		"user_id", sub.UserID(),
		"plan_name", plan.Name(),
		"permission_count", len(permissions))

	return nil
}
