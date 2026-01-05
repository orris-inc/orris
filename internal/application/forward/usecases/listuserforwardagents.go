package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ListUserForwardAgentsQuery represents the input for listing user's accessible forward agents.
type ListUserForwardAgentsQuery struct {
	UserID   uint
	Page     int
	PageSize int
	Name     string
	Status   string
	OrderBy  string
	Order    string
}

// ListUserForwardAgentsResult represents the output of listing user's forward agents.
type ListUserForwardAgentsResult struct {
	Agents []*dto.UserForwardAgentDTO `json:"agents"`
	Total  int64                      `json:"total"`
	Page   int                        `json:"page"`
	Pages  int                        `json:"pages"`
}

// ListUserForwardAgentsUseCase handles listing forward agents accessible to a user.
type ListUserForwardAgentsUseCase struct {
	agentRepo         forward.AgentRepository
	subscriptionRepo  subscription.SubscriptionRepository
	planRepo          subscription.PlanRepository
	resourceGroupRepo resource.Repository
	logger            logger.Interface
}

// NewListUserForwardAgentsUseCase creates a new ListUserForwardAgentsUseCase.
func NewListUserForwardAgentsUseCase(
	agentRepo forward.AgentRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	resourceGroupRepo resource.Repository,
	logger logger.Interface,
) *ListUserForwardAgentsUseCase {
	return &ListUserForwardAgentsUseCase{
		agentRepo:         agentRepo,
		subscriptionRepo:  subscriptionRepo,
		planRepo:          planRepo,
		resourceGroupRepo: resourceGroupRepo,
		logger:            logger,
	}
}

// Execute retrieves a list of forward agents accessible to a user.
// The access is determined by: User -> Subscription -> Plan(forward) -> ResourceGroup -> ForwardAgent
func (uc *ListUserForwardAgentsUseCase) Execute(ctx context.Context, query ListUserForwardAgentsQuery) (*ListUserForwardAgentsResult, error) {
	uc.logger.Infow("executing list user forward agents use case",
		"user_id", query.UserID,
		"page", query.Page,
		"page_size", query.PageSize,
	)

	// Validate user ID
	if query.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}

	// Set defaults
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	// Step 1: Get user's active subscriptions
	subscriptions, err := uc.subscriptionRepo.GetActiveByUserID(ctx, query.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get user subscriptions",
			"user_id", query.UserID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get user subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		uc.logger.Debugw("user has no active subscriptions", "user_id", query.UserID)
		return &ListUserForwardAgentsResult{
			Agents: []*dto.UserForwardAgentDTO{},
			Total:  0,
			Page:   query.Page,
			Pages:  0,
		}, nil
	}

	// Step 2: Collect plan IDs from subscriptions
	planIDs := make([]uint, 0, len(subscriptions))
	for _, sub := range subscriptions {
		planIDs = append(planIDs, sub.PlanID())
	}

	// Step 3: Get plans and filter forward type plans
	plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
	if err != nil {
		uc.logger.Errorw("failed to get plans",
			"user_id", query.UserID,
			"plan_ids", planIDs,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}

	forwardPlanIDs := make([]uint, 0, len(plans))
	for _, plan := range plans {
		if plan.PlanType().IsForward() {
			forwardPlanIDs = append(forwardPlanIDs, plan.ID())
		}
	}

	if len(forwardPlanIDs) == 0 {
		uc.logger.Debugw("user has no forward type subscriptions", "user_id", query.UserID)
		return &ListUserForwardAgentsResult{
			Agents: []*dto.UserForwardAgentDTO{},
			Total:  0,
			Page:   query.Page,
			Pages:  0,
		}, nil
	}

	// Step 4: Get active resource groups for these plans
	groupIDs := make([]uint, 0)
	groupInfoMap := make(dto.GroupInfoMap)

	for _, planID := range forwardPlanIDs {
		groups, err := uc.resourceGroupRepo.GetByPlanID(ctx, planID)
		if err != nil {
			uc.logger.Warnw("failed to get resource groups for plan",
				"plan_id", planID,
				"error", err,
			)
			continue
		}

		for _, group := range groups {
			if group.IsActive() {
				groupIDs = append(groupIDs, group.ID())
				groupInfoMap[group.ID()] = &dto.GroupInfo{
					SID:  group.SID(), // SID already contains prefix (rg_xxxxxxxx)
					Name: group.Name(),
				}
			}
		}
	}

	if len(groupIDs) == 0 {
		uc.logger.Debugw("no active resource groups found for user", "user_id", query.UserID)
		return &ListUserForwardAgentsResult{
			Agents: []*dto.UserForwardAgentDTO{},
			Total:  0,
			Page:   query.Page,
			Pages:  0,
		}, nil
	}

	// Step 5: Query forward agents with group filter
	filter := forward.AgentListFilter{
		Page:     query.Page,
		PageSize: query.PageSize,
		Name:     query.Name,
		Status:   query.Status,
		OrderBy:  query.OrderBy,
		Order:    query.Order,
		GroupIDs: groupIDs,
	}

	agents, total, err := uc.agentRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward agents",
			"user_id", query.UserID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list forward agents: %w", err)
	}

	// Calculate total pages
	pages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		pages++
	}

	// Step 6: Convert to user-facing DTOs
	dtos := dto.ToUserForwardAgentDTOs(agents, groupInfoMap)

	uc.logger.Infow("user forward agents listed successfully",
		"user_id", query.UserID,
		"total", total,
	)

	return &ListUserForwardAgentsResult{
		Agents: dtos,
		Total:  total,
		Page:   query.Page,
		Pages:  pages,
	}, nil
}
