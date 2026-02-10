package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/testutil"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// =====================================================================
// Mock use cases
// =====================================================================

type mockCreatePlanUC struct {
	result *subdto.PlanDTO
	err    error
}

func (m *mockCreatePlanUC) Execute(ctx context.Context, cmd usecases.CreatePlanCommand) (*subdto.PlanDTO, error) {
	return m.result, m.err
}

type mockUpdatePlanUC struct {
	result *subdto.PlanDTO
	err    error
}

func (m *mockUpdatePlanUC) Execute(ctx context.Context, cmd usecases.UpdatePlanCommand) (*subdto.PlanDTO, error) {
	return m.result, m.err
}

type mockGetPlanUC struct {
	result *subdto.PlanDTO
	err    error
}

func (m *mockGetPlanUC) ExecuteBySID(ctx context.Context, sid string) (*subdto.PlanDTO, error) {
	return m.result, m.err
}

type mockListPlansUC struct {
	result *usecases.ListPlansResult
	err    error
}

func (m *mockListPlansUC) Execute(ctx context.Context, query usecases.ListPlansQuery) (*usecases.ListPlansResult, error) {
	return m.result, m.err
}

type mockGetPublicPlansUC struct {
	result []*subdto.PlanDTO
	err    error
}

func (m *mockGetPublicPlansUC) Execute(ctx context.Context) ([]*subdto.PlanDTO, error) {
	return m.result, m.err
}

type mockActivatePlanUC struct {
	err error
}

func (m *mockActivatePlanUC) Execute(ctx context.Context, planSID string) error {
	return m.err
}

type mockDeactivatePlanUC struct {
	err error
}

func (m *mockDeactivatePlanUC) Execute(ctx context.Context, planSID string) error {
	return m.err
}

type mockDeletePlanUC struct {
	err error
}

func (m *mockDeletePlanUC) Execute(ctx context.Context, planSID string) error {
	return m.err
}

type mockGetPlanPricingsUC struct {
	result []*subdto.PricingOptionDTO
	err    error
}

func (m *mockGetPlanPricingsUC) Execute(ctx context.Context, query usecases.GetPlanPricingsQuery) ([]*subdto.PricingOptionDTO, error) {
	return m.result, m.err
}

// =====================================================================
// Test helpers
// =====================================================================

func createTestPlan() *subscription.Plan {
	plan, _ := subscription.NewPlan("Basic Plan", "basic", "A basic subscription plan", vo.PlanTypeNode)
	_ = plan.SetID(1)
	return plan
}

func createTestPlanDTO() *subdto.PlanDTO {
	plan := createTestPlan()
	return subdto.ToPlanDTO(plan)
}

func newTestPlanHandler(
	createPlanUC createPlanUseCase,
	updatePlanUC updatePlanUseCase,
	getPlanUC getPlanUseCase,
	listPlansUC listPlansUseCase,
	getPublicPlansUC getPublicPlansUseCase,
	activatePlanUC activatePlanUseCase,
	deactivatePlanUC deactivatePlanUseCase,
	deletePlanUC deletePlanUseCase,
	getPlanPricingsUC getPlanPricingsUseCase,
) *PlanHandler {
	return NewPlanHandler(
		createPlanUC, updatePlanUC, getPlanUC, listPlansUC,
		getPublicPlansUC, activatePlanUC, deactivatePlanUC,
		deletePlanUC, getPlanPricingsUC, testutil.NewMockLogger(),
	)
}

// =====================================================================
// TestPlanHandler_CreatePlan
// =====================================================================

