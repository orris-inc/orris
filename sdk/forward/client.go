package forward

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the Forward Agent API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Option is a function that configures the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(client *Client) {
		client.httpClient = c
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(client *Client) {
		client.httpClient.Timeout = d
	}
}

// NewClient creates a new Forward Agent API client.
//
// Parameters:
//   - baseURL: The API base URL (e.g., "https://api.example.com")
//   - token: The forward agent token (e.g., "fwd_xxx")
func NewClient(baseURL, token string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Token returns the agent token used by this client.
// This token should be used as AgentToken in TunnelHandshake when establishing
// tunnel connections to exit agents.
func (c *Client) Token() string {
	return c.token
}

// GetRules retrieves all enabled forward rules along with the token signing secret.
func (c *Client) GetRules(ctx context.Context) (*RulesResponse, error) {
	url := fmt.Sprintf("%s/forward-agent-api/rules", c.baseURL)

	var resp RulesResponse
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &resp); err != nil {
		return nil, fmt.Errorf("get rules: %w", err)
	}
	return &resp, nil
}

// RefreshRule retrieves the latest configuration for a specific rule.
// This is useful when connection to the next hop fails and the agent needs
// to get the latest ws_listen_port or other dynamic configuration.
// ruleID should be the Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ").
func (c *Client) RefreshRule(ctx context.Context, ruleID string) (*Rule, error) {
	url := fmt.Sprintf("%s/forward-agent-api/rules/%s", c.baseURL, ruleID)

	var rule Rule
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &rule); err != nil {
		return nil, fmt.Errorf("refresh rule: %w", err)
	}
	return &rule, nil
}

// ReportTraffic reports traffic data for forward rules.
func (c *Client) ReportTraffic(ctx context.Context, items []TrafficItem) (*TrafficReportResult, error) {
	url := fmt.Sprintf("%s/forward-agent-api/traffic", c.baseURL)

	body := map[string]any{
		"rules": items,
	}

	var result TrafficReportResult
	if err := c.doRequest(ctx, http.MethodPost, url, body, &result); err != nil {
		return nil, fmt.Errorf("report traffic: %w", err)
	}
	return &result, nil
}

// GetExitEndpoint retrieves the connection information for an exit agent.
// This is used by entry agents to establish WS tunnel connections to exit agents.
// exitAgentID should be the Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ").
func (c *Client) GetExitEndpoint(ctx context.Context, exitAgentID string) (*ExitEndpoint, error) {
	url := fmt.Sprintf("%s/forward-agent-api/exit-endpoint/%s", c.baseURL, exitAgentID)

	var endpoint ExitEndpoint
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &endpoint); err != nil {
		return nil, fmt.Errorf("get exit endpoint: %w", err)
	}
	return &endpoint, nil
}

// ReportStatus reports the agent status to the server.
func (c *Client) ReportStatus(ctx context.Context, status *AgentStatus) error {
	url := fmt.Sprintf("%s/forward-agent-api/status", c.baseURL)

	if err := c.doRequest(ctx, http.MethodPost, url, status, nil); err != nil {
		return fmt.Errorf("report status: %w", err)
	}
	return nil
}

// VerifyTunnelHandshakeViaServer verifies a tunnel handshake from an entry agent
// by calling the server API. This is the recommended way to verify tunnel handshakes
// as it does not require the signing secret to be distributed to agents.
//
// This method should be called by exit agents when they receive a tunnel connection
// request from an entry agent.
//
// Parameters:
//   - ctx: Context for the request
//   - handshake: The handshake message from entry agent
//
// Returns:
//   - result: Contains success status and entry agent ID if verified
//   - error: Non-nil if the API request failed
func (c *Client) VerifyTunnelHandshakeViaServer(ctx context.Context, handshake *TunnelHandshake) (*TunnelHandshakeResult, error) {
	if handshake == nil {
		return &TunnelHandshakeResult{Success: false, Error: "handshake is nil"}, errors.New("handshake is nil")
	}

	url := fmt.Sprintf("%s/forward-agent-api/verify-tunnel-handshake", c.baseURL)

	body := map[string]any{
		"agent_token": handshake.AgentToken,
		"rule_id":     handshake.RuleID,
	}

	var result TunnelHandshakeResult
	if err := c.doRequest(ctx, http.MethodPost, url, body, &result); err != nil {
		return &TunnelHandshakeResult{Success: false, Error: "verification failed"}, fmt.Errorf("verify tunnel handshake: %w", err)
	}

	return &result, nil
}

// doRequest performs an HTTP request and decodes the response.
func (c *Client) doRequest(ctx context.Context, method, url string, body any, result any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: invalid agent token")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	if result == nil {
		return nil
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("api error: %s", apiResp.Message)
	}

	if apiResp.Data == nil {
		return nil
	}

	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, result); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}

	return nil
}
