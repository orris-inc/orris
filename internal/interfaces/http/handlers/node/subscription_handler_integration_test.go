package node

// COMMENTED: All imports below were only used by Mock tests
// Since all Mock tests are commented out, these imports are no longer needed
/*
import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

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
type mockGenerateSubscriptionUC struct {
	executeFunc func(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error)
}

func (m *mockGenerateSubscriptionUC) Execute(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}

	switch cmd.Format {
	case "base64":
		content := "ss://YWVzLTI1Ni1nY206dGVzdHBhc3N3b3JkQDE5Mi4xNjguMS4xOjgzODg=#TestNode"
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		return &usecases.GenerateSubscriptionResult{
			Content:     encoded,
			ContentType: "text/plain; charset=utf-8",
			Format:      "base64",
		}, nil
	case "clash":
		yamlContent := `proxies:
  - name: TestNode
    type: ss
    server: 192.168.1.1
    port: 8388
    cipher: aes-256-gcm
    password: testpassword`
		return &usecases.GenerateSubscriptionResult{
			Content:     yamlContent,
			ContentType: "application/yaml; charset=utf-8",
			Format:      "clash",
		}, nil
	case "v2ray":
		v2rayConfig := map[string]interface{}{
			"outbounds": []interface{}{
				map[string]interface{}{
					"tag":      "proxy",
					"protocol": "shadowsocks",
					"settings": map[string]interface{}{
						"servers": []interface{}{
							map[string]interface{}{
								"address":  "192.168.1.1",
								"port":     8388,
								"method":   "aes-256-gcm",
								"password": "testpassword",
							},
						},
					},
				},
			},
		}
		jsonBytes, _ := json.Marshal(v2rayConfig)
		return &usecases.GenerateSubscriptionResult{
			Content:     string(jsonBytes),
			ContentType: "application/json; charset=utf-8",
			Format:      "v2ray",
		}, nil
	case "sip008":
		sip008Config := map[string]interface{}{
			"version": 1,
			"servers": []interface{}{
				map[string]interface{}{
					"server":      "192.168.1.1",
					"server_port": 8388,
					"method":      "aes-256-gcm",
					"password":    "testpassword",
					"remarks":     "TestNode",
				},
			},
		}
		jsonBytes, _ := json.Marshal(sip008Config)
		return &usecases.GenerateSubscriptionResult{
			Content:     string(jsonBytes),
			ContentType: "application/json; charset=utf-8",
			Format:      "sip008",
		}, nil
	case "surge":
		surgeContent := `[Proxy]
TestNode = ss, 192.168.1.1, 8388, encrypt-method=aes-256-gcm, password=testpassword`
		return &usecases.GenerateSubscriptionResult{
			Content:     surgeContent,
			ContentType: "text/plain; charset=utf-8",
			Format:      "surge",
		}, nil
	default:
		return nil, errors.NewValidationError("Invalid subscription format")
	}
}
*/

// ============================================================================
// Helper Functions - COMMENTED OUT
// ============================================================================
// REASON: Uses mockGenerateSubscriptionUC - violates CLAUDE.md rule
/*
func setupSubscriptionTestRouter() (*gin.Engine, *SubscriptionHandler) {
	gin.SetMode(gin.TestMode)

	generateUC := &mockGenerateSubscriptionUC{}
	handler := NewSubscriptionHandler(generateUC)

	router := gin.New()
	router.GET("/sub/:token", handler.GetSubscription)
	router.GET("/sub/:token/clash", handler.GetClashSubscription)
	router.GET("/sub/:token/v2ray", handler.GetV2RaySubscription)
	router.GET("/sub/:token/sip008", handler.GetSIP008Subscription)
	router.GET("/sub/:token/surge", handler.GetSurgeSubscription)

	return router, handler
}
*/

