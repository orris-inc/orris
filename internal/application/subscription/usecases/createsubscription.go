package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CreateSubscriptionCommand struct {
	UserID              uint   // Internal user ID (used if UserSID is empty)
	UserSID             string // Stripe-style user SID (takes precedence over UserID)
	PlanID              uint   // Internal plan ID (used if PlanSID is empty)
	PlanSID             string // Stripe-style plan SID (takes precedence over PlanID)
	StartDate           time.Time
	AutoRenew           bool
	PaymentInfo         map[string]interface{}
	BillingCycle        string // User selected billing cycle
	ActivateImmediately bool   // If true, activate subscription immediately (for admin use)
}

type CreateSubscriptionResult struct {
	Subscription *subscription.Subscription
	Token        *subscription.SubscriptionToken
	PlainToken   string // Plain token value, only available at creation time
}

type CreateSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	tokenRepo        subscription.SubscriptionTokenRepository
	pricingRepo      subscription.PlanPricingRepository
	userRepo         user.Repository
	tokenGenerator   TokenGenerator
	logger           logger.Interface
}

func NewCreateSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	tokenRepo subscription.SubscriptionTokenRepository,
	pricingRepo subscription.PlanPricingRepository,
	userRepo user.Repository,
	tokenGenerator TokenGenerator,
	logger logger.Interface,
) *CreateSubscriptionUseCase {
	return &CreateSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		tokenRepo:        tokenRepo,
		pricingRepo:      pricingRepo,
		userRepo:         userRepo,
		tokenGenerator:   tokenGenerator,
		logger:           logger,
	}
}

