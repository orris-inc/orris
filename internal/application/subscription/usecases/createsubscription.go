package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CreateSubscriptionCommand struct {
	UserID              uint
	PlanID              uint
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
	subscriptionRepo    subscription.SubscriptionRepository
	planRepo            subscription.PlanRepository
	tokenRepo           subscription.SubscriptionTokenRepository
	pricingRepo         subscription.PlanPricingRepository
	userRepo            user.Repository
	tokenGenerator      TokenGenerator
	planEntitlementRepo subscription.EntitlementRepository
	entitlementRepo     entitlement.Repository
	logger              logger.Interface
}

func NewCreateSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	tokenRepo subscription.SubscriptionTokenRepository,
	pricingRepo subscription.PlanPricingRepository,
	userRepo user.Repository,
	tokenGenerator TokenGenerator,
	planEntitlementRepo subscription.EntitlementRepository,
	entitlementRepo entitlement.Repository,
	logger logger.Interface,
) *CreateSubscriptionUseCase {
	return &CreateSubscriptionUseCase{
		subscriptionRepo:    subscriptionRepo,
		planRepo:            planRepo,
		tokenRepo:           tokenRepo,
		pricingRepo:         pricingRepo,
		userRepo:            userRepo,
		tokenGenerator:      tokenGenerator,
		planEntitlementRepo: planEntitlementRepo,
		entitlementRepo:     entitlementRepo,
		logger:              logger,
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
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", cmd.PlanID)
		return nil, fmt.Errorf("failed to get plan: %w", err)
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
	pricing, err := uc.pricingRepo.GetByPlanAndCycle(ctx, cmd.PlanID, billingCycle)
	if err != nil {
		uc.logger.Warnw("failed to get pricing for billing cycle", "error", err, "plan_id", cmd.PlanID, "billing_cycle", billingCycle)
		return nil, fmt.Errorf("pricing not available for selected billing cycle: %w", err)
	}

	if pricing == nil {
		uc.logger.Warnw("pricing not found for billing cycle", "plan_id", cmd.PlanID, "billing_cycle", billingCycle)
		return nil, fmt.Errorf("pricing not found for selected billing cycle")
	}

	uc.logger.Infow("pricing selected for billing cycle", "plan_id", cmd.PlanID, "billing_cycle", billingCycle, "price", pricing.Price())

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

		// Grant entitlements based on the plan's resources
		if err := uc.grantPlanEntitlements(ctx, cmd.UserID, sub.ID(), cmd.PlanID, endDate); err != nil {
			uc.logger.Errorw("failed to grant plan entitlements", "error", err, "subscription_id", sub.ID(), "user_id", cmd.UserID)
			return nil, fmt.Errorf("failed to grant entitlements: %w", err)
		}
	}

	token, plainToken, err := uc.createDefaultToken(ctx, sub.ID())
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

// grantPlanEntitlements grants entitlements to the user based on the plan's associated resources
func (uc *CreateSubscriptionUseCase) grantPlanEntitlements(
	ctx context.Context,
	userID uint,
	subscriptionID uint,
	planID uint,
	expiresAt time.Time,
) error {
	// Get all plan entitlements (plan-resource associations)
	planEntitlements, err := uc.planEntitlementRepo.GetByPlan(ctx, planID)
	if err != nil {
		return fmt.Errorf("failed to get plan entitlements: %w", err)
	}

	if len(planEntitlements) == 0 {
		uc.logger.Infow("no plan entitlements to grant", "plan_id", planID)
		return nil
	}

	// Create user entitlements for each plan resource
	userEntitlements := make([]*entitlement.Entitlement, 0, len(planEntitlements))
	for _, planEnt := range planEntitlements {
		// Map subscription.EntitlementResourceType to entitlement.ResourceType
		var resourceType entitlement.ResourceType
		switch planEnt.ResourceType() {
		case subscription.EntitlementResourceTypeNode:
			resourceType = entitlement.ResourceTypeNode
		case subscription.EntitlementResourceTypeForwardAgent:
			resourceType = entitlement.ResourceTypeForwardAgent
		default:
			uc.logger.Warnw("unknown plan resource type, skipping",
				"resource_type", planEnt.ResourceType(),
				"resource_id", planEnt.ResourceID(),
			)
			continue
		}

		// Create user entitlement with subscription as source
		userEnt, err := entitlement.NewEntitlement(
			entitlement.SubjectTypeUser,
			userID,
			resourceType,
			planEnt.ResourceID(),
			entitlement.SourceTypeSubscription,
			subscriptionID,
			&expiresAt,
		)
		if err != nil {
			uc.logger.Errorw("failed to create user entitlement",
				"error", err,
				"user_id", userID,
				"resource_type", resourceType,
				"resource_id", planEnt.ResourceID(),
			)
			return fmt.Errorf("failed to create user entitlement: %w", err)
		}

		userEntitlements = append(userEntitlements, userEnt)
	}

	// Batch create user entitlements
	if len(userEntitlements) > 0 {
		if err := uc.entitlementRepo.BatchCreate(ctx, userEntitlements); err != nil {
			return fmt.Errorf("failed to batch create user entitlements: %w", err)
		}

		uc.logger.Infow("granted plan entitlements to user",
			"user_id", userID,
			"subscription_id", subscriptionID,
			"plan_id", planID,
			"entitlement_count", len(userEntitlements),
		)
	}

	return nil
}
