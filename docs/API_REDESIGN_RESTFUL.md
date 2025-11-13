# API Redesign: RESTful Architecture Migration

## Executive Summary

### Purpose of Redesign

This document outlines the migration from a legacy query-parameter-based API design to a RESTful architecture for the Node Backend API endpoints. The current implementation uses a single endpoint (`/api/node`) with action routing via the `act` query parameter, which violates RESTful principles and HTTP semantics.

### Current vs RESTful Design

**Current Design (Legacy):**
- Single endpoint with action routing via query parameters
- Violates HTTP verb semantics (using GET/POST for all operations)
- Non-standard authentication via query string tokens
- Inconsistent with the rest of the Orris API architecture

**RESTful Design (Proposed):**
- Resource-oriented URL structure
- Proper HTTP verb usage (GET for reads, POST for writes)
- Header-based authentication
- Consistent with existing `/nodes` API design

### Migration Strategy

A dual-track approach will be implemented:
1. **Phase 1 (Weeks 1-2):** Implement new RESTful endpoints
2. **Phase 2 (Weeks 3-4):** Test and document new APIs
3. **Phase 3 (Months 2-6):** Maintain both old and new APIs in parallel
4. **Phase 4 (Month 7):** Deprecate and remove legacy API

---

## Current API Design (Legacy)

### Overview

The current Node Backend API (v2raysocks compatible) uses a single unified endpoint with action-based routing through query parameters.

### Endpoint Structure

```
Base URL: /api/node
Methods: GET, POST
```

### Five Operations

#### 1. Get Node Configuration
```http
GET /api/node?act=config&node_id={id}&token={token}&node_type={type}
```

**Query Parameters:**
- `act=config` - Action identifier
- `node_id` - Node identifier (required)
- `token` - Authentication token (required)
- `node_type` - Protocol type: shadowsocks/trojan (optional)

**Response Format:**
```json
{
  "data": {
    "node_id": 1,
    "node_type": "shadowsocks",
    "server_host": "1.2.3.4",
    "server_port": 8388,
    "method": "aes-256-gcm",
    "transport_protocol": "tcp",
    "enable_vless": false,
    "enable_xtls": false,
    "speed_limit": 0,
    "device_limit": 0
  },
  "ret": 1,
  "msg": "success"
}
```

#### 2. Get Authorized Users
```http
GET /api/node?act=user&node_id={id}&token={token}
```

**Query Parameters:**
- `act=user` - Action identifier
- `node_id` - Node identifier (required)
- `token` - Authentication token (required)

**Response Format:**
```json
{
  "data": [
    {
      "id": 1,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user1@example.com",
      "st": 0,
      "dt": 0,
      "expire_time": 1735689600
    }
  ],
  "ret": 1,
  "msg": "success"
}
```

#### 3. Submit User Traffic
```http
POST /api/node?act=submit&node_id={id}&token={token}
Content-Type: application/json

[
  {
    "UID": 1,
    "Upload": 1048576,
    "Download": 5242880
  }
]
```

**Query Parameters:**
- `act=submit` - Action identifier
- `node_id` - Node identifier (required)
- `token` - Authentication token (required)

**Request Body:** Array of `UserTrafficItem`

**Response Format:**
```json
{
  "data": {
    "users_updated": 1
  },
  "ret": 1,
  "msg": "success"
}
```

#### 4. Report Node Status
```http
POST /api/node?act=nodestatus&node_id={id}&token={token}
Content-Type: application/json

{
  "CPU": "25%",
  "Mem": "60%",
  "Net": "100 MB",
  "Disk": "45%",
  "Uptime": 86400
}
```

**Query Parameters:**
- `act=nodestatus` - Action identifier
- `node_id` - Node identifier (required)
- `token` - Authentication token (required)

**Request Body:** `ReportNodeStatusRequest`

**Response Format:**
```json
{
  "data": {
    "status": "ok"
  },
  "ret": 1,
  "msg": "success"
}
```

#### 5. Report Online Users
```http
POST /api/node?act=onlineusers&node_id={id}&token={token}
Content-Type: application/json

{
  "users": [
    {
      "UID": 1,
      "IP": "192.168.1.100"
    }
  ]
}
```

**Query Parameters:**
- `act=onlineusers` - Action identifier
- `node_id` - Node identifier (required)
- `token` - Authentication token (required)

**Request Body:** `ReportOnlineUsersRequest`

**Response Format:**
```json
{
  "data": {
    "online_count": 1
  },
  "ret": 1,
  "msg": "success"
}
```

### Problems with Current Design

1. **Violates RESTful Principles**
   - Uses query parameter (`act`) for action routing instead of URL paths
   - Single endpoint handles multiple resources and operations
   - URL does not represent a resource

