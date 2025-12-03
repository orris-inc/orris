# Node API Documentation

Node management API for proxy node configuration, node group management, and subscription generation.

## Base URL

```
/nodes          - Node management
/node-groups    - Node group management
/sub            - Subscription endpoints
```

## Authentication

### Admin API (Node & Node Group Management)

All management endpoints require JWT Bearer token authentication with admin role.

**Request Header**:
```
Authorization: Bearer <jwt_token>
```

### Subscription API

Subscription endpoints use token-based authentication via URL path parameter.

**Format**: `GET /sub/{subscription_uuid}`

---

## 1. Node Management

### 1.1 Create Node

Create a new proxy node.

**Request**

```
POST /nodes
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body**

```json
{
  "name": "US-Node-01",
  "server_address": "proxy.example.com",
  "server_port": 8388,
  "protocol": "shadowsocks",
  "encryption_method": "aes-256-gcm",
  "plugin": "obfs-local",
  "plugin_opts": {
    "obfs": "http",
    "obfs-host": "example.com"
  },
  "region": "us-west",
  "tags": ["premium", "fast"],
  "description": "High-speed US server",
  "sort_order": 1
}
```

**Request Fields**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Node display name (unique) |
| `server_address` | string | Yes | Server hostname or IP address |
| `server_port` | uint16 | Yes | Server port (1-65535) |
| `protocol` | string | Yes | Protocol type: `shadowsocks`, `trojan` |
| `encryption_method` | string | Yes | Encryption method |
| `plugin` | string | No | Plugin name (e.g., `obfs-local`, `v2ray-plugin`) |
| `plugin_opts` | object | No | Plugin configuration options |
| `region` | string | No | Geographic region identifier |
| `tags` | array | No | Custom tags for categorization |
| `description` | string | No | Node description |
| `sort_order` | int | No | Display order for sorting |

**Supported Encryption Methods**

| Protocol | Methods |
|----------|---------|
| shadowsocks | `aes-256-gcm`, `aes-128-gcm`, `chacha20-ietf-poly1305` |
| trojan | N/A (uses TLS) |

**Response**

**Success (201)**

```json
{
  "success": true,
  "message": "Node created successfully",
  "data": {
    "id": 1,
    "name": "US-Node-01",
    "server_address": "proxy.example.com",
    "server_port": 8388,
    "protocol": "shadowsocks",
    "encryption_method": "aes-256-gcm",
    "plugin": "obfs-local",
    "plugin_opts": {"obfs": "http", "obfs-host": "example.com"},
    "status": "inactive",
    "region": "us-west",
    "tags": ["premium", "fast"],
    "sort_order": 1,
    "is_available": false,
    "version": 1,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "api_token": "node_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
```

> **Important**: The `api_token` is only returned once during creation. Store it securely for node status reporting.

---

### 1.2 List Nodes

Get a paginated list of nodes.

**Request**

```
GET /nodes?page=1&page_size=20&status=active
Authorization: Bearer <jwt_token>
```

**Query Parameters**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `page_size` | int | 20 | Items per page (max: 100) |
| `status` | string | - | Filter: `active`, `inactive`, `maintenance` |
| `region` | string | - | Filter by region |
| `tags` | string | - | Filter by tags (comma-separated) |
| `order_by` | string | sort_order | Sort field |
| `order` | string | asc | Sort direction: `asc`, `desc` |

**Response**

**Success (200)**

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "US-Node-01",
        "server_address": "proxy.example.com",
        "server_port": 8388,
        "protocol": "shadowsocks",
        "encryption_method": "aes-256-gcm",
        "status": "active",
        "region": "us-west",
        "tags": ["premium", "fast"],
        "sort_order": 1,
        "is_available": true,
        "version": 1,
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T14:20:00Z",
        "system_status": {
          "cpu": "45.50",
          "memory": "65.30",
          "disk": "80.20",
          "uptime": 86400,
          "updated_at": 1705324800
        }
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20,
    "total_pages": 3
  }
}
```

---

### 1.3 Get Node

Get details of a specific node.

**Request**

```
GET /nodes/{id}
Authorization: Bearer <jwt_token>
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "US-Node-01",
    "server_address": "proxy.example.com",
    "server_port": 8388,
    "protocol": "shadowsocks",
    "encryption_method": "aes-256-gcm",
    "plugin": "obfs-local",
    "plugin_opts": {"obfs": "http"},
    "status": "active",
    "region": "us-west",
    "tags": ["premium", "fast"],
    "sort_order": 1,
    "is_available": true,
    "version": 1,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T14:20:00Z"
  }
}
```

**Not Found (404)**

```json
{
  "success": false,
  "message": "Node not found",
  "error": {
    "code": "NOT_FOUND",
    "message": "Node with ID 999 not found"
  }
}
```

---

### 1.4 Update Node

Update node information.

**Request**

```
PUT /nodes/{id}
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body** (all fields optional)

```json
{
  "name": "US-Node-01-Updated",
  "server_address": "new-proxy.example.com",
  "server_port": 8389,
  "encryption_method": "chacha20-ietf-poly1305",
  "plugin": "v2ray-plugin",
  "plugin_opts": {"mode": "websocket"},
  "region": "us-east",
  "tags": ["premium", "low-latency"],
  "description": "Updated description",
  "sort_order": 2
}
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "message": "Node updated successfully",
  "data": {
    "id": 1,
    "name": "US-Node-01-Updated",
    "server_address": "new-proxy.example.com",
    "server_port": 8389,
    "status": "active",
    "version": 2,
    "updated_at": "2024-01-15T16:00:00Z"
  }
}
```

---

### 1.5 Update Node Status

Update node operational status.

**Request**

```
PATCH /nodes/{id}/status
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body**

```json
{
  "status": "active"
}
```

**Status Values**

| Status | Description |
|--------|-------------|
| `active` | Node is active and available for use |
| `inactive` | Node is disabled |
| `maintenance` | Node is under maintenance |

**Status Transition Rules**

```
inactive  → active
active    → inactive, maintenance
maintenance → active, inactive
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "message": "Node status updated successfully",
  "data": {
    "id": 1,
    "status": "active",
    "is_available": true
  }
}
```

---

### 1.6 Generate Node Token

Generate a new API token for node authentication.

**Request**

```
POST /nodes/{id}/tokens
Authorization: Bearer <jwt_token>
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "message": "Token generated successfully",
  "data": {
    "node_id": 1,
    "token": "node_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
```

> **Important**: The previous token will be invalidated. Store the new token securely.

**Token Format**: `node_<base64_encoded_random_bytes>`

---

### 1.7 Delete Node

Delete a node permanently.

**Request**

```
DELETE /nodes/{id}
Authorization: Bearer <jwt_token>
```

**Response**

**Success (204)**: No content

---

## 2. Node Group Management

### 2.1 Create Node Group

Create a new node group for organizing nodes.

**Request**

```
POST /node-groups
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body**

```json
{
  "name": "Premium Nodes",
  "description": "High-speed premium nodes",
  "is_public": true,
  "sort_order": 1
}
```

**Request Fields**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Group name (unique) |
| `description` | string | No | Group description |
| `is_public` | bool | No | Public visibility (default: false) |
| `sort_order` | int | No | Display order |

**Response**

**Success (201)**

```json
{
  "success": true,
  "message": "Node group created successfully",
  "data": {
    "id": 1,
    "name": "Premium Nodes",
    "description": "High-speed premium nodes",
    "node_ids": [],
    "subscription_plan_ids": [],
    "is_public": true,
    "sort_order": 1,
    "node_count": 0,
    "version": 1,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

---

### 2.2 List Node Groups

Get a paginated list of node groups.

**Request**

```
GET /node-groups?page=1&page_size=20&is_public=true
Authorization: Bearer <jwt_token>
```

**Query Parameters**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `page_size` | int | 20 | Items per page (max: 100) |
| `is_public` | bool | - | Filter by public visibility |

**Response**

**Success (200)**

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "Premium Nodes",
        "description": "High-speed premium nodes",
        "node_ids": [1, 2, 3],
        "subscription_plan_ids": [1, 2],
        "is_public": true,
        "sort_order": 1,
        "node_count": 3,
        "version": 1,
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T14:20:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20,
    "total_pages": 1
  }
}
```

---

### 2.3 Get Node Group

Get details of a specific node group.

**Request**

```
GET /node-groups/{id}
Authorization: Bearer <jwt_token>
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "Premium Nodes",
    "description": "High-speed premium nodes",
    "node_ids": [1, 2, 3],
    "subscription_plan_ids": [1, 2],
    "is_public": true,
    "sort_order": 1,
    "node_count": 3,
    "version": 1,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T14:20:00Z"
  }
}
```

---

### 2.4 Update Node Group

Update node group information.

**Request**

```
PUT /node-groups/{id}?version=1
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body** (all fields optional)

