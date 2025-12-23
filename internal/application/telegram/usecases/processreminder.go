package usecases

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TelegramMessageSender sends messages via Telegram
type TelegramMessageSender interface {
	SendMessageMarkdown(chatID int64, text string) error
}

// highUsageInfo represents high traffic usage information for a plan
type highUsageInfo struct {
	PlanName     string
	ResourceType string
	UsedBytes    uint64
	Limit        uint64
	Percent      int
}

// ProcessReminderUseCase processes subscription expiring and traffic reminders
type ProcessReminderUseCase struct {
	bindingRepo      telegram.TelegramBindingRepository
	subscriptionRepo subscription.SubscriptionRepository
	usageRepo        subscription.SubscriptionUsageRepository
	planRepo         subscription.PlanRepository
	botService       TelegramMessageSender
	logger           logger.Interface
}

// NewProcessReminderUseCase creates a new ProcessReminderUseCase
func NewProcessReminderUseCase(
	bindingRepo telegram.TelegramBindingRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	usageRepo subscription.SubscriptionUsageRepository,
	planRepo subscription.PlanRepository,
	botService TelegramMessageSender,
	logger logger.Interface,
) *ProcessReminderUseCase {
	return &ProcessReminderUseCase{
		bindingRepo:      bindingRepo,
		subscriptionRepo: subscriptionRepo,
		usageRepo:        usageRepo,
		planRepo:         planRepo,
		botService:       botService,
		logger:           logger,
	}
}

// ProcessReminders implements the scheduler.ReminderProcessor interface
func (uc *ProcessReminderUseCase) ProcessReminders(ctx context.Context) error {
	expiringCount, expiringErrors := uc.processExpiringSubscriptions(ctx)
	trafficCount, trafficErrors := uc.processTrafficUsage(ctx)

	uc.logger.Infow("reminder processing completed",
		"expiring_notified", expiringCount,
		"traffic_notified", trafficCount,
		"expiring_errors", expiringErrors,
		"traffic_errors", trafficErrors,
	)

	return nil
}