2. **HTTP Semantics Violation**
   - Uses POST for read operations (should use GET)
   - Inconsistent HTTP verb usage

3. **Security Concerns**
   - Authentication token in query string (visible in logs, browser history)
   - Query strings are often logged by proxies and web servers

4. **Inconsistent Architecture**
   - Rest of Orris API uses RESTful design (`/nodes`, `/users`, `/subscriptions`)
   - Creates confusion for API consumers

5. **Poor Discoverability**
   - Cannot determine available operations from URL structure
   - Requires documentation to understand `act` parameter values

6. **Caching Issues**
   - Query parameters make HTTP caching more complex
   - Cannot use standard cache-control mechanisms effectively

---

## New API Design (RESTful)

### Overview

The new design follows RESTful principles with resource-oriented URLs, proper HTTP verb usage, and header-based authentication.

### Resource Hierarchy

```
/api/v1/nodes/{id}/
├── config         (Node configuration)
├── users          (Authorized users)
├── traffic        (Traffic data reporting)
├── status         (Node system status)
└── online-users   (Online user sessions)
```

### Version Control

- Base path: `/api/v1/nodes`
- Version included in URL for future-proofing
- Allows parallel operation with legacy endpoints

### Authentication

**Header-based authentication:**
```http
X-Node-Token: {authentication_token}
```

**Benefits:**
- Tokens not visible in URLs or logs
- Follows HTTP best practices
- Consistent with JWT authentication used elsewhere in Orris

---

## Detailed API Specifications

### 1. Get Node Configuration

#### Endpoint
```http
GET /api/v1/nodes/{id}/config
```

#### Request

**Path Parameters:**
- `id` (integer, required) - Node identifier

**Headers:**
```http
X-Node-Token: abc123def456
Accept: application/json
```

**Query Parameters (Optional):**
- `node_type` (string) - Protocol type filter: `shadowsocks`, `trojan`

#### Response

**Success (200 OK):**
```json
{
  "data": {
    "node_id": 1,
    "node_type": "shadowsocks",
    "server_host": "1.2.3.4",
    "server_port": 8388,
    "method": "aes-256-gcm",
    "server_key": "",
    "transport_protocol": "tcp",
    "host": "",
    "path": "",
    "enable_vless": false,
    "enable_xtls": false,
    "speed_limit": 0,
    "device_limit": 0,
    "rule_list_path": ""
  },
  "ret": 1,
  "msg": "success"
}
```

**Error Responses:**

*401 Unauthorized:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid or missing authentication token"
}
```

*404 Not Found:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "node not found"
}
```

*500 Internal Server Error:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "failed to retrieve node configuration"
}
```

#### HTTP Status Codes
- `200 OK` - Configuration retrieved successfully
- `304 Not Modified` - Resource not modified (ETag support)
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - Node does not exist
- `500 Internal Server Error` - Server-side error

#### Caching
- Supports ETag for efficient caching
- `If-None-Match` header support
- Reduces bandwidth for unchanged configurations

#### Use Cases
- XrayR node initialization
- Periodic configuration synchronization
- Configuration change detection

---

### 2. Get Authorized Users

#### Endpoint
```http
GET /api/v1/nodes/{id}/users
```

#### Request

**Path Parameters:**
- `id` (integer, required) - Node identifier

**Headers:**
```http
X-Node-Token: abc123def456
Accept: application/json
```

#### Response

**Success (200 OK):**
```json
{
  "data": [
    {
      "id": 1,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user1@example.com",
      "st": 0,
      "dt": 0,
      "expire_time": 1735689600
    },
    {
      "id": 2,
      "uuid": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "email": "user2@example.com",
      "st": 1048576,
      "dt": 5,
      "expire_time": 1738368000
    }
  ],
  "ret": 1,
  "msg": "success"
}
```

**Error Responses:**

*401 Unauthorized:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid or missing authentication token"
}
```

*404 Not Found:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "node not found"
}
```

*500 Internal Server Error:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "failed to retrieve user list"
}
```

#### HTTP Status Codes
- `200 OK` - User list retrieved successfully
- `304 Not Modified` - Resource not modified (ETag support)
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - Node does not exist
- `500 Internal Server Error` - Server-side error

#### User Information Fields
- `id` - User unique identifier
- `uuid` - Trojan password / SS identifier (deterministic UUID v5)
- `email` - User email address
- `st` - Speed limit in bps (0 = unlimited)
- `dt` - Device connection limit (0 = unlimited)
- `expire_time` - Unix timestamp of subscription expiration

#### Caching
- Supports ETag for efficient caching
- User list may change frequently (new subscriptions, expirations)

#### Use Cases
- Node startup user synchronization
- Periodic user list refresh
- Access control updates