```json
{
  "name": "Premium Nodes Updated",
  "description": "Updated description",
  "is_public": false,
  "sort_order": 2
}
```

> **Note**: Pass `version` query parameter for optimistic locking.

**Response**

**Success (200)**

```json
{
  "success": true,
  "message": "Node group updated successfully",
  "data": {
    "id": 1,
    "name": "Premium Nodes Updated",
    "version": 2
  }
}
```

---

### 2.5 Delete Node Group

Delete a node group.

**Request**

```
DELETE /node-groups/{id}
Authorization: Bearer <jwt_token>
```

**Response**

**Success (204)**: No content

---

### 2.6 List Group Nodes

Get all nodes in a specific group.

**Request**

```
GET /node-groups/{id}/nodes
Authorization: Bearer <jwt_token>
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "US-Node-01",
      "server_address": "proxy.example.com",
      "server_port": 8388,
      "protocol": "shadowsocks",
      "status": "active",
      "region": "us-west"
    }
  ]
}
```

---

### 2.7 Add Node to Group

Add a single node to a group.

**Request**

```
POST /node-groups/{id}/nodes
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body**

```json
{
  "node_id": 1
}
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "message": "Node added to group successfully",
  "data": {
    "group_id": 1,
    "node_id": 1
  }
}
```

---

### 2.8 Remove Node from Group

Remove a node from a group.

**Request**

```
DELETE /node-groups/{id}/nodes/{node_id}
Authorization: Bearer <jwt_token>
```

**Response**

**Success (204)**: No content

---

### 2.9 Batch Add Nodes to Group

Add multiple nodes to a group in a single operation.

**Request**

```
POST /node-groups/{id}/nodes/batch
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body**

