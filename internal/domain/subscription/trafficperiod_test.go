package subscription

import (
	"testing"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trafficCyclePlan builds a Plan whose traffic_reset_mode is the given value.
// Pass empty string to leave the mode unset (caller verifies fallback behavior).
func trafficCyclePlan(t *testing.T, mode string) *Plan {
	t.Helper()

	plan, err := NewPlan("Test Plan", "test", "desc", vo.PlanTypeNode)
	require.NoError(t, err)

	features := vo.NewPlanFeatures(nil)
	if mode != "" {
		require.NoError(t, features.SetTrafficResetMode(mode))
	}
	require.NoError(t, plan.UpdateFeatures(features))
	return plan
}

// trafficCycleSubscription builds an active monthly subscription with the
// given billing-period bounds. The cycle dates are also used as start/end so
// the subscription validates without renewal handling.
func trafficCycleSubscription(t *testing.T, periodStart, periodEnd time.Time) *Subscription {
	t.Helper()

	bc, err := vo.NewBillingCycle("monthly")
	require.NoError(t, err)

	sub, err := ReconstructSubscriptionWithParams(SubscriptionReconstructParams{
		ID:                 1,
		UserID:             10,
		PlanID:             100,
		SubjectType:        "user",
		SubjectID:          10,
		SID:                "sub_test",
		UUID:               "00000000-0000-0000-0000-000000000001",
		LinkToken:          "dGVzdHRva2VuMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkw",
		Status:             vo.StatusActive,
		StartDate:          periodStart,
		EndDate:            periodEnd,
		AutoRenew:          true,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		BillingCycle:       bc,
		Version:            1,
		CreatedAt:          periodStart,
		UpdatedAt:          periodStart,
	})
	require.NoError(t, err)
	return sub
}

func TestResolveTrafficPeriod_CalendarMonth_DiffersFromBillingPeriod(t *testing.T) {
	// Subscription billing window mid-Jan to mid-Feb; calendar_month plan
	// must resolve to the business-tz calendar month containing "now",
	// NOT the subscription billing period.
	periodStart := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)

	plan := trafficCyclePlan(t, "calendar_month")
	sub := trafficCycleSubscription(t, periodStart, periodEnd)

	got := ResolveTrafficPeriod(plan, sub)

	bizNow := biztime.ToBizTimezone(biztime.NowUTC())
	expectedStart := biztime.StartOfMonthUTC(bizNow.Year(), bizNow.Month())
	expectedEnd := biztime.EndOfMonthUTC(bizNow.Year(), bizNow.Month())

	// Today's calendar month start is far from Jan 15: confirm we're not
	// silently returning the billing period.
	assert.Equal(t, expectedStart, got.Start, "calendar_month plan must use calendar month start")
	assert.Equal(t, expectedEnd, got.End, "calendar_month plan must use calendar month end")
	assert.NotEqual(t, sub.CurrentPeriodStart(), got.Start, "calendar_month must NOT use billing period start")
}

func TestResolveTrafficPeriod_CalendarMonth_FloorsToManualReset(t *testing.T) {
	// If a manual reset moves CurrentPeriodStart past the calendar month
	// start, that reset wins as a floor (excludes pre-reset traffic).
	bizNow := biztime.ToBizTimezone(biztime.NowUTC())
	monthStart := biztime.StartOfMonthUTC(bizNow.Year(), bizNow.Month())
	resetAt := monthStart.AddDate(0, 0, 5) // 5 days into the month

	plan := trafficCyclePlan(t, "calendar_month")
	sub := trafficCycleSubscription(t, resetAt, resetAt.AddDate(0, 1, 0))

	got := ResolveTrafficPeriod(plan, sub)

	assert.Equal(t, resetAt, got.Start, "manual reset (post month-start) must floor the cycle start")
	assert.Equal(t, biztime.EndOfMonthUTC(bizNow.Year(), bizNow.Month()), got.End)
}

func TestResolveTrafficPeriod_BillingCycle_UsesSubscriptionPeriod(t *testing.T) {
	periodStart := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)

	plan := trafficCyclePlan(t, "billing_cycle")
	sub := trafficCycleSubscription(t, periodStart, periodEnd)

	got := ResolveTrafficPeriod(plan, sub)

	assert.Equal(t, periodStart, got.Start)
	assert.Equal(t, periodEnd, got.End)
}

func TestResolveTrafficPeriod_FallsBackToCalendarMonthOnNilPlan(t *testing.T) {
	periodStart := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	sub := trafficCycleSubscription(t, periodStart, periodEnd)

	got := ResolveTrafficPeriod(nil, sub)

	bizNow := biztime.ToBizTimezone(biztime.NowUTC())
	assert.Equal(t, biztime.StartOfMonthUTC(bizNow.Year(), bizNow.Month()), got.Start)
	assert.Equal(t, biztime.EndOfMonthUTC(bizNow.Year(), bizNow.Month()), got.End)
}

func TestResolveTrafficPeriod_LifetimeAlwaysUsesSubscriptionPeriod(t *testing.T) {
	// Lifetime subscriptions must NEVER be reset by calendar_month even if
	// the plan declares calendar_month — they accumulate from start to end.
	periodStart := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	bc, err := vo.NewBillingCycle("lifetime")
	require.NoError(t, err)
	sub, err := ReconstructSubscriptionWithParams(SubscriptionReconstructParams{
		ID:                 2,
		UserID:             10,
		PlanID:             100,
		SubjectType:        "user",
		SubjectID:          10,
		SID:                "sub_lifetime",
		UUID:               "00000000-0000-0000-0000-000000000002",
		LinkToken:          "dGVzdHRva2VuMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkx",
		Status:             vo.StatusActive,
		StartDate:          periodStart,
		EndDate:            periodEnd,
		AutoRenew:          false,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		BillingCycle:       bc,
		Version:            1,
		CreatedAt:          periodStart,
		UpdatedAt:          periodStart,
	})
	require.NoError(t, err)

	plan := trafficCyclePlan(t, "calendar_month")
	got := ResolveTrafficPeriod(plan, sub)

	assert.Equal(t, periodStart, got.Start)
	assert.Equal(t, periodEnd, got.End)
}
