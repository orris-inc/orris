package subscription

import (
	"testing"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func newBillingCycle(t *testing.T, value string) *vo.BillingCycle {
	t.Helper()
	bc, err := vo.NewBillingCycle(value)
	require.NoError(t, err)
	return bc
}

func newValidSubscription(t *testing.T) *Subscription {
	t.Helper()
	start := time.Now().UTC()
	end := start.AddDate(0, 1, 0)
	bc := newBillingCycle(t, "monthly")
	sub, err := NewSubscription(1, 1, start, end, true, bc)
	require.NoError(t, err)
	require.NotNil(t, sub)
	return sub
}

func newActiveSubscription(t *testing.T) *Subscription {
	t.Helper()
	sub := newValidSubscription(t)
	require.NoError(t, sub.Activate())
	return sub
}

// reconstructSubscription builds a Subscription from SubscriptionReconstructParams with
// sensible defaults. Callers can override fields before calling this helper.
func reconstructSubscription(t *testing.T, status vo.SubscriptionStatus, startDate, endDate time.Time) *Subscription {
	t.Helper()
	sub, err := ReconstructSubscriptionWithParams(SubscriptionReconstructParams{
		ID:                 1,
		UserID:             10,
		PlanID:             100,
		SubjectType:        "user",
		SubjectID:          10,
		SID:                "sub_test123",
		UUID:               "00000000-0000-0000-0000-000000000001",
		LinkToken:          "dGVzdHRva2VuMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkw",
		Status:             status,
		StartDate:          startDate,
		EndDate:            endDate,
		AutoRenew:          true,
		CurrentPeriodStart: startDate,
		CurrentPeriodEnd:   endDate,
		Version:            1,
		CreatedAt:          startDate,
		UpdatedAt:          startDate,
	})
	require.NoError(t, err)
	return sub
}

// =====================================================================
// TestNewSubscription_*
// =====================================================================

func TestNewSubscription_ValidInput(t *testing.T) {
	start := time.Now().UTC()
	end := start.AddDate(0, 1, 0)
	bc := newBillingCycle(t, "monthly")

	sub, err := NewSubscription(1, 1, start, end, true, bc)

	require.NoError(t, err)
	require.NotNil(t, sub)

	assert.NotEmpty(t, sub.SID(), "SID should be generated")
	assert.NotEmpty(t, sub.UUID(), "UUID should be generated")
	assert.NotEmpty(t, sub.LinkToken(), "link token should be generated")
	assert.Equal(t, uint(1), sub.SubjectID())
	assert.Equal(t, uint(1), sub.PlanID())
	assert.Equal(t, "user", sub.SubjectType(), "default subject type should be 'user'")
	assert.Equal(t, vo.StatusInactive, sub.Status(), "initial status should be inactive")
	assert.True(t, sub.AutoRenew())
	assert.Equal(t, start, sub.StartDate())
	assert.Equal(t, end, sub.EndDate())
	assert.Equal(t, start, sub.CurrentPeriodStart())
	assert.Equal(t, end, sub.CurrentPeriodEnd())
	assert.Equal(t, 1, sub.Version())
	assert.NotNil(t, sub.Metadata())
	assert.Nil(t, sub.CancelledAt())
	assert.Nil(t, sub.CancelReason())
}

func TestNewSubscription_WithNilBillingCycle(t *testing.T) {
	start := time.Now().UTC()
	end := start.AddDate(0, 1, 0)

	sub, err := NewSubscription(1, 1, start, end, false, nil)

	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Nil(t, sub.BillingCycle())
}

func TestNewSubscription_ZeroUserID(t *testing.T) {
	start := time.Now().UTC()
	end := start.AddDate(0, 1, 0)

	sub, err := NewSubscription(0, 1, start, end, true, nil)

	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Contains(t, err.Error(), "subject ID is required")
}

func TestNewSubscription_ZeroPlanID(t *testing.T) {
	start := time.Now().UTC()
	end := start.AddDate(0, 1, 0)

	sub, err := NewSubscription(1, 0, start, end, true, nil)

	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Contains(t, err.Error(), "plan ID is required")
}

func TestNewSubscription_EndDateBeforeStartDate(t *testing.T) {
	start := time.Now().UTC()
	end := start.AddDate(0, -1, 0) // before start

	sub, err := NewSubscription(1, 1, start, end, true, nil)

	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Contains(t, err.Error(), "end date must be after start date")
}

func TestNewSubscriptionWithSubject_EmptySubjectType(t *testing.T) {
	start := time.Now().UTC()
	end := start.AddDate(0, 1, 0)

	sub, err := NewSubscriptionWithSubject("", 1, 1, start, end, true, nil)

	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Contains(t, err.Error(), "subject type is required")
}

func TestNewSubscriptionWithSubject_CustomSubjectType(t *testing.T) {
	start := time.Now().UTC()
	end := start.AddDate(0, 1, 0)

	sub, err := NewSubscriptionWithSubject("user_group", 5, 1, start, end, false, nil)

	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, "user_group", sub.SubjectType())
	assert.Equal(t, uint(5), sub.SubjectID())
}

