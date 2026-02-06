package i18n

import (
	"fmt"
	"html"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// BuildNewUserMessage builds a new user notification message (HTML format)
func BuildNewUserMessage(lang Lang, userSID, email, name, source string, createdAt time.Time) string {
	timeStr := biztime.FormatInBizTimezone(createdAt, "2006-01-02 15:04:05")

	if lang == EN {
		sourceText := "registration"
		if source != "" {
			sourceText = source
		}
		return fmt.Sprintf(`👤 <b>New User Registration</b>

Email: %s
Name: %s
ID: <code>%s</code>
Source: %s
Registered: %s`,
			html.EscapeString(email),
			html.EscapeString(name),
			userSID,
			html.EscapeString(sourceText),
			timeStr,
		)
	}

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
		html.EscapeString(email),
		html.EscapeString(name),
		userSID,
		html.EscapeString(sourceTextZH),
		timeStr,
	)
}

// BuildPaymentSuccessMessage builds a payment success notification message (HTML format)
func BuildPaymentSuccessMessage(lang Lang, paymentSID, userSID, userEmail, planName string, amount float64, currency, paymentMethod, transactionID string, paidAt time.Time) string {
	amountStr := formatAmount(amount, currency)
	timeStr := biztime.FormatInBizTimezone(paidAt, "2006-01-02 15:04:05")

	if lang == EN {
		return fmt.Sprintf(`💰 <b>Payment Successful</b>

Amount: %s
User: <code>%s</code>
Email: %s
Plan: %s
Method: %s
Transaction: %s
Paid at: %s`,
			amountStr,
			userSID,
			html.EscapeString(userEmail),
			html.EscapeString(planName),
			html.EscapeString(paymentMethod),
			html.EscapeString(transactionID),
			timeStr,
		)
	}

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
		html.EscapeString(userEmail),
		html.EscapeString(planName),
		html.EscapeString(paymentMethod),
		html.EscapeString(transactionID),
		timeStr,
	)
}

// BuildNodeOnlineMessage builds a node online notification message (HTML format)
func BuildNodeOnlineMessage(lang Lang, nodeSID, nodeName string, onlineAt time.Time) string {
	timeStr := biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05")

	if lang == EN {
		return fmt.Sprintf(`🟢 <b>Node Agent Online</b>

Node Agent: %s
ID: <code>%s</code>
Online at: %s

✅ Node Agent connection restored`,
			html.EscapeString(nodeName),
			nodeSID,
			timeStr,
		)
	}

	return fmt.Sprintf(`🟢 <b>Node Agent 上线通知</b>

Node Agent：%s
ID：<code>%s</code>
上线时间：%s

✅ Node Agent 已恢复连接`,
		html.EscapeString(nodeName),
		nodeSID,
		timeStr,
	)
}

// BuildNodeOfflineMessage builds a node offline notification message (HTML format)
func BuildNodeOfflineMessage(lang Lang, nodeSID, nodeName string, lastSeenAt time.Time, offlineMinutes int) string {
	lastSeenStr := biztime.FormatInBizTimezone(lastSeenAt, "2006-01-02 15:04:05")

	if lang == EN {
		return fmt.Sprintf(`🔴 <b>Node Agent Offline Alert</b>

Node Agent: <code>%s</code>
ID: <code>%s</code>
Last seen: %s
Offline: %d min

⚠️ Please check Node Agent status`,
			html.EscapeString(nodeName), nodeSID, lastSeenStr, offlineMinutes,
		)
	}

	return fmt.Sprintf(`🔴 <b>Node Agent 离线告警</b>

Node Agent：<code>%s</code>
ID：<code>%s</code>
最后在线：%s
离线时长：%d 分钟

⚠️ 请检查 Node Agent 状态`,
		html.EscapeString(nodeName), nodeSID, lastSeenStr, offlineMinutes,
	)
}

