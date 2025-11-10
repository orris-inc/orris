package node

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// COMMENTED: Unused imports removed (only used by Mock tests)
// "bytes", "encoding/json", "fmt", "net/http"
// "github.com/stretchr/testify/require"
// "context", "orris/internal/application/node/usecases"

// ============================================================================
// Mock Objects - COMMENTED OUT
// ============================================================================
// REASON: Violates CLAUDE.md rule - "不允许mock数据"
// TODO: Refactor these tests to use real dependencies or integration test setup
// ============================================================================
/*
type mockCreateNodeUC struct {
	executeFunc func(ctx context.Context, cmd usecases.CreateNodeCommand) (*usecases.CreateNodeResult, error)
}

func (m *mockCreateNodeUC) Execute(ctx context.Context, cmd usecases.CreateNodeCommand) (*usecases.CreateNodeResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.CreateNodeResult{
		NodeID:        1,
		APIToken:      "test-token-123",
		ServerAddress: cmd.ServerAddress,
		ServerPort:    cmd.ServerPort,
	}, nil
}

type mockUpdateNodeUC struct {
	executeFunc func(ctx context.Context, cmd usecases.UpdateNodeCommand) (*usecases.UpdateNodeResult, error)
}

func (m *mockUpdateNodeUC) Execute(ctx context.Context, cmd usecases.UpdateNodeCommand) (*usecases.UpdateNodeResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.UpdateNodeResult{
		NodeID:        cmd.NodeID,
		Name:          *cmd.Name,
		ServerAddress: *cmd.ServerAddress,
	}, nil
}

type mockDeleteNodeUC struct {
	executeFunc func(ctx context.Context, cmd usecases.DeleteNodeCommand) (*usecases.DeleteNodeResult, error)
}

func (m *mockDeleteNodeUC) Execute(ctx context.Context, cmd usecases.DeleteNodeCommand) (*usecases.DeleteNodeResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.DeleteNodeResult{}, nil
}

type mockListNodesUC struct {
	executeFunc func(ctx context.Context, query usecases.ListNodesQuery) (*usecases.ListNodesResult, error)
}

func (m *mockListNodesUC) Execute(ctx context.Context, query usecases.ListNodesQuery) (*usecases.ListNodesResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, query)
	}
	return &usecases.ListNodesResult{
		Nodes: []usecases.NodeListItem{
			{
				ID:            1,
				Name:          "Test Node 1",
				ServerAddress: "192.168.1.1",
				ServerPort:    8388,
				Status:        "active",
				Country:       "US",
			},
			{
				ID:            2,
				Name:          "Test Node 2",
				ServerAddress: "192.168.1.2",
				ServerPort:    8388,
				Status:        "active",
				Country:       "US",
			},
		},
		TotalCount: 2,
	}, nil
}

type mockGenerateTokenUC struct {
	executeFunc func(ctx context.Context, cmd usecases.GenerateNodeTokenCommand) (*usecases.GenerateNodeTokenResult, error)
}

func (m *mockGenerateTokenUC) Execute(ctx context.Context, cmd usecases.GenerateNodeTokenCommand) (*usecases.GenerateNodeTokenResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.GenerateNodeTokenResult{
		Token:     "new-test-token-456",
		ExpiresAt: nil,
	}, nil
}
*/

// ============================================================================
// Helper Function: setupTestRouter - COMMENTED OUT
// ============================================================================
// REASON: Uses Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or test container setup
// ============================================================================
/*
func setupTestRouter() (*gin.Engine, *NodeHandler) {
	gin.SetMode(gin.TestMode)

	createUC := &mockCreateNodeUC{}
	updateUC := &mockUpdateNodeUC{}
	deleteUC := &mockDeleteNodeUC{}
	listUC := &mockListNodesUC{}
	generateTokenUC := &mockGenerateTokenUC{}

	handler := NewNodeHandler(createUC, updateUC, deleteUC, listUC, generateTokenUC)

	router := gin.New()
	router.POST("/nodes", handler.CreateNode)
	router.GET("/nodes/:id", handler.GetNode)
	router.PUT("/nodes/:id", handler.UpdateNode)
	router.DELETE("/nodes/:id", handler.DeleteNode)
	router.GET("/nodes", handler.ListNodes)
	router.POST("/nodes/:id/token", handler.GenerateToken)

	return router, handler
}
*/

// ============================================================================
// Test: TestCreateNode_Success - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestCreateNode_Success(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := CreateNodeRequest{
		Name:          "Test Node",
		ServerAddress: "192.168.1.100",
		ServerPort:    8388,
		Method:        "aes-256-gcm",
		Password:      "testpassword123",
		Country:       "US",
		Region:        "California",
		Description:   "Test node description",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Node created successfully", response["message"])
	assert.NotNil(t, response["data"])
}
*/