// ============================================================================
// Tests for GetSubscription (Base64) - COMMENTED OUT
// ============================================================================
// REASON: Uses setupSubscriptionTestRouter which depends on mockGenerateSubscriptionUC - violates CLAUDE.md rule
/*
func TestGetSubscription_Base64_Success(t *testing.T) {
	router, _ := setupSubscriptionTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/sub/test-token-abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Subscription-Userinfo"), "upload=0")
	assert.Contains(t, w.Header().Get("Subscription-Userinfo"), "download=0")

	decoded, err := base64.StdEncoding.DecodeString(w.Body.String())
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "ss://")
}
*/

// ============================================================================
// Tests for GetSubscription (EmptyToken) - COMMENTED OUT
// ============================================================================
// REASON: Uses setupSubscriptionTestRouter which depends on mockGenerateSubscriptionUC - violates CLAUDE.md rule
/*
func TestGetSubscription_EmptyToken(t *testing.T) {
	router, _ := setupSubscriptionTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/sub/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
*/

// ============================================================================
// Tests for GetSubscription (InvalidToken) - COMMENTED OUT
// ============================================================================
// REASON: Uses mockGenerateSubscriptionUC directly - violates CLAUDE.md rule
/*
func TestGetSubscription_InvalidToken(t *testing.T) {
	mockUC := &mockGenerateSubscriptionUC{
		executeFunc: func(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error) {
			return nil, errors.NewNotFoundError("Subscription not found")
		},
	}

	handler := NewSubscriptionHandler(mockUC)
	router := gin.New()
	router.GET("/sub/:token", handler.GetSubscription)

	req := httptest.NewRequest(http.MethodGet, "/sub/invalid-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])
}
*/

// ============================================================================
// Tests for GetClashSubscription (Success) - COMMENTED OUT
// ============================================================================
// REASON: Uses setupSubscriptionTestRouter which depends on mockGenerateSubscriptionUC - violates CLAUDE.md rule
/*
func TestGetClashSubscription_Success(t *testing.T) {
	router, _ := setupSubscriptionTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/sub/test-token-abc123/clash", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/yaml; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "attachment; filename=clash.yaml", w.Header().Get("Content-Disposition"))

	var clashConfig map[string]interface{}
	err := yaml.Unmarshal(w.Body.Bytes(), &clashConfig)
	require.NoError(t, err)

	assert.Contains(t, clashConfig, "proxies")
	proxies := clashConfig["proxies"].([]interface{})
	assert.NotEmpty(t, proxies)
}
*/

// ============================================================================
// Tests for GetClashSubscription (InvalidFormat) - COMMENTED OUT
// ============================================================================
// REASON: Uses mockGenerateSubscriptionUC directly - violates CLAUDE.md rule
/*
func TestGetClashSubscription_InvalidFormat(t *testing.T) {
	mockUC := &mockGenerateSubscriptionUC{
		executeFunc: func(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error) {
			return nil, errors.NewValidationError("Invalid format")
		},
	}

	handler := NewSubscriptionHandler(mockUC)
	router := gin.New()
	router.GET("/sub/:token/clash", handler.GetClashSubscription)

	req := httptest.NewRequest(http.MethodGet, "/sub/test-token/clash", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
*/

// ============================================================================
// Tests for GetV2RaySubscription (Success) - COMMENTED OUT
// ============================================================================
// REASON: Uses setupSubscriptionTestRouter which depends on mockGenerateSubscriptionUC - violates CLAUDE.md rule
/*
func TestGetV2RaySubscription_Success(t *testing.T) {
	router, _ := setupSubscriptionTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/sub/test-token-abc123/v2ray", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var v2rayConfig map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &v2rayConfig)
	require.NoError(t, err)

	assert.Contains(t, v2rayConfig, "outbounds")
	outbounds := v2rayConfig["outbounds"].([]interface{})
	assert.NotEmpty(t, outbounds)
}
*/

