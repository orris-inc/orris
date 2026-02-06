package i18n

import (
	"fmt"
	"html"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// BuildResourceExpiringMessage builds a resource expiring notification message (HTML format)
// for agents and nodes that will expire soon
func BuildResourceExpiringMessage(lang Lang, agents []dto.ExpiringAgentInfo, nodes []dto.ExpiringNodeInfo) string {
	if len(agents) == 0 && len(nodes) == 0 {
		return ""
	}

	var msg string

	if lang == EN {
		msg = "⏰ <b>Resource Expiration Reminder</b>\n"

		// Forward Agents section
		if len(agents) > 0 {
			msg += fmt.Sprintf("\n📡 <b>Forward Agent (%d)</b>\n", len(agents))
			for _, a := range agents {
				urgencyIcon := getUrgencyIcon(a.DaysRemaining)
				expiresAtStr := biztime.FormatInBizTimezone(a.ExpiresAt, "2006-01-02")
				msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, a.SID, html.EscapeString(a.Name))
				msg += fmt.Sprintf("   └ %s (%s)\n", formatDaysRemainingEN(a.DaysRemaining), expiresAtStr)
				if a.CostLabel != nil && *a.CostLabel != "" {
					msg += fmt.Sprintf("   └ Cost: %s\n", html.EscapeString(*a.CostLabel))
				}
			}
		}

		// Nodes section
		if len(nodes) > 0 {
			msg += fmt.Sprintf("\n🖥 <b>Node (%d)</b>\n", len(nodes))
			for _, n := range nodes {
				urgencyIcon := getUrgencyIcon(n.DaysRemaining)
				expiresAtStr := biztime.FormatInBizTimezone(n.ExpiresAt, "2006-01-02")
				msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, n.SID, html.EscapeString(n.Name))
				msg += fmt.Sprintf("   └ %s (%s)\n", formatDaysRemainingEN(n.DaysRemaining), expiresAtStr)
				if n.CostLabel != nil && *n.CostLabel != "" {
					msg += fmt.Sprintf("   └ Cost: %s\n", html.EscapeString(*n.CostLabel))
				}
			}
		}

		msg += "\n💡 Please renew in time to avoid service interruption"
	} else {
		msg = "⏰ <b>资源即将到期提醒</b>\n"

		// Forward Agents section
		if len(agents) > 0 {
			msg += fmt.Sprintf("\n📡 <b>Forward Agent (%d个)</b>\n", len(agents))
			for _, a := range agents {
				urgencyIcon := getUrgencyIcon(a.DaysRemaining)
				expiresAtStr := biztime.FormatInBizTimezone(a.ExpiresAt, "2006-01-02")
				msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, a.SID, html.EscapeString(a.Name))
				msg += fmt.Sprintf("   └ %s (%s)\n", formatDaysRemainingZH(a.DaysRemaining), expiresAtStr)
				if a.CostLabel != nil && *a.CostLabel != "" {
					msg += fmt.Sprintf("   └ 费用: %s\n", html.EscapeString(*a.CostLabel))
				}
			}
		}

		// Nodes section
		if len(nodes) > 0 {
			msg += fmt.Sprintf("\n🖥 <b>Node (%d个)</b>\n", len(nodes))
			for _, n := range nodes {
				urgencyIcon := getUrgencyIcon(n.DaysRemaining)
				expiresAtStr := biztime.FormatInBizTimezone(n.ExpiresAt, "2006-01-02")
				msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, n.SID, html.EscapeString(n.Name))
				msg += fmt.Sprintf("   └ %s (%s)\n", formatDaysRemainingZH(n.DaysRemaining), expiresAtStr)
				if n.CostLabel != nil && *n.CostLabel != "" {
					msg += fmt.Sprintf("   └ 费用: %s\n", html.EscapeString(*n.CostLabel))
				}
			}
		}

		msg += "\n💡 请及时续费，避免服务中断"
	}

	return msg
}

// formatDaysRemainingZH returns a Chinese human-readable string for days remaining
func formatDaysRemainingZH(days int) string {
	switch days {
	case 0:
		return "今天到期"
	case 1:
		return "明天到期"
	default:
		return fmt.Sprintf("%d天后到期", days)
	}
}

// formatDaysRemainingEN returns an English human-readable string for days remaining
func formatDaysRemainingEN(days int) string {
	switch days {
	case 0:
		return "expires today"
	case 1:
		return "expires tomorrow"
	default:
		return fmt.Sprintf("expires in %d days", days)
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