// BuildAgentOnlineMessage builds a forward agent online notification message (HTML format)
func BuildAgentOnlineMessage(lang Lang, agentSID, agentName string, onlineAt time.Time) string {
	timeStr := biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05")

	if lang == EN {
		return fmt.Sprintf(`🟢 <b>Forward Agent Online</b>

Forward Agent: %s
ID: <code>%s</code>
Online at: %s

✅ Forward Agent connection restored`,
			html.EscapeString(agentName),
			agentSID,
			timeStr,
		)
	}

	return fmt.Sprintf(`🟢 <b>转发代理上线通知</b>

转发代理：%s
ID：<code>%s</code>
上线时间：%s

✅ 转发代理已恢复连接`,
		html.EscapeString(agentName),
		agentSID,
		timeStr,
	)
}

// BuildAgentOfflineMessage builds a forward agent offline notification message (HTML format)
func BuildAgentOfflineMessage(lang Lang, agentSID, agentName string, lastSeenAt time.Time, offlineMinutes int) string {
	lastSeenStr := biztime.FormatInBizTimezone(lastSeenAt, "2006-01-02 15:04:05")

	if lang == EN {
		return fmt.Sprintf(`🔴 <b>Forward Agent Offline Alert</b>

Forward Agent: <code>%s</code>
ID: <code>%s</code>
Last seen: %s
Offline: %d min

⚠️ Please check forward agent status`,
			html.EscapeString(agentName), agentSID, lastSeenStr, offlineMinutes,
		)
	}

	return fmt.Sprintf(`🔴 <b>转发代理离线告警</b>

转发代理：<code>%s</code>
ID：<code>%s</code>
最后在线：%s
离线时长：%d 分钟

⚠️ 请检查转发代理状态`,
		html.EscapeString(agentName), agentSID, lastSeenStr, offlineMinutes,
	)
}

// BuildNodeRecoveryMessage builds a node recovery notification message (HTML format)
// This is sent when a node transitions from Firing state back to Normal
func BuildNodeRecoveryMessage(lang Lang, nodeSID, nodeName string, onlineAt time.Time, downtimeMinutes int64) string {
	onlineAtStr := biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05")

	if lang == EN {
		return fmt.Sprintf(`🟢 <b>Node Agent Recovery</b>

Node Agent: %s
ID: <code>%s</code>
Recovered at: %s
Downtime: %d min

✅ Node Agent is back online`,
			html.EscapeString(nodeName), nodeSID, onlineAtStr, downtimeMinutes,
		)
	}

	return fmt.Sprintf(`🟢 <b>Node Agent 恢复通知</b>

Node Agent：%s
ID：<code>%s</code>
恢复时间：%s
离线时长：%d 分钟

✅ Node Agent 已恢复正常运行`,
		html.EscapeString(nodeName), nodeSID, onlineAtStr, downtimeMinutes,
	)
}

// BuildAgentRecoveryMessage builds a forward agent recovery notification message (HTML format)
// This is sent when an agent transitions from Firing state back to Normal
func BuildAgentRecoveryMessage(lang Lang, agentSID, agentName string, onlineAt time.Time, downtimeMinutes int64) string {
	onlineAtStr := biztime.FormatInBizTimezone(onlineAt, "2006-01-02 15:04:05")

	if lang == EN {
		return fmt.Sprintf(`🟢 <b>Forward Agent Recovery</b>

Forward Agent: %s
ID: <code>%s</code>
Recovered at: %s
Downtime: %d min

✅ Forward Agent is back online`,
			html.EscapeString(agentName), agentSID, onlineAtStr, downtimeMinutes,
		)
	}

	return fmt.Sprintf(`🟢 <b>转发代理恢复通知</b>

转发代理：%s
ID：<code>%s</code>
恢复时间：%s
离线时长：%d 分钟

✅ 转发代理已恢复正常运行`,
		html.EscapeString(agentName), agentSID, onlineAtStr, downtimeMinutes,
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