// ============================================================================
// Tests for GetSIP008Subscription (Success) - COMMENTED OUT
// ============================================================================
// REASON: Uses setupSubscriptionTestRouter which depends on mockGenerateSubscriptionUC - violates CLAUDE.md rule
/*
func TestGetSIP008Subscription_Success(t *testing.T) {
	router, _ := setupSubscriptionTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/sub/test-token-abc123/sip008", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var sip008Config map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &sip008Config)
	require.NoError(t, err)

	assert.Equal(t, float64(1), sip008Config["version"])
	assert.Contains(t, sip008Config, "servers")
	servers := sip008Config["servers"].([]interface{})
	assert.NotEmpty(t, servers)

	server := servers[0].(map[string]interface{})
	assert.Equal(t, "192.168.1.1", server["server"])
	assert.Equal(t, float64(8388), server["server_port"])
	assert.Equal(t, "aes-256-gcm", server["method"])
}
*/

// ============================================================================
// Tests for GetSurgeSubscription (Success) - COMMENTED OUT
// ============================================================================
// REASON: Uses setupSubscriptionTestRouter which depends on mockGenerateSubscriptionUC - violates CLAUDE.md rule
/*
func TestGetSurgeSubscription_Success(t *testing.T) {
	router, _ := setupSubscriptionTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/sub/test-token-abc123/surge", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "[Proxy]")
	assert.Contains(t, body, "ss")
	assert.Contains(t, body, "192.168.1.1")
}
*/

// ============================================================================
// Tests for AllSubscriptionFormats_TokenValidation - COMMENTED OUT
// ============================================================================
// REASON: Uses mockGenerateSubscriptionUC directly - violates CLAUDE.md rule
/*
func TestAllSubscriptionFormats_TokenValidation(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		format   string
	}{
		{"base64", "/sub/%s", "base64"},
		{"clash", "/sub/%s/clash", "clash"},
		{"v2ray", "/sub/%s/v2ray", "v2ray"},
		{"sip008", "/sub/%s/sip008", "sip008"},
		{"surge", "/sub/%s/surge", "surge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockGenerateSubscriptionUC{
				executeFunc: func(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error) {
					assert.Equal(t, "valid-token-123", cmd.SubscriptionToken)
					assert.Equal(t, tt.format, cmd.Format)

					return &usecases.GenerateSubscriptionResult{
						Content:     "test-content",
						ContentType: "text/plain",
						Format:      tt.format,
					}, nil
				},
			}

			handler := NewSubscriptionHandler(mockUC)
			router := gin.New()

			switch tt.format {
			case "base64":
				router.GET("/sub/:token", handler.GetSubscription)
			case "clash":
				router.GET("/sub/:token/clash", handler.GetClashSubscription)
			case "v2ray":
				router.GET("/sub/:token/v2ray", handler.GetV2RaySubscription)
			case "sip008":
				router.GET("/sub/:token/sip008", handler.GetSIP008Subscription)
			case "surge":
				router.GET("/sub/:token/surge", handler.GetSurgeSubscription)
			}

			url := fmt.Sprintf(tt.endpoint, "valid-token-123")
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
*/

// ============================================================================
// Tests for Subscription_RateLimiting - COMMENTED OUT
// ============================================================================
// REASON: Uses mockGenerateSubscriptionUC directly - violates CLAUDE.md rule
/*
func TestSubscription_RateLimiting(t *testing.T) {
	callCount := 0
	mockUC := &mockGenerateSubscriptionUC{
		executeFunc: func(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error) {
			callCount++
			if callCount > 10 {
				return nil, errors.NewBadRequestError("Rate limit exceeded")
			}
			return &usecases.GenerateSubscriptionResult{
				Content:     "test",
				ContentType: "text/plain",
				Format:      "base64",
			}, nil
		},
	}

	handler := NewSubscriptionHandler(mockUC)
	testRouter := gin.New()
	testRouter.GET("/sub/:token", handler.GetSubscription)

	for i := 0; i < 15; i++ {
		req := httptest.NewRequest(http.MethodGet, "/sub/test-token", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		if i < 10 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code)
		}
	}
}
*/

