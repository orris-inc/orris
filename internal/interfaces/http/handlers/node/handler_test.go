package node

import (
	"context"
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/testutil"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// =====================================================================
// Mock use cases for NodeHandler
// =====================================================================

type mockCreateNodeUC struct {
	result *usecases.CreateNodeResult
	err    error
}

func (m *mockCreateNodeUC) Execute(_ context.Context, _ usecases.CreateNodeCommand) (*usecases.CreateNodeResult, error) {
	return m.result, m.err
}

type mockGetNodeUC struct {
	result *usecases.GetNodeResult
	err    error
}

func (m *mockGetNodeUC) Execute(_ context.Context, _ usecases.GetNodeQuery) (*usecases.GetNodeResult, error) {
	return m.result, m.err
}

type mockUpdateNodeUC struct {
	result *usecases.UpdateNodeResult
	err    error
}

func (m *mockUpdateNodeUC) Execute(_ context.Context, _ usecases.UpdateNodeCommand) (*usecases.UpdateNodeResult, error) {
	return m.result, m.err
}

type mockDeleteNodeUC struct {
	result *usecases.DeleteNodeResult
	err    error
}

func (m *mockDeleteNodeUC) Execute(_ context.Context, _ usecases.DeleteNodeCommand) (*usecases.DeleteNodeResult, error) {
	return m.result, m.err
}

type mockListNodesUC struct {
	result *usecases.ListNodesResult
	err    error
}

func (m *mockListNodesUC) Execute(_ context.Context, _ usecases.ListNodesQuery) (*usecases.ListNodesResult, error) {
	return m.result, m.err
}

type mockGenerateNodeTokenUC struct {
	result *usecases.GenerateNodeTokenResult
	err    error
}

func (m *mockGenerateNodeTokenUC) Execute(_ context.Context, _ usecases.GenerateNodeTokenCommand) (*usecases.GenerateNodeTokenResult, error) {
	return m.result, m.err
}

type mockGenerateNodeInstallScriptUC struct {
	result *usecases.GenerateNodeInstallScriptResult
	err    error
}

func (m *mockGenerateNodeInstallScriptUC) Execute(_ context.Context, _ usecases.GenerateNodeInstallScriptQuery) (*usecases.GenerateNodeInstallScriptResult, error) {
	return m.result, m.err
}

type mockGenerateBatchInstallScriptUC struct {
	result *usecases.GenerateBatchInstallScriptResult
	err    error
}

func (m *mockGenerateBatchInstallScriptUC) Execute(_ context.Context, _ usecases.GenerateBatchInstallScriptQuery) (*usecases.GenerateBatchInstallScriptResult, error) {
	return m.result, m.err
}

// =====================================================================
// Mock use cases for UserNodeHandler
// =====================================================================

type mockCreateUserNodeUC struct {
	result *usecases.CreateUserNodeResult
	err    error
}

func (m *mockCreateUserNodeUC) Execute(_ context.Context, _ usecases.CreateUserNodeCommand) (*usecases.CreateUserNodeResult, error) {
	return m.result, m.err
}

type mockListUserNodesUC struct {
	result *usecases.ListUserNodesResult
	err    error
}

func (m *mockListUserNodesUC) Execute(_ context.Context, _ usecases.ListUserNodesQuery) (*usecases.ListUserNodesResult, error) {
	return m.result, m.err
}

type mockGetUserNodeUC struct {
	result *dto.UserNodeDTO
	err    error
}

func (m *mockGetUserNodeUC) Execute(_ context.Context, _ usecases.GetUserNodeQuery) (*dto.UserNodeDTO, error) {
	return m.result, m.err
}

type mockUpdateUserNodeUC struct {
	result *dto.UserNodeDTO
	err    error
}

func (m *mockUpdateUserNodeUC) Execute(_ context.Context, _ usecases.UpdateUserNodeCommand) (*dto.UserNodeDTO, error) {
	return m.result, m.err
}

type mockDeleteUserNodeUC struct {
	err error
}

func (m *mockDeleteUserNodeUC) Execute(_ context.Context, _ usecases.DeleteUserNodeCommand) error {
	return m.err
}

type mockRegenerateUserNodeTokenUC struct {
	result *usecases.RegenerateUserNodeTokenResult
	err    error
}

func (m *mockRegenerateUserNodeTokenUC) Execute(_ context.Context, _ usecases.RegenerateUserNodeTokenCommand) (*usecases.RegenerateUserNodeTokenResult, error) {
	return m.result, m.err
}

type mockGetUserNodeUsageUC struct {
	result *usecases.GetUserNodeUsageResult
	err    error
}

func (m *mockGetUserNodeUsageUC) Execute(_ context.Context, _ usecases.GetUserNodeUsageQuery) (*usecases.GetUserNodeUsageResult, error) {
	return m.result, m.err
}

type mockGetUserNodeInstallScriptUC struct {
	result *usecases.GetUserNodeInstallScriptResult
	err    error
}

func (m *mockGetUserNodeInstallScriptUC) Execute(_ context.Context, _ usecases.GetUserNodeInstallScriptQuery) (*usecases.GetUserNodeInstallScriptResult, error) {
	return m.result, m.err
}

type mockGetUserBatchInstallScriptUC struct {
	result *usecases.GetUserBatchInstallScriptResult
	err    error
}

func (m *mockGetUserBatchInstallScriptUC) Execute(_ context.Context, _ usecases.GetUserBatchInstallScriptQuery) (*usecases.GetUserBatchInstallScriptResult, error) {
	return m.result, m.err
}

// =====================================================================
// Test constants
// =====================================================================

// Valid SIDs must be "node_" + 12 alphanumeric characters.
const (
	testNodeSID  = "node_xK9mP2vL3nQ7"
	testNodeSID2 = "node_aB3cD4eF5gH6"
)

// =====================================================================
// Test helpers
// =====================================================================

func newTestNodeHandler(
	createUC createNodeUseCase,
	getUC getNodeUseCase,
	updateUC updateNodeUseCase,
	deleteUC deleteNodeUseCase,
	listUC listNodesUseCase,
	generateTokenUC generateNodeTokenUseCase,
	generateInstallScriptUC generateNodeInstallScriptUseCase,
	generateBatchInstallScriptUC generateBatchInstallScriptUseCase,
) *NodeHandler {
	return NewNodeHandler(
		createUC, getUC, updateUC, deleteUC, listUC,
		generateTokenUC, generateInstallScriptUC, generateBatchInstallScriptUC,
		"https://api.test.example.com",
		testutil.NewMockLogger(),
	)
}

func newTestUserNodeHandler(
	createUC createUserNodeUseCase,
	listUC listUserNodesUseCase,
	getUC getUserNodeUseCase,
	updateUC updateUserNodeUseCase,
	deleteUC deleteUserNodeUseCase,
	regenerateTokenUC regenerateUserNodeTokenUseCase,
	getUsageUC getUserNodeUsageUseCase,
	getInstallScriptUC getUserNodeInstallScriptUseCase,
	getBatchInstallScriptUC getUserBatchInstallScriptUseCase,
) *UserNodeHandler {
	return NewUserNodeHandler(
		createUC, listUC, getUC, updateUC, deleteUC,
		regenerateTokenUC, getUsageUC, getInstallScriptUC, getBatchInstallScriptUC,
		"https://api.test.example.com",
		testutil.NewMockLogger(),
	)
}

// =====================================================================
// TestNodeHandler_CreateNode
// =====================================================================

func TestNodeHandler_CreateNode_Success(t *testing.T) {
	mockResult := &usecases.CreateNodeResult{
		NodeID:        1,
		APIToken:      "token_xxx",
		ServerAddress: "1.2.3.4",
		AgentPort:     8388,
		Protocol:      "shadowsocks",
		Status:        "active",
		CreatedAt:     "2025-01-15T10:00:00Z",
	}
	handler := newTestNodeHandler(&mockCreateNodeUC{result: mockResult}, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateNodeRequest{
		Name:             "US-Node-01",
		ServerAddress:    "1.2.3.4",
		AgentPort:        8388,
		Protocol:         "shadowsocks",
		EncryptionMethod: "aes-256-gcm",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/nodes", reqBody)

	handler.CreateNode(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNodeHandler_CreateNode_BindingError(t *testing.T) {
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	// Missing required fields: name, agent_port, protocol
	reqBody := map[string]string{"description": "test"}
	c, w := testutil.NewTestContext(http.MethodPost, "/nodes", reqBody)

	handler.CreateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestNodeHandler_CreateNode_UseCaseError(t *testing.T) {
	mockUC := &mockCreateNodeUC{err: errors.NewConflictError("node with this name already exists")}
	handler := newTestNodeHandler(mockUC, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateNodeRequest{
		Name:             "US-Node-01",
		ServerAddress:    "1.2.3.4",
		AgentPort:        8388,
		Protocol:         "shadowsocks",
		EncryptionMethod: "aes-256-gcm",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/nodes", reqBody)

	handler.CreateNode(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestNodeHandler_GetNode
// =====================================================================

func TestNodeHandler_GetNode_Success(t *testing.T) {
	nodeDTO := &dto.NodeDTO{
		ID:               "node_xK9mP2vL3nQ7",
		Name:             "US-Node-01",
		ServerAddress:    "1.2.3.4",
		AgentPort:        8388,
		Protocol:         "shadowsocks",
		EncryptionMethod: "aes-256-gcm",
		Status:           "active",
	}
	mockUC := &mockGetNodeUC{result: &usecases.GetNodeResult{Node: nodeDTO}}
	handler := newTestNodeHandler(nil, mockUC, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/nodes/node_xK9mP2vL3nQ7", nil)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.GetNode(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNodeHandler_GetNode_InvalidID(t *testing.T) {
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/nodes/invalid_id", nil)
	testutil.SetURLParam(c, "id", "invalid_id")

	handler.GetNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestNodeHandler_GetNode_NotFound(t *testing.T) {
	mockUC := &mockGetNodeUC{err: errors.NewNotFoundError("node not found", "node_xK9mP2vL3nQ7")}
	handler := newTestNodeHandler(nil, mockUC, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/nodes/node_xK9mP2vL3nQ7", nil)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.GetNode(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestNodeHandler_UpdateNode
// =====================================================================

func TestNodeHandler_UpdateNode_Success(t *testing.T) {
	mockResult := &usecases.UpdateNodeResult{
		NodeID:        1,
		Name:          "US-Node-01-Updated",
		ServerAddress: "1.2.3.4",
		AgentPort:     8388,
		Protocol:      "shadowsocks",
		Status:        "active",
		UpdatedAt:     "2025-01-15T12:00:00Z",
	}
	mockUC := &mockUpdateNodeUC{result: mockResult}
	handler := newTestNodeHandler(nil, nil, mockUC, nil, nil, nil, nil, nil)

	name := "US-Node-01-Updated"
	reqBody := UpdateNodeRequest{Name: &name}
	c, w := testutil.NewTestContext(http.MethodPut, "/nodes/node_xK9mP2vL3nQ7", reqBody)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.UpdateNode(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNodeHandler_UpdateNode_InvalidID(t *testing.T) {
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	name := "Updated"
	reqBody := UpdateNodeRequest{Name: &name}
	c, w := testutil.NewTestContext(http.MethodPut, "/nodes/invalid_id", reqBody)
	testutil.SetURLParam(c, "id", "invalid_id")

	handler.UpdateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNodeHandler_UpdateNode_InvalidExpiresAt(t *testing.T) {
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	badDate := "not-a-date"
	reqBody := UpdateNodeRequest{ExpiresAt: &badDate}
	c, w := testutil.NewTestContext(http.MethodPut, "/nodes/node_xK9mP2vL3nQ7", reqBody)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.UpdateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestNodeHandler_UpdateNode_UseCaseError(t *testing.T) {
	mockUC := &mockUpdateNodeUC{err: stderrors.New("unexpected error")}
	handler := newTestNodeHandler(nil, nil, mockUC, nil, nil, nil, nil, nil)

	name := "Updated"
	reqBody := UpdateNodeRequest{Name: &name}
	c, w := testutil.NewTestContext(http.MethodPut, "/nodes/node_xK9mP2vL3nQ7", reqBody)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.UpdateNode(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestNodeHandler_DeleteNode
// =====================================================================

func TestNodeHandler_DeleteNode_Success(t *testing.T) {
	mockResult := &usecases.DeleteNodeResult{
		NodeID:    1,
		DeletedAt: "2025-01-15T12:00:00Z",
	}
	mockUC := &mockDeleteNodeUC{result: mockResult}
	handler := newTestNodeHandler(nil, nil, nil, mockUC, nil, nil, nil, nil)

	c, _ := testutil.NewTestContext(http.MethodDelete, "/nodes/node_xK9mP2vL3nQ7", nil)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.DeleteNode(c)

	// NoContentResponse sets status via c.Status() which may not flush to ResponseRecorder,
	// so we check the gin writer's status directly.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
}

func TestNodeHandler_DeleteNode_InvalidID(t *testing.T) {
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/nodes/bad_format", nil)
	testutil.SetURLParam(c, "id", "bad_format")

	handler.DeleteNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNodeHandler_DeleteNode_NotFound(t *testing.T) {
	mockUC := &mockDeleteNodeUC{err: errors.NewNotFoundError("node not found", "node_xK9mP2vL3nQ7")}
	handler := newTestNodeHandler(nil, nil, nil, mockUC, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/nodes/node_xK9mP2vL3nQ7", nil)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.DeleteNode(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =====================================================================
// TestNodeHandler_ListNodes
// =====================================================================

func TestNodeHandler_ListNodes_Success(t *testing.T) {
	mockResult := &usecases.ListNodesResult{
		Nodes: []*dto.NodeDTO{
			{ID: "node_xK9mP2vL3nQ7", Name: "US-Node-01", Status: "active"},
			{ID: "node_aB3cD4eF5gH6", Name: "JP-Node-01", Status: "active"},
		},
		TotalCount: 2,
		Limit:      20,
		Offset:     0,
	}
	mockUC := &mockListNodesUC{result: mockResult}
	handler := newTestNodeHandler(nil, nil, nil, nil, mockUC, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/nodes", nil)

	handler.ListNodes(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNodeHandler_ListNodes_UseCaseError(t *testing.T) {
	mockUC := &mockListNodesUC{err: stderrors.New("database error")}
	handler := newTestNodeHandler(nil, nil, nil, nil, mockUC, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/nodes", nil)

	handler.ListNodes(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestNodeHandler_GenerateToken
// =====================================================================

func TestNodeHandler_GenerateToken_Success(t *testing.T) {
	mockResult := &usecases.GenerateNodeTokenResult{
		NodeID:      1,
		Token:       "test_token_xxx",
		TokenPrefix: "test_",
	}
	mockUC := &mockGenerateNodeTokenUC{result: mockResult}
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, mockUC, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/nodes/node_xK9mP2vL3nQ7/tokens", nil)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.GenerateToken(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNodeHandler_GenerateToken_InvalidID(t *testing.T) {
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/nodes/bad_id/tokens", nil)
	testutil.SetURLParam(c, "id", "bad_id")

	handler.GenerateToken(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =====================================================================
// TestNodeHandler_GetInstallScript
// =====================================================================

func TestNodeHandler_GetInstallScript_Success(t *testing.T) {
	mockResult := &usecases.GenerateNodeInstallScriptResult{
		InstallCommand: "curl -fsSL ...",
		NodeSID:        "node_xK9mP2vL3nQ7",
		Token:          "test_token",
	}
	mockUC := &mockGenerateNodeInstallScriptUC{result: mockResult}
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, mockUC, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/nodes/node_xK9mP2vL3nQ7/install-script", nil)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.GetInstallScript(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// =====================================================================
// TestNodeHandler_UpdateNodeStatus
// =====================================================================

func TestNodeHandler_UpdateNodeStatus_Success(t *testing.T) {
	mockResult := &usecases.UpdateNodeResult{
		NodeID: 1,
		Status: "maintenance",
	}
	mockUC := &mockUpdateNodeUC{result: mockResult}
	handler := newTestNodeHandler(nil, nil, mockUC, nil, nil, nil, nil, nil)

	reqBody := UpdateNodeStatusRequest{Status: "maintenance"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/nodes/node_xK9mP2vL3nQ7/status", reqBody)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.UpdateNodeStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNodeHandler_UpdateNodeStatus_InvalidStatus(t *testing.T) {
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := map[string]string{"status": "invalid_status"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/nodes/node_xK9mP2vL3nQ7/status", reqBody)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.UpdateNodeStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =====================================================================
// TestNodeHandler_GetBatchInstallScript
// =====================================================================

func TestNodeHandler_GetBatchInstallScript_Success(t *testing.T) {
	mockResult := &usecases.GenerateBatchInstallScriptResult{
		InstallCommand: "curl -fsSL ...",
		Nodes: []usecases.NodeInstallInfo{
			{NodeSID: "node_xK9mP2vL3nQ7", Token: "token1"},
		},
	}
	mockUC := &mockGenerateBatchInstallScriptUC{result: mockResult}
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, mockUC)

	reqBody := BatchInstallScriptRequest{NodeIDs: []string{"node_xK9mP2vL3nQ7"}}
	c, w := testutil.NewTestContext(http.MethodPost, "/nodes/batch-install-script", reqBody)

	handler.GetBatchInstallScript(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNodeHandler_GetBatchInstallScript_BindingError(t *testing.T) {
	handler := newTestNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	// Missing required node_ids
	reqBody := map[string]string{}
	c, w := testutil.NewTestContext(http.MethodPost, "/nodes/batch-install-script", reqBody)

	handler.GetBatchInstallScript(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =====================================================================
// TestUserNodeHandler_CreateNode
// =====================================================================

func TestUserNodeHandler_CreateNode_Success(t *testing.T) {
	mockResult := &usecases.CreateUserNodeResult{
		NodeSID:       "node_xK9mP2vL3nQ7",
		APIToken:      "token_xxx",
		ServerAddress: "1.2.3.4",
		AgentPort:     8388,
		Protocol:      "shadowsocks",
		Status:        "active",
		CreatedAt:     "2025-01-15T10:00:00Z",
	}
	handler := newTestUserNodeHandler(&mockCreateUserNodeUC{result: mockResult}, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateUserNodeRequest{
		Name:      "My-Node-01",
		AgentPort: 8388,
		Protocol:  "shadowsocks",
		Method:    "aes-256-gcm",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/user/nodes", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.CreateNode(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestUserNodeHandler_CreateNode_Unauthenticated(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateUserNodeRequest{
		Name:      "My-Node-01",
		AgentPort: 8388,
		Protocol:  "shadowsocks",
		Method:    "aes-256-gcm",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/user/nodes", reqBody)
	// No auth context set

	handler.CreateNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserNodeHandler_CreateNode_BindingError(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Missing required fields: name, agent_port, protocol
	reqBody := map[string]string{"description": "test"}
	c, w := testutil.NewTestContext(http.MethodPost, "/user/nodes", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.CreateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestUserNodeHandler_CreateNode_UseCaseError(t *testing.T) {
	mockUC := &mockCreateUserNodeUC{err: errors.NewConflictError("node with this name already exists", "")}
	handler := newTestUserNodeHandler(mockUC, nil, nil, nil, nil, nil, nil, nil, nil)

	reqBody := CreateUserNodeRequest{
		Name:      "My-Node-01",
		AgentPort: 8388,
		Protocol:  "shadowsocks",
		Method:    "aes-256-gcm",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/user/nodes", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.CreateNode(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestUserNodeHandler_GetNode
// =====================================================================

func TestUserNodeHandler_GetNode_Success(t *testing.T) {
	mockResult := &dto.UserNodeDTO{
		ID:            "node_xK9mP2vL3nQ7",
		Name:          "My-Node-01",
		ServerAddress: "1.2.3.4",
		AgentPort:     8388,
		Protocol:      "shadowsocks",
		Status:        "active",
	}
	handler := newTestUserNodeHandler(nil, nil, &mockGetUserNodeUC{result: mockResult}, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/user/nodes/node_xK9mP2vL3nQ7", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.GetNode(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestUserNodeHandler_GetNode_NotFound(t *testing.T) {
	mockUC := &mockGetUserNodeUC{err: errors.NewNotFoundError("node not found", "node_xK9mP2vL3nQ7")}
	handler := newTestUserNodeHandler(nil, nil, mockUC, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/user/nodes/node_xK9mP2vL3nQ7", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.GetNode(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestUserNodeHandler_GetNode_InvalidID(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/user/nodes/bad_id", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "bad_id")

	handler.GetNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =====================================================================
// TestUserNodeHandler_UpdateNode
// =====================================================================

func TestUserNodeHandler_UpdateNode_Success(t *testing.T) {
	mockResult := &dto.UserNodeDTO{
		ID:            "node_xK9mP2vL3nQ7",
		Name:          "My-Node-Updated",
		ServerAddress: "1.2.3.4",
		AgentPort:     8388,
		Protocol:      "shadowsocks",
		Status:        "active",
	}
	handler := newTestUserNodeHandler(nil, nil, nil, &mockUpdateUserNodeUC{result: mockResult}, nil, nil, nil, nil, nil)

	name := "My-Node-Updated"
	reqBody := UpdateUserNodeRequest{Name: &name}
	c, w := testutil.NewTestContext(http.MethodPut, "/user/nodes/node_xK9mP2vL3nQ7", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.UpdateNode(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestUserNodeHandler_UpdateNode_Unauthenticated(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	name := "Updated"
	reqBody := UpdateUserNodeRequest{Name: &name}
	c, w := testutil.NewTestContext(http.MethodPut, "/user/nodes/node_xK9mP2vL3nQ7", reqBody)
	// No auth context
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.UpdateNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserNodeHandler_UpdateNode_UseCaseError(t *testing.T) {
	mockUC := &mockUpdateUserNodeUC{err: errors.NewConflictError("node with this name already exists")}
	handler := newTestUserNodeHandler(nil, nil, nil, mockUC, nil, nil, nil, nil, nil)

	name := "Duplicate"
	reqBody := UpdateUserNodeRequest{Name: &name}
	c, w := testutil.NewTestContext(http.MethodPut, "/user/nodes/node_xK9mP2vL3nQ7", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.UpdateNode(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestUserNodeHandler_ListNodes
// =====================================================================

func TestUserNodeHandler_ListNodes_Success(t *testing.T) {
	mockResult := &usecases.ListUserNodesResult{
		Nodes: []*dto.UserNodeDTO{
			{ID: "node_xK9mP2vL3nQ7", Name: "My-Node-01", Status: "active"},
		},
		TotalCount: 1,
		Limit:      20,
		Offset:     0,
	}
	handler := newTestUserNodeHandler(nil, &mockListUserNodesUC{result: mockResult}, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/user/nodes", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListNodes(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestUserNodeHandler_ListNodes_Unauthenticated(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/user/nodes", nil)
	// No auth context

	handler.ListNodes(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// =====================================================================
// TestUserNodeHandler_DeleteNode
// =====================================================================

func TestUserNodeHandler_DeleteNode_Success(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, &mockDeleteUserNodeUC{err: nil}, nil, nil, nil, nil)

	c, _ := testutil.NewTestContext(http.MethodDelete, "/user/nodes/node_xK9mP2vL3nQ7", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.DeleteNode(c)

	// NoContentResponse sets status via c.Status() which may not flush to ResponseRecorder,
	// so we check the gin writer's status directly.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
}

func TestUserNodeHandler_DeleteNode_NotFound(t *testing.T) {
	mockUC := &mockDeleteUserNodeUC{err: errors.NewNotFoundError("node not found", "node_xK9mP2vL3nQ7")}
	handler := newTestUserNodeHandler(nil, nil, nil, nil, mockUC, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/user/nodes/node_xK9mP2vL3nQ7", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.DeleteNode(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserNodeHandler_DeleteNode_Unauthenticated(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/user/nodes/node_xK9mP2vL3nQ7", nil)
	// No auth context
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.DeleteNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// =====================================================================
// TestUserNodeHandler_RegenerateToken
// =====================================================================

func TestUserNodeHandler_RegenerateToken_Success(t *testing.T) {
	mockResult := &usecases.RegenerateUserNodeTokenResult{
		NodeSID:  "node_xK9mP2vL3nQ7",
		APIToken: "new_token_xxx",
	}
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, &mockRegenerateUserNodeTokenUC{result: mockResult}, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/user/nodes/node_xK9mP2vL3nQ7/regenerate-token", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.RegenerateToken(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestUserNodeHandler_RegenerateToken_Forbidden(t *testing.T) {
	mockUC := &mockRegenerateUserNodeTokenUC{err: errors.NewForbiddenError("access denied to this node")}
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, mockUC, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/user/nodes/node_xK9mP2vL3nQ7/regenerate-token", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.RegenerateToken(c)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestUserNodeHandler_GetUsage
// =====================================================================

func TestUserNodeHandler_GetUsage_Success(t *testing.T) {
	mockResult := &usecases.GetUserNodeUsageResult{
		NodeCount: 3,
		NodeLimit: 10,
	}
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, &mockGetUserNodeUsageUC{result: mockResult}, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/user/nodes/usage", nil)
	testutil.SetAuthContext(c, 1)

	handler.GetUsage(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// =====================================================================
// TestUserNodeHandler_GetInstallScript
// =====================================================================

func TestUserNodeHandler_GetInstallScript_Success(t *testing.T) {
	mockResult := &usecases.GetUserNodeInstallScriptResult{
		InstallCommand: "curl -fsSL ...",
		NodeSID:        "node_xK9mP2vL3nQ7",
		Token:          "token_xxx",
	}
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, &mockGetUserNodeInstallScriptUC{result: mockResult}, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/user/nodes/node_xK9mP2vL3nQ7/install-script", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.GetInstallScript(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestUserNodeHandler_GetInstallScript_Unauthenticated(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/user/nodes/node_xK9mP2vL3nQ7/install-script", nil)
	// No auth context
	testutil.SetURLParam(c, "id", "node_xK9mP2vL3nQ7")

	handler.GetInstallScript(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// =====================================================================
// TestUserNodeHandler_GetBatchInstallScript
// =====================================================================

func TestUserNodeHandler_GetBatchInstallScript_Success(t *testing.T) {
	mockResult := &usecases.GetUserBatchInstallScriptResult{
		InstallCommand: "curl -fsSL ...",
		Nodes: []usecases.NodeInstallInfo{
			{NodeSID: "node_xK9mP2vL3nQ7", Token: "token1"},
		},
	}
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, &mockGetUserBatchInstallScriptUC{result: mockResult})

	reqBody := UserBatchInstallScriptRequest{NodeIDs: []string{"node_xK9mP2vL3nQ7"}}
	c, w := testutil.NewTestContext(http.MethodPost, "/user/nodes/batch-install-script", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.GetBatchInstallScript(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestUserNodeHandler_GetBatchInstallScript_BindingError(t *testing.T) {
	handler := newTestUserNodeHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Missing required node_ids
	reqBody := map[string]string{}
	c, w := testutil.NewTestContext(http.MethodPost, "/user/nodes/batch-install-script", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.GetBatchInstallScript(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
