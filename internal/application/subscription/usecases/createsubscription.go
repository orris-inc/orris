package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/subscription"
	vo "orris/internal/domain/subscription/value_objects"
	"orris/internal/domain/user"
	"orris/internal/shared/logger"
)

type CreateSubscriptionCommand struct {
	UserID       uint
	PlanID       uint
	StartDate    time.Time
	AutoRenew    bool
	PaymentInfo  map[string]interface{}
	BillingCycle string // User selected billing cycle
}

type CreateSubscriptionResult struct {
	Subscription *subscription.Subscription
	Token        *subscription.SubscriptionToken
}

type CreateSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	tokenRepo        subscription.SubscriptionTokenRepository
	pricingRepo      subscription.PlanPricingRepository
	userRepo         user.Repository
	tokenGenerator   TokenGenerator
	logger           logger.Interface
}

func NewCreateSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
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
	// Verify target user exists
	targetUser, err := uc.userRepo.GetByID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get target user", "error", err, "user_id", cmd.UserID)
		return nil, fmt.Errorf("failed to get target user: %w", err)
	}
	if targetUser == nil {
		uc.logger.Warnw("target user not found", "user_id", cmd.UserID)
		return nil, fmt.Errorf("user not found")
	}

	plan, err := uc.planRepo.GetByID(ctx, cmd.PlanID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", cmd.PlanID)
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	if !plan.IsActive() {
		return nil, fmt.Errorf("subscription plan is not active")
	}

	// Determine the billing cycle to use
	billingCycle := plan.BillingCycle()

	// If user specified a billing cycle, validate and use it instead
	if cmd.BillingCycle != "" {
		// Parse the user-provided billing cycle
		userCycle, err := vo.ParseBillingCycle(cmd.BillingCycle)
		if err != nil {
			uc.logger.Warnw("invalid billing cycle provided", "error", err, "billing_cycle", cmd.BillingCycle)
			return nil, fmt.Errorf("invalid billing cycle: %w", err)
		}

		// Verify that pricing exists for this plan and billing cycle
		pricing, err := uc.pricingRepo.GetByPlanAndCycle(ctx, cmd.PlanID, userCycle)
		if err != nil {
			uc.logger.Warnw("failed to get pricing for billing cycle", "error", err, "plan_id", cmd.PlanID, "billing_cycle", userCycle)
			return nil, fmt.Errorf("pricing not available for selected billing cycle: %w", err)
		}

		if pricing == nil {
			uc.logger.Warnw("pricing not found for billing cycle", "plan_id", cmd.PlanID, "billing_cycle", userCycle)
			return nil, fmt.Errorf("pricing not found for selected billing cycle")
		}

		billingCycle = userCycle
		uc.logger.Infow("pricing selected for billing cycle", "plan_id", cmd.PlanID, "billing_cycle", billingCycle, "price", pricing.Price())
	}

	// Allow multiple active subscriptions per user
	// No restriction on creating new subscriptions

	startDate := cmd.StartDate
	if startDate.IsZero() {
		startDate = time.Now()
	}

	endDate := uc.calculateEndDate(startDate, billingCycle)

	sub, err := subscription.NewSubscription(cmd.UserID, cmd.PlanID, startDate, endDate, cmd.AutoRenew)
	if err != nil {
		uc.logger.Errorw("failed to create subscription aggregate", "error", err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	if plan.TrialDays() > 0 {

	}

	if err := uc.subscriptionRepo.Create(ctx, sub); err != nil {
		uc.logger.Errorw("failed to create subscription in database", "error", err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	token, err := uc.createDefaultToken(ctx, sub.ID())
	if err != nil {
		uc.logger.Warnw("failed to create default token", "error", err, "subscription_id", sub.ID())
	}

	uc.logger.Infow("subscription created successfully",
		"subscription_id", sub.ID(),
		"user_id", cmd.UserID,
		"plan_id", cmd.PlanID,
		"billing_cycle", billingCycle,
		"status", sub.Status(),
	)

	return &CreateSubscriptionResult{
		Subscription: sub,
		Token:        token,
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

func (uc *CreateSubscriptionUseCase) createDefaultToken(ctx context.Context, subscriptionID uint) (*subscription.SubscriptionToken, error) {
	_, hashedToken, err := uc.tokenGenerator.Generate("sub")
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	token, err := subscription.NewSubscriptionToken(subscriptionID, "Default Token", hashedToken, "sub", vo.TokenScopeFull, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	if err := uc.tokenRepo.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return token, nil
}