// =====================================================================
// TestReconstructSubscriptionWithParams
// =====================================================================

func TestReconstructSubscriptionWithParams_Valid(t *testing.T) {
	now := time.Now().UTC()
	bc := newBillingCycle(t, "yearly")

	sub, err := ReconstructSubscriptionWithParams(SubscriptionReconstructParams{
		ID:                 42,
		UserID:             10,
		PlanID:             100,
		SubjectType:        "user",
		SubjectID:          10,
		SID:                "sub_abc123",
		UUID:               "550e8400-e29b-41d4-a716-446655440000",
		LinkToken:          "dGVzdHRva2Vu",
		Status:             vo.StatusActive,
		StartDate:          now,
		EndDate:            now.AddDate(1, 0, 0),
		AutoRenew:          true,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		Version:            3,
		CreatedAt:          now,
		UpdatedAt:          now,
		BillingCycle:       bc,
	})

	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, uint(42), sub.ID())
	assert.Equal(t, "sub_abc123", sub.SID())
	assert.Equal(t, vo.StatusActive, sub.Status())
	assert.Equal(t, 3, sub.Version())
	assert.NotNil(t, sub.BillingCycle())
}

func TestReconstructSubscriptionWithParams_Errors(t *testing.T) {
	base := SubscriptionReconstructParams{
		ID:          1,
		UserID:      10,
		PlanID:      100,
		SubjectType: "user",
		SubjectID:   10,
		SID:         "sub_test",
		UUID:        "uuid-test",
		LinkToken:   "token-test",
		Status:      vo.StatusActive,
		StartDate:   time.Now().UTC(),
		EndDate:     time.Now().UTC().AddDate(0, 1, 0),
		Version:     1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	tests := []struct {
		name    string
		modify  func(p *SubscriptionReconstructParams)
		errMsg  string
	}{
		{
			name:   "zero ID",
			modify: func(p *SubscriptionReconstructParams) { p.ID = 0 },
			errMsg: "subscription ID cannot be zero",
		},
		{
			name:   "empty SID",
			modify: func(p *SubscriptionReconstructParams) { p.SID = "" },
			errMsg: "subscription SID is required",
		},
		{
			name:   "empty UUID",
			modify: func(p *SubscriptionReconstructParams) { p.UUID = "" },
			errMsg: "subscription UUID is required",
		},
		{
			name:   "empty LinkToken",
			modify: func(p *SubscriptionReconstructParams) { p.LinkToken = "" },
			errMsg: "subscription link token is required",
		},
		{
			name:   "empty SubjectType",
			modify: func(p *SubscriptionReconstructParams) { p.SubjectType = "" },
			errMsg: "subject type is required",
		},
		{
			name:   "zero SubjectID",
			modify: func(p *SubscriptionReconstructParams) { p.SubjectID = 0 },
			errMsg: "subject ID is required",
		},
		{
			name:   "zero PlanID",
			modify: func(p *SubscriptionReconstructParams) { p.PlanID = 0 },
			errMsg: "plan ID is required",
		},
		{
			name:   "invalid status",
			modify: func(p *SubscriptionReconstructParams) { p.Status = "invalid_status" },
			errMsg: "invalid subscription status",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := base
			tc.modify(&params)
			sub, err := ReconstructSubscriptionWithParams(params)
			assert.Error(t, err)
			assert.Nil(t, sub)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func TestReconstructSubscriptionWithParams_NilMetadataInitialized(t *testing.T) {
	now := time.Now().UTC()
	sub, err := ReconstructSubscriptionWithParams(SubscriptionReconstructParams{
		ID:          1,
		UserID:      10,
		PlanID:      100,
		SubjectType: "user",
		SubjectID:   10,
		SID:         "sub_test",
		UUID:        "uuid-test",
		LinkToken:   "token-test",
		Status:      vo.StatusActive,
		StartDate:   now,
		EndDate:     now.AddDate(0, 1, 0),
		Metadata:    nil, // explicitly nil
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	require.NoError(t, err)
	assert.NotNil(t, sub.Metadata(), "nil metadata should be initialized to empty map")
}

// =====================================================================
// TestSubscription_SetID
// =====================================================================

func TestSubscription_SetID_Success(t *testing.T) {
	sub := newValidSubscription(t)
	err := sub.SetID(42)
	require.NoError(t, err)
	assert.Equal(t, uint(42), sub.ID())
}

func TestSubscription_SetID_AlreadySet(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusInactive, now, now.AddDate(0, 1, 0))
	// sub already has ID = 1
	err := sub.SetID(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already set")
}

func TestSubscription_SetID_Zero(t *testing.T) {
	sub := newValidSubscription(t)
	err := sub.SetID(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be zero")
}

// =====================================================================
// TestSubscription_Activate
// =====================================================================

func TestSubscription_Activate_FromInactive(t *testing.T) {
	sub := newValidSubscription(t)
	assert.Equal(t, vo.StatusInactive, sub.Status())
	initialVersion := sub.Version()

	err := sub.Activate()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, sub.Status())
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_Activate_FromPendingPayment(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusPendingPayment, now, now.AddDate(0, 1, 0))

	err := sub.Activate()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, sub.Status())
}

func TestSubscription_Activate_IdempotentWhenAlreadyActive(t *testing.T) {
	sub := newActiveSubscription(t)
	versionBefore := sub.Version()

	err := sub.Activate()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, sub.Status())
	assert.Equal(t, versionBefore, sub.Version(), "version should not change on idempotent call")
}

func TestSubscription_Activate_FromCancelled(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Cancel("test reason"))

	err := sub.Activate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot activate")
	assert.Equal(t, vo.StatusCancelled, sub.Status())
}

func TestSubscription_Activate_FromExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusExpired, now.AddDate(-1, 0, 0), now.AddDate(0, -1, 0))

	err := sub.Activate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot activate")
}

func TestSubscription_Activate_FromSuspended(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusSuspended, now, now.AddDate(0, 1, 0))

	// Suspended -> Active is not allowed via Activate(), must use Unsuspend()
	err := sub.Activate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot activate")
}