```json
{
  "node_ids": [1, 2, 3, 4, 5]
}
```

> **Note**: Maximum 100 nodes per request.

**Response**

**Success (200)**

```json
{
  "success": true,
  "message": "5 nodes added to group successfully",
  "data": {
    "added_count": 5,
    "skipped_count": 0
  }
}
```

---

### 2.10 Batch Remove Nodes from Group

Remove multiple nodes from a group.

**Request**

```
DELETE /node-groups/{id}/nodes/batch
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body**

```json
{
  "node_ids": [1, 2, 3]
}
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "message": "3 nodes removed from group successfully",
  "data": {
    "removed_count": 3
  }
}
```

---

### 2.11 Associate Subscription Plan

Associate a subscription plan with a node group.

**Request**

```
POST /node-groups/{id}/plans
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body**

```json
{
  "plan_id": 1
}
```

**Response**

**Success (200)**

```json
{
  "success": true,
  "message": "Plan associated successfully",
  "data": {
    "group_id": 1,
    "plan_id": 1
  }
}
```

---

### 2.12 Disassociate Subscription Plan

Remove association between a plan and a group.

**Request**

```
DELETE /node-groups/{id}/plans/{plan_id}
Authorization: Bearer <jwt_token>
```

**Response**

**Success (204)**: No content

---

## 3. Subscription Endpoints

Public endpoints for fetching subscription configurations in various formats.

### 3.1 Base64 Subscription (Default)

Get subscription in Base64-encoded format.

**Request**

```
GET /sub/{token}
```

**Response**

**Success (200)**

```
Content-Type: text/plain

c3M6Ly9ZV1Z6TFRJMU5pMW5ZMjBLYUc1aGNIQjVMbVY0WVcxd2JHVXVZMjl0T2pnek9EZz0=
```

The Base64 content decodes to Shadowsocks/Trojan URIs, one per line:
```
ss://YWVzLTI1Ni1nY20KaG5hcHB5LmV4YW1wbGUuY29tOjgzODg=
ss://YWVzLTI1Ni1nY20KaG5hcHB5Mi5leGFtcGxlLmNvbTo4Mzg5
```

---

### 3.2 Clash Subscription

Get subscription in Clash YAML format.

**Request**

```
GET /sub/{token}/clash
```

**Response**

**Success (200)**

```yaml
Content-Type: text/yaml

proxies:
  - name: "US-Node-01"
    type: ss
    server: proxy.example.com
    port: 8388
    cipher: aes-256-gcm
    password: "subscription_uuid"
    plugin: obfs
    plugin-opts:
      mode: http
      host: example.com

proxy-groups:
  - name: "Proxy"
    type: select
    proxies:
      - "US-Node-01"
```

---

### 3.3 V2Ray Subscription

