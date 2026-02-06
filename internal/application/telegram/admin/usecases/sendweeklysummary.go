package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	telegram "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/telegram/i18n"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SendWeeklySummaryUseCase handles sending weekly business summary to admins
type SendWeeklySummaryUseCase struct {
	bindingRepo      admin.AdminTelegramBindingRepository
	userRepo         user.Repository
	subscriptionRepo subscription.SubscriptionRepository
	usageStatsRepo   subscription.SubscriptionUsageStatsRepository
	hourlyCache      cache.HourlyTrafficCache
	nodeRepo         node.NodeRepository
	agentRepo        forward.AgentRepository
	botService       TelegramMessageSender
	logger           logger.Interface
}

// NewSendWeeklySummaryUseCase creates a new SendWeeklySummaryUseCase
func NewSendWeeklySummaryUseCase(
	bindingRepo admin.AdminTelegramBindingRepository,
	userRepo user.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	nodeRepo node.NodeRepository,
	agentRepo forward.AgentRepository,
	botService TelegramMessageSender,
	logger logger.Interface,
) *SendWeeklySummaryUseCase {
	return &SendWeeklySummaryUseCase{
		bindingRepo:      bindingRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		usageStatsRepo:   usageStatsRepo,
		hourlyCache:      hourlyCache,
		nodeRepo:         nodeRepo,
		agentRepo:        agentRepo,
		botService:       botService,
		logger:           logger,
	}
}

// SendSummary sends weekly summary to all subscribed admins
func (uc *SendWeeklySummaryUseCase) SendSummary(ctx context.Context) error {
	if uc.botService == nil {
		uc.logger.Debugw("weekly summary skipped: bot service not available")
		return nil
	}

	// Calculate current business timezone hour and weekday
	now := biztime.NowUTC()
	bizNow := biztime.ToBizTimezone(now)
	currentBizHour := bizNow.Hour()
	currentBizWeekday := int(bizNow.Weekday()) // 0=Sunday, 1=Monday...6=Saturday

	// Get bindings that want weekly summary at the current business hour and weekday
	bindings, err := uc.bindingRepo.FindBindingsForWeeklySummary(ctx, currentBizHour, currentBizWeekday)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for weekly summary", "error", err)
		return fmt.Errorf("failed to find bindings: %w", err)
	}

	if len(bindings) == 0 {
		return nil
	}

	// Calendar-based dedup: filter bindings not yet sent this weekly period
	var matchedBindings []*admin.AdminTelegramBinding
	for _, binding := range bindings {
		if binding.CanSendWeeklySummary() {
			matchedBindings = append(matchedBindings, binding)
		}
	}

	if len(matchedBindings) == 0 {
		return nil
	}

	// Calculate last week and week before that
	lastWeekStart, lastWeekEnd := uc.getLastWeekRange(now)
	prevWeekStart, prevWeekEnd := uc.getPreviousWeekRange(now)

	// Gather statistics
	summary, err := uc.gatherWeeklyStats(ctx, lastWeekStart, lastWeekEnd, prevWeekStart, prevWeekEnd)
	if err != nil {
		uc.logger.Errorw("failed to gather weekly stats", "error", err)
		return fmt.Errorf("failed to gather stats: %w", err)
	}

	sentCount := 0
	errorCount := 0

	for _, binding := range matchedBindings {
		lang := i18n.ParseLang(binding.Language())
		message := uc.buildWeeklySummaryMessage(summary, lang)

		_ = uc.botService.SendChatAction(binding.TelegramUserID(), "typing")
		if err := uc.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			if telegram.IsBotBlocked(err) {
				uc.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
			uc.logger.Errorw("failed to send weekly summary",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			errorCount++
			continue
		}

		// Record that weekly summary was sent
		binding.RecordWeeklySummary()
		if err := uc.bindingRepo.Update(ctx, binding); err != nil {
			uc.logger.Errorw("failed to update binding after weekly summary", "error", err)
		}

		sentCount++
	}

	uc.logger.Infow("weekly summary sent",
		"week_start", summary.WeekStart,
		"week_end", summary.WeekEnd,
		"sent_count", sentCount,
		"error_count", errorCount,
	)

	return nil
}

// getLastWeekRange returns the start and end of last week in UTC
func (uc *SendWeeklySummaryUseCase) getLastWeekRange(now time.Time) (time.Time, time.Time) {
	// Convert to business timezone
	bizNow := biztime.ToBizTimezone(now)

	// Find the start of this week (Monday)
	weekday := int(bizNow.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday -> 7
	}
	daysToMonday := weekday - 1

	// Start of this week
	thisWeekStart := bizNow.AddDate(0, 0, -daysToMonday)
	thisWeekStart = time.Date(thisWeekStart.Year(), thisWeekStart.Month(), thisWeekStart.Day(), 0, 0, 0, 0, biztime.Location())

	// Last week start (7 days before this week)
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)

	// Last week end (Sunday 23:59:59)
	lastWeekEnd := thisWeekStart.Add(-time.Nanosecond)

	return lastWeekStart.UTC(), lastWeekEnd.UTC()
}