func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, cmd CreateSubscriptionCommand) (*CreateSubscriptionResult, error) {
	// Resolve user: prefer SID over internal ID
	var targetUser *user.User
	var err error
	userID := cmd.UserID

	if cmd.UserSID != "" {
		targetUser, err = uc.userRepo.GetBySID(ctx, cmd.UserSID)
		if err != nil {
			uc.logger.Errorw("failed to get user by SID", "error", err, "user_sid", cmd.UserSID)
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		if targetUser == nil {
			uc.logger.Warnw("user not found by SID", "user_sid", cmd.UserSID)
			return nil, fmt.Errorf("user not found")
		}
		userID = targetUser.ID()
	} else {
		targetUser, err = uc.userRepo.GetByID(ctx, cmd.UserID)
		if err != nil {
			uc.logger.Errorw("failed to get target user", "error", err, "user_id", cmd.UserID)
			return nil, fmt.Errorf("failed to get target user: %w", err)
		}
		if targetUser == nil {
			uc.logger.Warnw("target user not found", "user_id", cmd.UserID)
			return nil, fmt.Errorf("user not found")
		}
	}

	// Resolve plan: prefer SID over internal ID
	var plan *subscription.Plan
	planID := cmd.PlanID

	if cmd.PlanSID != "" {
		plan, err = uc.planRepo.GetBySID(ctx, cmd.PlanSID)
		if err != nil {
			uc.logger.Errorw("failed to get plan by SID", "error", err, "plan_sid", cmd.PlanSID)
			return nil, fmt.Errorf("failed to get plan: %w", err)
		}
		if plan == nil {
			uc.logger.Warnw("plan not found by SID", "plan_sid", cmd.PlanSID)
			return nil, fmt.Errorf("plan not found")
		}
		planID = plan.ID()
	} else {
		plan, err = uc.planRepo.GetByID(ctx, cmd.PlanID)
		if err != nil {
			uc.logger.Errorw("failed to get plan", "error", err, "plan_id", cmd.PlanID)
			return nil, fmt.Errorf("failed to get plan: %w", err)
		}
		if plan == nil {
			uc.logger.Warnw("plan not found", "plan_id", cmd.PlanID)
			return nil, fmt.Errorf("plan not found")
		}
	}

	if !plan.IsActive() {
		return nil, fmt.Errorf("plan is not active")
	}

	// BillingCycle is required since Plan no longer has a default billing cycle
	if cmd.BillingCycle == "" {
		return nil, fmt.Errorf("billing cycle is required")
	}

	// Parse and validate the billing cycle
	billingCycle, err := vo.ParseBillingCycle(cmd.BillingCycle)
	if err != nil {
		uc.logger.Warnw("invalid billing cycle provided", "error", err, "billing_cycle", cmd.BillingCycle)
		return nil, fmt.Errorf("invalid billing cycle: %w", err)
	}

	// Verify that pricing exists for this plan and billing cycle
	pricing, err := uc.pricingRepo.GetByPlanAndCycle(ctx, planID, billingCycle)
	if err != nil {
		uc.logger.Warnw("failed to get pricing for billing cycle", "error", err, "plan_id", planID, "billing_cycle", billingCycle)
		return nil, fmt.Errorf("pricing not available for selected billing cycle: %w", err)
	}

	if pricing == nil {
		uc.logger.Warnw("pricing not found for billing cycle", "plan_id", planID, "billing_cycle", billingCycle)
		return nil, fmt.Errorf("pricing not found for selected billing cycle")
	}

	uc.logger.Infow("pricing selected for billing cycle", "plan_id", planID, "billing_cycle", billingCycle, "price", pricing.Price())

	// Allow multiple active subscriptions per user
	// No restriction on creating new subscriptions

	startDate := cmd.StartDate
	if startDate.IsZero() {
		startDate = biztime.NowUTC()
	}

	// Calculate subscription end date based on billing cycle
	endDate := uc.calculateEndDate(startDate, billingCycle)

	sub, err := subscription.NewSubscription(userID, planID, startDate, endDate, cmd.AutoRenew)
	if err != nil {
		uc.logger.Errorw("failed to create subscription aggregate", "error", err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	if err := uc.subscriptionRepo.Create(ctx, sub); err != nil {
		uc.logger.Errorw("failed to create subscription in database", "error", err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Activate subscription immediately if requested (typically for admin-created subscriptions)
	if cmd.ActivateImmediately {
		if err := sub.Activate(); err != nil {
			uc.logger.Errorw("failed to activate subscription", "error", err, "subscription_id", sub.ID())
			return nil, fmt.Errorf("failed to activate subscription: %w", err)
		}
		if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
			uc.logger.Errorw("failed to update subscription after activation", "error", err, "subscription_id", sub.ID())
			return nil, fmt.Errorf("failed to update subscription: %w", err)
		}
		uc.logger.Infow("subscription activated immediately", "subscription_id", sub.ID())
	}

	// Create default token - this is critical for subscription usability
	// Token creation failure should fail the entire subscription creation
	token, plainToken, err := uc.createDefaultToken(ctx, sub.ID())
	if err != nil {
		uc.logger.Errorw("failed to create default token", "error", err, "subscription_id", sub.ID())

		// Rollback: delete the subscription to maintain data consistency
		if deleteErr := uc.subscriptionRepo.Delete(ctx, sub.ID()); deleteErr != nil {
			uc.logger.Errorw("failed to rollback subscription after token creation failure",
				"error", deleteErr,
				"subscription_id", sub.ID(),
			)
			// Return original error but log the rollback failure
		} else {
			uc.logger.Infow("subscription rolled back after token creation failure",
				"subscription_id", sub.ID(),
			)
		}

		return nil, fmt.Errorf("failed to create default token for subscription: %w", err)
	}

	uc.logger.Infow("subscription created successfully",
		"subscription_id", sub.ID(),
		"user_id", userID,
		"plan_id", planID,
		"billing_cycle", billingCycle,
		"status", sub.Status(),
	)

	return &CreateSubscriptionResult{
		Subscription: sub,
		Token:        token,
		PlainToken:   plainToken,
	}, nil
}

func (uc *CreateSubscriptionUseCase) calculateEndDate(startDate time.Time, billingCycle vo.BillingCycle) time.Time {
	switch billingCycle {
	case vo.BillingCycleMonthly:
		return startDate.AddDate(0, 1, 0)
	case vo.BillingCycleQuarterly:
		return startDate.AddDate(0, 3, 0)
	case vo.BillingCycleYearly:
		return startDate.AddDate(1, 0, 0)
	default:
		return startDate.AddDate(0, 1, 0)
	}
}

func (uc *CreateSubscriptionUseCase) createDefaultToken(ctx context.Context, subscriptionID uint) (*subscription.SubscriptionToken, string, error) {
	plainToken, hashedToken, err := uc.tokenGenerator.Generate("sub")
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	prefix := plainToken[:8]

	token, err := subscription.NewSubscriptionToken(subscriptionID, "Default Token", hashedToken, prefix, vo.TokenScopeFull, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create token: %w", err)
	}

	if err := uc.tokenRepo.Create(ctx, token); err != nil {
		return nil, "", fmt.Errorf("failed to save token: %w", err)
	}

	return token, plainToken, nil
}