---

### 3. Report User Traffic

#### Endpoint
```http
POST /api/v1/nodes/{id}/traffic
```

#### Request

**Path Parameters:**
- `id` (integer, required) - Node identifier

**Headers:**
```http
X-Node-Token: abc123def456
Content-Type: application/json
Accept: application/json
```

**Request Body:**
```json
[
  {
    "UID": 1,
    "Upload": 1048576,
    "Download": 5242880
  },
  {
    "UID": 2,
    "Upload": 2097152,
    "Download": 10485760
  }
]
```

**Body Schema:** Array of `UserTrafficItem`
- `UID` (integer, required) - User unique identifier
- `Upload` (integer, required) - Upload traffic in bytes (>= 0)
- `Download` (integer, required) - Download traffic in bytes (>= 0)

#### Response

**Success (200 OK):**
```json
{
  "data": {
    "users_updated": 2
  },
  "ret": 1,
  "msg": "success"
}
```

**Error Responses:**

*400 Bad Request:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid request body"
}
```

*401 Unauthorized:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid or missing authentication token"
}
```

*404 Not Found:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "node not found"
}
```

*500 Internal Server Error:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "failed to process traffic report"
}
```

#### HTTP Status Codes
- `200 OK` - Traffic data processed successfully
- `400 Bad Request` - Invalid request body format
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - Node does not exist
- `500 Internal Server Error` - Server-side error

#### Use Cases
- Periodic traffic reporting (e.g., every 60 seconds)
- Bandwidth quota tracking
- User traffic statistics

---

### 4. Report Node Status

#### Endpoint
```http
POST /api/v1/nodes/{id}/status
```

#### Request

**Path Parameters:**
- `id` (integer, required) - Node identifier

**Headers:**
```http
X-Node-Token: abc123def456
Content-Type: application/json
Accept: application/json
```

**Request Body:**
```json
{
  "CPU": "25%",
  "Mem": "60%",
  "Net": "100 MB",
  "Disk": "45%",
  "Uptime": 86400
}
```

**Body Schema:** `ReportNodeStatusRequest`
- `CPU` (string, required) - CPU usage percentage (format: "XX%")
- `Mem` (string, required) - Memory usage percentage (format: "XX%")
- `Net` (string, required) - Network usage (format: "XX MB")
- `Disk` (string, required) - Disk usage percentage (format: "XX%")
- `Uptime` (integer, required) - System uptime in seconds (>= 0)

#### Response

**Success (200 OK):**
```json
{
  "data": {
    "status": "ok"
  },
  "ret": 1,
  "msg": "success"
}
```

**Error Responses:**

*400 Bad Request:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid request body"
}
```

*401 Unauthorized:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid or missing authentication token"
}
```

*404 Not Found:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "node not found"
}
```

*500 Internal Server Error:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "failed to process status report"
}
```

#### HTTP Status Codes
- `200 OK` - Status data processed successfully
- `400 Bad Request` - Invalid request body format
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - Node does not exist
- `500 Internal Server Error` - Server-side error

#### Use Cases
- Node health monitoring
- System resource tracking
- Alert generation for resource thresholds

---

### 5. Report Online Users

#### Endpoint
```http
POST /api/v1/nodes/{id}/online-users
```

#### Request

**Path Parameters:**
- `id` (integer, required) - Node identifier

**Headers:**
```http
X-Node-Token: abc123def456
Content-Type: application/json
Accept: application/json
```

**Request Body:**
```json
{
  "users": [
    {
      "UID": 1,
      "IP": "192.168.1.100"
    },
    {
      "UID": 2,
      "IP": "192.168.1.101"
    }
  ]
}
```

**Body Schema:** `ReportOnlineUsersRequest`
- `users` (array, required) - Array of `OnlineUserItem`
  - `UID` (integer, required) - User unique identifier
  - `IP` (string, required) - User connection IP address

#### Response

**Success (200 OK):**
```json
{
  "data": {
    "online_count": 2
  },
  "ret": 1,
  "msg": "success"
}
```

**Error Responses:**

*400 Bad Request:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid request body"
}
```

*401 Unauthorized:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid or missing authentication token"
}
```

