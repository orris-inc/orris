package usecases

import (
	"fmt"
	"html"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// escapeHTML escapes HTML special characters
// to prevent format injection from user-provided data
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

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
		escapeHTML(email),
		escapeHTML(name),
		userSID,
		escapeHTML(sourceTextZH),
		biztime.FormatInBizTimezone(createdAt, "2006-01-02 15:04:05"),
	)
}

// BuildPaymentSuccessMessage builds a payment success notification message (HTML format)
func BuildPaymentSuccessMessage(paymentSID, userSID, userEmail, planName string, amount float64, currency, paymentMethod, transactionID string, paidAt time.Time) string {
	amountStr := formatAmount(amount, currency)

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
		escapeHTML(userEmail),
		escapeHTML(planName),
		escapeHTML(paymentMethod),
		escapeHTML(transactionID),
		biztime.FormatInBizTimezone(paidAt, "2006-01-02 15:04:05"),
	)
}

// BuildNodeOnlineMessage builds a node online notification message (HTML format)
func BuildNodeOnlineMessage(nodeSID, nodeName string, onlineAt time.Time) string {
	return fmt.Sprintf(`🟢 <b>Node Agent 上线通知</b>

Node Agent：%s
ID：<code>%s</code>
上线时间：%s

✅ Node Agent 已恢复连接`,
		escapeHTML(nodeName),
		nodeSID,
		biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05"),
	)
}

// BuildNodeOfflineMessage builds a node offline notification message (HTML format)
func BuildNodeOfflineMessage(nodeSID, nodeName string, lastSeenAt time.Time, offlineMinutes int) string {
	return fmt.Sprintf(`🔴 <b>Node Agent 离线告警</b>

Node Agent：%s
ID：<code>%s</code>
最后在线：%s
离线时长：%d 分钟

⚠️ 请检查 Node Agent 状态`,
		escapeHTML(nodeName),
		nodeSID,
		biztime.FormatInBizTimezone(lastSeenAt, "2006-01-02 15:04:05"),
		offlineMinutes,
	)
}

// BuildAgentOnlineMessage builds a forward agent online notification message (HTML format)
func BuildAgentOnlineMessage(agentSID, agentName string, onlineAt time.Time) string {
	return fmt.Sprintf(`🟢 <b>转发代理上线通知</b>

转发代理：%s
ID：<code>%s</code>
上线时间：%s

✅ 转发代理已恢复连接`,
		escapeHTML(agentName),
		agentSID,
		biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05"),
	)
}

// BuildAgentOfflineMessage builds a forward agent offline notification message (HTML format)
func BuildAgentOfflineMessage(agentSID, agentName string, lastSeenAt time.Time, offlineMinutes int) string {
	return fmt.Sprintf(`🔴 <b>转发代理离线告警</b>

转发代理：%s
ID：<code>%s</code>
最后在线：%s
离线时长：%d 分钟

⚠️ 请检查转发代理状态`,
		escapeHTML(agentName),
		agentSID,
		biztime.FormatInBizTimezone(lastSeenAt, "2006-01-02 15:04:05"),
		offlineMinutes,
	)
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
	amountStr := formatAmount(revenue, currency)

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
	amountStr := formatAmount(revenue, currency)
	userChange := formatPercentChange(userChangePercent)
	orderChange := formatPercentChange(orderChangePercent)
	revenueChange := formatPercentChange(revenueChangePercent)

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

// formatAmount formats amount to display string
// amount is in main currency unit (e.g., 99.00), not cents
func formatAmount(amount float64, currency string) string {
	if currency == "" {
		currency = "CNY"
	}

	symbol := "¥"
	switch currency {
	case "USD":
		symbol = "$"
	case "EUR":
		symbol = "€"
	case "GBP":
		symbol = "£"
	case "JPY":
		return fmt.Sprintf("¥%.0f", amount) // JPY doesn't use decimals
	}

	return fmt.Sprintf("%s%.2f", symbol, amount)
}

// formatPercentChange formats percent change with color indicator
func formatPercentChange(percent float64) string {
	if percent > 0 {
		return fmt.Sprintf("📈 +%.1f%%", percent)
	} else if percent < 0 {
		return fmt.Sprintf("📉 %.1f%%", percent)
	}
	return "➡️ 0%"
}
