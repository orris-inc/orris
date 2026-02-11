package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetUserNodeUsageQuery represents the input for getting user node usage.
type GetUserNodeUsageQuery struct {
	UserID uint
}

// GetUserNodeUsageResult represents the user's node usage and quota information.
type GetUserNodeUsageResult struct {
	NodeCount int `json:"node_count"`
	NodeLimit int `json:"node_limit"` // 0 means unlimited
}

// GetUserNodeUsageExecutor defines the interface for getting user node usage.
type GetUserNodeUsageExecutor interface {
	Execute(ctx context.Context, query GetUserNodeUsageQuery) (*GetUserNodeUsageResult, error)
}

// GetUserNodeUsageUseCase handles getting user node usage.
type GetUserNodeUsageUseCase struct {
	nodeRepo         node.NodeRepository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

// NewGetUserNodeUsageUseCase creates a new GetUserNodeUsageUseCase.
func NewGetUserNodeUsageUseCase(
	nodeRepo node.NodeRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *GetUserNodeUsageUseCase {
	return &GetUserNodeUsageUseCase{
		nodeRepo:         nodeRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

// Execute retrieves node usage statistics for a user.
func (uc *GetUserNodeUsageUseCase) Execute(ctx context.Context, query GetUserNodeUsageQuery) (*GetUserNodeUsageResult, error) {
	uc.logger.Debugw("executing get user node usage use case", "user_id", query.UserID)

	// Validate user ID
	if query.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}

	// Get user's active subscriptions
	subscriptions, err := uc.subscriptionRepo.GetActiveByUserID(ctx, query.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get active subscriptions", "user_id", query.UserID, "error", err)
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	// Initialize limits
	maxNodeLimit := 0
	hasUnlimitedNodes := false

	// Batch fetch all plans to avoid N+1 queries
	planIDs := make([]uint, 0, len(subscriptions))
	for _, sub := range subscriptions {
		planIDs = append(planIDs, sub.PlanID())
	}
	plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
	if err != nil {
		uc.logger.Errorw("failed to batch fetch plans", "user_id", query.UserID, "error", err)
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}
	planMap := make(map[uint]*subscription.Plan, len(plans))
	for _, plan := range plans {
		planMap[plan.ID()] = plan
	}

	// Find the highest limit among all active subscriptions
	for _, sub := range subscriptions {
		plan, ok := planMap[sub.PlanID()]
		if !ok {
			uc.logger.Warnw("plan not found for subscription", "subscription_id", sub.ID(), "plan_id", sub.PlanID())
			continue
		}

		// Check if plan is node type
		if !plan.PlanType().IsNode() {
			continue
		}

		// Check node limit
		if !plan.HasNodeLimit() {
			hasUnlimitedNodes = true
		} else if !hasUnlimitedNodes {
			limit := plan.GetNodeLimit()
			if limit > maxNodeLimit {
				maxNodeLimit = limit
			}
		}
	}

	// Apply unlimited flag - 0 represents unlimited
	if hasUnlimitedNodes {
		maxNodeLimit = 0
	}

	// Get current usage
	nodeCount, err := uc.nodeRepo.CountByUserID(ctx, query.UserID)
	if err != nil {
		uc.logger.Errorw("failed to count user nodes", "user_id", query.UserID, "error", err)
		return nil, fmt.Errorf("failed to get node count: %w", err)
	}

	result := &GetUserNodeUsageResult{
		NodeCount: int(nodeCount),
		NodeLimit: maxNodeLimit,
	}

	uc.logger.Debugw("user node usage retrieved successfully",
		"user_id", query.UserID,
		"node_count", nodeCount,
		"node_limit", maxNodeLimit,
	)

	return result, nil
}