// =====================================================================
// TestSubscription_Cancel
// =====================================================================

func TestSubscription_Cancel_FromActive(t *testing.T) {
	sub := newActiveSubscription(t)
	initialVersion := sub.Version()

	err := sub.Cancel("user requested cancellation")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusCancelled, sub.Status())
	assert.NotNil(t, sub.CancelledAt())
	require.NotNil(t, sub.CancelReason())
	assert.Equal(t, "user requested cancellation", *sub.CancelReason())
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_Cancel_FromInactive(t *testing.T) {
	sub := newValidSubscription(t) // inactive

	err := sub.Cancel("no longer needed")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusCancelled, sub.Status())
}

func TestSubscription_Cancel_FromPendingPayment(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusPendingPayment, now, now.AddDate(0, 1, 0))

	err := sub.Cancel("payment failed")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusCancelled, sub.Status())
}

func TestSubscription_Cancel_IdempotentWhenAlreadyCancelled(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Cancel("first cancel"))
	versionBefore := sub.Version()

	err := sub.Cancel("second cancel")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusCancelled, sub.Status())
	assert.Equal(t, versionBefore, sub.Version(), "version should not change on idempotent cancel")
}

func TestSubscription_Cancel_EmptyReason(t *testing.T) {
	sub := newActiveSubscription(t)

	err := sub.Cancel("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancel reason is required")
	assert.Equal(t, vo.StatusActive, sub.Status(), "status should not change on error")
}

