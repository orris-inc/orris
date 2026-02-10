package subscription

import (
	"testing"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func newValidPlan(t *testing.T) *Plan {
	t.Helper()
	plan, err := NewPlan("Basic Plan", "basic", "A basic subscription plan", vo.PlanTypeNode)
	require.NoError(t, err)
	require.NotNil(t, plan)
	return plan
}

func planTestNow() time.Time {
	return time.Now().UTC()
}

// =====================================================================
// TestNewPlan_*
// =====================================================================

func TestNewPlan_ValidInput(t *testing.T) {
	plan, err := NewPlan("Premium Plan", "premium", "A premium plan", vo.PlanTypeForward)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.NotEmpty(t, plan.SID())
	assert.Equal(t, "Premium Plan", plan.Name())
	assert.Equal(t, "premium", plan.Slug())
	assert.Equal(t, "A premium plan", plan.Description())
	assert.Equal(t, PlanStatusActive, plan.Status())
	assert.Equal(t, vo.PlanTypeForward, plan.PlanType())
	assert.True(t, plan.IsPublic())
	assert.Equal(t, 0, plan.SortOrder())
	assert.NotNil(t, plan.Metadata())
	assert.Nil(t, plan.Features())
	assert.Nil(t, plan.NodeLimit())
	assert.Equal(t, 1, plan.Version())
	assert.True(t, plan.IsActive())
}

func TestNewPlan_AllPlanTypes(t *testing.T) {
	tests := []struct {
		name     string
		planType vo.PlanType
	}{
		{"node type", vo.PlanTypeNode},
		{"forward type", vo.PlanTypeForward},
		{"hybrid type", vo.PlanTypeHybrid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := NewPlan("Test", "test-"+tc.planType.String(), "desc", tc.planType)
			require.NoError(t, err)
			require.NotNil(t, plan)
			assert.Equal(t, tc.planType, plan.PlanType())
		})
	}
}

func TestNewPlan_EmptyName(t *testing.T) {
	plan, err := NewPlan("", "slug", "desc", vo.PlanTypeNode)

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "plan name is required")
}

func TestNewPlan_EmptySlug(t *testing.T) {
	plan, err := NewPlan("Plan", "", "desc", vo.PlanTypeNode)

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "plan slug is required")
}

func TestNewPlan_NameTooLong(t *testing.T) {
	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}

	plan, err := NewPlan(string(longName), "slug", "desc", vo.PlanTypeNode)

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "plan name too long")
}

func TestNewPlan_SlugTooLong(t *testing.T) {
	longSlug := make([]byte, 101)
	for i := range longSlug {
		longSlug[i] = 'a'
	}

	plan, err := NewPlan("Plan", string(longSlug), "desc", vo.PlanTypeNode)

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "plan slug too long")
}

func TestNewPlan_InvalidPlanType(t *testing.T) {
	plan, err := NewPlan("Plan", "slug", "desc", vo.PlanType("invalid"))

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "invalid plan type")
}

func TestNewPlan_EmptyDescription(t *testing.T) {
	// Empty description is allowed
	plan, err := NewPlan("Plan", "slug", "", vo.PlanTypeNode)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Empty(t, plan.Description())
}

// =====================================================================
// TestReconstructPlan
// =====================================================================

func TestReconstructPlan_Valid(t *testing.T) {
	now := planTestNow()
	nodeLimit := 10
	features := vo.NewPlanFeatures(map[string]interface{}{
		"traffic_limit": uint64(1073741824),
	})

	plan, err := ReconstructPlan(
		1, "plan_abc", "Pro", "pro", "Pro plan",
		"active", "node", features,
		&nodeLimit, true, 5,
		map[string]interface{}{"key": "value"},
		3,
		now, now,
	)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, uint(1), plan.ID())
	assert.Equal(t, "plan_abc", plan.SID())
	assert.Equal(t, "Pro", plan.Name())
	assert.Equal(t, "pro", plan.Slug())
	assert.Equal(t, PlanStatusActive, plan.Status())
	assert.Equal(t, vo.PlanTypeNode, plan.PlanType())
	assert.NotNil(t, plan.Features())
	require.NotNil(t, plan.NodeLimit())
	assert.Equal(t, 10, *plan.NodeLimit())
	assert.True(t, plan.IsPublic())
	assert.Equal(t, 5, plan.SortOrder())
	assert.Equal(t, 3, plan.Version())
}

