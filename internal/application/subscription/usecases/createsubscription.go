package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/db"
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
	subscriptionRepo     subscription.SubscriptionRepository
	planRepo             subscription.PlanRepository
	tokenRepo            subscription.SubscriptionTokenRepository
	pricingRepo          subscription.PlanPricingRepository
	userRepo             user.Repository
	tokenGenerator       TokenGenerator
	txMgr                *db.TransactionManager
	subscriptionNotifier SubscriptionChangeNotifier // Optional: for notifying node agents
	logger               logger.Interface
}

func NewCreateSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	tokenRepo subscription.SubscriptionTokenRepository,
	pricingRepo subscription.PlanPricingRepository,
	userRepo user.Repository,
	tokenGenerator TokenGenerator,
	txMgr *db.TransactionManager,
	logger logger.Interface,
) *CreateSubscriptionUseCase {
	return &CreateSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		tokenRepo:        tokenRepo,
		pricingRepo:      pricingRepo,
		userRepo:         userRepo,
		tokenGenerator:   tokenGenerator,
		txMgr:            txMgr,
		logger:           logger,
	}
}

// SetSubscriptionNotifier sets the subscription change notifier (optional).
func (uc *CreateSubscriptionUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
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
		return nil, err
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

	sub, err := subscription.NewSubscription(userID, planID, startDate, endDate, cmd.AutoRenew, &billingCycle)
	if err != nil {
		uc.logger.Errorw("failed to create subscription aggregate", "error", err)
		return nil, err
	}

	// Use database transaction to ensure subscription + token creation is atomic.
	// If any step fails, the entire operation is rolled back automatically.
	var token *subscription.SubscriptionToken
	var plainToken string

	txErr := uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.subscriptionRepo.Create(txCtx, sub); err != nil {
			uc.logger.Errorw("failed to create subscription in database", "error", err)
			return fmt.Errorf("failed to create subscription: %w", err)
		}

		// Activate subscription immediately if requested (typically for admin-created subscriptions)
		if cmd.ActivateImmediately {
			if err := sub.Activate(); err != nil {
				uc.logger.Errorw("failed to activate subscription", "error", err, "subscription_id", sub.ID())
				return err
			}
			if err := uc.subscriptionRepo.Update(txCtx, sub); err != nil {
				uc.logger.Errorw("failed to update subscription after activation", "error", err, "subscription_id", sub.ID())
				return fmt.Errorf("failed to update subscription: %w", err)
			}
			uc.logger.Infow("subscription activated immediately", "subscription_id", sub.ID())
		}

		// Create default token within the same transaction
		var tokenErr error
		token, plainToken, tokenErr = uc.createDefaultToken(txCtx, sub.ID())
		if tokenErr != nil {
			uc.logger.Errorw("failed to create default token", "error", tokenErr, "subscription_id", sub.ID())
			return fmt.Errorf("failed to create default token for subscription: %w", tokenErr)
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	// Post-transaction side effects (notifications are best-effort, not transactional)
	if cmd.ActivateImmediately && uc.subscriptionNotifier != nil {
		notifyCtx := context.Background()
		if err := uc.subscriptionNotifier.NotifySubscriptionActivation(notifyCtx, sub); err != nil {
			uc.logger.Warnw("failed to notify nodes of subscription activation",
				"subscription_id", sub.ID(),
				"error", err,
			)
		}
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
	// Use fixed days to ensure consistent subscription periods
	// This prevents "drifting" when starting on month boundaries (e.g., Jan 31 -> Feb 28 -> Mar 28)
	switch billingCycle {
	case vo.BillingCycleWeekly:
		return startDate.Add(7 * 24 * time.Hour) // 7 days
	case vo.BillingCycleMonthly:
		return startDate.Add(31 * 24 * time.Hour) // 31 days
	case vo.BillingCycleQuarterly:
		return startDate.Add(93 * 24 * time.Hour) // 93 days (31 * 3)
	case vo.BillingCycleSemiAnnual:
		return startDate.Add(180 * 24 * time.Hour) // 180 days
	case vo.BillingCycleYearly:
		return startDate.Add(365 * 24 * time.Hour) // 365 days
	case vo.BillingCycleLifetime:
		// For lifetime subscriptions, set a far future date (effectively never expires)
		// Use Jan 1 instead of Dec 31 23:59:59 to avoid year overflow when converting to eastern timezones
		return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		return startDate.Add(31 * 24 * time.Hour) // Default to 31 days
	}
}

func (uc *CreateSubscriptionUseCase) createDefaultToken(ctx context.Context, subscriptionID uint) (*subscription.SubscriptionToken, string, error) {
	plainToken, hashedToken, err := uc.tokenGenerator.Generate("sub")
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	if len(plainToken) < 8 {
		return nil, "", fmt.Errorf("generated token too short: got %d chars, need at least 8", len(plainToken))
	}
	prefix := plainToken[:8]

	token, err := subscription.NewSubscriptionToken(subscriptionID, "Default Token", hashedToken, prefix, vo.TokenScopeFull, nil)
	if err != nil {
		return nil, "", err
	}

	if err := uc.tokenRepo.Create(ctx, token); err != nil {
		return nil, "", fmt.Errorf("failed to save token: %w", err)
	}

	return token, plainToken, nil
}