func TestSubscription_Cancel_FromExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusExpired, now.AddDate(-1, 0, 0), now.AddDate(0, -1, 0))

	err := sub.Cancel("cleanup")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel")
}

// =====================================================================
// TestSubscription_Suspend
// =====================================================================

func TestSubscription_Suspend_FromActive(t *testing.T) {
	sub := newActiveSubscription(t)
	initialVersion := sub.Version()

	err := sub.Suspend("traffic limit exceeded")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusSuspended, sub.Status())
	require.NotNil(t, sub.CancelReason()) // reuses cancelReason field
	assert.Equal(t, "traffic limit exceeded", *sub.CancelReason())
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_Suspend_IdempotentWhenAlreadySuspended(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Suspend("first suspend"))
	versionBefore := sub.Version()

	err := sub.Suspend("second suspend")

	require.NoError(t, err)
	assert.Equal(t, vo.StatusSuspended, sub.Status())
	assert.Equal(t, versionBefore, sub.Version())
}

func TestSubscription_Suspend_EmptyReason(t *testing.T) {
	sub := newActiveSubscription(t)

	err := sub.Suspend("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "suspend reason is required")
	assert.Equal(t, vo.StatusActive, sub.Status())
}

func TestSubscription_Suspend_FromInactive(t *testing.T) {
	sub := newValidSubscription(t) // inactive

	err := sub.Suspend("admin action")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot suspend")
}

func TestSubscription_Suspend_FromCancelled(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Cancel("test"))

	err := sub.Suspend("admin action")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot suspend")
}

// =====================================================================
// TestSubscription_Unsuspend
// =====================================================================

func TestSubscription_Unsuspend_FromSuspended(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Suspend("traffic limit"))
	initialVersion := sub.Version()

	err := sub.Unsuspend()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, sub.Status())
	assert.Nil(t, sub.CancelReason(), "cancel reason should be cleared")
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_Unsuspend_FromNonSuspended(t *testing.T) {
	sub := newActiveSubscription(t)

	err := sub.Unsuspend()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot unsuspend")
}

func TestSubscription_Unsuspend_FromInactive(t *testing.T) {
	sub := newValidSubscription(t)

	err := sub.Unsuspend()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot unsuspend")
}

// =====================================================================
// TestSubscription_Renew
// =====================================================================

func TestSubscription_Renew_FromActive(t *testing.T) {
	sub := newActiveSubscription(t)
	originalEnd := sub.EndDate()
	newEnd := originalEnd.AddDate(0, 1, 0)
	initialVersion := sub.Version()

	err := sub.Renew(newEnd)

	require.NoError(t, err)
	assert.Equal(t, newEnd, sub.EndDate())
	assert.Equal(t, originalEnd, sub.CurrentPeriodStart(), "current period start should be old period end")
	assert.Equal(t, newEnd, sub.CurrentPeriodEnd())
	assert.Equal(t, vo.StatusActive, sub.Status())
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_Renew_FromExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusExpired, now.AddDate(-1, 0, 0), now.AddDate(0, -1, 0))
	newEnd := now.AddDate(0, 1, 0)

	err := sub.Renew(newEnd)

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, sub.Status(), "expired subscription should become active after renewal")
	assert.Equal(t, newEnd, sub.EndDate())
}

