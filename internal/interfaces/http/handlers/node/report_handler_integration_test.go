package node

// COMMENTED: All imports below were only used by Mock tests
// Since all Mock tests are commented out, these imports are no longer needed
/*
import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"context"
	"orris/internal/application/node/usecases"
	"orris/internal/shared/errors"
)
*/

// ============================================================================
// Mock Objects - COMMENTED OUT
// ============================================================================
// REASON: Violates CLAUDE.md rule - "不允许mock数据"
// TODO: Refactor these tests to use real dependencies
// ============================================================================
/*
type mockReportNodeDataUC struct {
	executeFunc func(ctx context.Context, cmd usecases.ReportNodeDataCommand) (*usecases.ReportNodeDataResult, error)
}

func (m *mockReportNodeDataUC) Execute(ctx context.Context, cmd usecases.ReportNodeDataCommand) (*usecases.ReportNodeDataResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.ReportNodeDataResult{
		ShouldReload:     false,
		ConfigVersion:    1,
		TrafficExceeded:  false,
		TrafficRemaining: 0,
	}, nil
}

type mockValidateNodeTokenUC struct {
	executeFunc func(ctx context.Context, cmd usecases.ValidateNodeTokenCommand) (*usecases.ValidateNodeTokenResult, error)
}

func (m *mockValidateNodeTokenUC) Execute(ctx context.Context, cmd usecases.ValidateNodeTokenCommand) (*usecases.ValidateNodeTokenResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}

	if cmd.PlainToken == "valid-token-123" {
		return &usecases.ValidateNodeTokenResult{
			NodeID: 1,
			Name:   "Test Node",
		}, nil
	}

	return nil, errors.NewUnauthorizedError("Invalid token")
}

func setupReportTestRouter() (*gin.Engine, *ReportHandler) {
	gin.SetMode(gin.TestMode)

	reportUC := &mockReportNodeDataUC{}
	validateUC := &mockValidateNodeTokenUC{}

	handler := NewReportHandler(reportUC, validateUC)

	router := gin.New()
	router.POST("/nodes/report", handler.NodeTokenMiddleware(), handler.ReportNodeData)

	return router, handler
}
*/

// ============================================================================
// Tests for TestReportNodeData_Success - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_Success(t *testing.T) {
	router, _ := setupReportTestRouter()

	reqBody := ReportNodeDataRequest{
		Upload:      1024000,
		Download:    2048000,
		OnlineUsers: 5,
		Status:      "active",
		SystemInfo: &SystemInfo{
			Load:        0.75,
			MemoryUsage: 60.5,
			DiskUsage:   45.2,
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Data reported successfully", response["message"])
}
*/

// ============================================================================
// Tests for TestReportNodeData_WithoutSystemInfo - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_WithoutSystemInfo(t *testing.T) {
	router, _ := setupReportTestRouter()

	reqBody := ReportNodeDataRequest{
		Upload:      1024000,
		Download:    2048000,
		OnlineUsers: 3,
		Status:      "active",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token-123")

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
// Tests for TestReportNodeData_MissingAuthHeader - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_MissingAuthHeader(t *testing.T) {
	router, _ := setupReportTestRouter()

	reqBody := ReportNodeDataRequest{
		Upload:   1024000,
		Download: 2048000,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorInfo := response["error"].(map[string]interface{})
	assert.Equal(t, "unauthorized", errorInfo["type"])
	assert.Equal(t, "Authorization header required", errorInfo["message"])
}
*/

// ============================================================================
// Tests for TestReportNodeData_InvalidAuthHeaderFormat - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_InvalidAuthHeaderFormat(t *testing.T) {
	router, _ := setupReportTestRouter()

	tests := []struct {
		name      string
		authValue string
	}{
		{
			name:      "missing Bearer prefix",
			authValue: "invalid-token-123",
		},
		{
			name:      "empty token after Bearer",
			authValue: "Bearer ",
		},
		{
			name:      "only Bearer",
			authValue: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := ReportNodeDataRequest{
				Upload:   1024000,
				Download: 2048000,
			}

			bodyBytes, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", tt.authValue)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
		})
	}
}
*/

// ============================================================================
// Tests for TestReportNodeData_InvalidToken - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_InvalidToken(t *testing.T) {
	router, _ := setupReportTestRouter()

	reqBody := ReportNodeDataRequest{
		Upload:   1024000,
		Download: 2048000,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid-token-999")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorInfo := response["error"].(map[string]interface{})
	assert.Equal(t, "Invalid or expired token", errorInfo["message"])
}
*/