func TestReconstructPlan_InactiveStatus(t *testing.T) {
	now := planTestNow()

	plan, err := ReconstructPlan(1, "plan_abc", "Plan", "slug", "desc",
		"inactive", "forward", nil, nil, false, 0, nil, 1, now, now)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, PlanStatusInactive, plan.Status())
	assert.False(t, plan.IsActive())
	assert.Equal(t, vo.PlanTypeForward, plan.PlanType())
}

func TestReconstructPlan_ZeroID(t *testing.T) {
	now := planTestNow()

	plan, err := ReconstructPlan(0, "plan_abc", "Plan", "slug", "desc",
		"active", "node", nil, nil, true, 0, nil, 1, now, now)

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "plan ID cannot be zero")
}

func TestReconstructPlan_EmptySID(t *testing.T) {
	now := planTestNow()

	plan, err := ReconstructPlan(1, "", "Plan", "slug", "desc",
		"active", "node", nil, nil, true, 0, nil, 1, now, now)

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "plan SID is required")
}

func TestReconstructPlan_InvalidStatus(t *testing.T) {
	now := planTestNow()

	plan, err := ReconstructPlan(1, "plan_abc", "Plan", "slug", "desc",
		"invalid", "node", nil, nil, true, 0, nil, 1, now, now)

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "invalid plan status")
}

func TestReconstructPlan_InvalidPlanType(t *testing.T) {
	now := planTestNow()

	plan, err := ReconstructPlan(1, "plan_abc", "Plan", "slug", "desc",
		"active", "invalid_type", nil, nil, true, 0, nil, 1, now, now)

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "invalid plan type")
}

func TestReconstructPlan_NilMetadataInitialized(t *testing.T) {
	now := planTestNow()

	plan, err := ReconstructPlan(1, "plan_abc", "Plan", "slug", "desc",
		"active", "node", nil, nil, true, 0, nil, 1, now, now)

	require.NoError(t, err)
	assert.NotNil(t, plan.Metadata(), "nil metadata should be initialized")
}

// =====================================================================
// TestPlan_SetID
// =====================================================================

func TestPlan_SetID_Success(t *testing.T) {
	plan := newValidPlan(t)
	err := plan.SetID(42)
	require.NoError(t, err)
	assert.Equal(t, uint(42), plan.ID())
}

func TestPlan_SetID_AlreadySet(t *testing.T) {
	now := planTestNow()
	plan, err := ReconstructPlan(1, "plan_abc", "Plan", "slug", "desc",
		"active", "node", nil, nil, true, 0, nil, 1, now, now)
	require.NoError(t, err)

	err = plan.SetID(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already set")
}

func TestPlan_SetID_Zero(t *testing.T) {
	plan := newValidPlan(t)
	err := plan.SetID(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be zero")
}

// =====================================================================
// TestPlan_Activate / Deactivate
// =====================================================================

func TestPlan_Activate_FromInactive(t *testing.T) {
	plan := newValidPlan(t)
	require.NoError(t, plan.Deactivate())
	assert.Equal(t, PlanStatusInactive, plan.Status())
	initialVersion := plan.Version()

	err := plan.Activate()

	require.NoError(t, err)
	assert.Equal(t, PlanStatusActive, plan.Status())
	assert.Equal(t, initialVersion+1, plan.Version())
}

func TestPlan_Activate_IdempotentWhenAlreadyActive(t *testing.T) {
	plan := newValidPlan(t)
	assert.Equal(t, PlanStatusActive, plan.Status())
	versionBefore := plan.Version()

	err := plan.Activate()

	require.NoError(t, err)
	assert.Equal(t, PlanStatusActive, plan.Status())
	assert.Equal(t, versionBefore, plan.Version(), "version should not change on idempotent call")
}

func TestPlan_Deactivate_FromActive(t *testing.T) {
	plan := newValidPlan(t)
	initialVersion := plan.Version()

	err := plan.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, PlanStatusInactive, plan.Status())
	assert.False(t, plan.IsActive())
	assert.Equal(t, initialVersion+1, plan.Version())
}

func TestPlan_Deactivate_IdempotentWhenAlreadyInactive(t *testing.T) {
	plan := newValidPlan(t)
	require.NoError(t, plan.Deactivate())
	versionBefore := plan.Version()

	err := plan.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, PlanStatusInactive, plan.Status())
	assert.Equal(t, versionBefore, plan.Version())
}

// =====================================================================
// TestPlan_UpdateDescription
// =====================================================================

