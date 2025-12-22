package agent

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

// ErrUnauthorized is returned when the API returns a 401 status code.
// This typically means the node token is invalid or expired.
var ErrUnauthorized = errors.New("unauthorized: invalid or expired node token")

// Client is the Agent API client.
type Client struct {
	baseURL    string
	token      string
	nodeSID    string
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

// NewClient creates a new Agent API client.
//
// Parameters:
//   - baseURL: The API base URL (e.g., "https://api.example.com")
//   - token: The node authentication token
//   - nodeSID: The node SID assigned by the server (Stripe-style: node_xxx)
func NewClient(baseURL, token string, nodeSID string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		token:   token,
		nodeSID: nodeSID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetConfig retrieves the node configuration.
func (c *Client) GetConfig(ctx context.Context) (*NodeConfig, error) {
	url := fmt.Sprintf("%s/agents/%s/config", c.baseURL, c.nodeSID)

	var config NodeConfig
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &config); err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	return &config, nil
}

// GetSubscriptions retrieves the list of active subscriptions for this node.
func (c *Client) GetSubscriptions(ctx context.Context) ([]Subscription, error) {
	url := fmt.Sprintf("%s/agents/%s/subscriptions", c.baseURL, c.nodeSID)

	var subscriptions []Subscription
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &subscriptions); err != nil {
		return nil, fmt.Errorf("get subscriptions: %w", err)
	}
	return subscriptions, nil
}

// ReportTraffic reports subscription traffic data.
func (c *Client) ReportTraffic(ctx context.Context, reports []TrafficReport) (*TrafficReportResult, error) {
	url := fmt.Sprintf("%s/agents/%s/traffic", c.baseURL, c.nodeSID)

	var result TrafficReportResult
	if err := c.doRequest(ctx, http.MethodPost, url, reports, &result); err != nil {
		return nil, fmt.Errorf("report traffic: %w", err)
	}
	return &result, nil
}

// UpdateStatus updates the node system status.
func (c *Client) UpdateStatus(ctx context.Context, status *NodeStatus) error {
	url := fmt.Sprintf("%s/agents/%s/status", c.baseURL, c.nodeSID)

	if err := c.doRequest(ctx, http.MethodPut, url, status, nil); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// UpdateOnlineSubscriptions updates the list of online subscriptions.
func (c *Client) UpdateOnlineSubscriptions(ctx context.Context, subscriptions []OnlineSubscription) (*OnlineSubscriptionsResult, error) {
	url := fmt.Sprintf("%s/agents/%s/online-subscriptions", c.baseURL, c.nodeSID)

	body := map[string]any{
		"subscriptions": subscriptions,
	}

	var result OnlineSubscriptionsResult
	if err := c.doRequest(ctx, http.MethodPut, url, body, &result); err != nil {
		return nil, fmt.Errorf("update online subscriptions: %w", err)
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

	req.Header.Set("X-Node-Token", c.token)
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
		return ErrUnauthorized
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

	// Re-marshal and unmarshal to convert Data to the target type
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, result); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}

	return nil
}
