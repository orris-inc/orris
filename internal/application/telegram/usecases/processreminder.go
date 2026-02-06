package usecases

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	telegramInfra "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TelegramMessageSender sends messages via Telegram
type TelegramMessageSender interface {
	SendMessageMarkdown(chatID int64, text string) error
	SendChatAction(chatID int64, action string) error
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
	usageStatsRepo   subscription.SubscriptionUsageStatsRepository
	hourlyCache      cache.HourlyTrafficCache
	planRepo         subscription.PlanRepository
	botService       TelegramMessageSender
	logger           logger.Interface
}

// NewProcessReminderUseCase creates a new ProcessReminderUseCase
func NewProcessReminderUseCase(
	bindingRepo telegram.TelegramBindingRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	planRepo subscription.PlanRepository,
	botService TelegramMessageSender,
	logger logger.Interface,
) *ProcessReminderUseCase {
	return &ProcessReminderUseCase{
		bindingRepo:      bindingRepo,
		subscriptionRepo: subscriptionRepo,
		usageStatsRepo:   usageStatsRepo,
		hourlyCache:      hourlyCache,
		planRepo:         planRepo,
		botService:       botService,
		logger:           logger,
	}
}

// SetBotService sets the bot service for sending messages.
// This allows injecting the bot service after the use case is created.
func (uc *ProcessReminderUseCase) SetBotService(botService TelegramMessageSender) {
	uc.botService = botService
}

