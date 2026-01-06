// Package usecases provides application-level use cases for subscription management.
package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RawDataRetentionDays is the number of days raw usage data is retained.
// For periods longer than this, we need to query the aggregated stats table.
const RawDataRetentionDays = 90

// QuotaCheckResult represents the quota usage status for a subscription.
type QuotaCheckResult struct {
	SubscriptionID  uint      // Internal subscription ID
	SubscriptionSID string    // Stripe-style subscription ID
	PlanType        string    // Plan type (node, forward, hybrid)
	UsedBytes       uint64    // Total traffic used in current period
	LimitBytes      uint64    // Traffic limit (0 = unlimited)
	PeriodStart     time.Time // Current billing period start
	PeriodEnd       time.Time // Current billing period end
	IsExceeded      bool      // Whether quota is exceeded
	RemainingBytes  uint64    // Remaining traffic (0 if exceeded or unlimited)
}

// QuotaService provides unified quota calculation for subscriptions.
type QuotaService interface {
	// GetSubscriptionQuota returns the quota usage for a single subscription.
	GetSubscriptionQuota(ctx context.Context, subscriptionID uint) (*QuotaCheckResult, error)

	// GetUserForwardQuota returns quota usage for all Forward-type subscriptions of a user.
	GetUserForwardQuota(ctx context.Context, userID uint) ([]*QuotaCheckResult, error)

	// GetUserNodeQuota returns quota usage for all Node-type subscriptions of a user.
	GetUserNodeQuota(ctx context.Context, userID uint) ([]*QuotaCheckResult, error)

	// CheckUserForwardQuotaExceeded checks if user's Forward quota is exceeded.
	// Returns true if all Forward subscriptions have exceeded their quota.
	CheckUserForwardQuotaExceeded(ctx context.Context, userID uint) (bool, error)
}

// QuotaServiceImpl implements the QuotaService interface.
type QuotaServiceImpl struct {
	subscriptionRepo      subscription.SubscriptionRepository
	subscriptionUsageRepo subscription.SubscriptionUsageRepository
	usageStatsRepo        subscription.SubscriptionUsageStatsRepository
	planRepo              subscription.PlanRepository
	logger                logger.Interface
}

// NewQuotaService creates a new QuotaServiceImpl instance.
func NewQuotaService(
	subscriptionRepo subscription.SubscriptionRepository,
	subscriptionUsageRepo subscription.SubscriptionUsageRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *QuotaServiceImpl {
	return &QuotaServiceImpl{
		subscriptionRepo:      subscriptionRepo,
		subscriptionUsageRepo: subscriptionUsageRepo,
		usageStatsRepo:        usageStatsRepo,
		planRepo:              planRepo,
		logger:                logger,
	}
}

// GetSubscriptionQuota returns the quota usage for a single subscription.
func (s *QuotaServiceImpl) GetSubscriptionQuota(ctx context.Context, subscriptionID uint) (*QuotaCheckResult, error) {
	sub, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		s.logger.Errorw("failed to get subscription",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return nil, err
	}

	if sub == nil {
		s.logger.Warnw("subscription not found", "subscription_id", subscriptionID)
		return nil, nil
	}

	plan, err := s.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		s.logger.Errorw("failed to get plan",
			"subscription_id", subscriptionID,
			"plan_id", sub.PlanID(),
			"error", err,
		)
		return nil, err
	}

	if plan == nil {
		s.logger.Warnw("plan not found for subscription",
			"subscription_id", subscriptionID,
			"plan_id", sub.PlanID(),
		)
		return nil, nil
	}

	return s.buildQuotaResult(ctx, sub, plan)
}

// GetUserForwardQuota returns quota usage for all Forward-type subscriptions of a user.
func (s *QuotaServiceImpl) GetUserForwardQuota(ctx context.Context, userID uint) ([]*QuotaCheckResult, error) {
	return s.getUserQuotaByPlanType(ctx, userID, vo.PlanTypeForward)
}

