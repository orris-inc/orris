package i18n

import (
	"fmt"
	"html"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// BuildNewUserMessage builds a new user notification message (HTML format)
func BuildNewUserMessage(lang Lang, userSID, email, name, source string, createdAt time.Time) string {
	timeStr := html.EscapeString(biztime.FormatInBizTimezone(createdAt, "2006-01-02 15:04:05"))

	if lang == EN {
		sourceText := "registration"
		if source != "" {
			sourceText = source
		}
		return fmt.Sprintf("👤 <b>New User Registration</b>\n\n"+
			"<blockquote>Email: %s\n"+
			"Name: %s\n"+
			"ID: <code>%s</code>\n"+
			"Source: %s\n"+
			"Registered: %s</blockquote>",
			html.EscapeString(email),
			html.EscapeString(name),
			html.EscapeString(userSID),
			html.EscapeString(sourceText),
			timeStr,
		)
	}

	sourceTextZH := "注册"
	if source != "" {
		sourceTextZH = source
	}
	return fmt.Sprintf("👤 <b>新用户注册</b>\n\n"+
		"<blockquote>邮箱：%s\n"+
		"名称：%s\n"+
		"ID：<code>%s</code>\n"+
		"来源：%s\n"+
		"注册时间：%s</blockquote>",
		html.EscapeString(email),
		html.EscapeString(name),
		html.EscapeString(userSID),
		html.EscapeString(sourceTextZH),
		timeStr,
	)
}

// BuildPaymentSuccessMessage builds a payment success notification message (HTML format)
func BuildPaymentSuccessMessage(lang Lang, paymentSID, userSID, userEmail, planName string, amount float64, currency, paymentMethod, transactionID string, paidAt time.Time) string {
	amountStr := html.EscapeString(formatAmount(amount, currency))
	timeStr := html.EscapeString(biztime.FormatInBizTimezone(paidAt, "2006-01-02 15:04:05"))

	if lang == EN {
		return fmt.Sprintf("💰 <b>Payment Successful</b>\n\n"+
			"<blockquote>Amount: %s\n"+
			"User: <code>%s</code>\n"+
			"Email: %s\n"+
			"Plan: %s\n"+
			"Method: %s\n"+
			"Transaction: %s\n"+
			"Paid at: %s</blockquote>",
			amountStr,
			html.EscapeString(userSID),
			html.EscapeString(userEmail),
			html.EscapeString(planName),
			html.EscapeString(paymentMethod),
			html.EscapeString(transactionID),
			timeStr,
		)
	}

	return fmt.Sprintf("💰 <b>支付成功</b>\n\n"+
		"<blockquote>金额：%s\n"+
		"用户：<code>%s</code>\n"+
		"邮箱：%s\n"+
		"套餐：%s\n"+
		"支付方式：%s\n"+
		"交易号：%s\n"+
		"支付时间：%s</blockquote>",
		amountStr,
		html.EscapeString(userSID),
		html.EscapeString(userEmail),
		html.EscapeString(planName),
		html.EscapeString(paymentMethod),
		html.EscapeString(transactionID),
		timeStr,
	)
}

// BuildNodeOnlineMessage builds a node online notification message (HTML format)
func BuildNodeOnlineMessage(lang Lang, nodeSID, nodeName string, onlineAt time.Time) string {
	timeStr := html.EscapeString(biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05"))

	if lang == EN {
		return fmt.Sprintf("🟢 <b>Node Agent Online</b>\n\n"+
			"<blockquote>Node Agent: %s\n"+
			"ID: <code>%s</code>\n"+
			"Online at: %s</blockquote>\n\n"+
			"✅ Node Agent connection restored",
			html.EscapeString(nodeName),
			html.EscapeString(nodeSID),
			timeStr,
		)
	}

	return fmt.Sprintf("🟢 <b>Node Agent 上线通知</b>\n\n"+
		"<blockquote>Node Agent：%s\n"+
		"ID：<code>%s</code>\n"+
		"上线时间：%s</blockquote>\n\n"+
		"✅ Node Agent 已恢复连接",
		html.EscapeString(nodeName),
		html.EscapeString(nodeSID),
		timeStr,
	)
}

// BuildNodeOfflineMessage builds a node offline notification message (HTML format)
func BuildNodeOfflineMessage(lang Lang, nodeSID, nodeName string, lastSeenAt time.Time, offlineMinutes int) string {
	lastSeenStr := html.EscapeString(biztime.FormatInBizTimezone(lastSeenAt, "2006-01-02 15:04:05"))

	if lang == EN {
		return fmt.Sprintf("🔴 <b>Node Agent Offline Alert</b>\n\n"+
			"<blockquote>Node Agent: <code>%s</code>\n"+
			"ID: <code>%s</code>\n"+
			"Last seen: %s\n"+
			"Offline: %d min</blockquote>\n\n"+
			"⚠️ Please check Node Agent status",
			html.EscapeString(nodeName), html.EscapeString(nodeSID), lastSeenStr, offlineMinutes,
		)
	}

	return fmt.Sprintf("🔴 <b>Node Agent 离线告警</b>\n\n"+
		"<blockquote>Node Agent：<code>%s</code>\n"+
		"ID：<code>%s</code>\n"+
		"最后在线：%s\n"+
		"离线时长：%d 分钟</blockquote>\n\n"+
		"⚠️ 请检查 Node Agent 状态",
		html.EscapeString(nodeName), html.EscapeString(nodeSID), lastSeenStr, offlineMinutes,
	)
}

// BuildAgentOnlineMessage builds a forward agent online notification message (HTML format)
func BuildAgentOnlineMessage(lang Lang, agentSID, agentName string, onlineAt time.Time) string {
	timeStr := html.EscapeString(biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05"))

	if lang == EN {
		return fmt.Sprintf("🟢 <b>Forward Agent Online</b>\n\n"+
			"<blockquote>Forward Agent: %s\n"+
			"ID: <code>%s</code>\n"+
			"Online at: %s</blockquote>\n\n"+
			"✅ Forward Agent connection restored",
			html.EscapeString(agentName),
			html.EscapeString(agentSID),
			timeStr,
		)
	}

	return fmt.Sprintf("🟢 <b>转发代理上线通知</b>\n\n"+
		"<blockquote>转发代理：%s\n"+
		"ID：<code>%s</code>\n"+
		"上线时间：%s</blockquote>\n\n"+
		"✅ 转发代理已恢复连接",
		html.EscapeString(agentName),
		html.EscapeString(agentSID),
		timeStr,
	)
}

// BuildAgentOfflineMessage builds a forward agent offline notification message (HTML format)
func BuildAgentOfflineMessage(lang Lang, agentSID, agentName string, lastSeenAt time.Time, offlineMinutes int) string {
	lastSeenStr := html.EscapeString(biztime.FormatInBizTimezone(lastSeenAt, "2006-01-02 15:04:05"))

	if lang == EN {
		return fmt.Sprintf("🔴 <b>Forward Agent Offline Alert</b>\n\n"+
			"<blockquote>Forward Agent: <code>%s</code>\n"+
			"ID: <code>%s</code>\n"+
			"Last seen: %s\n"+
			"Offline: %d min</blockquote>\n\n"+
			"⚠️ Please check forward agent status",
			html.EscapeString(agentName), html.EscapeString(agentSID), lastSeenStr, offlineMinutes,
		)
	}

	return fmt.Sprintf("🔴 <b>转发代理离线告警</b>\n\n"+
		"<blockquote>转发代理：<code>%s</code>\n"+
		"ID：<code>%s</code>\n"+
		"最后在线：%s\n"+
		"离线时长：%d 分钟</blockquote>\n\n"+
		"⚠️ 请检查转发代理状态",
		html.EscapeString(agentName), html.EscapeString(agentSID), lastSeenStr, offlineMinutes,
	)
}

// BuildNodeRecoveryMessage builds a node recovery notification message (HTML format)
// This is sent when a node transitions from Firing state back to Normal
func BuildNodeRecoveryMessage(lang Lang, nodeSID, nodeName string, onlineAt time.Time, downtimeMinutes int64) string {
	onlineAtStr := html.EscapeString(biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05"))

	if lang == EN {
		return fmt.Sprintf("🟢 <b>Node Agent Recovery</b>\n\n"+
			"<blockquote>Node Agent: %s\n"+
			"ID: <code>%s</code>\n"+
			"Recovered at: %s\n"+
			"Downtime: %d min</blockquote>\n\n"+
			"✅ Node Agent is back online",
			html.EscapeString(nodeName), html.EscapeString(nodeSID), onlineAtStr, downtimeMinutes,
		)
	}

	return fmt.Sprintf("🟢 <b>Node Agent 恢复通知</b>\n\n"+
		"<blockquote>Node Agent：%s\n"+
		"ID：<code>%s</code>\n"+
		"恢复时间：%s\n"+
		"离线时长：%d 分钟</blockquote>\n\n"+
		"✅ Node Agent 已恢复正常运行",
		html.EscapeString(nodeName), html.EscapeString(nodeSID), onlineAtStr, downtimeMinutes,
	)
}

// BuildAgentRecoveryMessage builds a forward agent recovery notification message (HTML format)
// This is sent when an agent transitions from Firing state back to Normal
func BuildAgentRecoveryMessage(lang Lang, agentSID, agentName string, onlineAt time.Time, downtimeMinutes int64) string {
	onlineAtStr := html.EscapeString(biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05"))

	if lang == EN {
		return fmt.Sprintf("🟢 <b>Forward Agent Recovery</b>\n\n"+
			"<blockquote>Forward Agent: %s\n"+
			"ID: <code>%s</code>\n"+
			"Recovered at: %s\n"+
			"Downtime: %d min</blockquote>\n\n"+
			"✅ Forward Agent is back online",
			html.EscapeString(agentName), html.EscapeString(agentSID), onlineAtStr, downtimeMinutes,
		)
	}

	return fmt.Sprintf("🟢 <b>转发代理恢复通知</b>\n\n"+
		"<blockquote>转发代理：%s\n"+
		"ID：<code>%s</code>\n"+
		"恢复时间：%s\n"+
		"离线时长：%d 分钟</blockquote>\n\n"+
		"✅ 转发代理已恢复正常运行",
		html.EscapeString(agentName), html.EscapeString(agentSID), onlineAtStr, downtimeMinutes,
	)
}

// BuildMuteKeyboard builds an inline keyboard with mute button for offline alerts.
// Returns any to avoid circular import with the telegram package which defines
// its own exported InlineKeyboardMarkup type.
func BuildMuteKeyboard(lang Lang, resourceType, resourceSID string) any {
	callbackData := fmt.Sprintf("mute:%s:%s", resourceType, resourceSID)

	text := "🔕 静默此通知"
	if lang == EN {
		text = "🔕 Mute"
	}

	return &inlineKeyboardMarkup{
		InlineKeyboard: [][]inlineKeyboardButton{
			{
				{
					Text:         text,
					CallbackData: callbackData,
				},
			},
		},
	}
}

// BuildUnmuteKeyboard builds an inline keyboard with unmute button for muted alerts.
func BuildUnmuteKeyboard(lang Lang, resourceType, resourceSID string) any {
	callbackData := fmt.Sprintf("unmute:%s:%s", resourceType, resourceSID)

	text := "🔔 解除静默"
	if lang == EN {
		text = "🔔 Unmute"
	}

	return &inlineKeyboardMarkup{
		InlineKeyboard: [][]inlineKeyboardButton{
			{
				{
					Text:         text,
					CallbackData: callbackData,
				},
			},
		},
	}
}

// inlineKeyboardMarkup is a local type for building Telegram inline keyboards.
// The telegram package has its own exported types; this avoids circular imports.
type inlineKeyboardMarkup struct {
	InlineKeyboard [][]inlineKeyboardButton `json:"inline_keyboard"`
}

// inlineKeyboardButton represents a button in an inline keyboard.
type inlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// formatAmount formats amount to display string.
// amount is in main currency unit (e.g., 99.00), not cents.
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
