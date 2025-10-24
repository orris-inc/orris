package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/subscription"
	vo "orris/internal/domain/subscription/value_objects"
	"orris/internal/shared/logger"
)

type CreateSubscriptionCommand struct {
	UserID      uint
	PlanID      uint
	StartDate   time.Time
	AutoRenew   bool
	PaymentInfo map[string]interface{}
}

type CreateSubscriptionResult struct {
	Subscription *subscription.Subscription
	Token        *subscription.SubscriptionToken
}

type CreateSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	tokenRepo        subscription.SubscriptionTokenRepository
	tokenGenerator   TokenGenerator
	logger           logger.Interface
}

func NewCreateSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
	tokenRepo subscription.SubscriptionTokenRepository,
	tokenGenerator TokenGenerator,
	logger logger.Interface,
) *CreateSubscriptionUseCase {
	return &CreateSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		tokenRepo:        tokenRepo,
		tokenGenerator:   tokenGenerator,
		logger:           logger,
	}
}

func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, cmd CreateSubscriptionCommand) (*CreateSubscriptionResult, error) {
	plan, err := uc.planRepo.GetByID(ctx, cmd.PlanID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", cmd.PlanID)
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	if !plan.IsActive() {
		return nil, fmt.Errorf("subscription plan is not active")
	}

	activeSubscription, err := uc.subscriptionRepo.GetActiveByUserID(ctx, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get active subscriptions", "error", err, "user_id", cmd.UserID)
		return nil, fmt.Errorf("failed to check active subscriptions: %w", err)
	}

	if activeSubscription != nil {
		return nil, fmt.Errorf("user already has an active subscription")
	}

	startDate := cmd.StartDate
	if startDate.IsZero() {
		startDate = time.Now()
	}

	endDate := uc.calculateEndDate(startDate, plan.BillingCycle())

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