// ============================================================================
// Test: TestCreateNode_ValidationError - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestCreateNode_ValidationError(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name           string
		reqBody        interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing required name",
			reqBody: map[string]interface{}{
				"server_address": "192.168.1.100",
				"server_port":    8388,
				"method":         "aes-256-gcm",
				"password":       "testpassword123",
				"country":        "US",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal",
		},
		{
			name: "missing required server_address",
			reqBody: map[string]interface{}{
				"name":        "Test Node",
				"server_port": 8388,
				"method":      "aes-256-gcm",
				"password":    "testpassword123",
				"country":     "US",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal",
		},
		{
			name:           "invalid json",
			reqBody:        "invalid-json",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.reqBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/nodes", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
			assert.NotNil(t, response["error"])
		})
	}
}
*/

// ============================================================================
// Test: TestGetNode_Success - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestGetNode_Success(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/nodes/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}
*/

// ============================================================================
// Test: TestGetNode_InvalidID - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestGetNode_InvalidID(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name     string
		nodeID   string
		wantCode int
	}{
		{
			name:     "non-numeric id",
			nodeID:   "invalid",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "zero id",
			nodeID:   "0",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "negative id",
			nodeID:   "-1",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/nodes/%s", tt.nodeID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
		})
	}
}
*/

// ============================================================================
// Test: TestUpdateNode_Success - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestUpdateNode_Success(t *testing.T) {
	router, _ := setupTestRouter()

	name := "Updated Node"
	serverAddr := "192.168.1.200"

	reqBody := UpdateNodeRequest{
		Name:          &name,
		ServerAddress: &serverAddr,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/nodes/1", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Node updated successfully", response["message"])
}
*/

// ============================================================================
// Test: TestUpdateNode_InvalidJSON - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestUpdateNode_InvalidJSON(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodPut, "/nodes/1", bytes.NewBufferString("invalid-json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}
*/

// ============================================================================
// Test: TestDeleteNode_Success - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestDeleteNode_Success(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/nodes/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}
*/

// ============================================================================
// Test: TestDeleteNode_InvalidID - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestDeleteNode_InvalidID(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/nodes/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
*/

// ============================================================================
// Test: TestListNodes_Success - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestListNodes_Success(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["items"])
	assert.Equal(t, float64(2), data["total"])
	assert.Equal(t, float64(1), data["page"])
}
*/

// ============================================================================
// Test: TestListNodes_WithPagination - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestListNodes_WithPagination(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name         string
		queryParams  string
		expectedPage int
		expectedSize int
	}{
		{
			name:         "default pagination",
			queryParams:  "",
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name:         "custom page and size",
			queryParams:  "?page=2&page_size=10",
			expectedPage: 2,
			expectedSize: 10,
		},
		{
			name:         "invalid page defaults to 1",
			queryParams:  "?page=-1",
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name:         "page_size too large defaults to 20",
			queryParams:  "?page_size=200",
			expectedPage: 1,
			expectedSize: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/nodes"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			assert.Equal(t, float64(tt.expectedPage), data["page"])
			assert.Equal(t, float64(tt.expectedSize), data["page_size"])
		})
	}
}
*/

// ============================================================================
// Test: TestListNodes_WithFilters - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestListNodes_WithFilters(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/nodes?status=active&country=US", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
}
*/

// ============================================================================
// Test: TestGenerateToken_Success - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestGenerateToken_Success(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/nodes/1/token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Token generated successfully", response["message"])
	assert.NotNil(t, response["data"])
}
*/

// ============================================================================
// Test: TestGenerateToken_InvalidID - COMMENTED OUT
// ============================================================================
// REASON: Uses setupTestRouter which depends on Mock objects - violates CLAUDE.md rule
// TODO: Refactor to use real dependencies or integration test setup
// ============================================================================
/*
func TestGenerateToken_InvalidID(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/nodes/invalid/token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}
*/

func TestParseNodeID(t *testing.T) {
	tests := []struct {
		name      string
		urlParam  string
		wantID    uint
		wantError bool
	}{
		{
			name:      "valid id",
			urlParam:  "123",
			wantID:    123,
			wantError: false,
		},
		{
			name:      "zero id",
			urlParam:  "0",
			wantID:    0,
			wantError: true,
		},
		{
			name:      "negative id",
			urlParam:  "-1",
			wantID:    0,
			wantError: true,
		},
		{
			name:      "non-numeric",
			urlParam:  "abc",
			wantID:    0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Params = gin.Params{
				{Key: "id", Value: tt.urlParam},
			}

			id, err := parseNodeID(c)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}