// GetUserNodeQuota returns quota usage for all Node-type subscriptions of a user.
func (s *QuotaServiceImpl) GetUserNodeQuota(ctx context.Context, userID uint) ([]*QuotaCheckResult, error) {
	return s.getUserQuotaByPlanType(ctx, userID, vo.PlanTypeNode)
}

// CheckUserForwardQuotaExceeded checks if user's Forward quota is exceeded.
func (s *QuotaServiceImpl) CheckUserForwardQuotaExceeded(ctx context.Context, userID uint) (bool, error) {
	quotas, err := s.GetUserForwardQuota(ctx, userID)
	if err != nil {
		return false, err
	}

	// If no Forward subscriptions, consider as exceeded (no access)
	if len(quotas) == 0 {
		return true, nil
	}

	// Check if at least one subscription has remaining quota
	for _, q := range quotas {
		// Unlimited quota
		if q.LimitBytes == 0 {
			return false, nil
		}
		// Has remaining quota
		if !q.IsExceeded {
			return false, nil
		}
	}

	// All subscriptions exceeded
	return true, nil
}

// getUserQuotaByPlanType returns quota usage for subscriptions of a specific plan type.
func (s *QuotaServiceImpl) getUserQuotaByPlanType(
	ctx context.Context,
	userID uint,
	planType vo.PlanType,
) ([]*QuotaCheckResult, error) {
	subs, err := s.subscriptionRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		s.logger.Errorw("failed to get active subscriptions",
			"user_id", userID,
			"error", err,
		)
		return nil, err
	}

	if len(subs) == 0 {
		return nil, nil
	}

	// Collect plan IDs for batch lookup
	planIDs := make([]uint, 0, len(subs))
	for _, sub := range subs {
		planIDs = append(planIDs, sub.PlanID())
	}

	plans, err := s.planRepo.GetByIDs(ctx, planIDs)
	if err != nil {
		s.logger.Errorw("failed to get plans",
			"user_id", userID,
			"plan_ids", planIDs,
			"error", err,
		)
		return nil, err
	}

	// Build plan ID to plan map
	planMap := make(map[uint]*subscription.Plan)
	for _, p := range plans {
		planMap[p.ID()] = p
	}

	// Build quota results for matching plan type
	var results []*QuotaCheckResult
	for _, sub := range subs {
		plan, ok := planMap[sub.PlanID()]
		if !ok {
			s.logger.Warnw("plan not found for subscription",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
			)
			continue
		}

		// Filter by plan type
		// Hybrid plans are included for both Node and Forward
		if plan.PlanType() != planType && !plan.PlanType().IsHybrid() {
			continue
		}

		result, err := s.buildQuotaResult(ctx, sub, plan)
		if err != nil {
			s.logger.Warnw("failed to build quota result",
				"subscription_id", sub.ID(),
				"error", err,
			)
			continue
		}

		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

// buildQuotaResult constructs a QuotaCheckResult for a subscription.
func (s *QuotaServiceImpl) buildQuotaResult(
	ctx context.Context,
	sub *subscription.Subscription,
	plan *subscription.Plan,
) (*QuotaCheckResult, error) {
	periodStart := sub.CurrentPeriodStart()
	periodEnd := sub.CurrentPeriodEnd()

	// Get traffic limit from plan
	limitBytes, err := plan.GetTrafficLimit()
	if err != nil {
		s.logger.Warnw("failed to get traffic limit from plan",
			"subscription_id", sub.ID(),
			"plan_id", plan.ID(),
			"error", err,
		)
		limitBytes = 0 // Treat as unlimited on error
	}

	// Determine resource type based on plan type
	resourceType := s.getResourceTypeForPlan(plan.PlanType())

	// Calculate period usage
	usedBytes, err := s.calculatePeriodUsage(
		ctx,
		[]uint{sub.ID()},
		resourceType,
		periodStart,
		periodEnd,
	)
	if err != nil {
		s.logger.Errorw("failed to calculate period usage",
			"subscription_id", sub.ID(),
			"period_start", periodStart,
			"period_end", periodEnd,
			"error", err,
		)
		return nil, err
	}

	// Calculate quota status
	isExceeded := false
	var remainingBytes uint64

	if limitBytes > 0 {
		if usedBytes >= limitBytes {
			isExceeded = true
			remainingBytes = 0
		} else {
			remainingBytes = limitBytes - usedBytes
		}
	}
	// If limitBytes is 0 (unlimited), isExceeded remains false and remainingBytes remains 0

	return &QuotaCheckResult{
		SubscriptionID:  sub.ID(),
		SubscriptionSID: sub.SID(),
		PlanType:        plan.PlanType().String(),
		UsedBytes:       usedBytes,
		LimitBytes:      limitBytes,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		IsExceeded:      isExceeded,
		RemainingBytes:  remainingBytes,
	}, nil
}

// getResourceTypeForPlan returns the appropriate resource type for a plan type.
func (s *QuotaServiceImpl) getResourceTypeForPlan(planType vo.PlanType) string {
	switch planType {
	case vo.PlanTypeNode:
		return subscription.ResourceTypeNode.String()
	case vo.PlanTypeForward:
		return subscription.ResourceTypeForwardRule.String()
	case vo.PlanTypeHybrid:
		// For hybrid plans, we track both types separately
		// This method returns the primary type; caller may need to query both
		return subscription.ResourceTypeNode.String()
	default:
		return subscription.ResourceTypeNode.String()
	}
}

// calculatePeriodUsage calculates total usage for subscriptions within a billing period.
// For long periods (exceeding raw data retention), it combines data from both
// the raw usage table and the aggregated stats table.
func (s *QuotaServiceImpl) calculatePeriodUsage(
	ctx context.Context,
	subscriptionIDs []uint,
	resourceType string,
	periodStart time.Time,
	periodEnd time.Time,
) (uint64, error) {
	if len(subscriptionIDs) == 0 {
		return 0, nil
	}

	now := biztime.NowUTC()

	// If period end is zero (no end limit), use current time
	if periodEnd.IsZero() {
		periodEnd = now
	}

	// Calculate the retention cutoff point
	// Raw data is retained for RawDataRetentionDays days
	retentionCutoff := now.AddDate(0, 0, -RawDataRetentionDays)

	// Case 1: Period is entirely within raw data retention window
	// Query only the raw usage table
	if periodStart.After(retentionCutoff) || periodStart.Equal(retentionCutoff) {
		usage, err := s.subscriptionUsageRepo.GetTotalUsageBySubscriptionIDs(
			ctx, resourceType, subscriptionIDs, periodStart, periodEnd,
		)
		if err != nil {
			return 0, err
		}
		return usage.Total, nil
	}

	// Case 2: Period starts before retention cutoff
	// Need to query both aggregated stats and raw data
	var totalUsage uint64

	// 2a: Query aggregated stats for the archived period (periodStart to retentionCutoff)
	statsUsage, err := s.usageStatsRepo.GetTotalBySubscriptionIDs(
		ctx, subscriptionIDs, subscription.GranularityDaily, periodStart, retentionCutoff,
	)
	if err != nil {
		s.logger.Warnw("failed to get aggregated stats, falling back to raw data only",
			"subscription_ids", subscriptionIDs,
			"period_start", periodStart,
			"retention_cutoff", retentionCutoff,
			"error", err,
		)
		// Fall back to raw data query only
		// This may undercount for very old periods, but is better than failing
	} else if statsUsage != nil {
		totalUsage += statsUsage.Total
	}

	// 2b: Query raw usage table for the recent period (retentionCutoff to periodEnd)
	rawUsage, err := s.subscriptionUsageRepo.GetTotalUsageBySubscriptionIDs(
		ctx, resourceType, subscriptionIDs, retentionCutoff, periodEnd,
	)
	if err != nil {
		return 0, err
	}
	totalUsage += rawUsage.Total

	return totalUsage, nil
}