func TestSubscription_Renew_FromPastDue(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusPastDue, now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))
	newEnd := now.AddDate(0, 2, 0)

	err := sub.Renew(newEnd)

	require.NoError(t, err)
	assert.Equal(t, newEnd, sub.EndDate())
}

func TestSubscription_Renew_EndDateBeforeCurrent(t *testing.T) {
	sub := newActiveSubscription(t)
	oldEnd := sub.EndDate()
	newEnd := oldEnd.Add(-24 * time.Hour)

	err := sub.Renew(newEnd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "new end date must be after current end date")
}

func TestSubscription_Renew_FromCancelled(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Cancel("done"))
	newEnd := time.Now().UTC().AddDate(1, 0, 0)

	err := sub.Renew(newEnd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot renew")
}

func TestSubscription_Renew_FromInactive(t *testing.T) {
	sub := newValidSubscription(t)
	newEnd := time.Now().UTC().AddDate(1, 0, 0)

	err := sub.Renew(newEnd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot renew")
}

func TestSubscription_Renew_FromSuspended(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Suspend("traffic limit"))
	newEnd := time.Now().UTC().AddDate(1, 0, 0)

	err := sub.Renew(newEnd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot renew")
}

// =====================================================================
// TestSubscription_ChangePlan
// =====================================================================

func TestSubscription_ChangePlan_Success(t *testing.T) {
	sub := newActiveSubscription(t)
	initialVersion := sub.Version()

	err := sub.ChangePlan(99)

	require.NoError(t, err)
	assert.Equal(t, uint(99), sub.PlanID())
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_ChangePlan_SamePlanIdempotent(t *testing.T) {
	sub := newActiveSubscription(t)
	currentPlan := sub.PlanID()
	versionBefore := sub.Version()

	err := sub.ChangePlan(currentPlan)

	require.NoError(t, err)
	assert.Equal(t, versionBefore, sub.Version(), "version should not change for same plan")
}

func TestSubscription_ChangePlan_ZeroPlanID(t *testing.T) {
	sub := newActiveSubscription(t)

	err := sub.ChangePlan(0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "new plan ID is required")
}

func TestSubscription_ChangePlan_NotActive(t *testing.T) {
	sub := newValidSubscription(t) // inactive

	err := sub.ChangePlan(99)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot change plan")
}

func TestSubscription_ChangePlan_FromSuspended(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Suspend("admin"))

	err := sub.ChangePlan(99)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot change plan")
}

// =====================================================================
// TestSubscription_IsExpired
// =====================================================================

func TestSubscription_IsExpired_NotExpired(t *testing.T) {
	sub := newActiveSubscription(t)
	// end date is 1 month in the future
	assert.False(t, sub.IsExpired())
}

func TestSubscription_IsExpired_PastEndDate(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusActive, now.AddDate(-1, 0, 0), now.Add(-time.Hour))

	assert.True(t, sub.IsExpired())
}

// =====================================================================
// TestSubscription_IsActive
// =====================================================================

func TestSubscription_IsActive_ActiveAndNotExpired(t *testing.T) {
	sub := newActiveSubscription(t)
	assert.True(t, sub.IsActive())
}

func TestSubscription_IsActive_ActiveButExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusActive, now.AddDate(-1, 0, 0), now.Add(-time.Hour))

	assert.False(t, sub.IsActive(), "should not be active if end date has passed")
}

func TestSubscription_IsActive_InactiveStatus(t *testing.T) {
	sub := newValidSubscription(t) // inactive
	assert.False(t, sub.IsActive())
}

func TestSubscription_IsActive_CancelledStatus(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Cancel("done"))
	assert.False(t, sub.IsActive())
}

