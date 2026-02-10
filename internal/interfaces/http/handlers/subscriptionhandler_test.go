package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/testutil"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
)

// =====================================================================
// Mock use cases
// =====================================================================

type mockCreateSubscriptionUC struct {
	result *usecases.CreateSubscriptionResult
	err    error
}

func (m *mockCreateSubscriptionUC) Execute(ctx context.Context, cmd usecases.CreateSubscriptionCommand) (*usecases.CreateSubscriptionResult, error) {
	return m.result, m.err
}

type mockGetSubscriptionUC struct {
	result *subdto.SubscriptionDTO
	err    error
}

func (m *mockGetSubscriptionUC) Execute(ctx context.Context, query usecases.GetSubscriptionQuery) (*subdto.SubscriptionDTO, error) {
	return m.result, m.err
}

func (m *mockGetSubscriptionUC) ExecuteBySID(ctx context.Context, sid string) (*subdto.SubscriptionDTO, error) {
	return m.result, m.err
}

type mockListUserSubscriptionsUC struct {
	result *usecases.ListUserSubscriptionsResult
	err    error
}

func (m *mockListUserSubscriptionsUC) Execute(ctx context.Context, query usecases.ListUserSubscriptionsQuery) (*usecases.ListUserSubscriptionsResult, error) {
	return m.result, m.err
}

type mockCancelSubscriptionUC struct {
	err error
}

func (m *mockCancelSubscriptionUC) Execute(ctx context.Context, cmd usecases.CancelSubscriptionCommand) error {
	return m.err
}

type mockDeleteSubscriptionUC struct {
	err error
}

func (m *mockDeleteSubscriptionUC) Execute(ctx context.Context, subscriptionID uint) error {
	return m.err
}

type mockChangePlanUC struct {
	err error
}

func (m *mockChangePlanUC) Execute(ctx context.Context, cmd usecases.ChangePlanCommand) error {
	return m.err
}

type mockGetSubscriptionUsageStatsUC struct {
	result *usecases.GetSubscriptionUsageStatsResponse
	err    error
}

func (m *mockGetSubscriptionUsageStatsUC) Execute(ctx context.Context, query usecases.GetSubscriptionUsageStatsQuery) (*usecases.GetSubscriptionUsageStatsResponse, error) {
	return m.result, m.err
}

type mockResetSubscriptionLinkUC struct {
	result *subdto.SubscriptionDTO
	err    error
}

func (m *mockResetSubscriptionLinkUC) Execute(ctx context.Context, cmd usecases.ResetSubscriptionLinkCommand) (*subdto.SubscriptionDTO, error) {
	return m.result, m.err
}

// =====================================================================
// Test helpers
// =====================================================================

func createTestSubscription() *subscription.Subscription {
	now := time.Now().UTC()
	bc, _ := vo.NewBillingCycle("monthly")
	sub, _ := subscription.ReconstructSubscriptionWithParams(subscription.SubscriptionReconstructParams{
		ID:                 1,
		UserID:             10,
		PlanID:             100,
		SubjectType:        "user",
		SubjectID:          10,
		SID:                "sub_test123",
		UUID:               "00000000-0000-0000-0000-000000000001",
		LinkToken:          "test_link_token_base64_encoded_value_here",
		Status:             vo.StatusActive,
		StartDate:          now,
		EndDate:            now.AddDate(0, 1, 0),
		AutoRenew:          true,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		Version:            1,
		CreatedAt:          now,
		UpdatedAt:          now,
		BillingCycle:       bc,
	})
	return sub
}

func createTestSubscriptionDTO() *subdto.SubscriptionDTO {
	sub := createTestSubscription()
	return subdto.ToSubscriptionDTO(sub, nil, nil, "")
}

func newTestSubscriptionHandler(
	createUC createSubscriptionUseCase,
	getUC getSubscriptionUseCase,
	listUserUC listUserSubscriptionsUseCase,
	cancelUC cancelSubscriptionUseCase,
	deleteUC deleteSubscriptionUseCase,
	changePlanUC changePlanUseCase,
	getUsageStatsUC getSubscriptionUsageStatsUseCase,
	resetLinkUC resetSubscriptionLinkUseCase,
) *SubscriptionHandler {
	return NewSubscriptionHandler(
		createUC, getUC, listUserUC, cancelUC, deleteUC, changePlanUC,
		getUsageStatsUC, resetLinkUC, testutil.NewMockLogger(),
	)
}

// =====================================================================
// TestSubscriptionHandler_CreateSubscription
// =====================================================================