// ============================================================================
// Tests for TestReportNodeData_InvalidRequestBody - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_InvalidRequestBody(t *testing.T) {
	router, _ := setupReportTestRouter()

	tests := []struct {
		name     string
		reqBody  interface{}
		wantCode int
	}{
		{
			name:     "invalid json",
			reqBody:  "invalid-json",
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "missing upload field",
			reqBody: map[string]interface{}{
				"download": 2048000,
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "missing download field",
			reqBody: map[string]interface{}{
				"upload": 1024000,
			},
			wantCode: http.StatusInternalServerError,
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

			req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer valid-token-123")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}
*/

// ============================================================================
// Tests for TestReportNodeData_TrafficRecording - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_TrafficRecording(t *testing.T) {
	capturedCmd := &usecases.ReportNodeDataCommand{}

	mockReportUC := &mockReportNodeDataUC{
		executeFunc: func(ctx context.Context, cmd usecases.ReportNodeDataCommand) (*usecases.ReportNodeDataResult, error) {
			*capturedCmd = cmd
			return &usecases.ReportNodeDataResult{
				ShouldReload:     false,
				ConfigVersion:    1,
				TrafficExceeded:  false,
				TrafficRemaining: 0,
			}, nil
		},
	}

	mockValidateUC := &mockValidateNodeTokenUC{
		executeFunc: func(ctx context.Context, cmd usecases.ValidateNodeTokenCommand) (*usecases.ValidateNodeTokenResult, error) {
			return &usecases.ValidateNodeTokenResult{
				NodeID: 123,
				Name:   "Test Node",
			}, nil
		},
	}

	handler := NewReportHandler(mockReportUC, mockValidateUC)
	router := gin.New()
	router.POST("/nodes/report", handler.NodeTokenMiddleware(), handler.ReportNodeData)

	reqBody := ReportNodeDataRequest{
		Upload:      5000000,
		Download:    10000000,
		OnlineUsers: 7,
		Status:      "active",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, uint(123), capturedCmd.NodeID)
	assert.Equal(t, uint64(5000000), capturedCmd.Upload)
	assert.Equal(t, uint64(10000000), capturedCmd.Download)
	assert.Equal(t, 7, capturedCmd.OnlineUsers)
	assert.Equal(t, "active", capturedCmd.Status)
}
*/

// ============================================================================
// Tests for TestReportNodeData_SystemInfoRecording - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_SystemInfoRecording(t *testing.T) {
	capturedCmd := &usecases.ReportNodeDataCommand{}

	mockReportUC := &mockReportNodeDataUC{
		executeFunc: func(ctx context.Context, cmd usecases.ReportNodeDataCommand) (*usecases.ReportNodeDataResult, error) {
			*capturedCmd = cmd
			return &usecases.ReportNodeDataResult{
				ShouldReload:     false,
				ConfigVersion:    1,
				TrafficExceeded:  false,
				TrafficRemaining: 0,
			}, nil
		},
	}

	mockValidateUC := &mockValidateNodeTokenUC{}

	handler := NewReportHandler(mockReportUC, mockValidateUC)
	router := gin.New()
	router.POST("/nodes/report", handler.NodeTokenMiddleware(), handler.ReportNodeData)

	reqBody := ReportNodeDataRequest{
		Upload:   1000000,
		Download: 2000000,
		SystemInfo: &SystemInfo{
			Load:        1.23,
			MemoryUsage: 78.9,
			DiskUsage:   56.7,
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedCmd.SystemInfo)
	assert.Equal(t, 1.23, capturedCmd.SystemInfo.Load)
	assert.Equal(t, 78.9, capturedCmd.SystemInfo.MemoryUsage)
	assert.Equal(t, 56.7, capturedCmd.SystemInfo.DiskUsage)
}
*/

// ============================================================================
// Tests for TestNodeTokenMiddleware_IPAddressCapture - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestNodeTokenMiddleware_IPAddressCapture(t *testing.T) {
	capturedIP := ""

	mockValidateUC := &mockValidateNodeTokenUC{
		executeFunc: func(ctx context.Context, cmd usecases.ValidateNodeTokenCommand) (*usecases.ValidateNodeTokenResult, error) {
			capturedIP = cmd.IPAddress
			return &usecases.ValidateNodeTokenResult{
				NodeID: 1,
				Name:   "Test Node",
			}, nil
		},
	}

	mockReportUC := &mockReportNodeDataUC{}
	handler := NewReportHandler(mockReportUC, mockValidateUC)

	router := gin.New()
	router.POST("/nodes/report", handler.NodeTokenMiddleware(), handler.ReportNodeData)

	reqBody := ReportNodeDataRequest{
		Upload:   1000,
		Download: 2000,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token-123")
	req.RemoteAddr = "192.168.1.100:12345"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedIP)
}
*/

// ============================================================================
// Tests for TestNodeTokenMiddleware_ContextValues - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestNodeTokenMiddleware_ContextValues(t *testing.T) {
	mockValidateUC := &mockValidateNodeTokenUC{
		executeFunc: func(ctx context.Context, cmd usecases.ValidateNodeTokenCommand) (*usecases.ValidateNodeTokenResult, error) {
			return &usecases.ValidateNodeTokenResult{
				NodeID: 456,
				Name:   "Production Node",
			}, nil
		},
	}

	mockReportUC := &mockReportNodeDataUC{}
	handler := NewReportHandler(mockReportUC, mockValidateUC)

	router := gin.New()
	router.POST("/nodes/report", handler.NodeTokenMiddleware(), func(c *gin.Context) {
		nodeID, exists := c.Get("node_id")
		assert.True(t, exists)
		assert.Equal(t, uint(456), nodeID)

		nodeName, exists := c.Get("node_name")
		assert.True(t, exists)
		assert.Equal(t, "Production Node", nodeName)

		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	reqBody := ReportNodeDataRequest{
		Upload:   1000,
		Download: 2000,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
*/

// ============================================================================
// Tests for TestReportNodeData_ConcurrentRequests - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_ConcurrentRequests(t *testing.T) {
	router, _ := setupReportTestRouter()

	numRequests := 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			reqBody := ReportNodeDataRequest{
				Upload:   1024000,
				Download: 2048000,
			}

			bodyBytes, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer valid-token-123")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			results <- w.Code
		}()
	}

	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		assert.Equal(t, http.StatusOK, statusCode)
	}
}
*/