*404 Not Found:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "node not found"
}
```

*500 Internal Server Error:*
```json
{
  "data": null,
  "ret": 0,
  "msg": "failed to process online users report"
}
```

#### HTTP Status Codes
- `200 OK` - Online users data processed successfully
- `400 Bad Request` - Invalid request body format
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - Node does not exist
- `500 Internal Server Error` - Server-side error

#### Use Cases
- Real-time connection tracking
- Concurrent user monitoring
- Device limit enforcement

---

## API Comparison Table

| Feature | Legacy API | RESTful API | HTTP Method | Improvements |
|---------|-----------|-------------|-------------|--------------|
| **Get Configuration** | `/api/node?act=config&node_id={id}&token={token}` | `/api/v1/nodes/{id}/config` | `GET` | - Resource-oriented URL<br>- Clean path structure<br>- Header authentication |
| **Get Users** | `/api/node?act=user&node_id={id}&token={token}` | `/api/v1/nodes/{id}/users` | `GET` | - Resource hierarchy clear<br>- RESTful naming<br>- Secure token handling |
| **Report Traffic** | `/api/node?act=submit&node_id={id}&token={token}` | `/api/v1/nodes/{id}/traffic` | `POST` | - Semantic URL<br>- Proper HTTP verb<br>- Clear resource intent |
| **Report Status** | `/api/node?act=nodestatus&node_id={id}&token={token}` | `/api/v1/nodes/{id}/status` | `POST` | - Self-documenting URL<br>- Standard REST pattern<br>- Version control support |
| **Report Online Users** | `/api/node?act=onlineusers&node_id={id}&token={token}` | `/api/v1/nodes/{id}/online-users` | `POST` | - Hyphenated resource naming<br>- URL hierarchy<br>- Better discoverability |

### Key Improvements

1. **URL Structure**
   - Legacy: Query parameter routing (`?act=config`)
   - RESTful: Resource-oriented paths (`/nodes/{id}/config`)

2. **Authentication**
   - Legacy: `?token={token}` in query string
   - RESTful: `X-Node-Token` header

3. **HTTP Semantics**
   - Legacy: Mixed GET/POST usage
   - RESTful: GET for reads, POST for writes

4. **Versioning**
   - Legacy: No versioning
   - RESTful: `/api/v1` prefix

5. **Discoverability**
   - Legacy: Requires documentation to know `act` values
   - RESTful: Self-documenting URL structure

---

## Authentication Comparison

### Legacy Authentication (Query String)

```http
GET /api/node?act=config&node_id=1&token=abc123def456
```

**Problems:**
- Token visible in URL
- Logged by web servers and proxies
- Stored in browser history
- Included in referrer headers
- Cannot be easily rotated without client updates

### RESTful Authentication (Header)

```http
GET /api/v1/nodes/1/config
X-Node-Token: abc123def456
```

**Benefits:**
- Token not visible in URLs
- Not logged by default
- Not stored in browser history
- Not sent in referrer headers
- Can be rotated independently
- Follows OAuth 2.0 and JWT patterns
- Compatible with API gateways

### Security Comparison

| Aspect | Query String | Header |
|--------|--------------|--------|
| **Log Visibility** | Visible in access logs | Not logged by default |
| **Browser History** | Stored in history | Not stored |
| **Referrer Leakage** | Sent in Referer header | Never sent |
| **Cache Keys** | May be cached | Separated from cache key |
| **Token Rotation** | Requires URL change | Header-only change |
| **Standard Practice** | Non-standard | Industry standard |

---

## Response Format

Both legacy and RESTful APIs maintain **v2raysocks compatibility** with the standard response envelope:

```json
{
  "data": { ... },
  "ret": 1,
  "msg": "success"
}
```

### Response Fields

- `data` - Response payload (varies by endpoint)
- `ret` - Return code
  - `1` = Success
  - `0` = Error
- `msg` - Human-readable message

### Why Maintain This Format?

1. **Backward Compatibility** - XrayR and other node backends expect this format
2. **Protocol Compliance** - v2raysocks specification requirement
3. **Client Compatibility** - Existing node software doesn't need response parsing changes

### Alternative Considered

Standard REST API response:
```json
{
  "success": true,
  "data": { ... },
  "message": "success"
}
```

**Decision:** Keep v2raysocks format for node backend endpoints to ensure compatibility with existing node software (XrayR, etc.)

---

## Error Handling

### Standardized HTTP Status Codes

| Status Code | Meaning | When to Use |
|-------------|---------|-------------|
| `200 OK` | Success | Request processed successfully |
| `304 Not Modified` | Not Modified | Resource unchanged (ETag match) |
| `400 Bad Request` | Client Error | Invalid request body or parameters |
| `401 Unauthorized` | Authentication Failed | Missing or invalid token |
| `404 Not Found` | Resource Not Found | Node ID does not exist |
| `500 Internal Server Error` | Server Error | Unexpected server-side error |

### Error Response Format

All errors follow the v2raysocks format:

```json
{
  "data": null,
  "ret": 0,
  "msg": "descriptive error message"
}
```

### Error Message Examples

**400 Bad Request:**
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid request body: field 'UID' is required"
}
```

**401 Unauthorized:**
```json
{
  "data": null,
  "ret": 0,
  "msg": "invalid or missing authentication token"
}
```

