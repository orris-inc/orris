package rule

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/testutil"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// =====================================================================
// Mock use cases
// =====================================================================

type mockCreateRuleUC struct {
	result *usecases.CreateForwardRuleResult
	err    error
}

func (m *mockCreateRuleUC) Execute(ctx context.Context, cmd usecases.CreateForwardRuleCommand) (*usecases.CreateForwardRuleResult, error) {
	return m.result, m.err
}

type mockGetRuleUC struct {
	result *dto.ForwardRuleDTO
	err    error
}

func (m *mockGetRuleUC) Execute(ctx context.Context, query usecases.GetForwardRuleQuery) (*dto.ForwardRuleDTO, error) {
	return m.result, m.err
}

type mockUpdateRuleUC struct {
	err error
}

func (m *mockUpdateRuleUC) Execute(ctx context.Context, cmd usecases.UpdateForwardRuleCommand) error {
	return m.err
}

type mockDeleteRuleUC struct {
	err error
}

func (m *mockDeleteRuleUC) Execute(ctx context.Context, cmd usecases.DeleteForwardRuleCommand) error {
	return m.err
}

type mockListRulesUC struct {
	result *usecases.ListForwardRulesResult
	err    error
}

func (m *mockListRulesUC) Execute(ctx context.Context, query usecases.ListForwardRulesQuery) (*usecases.ListForwardRulesResult, error) {
	return m.result, m.err
}

type mockEnableRuleUC struct {
	err error
}

func (m *mockEnableRuleUC) Execute(ctx context.Context, cmd usecases.EnableForwardRuleCommand) error {
	return m.err
}

type mockDisableRuleUC struct {
	err error
}

func (m *mockDisableRuleUC) Execute(ctx context.Context, cmd usecases.DisableForwardRuleCommand) error {
	return m.err
}

type mockResetTrafficUC struct {
	err error
}

func (m *mockResetTrafficUC) Execute(ctx context.Context, cmd usecases.ResetForwardRuleTrafficCommand) error {
	return m.err
}

type mockReorderRulesUC struct {
	err error
}

func (m *mockReorderRulesUC) Execute(ctx context.Context, cmd usecases.ReorderForwardRulesCommand) error {
	return m.err
}

type mockBatchRuleUC struct {
	createResult       *dto.BatchCreateResponse
	createErr          error
	deleteResult       *dto.BatchOperationResult
	deleteErr          error
	toggleStatusResult *dto.BatchOperationResult
	toggleStatusErr    error
	updateResult       *dto.BatchOperationResult
	updateErr          error
}

func (m *mockBatchRuleUC) BatchCreate(ctx context.Context, cmd usecases.BatchCreateCommand) (*dto.BatchCreateResponse, error) {
	return m.createResult, m.createErr
}

func (m *mockBatchRuleUC) BatchDelete(ctx context.Context, cmd usecases.BatchDeleteCommand) (*dto.BatchOperationResult, error) {
	return m.deleteResult, m.deleteErr
}

func (m *mockBatchRuleUC) BatchToggleStatus(ctx context.Context, cmd usecases.BatchToggleStatusCommand) (*dto.BatchOperationResult, error) {
	return m.toggleStatusResult, m.toggleStatusErr
}

func (m *mockBatchRuleUC) BatchUpdate(ctx context.Context, cmd usecases.BatchUpdateCommand) (*dto.BatchOperationResult, error) {
	return m.updateResult, m.updateErr
}

type mockProbeService struct {
	result *dto.RuleProbeResponse
	err    error
}

func (m *mockProbeService) ProbeRuleByShortID(ctx context.Context, shortID string, ipVersionOverride string) (*dto.RuleProbeResponse, error) {
	return m.result, m.err
}

// =====================================================================
// Test helpers
// =====================================================================