// ============================================================================
// Tests for TestReportNodeData_LargeTrafficValues - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_LargeTrafficValues(t *testing.T) {
	router, _ := setupReportTestRouter()

	reqBody := ReportNodeDataRequest{
		Upload:      18446744073709551615,
		Download:    18446744073709551615,
		OnlineUsers: 1000000,
		Status:      "active",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
*/

// ============================================================================
// Tests for TestReportNodeData_TimestampValidation - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_TimestampValidation(t *testing.T) {
	capturedTime := time.Time{}

	mockReportUC := &mockReportNodeDataUC{
		executeFunc: func(ctx context.Context, cmd usecases.ReportNodeDataCommand) (*usecases.ReportNodeDataResult, error) {
			capturedTime = cmd.Timestamp
			return &usecases.ReportNodeDataResult{
				ShouldReload:     false,
				ConfigVersion:    1,
				TrafficExceeded:  false,
				TrafficRemaining: 0,
			}, nil
		},
	}

	mockValidateUC := &mockValidateNodeTokenUC{}
	handler := NewReportHandler(mockReportUC, mockValidateUC)

	router := gin.New()
	router.POST("/nodes/report", handler.NodeTokenMiddleware(), handler.ReportNodeData)

	beforeRequest := time.Now()

	reqBody := ReportNodeDataRequest{
		Upload:   1000,
		Download: 2000,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	afterRequest := time.Now()

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, capturedTime.After(beforeRequest) || capturedTime.Equal(beforeRequest))
	assert.True(t, capturedTime.Before(afterRequest) || capturedTime.Equal(afterRequest))
}
*/

// ============================================================================
// Tests for TestNodeTokenMiddleware_AbortOnFailure - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestNodeTokenMiddleware_AbortOnFailure(t *testing.T) {
	mockValidateUC := &mockValidateNodeTokenUC{
		executeFunc: func(ctx context.Context, cmd usecases.ValidateNodeTokenCommand) (*usecases.ValidateNodeTokenResult, error) {
			return nil, errors.NewUnauthorizedError("Token expired")
		},
	}

	mockReportUC := &mockReportNodeDataUC{
		executeFunc: func(ctx context.Context, cmd usecases.ReportNodeDataCommand) (*usecases.ReportNodeDataResult, error) {
			t.Error("ReportNodeData should not be called when token validation fails")
			return nil, nil
		},
	}

	handler := NewReportHandler(mockReportUC, mockValidateUC)
	router := gin.New()
	router.POST("/nodes/report", handler.NodeTokenMiddleware(), handler.ReportNodeData)

	reqBody := ReportNodeDataRequest{
		Upload:   1000,
		Download: 2000,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer expired-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
*/

// ============================================================================
// Tests for TestReportNodeData_ErrorHandling - COMMENTED OUT
// ============================================================================
// REASON: Uses mockReportNodeDataUC and mockValidateNodeTokenUC - violates CLAUDE.md rule
/*
func TestReportNodeData_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		mockError      error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "internal server error",
			mockError:      errors.NewInternalError("Database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal_error",
		},
		{
			name:           "validation error",
			mockError:      errors.NewValidationError("Invalid traffic data"),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReportUC := &mockReportNodeDataUC{
				executeFunc: func(ctx context.Context, cmd usecases.ReportNodeDataCommand) (*usecases.ReportNodeDataResult, error) {
					return nil, tt.mockError
				},
			}

			mockValidateUC := &mockValidateNodeTokenUC{}
			handler := NewReportHandler(mockReportUC, mockValidateUC)

			router := gin.New()
			router.POST("/nodes/report", handler.NodeTokenMiddleware(), handler.ReportNodeData)

			reqBody := ReportNodeDataRequest{
				Upload:   1000,
				Download: 2000,
			}

			bodyBytes, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/nodes/report", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer valid-token-123")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
			errorInfo := response["error"].(map[string]interface{})
			assert.Equal(t, tt.expectedError, errorInfo["type"])
		})
	}
}
*/
