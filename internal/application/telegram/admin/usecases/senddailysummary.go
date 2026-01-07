package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SendDailySummaryUseCase handles sending daily business summary to admins
type SendDailySummaryUseCase struct {
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

// NewSendDailySummaryUseCase creates a new SendDailySummaryUseCase
func NewSendDailySummaryUseCase(
	bindingRepo admin.AdminTelegramBindingRepository,
	userRepo user.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	nodeRepo node.NodeRepository,
	agentRepo forward.AgentRepository,
	botService TelegramMessageSender,
	logger logger.Interface,
) *SendDailySummaryUseCase {
	return &SendDailySummaryUseCase{
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

// DailySummaryData contains aggregated daily business data
type DailySummaryData struct {
	Date             string  // Report date (business timezone)
	NewUsers         int64   // New user registrations
	ActiveUsers      int64   // Total active users
	NewSubscriptions int64   // New subscriptions
	TotalRevenue     float64 // Total revenue for the day
	Currency         string  // Revenue currency

	// Node status
	TotalNodes   int64
	OnlineNodes  int64
	OfflineNodes int64

	// Agent status
	TotalAgents   int64
	OnlineAgents  int64
	OfflineAgents int64

	// Traffic stats
	TotalTrafficBytes uint64 // Total traffic in bytes
}

// SendSummary sends daily summary to all subscribed admins
func (uc *SendDailySummaryUseCase) SendSummary(ctx context.Context) error {
	if uc.botService == nil {
		uc.logger.Debugw("daily summary skipped: bot service not available")
		return nil
	}

	// Get bindings that want daily summary
	bindings, err := uc.bindingRepo.FindBindingsForDailySummary(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for daily summary", "error", err)
		return fmt.Errorf("failed to find bindings: %w", err)
	}

	if len(bindings) == 0 {
		uc.logger.Debugw("no bindings configured for daily summary")
		return nil
	}

	// Calculate yesterday's date range in business timezone
	now := biztime.NowUTC()
	yesterdayStart, yesterdayEnd := uc.getYesterdayRange(now)

	// Gather statistics
	summary, err := uc.gatherDailyStats(ctx, yesterdayStart, yesterdayEnd)
	if err != nil {
		uc.logger.Errorw("failed to gather daily stats", "error", err)
		return fmt.Errorf("failed to gather stats: %w", err)
	}

	message := uc.buildDailySummaryMessage(summary)

	sentCount := 0
	errorCount := 0

	for _, binding := range bindings {
		if !binding.CanSendDailySummary() {
			continue
		}

		if err := uc.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			uc.logger.Errorw("failed to send daily summary",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			errorCount++
			continue
		}

		// Record that daily summary was sent
		binding.RecordDailySummary()
		if err := uc.bindingRepo.Update(ctx, binding); err != nil {
			uc.logger.Errorw("failed to update binding after daily summary", "error", err)
		}

		sentCount++
	}

	uc.logger.Infow("daily summary sent",
		"date", summary.Date,
		"sent_count", sentCount,
		"error_count", errorCount,
	)

	return nil
}

// getYesterdayRange returns the start and end of yesterday in UTC
// based on business timezone boundaries
func (uc *SendDailySummaryUseCase) getYesterdayRange(now time.Time) (time.Time, time.Time) {
	// Convert to business timezone to get yesterday's boundaries
	bizNow := biztime.ToBizTimezone(now)

	// Get yesterday in business timezone
	yesterday := bizNow.AddDate(0, 0, -1)

	// Start of yesterday (00:00:00 in business timezone)
	startOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, biztime.Location())

	// End of yesterday (23:59:59 in business timezone)
	endOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, biztime.Location())

	// Convert back to UTC for database queries
	return startOfDay.UTC(), endOfDay.UTC()
}

func (uc *SendDailySummaryUseCase) gatherDailyStats(ctx context.Context, start, end time.Time) (*DailySummaryData, error) {
	summary := &DailySummaryData{
		Date:     biztime.FormatInBizTimezone(start, "2006-01-02"),
		Currency: "USD", // Default currency
	}

	// Count new users registered yesterday using pagination to avoid OOM
	const pageSize = 500
	page := 1
	for {
		users, total, err := uc.userRepo.List(ctx, user.ListFilter{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			uc.logger.Warnw("failed to list users for daily summary", "error", err)
			break
		}

		for _, u := range users {
			if u.CreatedAt().After(start) && u.CreatedAt().Before(end) {
				summary.NewUsers++
			}
			if u.Status().IsActive() {
				summary.ActiveUsers++
			}
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	// Count new subscriptions using pagination
	page = 1
	for {
		subs, total, err := uc.subscriptionRepo.List(ctx, subscription.SubscriptionFilter{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			uc.logger.Warnw("failed to list subscriptions for daily summary", "error", err)
			break
		}

		for _, s := range subs {
			if s.CreatedAt().After(start) && s.CreatedAt().Before(end) {
				summary.NewSubscriptions++
			}
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

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
			if a.IsEnabled() {
				// Check if agent reported recently (within 5 minutes)
				if time.Since(a.UpdatedAt()) < 5*time.Minute {
					summary.OnlineAgents++
				}
			}
		}
		summary.OfflineAgents = summary.TotalAgents - summary.OnlineAgents
	}

	// Get traffic usage from combined sources:
	// 1. MySQL subscription_usage_stats for historical data (daily granularity)
	// 2. Redis hourly cache for recent data (last 24h)
	summary.TotalTrafficBytes = uc.getPlatformTrafficForPeriod(ctx, start, end)

	return summary, nil
}

// getPlatformTrafficForPeriod retrieves platform-wide traffic for a time period.
// Uses MySQL subscription_usage_stats with daily granularity as primary data source.
// Falls back to Redis hourly cache only if MySQL has no data AND the time range
// is within Redis TTL (25 hours).
func (uc *SendDailySummaryUseCase) getPlatformTrafficForPeriod(ctx context.Context, start, end time.Time) uint64 {
	var totalTraffic uint64

	// Daily summary is for yesterday, so all data should be in MySQL daily stats
	// Query MySQL subscription_usage_stats with daily granularity
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
	// 1. MySQL returned no data (possibly daily aggregation hasn't run yet)
	// 2. The time range overlaps with Redis TTL (25 hours)
	// 3. hourlyCache is available
	if totalTraffic == 0 && uc.hourlyCache != nil {
		now := biztime.NowUTC()
		redisTTLBoundary := now.Add(-25 * time.Hour)

		// Only attempt Redis fallback if end time is within Redis TTL
		if end.After(redisTTLBoundary) {
			// Adjust start time to Redis TTL boundary if needed
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
func (uc *SendDailySummaryUseCase) getTrafficFromHourlyCache(ctx context.Context, start, end time.Time) uint64 {
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

func (uc *SendDailySummaryUseCase) buildDailySummaryMessage(summary *DailySummaryData) string {
	// Format traffic
	trafficStr := formatBytesHuman(summary.TotalTrafficBytes)

	// Node status indicator
	nodeStatus := "🟢"
	if summary.OfflineNodes > 0 {
		nodeStatus = "🟡"
	}
	if summary.OnlineNodes == 0 && summary.TotalNodes > 0 {
		nodeStatus = "🔴"
	}

	// Agent status indicator
	agentStatus := "🟢"
	if summary.OfflineAgents > 0 {
		agentStatus = "🟡"
	}
	if summary.OnlineAgents == 0 && summary.TotalAgents > 0 {
		agentStatus = "🔴"
	}

	return fmt.Sprintf(`📊 <b>Daily Summary / 每日摘要</b>
📅 %s

👥 <b>Users / 用户</b>
   New 新增: <b>%d</b>
   Active 活跃: <b>%d</b>

📦 <b>Subscriptions / 订阅</b>
   New 新增: <b>%d</b>

%s <b>Nodes / 节点</b>
   Online 在线: <b>%d</b> / %d
   Offline 离线: <b>%d</b>

%s <b>Forward Agents / 转发代理</b>
   Online 在线: <b>%d</b> / %d
   Offline 离线: <b>%d</b>

📈 <b>Traffic / 流量</b>
   Total 总计: <b>%s</b>

━━━━━━━━━━━━━━━
Generated at %s`,
		summary.Date,
		summary.NewUsers,
		summary.ActiveUsers,
		summary.NewSubscriptions,
		nodeStatus,
		summary.OnlineNodes, summary.TotalNodes,
		summary.OfflineNodes,
		agentStatus,
		summary.OnlineAgents, summary.TotalAgents,
		summary.OfflineAgents,
		trafficStr,
		biztime.FormatInBizTimezone(biztime.NowUTC(), "2006-01-02 15:04:05"))
}

// formatBytesHuman formats bytes into human readable format
func formatBytesHuman(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