func newTestHandler(
	createUC createRuleUseCase,
	getUC getRuleUseCase,
	updateUC updateRuleUseCase,
	deleteUC deleteRuleUseCase,
	listUC listRulesUseCase,
	enableUC enableRuleUseCase,
	disableUC disableRuleUseCase,
	resetTrafficUC resetTrafficUseCase,
	reorderUC reorderRulesUseCase,
	batchUC batchRuleUseCase,
	probeSvc probeService,
) *Handler {
	return NewHandler(
		createUC, getUC, updateUC, deleteUC, listUC,
		enableUC, disableUC, resetTrafficUC, reorderUC,
		batchUC, probeSvc, testutil.NewMockLogger(),
	)
}

// validRuleSID returns a valid forward rule SID for testing.
func validRuleSID() string {
	return "fr_xK9mP2vL3nQz"
}

// validAgentSID returns a valid forward agent SID for testing.
func validAgentSID() string {
	return "fa_xK9mP2vL3nQz"
}

// validNodeSID returns a valid node SID for testing.
func validNodeSID() string {
	return "node_xK9mP2vL3nQ7"
}

// =====================================================================
// TestHandler_CreateRule
// =====================================================================

func TestHandler_CreateRule_Success(t *testing.T) {
	mockResult := &usecases.CreateForwardRuleResult{
		ID:         validRuleSID(),
		Name:       "Test Rule",
		RuleType:   "direct",
		ListenPort: 8080,
		Protocol:   "tcp",
		Status:     "enabled",
	}
	handler := newTestHandler(&mockCreateRuleUC{result: mockResult}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateForwardRuleRequest{
		AgentID:    validAgentSID(),
		RuleType:   "direct",
		Name:       "Test Rule",
		ListenPort: 8080,
		Protocol:   "tcp",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_CreateRule_InvalidRequest(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Missing required fields (name, rule_type)
	reqBody := map[string]string{"agent_id": validAgentSID()}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_CreateRule_MissingAgentID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateForwardRuleRequest{
		RuleType: "direct",
		Name:     "Test Rule",
		Protocol: "tcp",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_CreateRule_InvalidAgentIDFormat(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateForwardRuleRequest{
		AgentID:    "invalid_id",
		RuleType:   "direct",
		Name:       "Test Rule",
		ListenPort: 8080,
		Protocol:   "tcp",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_CreateRule_MissingProtocol(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateForwardRuleRequest{
		AgentID:    validAgentSID(),
		RuleType:   "direct",
		Name:       "Test Rule",
		ListenPort: 8080,
		// Missing protocol
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_CreateRule_UseCaseError(t *testing.T) {
	mockUC := &mockCreateRuleUC{err: errors.NewConflictError("port already in use", "")}
	handler := newTestHandler(mockUC, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateForwardRuleRequest{
		AgentID:    validAgentSID(),
		RuleType:   "direct",
		Name:       "Test Rule",
		ListenPort: 8080,
		Protocol:   "tcp",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_CreateRule_ExternalType_Success(t *testing.T) {
	mockResult := &usecases.CreateForwardRuleResult{
		ID:         validRuleSID(),
		Name:       "External Rule",
		RuleType:   "external",
		ListenPort: 8080,
		Protocol:   "tcp",
		Status:     "enabled",
	}
	handler := newTestHandler(&mockCreateRuleUC{result: mockResult}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateForwardRuleRequest{
		RuleType:      "external",
		Name:          "External Rule",
		ServerAddress: "example.com",
		ListenPort:    8080,
		TargetNodeID:  validNodeSID(),
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_CreateRule_ExternalType_MissingServerAddress(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateForwardRuleRequest{
		RuleType:     "external",
		Name:         "External Rule",
		ListenPort:   8080,
		TargetNodeID: validNodeSID(),
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_CreateRule_MutuallyExclusiveExitAgents(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateForwardRuleRequest{
		AgentID:     validAgentSID(),
		RuleType:    "entry",
		Name:        "Entry Rule",
		ListenPort:  8080,
		Protocol:    "tcp",
		ExitAgentID: validAgentSID(),
		ExitAgents:  []ExitAgentRequest{{AgentID: validAgentSID(), Weight: 50}},
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules", reqBody)

	handler.CreateRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_GetRule
// =====================================================================

func TestHandler_GetRule_Success(t *testing.T) {
	mockResult := &dto.ForwardRuleDTO{
		ID:       validRuleSID(),
		Name:     "Test Rule",
		RuleType: "direct",
		Protocol: "tcp",
	}
	handler := newTestHandler(nil, &mockGetRuleUC{result: mockResult}, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/forward-rules/"+validRuleSID(), nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.GetRule(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_GetRule_InvalidID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/forward-rules/invalid_id", nil)
	testutil.SetURLParam(c, "id", "invalid_id")

	handler.GetRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_GetRule_NotFound(t *testing.T) {
	mockUC := &mockGetRuleUC{err: errors.NewNotFoundError("forward rule not found", "")}
	handler := newTestHandler(nil, mockUC, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/forward-rules/"+validRuleSID(), nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.GetRule(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_GetRule_MissingID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/forward-rules/", nil)
	// No URL param set

	handler.GetRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_UpdateRule
// =====================================================================

func TestHandler_UpdateRule_Success(t *testing.T) {
	mockUC := &mockUpdateRuleUC{err: nil}
	handler := newTestHandler(nil, nil, mockUC, nil, nil, nil, nil, nil, nil, nil, nil)

	name := "Updated Rule"
	reqBody := UpdateForwardRuleRequest{
		Name: &name,
	}
	c, w := testutil.NewTestContext(http.MethodPut, "/forward-rules/"+validRuleSID(), reqBody)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.UpdateRule(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_UpdateRule_InvalidID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPut, "/forward-rules/invalid_id", map[string]string{"name": "test"})
	testutil.SetURLParam(c, "id", "invalid_id")

	handler.UpdateRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_UpdateRule_UseCaseError(t *testing.T) {
	mockUC := &mockUpdateRuleUC{err: errors.NewNotFoundError("forward rule not found", "")}
	handler := newTestHandler(nil, nil, mockUC, nil, nil, nil, nil, nil, nil, nil, nil)

	name := "Updated Rule"
	reqBody := UpdateForwardRuleRequest{
		Name: &name,
	}
	c, w := testutil.NewTestContext(http.MethodPut, "/forward-rules/"+validRuleSID(), reqBody)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.UpdateRule(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_UpdateRule_InvalidAgentID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	invalidID := "invalid_agent"
	reqBody := UpdateForwardRuleRequest{
		AgentID: &invalidID,
	}
	c, w := testutil.NewTestContext(http.MethodPut, "/forward-rules/"+validRuleSID(), reqBody)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.UpdateRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_DeleteRule
// =====================================================================

func TestHandler_DeleteRule_Success(t *testing.T) {
	mockUC := &mockDeleteRuleUC{err: nil}
	handler := newTestHandler(nil, nil, nil, mockUC, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/forward-rules/"+validRuleSID(), nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.DeleteRule(c)

	// gin's c.Status() sets the status on the writer; use Writer.Status() for reliable check.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
	assert.Empty(t, w.Body.String())
}

func TestHandler_DeleteRule_InvalidID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/forward-rules/invalid_id", nil)
	testutil.SetURLParam(c, "id", "invalid_id")

	handler.DeleteRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_DeleteRule_UseCaseError(t *testing.T) {
	mockUC := &mockDeleteRuleUC{err: errors.NewNotFoundError("forward rule not found", "")}
	handler := newTestHandler(nil, nil, nil, mockUC, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/forward-rules/"+validRuleSID(), nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.DeleteRule(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_ListRules
// =====================================================================

func TestHandler_ListRules_Success(t *testing.T) {
	mockResult := &usecases.ListForwardRulesResult{
		Rules: []*dto.ForwardRuleDTO{
			{ID: validRuleSID(), Name: "Rule 1", RuleType: "direct"},
		},
		Total: 1,
		Page:  1,
		Pages: 1,
	}
	handler := newTestHandler(nil, nil, nil, nil, &mockListRulesUC{result: mockResult}, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/forward-rules", nil)

	handler.ListRules(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_ListRules_WithQueryParams(t *testing.T) {
	mockResult := &usecases.ListForwardRulesResult{
		Rules: []*dto.ForwardRuleDTO{},
		Total: 0,
		Page:  1,
		Pages: 0,
	}
	handler := newTestHandler(nil, nil, nil, nil, &mockListRulesUC{result: mockResult}, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/forward-rules", nil)
	testutil.SetQueryParams(c, map[string]string{
		"page":      "1",
		"page_size": "10",
		"name":      "test",
		"protocol":  "tcp",
		"status":    "enabled",
	})

	handler.ListRules(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_ListRules_UseCaseError(t *testing.T) {
	mockUC := &mockListRulesUC{err: errors.NewValidationError("invalid filter", "")}
	handler := newTestHandler(nil, nil, nil, nil, mockUC, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/forward-rules", nil)

	handler.ListRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_EnableRule
// =====================================================================

func TestHandler_EnableRule_Success(t *testing.T) {
	mockUC := &mockEnableRuleUC{err: nil}
	handler := newTestHandler(nil, nil, nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/enable", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.EnableRule(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_EnableRule_InvalidID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/invalid_id/enable", nil)
	testutil.SetURLParam(c, "id", "invalid_id")

	handler.EnableRule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_EnableRule_UseCaseError(t *testing.T) {
	mockUC := &mockEnableRuleUC{err: errors.NewNotFoundError("rule not found", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/enable", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.EnableRule(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_DisableRule
// =====================================================================

func TestHandler_DisableRule_Success(t *testing.T) {
	mockUC := &mockDisableRuleUC{err: nil}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, mockUC, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/disable", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.DisableRule(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_DisableRule_UseCaseError(t *testing.T) {
	mockUC := &mockDisableRuleUC{err: errors.NewConflictError("rule already disabled", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, mockUC, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/disable", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.DisableRule(c)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_ResetTraffic
// =====================================================================

func TestHandler_ResetTraffic_Success(t *testing.T) {
	mockUC := &mockResetTrafficUC{err: nil}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, mockUC, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/reset-traffic", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.ResetTraffic(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_ResetTraffic_InvalidID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/invalid_id/reset-traffic", nil)
	testutil.SetURLParam(c, "id", "invalid_id")

	handler.ResetTraffic(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_ResetTraffic_UseCaseError(t *testing.T) {
	mockUC := &mockResetTrafficUC{err: errors.NewNotFoundError("rule not found", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, mockUC, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/reset-traffic", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.ResetTraffic(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_ProbeRule
// =====================================================================

func TestHandler_ProbeRule_Success(t *testing.T) {
	mockResult := &dto.RuleProbeResponse{
		RuleID:   validRuleSID(),
		RuleType: "direct",
		Success:  true,
	}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &mockProbeService{result: mockResult})

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/probe", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.ProbeRule(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_ProbeRule_ServiceUnavailable(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/probe", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.ProbeRule(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandler_ProbeRule_ProbeError(t *testing.T) {
	mockSvc := &mockProbeService{err: errors.NewNotFoundError("rule not found", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockSvc)

	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/"+validRuleSID()+"/probe", nil)
	testutil.SetURLParam(c, "id", validRuleSID())

	handler.ProbeRule(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_ReorderRules
// =====================================================================

func TestHandler_ReorderRules_Success(t *testing.T) {
	mockUC := &mockReorderRulesUC{err: nil}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil, nil)

	reqBody := ReorderForwardRulesRequest{
		RuleOrders: []ForwardRuleOrder{
			{RuleID: validRuleSID(), SortOrder: 1},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/reorder", reqBody)

	handler.ReorderRules(c)

	// gin's c.Status() sets the status on the writer; use Writer.Status() for reliable check.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
	assert.Empty(t, w.Body.String())
}

func TestHandler_ReorderRules_InvalidRequest(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Empty rule_orders
	reqBody := map[string]interface{}{}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/reorder", reqBody)

	handler.ReorderRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_ReorderRules_InvalidRuleID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := ReorderForwardRulesRequest{
		RuleOrders: []ForwardRuleOrder{
			{RuleID: "invalid_id", SortOrder: 1},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/reorder", reqBody)

	handler.ReorderRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_ReorderRules_UseCaseError(t *testing.T) {
	mockUC := &mockReorderRulesUC{err: errors.NewNotFoundError("rule not found", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil, nil)

	reqBody := ReorderForwardRulesRequest{
		RuleOrders: []ForwardRuleOrder{
			{RuleID: validRuleSID(), SortOrder: 1},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/reorder", reqBody)

	handler.ReorderRules(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_BatchCreateRules
// =====================================================================

func TestHandler_BatchCreateRules_Success(t *testing.T) {
	mockResult := &dto.BatchCreateResponse{
		Succeeded: []dto.BatchCreateResult{
			{Index: 0, ID: validRuleSID()},
		},
		Failed: []dto.BatchOperationErr{},
	}
	mockUC := &mockBatchRuleUC{createResult: mockResult}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	reqBody := BatchCreateForwardRulesRequest{
		Rules: []CreateForwardRuleRequest{
			{
				AgentID:    validAgentSID(),
				RuleType:   "direct",
				Name:       "Batch Rule 1",
				ListenPort: 8080,
				Protocol:   "tcp",
			},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/batch", reqBody)

	handler.BatchCreateRules(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_BatchCreateRules_InvalidRequest(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Empty rules array
	reqBody := map[string]interface{}{}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/batch", reqBody)

	handler.BatchCreateRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_BatchCreateRules_PreValidationFailure(t *testing.T) {
	// All rules fail pre-validation (invalid agent_id), so UseCase is NOT called
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, &mockBatchRuleUC{}, nil)

	reqBody := BatchCreateForwardRulesRequest{
		Rules: []CreateForwardRuleRequest{
			{
				AgentID:    "invalid_agent",
				RuleType:   "direct",
				Name:       "Bad Rule",
				ListenPort: 8080,
				Protocol:   "tcp",
			},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/batch", reqBody)

	handler.BatchCreateRules(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Check that response has failures
	var batchResp dto.BatchCreateResponse
	err = json.Unmarshal(resp.Data, &batchResp)
	require.NoError(t, err)
	assert.Empty(t, batchResp.Succeeded)
	assert.Len(t, batchResp.Failed, 1)
}

func TestHandler_BatchCreateRules_UseCaseError(t *testing.T) {
	mockUC := &mockBatchRuleUC{createErr: errors.NewValidationError("batch size exceeds limit", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	reqBody := BatchCreateForwardRulesRequest{
		Rules: []CreateForwardRuleRequest{
			{
				AgentID:    validAgentSID(),
				RuleType:   "direct",
				Name:       "Rule 1",
				ListenPort: 8080,
				Protocol:   "tcp",
			},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/forward-rules/batch", reqBody)

	handler.BatchCreateRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_BatchDeleteRules
// =====================================================================

func TestHandler_BatchDeleteRules_Success(t *testing.T) {
	mockResult := &dto.BatchOperationResult{
		Succeeded: []string{validRuleSID()},
		Failed:    []dto.BatchOperationErr{},
	}
	mockUC := &mockBatchRuleUC{deleteResult: mockResult}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	reqBody := BatchDeleteForwardRulesRequest{
		RuleIDs: []string{validRuleSID()},
	}
	c, w := testutil.NewTestContext(http.MethodDelete, "/forward-rules/batch", reqBody)

	handler.BatchDeleteRules(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_BatchDeleteRules_InvalidRequest(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Missing rule_ids
	reqBody := map[string]interface{}{}
	c, w := testutil.NewTestContext(http.MethodDelete, "/forward-rules/batch", reqBody)

	handler.BatchDeleteRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_BatchDeleteRules_UseCaseError(t *testing.T) {
	mockUC := &mockBatchRuleUC{deleteErr: errors.NewValidationError("rule_ids is required", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	reqBody := BatchDeleteForwardRulesRequest{
		RuleIDs: []string{validRuleSID()},
	}
	c, w := testutil.NewTestContext(http.MethodDelete, "/forward-rules/batch", reqBody)

	handler.BatchDeleteRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_BatchToggleStatus
// =====================================================================

func TestHandler_BatchToggleStatus_Enable_Success(t *testing.T) {
	mockResult := &dto.BatchOperationResult{
		Succeeded: []string{validRuleSID()},
		Failed:    []dto.BatchOperationErr{},
	}
	mockUC := &mockBatchRuleUC{toggleStatusResult: mockResult}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	reqBody := BatchToggleStatusRequest{
		RuleIDs: []string{validRuleSID()},
		Status:  "enabled",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/batch/status", reqBody)

	handler.BatchToggleStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_BatchToggleStatus_InvalidRequest(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Missing status
	reqBody := map[string]interface{}{
		"rule_ids": []string{validRuleSID()},
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/batch/status", reqBody)

	handler.BatchToggleStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_BatchToggleStatus_UseCaseError(t *testing.T) {
	mockUC := &mockBatchRuleUC{toggleStatusErr: errors.NewNotFoundError("rules not found", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	reqBody := BatchToggleStatusRequest{
		RuleIDs: []string{validRuleSID()},
		Status:  "disabled",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/batch/status", reqBody)

	handler.BatchToggleStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_BatchUpdateRules
// =====================================================================

func TestHandler_BatchUpdateRules_Success(t *testing.T) {
	mockResult := &dto.BatchOperationResult{
		Succeeded: []string{validRuleSID()},
		Failed:    []dto.BatchOperationErr{},
	}
	mockUC := &mockBatchRuleUC{updateResult: mockResult}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	name := "Updated"
	reqBody := BatchUpdateForwardRulesRequest{
		Updates: []BatchUpdateItem{
			{RuleID: validRuleSID(), Name: &name},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/batch", reqBody)

	handler.BatchUpdateRules(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_BatchUpdateRules_InvalidRuleID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, &mockBatchRuleUC{}, nil)

	name := "Updated"
	reqBody := BatchUpdateForwardRulesRequest{
		Updates: []BatchUpdateItem{
			{RuleID: "invalid_id", Name: &name},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/batch", reqBody)

	handler.BatchUpdateRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_BatchUpdateRules_InvalidAgentID(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, &mockBatchRuleUC{}, nil)

	invalidAgent := "invalid_agent"
	reqBody := BatchUpdateForwardRulesRequest{
		Updates: []BatchUpdateItem{
			{RuleID: validRuleSID(), AgentID: &invalidAgent},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/batch", reqBody)

	handler.BatchUpdateRules(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestHandler_BatchUpdateRules_UseCaseError(t *testing.T) {
	mockUC := &mockBatchRuleUC{updateErr: errors.NewNotFoundError("rules not found", "")}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, mockUC, nil)

	name := "Updated"
	reqBody := BatchUpdateForwardRulesRequest{
		Updates: []BatchUpdateItem{
			{RuleID: validRuleSID(), Name: &name},
		},
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/batch", reqBody)

	handler.BatchUpdateRules(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestHandler_UpdateStatus
// =====================================================================

func TestHandler_UpdateStatus_Enable(t *testing.T) {
	mockUC := &mockEnableRuleUC{err: nil}
	handler := newTestHandler(nil, nil, nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	reqBody := UpdateStatusRequest{Status: "enabled"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/"+validRuleSID()+"/status", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: validRuleSID()})

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_UpdateStatus_Disable(t *testing.T) {
	mockUC := &mockDisableRuleUC{err: nil}
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, mockUC, nil, nil, nil, nil)

	reqBody := UpdateStatusRequest{Status: "disabled"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/"+validRuleSID()+"/status", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: validRuleSID()})

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_UpdateStatus_InvalidRequest(t *testing.T) {
	handler := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Invalid status value
	reqBody := map[string]string{"status": "invalid"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/forward-rules/"+validRuleSID()+"/status", reqBody)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: validRuleSID()})

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}