Get subscription in V2Ray JSON format.

**Request**

```
GET /sub/{token}/v2ray
```

**Response**

**Success (200)**

```json
Content-Type: application/json

{
  "outbounds": [
    {
      "protocol": "shadowsocks",
      "settings": {
        "servers": [
          {
            "address": "proxy.example.com",
            "port": 8388,
            "method": "aes-256-gcm",
            "password": "subscription_uuid"
          }
        ]
      },
      "tag": "US-Node-01"
    }
  ]
}
```

---

### 3.4 SIP008 Subscription

Get subscription in Shadowsocks SIP008 format.

**Request**

```
GET /sub/{token}/sip008
```

**Response**

**Success (200)**

```json
Content-Type: application/json

{
  "version": 1,
  "servers": [
    {
      "id": "uuid-1",
      "remarks": "US-Node-01",
      "server": "proxy.example.com",
      "server_port": 8388,
      "method": "aes-256-gcm",
      "password": "subscription_uuid",
      "plugin": "obfs-local",
      "plugin_opts": "obfs=http;obfs-host=example.com"
    }
  ],
  "bytes_used": 1073741824,
  "bytes_remaining": 9663676416
}
```

---

### 3.5 Surge Subscription

Get subscription in Surge configuration format.

**Request**

```
GET /sub/{token}/surge
```

**Response**

**Success (200)**

```ini
Content-Type: text/plain

[Proxy]
US-Node-01 = ss, proxy.example.com, 8388, encrypt-method=aes-256-gcm, password=subscription_uuid, obfs=http, obfs-host=example.com
```

---

## 4. Response Data Structures

### NodeDTO

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Unique node identifier |
| `name` | string | Node display name |
| `server_address` | string | Server hostname or IP |
| `server_port` | uint16 | Server port number |
| `protocol` | string | Protocol: `shadowsocks`, `trojan` |
| `encryption_method` | string | Encryption method |
| `plugin` | string | Plugin name (optional) |
| `plugin_opts` | object | Plugin options (optional) |
| `status` | string | Status: `active`, `inactive`, `maintenance` |
| `region` | string | Geographic region |
| `tags` | array | Custom tags |
| `sort_order` | int | Display order |
| `maintenance_reason` | string | Maintenance reason (if status is maintenance) |
| `is_available` | bool | Current availability |
| `version` | int | Version for optimistic locking |
| `created_at` | string | Creation timestamp (ISO 8601) |
| `updated_at` | string | Last update timestamp (ISO 8601) |
| `system_status` | object | Real-time system metrics (optional) |

### NodeSystemStatusDTO

| Field | Type | Description |
|-------|------|-------------|
| `cpu` | string | CPU usage percentage |
| `memory` | string | Memory usage percentage |
| `disk` | string | Disk usage percentage |
| `uptime` | int | Uptime in seconds |
| `updated_at` | int64 | Last update timestamp (Unix) |

### NodeGroupDTO

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Unique group identifier |
| `name` | string | Group name |
| `description` | string | Group description |
| `node_ids` | array | List of node IDs in group |
| `subscription_plan_ids` | array | Associated plan IDs |
| `is_public` | bool | Public visibility |
| `sort_order` | int | Display order |
| `node_count` | int | Number of nodes |
| `version` | int | Version for optimistic locking |
| `created_at` | string | Creation timestamp (ISO 8601) |
| `updated_at` | string | Last update timestamp (ISO 8601) |

---

## 5. Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | VALIDATION_ERROR | Invalid request body or parameters |
| 401 | UNAUTHORIZED | Missing or invalid authentication |
| 403 | FORBIDDEN | Insufficient permissions (admin required) |
| 404 | NOT_FOUND | Resource not found |
| 409 | CONFLICT | Resource conflict (e.g., duplicate name) |
| 500 | INTERNAL_ERROR | Server-side error |

**Error Response Format**

```json
{
  "success": false,
  "message": "Error description",
  "error": {
    "code": "ERROR_CODE",
    "message": "Detailed error message"
  }
}
```

---

## 6. Node Token Authentication

For node status reporting and heartbeat, nodes authenticate using API tokens.

### Token Format

```
node_<base64_encoded_random_bytes>
```

### Authentication Methods