**404 Not Found:**
```json
{
  "data": null,
  "ret": 0,
  "msg": "node not found with id: 999"
}
```

**500 Internal Server Error:**
```json
{
  "data": null,
  "ret": 0,
  "msg": "failed to process traffic report: database connection error"
}
```

### Error Logging

- Client errors (4xx): Logged at WARN level
- Server errors (5xx): Logged at ERROR level
- Include request context: node_id, act, client_ip

---

## Migration Strategy

### Dual-Track Approach

Both legacy and RESTful APIs will coexist during the migration period.

```
┌─────────────────────────────────────────────────┐
│          Legacy API (Deprecated)                │
│  /api/node?act=config&node_id={id}&token={...} │
│                                                 │
│  Status: DEPRECATED                             │
│  Timeline: Months 0-6                           │
│  Action: Redirect to docs, add deprecation     │
│          warnings in response headers           │
└─────────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────┐
│           RESTful API (Recommended)             │
│     /api/v1/nodes/{id}/config                   │
│     Header: X-Node-Token: {token}               │
│                                                 │
│  Status: ACTIVE                                 │
│  Timeline: Month 0 onwards                      │
│  Action: Promote in documentation               │
└─────────────────────────────────────────────────┘
```

### Phase 1: Implementation (Weeks 1-2)

**Tasks:**
1. Create new route handlers for `/api/v1/nodes/{id}/*`
2. Implement header-based authentication middleware
3. Add resource-oriented endpoints (config, users, traffic, status, online-users)
4. Maintain response format compatibility (v2raysocks)
5. Write unit tests for new endpoints

**Deliverables:**
- New RESTful endpoints fully functional
- Authentication middleware supporting `X-Node-Token`
- Unit test coverage >= 80%

### Phase 2: Testing & Documentation (Weeks 3-4)

**Tasks:**
1. Integration testing with XrayR test environment
2. Update API documentation (Swagger/OpenAPI)
3. Create migration guide for node operators
4. Performance testing and optimization
5. Security audit of authentication changes

**Deliverables:**
- Swagger documentation updated
- Migration guide published
- Performance benchmarks documented
- Security review completed

### Phase 3: Parallel Operation (Months 2-6)

**Tasks:**
1. Run both APIs in production
2. Monitor usage metrics (which API is being called)
3. Add deprecation warnings to legacy API responses
4. Gradual migration of node instances
5. Support node operators during migration

**Deprecation Headers (Legacy API):**
```http
HTTP/1.1 200 OK
Deprecated: true
Sunset: 2025-07-01
Link: </api/v1/nodes/1/config>; rel="alternate"
Warning: 299 - "This API is deprecated. Please migrate to /api/v1/nodes"
```

**Metrics to Track:**
- Legacy API request count (should decrease)
- RESTful API request count (should increase)
- Error rates on both APIs
- Node migration completion percentage

### Phase 4: Deprecation (Month 7)

**Tasks:**
1. Final communication to all node operators
2. Return 410 Gone for legacy endpoints
3. Remove legacy endpoint code
4. Update documentation to remove legacy references

**Final Response (Legacy API):**
```http
HTTP/1.1 410 Gone
Content-Type: application/json

{
  "data": null,
  "ret": 0,
  "msg": "This API has been removed. Please use /api/v1/nodes endpoints. See migration guide: https://docs.orris.example.com/api-migration"
}
```

### Migration Timeline

```
Month 0  │ Week 1-2: Implementation
         │ Week 3-4: Testing & Documentation
         │
Month 1  │ Deploy RESTful API to production
         │ Both APIs available
         │
Month 2  │ Add deprecation warnings
         │ Begin node migration
         │
Month 3  │ 25% nodes migrated
         │
Month 4  │ 50% nodes migrated
         │
Month 5  │ 75% nodes migrated
         │
Month 6  │ 100% nodes migrated
         │ Final migration deadline
         │
Month 7  │ Remove legacy API
         │ Return 410 Gone
```

### Rollback Plan

If critical issues arise:
1. Keep legacy API active
2. Fix issues in RESTful API
3. Re-test thoroughly
4. Resume migration

**Rollback Triggers:**
- Error rate > 5% on RESTful API
- Performance degradation > 20%
- Security vulnerability discovered

---

## Code Examples

### Legacy API Usage

#### Get Node Configuration
```bash
# Using query parameters for authentication and action routing
curl -X GET "https://api.example.com/api/node?act=config&node_id=1&token=abc123def456&node_type=shadowsocks"
```

#### Get Authorized Users
```bash
curl -X GET "https://api.example.com/api/node?act=user&node_id=1&token=abc123def456"
```

