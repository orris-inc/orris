package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/user/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetDashboardQuery represents the query parameters for user dashboard
type GetDashboardQuery struct {
	UserID uint
}

// GetDashboardUseCase handles retrieving user dashboard data
type GetDashboardUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	usageRepo        subscription.SubscriptionUsageRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

// NewGetDashboardUseCase creates a new GetDashboardUseCase
func NewGetDashboardUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	usageRepo subscription.SubscriptionUsageRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *GetDashboardUseCase {
	return &GetDashboardUseCase{
		subscriptionRepo: subscriptionRepo,
		usageRepo:        usageRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

// Execute retrieves user dashboard data including subscriptions and usage
func (uc *GetDashboardUseCase) Execute(
	ctx context.Context,
	query GetDashboardQuery,
) (*dto.DashboardResponse, error) {
	uc.logger.Infow("fetching user dashboard", "user_id", query.UserID)

	if query.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}

	// Get user's subscriptions
	subscriptions, err := uc.subscriptionRepo.GetByUserID(ctx, query.UserID)
	if err != nil {
		uc.logger.Errorw("failed to fetch user subscriptions", "user_id", query.UserID, "error", err)
		return nil, errors.NewInternalError("failed to fetch subscriptions")
	}

	// Prepare response
	response := &dto.DashboardResponse{
		Subscriptions: make([]*dto.DashboardSubscriptionDTO, 0, len(subscriptions)),
		TotalUsage: &dto.UsageSummary{
			Upload:   0,
			Download: 0,
			Total:    0,
		},
	}

	// Collect unique plan IDs for batch fetch
	planIDSet := make(map[uint]struct{}, len(subscriptions))
	for _, sub := range subscriptions {
		planIDSet[sub.PlanID()] = struct{}{}
	}
	planIDs := make([]uint, 0, len(planIDSet))
	for id := range planIDSet {
		planIDs = append(planIDs, id)
	}

	// Batch fetch plans
	planMap := make(map[uint]*subscription.Plan)
	if len(planIDs) > 0 {
		plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch plans", "error", err)
		} else {
			for _, plan := range plans {
				planMap[plan.ID()] = plan
			}
		}
	}

	// Process each subscription
	for _, sub := range subscriptions {
		// Get usage for current period using aggregation
		periodStart := sub.CurrentPeriodStart()
		periodEnd := biztime.EndOfDayUTC(sub.CurrentPeriodEnd())

		usageSummary, err := uc.usageRepo.GetTotalUsageBySubscriptionID(ctx, sub.ID(), periodStart, periodEnd)
		if err != nil {
			uc.logger.Warnw("failed to fetch subscription usage",
				"subscription_id", sub.ID(),
				"error", err,
			)
		}

		// Calculate subscription usage summary
		subUsage := &dto.UsageSummary{
			Upload:   0,
			Download: 0,
			Total:    0,
		}
		if usageSummary != nil {
			subUsage.Upload = usageSummary.Upload
			subUsage.Download = usageSummary.Download
			subUsage.Total = usageSummary.Total
		}

		// Add to total usage
		response.TotalUsage.Upload += subUsage.Upload
		response.TotalUsage.Download += subUsage.Download
		response.TotalUsage.Total += subUsage.Total

		// Build subscription DTO
		subDTO := &dto.DashboardSubscriptionDTO{
			SID:                sub.SID(),
			Status:             sub.Status().String(),
			CurrentPeriodStart: sub.CurrentPeriodStart(),
			CurrentPeriodEnd:   sub.CurrentPeriodEnd(),
			IsActive:           sub.IsActive(),
			Usage:              subUsage,
		}

		// Add plan info if available
		if plan, ok := planMap[sub.PlanID()]; ok {
			var limits map[string]interface{}
			if plan.Features() != nil {
				limits = plan.Features().Limits
			}
			subDTO.Plan = &dto.DashboardPlanDTO{
				SID:      plan.SID(),
				Name:     plan.Name(),
				PlanType: plan.PlanType().String(),
				Limits:   limits,
			}
		}

		response.Subscriptions = append(response.Subscriptions, subDTO)
	}

	uc.logger.Infow("user dashboard fetched successfully",
		"user_id", query.UserID,
		"subscriptions_count", len(response.Subscriptions),
	)

	return response, nil
}
