package forward

import (
	"bytes"
	"context"
	"encoding/json"
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

// GetRules retrieves all enabled forward rules.
func (c *Client) GetRules(ctx context.Context) ([]Rule, error) {
	url := fmt.Sprintf("%s/forward-agent-api/rules", c.baseURL)

	var rules []Rule
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &rules); err != nil {
		return nil, fmt.Errorf("get rules: %w", err)
	}
	return rules, nil
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
func (c *Client) GetExitEndpoint(ctx context.Context, exitAgentID uint) (*ExitEndpoint, error) {
	url := fmt.Sprintf("%s/forward-agent-api/exit-endpoint/%d", c.baseURL, exitAgentID)

	var endpoint ExitEndpoint
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &endpoint); err != nil {
		return nil, fmt.Errorf("get exit endpoint: %w", err)
	}
	return &endpoint, nil
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