func TestPlanHandler_CreatePlan_Success(t *testing.T) {
	mockDTO := createTestPlanDTO()
	mockUC := &mockCreatePlanUC{result: mockDTO}
	handler := newTestPlanHandler(mockUC, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreatePlanRequest{
		Name:        "Basic Plan",
		Slug:        "basic",
		Description: "A basic plan",
		PlanType:    "node",
		IsPublic:    true,
		SortOrder:   1,
		Pricings: []subdto.PricingOptionInput{
			{
				BillingCycle: "monthly",
				Price:        999,
				Currency:     "USD",
			},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/plans", reqBody)

	handler.CreatePlan(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPlanHandler_CreatePlan_InvalidRequest(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := map[string]string{"name": "Test Plan"} // missing required fields
	c, w := testutil.NewTestContext(http.MethodPost, "/plans", reqBody)

	handler.CreatePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_CreatePlan_UseCaseError(t *testing.T) {
	mockUC := &mockCreatePlanUC{err: errors.NewValidationError("slug already exists", "")}
	handler := newTestPlanHandler(mockUC, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreatePlanRequest{
		Name:        "Duplicate Plan",
		Slug:        "duplicate",
		Description: "A duplicate plan",
		PlanType:    "node",
		Pricings: []subdto.PricingOptionInput{
			{
				BillingCycle: "monthly",
				Price:        999,
				Currency:     "USD",
			},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/plans", reqBody)

	handler.CreatePlan(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_CreatePlan_InvalidPlanType(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreatePlanRequest{
		Name:        "Test Plan",
		Slug:        "test",
		Description: "A test plan",
		PlanType:    "invalid",
		Pricings: []subdto.PricingOptionInput{
			{
				BillingCycle: "monthly",
				Price:        999,
				Currency:     "USD",
			},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/plans", reqBody)

	handler.CreatePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestPlanHandler_GetPlan
// =====================================================================

func TestPlanHandler_GetPlan_Success(t *testing.T) {
	mockDTO := createTestPlanDTO()
	mockUC := &mockGetPlanUC{result: mockDTO}
	handler := newTestPlanHandler(nil, nil, mockUC, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/plan_test123", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.GetPlan(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPlanHandler_GetPlan_InvalidSID(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/invalid_id", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "invalid_id"})

	handler.GetPlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_GetPlan_NotFound(t *testing.T) {
	mockUC := &mockGetPlanUC{err: errors.NewNotFoundError("plan not found", "")}
	handler := newTestPlanHandler(nil, nil, mockUC, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/plan_nonexistent", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_nonexistent"})

	handler.GetPlan(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestPlanHandler_UpdatePlan
// =====================================================================

func TestPlanHandler_UpdatePlan_Success(t *testing.T) {
	mockDTO := createTestPlanDTO()
	mockUC := &mockUpdatePlanUC{result: mockDTO}
	handler := newTestPlanHandler(nil, mockUC, nil, nil, nil, nil, nil, nil, nil)

	desc := "Updated description"
	reqBody := UpdatePlanRequest{
		Description: &desc,
	}
	c, w := testutil.NewTestContext(http.MethodPut, "/plans/plan_test123", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.UpdatePlan(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPlanHandler_UpdatePlan_InvalidSID(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	desc := "Updated description"
	reqBody := UpdatePlanRequest{
		Description: &desc,
	}
	c, w := testutil.NewTestContext(http.MethodPut, "/plans/invalid_id", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "invalid_id"})

	handler.UpdatePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_UpdatePlan_NotFound(t *testing.T) {
	mockUC := &mockUpdatePlanUC{err: errors.NewNotFoundError("plan not found", "")}
	handler := newTestPlanHandler(nil, mockUC, nil, nil, nil, nil, nil, nil, nil)

	desc := "Updated description"
	reqBody := UpdatePlanRequest{
		Description: &desc,
	}
	c, w := testutil.NewTestContext(http.MethodPut, "/plans/plan_nonexistent", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_nonexistent"})

	handler.UpdatePlan(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestPlanHandler_ListPlans
// =====================================================================

func TestPlanHandler_ListPlans_Success(t *testing.T) {
	mockResult := &usecases.ListPlansResult{
		Plans: []*subdto.PlanDTO{createTestPlanDTO()},
		Total: 1,
	}
	mockUC := &mockListPlansUC{result: mockResult}
	handler := newTestPlanHandler(nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans", nil)

	handler.ListPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPlanHandler_ListPlans_WithFilters(t *testing.T) {
	mockResult := &usecases.ListPlansResult{
		Plans: []*subdto.PlanDTO{},
		Total: 0,
	}
	mockUC := &mockListPlansUC{result: mockResult}
	handler := newTestPlanHandler(nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans?status=active&is_public=true&plan_type=node", nil)
	testutil.SetQueryParams(c, map[string]string{
		"status":    "active",
		"is_public": "true",
		"plan_type": "node",
	})

	handler.ListPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPlanHandler_ListPlans_InvalidIsPublicParam(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans?is_public=invalid", nil)
	testutil.SetQueryParams(c, map[string]string{
		"is_public": "invalid",
	})

	handler.ListPlans(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_ListPlans_UseCaseError(t *testing.T) {
	mockUC := &mockListPlansUC{err: errors.NewInternalError("database error")}
	handler := newTestPlanHandler(nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans", nil)

	handler.ListPlans(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestPlanHandler_GetPublicPlans
// =====================================================================

func TestPlanHandler_GetPublicPlans_Success(t *testing.T) {
	mockResult := []*subdto.PlanDTO{createTestPlanDTO()}
	mockUC := &mockGetPublicPlansUC{result: mockResult}
	handler := newTestPlanHandler(nil, nil, nil, nil, mockUC, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/public", nil)

	handler.GetPublicPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	var data []*subdto.PlanDTO
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Len(t, data, 1)
}

func TestPlanHandler_GetPublicPlans_Empty(t *testing.T) {
	mockResult := []*subdto.PlanDTO{}
	mockUC := &mockGetPublicPlansUC{result: mockResult}
	handler := newTestPlanHandler(nil, nil, nil, nil, mockUC, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/public", nil)

	handler.GetPublicPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	var data []*subdto.PlanDTO
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Len(t, data, 0)
}

func TestPlanHandler_GetPublicPlans_UseCaseError(t *testing.T) {
	mockUC := &mockGetPublicPlansUC{err: errors.NewInternalError("database error")}
	handler := newTestPlanHandler(nil, nil, nil, nil, mockUC, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/public", nil)

	handler.GetPublicPlans(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestPlanHandler_UpdatePlanStatus (Activate/Deactivate)
// =====================================================================

func TestPlanHandler_UpdatePlanStatus_Activate(t *testing.T) {
	mockUC := &mockActivatePlanUC{err: nil}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, mockUC, nil, nil, nil)

	reqBody := UpdatePlanStatusRequest{
		Status: "active",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/plans/plan_test123/status", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.UpdatePlanStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPlanHandler_UpdatePlanStatus_Deactivate(t *testing.T) {
	mockUC := &mockDeactivatePlanUC{err: nil}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, mockUC, nil, nil)

	reqBody := UpdatePlanStatusRequest{
		Status: "inactive",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/plans/plan_test123/status", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.UpdatePlanStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPlanHandler_UpdatePlanStatus_InvalidStatus(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := UpdatePlanStatusRequest{
		Status: "invalid",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/plans/plan_test123/status", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.UpdatePlanStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_UpdatePlanStatus_InvalidSID(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := UpdatePlanStatusRequest{
		Status: "active",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/plans/invalid_id/status", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "invalid_id"})

	handler.UpdatePlanStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_UpdatePlanStatus_UseCaseError(t *testing.T) {
	mockUC := &mockActivatePlanUC{err: errors.NewNotFoundError("plan not found", "")}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, mockUC, nil, nil, nil)

	reqBody := UpdatePlanStatusRequest{
		Status: "active",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/plans/plan_nonexistent/status", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_nonexistent"})

	handler.UpdatePlanStatus(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestPlanHandler_DeletePlan
// =====================================================================

func TestPlanHandler_DeletePlan_Success(t *testing.T) {
	mockUC := &mockDeletePlanUC{err: nil}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	c, _ := testutil.NewTestContext(http.MethodDelete, "/plans/plan_test123", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.DeletePlan(c)

	// gin's c.Status() sets the status on the writer; use Writer.Status() for reliable check.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
}

func TestPlanHandler_DeletePlan_InvalidSID(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/plans/invalid_id", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "invalid_id"})

	handler.DeletePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_DeletePlan_NotFound(t *testing.T) {
	mockUC := &mockDeletePlanUC{err: errors.NewNotFoundError("plan not found", "")}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/plans/plan_nonexistent", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_nonexistent"})

	handler.DeletePlan(c)

	assert.NotEqual(t, http.StatusNoContent, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_DeletePlan_HasActiveSubscriptions(t *testing.T) {
	mockUC := &mockDeletePlanUC{err: errors.NewValidationError("cannot delete plan with active subscriptions", "")}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/plans/plan_test123", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.DeletePlan(c)

	assert.NotEqual(t, http.StatusNoContent, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestPlanHandler_GetPlanPricings
// =====================================================================

func TestPlanHandler_GetPlanPricings_Success(t *testing.T) {
	mockResult := []*subdto.PricingOptionDTO{
		{
			BillingCycle: "monthly",
			Price:        999,
			Currency:     "USD",
		},
	}
	mockUC := &mockGetPlanPricingsUC{result: mockResult}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, mockUC)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/plan_test123/pricings", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.GetPlanPricings(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	var data []*subdto.PricingOptionDTO
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Len(t, data, 1)
}

func TestPlanHandler_GetPlanPricings_Empty(t *testing.T) {
	mockResult := []*subdto.PricingOptionDTO{}
	mockUC := &mockGetPlanPricingsUC{result: mockResult}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, mockUC)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/plan_test123/pricings", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_abc123def456"})

	handler.GetPlanPricings(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	var data []*subdto.PricingOptionDTO
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Len(t, data, 0)
}

func TestPlanHandler_GetPlanPricings_InvalidSID(t *testing.T) {
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/invalid_id/pricings", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "invalid_id"})

	handler.GetPlanPricings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestPlanHandler_GetPlanPricings_PlanNotFound(t *testing.T) {
	mockUC := &mockGetPlanPricingsUC{err: errors.NewNotFoundError("plan not found", "")}
	handler := newTestPlanHandler(nil, nil, nil, nil, nil, nil, nil, nil, mockUC)

	c, w := testutil.NewTestContext(http.MethodGet, "/plans/plan_nonexistent/pricings", nil)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "plan_nonexistent"})

	handler.GetPlanPricings(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}
