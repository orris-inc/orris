package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// NewTestContext creates a test gin.Context with the given method, path, and optional body.
func NewTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()

	var req *http.Request
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	return c, w
}

// SetAuthContext sets user_id and session_id in gin context (simulating auth middleware).
func SetAuthContext(c *gin.Context, userID uint) {
	c.Set("user_id", userID)
	c.Set("session_id", "test-session-id")
}

// SetURLParam sets a URL parameter on the gin context.
func SetURLParam(c *gin.Context, key, value string) {
	c.Params = append(c.Params, gin.Param{Key: key, Value: value})
}

// SetQueryParams sets query parameters on the gin context.
func SetQueryParams(c *gin.Context, params map[string]string) {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	c.Request.URL.RawQuery = q.Encode()
}

// SetSubscriptionContext sets subscription_id in gin context (simulating ownership middleware).
func SetSubscriptionContext(c *gin.Context, subscriptionID uint) {
	c.Set("subscription_id", subscriptionID)
}

// ParseResponse parses the JSON response body into the target struct.
func ParseResponse(w *httptest.ResponseRecorder, target interface{}) error {
	return json.Unmarshal(w.Body.Bytes(), target)
}

// APIResponse mirrors utils.APIResponse for test assertions.
type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *ErrorInfo      `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
}

// ErrorInfo mirrors utils.ErrorInfo for test assertions.
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewMockLogger returns a no-op logger.Interface for tests.
func NewMockLogger() logger.Interface {
	return &mockLogger{}
}

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any)                    {}
func (m *mockLogger) Info(msg string, args ...any)                     {}
func (m *mockLogger) Warn(msg string, args ...any)                     {}
func (m *mockLogger) Error(msg string, args ...any)                    {}
func (m *mockLogger) Fatal(msg string, args ...any)                    {}
func (m *mockLogger) With(args ...any) logger.Interface                { return m }
func (m *mockLogger) Named(name string) logger.Interface               { return m }
func (m *mockLogger) Debugw(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Infow(msg string, keysAndValues ...interface{})   {}
func (m *mockLogger) Warnw(msg string, keysAndValues ...interface{})   {}
func (m *mockLogger) Errorw(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Fatalw(msg string, keysAndValues ...interface{})  {}
