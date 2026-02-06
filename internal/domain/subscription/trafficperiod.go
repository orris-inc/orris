package subscription

import (
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// TrafficResetMode determines how traffic usage periods are calculated.
type TrafficResetMode string

const (
	// TrafficResetCalendarMonth resets traffic on calendar month boundaries in business timezone.
	TrafficResetCalendarMonth TrafficResetMode = "calendar_month"
	// TrafficResetBillingCycle resets traffic on subscription billing cycle boundaries.
	TrafficResetBillingCycle TrafficResetMode = "billing_cycle"
)

// TrafficPeriod represents a time range for traffic usage calculation.
type TrafficPeriod struct {
	Start time.Time
	End   time.Time
}

// GetTrafficResetMode extracts the traffic reset mode from a plan's features.
// Returns TrafficResetCalendarMonth if plan is nil, features are missing, or mode is invalid.
func GetTrafficResetMode(plan *Plan) TrafficResetMode {
	if plan == nil || plan.Features() == nil {
		return TrafficResetCalendarMonth
	}

	mode := plan.Features().GetTrafficResetMode()
	switch TrafficResetMode(mode) {
	case TrafficResetBillingCycle:
		return TrafficResetBillingCycle
	default:
		return TrafficResetCalendarMonth
	}
}

// ResolveTrafficPeriod returns the traffic usage period based on the plan's reset mode
// and the subscription's billing cycle.
//
// For calendar_month: uses business timezone month boundaries (backward compatible default).
// For billing_cycle: uses the subscription's CurrentPeriodStart/CurrentPeriodEnd.
// Falls back to calendar_month if sub is nil.
func ResolveTrafficPeriod(plan *Plan, sub *Subscription) TrafficPeriod {
	mode := GetTrafficResetMode(plan)

	if mode == TrafficResetBillingCycle && sub != nil {
		return TrafficPeriod{
			Start: sub.CurrentPeriodStart(),
			End:   sub.CurrentPeriodEnd(),
		}
	}

	// Fallback: calendar month (default, or billing_cycle with nil sub)
	bizNow := biztime.ToBizTimezone(biztime.NowUTC())
	return TrafficPeriod{
		Start: biztime.StartOfMonthUTC(bizNow.Year(), bizNow.Month()),
		End:   biztime.EndOfMonthUTC(bizNow.Year(), bizNow.Month()),
	}
}