func TestSubscription_IsActive_SuspendedStatus(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Suspend("admin"))
	assert.False(t, sub.IsActive())
}

func TestSubscription_IsActive_TrialingAndNotExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusTrialing, now, now.AddDate(0, 1, 0))
	assert.True(t, sub.IsActive(), "trialing subscription should be considered active")
}

// =====================================================================
// TestSubscription_EffectiveStatus
// =====================================================================

func TestSubscription_EffectiveStatus_ActiveNotExpired(t *testing.T) {
	sub := newActiveSubscription(t)
	assert.Equal(t, vo.StatusActive, sub.EffectiveStatus())
}

func TestSubscription_EffectiveStatus_ActiveButExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusActive, now.AddDate(-1, 0, 0), now.Add(-time.Hour))

	assert.Equal(t, vo.StatusExpired, sub.EffectiveStatus(),
		"should return expired when active but past end date")
}

func TestSubscription_EffectiveStatus_TrialingButExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusTrialing, now.AddDate(-1, 0, 0), now.Add(-time.Hour))

	assert.Equal(t, vo.StatusExpired, sub.EffectiveStatus())
}

func TestSubscription_EffectiveStatus_PastDueButExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusPastDue, now.AddDate(-1, 0, 0), now.Add(-time.Hour))

	assert.Equal(t, vo.StatusExpired, sub.EffectiveStatus())
}

func TestSubscription_EffectiveStatus_CancelledAndExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusCancelled, now.AddDate(-1, 0, 0), now.Add(-time.Hour))

	// Cancelled cannot transition to expired, so EffectiveStatus returns cancelled
	assert.Equal(t, vo.StatusCancelled, sub.EffectiveStatus())
}

func TestSubscription_EffectiveStatus_AlreadyExpiredStatus(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusExpired, now.AddDate(-1, 0, 0), now.Add(-time.Hour))

	assert.Equal(t, vo.StatusExpired, sub.EffectiveStatus())
}

// =====================================================================
// TestSubscription_MarkAsExpired
// =====================================================================

func TestSubscription_MarkAsExpired_FromActive(t *testing.T) {
	sub := newActiveSubscription(t)
	initialVersion := sub.Version()

	err := sub.MarkAsExpired()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusExpired, sub.Status())
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_MarkAsExpired_FromTrialing(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusTrialing, now, now.AddDate(0, 1, 0))

	err := sub.MarkAsExpired()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusExpired, sub.Status())
}

func TestSubscription_MarkAsExpired_FromPastDue(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusPastDue, now, now.AddDate(0, 1, 0))

	err := sub.MarkAsExpired()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusExpired, sub.Status())
}

func TestSubscription_MarkAsExpired_FromPendingPayment(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusPendingPayment, now, now.AddDate(0, 1, 0))

	err := sub.MarkAsExpired()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusExpired, sub.Status())
}

func TestSubscription_MarkAsExpired_IdempotentWhenAlreadyExpired(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusExpired, now.AddDate(-1, 0, 0), now.Add(-time.Hour))
	versionBefore := sub.Version()

	err := sub.MarkAsExpired()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusExpired, sub.Status())
	assert.Equal(t, versionBefore, sub.Version())
}

func TestSubscription_MarkAsExpired_FromCancelled(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Cancel("test"))

	err := sub.MarkAsExpired()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot mark subscription as expired")
}

// =====================================================================
// TestSubscription_ResetUsage
// =====================================================================

func TestSubscription_ResetUsage_FromActive(t *testing.T) {
	sub := newActiveSubscription(t)
	originalPeriodEnd := sub.CurrentPeriodEnd()
	initialVersion := sub.Version()

	err := sub.ResetUsage()

	require.NoError(t, err)
	assert.Equal(t, originalPeriodEnd, sub.CurrentPeriodEnd(), "period end should not change")
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_ResetUsage_FromSuspended(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Suspend("traffic limit"))
	assert.Equal(t, vo.StatusSuspended, sub.Status())

	err := sub.ResetUsage()

	require.NoError(t, err)
	assert.Equal(t, vo.StatusActive, sub.Status(), "should unsuspend before resetting usage")
}

