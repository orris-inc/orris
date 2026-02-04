package usecases

import (
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// BuildNewUserMessage builds a new user notification message (HTML format)
func BuildNewUserMessage(userSID, email, name, source string, createdAt time.Time) string {
	sourceTextZH := "注册"
	if source != "" {
		sourceTextZH = source
	}

	return fmt.Sprintf(`👤 <b>新用户注册</b>

邮箱：%s
名称：%s
ID：<code>%s</code>
来源：%s
注册时间：%s`,
		EscapeHTML(email),
		EscapeHTML(name),
		userSID,
		EscapeHTML(sourceTextZH),
		biztime.FormatInBizTimezone(createdAt, "2006-01-02 15:04:05"),
	)
}

// BuildNewUserMessageFromDTO builds a new user notification message from DTO (HTML format)
func BuildNewUserMessageFromDTO(info dto.NewUserInfo) string {
	return BuildNewUserMessage(info.SID, info.Email, info.Name, info.Source, info.CreatedAt)
}

// BuildPaymentSuccessMessage builds a payment success notification message (HTML format)
func BuildPaymentSuccessMessage(paymentSID, userSID, userEmail, planName string, amount float64, currency, paymentMethod, transactionID string, paidAt time.Time) string {
	amountStr := FormatAmount(amount, currency)

	return fmt.Sprintf(`💰 <b>支付成功</b>

金额：%s
用户：<code>%s</code>
邮箱：%s
套餐：%s
支付方式：%s
交易号：%s
支付时间：%s`,
		amountStr,
		userSID,
		EscapeHTML(userEmail),
		EscapeHTML(planName),
		EscapeHTML(paymentMethod),
		EscapeHTML(transactionID),
		biztime.FormatInBizTimezone(paidAt, "2006-01-02 15:04:05"),
	)
}

// BuildPaymentSuccessMessageFromDTO builds a payment success notification message from DTO (HTML format)
func BuildPaymentSuccessMessageFromDTO(info dto.PaymentInfo) string {
	return BuildPaymentSuccessMessage(
		info.PaymentSID,
		info.UserSID,
		info.UserEmail,
		info.PlanName,
		info.Amount,
		info.Currency,
		info.PaymentMethod,
		info.TransactionID,
		info.PaidAt,
	)
}

// BuildNodeOnlineMessage builds a node online notification message (HTML format)
func BuildNodeOnlineMessage(nodeSID, nodeName string, onlineAt time.Time) string {
	return fmt.Sprintf(`🟢 <b>Node Agent 上线通知</b>

Node Agent：%s
ID：<code>%s</code>
上线时间：%s

✅ Node Agent 已恢复连接`,
		EscapeHTML(nodeName),
		nodeSID,
		biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05"),
	)
}

// BuildNodeOfflineMessage builds a node offline notification message (HTML format)
func BuildNodeOfflineMessage(nodeSID, nodeName string, lastSeenAt time.Time, offlineMinutes int) string {
	lastSeenStr := biztime.FormatInBizTimezone(lastSeenAt, "2006-01-02 15:04:05")

	return fmt.Sprintf(`🔴 <b>Node Agent 离线告警</b>

Node Agent：<code>%s</code>
ID：<code>%s</code>
最后在线：%s
离线时长：%d 分钟

⚠️ 请检查 Node Agent 状态

―――――――――――――

🔴 <b>Node Agent Offline Alert</b>

Node Agent: <code>%s</code>
ID: <code>%s</code>
Last seen: %s
Offline: %d min

⚠️ Please check Node Agent status`,
		EscapeHTML(nodeName), nodeSID, lastSeenStr, offlineMinutes,
		EscapeHTML(nodeName), nodeSID, lastSeenStr, offlineMinutes,
	)
}

// BuildNodeOfflineMessageFromDTO builds a node offline notification message from DTO (HTML format)
// Handles nil LastSeenAt by displaying "N/A"
func BuildNodeOfflineMessageFromDTO(info dto.OfflineNodeInfo) string {
	lastSeenStr := "N/A"
	if info.LastSeenAt != nil {
		lastSeenStr = biztime.FormatInBizTimezone(*info.LastSeenAt, "2006-01-02 15:04:05")
	}

	return fmt.Sprintf(`🔴 <b>Node Agent 离线告警</b>

Node Agent：<code>%s</code>
ID：<code>%s</code>
最后在线：%s
离线时长：%d 分钟

⚠️ 请检查 Node Agent 状态

―――――――――――――

🔴 <b>Node Agent Offline Alert</b>

Node Agent: <code>%s</code>
ID: <code>%s</code>
Last seen: %s
Offline: %d min

⚠️ Please check Node Agent status`,
		EscapeHTML(info.Name), info.SID, lastSeenStr, info.OfflineMinutes,
		EscapeHTML(info.Name), info.SID, lastSeenStr, info.OfflineMinutes)
}

// BuildAgentOnlineMessage builds a forward agent online notification message (HTML format)
func BuildAgentOnlineMessage(agentSID, agentName string, onlineAt time.Time) string {
	return fmt.Sprintf(`🟢 <b>转发代理上线通知</b>

转发代理：%s
ID：<code>%s</code>
上线时间：%s

✅ 转发代理已恢复连接`,
		EscapeHTML(agentName),
		agentSID,
		biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05"),
	)
}

// BuildAgentOfflineMessage builds a forward agent offline notification message (HTML format)
func BuildAgentOfflineMessage(agentSID, agentName string, lastSeenAt time.Time, offlineMinutes int) string {
	lastSeenStr := biztime.FormatInBizTimezone(lastSeenAt, "2006-01-02 15:04:05")

	return fmt.Sprintf(`🔴 <b>转发代理离线告警</b>

转发代理：<code>%s</code>
ID：<code>%s</code>
最后在线：%s
离线时长：%d 分钟

⚠️ 请检查转发代理状态

―――――――――――――

🔴 <b>Forward Agent Offline Alert</b>

Forward Agent: <code>%s</code>
ID: <code>%s</code>
Last seen: %s
Offline: %d min

⚠️ Please check forward agent status`,
		EscapeHTML(agentName), agentSID, lastSeenStr, offlineMinutes,
		EscapeHTML(agentName), agentSID, lastSeenStr, offlineMinutes,
	)
}

// BuildAgentOfflineMessageFromDTO builds a forward agent offline notification message from DTO (HTML format)
// Handles nil LastSeenAt by displaying "N/A"
func BuildAgentOfflineMessageFromDTO(info dto.OfflineAgentInfo) string {
	lastSeenStr := "N/A"
	if info.LastSeenAt != nil {
		lastSeenStr = biztime.FormatInBizTimezone(*info.LastSeenAt, "2006-01-02 15:04:05")
	}

	return fmt.Sprintf(`🔴 <b>转发代理离线告警</b>

转发代理：<code>%s</code>
ID：<code>%s</code>
最后在线：%s
离线时长：%d 分钟

⚠️ 请检查转发代理状态

―――――――――――――

🔴 <b>Forward Agent Offline Alert</b>

Forward Agent: <code>%s</code>
ID: <code>%s</code>
Last seen: %s
Offline: %d min

⚠️ Please check forward agent status`,
		EscapeHTML(info.Name), info.SID, lastSeenStr, info.OfflineMinutes,
		EscapeHTML(info.Name), info.SID, lastSeenStr, info.OfflineMinutes)
}

// BuildDailySummaryMessage builds a daily summary message (HTML format)
func BuildDailySummaryMessage(
	date time.Time,
	newUsers int64,
	newOrders int64,
	revenue float64,
	currency string,
	onlineNodes, offlineNodes int64,
	onlineAgents, offlineAgents int64,
	uploadGB, downloadGB, totalGB float64,
) string {
	dateStr := biztime.FormatInBizTimezone(date, "2006-01-02")
	amountStr := FormatAmount(revenue, currency)

	return fmt.Sprintf(`📊 <b>每日业务摘要</b>
📅 %s

👥 新增用户：%d
💳 新增订单：%d
💰 营收金额：%s

🖥️ Node Agent 状态：
   在线：%d | 离线：%d

📡 Forward Agent 状态：
   在线：%d | 离线：%d

📈 流量统计：
   上行：%.2f GB
   下行：%.2f GB
   总计：%.2f GB`,
		dateStr,
		newUsers,
		newOrders,
		amountStr,
		onlineNodes, offlineNodes,
		onlineAgents, offlineAgents,
		uploadGB, downloadGB, totalGB,
	)
}

// BuildWeeklySummaryMessage builds a weekly summary message (HTML format)
func BuildWeeklySummaryMessage(
	weekStart, weekEnd time.Time,
	newUsers int64,
	newOrders int64,
	revenue float64,
	currency string,
	userChangePercent, orderChangePercent, revenueChangePercent float64,
	onlineNodes, offlineNodes int64,
	onlineAgents, offlineAgents int64,
	totalGB float64,
) string {
	startStr := biztime.FormatInBizTimezone(weekStart, "2006-01-02")
	endStr := biztime.FormatInBizTimezone(weekEnd, "2006-01-02")
	amountStr := FormatAmount(revenue, currency)
	userChange := FormatPercentChange(userChangePercent)
	orderChange := FormatPercentChange(orderChangePercent)
	revenueChange := FormatPercentChange(revenueChangePercent)

	return fmt.Sprintf(`📊 <b>每周业务摘要</b>
📅 %s ~ %s

👥 新增用户：%d
💳 新增订单：%d
💰 营收金额：%s

📈 对比上周：
   用户：%s
   订单：%s
   营收：%s

🖥️ Node Agent 状态：
   在线：%d | 离线：%d

📡 Forward Agent 状态：
   在线：%d | 离线：%d

📈 本周流量：%.2f GB`,
		startStr, endStr,
		newUsers,
		newOrders,
		amountStr,
		userChange,
		orderChange,
		revenueChange,
		onlineNodes, offlineNodes,
		onlineAgents, offlineAgents,
		totalGB,
	)
}

// BuildNodeRecoveryMessage builds a node recovery notification message (HTML format)
// This is sent when a node transitions from Firing state back to Normal
func BuildNodeRecoveryMessage(nodeSID, nodeName string, onlineAt time.Time, downtimeMinutes int64) string {
	onlineAtStr := biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05")

	return fmt.Sprintf(`🟢 <b>Node Agent 恢复通知</b>

Node Agent：%s
ID：<code>%s</code>
恢复时间：%s
离线时长：%d 分钟

✅ Node Agent 已恢复正常运行

―――――――――――――

🟢 <b>Node Agent Recovery</b>

Node Agent: %s
ID: <code>%s</code>
Recovered at: %s
Downtime: %d min

✅ Node Agent is back online`,
		EscapeHTML(nodeName), nodeSID, onlineAtStr, downtimeMinutes,
		EscapeHTML(nodeName), nodeSID, onlineAtStr, downtimeMinutes,
	)
}

// BuildAgentRecoveryMessage builds a forward agent recovery notification message (HTML format)
// This is sent when an agent transitions from Firing state back to Normal
func BuildAgentRecoveryMessage(agentSID, agentName string, onlineAt time.Time, downtimeMinutes int64) string {
	onlineAtStr := biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05")

	return fmt.Sprintf(`🟢 <b>转发代理恢复通知</b>

转发代理：%s
ID：<code>%s</code>
恢复时间：%s
离线时长：%d 分钟

✅ 转发代理已恢复正常运行

―――――――――――――

🟢 <b>Forward Agent Recovery</b>

Forward Agent: %s
ID: <code>%s</code>
Recovered at: %s
Downtime: %d min

✅ Forward Agent is back online`,
		EscapeHTML(agentName), agentSID, onlineAtStr, downtimeMinutes,
		EscapeHTML(agentName), agentSID, onlineAtStr, downtimeMinutes,
	)
}

// BuildResourceExpiringMessage builds a resource expiring notification message (HTML format)
// for agents and nodes that will expire soon
func BuildResourceExpiringMessage(agents []dto.ExpiringAgentInfo, nodes []dto.ExpiringNodeInfo) string {
	if len(agents) == 0 && len(nodes) == 0 {
		return ""
	}

	var msg string
	msg = "⏰ <b>资源即将到期提醒</b>\n"

	// Forward Agents section
	if len(agents) > 0 {
		msg += fmt.Sprintf("\n📡 <b>Forward Agent (%d个)</b>\n", len(agents))
		for _, a := range agents {
			urgencyIcon := getUrgencyIcon(a.DaysRemaining)
			expiresAtStr := biztime.FormatInBizTimezone(a.ExpiresAt, "2006-01-02")
			msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, a.SID, EscapeHTML(a.Name))
			msg += fmt.Sprintf("   └ %s (%s)\n", formatDaysRemaining(a.DaysRemaining), expiresAtStr)
			if a.CostLabel != nil && *a.CostLabel != "" {
				msg += fmt.Sprintf("   └ 费用: %s\n", EscapeHTML(*a.CostLabel))
			}
		}
	}

	// Nodes section
	if len(nodes) > 0 {
		msg += fmt.Sprintf("\n🖥 <b>Node (%d个)</b>\n", len(nodes))
		for _, n := range nodes {
			urgencyIcon := getUrgencyIcon(n.DaysRemaining)
			expiresAtStr := biztime.FormatInBizTimezone(n.ExpiresAt, "2006-01-02")
			msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, n.SID, EscapeHTML(n.Name))
			msg += fmt.Sprintf("   └ %s (%s)\n", formatDaysRemaining(n.DaysRemaining), expiresAtStr)
			if n.CostLabel != nil && *n.CostLabel != "" {
				msg += fmt.Sprintf("   └ 费用: %s\n", EscapeHTML(*n.CostLabel))
			}
		}
	}

	msg += "\n💡 请及时续费，避免服务中断"

	return msg
}

// formatDaysRemaining returns a human-readable string for days remaining
func formatDaysRemaining(days int) string {
	switch days {
	case 0:
		return "今天到期"
	case 1:
		return "明天到期"
	default:
		return fmt.Sprintf("%d天后到期", days)
	}
}

// getUrgencyIcon returns an urgency indicator emoji based on days remaining
func getUrgencyIcon(daysRemaining int) string {
	switch {
	case daysRemaining <= 1:
		return "🔴" // Critical: 1 day or less
	case daysRemaining <= 3:
		return "🟠" // Urgent: within 3 days
	default:
		return "🟡" // Warning: other
	}
}
