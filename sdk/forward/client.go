package forward

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// GetRules retrieves all enabled forward rules along with the token signing secret.
func (c *Client) GetRules(ctx context.Context) (*RulesResponse, error) {
	url := fmt.Sprintf("%s/forward-agent-api/rules", c.baseURL)

	var resp RulesResponse
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &resp); err != nil {
		return nil, fmt.Errorf("get rules: %w", err)
	}
	return &resp, nil
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

const (
	agentTokenPrefix = "fwd"
)

// AgentTokenVerifyResult represents the result of verifying an agent token.
type AgentTokenVerifyResult struct {
	AgentShortID string // The short ID of the agent (without prefix)
}

// VerifyAgentToken verifies an agent token locally using the signing secret.
// This should be called by exit agents before accepting a tunnel connection.
// Returns the agent short ID if the token is valid.
func VerifyAgentToken(token, signingSecret string) (*AgentTokenVerifyResult, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}
	if signingSecret == "" {
		return nil, errors.New("signing secret cannot be empty")
	}

	// Parse token: fwd_<short_id>_<signature>
	parts := strings.Split(token, "_")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	prefix := parts[0]
	shortID := parts[1]
	providedSig := parts[2]

	if prefix != agentTokenPrefix {
		return nil, errors.New("invalid token prefix")
	}

	if shortID == "" {
		return nil, errors.New("invalid short ID in token")
	}

	// Compute expected signature and compare
	expectedSig := computeAgentTokenSignature(shortID, signingSecret)
	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return nil, errors.New("invalid token signature")
	}

	return &AgentTokenVerifyResult{
		AgentShortID: shortID,
	}, nil
}

// computeAgentTokenSignature computes the HMAC signature for an agent token.
func computeAgentTokenSignature(shortID, secret string) string {
	data := fmt.Sprintf("%s_%s", agentTokenPrefix, shortID)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	sig := h.Sum(nil)
	// Truncate to 16 bytes and encode
	return base64.RawURLEncoding.EncodeToString(sig[:16])
}

// VerifyTunnelHandshake verifies a tunnel handshake from an entry agent.
// It checks the token signature and whether the agent is allowed to access the specified rule.
//
// Parameters:
//   - handshake: The handshake message from entry agent
//   - signingSecret: The token signing secret from GetRules response
//   - rules: The list of rules this agent has (from GetRules response)
//
// Returns:
//   - result: Contains success status and entry agent ID if verified
//   - error: Non-nil if verification failed
func VerifyTunnelHandshake(handshake *TunnelHandshake, signingSecret string, rules []Rule) (*TunnelHandshakeResult, error) {
	if handshake == nil {
		return &TunnelHandshakeResult{Success: false, Error: "handshake is nil"}, errors.New("handshake is nil")
	}

	// Verify the agent token
	tokenResult, err := VerifyAgentToken(handshake.AgentToken, signingSecret)
	if err != nil {
		return &TunnelHandshakeResult{Success: false, Error: "invalid token"}, fmt.Errorf("verify token: %w", err)
	}

	// Format the full agent ID (with prefix)
	entryAgentID := fmt.Sprintf("fa_%s", tokenResult.AgentShortID)

	// Check if this agent is allowed to access the rule
	allowed := false
	for _, rule := range rules {
		if rule.ID != handshake.RuleID {
			continue
		}

		// For exit role: check if the entry agent matches the rule's agent
		if rule.Role == "exit" {
			if rule.AgentID == entryAgentID {
				allowed = true
				break
			}
		}

		// For chain relay/exit: check if entry agent is the previous hop
		// The entry agent should be at position (current position - 1) in the chain
		if (rule.RuleType == RuleTypeChain || rule.RuleType == RuleTypeDirectChain) && rule.ChainPosition > 0 {
			// Check if entry agent is in the chain at the previous position
			if rule.ChainPosition-1 < len(rule.ChainAgentIDs) {
				prevAgentID := rule.ChainAgentIDs[rule.ChainPosition-1]
				if prevAgentID == entryAgentID {
					allowed = true
					break
				}
			}
		}
	}

	if !allowed {
		return &TunnelHandshakeResult{Success: false, Error: "access denied"}, errors.New("agent not allowed to access this rule")
	}

	return &TunnelHandshakeResult{
		Success:      true,
		EntryAgentID: entryAgentID,
	}, nil
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