func TestSubscriptionHandler_CreateSubscription_Success(t *testing.T) {
	sub := createTestSubscription()
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	// Create a valid subscription token using NewSubscriptionToken
	token, _ := subscription.NewSubscriptionToken(
		sub.ID(),
		"test-token",
		"test_hashed_token_value",
		"st",
		vo.TokenScopeReadOnly,
		&expiresAt,
	)
	mockResult := &usecases.CreateSubscriptionResult{
		Subscription: sub,
		Token:        token,
		PlainToken:   "plain_token_value",
	}
	mockUC := &mockCreateSubscriptionUC{result: mockResult}
	handler := newTestSubscriptionHandler(mockUC, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateSubscriptionRequest{
		PlanID:       "plan_abc123def456",
		BillingCycle: "monthly",
		AutoRenew:    ptrBool(true),
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions", reqBody)
	c.Set("user_id", uint(10))

	handler.CreateSubscription(c)

	if w.Code != http.StatusCreated {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSubscriptionHandler_CreateSubscription_InvalidRequest(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := map[string]string{"plan_id": "plan_test"} // missing billing_cycle
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions", reqBody)
	c.Set("user_id", uint(10))

	handler.CreateSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_CreateSubscription_InvalidPlanIDFormat(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateSubscriptionRequest{
		PlanID:       "invalid_id",
		BillingCycle: "monthly",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions", reqBody)
	c.Set("user_id", uint(10))

	handler.CreateSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_CreateSubscription_UseCaseError(t *testing.T) {
	mockUC := &mockCreateSubscriptionUC{err: errors.NewValidationError("plan not found", "")}
	handler := newTestSubscriptionHandler(mockUC, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateSubscriptionRequest{
		PlanID:       "plan_nonexistent",
		BillingCycle: "monthly",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions", reqBody)
	c.Set("user_id", uint(10))

	handler.CreateSubscription(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_CreateSubscription_NotAuthenticated(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateSubscriptionRequest{
		PlanID:       "plan_abc123def456",
		BillingCycle: "monthly",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions", reqBody)
	// No user_id set

	handler.CreateSubscription(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestSubscriptionHandler_GetSubscription
// =====================================================================

func TestSubscriptionHandler_GetSubscription_Success(t *testing.T) {
	mockDTO := createTestSubscriptionDTO()
	mockUC := &mockGetSubscriptionUC{result: mockDTO}
	handler := newTestSubscriptionHandler(nil, mockUC, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/subscriptions/sub_test123", nil)
	c.Set("subscription_id", uint(1))

	handler.GetSubscription(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSubscriptionHandler_GetSubscription_NotFound(t *testing.T) {
	mockUC := &mockGetSubscriptionUC{err: errors.NewNotFoundError("subscription not found", "")}
	handler := newTestSubscriptionHandler(nil, mockUC, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/subscriptions/sub_nonexistent", nil)
	c.Set("subscription_id", uint(999))

	handler.GetSubscription(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_GetSubscription_NoSubscriptionID(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/subscriptions/sub_test123", nil)
	// No subscription_id set in context

	handler.GetSubscription(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestSubscriptionHandler_ListUserSubscriptions
// =====================================================================

func TestSubscriptionHandler_ListUserSubscriptions_Success(t *testing.T) {
	mockResult := &usecases.ListUserSubscriptionsResult{
		Subscriptions: []*subdto.SubscriptionDTO{createTestSubscriptionDTO()},
		Total:         1,
		Page:          1,
		PageSize:      10,
	}
	mockUC := &mockListUserSubscriptionsUC{result: mockResult}
	handler := newTestSubscriptionHandler(nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/subscriptions", nil)
	c.Set("user_id", uint(10))

	handler.ListUserSubscriptions(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSubscriptionHandler_ListUserSubscriptions_WithFilters(t *testing.T) {
	mockResult := &usecases.ListUserSubscriptionsResult{
		Subscriptions: []*subdto.SubscriptionDTO{},
		Total:         0,
		Page:          1,
		PageSize:      10,
	}
	mockUC := &mockListUserSubscriptionsUC{result: mockResult}
	handler := newTestSubscriptionHandler(nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/subscriptions?status=active&page=1&page_size=10", nil)
	c.Set("user_id", uint(10))
	testutil.SetQueryParams(c, map[string]string{
		"status":    "active",
		"page":      "1",
		"page_size": "10",
	})

	handler.ListUserSubscriptions(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSubscriptionHandler_ListUserSubscriptions_NotAuthenticated(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/subscriptions", nil)
	// No user_id set

	handler.ListUserSubscriptions(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestSubscriptionHandler_UpdateStatus (Cancel)
// =====================================================================

func TestSubscriptionHandler_UpdateStatus_CancelSuccess(t *testing.T) {
	mockUC := &mockCancelSubscriptionUC{err: nil}
	handler := newTestSubscriptionHandler(nil, nil, nil, mockUC, nil, nil, nil, nil)

	reason := "no longer needed"
	reqBody := UpdateStatusRequest{
		Status: "cancelled",
		Reason: &reason,
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/subscriptions/sub_test123/status", reqBody)
	c.Set("subscription_id", uint(1))

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSubscriptionHandler_UpdateStatus_MissingReason(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := UpdateStatusRequest{
		Status: "cancelled",
		// Reason not provided
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/subscriptions/sub_test123/status", reqBody)
	c.Set("subscription_id", uint(1))

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_UpdateStatus_InvalidStatus(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := UpdateStatusRequest{
		Status: "invalid",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/subscriptions/sub_test123/status", reqBody)
	c.Set("subscription_id", uint(1))

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_UpdateStatus_UseCaseError(t *testing.T) {
	mockUC := &mockCancelSubscriptionUC{err: errors.NewValidationError("cannot cancel expired subscription", "")}
	handler := newTestSubscriptionHandler(nil, nil, nil, mockUC, nil, nil, nil, nil)

	reason := "test"
	reqBody := UpdateStatusRequest{
		Status: "cancelled",
		Reason: &reason,
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/subscriptions/sub_test123/status", reqBody)
	c.Set("subscription_id", uint(1))

	handler.UpdateStatus(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestSubscriptionHandler_ChangePlan
// =====================================================================

func TestSubscriptionHandler_ChangePlan_Success(t *testing.T) {
	mockUC := &mockChangePlanUC{err: nil}
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, mockUC, nil, nil)

	// Generate a valid plan SID using the project's ID generator
	validPlanSID := id.MustNewSID(id.PrefixPlan)

	reqBody := ChangePlanRequest{
		NewPlanID:     validPlanSID,
		ChangeType:    "upgrade",
		EffectiveDate: "immediate",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions/sub_test123/change-plan", reqBody)
	c.Set("subscription_id", uint(1))

	handler.ChangePlan(c)

	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSubscriptionHandler_ChangePlan_InvalidPlanIDFormat(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := ChangePlanRequest{
		NewPlanID:     "invalid_id",
		ChangeType:    "upgrade",
		EffectiveDate: "immediate",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions/sub_test123/change-plan", reqBody)
	c.Set("subscription_id", uint(1))

	handler.ChangePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_ChangePlan_InvalidRequest(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := map[string]string{"new_plan_id": "plan_test"} // missing change_type, effective_date
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions/sub_test123/change-plan", reqBody)
	c.Set("subscription_id", uint(1))

	handler.ChangePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_ChangePlan_UseCaseError(t *testing.T) {
	mockUC := &mockChangePlanUC{err: errors.NewValidationError("new plan not found", "")}
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, mockUC, nil, nil)

	// Generate a valid plan SID for the error test case too
	validPlanSID := id.MustNewSID(id.PrefixPlan)

	reqBody := ChangePlanRequest{
		NewPlanID:     validPlanSID,
		ChangeType:    "upgrade",
		EffectiveDate: "immediate",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/subscriptions/sub_test123/change-plan", reqBody)
	c.Set("subscription_id", uint(1))

	handler.ChangePlan(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestSubscriptionHandler_DeleteSubscription
// =====================================================================

func TestSubscriptionHandler_DeleteSubscription_Success(t *testing.T) {
	mockUC := &mockDeleteSubscriptionUC{err: nil}
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, mockUC, nil, nil, nil)

	c, _ := testutil.NewTestContext(http.MethodDelete, "/subscriptions/sub_test123", nil)
	c.Set("subscription_id", uint(1))

	handler.DeleteSubscription(c)

	// gin's c.Status() sets the status on the writer; use Writer.Status() for reliable check.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
}

func TestSubscriptionHandler_DeleteSubscription_NotFound(t *testing.T) {
	mockUC := &mockDeleteSubscriptionUC{err: errors.NewNotFoundError("subscription not found", "")}
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, mockUC, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/subscriptions/sub_nonexistent", nil)
	c.Set("subscription_id", uint(999))

	handler.DeleteSubscription(c)

	assert.NotEqual(t, http.StatusNoContent, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestSubscriptionHandler_DeleteSubscription_NoSubscriptionID(t *testing.T) {
	handler := newTestSubscriptionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/subscriptions/sub_test123", nil)
	// No subscription_id set

	handler.DeleteSubscription(c)

	assert.NotEqual(t, http.StatusNoContent, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// Helper functions
// =====================================================================

func ptrBool(b bool) *bool {
	return &b
}
