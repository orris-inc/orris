package i18n

import (
	"fmt"
	"html"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// ExpiringResourceInfo holds lightweight resource info for building expiring notification messages.
type ExpiringResourceInfo struct {
	SID           string
	Name          string
	ExpiresAt     time.Time
	DaysRemaining int
	CostLabel     *string
}

// BuildResourceExpiringMessage builds a resource expiring notification message (HTML format)
// for agents and nodes that will expire soon.
func BuildResourceExpiringMessage(lang Lang, agents []ExpiringResourceInfo, nodes []ExpiringResourceInfo) string {
	if len(agents) == 0 && len(nodes) == 0 {
		return ""
	}

	var msg string

	if lang == EN {
		msg = "⏰ <b>Resource Expiration Reminder</b>\n"

		// Forward Agents section
		if len(agents) > 0 {
			msg += fmt.Sprintf("\n📡 <b>Forward Agent (%d)</b>\n<blockquote>", len(agents))
			for i, a := range agents {
				if i > 0 {
					msg += "\n"
				}
				urgencyIcon := getUrgencyIcon(a.DaysRemaining)
				expiresAtStr := biztime.FormatInBizTimezone(a.ExpiresAt, "2006-01-02")
				msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, html.EscapeString(a.SID), html.EscapeString(a.Name))
				msg += fmt.Sprintf("   └ %s (%s)", formatDaysRemainingEN(a.DaysRemaining), expiresAtStr)
				if a.CostLabel != nil && *a.CostLabel != "" {
					msg += fmt.Sprintf("\n   └ Cost: %s", html.EscapeString(*a.CostLabel))
				}
			}
			msg += "</blockquote>\n"
		}

		// Nodes section
		if len(nodes) > 0 {
			msg += fmt.Sprintf("\n🖥 <b>Node (%d)</b>\n<blockquote>", len(nodes))
			for i, n := range nodes {
				if i > 0 {
					msg += "\n"
				}
				urgencyIcon := getUrgencyIcon(n.DaysRemaining)
				expiresAtStr := biztime.FormatInBizTimezone(n.ExpiresAt, "2006-01-02")
				msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, html.EscapeString(n.SID), html.EscapeString(n.Name))
				msg += fmt.Sprintf("   └ %s (%s)", formatDaysRemainingEN(n.DaysRemaining), expiresAtStr)
				if n.CostLabel != nil && *n.CostLabel != "" {
					msg += fmt.Sprintf("\n   └ Cost: %s", html.EscapeString(*n.CostLabel))
				}
			}
			msg += "</blockquote>\n"
		}

		msg += "\n💡 Please renew in time to avoid service interruption"
	} else {
		msg = "⏰ <b>资源即将到期提醒</b>\n"

		// Forward Agents section
		if len(agents) > 0 {
			msg += fmt.Sprintf("\n📡 <b>Forward Agent (%d个)</b>\n<blockquote>", len(agents))
			for i, a := range agents {
				if i > 0 {
					msg += "\n"
				}
				urgencyIcon := getUrgencyIcon(a.DaysRemaining)
				expiresAtStr := biztime.FormatInBizTimezone(a.ExpiresAt, "2006-01-02")
				msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, html.EscapeString(a.SID), html.EscapeString(a.Name))
				msg += fmt.Sprintf("   └ %s (%s)", formatDaysRemainingZH(a.DaysRemaining), expiresAtStr)
				if a.CostLabel != nil && *a.CostLabel != "" {
					msg += fmt.Sprintf("\n   └ 费用: %s", html.EscapeString(*a.CostLabel))
				}
			}
			msg += "</blockquote>\n"
		}

		// Nodes section
		if len(nodes) > 0 {
			msg += fmt.Sprintf("\n🖥 <b>Node (%d个)</b>\n<blockquote>", len(nodes))
			for i, n := range nodes {
				if i > 0 {
					msg += "\n"
				}
				urgencyIcon := getUrgencyIcon(n.DaysRemaining)
				expiresAtStr := biztime.FormatInBizTimezone(n.ExpiresAt, "2006-01-02")
				msg += fmt.Sprintf("%s <code>%s</code> - %s\n", urgencyIcon, html.EscapeString(n.SID), html.EscapeString(n.Name))
				msg += fmt.Sprintf("   └ %s (%s)", formatDaysRemainingZH(n.DaysRemaining), expiresAtStr)
				if n.CostLabel != nil && *n.CostLabel != "" {
					msg += fmt.Sprintf("\n   └ 费用: %s", html.EscapeString(*n.CostLabel))
				}
			}
			msg += "</blockquote>\n"
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