func TestPlan_UpdateDescription(t *testing.T) {
	plan := newValidPlan(t)
	initialVersion := plan.Version()

	plan.UpdateDescription("new description")

	assert.Equal(t, "new description", plan.Description())
	assert.Equal(t, initialVersion+1, plan.Version())
}

func TestPlan_UpdateDescription_Empty(t *testing.T) {
	plan := newValidPlan(t)

	plan.UpdateDescription("")

	assert.Empty(t, plan.Description())
}

// =====================================================================
// TestPlan_UpdateFeatures
// =====================================================================

func TestPlan_UpdateFeatures_Valid(t *testing.T) {
	plan := newValidPlan(t)
	features := vo.NewPlanFeatures(map[string]interface{}{
		vo.LimitKeyTraffic:   uint64(1073741824),
		vo.LimitKeyRuleCount: 10,
	})
	initialVersion := plan.Version()

	err := plan.UpdateFeatures(features)

	require.NoError(t, err)
	assert.NotNil(t, plan.Features())
	assert.Equal(t, initialVersion+1, plan.Version())
}

func TestPlan_UpdateFeatures_Nil(t *testing.T) {
	plan := newValidPlan(t)

	err := plan.UpdateFeatures(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "features cannot be nil")
}

// =====================================================================
// TestPlan_NodeLimit
// =====================================================================

func TestPlan_NodeLimit_Default(t *testing.T) {
	plan := newValidPlan(t)

	assert.Nil(t, plan.NodeLimit())
	assert.False(t, plan.HasNodeLimit())
	assert.Equal(t, 0, plan.GetNodeLimit())
}

func TestPlan_SetNodeLimit(t *testing.T) {
	plan := newValidPlan(t)
	limit := 5
	initialVersion := plan.Version()

	plan.SetNodeLimit(&limit)

	require.NotNil(t, plan.NodeLimit())
	assert.Equal(t, 5, *plan.NodeLimit())
	assert.True(t, plan.HasNodeLimit())
	assert.Equal(t, 5, plan.GetNodeLimit())
	assert.Equal(t, initialVersion+1, plan.Version())
}

func TestPlan_SetNodeLimit_Zero(t *testing.T) {
	plan := newValidPlan(t)
	limit := 0

	plan.SetNodeLimit(&limit)

	require.NotNil(t, plan.NodeLimit())
	assert.Equal(t, 0, *plan.NodeLimit())
	assert.False(t, plan.HasNodeLimit(), "zero limit means unlimited")
	assert.Equal(t, 0, plan.GetNodeLimit())
}

func TestPlan_SetNodeLimit_Nil(t *testing.T) {
	plan := newValidPlan(t)
	limit := 5
	plan.SetNodeLimit(&limit)
	require.True(t, plan.HasNodeLimit())

	plan.SetNodeLimit(nil)

	assert.Nil(t, plan.NodeLimit())
	assert.False(t, plan.HasNodeLimit())
}

// =====================================================================
// TestPlan_SortOrder
// =====================================================================

func TestPlan_SetSortOrder(t *testing.T) {
	plan := newValidPlan(t)
	initialVersion := plan.Version()

	plan.SetSortOrder(10)

	assert.Equal(t, 10, plan.SortOrder())
	assert.Equal(t, initialVersion+1, plan.Version())
}

// =====================================================================
// TestPlan_SetPublic
// =====================================================================

func TestPlan_SetPublic(t *testing.T) {
	plan := newValidPlan(t)
	assert.True(t, plan.IsPublic())
	initialVersion := plan.Version()

	plan.SetPublic(false)

	assert.False(t, plan.IsPublic())
	assert.Equal(t, initialVersion+1, plan.Version())
}

// =====================================================================
// TestPlan_GetLimit
// =====================================================================

func TestPlan_GetLimit_NoFeatures(t *testing.T) {
	plan := newValidPlan(t)

	val, ok := plan.GetLimit("traffic_limit")

	assert.Nil(t, val)
	assert.False(t, ok)
}

func TestPlan_GetLimit_WithFeatures(t *testing.T) {
	plan := newValidPlan(t)
	features := vo.NewPlanFeatures(map[string]interface{}{
		vo.LimitKeyTraffic: uint64(1073741824),
	})
	require.NoError(t, plan.UpdateFeatures(features))

	val, ok := plan.GetLimit(vo.LimitKeyTraffic)

	assert.True(t, ok)
	assert.Equal(t, uint64(1073741824), val)
}