#### Report User Traffic
```bash
curl -X POST "https://api.example.com/api/node?act=submit&node_id=1&token=abc123def456" \
  -H "Content-Type: application/json" \
  -d '[
    {
      "UID": 1,
      "Upload": 1048576,
      "Download": 5242880
    }
  ]'
```

#### Report Node Status
```bash
curl -X POST "https://api.example.com/api/node?act=nodestatus&node_id=1&token=abc123def456" \
  -H "Content-Type: application/json" \
  -d '{
    "CPU": "25%",
    "Mem": "60%",
    "Net": "100 MB",
    "Disk": "45%",
    "Uptime": 86400
  }'
```

#### Report Online Users
```bash
curl -X POST "https://api.example.com/api/node?act=onlineusers&node_id=1&token=abc123def456" \
  -H "Content-Type: application/json" \
  -d '{
    "users": [
      {
        "UID": 1,
        "IP": "192.168.1.100"
      }
    ]
  }'
```

### RESTful API Usage

#### Get Node Configuration
```bash
# Using header-based authentication and resource-oriented URL
curl -X GET "https://api.example.com/api/v1/nodes/1/config?node_type=shadowsocks" \
  -H "X-Node-Token: abc123def456" \
  -H "Accept: application/json"
```

#### Get Authorized Users
```bash
curl -X GET "https://api.example.com/api/v1/nodes/1/users" \
  -H "X-Node-Token: abc123def456" \
  -H "Accept: application/json"
```

#### Report User Traffic
```bash
curl -X POST "https://api.example.com/api/v1/nodes/1/traffic" \
  -H "X-Node-Token: abc123def456" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '[
    {
      "UID": 1,
      "Upload": 1048576,
      "Download": 5242880
    }
  ]'
```

#### Report Node Status
```bash
curl -X POST "https://api.example.com/api/v1/nodes/1/status" \
  -H "X-Node-Token: abc123def456" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "CPU": "25%",
    "Mem": "60%",
    "Net": "100 MB",
    "Disk": "45%",
    "Uptime": 86400
  }'
```

#### Report Online Users
```bash
curl -X POST "https://api.example.com/api/v1/nodes/1/online-users" \
  -H "X-Node-Token: abc123def456" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "users": [
      {
        "UID": 1,
        "IP": "192.168.1.100"
      }
    ]
  }'
```

### Using ETag Caching

#### Initial Request
```bash
curl -X GET "https://api.example.com/api/v1/nodes/1/config" \
  -H "X-Node-Token: abc123def456" \
  -H "Accept: application/json" \
  -v
```

**Response:**
```http
HTTP/1.1 200 OK
ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4"
Content-Type: application/json

{
  "data": { ... },
  "ret": 1,
  "msg": "success"
}
```

#### Conditional Request
```bash
curl -X GET "https://api.example.com/api/v1/nodes/1/config" \
  -H "X-Node-Token: abc123def456" \
  -H "Accept: application/json" \
  -H "If-None-Match: \"33a64df551425fcc55e4d42a148795d9f25f89d4\"" \
  -v
```

**Response (Unchanged):**
```http
HTTP/1.1 304 Not Modified
ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4"
```

### Python Example (Legacy vs RESTful)

#### Legacy API
```python
import requests

# Configuration
base_url = "https://api.example.com"
node_id = 1
token = "abc123def456"

# Get node configuration
response = requests.get(
    f"{base_url}/api/node",
    params={
        "act": "config",
        "node_id": node_id,
        "token": token,
        "node_type": "shadowsocks"
    }
)
config = response.json()

# Report traffic
traffic_data = [
    {"UID": 1, "Upload": 1048576, "Download": 5242880}
]
response = requests.post(
    f"{base_url}/api/node",
    params={
        "act": "submit",
        "node_id": node_id,
        "token": token
    },
    json=traffic_data
)
result = response.json()
```

#### RESTful API
```python
import requests

# Configuration
base_url = "https://api.example.com"
node_id = 1
token = "abc123def456"

# Headers (reusable)
headers = {
    "X-Node-Token": token,
    "Accept": "application/json"
}

# Get node configuration
response = requests.get(
    f"{base_url}/api/v1/nodes/{node_id}/config",
    headers=headers,
    params={"node_type": "shadowsocks"}
)
config = response.json()

# Report traffic
traffic_data = [
    {"UID": 1, "Upload": 1048576, "Download": 5242880}
]
response = requests.post(
    f"{base_url}/api/v1/nodes/{node_id}/traffic",
    headers=headers,
    json=traffic_data
)
result = response.json()
```

### Go Example (XrayR Integration)

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

// RESTful API client
type NodeAPIClient struct {
    baseURL string
    nodeID  int
    token   string
    client  *http.Client
}