func TestSubscription_ResetUsage_FromInactive(t *testing.T) {
	sub := newValidSubscription(t)

	err := sub.ResetUsage()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reset usage")
}

func TestSubscription_ResetUsage_FromCancelled(t *testing.T) {
	sub := newActiveSubscription(t)
	require.NoError(t, sub.Cancel("test"))

	err := sub.ResetUsage()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reset usage")
}

// =====================================================================
// TestSubscription_Metadata
// =====================================================================

func TestSubscription_SetMetadata(t *testing.T) {
	sub := newValidSubscription(t)

	sub.SetMetadata("key1", "value1")
	sub.SetMetadata("key2", 42)

	md := sub.Metadata()
	assert.Equal(t, "value1", md["key1"])
	assert.Equal(t, 42, md["key2"])
}

func TestSubscription_DeleteMetadata(t *testing.T) {
	sub := newValidSubscription(t)
	sub.SetMetadata("key1", "value1")

	sub.DeleteMetadata("key1")

	_, exists := sub.Metadata()["key1"]
	assert.False(t, exists)
}

func TestSubscription_DeleteMetadata_NilMap(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusInactive, now, now.AddDate(0, 1, 0))
	// metadata is initialized by reconstructSubscription helper, so just test no panic
	sub.DeleteMetadata("nonexistent")
}

// =====================================================================
// TestSubscription_SetAutoRenew
// =====================================================================

