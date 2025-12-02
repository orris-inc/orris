# Forward Agent API Documentation

Forward Agent API for client-side port forwarding synchronization and traffic reporting.

## Base URL

```
/forward-agent-api
```

## Authentication

All endpoints require Agent token authentication. Token is obtained when creating a Forward Agent via admin API.

**Token Format**: `fwd_<base64_encoded_random_bytes>`

**Request Header**:
```
Authorization: Bearer fwd_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**Alternative**: Query parameter (not recommended for production)
```
GET /forward-agent-api/rules?token=fwd_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

---

## Getting Your Token

1. Admin creates a Forward Agent via `POST /forward-agents`
2. Response contains the token (only shown once)
3. Store the token securely in your client configuration

```json
{
  "success": true,
  "message": "forward agent created successfully",
  "data": {
    "id": 1,
    "name": "Production Agent",
    "token": "fwd_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "status": "enabled"
  }
}
```

> **Important**: The token is only returned once during creation. If lost, use `POST /forward-agents/:id/regenerate-token` to generate a new one.

---

## 1. Get Enabled Forward Rules

Retrieve all enabled forward rules for client configuration synchronization.

### Request

```
GET /forward-agent-api/rules
Authorization: Bearer fwd_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### Response

**Success (200)**

```json
{
  "success": true,
  "message": "enabled forward rules retrieved successfully",
  "data": [
    {
      "id": 1,
      "name": "MySQL-Forward",
      "listen_port": 13306,
      "target_address": "192.168.1.100",
      "target_port": 3306,
      "protocol": "tcp",
      "status": "enabled",
      "remark": "Forward to internal MySQL server",
      "upload_bytes": 1048576,
      "download_bytes": 2097152,
      "total_bytes": 3145728,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T12:00:00Z"
    }
  ]
}
```

**Unauthorized (401)**

```json
{
  "success": false,
  "message": "unauthorized",
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Missing or invalid agent token"
  }
}
```

**Error (500)**

```json
{
  "success": false,
  "message": "failed to retrieve enabled forward rules",
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Internal server error"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Rule ID |
| `name` | string | Rule name |
| `listen_port` | uint16 | Port to listen on locally |
| `target_address` | string | Target address (IP or domain) |
| `target_port` | uint16 | Target port to forward to |
| `protocol` | string | Protocol type: `tcp`, `udp`, `both` |
| `status` | string | Rule status: `enabled`, `disabled` |
| `remark` | string | Optional description |
| `upload_bytes` | int64 | Total uploaded bytes |
| `download_bytes` | int64 | Total downloaded bytes |
| `total_bytes` | int64 | Total traffic (upload + download) |
| `created_at` | string | Creation timestamp (ISO 8601) |
| `updated_at` | string | Last update timestamp (ISO 8601) |

---

## 2. Report Traffic

Submit forward rule traffic statistics from client.

### Request

```
POST /forward-agent-api/traffic
Authorization: Bearer fwd_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
Content-Type: application/json
```

**Request Body**

```json
{
  "rules": [
    {
      "rule_id": 1,
      "upload_bytes": 1024,
      "download_bytes": 2048
    },
    {
      "rule_id": 2,
      "upload_bytes": 512,
      "download_bytes": 1024
    }
  ]
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `rules` | array | Yes | Array of traffic items |
| `rules[].rule_id` | uint | Yes | Forward rule ID |
| `rules[].upload_bytes` | int64 | Yes | Uploaded bytes since last report (>= 0) |
| `rules[].download_bytes` | int64 | Yes | Downloaded bytes since last report (>= 0) |

### Response

**Success (200)**

```json
{
  "success": true,
  "message": "traffic reported successfully",
  "data": {
    "rules_updated": 2,
    "rules_failed": 0
  }
}
```

**Bad Request (400)**

```json
{
  "success": false,
  "message": "invalid request body",
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request body"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `rules_updated` | int | Number of rules successfully updated |
| `rules_failed` | int | Number of rules failed to update |

---

## Client Implementation Guide

### Recommended Workflow

```
1. Startup
   GET /forward-agent-api/rules
   -> Start forwarding for each enabled rule

2. Periodic Sync (every 30-60 seconds)
   GET /forward-agent-api/rules
   -> Compare with local rules
   -> Start new rules, stop removed rules

3. Traffic Reporting (every 60 seconds)
   POST /forward-agent-api/traffic
   -> Report accumulated traffic
   -> Reset local counters after successful report
```

### Configuration Example

```yaml
# config.yaml
server:
  base_url: "https://your-server.com"
  token: "fwd_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

sync:
  rule_interval: 30s    # Sync rules every 30 seconds
  traffic_interval: 60s # Report traffic every 60 seconds

forwarding:
  buffer_size: 32768    # 32KB buffer for TCP forwarding
  timeout: 30s          # Connection timeout
```

### Example: Go Client

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

const (
    baseURL = "https://your-server.com/forward-agent-api"
    token   = "fwd_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
)

type ForwardRule struct {
    ID            uint   `json:"id"`
    Name          string `json:"name"`
    ListenPort    uint16 `json:"listen_port"`
    TargetAddress string `json:"target_address"`
    TargetPort    uint16 `json:"target_port"`
    Protocol      string `json:"protocol"`
}

type RulesResponse struct {
    Success bool          `json:"success"`
    Message string        `json:"message"`
    Data    []ForwardRule `json:"data"`
}

type TrafficItem struct {
    RuleID        uint  `json:"rule_id"`
    UploadBytes   int64 `json:"upload_bytes"`
    DownloadBytes int64 `json:"download_bytes"`
}

type TrafficReport struct {
    Rules []TrafficItem `json:"rules"`
}

type TrafficResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Data    struct {
        RulesUpdated int `json:"rules_updated"`
        RulesFailed  int `json:"rules_failed"`
    } `json:"data"`
}

type ForwardAgent struct {
    client *http.Client
    token  string
}

func NewForwardAgent(token string) *ForwardAgent {
    return &ForwardAgent{
        client: &http.Client{Timeout: 10 * time.Second},
        token:  token,
    }
}

// GetRules fetches all enabled forward rules
func (a *ForwardAgent) GetRules() ([]ForwardRule, error) {
    req, err := http.NewRequest("GET", baseURL+"/rules", nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+a.token)

    resp, err := a.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusUnauthorized {
        return nil, fmt.Errorf("unauthorized: invalid agent token")
    }

    var result RulesResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    if !result.Success {
        return nil, fmt.Errorf("failed to get rules: %s", result.Message)
    }

    return result.Data, nil
}

// ReportTraffic reports traffic statistics
func (a *ForwardAgent) ReportTraffic(items []TrafficItem) error {
    report := TrafficReport{Rules: items}
    body, err := json.Marshal(report)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", baseURL+"/traffic", bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer "+a.token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := a.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusUnauthorized {
        return fmt.Errorf("unauthorized: invalid agent token")
    }

    var result TrafficResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

    if !result.Success {
        return fmt.Errorf("failed to report traffic: %s", result.Message)
    }

    return nil
}

func main() {
    agent := NewForwardAgent(token)

    // 1. Get enabled rules on startup
    rules, err := agent.GetRules()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Got %d rules\n", len(rules))

    // 2. Start forwarding for each rule...
    // (implement your TCP/UDP forwarding logic here)

    // 3. Periodically report traffic
    err = agent.ReportTraffic([]TrafficItem{
        {RuleID: 1, UploadBytes: 1024, DownloadBytes: 2048},
    })
    if err != nil {
        panic(err)
    }
}
```

---

## Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | VALIDATION_ERROR | Invalid request body or parameters |
| 401 | UNAUTHORIZED | Missing or invalid agent token |
| 500 | INTERNAL_ERROR | Server-side error |

---

## Notes

1. **Token Security**: Store the token securely, treat it like a password
2. **Traffic Reporting**: Report incremental traffic (bytes since last report), not cumulative totals
3. **Polling Interval**: Recommended 30-60 seconds for rule sync, 60 seconds for traffic report
4. **Graceful Shutdown**: Report final traffic before client shutdown
5. **Zero Traffic**: Rules with zero upload and download bytes are skipped automatically
6. **Agent Status**: If the agent is disabled, API requests will return 401 Unauthorized
