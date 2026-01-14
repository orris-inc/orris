package services

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// QuotaCacheSyncService handles synchronization of subscription quota to cache
type QuotaCacheSyncService struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	quotaCache       cache.SubscriptionQuotaCache
	logger           logger.Interface
}

// NewQuotaCacheSyncService creates a new QuotaCacheSyncService
func NewQuotaCacheSyncService(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	quotaCache cache.SubscriptionQuotaCache,
	logger logger.Interface,
) *QuotaCacheSyncService {
	return &QuotaCacheSyncService{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		quotaCache:       quotaCache,
		logger:           logger,
	}
}

// SyncQuotaFromSubscription syncs quota information from a subscription to cache
func (s *QuotaCacheSyncService) SyncQuotaFromSubscription(ctx context.Context, sub *subscription.Subscription) error {
	if sub == nil {
		return nil
	}

	// Get plan information
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}

	if plan == nil {
		s.logger.Warnw("plan not found for subscription",
			"subscription_id", sub.ID(),
			"plan_id", sub.PlanID(),
		)
		return nil
	}

	// Get traffic limit
	trafficLimit, err := plan.GetTrafficLimit()
	if err != nil {
		return fmt.Errorf("failed to get traffic limit: %w", err)
	}

	// Build cached quota object
	quota := &cache.CachedQuota{
		Limit:       int64(trafficLimit),
		PeriodStart: sub.CurrentPeriodStart(),
		PeriodEnd:   sub.CurrentPeriodEnd(),
		PlanType:    plan.PlanType().String(),
		Suspended:   false, // Only set to false when syncing (active subscription)
	}

	// Write to cache
	if err := s.quotaCache.SetQuota(ctx, sub.ID(), quota); err != nil {
		return fmt.Errorf("failed to cache quota: %w", err)
	}

	s.logger.Debugw("subscription quota synced to cache",
		"subscription_id", sub.ID(),
		"limit", trafficLimit,
		"plan_type", quota.PlanType,
	)

	return nil
}

// InvalidateQuota removes quota information from cache
func (s *QuotaCacheSyncService) InvalidateQuota(ctx context.Context, subscriptionID uint) error {
	return s.quotaCache.InvalidateQuota(ctx, subscriptionID)
}

// LoadQuotaByID loads quota information from database and caches it
// This is used for lazy loading when cache misses
func (s *QuotaCacheSyncService) LoadQuotaByID(ctx context.Context, subscriptionID uint) (*cache.CachedQuota, error) {
	// Get subscription from database
	sub, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub == nil {
		return nil, nil
	}

	// Only cache active subscriptions
	if !sub.IsActive() {
		return nil, nil
	}

	// Get plan information
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	if plan == nil {
		return nil, nil
	}

	// Get traffic limit
	trafficLimit, err := plan.GetTrafficLimit()
	if err != nil {
		return nil, fmt.Errorf("failed to get traffic limit: %w", err)
	}

	// Build cached quota object
	quota := &cache.CachedQuota{
		Limit:       int64(trafficLimit),
		PeriodStart: sub.CurrentPeriodStart(),
		PeriodEnd:   sub.CurrentPeriodEnd(),
		PlanType:    plan.PlanType().String(),
		Suspended:   false,
	}

	// Cache the quota (ignore error, still return the quota)
	if err := s.quotaCache.SetQuota(ctx, subscriptionID, quota); err != nil {
		s.logger.Warnw("failed to cache loaded quota",
			"subscription_id", subscriptionID,
			"error", err,
		)
	}

	return quota, nil
}

// SetSuspended updates only the suspended status in cache
func (s *QuotaCacheSyncService) SetSuspended(ctx context.Context, subscriptionID uint, suspended bool) error {
	return s.quotaCache.SetSuspended(ctx, subscriptionID, suspended)
}