**1. Authorization Header (Recommended)**
```
Authorization: Bearer node_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**2. Query Parameter**
```
GET /endpoint?token=node_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**3. X-Node-Token Header (RESTful)**
```
X-Node-Token: node_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### Token Lifecycle

1. Token is generated when creating a node
2. Token can be regenerated via `POST /nodes/{id}/tokens`
3. Only the hash is stored; plaintext is returned once
4. Old token is invalidated when regenerating

---

## 7. Client Implementation Example

### Go Client

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
    baseURL = "https://api.example.com"
)

type NodeClient struct {
    client *http.Client
    token  string
}

func NewNodeClient(token string) *NodeClient {
    return &NodeClient{
        client: &http.Client{Timeout: 10 * time.Second},
        token:  token,
    }
}

// CreateNode creates a new proxy node
func (c *NodeClient) CreateNode(req CreateNodeRequest) (*NodeResponse, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequest("POST", baseURL+"/nodes", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Authorization", "Bearer "+c.token)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result NodeResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}

// ListNodes retrieves paginated node list
func (c *NodeClient) ListNodes(page, pageSize int, status string) (*ListNodesResponse, error) {
    url := fmt.Sprintf("%s/nodes?page=%d&page_size=%d", baseURL, page, pageSize)
    if status != "" {
        url += "&status=" + status
    }

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+c.token)

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result ListNodesResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}

// UpdateNodeStatus updates node operational status
func (c *NodeClient) UpdateNodeStatus(nodeID uint, status string) error {
    body, _ := json.Marshal(map[string]string{"status": status})

    req, err := http.NewRequest("PATCH",
        fmt.Sprintf("%s/nodes/%d/status", baseURL, nodeID),
        bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer "+c.token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("failed to update status: %d", resp.StatusCode)
    }

    return nil
}

// Types
type CreateNodeRequest struct {
    Name             string            `json:"name"`
    ServerAddress    string            `json:"server_address"`
    ServerPort       uint16            `json:"server_port"`
    Protocol         string            `json:"protocol"`
    EncryptionMethod string            `json:"encryption_method"`
    Plugin           string            `json:"plugin,omitempty"`
    PluginOpts       map[string]string `json:"plugin_opts,omitempty"`
    Region           string            `json:"region,omitempty"`
    Tags             []string          `json:"tags,omitempty"`
    SortOrder        int               `json:"sort_order,omitempty"`
}

type NodeResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Data    Node   `json:"data"`
}

type Node struct {
    ID               uint              `json:"id"`
    Name             string            `json:"name"`
    ServerAddress    string            `json:"server_address"`
    ServerPort       uint16            `json:"server_port"`
    Protocol         string            `json:"protocol"`
    EncryptionMethod string            `json:"encryption_method"`
    Status           string            `json:"status"`
    APIToken         string            `json:"api_token,omitempty"`
}

type ListNodesResponse struct {
    Success bool `json:"success"`
    Data    struct {
        Items      []Node `json:"items"`
        Total      int    `json:"total"`
        Page       int    `json:"page"`
        PageSize   int    `json:"page_size"`
        TotalPages int    `json:"total_pages"`
    } `json:"data"`
}

func main() {
    client := NewNodeClient("your-jwt-token")

    // Create a new node
    node, err := client.CreateNode(CreateNodeRequest{
        Name:             "US-Node-01",
        ServerAddress:    "proxy.example.com",
        ServerPort:       8388,
        Protocol:         "shadowsocks",
        EncryptionMethod: "aes-256-gcm",
        Region:           "us-west",
        Tags:             []string{"premium"},
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Created node: %s (ID: %d)\n", node.Data.Name, node.Data.ID)
    fmt.Printf("API Token: %s\n", node.Data.APIToken)

    // Activate the node
    if err := client.UpdateNodeStatus(node.Data.ID, "active"); err != nil {
        panic(err)
    }
    fmt.Println("Node activated")

    // List active nodes
    nodes, err := client.ListNodes(1, 20, "active")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Found %d active nodes\n", nodes.Data.Total)
}
```

---

## 8. Notes

1. **Password Handling**: For Shadowsocks nodes, an HMAC-SHA256 signed password (derived from subscription UUID) is used when generating subscription URIs. The original UUID is never exposed to agents.
2. **Rate Limiting**: Subscription endpoints have rate limiting enabled to prevent abuse
3. **Optimistic Locking**: Use `version` parameter when updating to prevent concurrent modification conflicts
4. **Token Security**: Node API tokens should be treated like passwords and stored securely
5. **Status Transitions**: Follow the state machine rules when changing node status
6. **Batch Operations**: Maximum 100 items per batch request for adding/removing nodes from groups