// ProcessReminders implements the scheduler.ReminderProcessor interface
func (uc *ProcessReminderUseCase) ProcessReminders(ctx context.Context) error {
	if uc.botService == nil {
		uc.logger.Debugw("reminder processing skipped: bot service not available")
		return nil
	}

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

	// Collect unique expiringDays values to avoid N+1 queries
	expiringDaysSet := make(map[int]struct{})
	validBindings := make([]*telegram.TelegramBinding, 0, len(bindings))
	for _, binding := range bindings {
		if !binding.CanNotifyExpiring() {
			continue
		}
		expiringDaysSet[binding.ExpiringDays()] = struct{}{}
		validBindings = append(validBindings, binding)
	}

	if len(validBindings) == 0 {
		return 0, 0
	}

	// Batch fetch expiring subscriptions for each unique expiringDays value
	// Key: expiringDays, Value: subscriptions grouped by userID
	subscriptionsByDaysAndUser := make(map[int]map[uint][]*subscription.Subscription)
	for days := range expiringDaysSet {
		subs, err := uc.subscriptionRepo.FindExpiringSubscriptions(ctx, days)
		if err != nil {
			uc.logger.Errorw("failed to find expiring subscriptions", "days", days, "error", err)
			errors++
			continue
		}
		// Group subscriptions by userID
		userSubsMap := make(map[uint][]*subscription.Subscription)
		for _, sub := range subs {
			userSubsMap[sub.UserID()] = append(userSubsMap[sub.UserID()], sub)
		}
		subscriptionsByDaysAndUser[days] = userSubsMap
	}

	for _, binding := range validBindings {
		// Get pre-fetched subscriptions for this binding's expiringDays and userID
		userSubsMap, ok := subscriptionsByDaysAndUser[binding.ExpiringDays()]
		if !ok {
			continue
		}
		userSubs := userSubsMap[binding.UserID()]
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
		_ = uc.botService.SendChatAction(binding.TelegramUserID(), "typing")
		if err := uc.botService.SendMessageMarkdown(binding.TelegramUserID(), message); err != nil {
			if telegramInfra.IsBotBlocked(err) {
				uc.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
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

		// Batch fetch all plans to avoid N+1 queries
		planIDs := make([]uint, 0, len(planSubscriptions))
		for planID := range planSubscriptions {
			planIDs = append(planIDs, planID)
		}
		plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
		if err != nil {
			uc.logger.Warnw("failed to batch fetch plans", "error", err)
			continue
		}
		planMap := make(map[uint]*subscription.Plan, len(plans))
		for _, plan := range plans {
			planMap[plan.ID()] = plan
		}

		for planID, planSubs := range planSubscriptions {
			plan, ok := planMap[planID]
			if !ok {
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

			// Get current period usage - resolve period based on plan's traffic_reset_mode
			now := biztime.NowUTC()
			periodStart := now
			for _, sub := range planSubs {
				period := subscription.ResolveTrafficPeriod(plan, sub)
				if period.Start.Before(periodStart) {
					periodStart = period.Start
				}
			}

			summary, err := uc.getTotalUsageBySubscriptionIDs(
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
		_ = uc.botService.SendChatAction(binding.TelegramUserID(), "typing")
		if err := uc.botService.SendMessageMarkdown(binding.TelegramUserID(), message); err != nil {
			if telegramInfra.IsBotBlocked(err) {
				uc.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
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
	msg := fmt.Sprintf("⏰ *订阅即将到期 / Expiring Soon*\n\n"+
		"您有 %d 个订阅将在 %d 天内到期\n"+
		"%d subscription(s) expiring within %d days:\n\n", len(subs), days, len(subs), days)
	for _, sub := range subs {
		// Use ceiling to ensure 23.5 hours shows as 1 day, not 0
		hoursLeft := time.Until(sub.EndDate()).Hours()
		daysLeft := int(math.Ceil(hoursLeft / 24))
		if daysLeft < 0 {
			daysLeft = 0
		}
		urgency := "🟡"
		if daysLeft <= 1 {
			urgency = "🔴"
		} else if daysLeft <= 3 {
			urgency = "🟠"
		}
		msg += fmt.Sprintf("%s `%s`\n   └ *%d 天后到期* / Expires in *%d day(s)*\n   └ %s\n",
			urgency,
			sub.SID(),
			daysLeft,
			daysLeft,
			biztime.FormatInBizTimezone(sub.EndDate(), "2006-01-02"),
		)
	}
	msg += "\n💡 请及时续费，避免服务中断\nRenew now to avoid interruption"
	return msg
}

func (uc *ProcessReminderUseCase) buildTrafficMessage(subs []highUsageInfo, threshold int) string {
	msg := fmt.Sprintf("📊 *流量使用警告 / Traffic Alert*\n\n"+
		"以下套餐已使用超过 %d%% 流量\n"+
		"Plans exceeded %d%% traffic usage:\n\n", threshold, threshold)
	for _, item := range subs {
		bar := buildProgressBar(item.Percent)
		msg += fmt.Sprintf("📦 `%s`\n"+
			"   %s *%d%%*\n"+
			"   已用 Used: %s / %s\n\n",
			telegramInfra.EscapeMarkdownV1(item.PlanName),
			bar,
			item.Percent,
			formatBytes(item.UsedBytes),
			formatBytes(item.Limit),
		)
	}
	msg += "💡 请注意流量使用，或考虑升级套餐\nMonitor usage or consider upgrading"
	return msg
}

func buildProgressBar(percent int) string {
	filled := percent / 10
	if filled < 0 {
		filled = 0
	}
	if filled > 10 {
		filled = 10
	}
	empty := 10 - filled
	return "▓" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "▓"
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

// getTotalUsageBySubscriptionIDs retrieves total traffic combining Redis (recent 24h) and MySQL stats (historical).
// This method aggregates traffic from two sources:
// - Last 24 hours: Redis HourlyTrafficCache (real-time data)
// - Before 24 hours: MySQL subscription_usage_stats table (aggregated historical data)
func (uc *ProcessReminderUseCase) getTotalUsageBySubscriptionIDs(
	ctx context.Context,
	resourceType string,
	subscriptionIDs []uint,
	from, to time.Time,
) (*subscription.UsageSummary, error) {
	if len(subscriptionIDs) == 0 {
		return &subscription.UsageSummary{Total: 0}, nil
	}

	now := biztime.NowUTC()

	// Use start of yesterday's business day as batch/speed boundary (Lambda architecture)
	// MySQL: complete days before yesterday; Redis: yesterday + today (within 48h TTL)
	recentBoundary := biztime.StartOfDayUTC(now.AddDate(0, 0, -1))

	var total uint64

	// Determine time boundaries for recent data (yesterday + today from Redis)
	recentFrom := from
	if recentFrom.Before(recentBoundary) {
		recentFrom = recentBoundary
	}

	// Get recent traffic from Redis (yesterday + today)
	if recentFrom.Before(to) && recentFrom.Before(now) {
		recentTo := to
		if recentTo.After(now) {
			recentTo = now
		}
		recentTraffic, err := uc.hourlyCache.GetTotalTrafficBySubscriptionIDs(
			ctx, subscriptionIDs, resourceType, recentFrom, recentTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get recent traffic from Redis",
				"subscription_ids_count", len(subscriptionIDs),
				"resource_type", resourceType,
				"error", err,
			)
			// Continue with historical data only
		} else {
			for _, t := range recentTraffic {
				total += t.Total
			}
			uc.logger.Debugw("got recent 24h traffic from Redis",
				"subscription_ids_count", len(subscriptionIDs),
				"recent_total", total,
			)
		}
	}

	// Get historical traffic from MySQL stats (complete days before yesterday)
	if from.Before(recentBoundary) {
		historicalTo := recentBoundary.Add(-time.Second)
		if historicalTo.After(to) {
			historicalTo = to
		}
		historicalTraffic, err := uc.usageStatsRepo.GetTotalBySubscriptionIDs(
			ctx, subscriptionIDs, nil, subscription.GranularityDaily, from, historicalTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get historical traffic from stats",
				"subscription_ids_count", len(subscriptionIDs),
				"error", err,
			)
			// Continue with Redis data only if available
		} else if historicalTraffic != nil {
			total += historicalTraffic.Total
			uc.logger.Debugw("got historical traffic from MySQL stats",
				"subscription_ids_count", len(subscriptionIDs),
				"historical_total", historicalTraffic.Total,
				"combined_total", total,
			)
		}
	}

	return &subscription.UsageSummary{Total: total}, nil
}