func TestSubscription_SetAutoRenew_Toggle(t *testing.T) {
	sub := newValidSubscription(t) // autoRenew=true
	initialVersion := sub.Version()

	sub.SetAutoRenew(false)

	assert.False(t, sub.AutoRenew())
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_SetAutoRenew_NoChangeIdempotent(t *testing.T) {
	sub := newValidSubscription(t) // autoRenew=true
	versionBefore := sub.Version()

	sub.SetAutoRenew(true)

	assert.True(t, sub.AutoRenew())
	assert.Equal(t, versionBefore, sub.Version(), "version should not change when value unchanged")
}

// =====================================================================
// TestSubscription_ResetLinkToken
// =====================================================================

func TestSubscription_ResetLinkToken(t *testing.T) {
	sub := newValidSubscription(t)
	originalToken := sub.LinkToken()
	initialVersion := sub.Version()

	err := sub.ResetLinkToken()

	require.NoError(t, err)
	assert.NotEqual(t, originalToken, sub.LinkToken(), "token should be regenerated")
	assert.NotEmpty(t, sub.LinkToken())
	assert.Equal(t, initialVersion+1, sub.Version())
}

// =====================================================================
// TestSubscription_ResetUUID
// =====================================================================

func TestSubscription_ResetUUID(t *testing.T) {
	sub := newValidSubscription(t)
	originalUUID := sub.UUID()
	initialVersion := sub.Version()

	sub.ResetUUID()

	assert.NotEqual(t, originalUUID, sub.UUID(), "UUID should be regenerated")
	assert.NotEmpty(t, sub.UUID())
	assert.Equal(t, initialVersion+1, sub.Version())
}

// =====================================================================
// TestSubscription_UpdateCurrentPeriod
// =====================================================================

func TestSubscription_UpdateCurrentPeriod_Valid(t *testing.T) {
	sub := newValidSubscription(t)
	newStart := time.Now().UTC()
	newEnd := newStart.AddDate(0, 1, 0)
	initialVersion := sub.Version()

	err := sub.UpdateCurrentPeriod(newStart, newEnd)

	require.NoError(t, err)
	assert.Equal(t, newStart, sub.CurrentPeriodStart())
	assert.Equal(t, newEnd, sub.CurrentPeriodEnd())
	assert.Equal(t, initialVersion+1, sub.Version())
}

func TestSubscription_UpdateCurrentPeriod_EndBeforeStart(t *testing.T) {
	sub := newValidSubscription(t)
	newStart := time.Now().UTC()
	newEnd := newStart.Add(-time.Hour)

	err := sub.UpdateCurrentPeriod(newStart, newEnd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "period end must be after period start")
}

// =====================================================================
// TestSubscription_Validate
// =====================================================================

func TestSubscription_Validate_ValidSubscription(t *testing.T) {
	sub := newValidSubscription(t)
	assert.NoError(t, sub.Validate())
}

func TestSubscription_Validate_ActiveSubscription(t *testing.T) {
	sub := newActiveSubscription(t)
	assert.NoError(t, sub.Validate())
}

// =====================================================================
// TestSubscription_UserID_Deprecated
// =====================================================================

func TestSubscription_UserID_BackwardCompatibility(t *testing.T) {
	sub := newValidSubscription(t)
	// UserID should equal SubjectID for backward compatibility
	assert.Equal(t, sub.SubjectID(), sub.UserID())
}

// =====================================================================
// TestSubscription_StateTransition_FullLifecycle
// =====================================================================

func TestSubscription_FullLifecycle_CreateActivateCancelExpire(t *testing.T) {
	// Create
	sub := newValidSubscription(t)
	assert.Equal(t, vo.StatusInactive, sub.Status())

	// Activate
	require.NoError(t, sub.Activate())
	assert.Equal(t, vo.StatusActive, sub.Status())
	assert.True(t, sub.IsActive())

	// Cancel
	require.NoError(t, sub.Cancel("user cancelled"))
	assert.Equal(t, vo.StatusCancelled, sub.Status())
	assert.False(t, sub.IsActive())

	// Cannot transition from cancelled
	assert.Error(t, sub.Activate())
	assert.Error(t, sub.MarkAsExpired())
	assert.Error(t, sub.Suspend("test"))
}

func TestSubscription_FullLifecycle_ActivateSuspendUnsuspend(t *testing.T) {
	sub := newValidSubscription(t)
	require.NoError(t, sub.Activate())

	// Suspend
	require.NoError(t, sub.Suspend("traffic limit"))
	assert.Equal(t, vo.StatusSuspended, sub.Status())

	// Unsuspend
	require.NoError(t, sub.Unsuspend())
	assert.Equal(t, vo.StatusActive, sub.Status())
	assert.Nil(t, sub.CancelReason())
}

func TestSubscription_FullLifecycle_ActivateExpireRenew(t *testing.T) {
	now := time.Now().UTC()
	sub := reconstructSubscription(t, vo.StatusActive, now.AddDate(-1, 0, 0), now.Add(-time.Hour))

	// Mark as expired
	require.NoError(t, sub.MarkAsExpired())
	assert.Equal(t, vo.StatusExpired, sub.Status())

	// Renew
	newEnd := now.AddDate(0, 1, 0)
	require.NoError(t, sub.Renew(newEnd))
	assert.Equal(t, vo.StatusActive, sub.Status())
	assert.Equal(t, newEnd, sub.EndDate())
}

// =====================================================================
// TestSubscription_VersionIncrement
// =====================================================================

func TestSubscription_VersionIncrement_MultipleOperations(t *testing.T) {
	sub := newValidSubscription(t)
	assert.Equal(t, 1, sub.Version())

	require.NoError(t, sub.Activate())
	assert.Equal(t, 2, sub.Version())

	sub.SetAutoRenew(false)
	assert.Equal(t, 3, sub.Version())

	require.NoError(t, sub.Suspend("test"))
	assert.Equal(t, 4, sub.Version())

	require.NoError(t, sub.Unsuspend())
	assert.Equal(t, 5, sub.Version())

	require.NoError(t, sub.Cancel("done"))
	assert.Equal(t, 6, sub.Version())
}