func (uc *ProcessReminderUseCase) processExpiringSubscriptions(ctx context.Context) (int, int) {
	notified := 0
	errors := 0

	// Get bindings that can receive expiring notifications
	bindings, err := uc.bindingRepo.FindBindingsForExpiringNotification(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for expiring notification", "error", err)
		return 0, 1
	}

	for _, binding := range bindings {
		if !binding.CanNotifyExpiring() {
			continue
		}

		// Find expiring subscriptions for this user
		subs, err := uc.subscriptionRepo.FindExpiringSubscriptions(ctx, binding.ExpiringDays())
		if err != nil {
			uc.logger.Errorw("failed to find expiring subscriptions", "user_id", binding.UserID(), "error", err)
			errors++
			continue
		}

		// Filter to only this user's subscriptions
		var userSubs []*subscription.Subscription
		for _, sub := range subs {
			if sub.UserID() == binding.UserID() {
				userSubs = append(userSubs, sub)
			}
		}

		if len(userSubs) == 0 {
			continue
		}

		// Build message
		message := uc.buildExpiringMessage(userSubs, binding.ExpiringDays())

		// Record notification timestamp BEFORE sending to prevent duplicates on partial failure
		binding.RecordExpiringNotification()
		if err := uc.bindingRepo.Update(ctx, binding); err != nil {
			uc.logger.Errorw("failed to update binding before notification", "error", err)
			errors++
			continue
		}

		// Send message
		if err := uc.botService.SendMessageMarkdown(binding.TelegramUserID(), message); err != nil {
			uc.logger.Errorw("failed to send expiring notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			// Note: timestamp already updated, message will be retried in next window
			errors++
			continue
		}

		notified++
	}

	return notified, errors
}

func (uc *ProcessReminderUseCase) processTrafficUsage(ctx context.Context) (int, int) {
	notified := 0
	errors := 0

	// Get bindings that can receive traffic notifications
	bindings, err := uc.bindingRepo.FindBindingsForTrafficNotification(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for traffic notification", "error", err)
		return 0, 1
	}

	for _, binding := range bindings {
		if !binding.CanNotifyTraffic() {
			continue
		}

		// Get user's active subscriptions
		subs, err := uc.subscriptionRepo.GetActiveByUserID(ctx, binding.UserID())
		if err != nil {
			uc.logger.Errorw("failed to get active subscriptions", "user_id", binding.UserID(), "error", err)
			errors++
			continue
		}

		var highUsageSubs []highUsageInfo

		// Group subscriptions by plan
		planSubscriptions := make(map[uint][]*subscription.Subscription)
		for _, sub := range subs {
			planSubscriptions[sub.PlanID()] = append(planSubscriptions[sub.PlanID()], sub)
		}

		for planID, planSubs := range planSubscriptions {
			plan, err := uc.planRepo.GetByID(ctx, planID)
			if err != nil {
				continue
			}

			trafficLimit, _ := plan.GetTrafficLimit()
			if trafficLimit == 0 {
				continue // Unlimited
			}

			// Determine resource type based on plan type
			resourceType := subscription.ResourceTypeNode.String()
			if plan.PlanType() == "forward" {
				resourceType = subscription.ResourceTypeForwardRule.String()
			}

			// Get subscription IDs for this plan
			var subIDs []uint
			for _, sub := range planSubs {
				subIDs = append(subIDs, sub.ID())
			}

			// Get current period usage - use the earliest period start among these subscriptions
			now := time.Now()
			periodStart := now
			for _, sub := range planSubs {
				if sub.CurrentPeriodStart().Before(periodStart) {
					periodStart = sub.CurrentPeriodStart()
				}
			}

			summary, err := uc.usageRepo.GetTotalUsageBySubscriptionIDs(
				ctx,
				resourceType,
				subIDs,
				periodStart,
				now,
			)
			if err != nil || summary == nil {
				continue
			}

			usagePercent := int(float64(summary.Total) / float64(trafficLimit) * 100)
			if usagePercent >= binding.TrafficThreshold() {
				highUsageSubs = append(highUsageSubs, highUsageInfo{
					PlanName:     plan.Name(),
					ResourceType: resourceType,
					UsedBytes:    summary.Total,
					Limit:        trafficLimit,
					Percent:      usagePercent,
				})
			}
		}

		if len(highUsageSubs) == 0 {
			continue
		}

		// Build message
		message := uc.buildTrafficMessage(highUsageSubs, binding.TrafficThreshold())

		// Record notification timestamp BEFORE sending to prevent duplicates on partial failure
		binding.RecordTrafficNotification()
		if err := uc.bindingRepo.Update(ctx, binding); err != nil {
			uc.logger.Errorw("failed to update binding before notification", "error", err)
			errors++
			continue
		}

		// Send message
		if err := uc.botService.SendMessageMarkdown(binding.TelegramUserID(), message); err != nil {
			uc.logger.Errorw("failed to send traffic notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			// Note: timestamp already updated, message will be retried in next window
			errors++
			continue
		}

		notified++
	}

	return notified, errors
}

func (uc *ProcessReminderUseCase) buildExpiringMessage(subs []*subscription.Subscription, days int) string {
	msg := fmt.Sprintf("*Subscription Expiring Soon*\n\nThe following subscriptions will expire within %d days:\n\n", days)
	for _, sub := range subs {
		// Use ceiling to ensure 23.5 hours shows as 1 day, not 0
		hoursLeft := time.Until(sub.EndDate()).Hours()
		daysLeft := int(math.Ceil(hoursLeft / 24))
		if daysLeft < 0 {
			daysLeft = 0
		}
		msg += fmt.Sprintf("• Subscription `%s`: expires in *%d days* (%s)\n",
			sub.SID(),
			daysLeft,
			sub.EndDate().Format("2006-01-02"),
		)
	}
	msg += "\nPlease renew your subscription to avoid service interruption."
	return msg
}

func (uc *ProcessReminderUseCase) buildTrafficMessage(subs []highUsageInfo, threshold int) string {
	msg := fmt.Sprintf("*Traffic Usage Alert*\n\nThe following plans have reached %d%% of their traffic limit:\n\n", threshold)
	for _, item := range subs {
		msg += fmt.Sprintf("• Plan `%s`: *%d%%* used (%s / %s)\n",
			item.PlanName,
			item.Percent,
			formatBytes(item.UsedBytes),
			formatBytes(item.Limit),
		)
	}
	msg += "\nConsider upgrading your plan or reducing usage."
	return msg
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