func TestPlan_GetLimit_KeyNotFound(t *testing.T) {
	plan := newValidPlan(t)
	features := vo.NewPlanFeatures(map[string]interface{}{})
	require.NoError(t, plan.UpdateFeatures(features))

	val, ok := plan.GetLimit("nonexistent")

	assert.Nil(t, val)
	assert.False(t, ok)
}

// =====================================================================
// TestPlan_TrafficLimit
// =====================================================================

func TestPlan_GetTrafficLimit_NoFeatures(t *testing.T) {
	plan := newValidPlan(t)

	limit, err := plan.GetTrafficLimit()

	require.NoError(t, err)
	assert.Equal(t, uint64(0), limit, "should return 0 (unlimited) when no features")
}

func TestPlan_GetTrafficLimit_WithFeatures(t *testing.T) {
	plan := newValidPlan(t)
	features := vo.NewPlanFeatures(map[string]interface{}{
		vo.LimitKeyTraffic: uint64(10737418240),
	})
	require.NoError(t, plan.UpdateFeatures(features))

	limit, err := plan.GetTrafficLimit()

	require.NoError(t, err)
	assert.Equal(t, uint64(10737418240), limit)
}

func TestPlan_IsUnlimitedTraffic_NoFeatures(t *testing.T) {
	plan := newValidPlan(t)
	assert.True(t, plan.IsUnlimitedTraffic())
}

func TestPlan_IsUnlimitedTraffic_WithZeroLimit(t *testing.T) {
	plan := newValidPlan(t)
	features := vo.NewPlanFeatures(map[string]interface{}{
		vo.LimitKeyTraffic: uint64(0),
	})
	require.NoError(t, plan.UpdateFeatures(features))

	assert.True(t, plan.IsUnlimitedTraffic())
}

func TestPlan_IsUnlimitedTraffic_WithLimit(t *testing.T) {
	plan := newValidPlan(t)
	features := vo.NewPlanFeatures(map[string]interface{}{
		vo.LimitKeyTraffic: uint64(1073741824),
	})
	require.NoError(t, plan.UpdateFeatures(features))

	assert.False(t, plan.IsUnlimitedTraffic())
}

func TestPlan_HasTrafficRemaining_NoFeatures(t *testing.T) {
	plan := newValidPlan(t)

	remaining, err := plan.HasTrafficRemaining(999999)

	require.NoError(t, err)
	assert.True(t, remaining, "should always have remaining when no features")
}

func TestPlan_HasTrafficRemaining_WithinLimit(t *testing.T) {
	plan := newValidPlan(t)
	features := vo.NewPlanFeatures(map[string]interface{}{
		vo.LimitKeyTraffic: uint64(1073741824), // 1 GB
	})
	require.NoError(t, plan.UpdateFeatures(features))

	remaining, err := plan.HasTrafficRemaining(536870912) // 512 MB

	require.NoError(t, err)
	assert.True(t, remaining)
}

func TestPlan_HasTrafficRemaining_ExceedsLimit(t *testing.T) {
	plan := newValidPlan(t)
	features := vo.NewPlanFeatures(map[string]interface{}{
		vo.LimitKeyTraffic: uint64(1073741824), // 1 GB
	})
	require.NoError(t, plan.UpdateFeatures(features))

	remaining, err := plan.HasTrafficRemaining(1073741824) // exactly at limit

	require.NoError(t, err)
	assert.False(t, remaining, "should not have remaining when at limit")
}

// =====================================================================
// TestPlan_IncrementVersion
// =====================================================================

func TestPlan_IncrementVersion(t *testing.T) {
	plan := newValidPlan(t)
	assert.Equal(t, 1, plan.Version())

	plan.IncrementVersion()

	assert.Equal(t, 2, plan.Version())
}

// =====================================================================
// TestPlan_VersionIncrement_MultipleOperations
// =====================================================================

func TestPlan_VersionIncrement_MultipleOperations(t *testing.T) {
	plan := newValidPlan(t)
	assert.Equal(t, 1, plan.Version())

	// Deactivate (+1)
	require.NoError(t, plan.Deactivate())
	assert.Equal(t, 2, plan.Version())

	// Activate (+1)
	require.NoError(t, plan.Activate())
	assert.Equal(t, 3, plan.Version())

	// UpdateDescription (+1)
	plan.UpdateDescription("updated")
	assert.Equal(t, 4, plan.Version())

	// SetPublic (+1)
	plan.SetPublic(false)
	assert.Equal(t, 5, plan.Version())

	// SetSortOrder (+1)
	plan.SetSortOrder(99)
	assert.Equal(t, 6, plan.Version())
}