// getPreviousWeekRange returns the start and end of the week before last
func (uc *SendWeeklySummaryUseCase) getPreviousWeekRange(now time.Time) (time.Time, time.Time) {
	lastWeekStart, _ := uc.getLastWeekRange(now)

	// Convert back to business timezone for calculation
	bizLastWeekStart := biztime.ToBizTimezone(lastWeekStart)

	// Previous week is 7 days before last week
	prevWeekStart := bizLastWeekStart.AddDate(0, 0, -7)
	prevWeekEnd := bizLastWeekStart.Add(-time.Nanosecond)

	return prevWeekStart.UTC(), prevWeekEnd.UTC()
}

func (uc *SendWeeklySummaryUseCase) gatherWeeklyStats(ctx context.Context, lastStart, lastEnd, prevStart, prevEnd time.Time) (*dto.WeeklySummaryData, error) {
	summary := &dto.WeeklySummaryData{
		WeekStart: biztime.FormatInBizTimezone(lastStart, "2006-01-02"),
		WeekEnd:   biztime.FormatInBizTimezone(lastEnd, "2006-01-02"),
		Currency:  "USD",
	}

	// Count users for both periods using pagination to avoid OOM
	const pageSize = 500
	page := 1
	for {
		users, total, err := uc.userRepo.List(ctx, user.ListFilter{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			uc.logger.Warnw("failed to list users for weekly summary", "error", err)
			break
		}

		for _, u := range users {
			// Last week new users - use closed interval [lastStart, lastEnd]
			if !u.CreatedAt().Before(lastStart) && !u.CreatedAt().After(lastEnd) {
				summary.NewUsers++
			}
			// Previous week new users - use closed interval [prevStart, prevEnd]
			if !u.CreatedAt().Before(prevStart) && !u.CreatedAt().After(prevEnd) {
				summary.PrevNewUsers++
			}
			// Active users
			if u.Status().IsActive() {
				summary.ActiveUsers++
			}
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	// Count subscriptions for both periods using pagination
	page = 1
	for {
		subs, total, err := uc.subscriptionRepo.List(ctx, subscription.SubscriptionFilter{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			uc.logger.Warnw("failed to list subscriptions for weekly summary", "error", err)
			break
		}

		for _, s := range subs {
			// Use closed interval [lastStart, lastEnd]
			if !s.CreatedAt().Before(lastStart) && !s.CreatedAt().After(lastEnd) {
				summary.NewSubscriptions++
			}
			// Use closed interval [prevStart, prevEnd]
			if !s.CreatedAt().Before(prevStart) && !s.CreatedAt().After(prevEnd) {
				summary.PrevNewSubscriptions++
			}
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	// Calculate change percentages
	summary.UserChangePercent = CalculateChangePercent(summary.NewUsers, summary.PrevNewUsers)
	summary.SubChangePercent = CalculateChangePercent(summary.NewSubscriptions, summary.PrevNewSubscriptions)

	// Get node status
	nodes, total, err := uc.nodeRepo.List(ctx, node.NodeFilter{})
	if err == nil {
		summary.TotalNodes = total
		for _, n := range nodes {
			if n.IsOnline() {
				summary.OnlineNodes++
			}
		}
		summary.OfflineNodes = summary.TotalNodes - summary.OnlineNodes
	}

	// Get agent status
	agents, agentTotal, err := uc.agentRepo.List(ctx, forward.AgentListFilter{})
	if err == nil {
		summary.TotalAgents = agentTotal
		for _, a := range agents {
			if a.IsOnline() {
				summary.OnlineAgents++
			}
		}
		summary.OfflineAgents = summary.TotalAgents - summary.OnlineAgents
	}

	// Get traffic for last week from MySQL subscription_usage_stats (daily granularity)
	summary.TotalTrafficBytes = uc.getPlatformTrafficForPeriod(ctx, lastStart, lastEnd)

	// Get traffic for previous week
	summary.PrevTotalTrafficBytes = uc.getPlatformTrafficForPeriod(ctx, prevStart, prevEnd)

	summary.TrafficChangePercent = CalculateChangePercentUint64(summary.TotalTrafficBytes, summary.PrevTotalTrafficBytes)

	return summary, nil
}

// getPlatformTrafficForPeriod retrieves platform-wide traffic for a time period.
// Uses MySQL subscription_usage_stats with daily granularity as primary data source.
// Falls back to Redis hourly cache only if MySQL has no data AND the time range
// is within Redis TTL (25 hours). For weekly summary, Redis fallback is typically
// not useful since weekly data is older than Redis TTL.
func (uc *SendWeeklySummaryUseCase) getPlatformTrafficForPeriod(ctx context.Context, start, end time.Time) uint64 {
	var totalTraffic uint64

	// Weekly summary uses historical data, so query MySQL subscription_usage_stats with daily granularity
	usageSummary, err := uc.usageStatsRepo.GetPlatformTotalUsage(ctx, subscription.GranularityDaily, start, end)
	if err != nil {
		uc.logger.Warnw("failed to get platform usage from stats repo",
			"error", err,
			"start", start,
			"end", end,
		)
	} else if usageSummary != nil {
		totalTraffic += usageSummary.Total
	}

	// Fallback to Redis hourly cache only if:
	// 1. MySQL returned no data
	// 2. The time range overlaps with Redis TTL (25 hours)
	// Note: For weekly summary, this fallback rarely applies since data is typically > 25h old
	if totalTraffic == 0 && uc.hourlyCache != nil {
		now := biztime.NowUTC()
		redisTTLBoundary := now.Add(-25 * time.Hour)

		// Only attempt Redis fallback if end time is within Redis TTL
		if end.After(redisTTLBoundary) {
			effectiveStart := start
			if effectiveStart.Before(redisTTLBoundary) {
				effectiveStart = redisTTLBoundary
			}
			redisTraffic := uc.getTrafficFromHourlyCache(ctx, effectiveStart, end)
			totalTraffic += redisTraffic
		}
	}

	return totalTraffic
}

// getTrafficFromHourlyCache retrieves platform-wide traffic from Redis hourly cache.
func (uc *SendWeeklySummaryUseCase) getTrafficFromHourlyCache(ctx context.Context, start, end time.Time) uint64 {
	var total uint64

	// Iterate through each hour in the time range
	current := biztime.TruncateToHourInBiz(start)
	endHour := biztime.TruncateToHourInBiz(end)

	for !current.After(endHour) {
		hourlyData, err := uc.hourlyCache.GetAllHourlyTraffic(ctx, current)
		if err != nil {
			uc.logger.Warnw("failed to get hourly traffic from cache",
				"hour", current.Format("2006-01-02 15:04"),
				"error", err,
			)
			current = current.Add(time.Hour)
			continue
		}

		for _, data := range hourlyData {
			// Safe conversion: only add positive values to prevent uint64 overflow
			if data.Upload > 0 {
				total += uint64(data.Upload)
			}
			if data.Download > 0 {
				total += uint64(data.Download)
			}
		}

		current = current.Add(time.Hour)
	}

	return total
}

func (uc *SendWeeklySummaryUseCase) buildWeeklySummaryMessage(summary *dto.WeeklySummaryData, lang i18n.Lang) string {
	trafficStr := FormatBytes(summary.TotalTrafficBytes)
	nodeStatus := statusIndicator(summary.OnlineNodes, summary.OfflineNodes, summary.TotalNodes)
	agentStatus := statusIndicator(summary.OnlineAgents, summary.OfflineAgents, summary.TotalAgents)
	userChange := FormatPercentChangeCompact(summary.UserChangePercent)
	subChange := FormatPercentChangeCompact(summary.SubChangePercent)
	trafficChange := FormatPercentChangeCompact(summary.TrafficChangePercent)
	generatedAt := biztime.FormatInBizTimezone(biztime.NowUTC(), "2006-01-02 15:04:05")

	if lang == i18n.EN {
		return fmt.Sprintf(`📊 <b>Weekly Summary</b>
📅 %s ~ %s

👥 <b>Users</b>
   New: <b>%d</b> %s
   Active: <b>%d</b>

📦 <b>Subscriptions</b>
   New: <b>%d</b> %s

%s <b>Nodes</b>
   Online: <b>%d</b> / %d
   Offline: <b>%d</b>

%s <b>Forward Agents</b>
   Online: <b>%d</b> / %d
   Offline: <b>%d</b>

📈 <b>Traffic</b>
   Total: <b>%s</b> %s

━━━━━━━━━━━━━━━
Generated at %s`,
			summary.WeekStart, summary.WeekEnd,
			summary.NewUsers, userChange, summary.ActiveUsers,
			summary.NewSubscriptions, subChange,
			nodeStatus, summary.OnlineNodes, summary.TotalNodes, summary.OfflineNodes,
			agentStatus, summary.OnlineAgents, summary.TotalAgents, summary.OfflineAgents,
			trafficStr, trafficChange, generatedAt)
	}

	return fmt.Sprintf(`📊 <b>每周摘要</b>
📅 %s ~ %s

👥 <b>用户</b>
   新增：<b>%d</b> %s
   活跃：<b>%d</b>

📦 <b>订阅</b>
   新增：<b>%d</b> %s

%s <b>节点</b>
   在线：<b>%d</b> / %d
   离线：<b>%d</b>

%s <b>转发代理</b>
   在线：<b>%d</b> / %d
   离线：<b>%d</b>

📈 <b>流量</b>
   总计：<b>%s</b> %s

━━━━━━━━━━━━━━━
生成于 %s`,
		summary.WeekStart, summary.WeekEnd,
		summary.NewUsers, userChange, summary.ActiveUsers,
		summary.NewSubscriptions, subChange,
		nodeStatus, summary.OnlineNodes, summary.TotalNodes, summary.OfflineNodes,
		agentStatus, summary.OnlineAgents, summary.TotalAgents, summary.OfflineAgents,
		trafficStr, trafficChange, generatedAt)
}