// ============================================================================
// Tests for Subscription_SignatureValidation - COMMENTED OUT
// ============================================================================
// REASON: Uses mockGenerateSubscriptionUC directly - violates CLAUDE.md rule
/*
func TestSubscription_SignatureValidation(t *testing.T) {
	mockUC := &mockGenerateSubscriptionUC{
		executeFunc: func(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error) {
			if cmd.SubscriptionToken == "signed-token-with-valid-signature" {
				return &usecases.GenerateSubscriptionResult{
					Content:     "valid-content",
					ContentType: "text/plain",
					Format:      "base64",
				}, nil
			}
			return nil, errors.NewUnauthorizedError("Invalid signature")
		},
	}

	handler := NewSubscriptionHandler(mockUC)
	router := gin.New()
	router.GET("/sub/:token", handler.GetSubscription)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "valid signature",
			token:          "signed-token-with-valid-signature",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid signature",
			token:          "tampered-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/sub/"+tt.token, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
*/

// ============================================================================
// Tests for Subscription_UserInfoHeader - COMMENTED OUT
// ============================================================================
// REASON: Uses mockGenerateSubscriptionUC directly - violates CLAUDE.md rule
/*
func TestSubscription_UserInfoHeader(t *testing.T) {
	mockUC := &mockGenerateSubscriptionUC{
		executeFunc: func(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error) {
			return &usecases.GenerateSubscriptionResult{
				Content:     "test-content",
				ContentType: "text/plain",
				Format:      "base64",
			}, nil
		},
	}

	handler := NewSubscriptionHandler(mockUC)
	router := gin.New()
	router.GET("/sub/:token", handler.GetSubscription)

	req := httptest.NewRequest(http.MethodGet, "/sub/test-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	userInfo := w.Header().Get("Subscription-Userinfo")
	assert.NotEmpty(t, userInfo)
	assert.Contains(t, userInfo, "upload=")
	assert.Contains(t, userInfo, "download=")
	assert.Contains(t, userInfo, "total=")
	assert.Contains(t, userInfo, "expire=")
}
*/

// ============================================================================
// Tests for Subscription_ContentTypeHeaders - COMMENTED OUT
// ============================================================================
// REASON: Uses setupSubscriptionTestRouter which depends on mockGenerateSubscriptionUC - violates CLAUDE.md rule
/*
func TestSubscription_ContentTypeHeaders(t *testing.T) {
	tests := []struct {
		name                string
		endpoint            string
		expectedContentType string
	}{
		{
			name:                "base64 content type",
			endpoint:            "/sub/test-token",
			expectedContentType: "text/plain; charset=utf-8",
		},
		{
			name:                "clash content type",
			endpoint:            "/sub/test-token/clash",
			expectedContentType: "application/yaml; charset=utf-8",
		},
		{
			name:                "v2ray content type",
			endpoint:            "/sub/test-token/v2ray",
			expectedContentType: "application/json; charset=utf-8",
		},
		{
			name:                "sip008 content type",
			endpoint:            "/sub/test-token/sip008",
			expectedContentType: "application/json; charset=utf-8",
		},
		{
			name:                "surge content type",
			endpoint:            "/sub/test-token/surge",
			expectedContentType: "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := setupSubscriptionTestRouter()

			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedContentType, w.Header().Get("Content-Type"))
		})
	}
}
*/

// ============================================================================
// Tests for Subscription_ErrorHandling - COMMENTED OUT
// ============================================================================
// REASON: Uses mockGenerateSubscriptionUC directly - violates CLAUDE.md rule
/*
func TestSubscription_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		mockError      error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "not found error",
			mockError:      errors.NewNotFoundError("Subscription not found"),
			expectedStatus: http.StatusNotFound,
			expectedError:  "not_found",
		},
		{
			name:           "unauthorized error",
			mockError:      errors.NewUnauthorizedError("Invalid token"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "unauthorized",
		},
		{
			name:           "validation error",
			mockError:      errors.NewValidationError("Invalid format"),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:           "bad request error",
			mockError:      errors.NewBadRequestError("Too many requests"),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockGenerateSubscriptionUC{
				executeFunc: func(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error) {
					return nil, tt.mockError
				},
			}

			handler := NewSubscriptionHandler(mockUC)
			router := gin.New()
			router.GET("/sub/:token", handler.GetSubscription)

			req := httptest.NewRequest(http.MethodGet, "/sub/test-token", nil)
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