func NewNodeAPIClient(baseURL string, nodeID int, token string) *NodeAPIClient {
    return &NodeAPIClient{
        baseURL: baseURL,
        nodeID:  nodeID,
        token:   token,
        client:  &http.Client{},
    }
}

func (c *NodeAPIClient) GetConfig() (*NodeConfig, error) {
    url := fmt.Sprintf("%s/api/v1/nodes/%d/config", c.baseURL, c.nodeID)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("X-Node-Token", c.token)
    req.Header.Set("Accept", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data NodeConfig `json:"data"`
        Ret  int        `json:"ret"`
        Msg  string     `json:"msg"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    if result.Ret != 1 {
        return nil, fmt.Errorf("API error: %s", result.Msg)
    }

    return &result.Data, nil
}

func (c *NodeAPIClient) ReportTraffic(traffic []UserTrafficItem) error {
    url := fmt.Sprintf("%s/api/v1/nodes/%d/traffic", c.baseURL, c.nodeID)

    body, err := json.Marshal(traffic)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return err
    }

    req.Header.Set("X-Node-Token", c.token)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var result struct {
        Data struct {
            UsersUpdated int `json:"users_updated"`
        } `json:"data"`
        Ret int    `json:"ret"`
        Msg string `json:"msg"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

    if result.Ret != 1 {
        return fmt.Errorf("API error: %s", result.Msg)
    }

    return nil
}

type NodeConfig struct {
    NodeID            int    `json:"node_id"`
    NodeType          string `json:"node_type"`
    ServerHost        string `json:"server_host"`
    ServerPort        int    `json:"server_port"`
    Method            string `json:"method"`
    TransportProtocol string `json:"transport_protocol"`
    // ... other fields
}

type UserTrafficItem struct {
    UID      int   `json:"UID"`
    Upload   int64 `json:"Upload"`
    Download int64 `json:"Download"`
}

func main() {
    client := NewNodeAPIClient("https://api.example.com", 1, "abc123def456")

    // Get node configuration
    config, err := client.GetConfig()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Node config: %+v\n", config)

    // Report traffic
    traffic := []UserTrafficItem{
        {UID: 1, Upload: 1048576, Download: 5242880},
    }
    if err := client.ReportTraffic(traffic); err != nil {
        panic(err)
    }
    fmt.Println("Traffic reported successfully")
}
```

---

## RESTful Best Practices Applied

This redesign adheres to the following RESTful principles:

### 1. Resource-Oriented Design

**URLs represent resources, not actions:**
- ✅ `/api/v1/nodes/{id}/config` (resource: node configuration)
- ❌ `/api/node?act=getconfig` (action-based)

**Resource hierarchy reflects relationships:**
```
/nodes/{id}/              → Node resource
    ├── /config           → Configuration sub-resource
    ├── /users            → Users sub-resource
    ├── /traffic          → Traffic sub-resource
    ├── /status           → Status sub-resource
    └── /online-users     → Online users sub-resource
```

### 2. HTTP Method Semantics

**GET for safe, idempotent reads:**
- `GET /api/v1/nodes/{id}/config` - Retrieve configuration
- `GET /api/v1/nodes/{id}/users` - Retrieve user list

**POST for writes and non-idempotent operations:**
- `POST /api/v1/nodes/{id}/traffic` - Submit traffic data
- `POST /api/v1/nodes/{id}/status` - Submit status update
- `POST /api/v1/nodes/{id}/online-users` - Submit online users

### 3. Stateless Communication

**Each request contains all necessary information:**
- Authentication in `X-Node-Token` header
- Node ID in URL path
- Request parameters in query string or body

**No server-side session required:**
- Token-based authentication (stateless)
- No cookies or session storage
- Horizontal scaling friendly

### 4. Uniform Interface

**Consistent patterns across all endpoints:**
- Path structure: `/api/v1/nodes/{id}/{resource}`
- Authentication: `X-Node-Token` header
- Response format: v2raysocks envelope
- Error handling: Standard HTTP status codes

### 5. Layered System

**API design supports intermediaries:**
- Reverse proxies can cache based on URL
- Load balancers can route based on path
- API gateways can apply policies per resource
- CDNs can cache configuration responses

### 6. Hypermedia (HATEOAS - Partial)

While full HATEOAS is not implemented (to maintain v2raysocks compatibility), we provide:

**Link headers in responses:**
```http
Link: </api/v1/nodes/1/users>; rel="related"
Link: </api/v1/nodes/1/traffic>; rel="related"
```

**Documentation links in error responses:**
```json
{
  "data": null,
  "ret": 0,
  "msg": "API deprecated. See: https://docs.orris.example.com/api-migration"
}
```

### 7. Caching

**Proper cache control headers:**
```http
ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4"
Cache-Control: private, max-age=60
Last-Modified: Wed, 21 Oct 2024 07:28:00 GMT
```

**Conditional requests:**
- `If-None-Match` for ETag validation
- `If-Modified-Since` for timestamp validation
- `304 Not Modified` responses when unchanged

### 8. Versioning

**URL-based versioning:**
- `/api/v1/nodes/{id}/config` - Version 1
- `/api/v2/nodes/{id}/config` - Future version 2

**Benefits:**
- Clear version boundaries
- Multiple versions can coexist
- Gradual migration path
- No breaking changes for existing clients

---

## Implementation Checklist

### Backend Development

- [ ] Create new route group `/api/v1/nodes`
- [ ] Implement `GET /api/v1/nodes/{id}/config` endpoint
- [ ] Implement `GET /api/v1/nodes/{id}/users` endpoint
- [ ] Implement `POST /api/v1/nodes/{id}/traffic` endpoint
- [ ] Implement `POST /api/v1/nodes/{id}/status` endpoint
- [ ] Implement `POST /api/v1/nodes/{id}/online-users` endpoint
- [ ] Create `X-Node-Token` authentication middleware
- [ ] Add ETag generation and validation
- [ ] Implement proper HTTP status code responses
- [ ] Add request validation for all endpoints
- [ ] Write unit tests (coverage >= 80%)
- [ ] Write integration tests

### Documentation

- [ ] Update Swagger/OpenAPI specifications
- [ ] Create migration guide for node operators
- [ ] Update API reference documentation
- [ ] Create comparison table (legacy vs RESTful)
- [ ] Write code examples (curl, Python, Go)
- [ ] Document authentication changes
- [ ] Document error codes and messages

### Testing

- [ ] Manual testing with curl
- [ ] Integration testing with XrayR
- [ ] Load testing (concurrent requests)
- [ ] Security testing (authentication, authorization)
- [ ] Performance benchmarking
- [ ] Caching validation (ETag)

### Deployment

- [ ] Deploy RESTful API to staging
- [ ] Test with sample node instances
- [ ] Deploy to production
- [ ] Monitor metrics (request counts, latency, errors)
- [ ] Add deprecation warnings to legacy API
- [ ] Communicate migration timeline to users

### Migration Support

- [ ] Create migration FAQ
- [ ] Set up support channel for migration questions
- [ ] Track migration progress (% of nodes migrated)
- [ ] Send reminder emails at key milestones
- [ ] Provide migration assistance for large node operators

### Deprecation

- [ ] Add `Deprecated` and `Sunset` headers to legacy API
- [ ] Monitor legacy API usage (should trend to zero)
- [ ] Final migration deadline communication
- [ ] Remove legacy API code
- [ ] Return `410 Gone` for legacy endpoints
- [ ] Clean up documentation references

---

## Conclusion

This API redesign brings the Node Backend API in line with RESTful principles and modern HTTP best practices. The migration strategy ensures a smooth transition with minimal disruption to existing node operators.

### Key Benefits

1. **Better Architecture** - Resource-oriented, self-documenting URLs
2. **Enhanced Security** - Header-based authentication, no token leakage
3. **Improved Performance** - HTTP caching with ETag support
4. **Future-Proof** - URL versioning allows evolution without breaking changes
5. **Consistency** - Aligns with existing Orris API patterns
6. **Developer Experience** - Intuitive, discoverable, well-documented

### Success Metrics

- **Migration Rate:** 100% of nodes migrated within 6 months
- **Error Rate:** < 1% on new RESTful API
- **Performance:** Response time <= legacy API baseline
- **Developer Satisfaction:** Positive feedback from node operators
- **Security:** Zero token leakage incidents

### Next Steps

1. Review and approve this design document
2. Create implementation tickets
3. Begin Phase 1 development
4. Schedule migration kickoff meeting with stakeholders

---

## Appendix

### Glossary

- **RESTful API** - API design following REST architectural principles
- **ETag** - HTTP header for cache validation (Entity Tag)
- **v2raysocks** - Protocol specification for proxy node communication
- **XrayR** - Xray node backend compatible with v2raysocks
- **HATEOAS** - Hypermedia As The Engine Of Application State

### References

- [REST API Design Best Practices](https://restfulapi.net/)
- [HTTP/1.1 Specification](https://www.rfc-editor.org/rfc/rfc7231)
- [v2raysocks Protocol Documentation](https://github.com/v2raysocks/v2raysocks)
- [OpenAPI Specification](https://swagger.io/specification/)

### Document Information

- **Version:** 1.0
- **Last Updated:** 2025-11-12
- **Author:** Claude (AI Assistant)
- **Status:** Draft for Review
