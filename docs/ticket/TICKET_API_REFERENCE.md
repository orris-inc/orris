# Ticket System - API Reference
# 工单系统 - API 参考文档

## 目录

1. [概述](#1-概述)
2. [认证与授权](#2-认证与授权)
3. [工单管理 API](#3-工单管理-api)
4. [评论管理 API](#4-评论管理-api)
5. [统计与报表 API](#5-统计与报表-api)
6. [错误码定义](#6-错误码定义)
7. [使用示例](#7-使用示例)

---

## 1. 概述

### 1.1 Base URL

```
生产环境: https://api.example.com/v1
开发环境: http://localhost:8080/v1
```

### 1.2 请求格式

所有请求必须：
- 使用 `Content-Type: application/json`
- 包含认证 Token（除公开接口）
- 遵循 RESTful 规范

### 1.3 响应格式

**成功响应**:

```json
{
  "success": true,
  "message": "Operation successful",
  "data": {
    "id": 1,
    "...": "..."
  }
}
```

**分页响应**:

```json
{
  "success": true,
  "message": "",
  "data": [...],
  "pagination": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

**错误响应**:

```json
{
  "success": false,
  "message": "Error message",
  "error_code": "VALIDATION_ERROR",
  "details": {
    "field": "error detail"
  }
}
```

---

## 2. 认证与授权

### 2.1 认证方式

使用 Bearer Token 认证：

```http
Authorization: Bearer <JWT_TOKEN>
```

### 2.2 权限要求

| 角色 | 权限 |
|------|------|
| User | 创建工单、查看自己的工单、添加评论 |
| Agent | User 权限 + 分配工单、更新状态、关闭工单、查看分配给自己的工单 |
| Admin | 所有权限 + 删除工单、查看所有工单 |

---

## 3. 工单管理 API

### 3.1 创建工单

创建一个新的支持工单。

**Endpoint**: `POST /tickets`

**权限**: `ticket:create` (所有角色)

**请求体**:

```json
{
  "title": "无法登录账号",
  "description": "我尝试使用正确的密码登录，但系统提示密码错误。我已经重置过密码，但问题依然存在。",
  "category": "technical",
  "priority": "high",
  "tags": ["login", "password"],
  "metadata": {
    "browser": "Chrome 120",
    "os": "Windows 11"
  }
}
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 工单标题，1-200字符 |
| description | string | 是 | 详细描述，最多5000字符 |
| category | string | 是 | 分类：technical/account/billing/feature/complaint/other |
| priority | string | 是 | 优先级：low/medium/high/urgent |
| tags | array | 否 | 标签列表 |
| metadata | object | 否 | 自定义元数据 |

**响应**: `201 Created`

```json
{
  "success": true,
  "message": "Ticket created successfully",
  "data": {
    "ticket_id": 123,
    "number": "T-20241023-0001",
    "title": "无法登录账号",
    "status": "new",
    "priority": "high",
    "sla_due_time": "2024-10-24T10:30:00Z",
    "created_at": "2024-10-23T10:30:00Z"
  }
}
```

**cURL 示例**:

```bash
curl -X POST https://api.example.com/v1/tickets \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "无法登录账号",
    "description": "我尝试使用正确的密码登录...",
    "category": "technical",
    "priority": "high"
  }'
```

---

### 3.2 获取工单详情

获取指定工单的详细信息。

**Endpoint**: `GET /tickets/:id`

**权限**: `ticket:read` + 可见性检查

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| id | integer | 工单ID |

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "id": 123,
    "number": "T-20241023-0001",
    "title": "无法登录账号",
    "description": "我尝试使用正确的密码登录...",
    "category": "technical",
    "priority": "high",
    "status": "in_progress",
    "creator": {
      "id": 1,
      "name": "张三",
      "email": "zhangsan@example.com"
    },
    "assignee": {
      "id": 5,
      "name": "客服李四",
      "email": "lisi@example.com"
    },
    "tags": ["login", "password"],
    "metadata": {
      "browser": "Chrome 120",
      "os": "Windows 11"
    },
    "sla_due_time": "2024-10-24T10:30:00Z",
    "response_time": "2024-10-23T11:15:00Z",
    "resolved_time": null,
    "created_at": "2024-10-23T10:30:00Z",
    "updated_at": "2024-10-23T14:20:00Z",
    "closed_at": null,
    "comments_count": 3,
    "is_overdue": false
  }
}
```

**cURL 示例**:

```bash
curl -X GET https://api.example.com/v1/tickets/123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 3.3 列出工单

获取工单列表，支持过滤和分页。

**Endpoint**: `GET /tickets`

**权限**: `ticket:read`

**查询参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | integer | 否 | 1 | 页码 |
| page_size | integer | 否 | 20 | 每页数量（1-100） |
| status | string | 否 | - | 状态过滤：new/open/in_progress/pending/resolved/closed/reopened |
| priority | string | 否 | - | 优先级过滤：low/medium/high/urgent |
| category | string | 否 | - | 分类过滤 |
| assignee_id | integer | 否 | - | 处理人ID |
| creator_id | integer | 否 | - | 创建人ID（仅Admin） |
| overdue | boolean | 否 | - | 是否超期：true/false |
| tags | string | 否 | - | 标签过滤，逗号分隔 |
| sort_by | string | 否 | created_at | 排序字段：created_at/priority/sla_due_time |
| sort_order | string | 否 | desc | 排序方向：asc/desc |
| search | string | 否 | - | 全文搜索（标题和描述） |

**响应**: `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": 123,
      "number": "T-20241023-0001",
      "title": "无法登录账号",
      "category": "technical",
      "priority": "high",
      "status": "in_progress",
      "creator_name": "张三",
      "assignee_name": "客服李四",
      "created_at": "2024-10-23T10:30:00Z",
      "updated_at": "2024-10-23T14:20:00Z",
      "is_overdue": false
    },
    {
      "id": 124,
      "number": "T-20241023-0002",
      "title": "账单问题",
      "category": "billing",
      "priority": "medium",
      "status": "new",
      "creator_name": "王五",
      "assignee_name": null,
      "created_at": "2024-10-23T11:00:00Z",
      "updated_at": "2024-10-23T11:00:00Z",
      "is_overdue": false
    }
  ],
  "pagination": {
    "total": 45,
    "page": 1,
    "page_size": 20,
    "total_pages": 3
  }
}
```

**cURL 示例**:

```bash
# 获取所有高优先级的未解决工单
curl -X GET "https://api.example.com/v1/tickets?priority=high&status=in_progress&page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 获取分配给我的工单
curl -X GET "https://api.example.com/v1/tickets?assignee_id=5" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 搜索包含"登录"的工单
curl -X GET "https://api.example.com/v1/tickets?search=登录" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 3.4 分配工单

将工单分配给指定的处理人员。

**Endpoint**: `POST /tickets/:id/assign`

**权限**: `ticket:assign` (Agent, Admin)

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| id | integer | 工单ID |

**请求体**:

```json
{
  "assignee_id": 5
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "message": "Ticket assigned successfully",
  "data": {
    "ticket_id": 123,
    "assignee_id": 5,
    "status": "open",
    "updated_at": "2024-10-23T14:20:00Z"
  }
}
```

**cURL 示例**:

```bash
curl -X POST https://api.example.com/v1/tickets/123/assign \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"assignee_id": 5}'
```

---

### 3.5 更新工单状态

更改工单的状态。

**Endpoint**: `PATCH /tickets/:id/status`

**权限**: `ticket:update` + 可见性检查

**请求体**:

```json
{
  "status": "in_progress",
  "reason": "开始处理此问题"
}
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | string | 是 | 目标状态 |
| reason | string | 否 | 状态变更原因 |

**响应**: `200 OK`

```json
{
  "success": true,
  "message": "Ticket status updated successfully",
  "data": {
    "ticket_id": 123,
    "old_status": "open",
    "new_status": "in_progress",
    "updated_at": "2024-10-23T14:25:00Z"
  }
}
```

**cURL 示例**:

```bash
curl -X PATCH https://api.example.com/v1/tickets/123/status \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_progress",
    "reason": "开始处理此问题"
  }'
```

---

### 3.6 更新工单优先级

更改工单的优先级。

**Endpoint**: `PATCH /tickets/:id/priority`

**权限**: `ticket:update` (Agent, Admin)

**请求体**:

```json
{
  "priority": "urgent"
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "message": "Ticket priority updated successfully",
  "data": {
    "ticket_id": 123,
    "old_priority": "high",
    "new_priority": "urgent",
    "sla_due_time": "2024-10-23T15:30:00Z",
    "updated_at": "2024-10-23T14:30:00Z"
  }
}
```

**cURL 示例**:

```bash
curl -X PATCH https://api.example.com/v1/tickets/123/priority \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"priority": "urgent"}'
```

---

### 3.7 关闭工单

关闭一个工单。

**Endpoint**: `POST /tickets/:id/close`

**权限**: `ticket:close` (Agent, Admin)

**请求体**:

```json
{
  "reason": "问题已解决，用户确认满意"
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "message": "Ticket closed successfully",
  "data": {
    "ticket_id": 123,
    "status": "closed",
    "closed_at": "2024-10-23T16:00:00Z"
  }
}
```

**cURL 示例**:

```bash
curl -X POST https://api.example.com/v1/tickets/123/close \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "问题已解决，用户确认满意"}'
```

---

### 3.8 重开工单

重新打开一个已关闭或已解决的工单。

**Endpoint**: `POST /tickets/:id/reopen`

**权限**: `ticket:reopen` (创建人, Agent, Admin)

**请求体**:

```json
{
  "reason": "问题仍然存在，需要重新处理"
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "message": "Ticket reopened successfully",
  "data": {
    "ticket_id": 123,
    "status": "reopened",
    "updated_at": "2024-10-24T09:00:00Z"
  }
}
```

**cURL 示例**:

```bash
curl -X POST https://api.example.com/v1/tickets/123/reopen \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "问题仍然存在"}'
```

---

### 3.9 删除工单

删除一个工单（仅管理员）。

**Endpoint**: `DELETE /tickets/:id`

**权限**: `ticket:delete` (仅 Admin)

**响应**: `204 No Content`

**cURL 示例**:

```bash
curl -X DELETE https://api.example.com/v1/tickets/123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 4. 评论管理 API

### 4.1 添加评论

为工单添加评论。

**Endpoint**: `POST /tickets/:id/comments`

**权限**: `ticket:comment` + 可见性检查

**请求体**:

```json
{
  "content": "我已经检查了数据库日志，发现是密码加密方式升级导致的问题。我会在今天下午修复。",
  "is_internal": false
}
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 是 | 评论内容，最多10000字符 |
| is_internal | boolean | 否 | 是否为内部备注（仅Agent/Admin可见），默认false |

**响应**: `201 Created`

```json
{
  "success": true,
  "message": "Comment added successfully",
  "data": {
    "comment_id": 456,
    "ticket_id": 123,
    "user": {
      "id": 5,
      "name": "客服李四"
    },
    "content": "我已经检查了数据库日志...",
    "is_internal": false,
    "created_at": "2024-10-23T15:30:00Z"
  }
}
```

**cURL 示例**:

```bash
# 添加普通评论
curl -X POST https://api.example.com/v1/tickets/123/comments \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "问题已解决，请确认",
    "is_internal": false
  }'

# 添加内部备注
curl -X POST https://api.example.com/v1/tickets/123/comments \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "此用户是VIP客户，优先处理",
    "is_internal": true
  }'
```

---

### 4.2 获取工单评论

获取指定工单的所有评论。

**Endpoint**: `GET /tickets/:id/comments`

**权限**: `ticket:read` + 可见性检查

**查询参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | integer | 否 | 1 | 页码 |
| page_size | integer | 否 | 50 | 每页数量 |
| include_internal | boolean | 否 | false | 是否包含内部备注（仅Agent/Admin） |

**响应**: `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": 456,
      "ticket_id": 123,
      "user": {
        "id": 1,
        "name": "张三",
        "avatar": "https://..."
      },
      "content": "谢谢，问题已解决！",
      "is_internal": false,
      "created_at": "2024-10-23T16:00:00Z"
    },
    {
      "id": 455,
      "ticket_id": 123,
      "user": {
        "id": 5,
        "name": "客服李四"
      },
      "content": "已修复，请重新登录测试",
      "is_internal": false,
      "created_at": "2024-10-23T15:45:00Z"
    }
  ],
  "pagination": {
    "total": 5,
    "page": 1,
    "page_size": 50,
    "total_pages": 1
  }
}
```

**cURL 示例**:

```bash
curl -X GET "https://api.example.com/v1/tickets/123/comments?include_internal=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 4.3 更新评论

更新评论内容（仅评论作者）。

**Endpoint**: `PATCH /tickets/:ticket_id/comments/:comment_id`

**权限**: 评论作者

**请求体**:

```json
{
  "content": "更新后的评论内容"
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "message": "Comment updated successfully",
  "data": {
    "comment_id": 456,
    "content": "更新后的评论内容",
    "updated_at": "2024-10-23T16:10:00Z"
  }
}
```

---

### 4.4 删除评论

删除评论（仅评论作者或Admin）。

**Endpoint**: `DELETE /tickets/:ticket_id/comments/:comment_id`

**权限**: 评论作者或Admin

**响应**: `204 No Content`

---

## 5. 统计与报表 API

### 5.1 获取工单统计

获取工单的统计信息。

**Endpoint**: `GET /tickets/stats`

**权限**: `ticket:read`

**查询参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 (YYYY-MM-DD) |
| end_date | string | 否 | 结束日期 (YYYY-MM-DD) |
| assignee_id | integer | 否 | 按处理人过滤 |

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "total_tickets": 156,
    "by_status": {
      "new": 12,
      "open": 8,
      "in_progress": 25,
      "pending": 15,
      "resolved": 30,
      "closed": 66
    },
    "by_priority": {
      "low": 45,
      "medium": 78,
      "high": 28,
      "urgent": 5
    },
    "by_category": {
      "technical": 89,
      "account": 34,
      "billing": 18,
      "feature": 10,
      "complaint": 3,
      "other": 2
    },
    "sla_metrics": {
      "average_response_time": "2h 15m",
      "average_resolution_time": "18h 30m",
      "sla_compliance_rate": 94.5,
      "overdue_tickets": 8
    }
  }
}
```

**cURL 示例**:

```bash
curl -X GET "https://api.example.com/v1/tickets/stats?start_date=2024-10-01&end_date=2024-10-31" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 5.2 获取我的工单统计

获取当前用户的工单统计。

**Endpoint**: `GET /tickets/my-stats`

**权限**: 已认证用户

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "created_by_me": {
      "total": 23,
      "open": 5,
      "resolved": 15,
      "closed": 3
    },
    "assigned_to_me": {
      "total": 12,
      "in_progress": 7,
      "pending": 3,
      "resolved": 2
    },
    "overdue_assigned": 2,
    "average_resolution_time": "15h 20m"
  }
}
```

---

### 5.3 获取 SLA 报表

获取 SLA 合规性报表（仅 Admin）。

**Endpoint**: `GET /tickets/sla-report`

**权限**: `ticket:read_all` (仅 Admin)

**查询参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 是 | 开始日期 |
| end_date | string | 是 | 结束日期 |
| group_by | string | 否 | 分组维度：assignee/priority/category |

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "period": {
      "start": "2024-10-01",
      "end": "2024-10-31"
    },
    "overall": {
      "total_tickets": 156,
      "met_response_sla": 148,
      "met_resolution_sla": 142,
      "response_sla_rate": 94.9,
      "resolution_sla_rate": 91.0
    },
    "by_priority": [
      {
        "priority": "urgent",
        "total": 5,
        "met_sla": 5,
        "sla_rate": 100.0
      },
      {
        "priority": "high",
        "total": 28,
        "met_sla": 26,
        "sla_rate": 92.9
      }
    ]
  }
}
```

---

## 6. 错误码定义

### 6.1 HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 201 | 资源创建成功 |
| 204 | 删除成功，无返回内容 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 422 | 验证失败 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |

### 6.2 业务错误码

| 错误码 | 说明 |
|--------|------|
| `VALIDATION_ERROR` | 请求参数验证失败 |
| `UNAUTHORIZED` | 未认证或认证失败 |
| `FORBIDDEN` | 无权限访问此资源 |
| `NOT_FOUND` | 资源不存在 |
| `TICKET_NOT_FOUND` | 工单不存在 |
| `INVALID_STATUS_TRANSITION` | 无效的状态转换 |
| `ASSIGNEE_NOT_FOUND` | 处理人不存在 |
| `COMMENT_TOO_LONG` | 评论内容过长 |
| `SLA_VIOLATED` | SLA 已违规 |
| `INTERNAL_ERROR` | 服务器内部错误 |

### 6.3 错误响应示例

```json
{
  "success": false,
  "message": "Validation failed",
  "error_code": "VALIDATION_ERROR",
  "details": {
    "title": "title is required",
    "description": "description must be at least 10 characters"
  }
}
```

---

## 7. 使用示例

### 7.1 完整工单处理流程

```bash
# 1. 用户创建工单
TICKET_ID=$(curl -X POST https://api.example.com/v1/tickets \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "无法访问服务",
    "description": "从今天上午开始无法访问服务",
    "category": "technical",
    "priority": "high"
  }' | jq -r '.data.ticket_id')

# 2. Agent 分配工单给自己
curl -X POST "https://api.example.com/v1/tickets/$TICKET_ID/assign" \
  -H "Authorization: Bearer AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"assignee_id": 5}'

# 3. Agent 更新状态为"处理中"
curl -X PATCH "https://api.example.com/v1/tickets/$TICKET_ID/status" \
  -H "Authorization: Bearer AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_progress",
    "reason": "开始调查问题"
  }'

# 4. Agent 添加评论
curl -X POST "https://api.example.com/v1/tickets/$TICKET_ID/comments" \
  -H "Authorization: Bearer AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "已发现问题原因，正在修复",
    "is_internal": false
  }'

# 5. Agent 更新状态为"已解决"
curl -X PATCH "https://api.example.com/v1/tickets/$TICKET_ID/status" \
  -H "Authorization: Bearer AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "resolved",
    "reason": "问题已修复"
  }'

# 6. 用户确认并关闭工单
curl -X POST "https://api.example.com/v1/tickets/$TICKET_ID/close" \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "问题已解决，感谢"
  }'
```

### 7.2 查询工单示例

```bash
# 查看所有高优先级的处理中工单
curl -X GET "https://api.example.com/v1/tickets?priority=high&status=in_progress" \
  -H "Authorization: Bearer TOKEN"

# 查看分配给我且超期的工单
curl -X GET "https://api.example.com/v1/tickets?assignee_id=5&overdue=true" \
  -H "Authorization: Bearer TOKEN"

# 全文搜索包含"登录"的工单
curl -X GET "https://api.example.com/v1/tickets?search=登录" \
  -H "Authorization: Bearer TOKEN"

# 按 SLA 到期时间排序
curl -X GET "https://api.example.com/v1/tickets?sort_by=sla_due_time&sort_order=asc" \
  -H "Authorization: Bearer TOKEN"
```

### 7.3 批量操作示例

```bash
# 获取所有新工单并批量分配
TICKETS=$(curl -X GET "https://api.example.com/v1/tickets?status=new" \
  -H "Authorization: Bearer TOKEN" | jq -r '.data[].id')

for TICKET_ID in $TICKETS; do
  curl -X POST "https://api.example.com/v1/tickets/$TICKET_ID/assign" \
    -H "Authorization: Bearer TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"assignee_id": 5}'
done
```

### 7.4 使用 Python SDK 示例

```python
import requests

class TicketClient:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.headers = {
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        }

    def create_ticket(self, title, description, category, priority):
        url = f'{self.base_url}/tickets'
        data = {
            'title': title,
            'description': description,
            'category': category,
            'priority': priority
        }
        response = requests.post(url, json=data, headers=self.headers)
        return response.json()

    def get_ticket(self, ticket_id):
        url = f'{self.base_url}/tickets/{ticket_id}'
        response = requests.get(url, headers=self.headers)
        return response.json()

    def add_comment(self, ticket_id, content, is_internal=False):
        url = f'{self.base_url}/tickets/{ticket_id}/comments'
        data = {
            'content': content,
            'is_internal': is_internal
        }
        response = requests.post(url, json=data, headers=self.headers)
        return response.json()

# 使用示例
client = TicketClient('https://api.example.com/v1', 'YOUR_TOKEN')

# 创建工单
result = client.create_ticket(
    title='测试工单',
    description='这是一个测试',
    category='technical',
    priority='medium'
)
print(f"Created ticket: {result['data']['number']}")

# 添加评论
ticket_id = result['data']['ticket_id']
client.add_comment(ticket_id, '这是一条评论')
```

### 7.5 使用 JavaScript/TypeScript SDK 示例

```typescript
class TicketAPI {
  private baseUrl: string;
  private token: string;

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl;
    this.token = token;
  }

  private async request(endpoint: string, options: RequestInit = {}) {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });
    return response.json();
  }

  async createTicket(data: CreateTicketRequest) {
    return this.request('/tickets', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async listTickets(params: ListTicketsParams) {
    const query = new URLSearchParams(params as any).toString();
    return this.request(`/tickets?${query}`);
  }

  async addComment(ticketId: number, content: string, isInternal = false) {
    return this.request(`/tickets/${ticketId}/comments`, {
      method: 'POST',
      body: JSON.stringify({ content, is_internal: isInternal }),
    });
  }
}

// 使用示例
const api = new TicketAPI('https://api.example.com/v1', 'YOUR_TOKEN');

// 创建工单
const ticket = await api.createTicket({
  title: '测试工单',
  description: '这是一个测试',
  category: 'technical',
  priority: 'medium',
});

// 列出工单
const tickets = await api.listTickets({
  status: 'in_progress',
  page: 1,
  page_size: 20,
});

// 添加评论
await api.addComment(ticket.data.ticket_id, '处理中');
```

---

## 8. Webhook 通知

### 8.1 配置 Webhook

在系统设置中配置 Webhook URL，接收工单事件通知。

**事件类型**:

- `ticket.created` - 工单创建
- `ticket.assigned` - 工单分配
- `ticket.status_changed` - 状态变更
- `ticket.closed` - 工单关闭
- `ticket.reopened` - 工单重开
- `ticket.comment_added` - 新增评论
- `ticket.sla_violated` - SLA 违规

### 8.2 Webhook Payload 示例

```json
{
  "event": "ticket.created",
  "timestamp": "2024-10-23T10:30:00Z",
  "data": {
    "ticket_id": 123,
    "number": "T-20241023-0001",
    "title": "无法登录账号",
    "status": "new",
    "priority": "high",
    "creator": {
      "id": 1,
      "name": "张三",
      "email": "zhangsan@example.com"
    },
    "url": "https://app.example.com/tickets/123"
  }
}
```

### 8.3 验证 Webhook 签名

每个 Webhook 请求包含 `X-Signature` 头部，用于验证请求来源：

```python
import hmac
import hashlib

def verify_webhook(payload, signature, secret):
    computed = hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(computed, signature)
```

---

## 9. 速率限制

### 9.1 限制规则

| 用户类型 | 限制 |
|----------|------|
| 未认证 | 10 req/min |
| User | 60 req/min |
| Agent | 120 req/min |
| Admin | 无限制 |

### 9.2 响应头

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1698076800
```

### 9.3 超限响应

```json
{
  "success": false,
  "message": "Rate limit exceeded",
  "error_code": "RATE_LIMIT_EXCEEDED",
  "details": {
    "retry_after": 30
  }
}
```

---

## 10. 版本管理

### 10.1 API 版本

当前版本: `v1`

使用 URL 路径版本控制: `https://api.example.com/v1/...`

### 10.2 废弃策略

- 新版本发布后，旧版本至少维护 6 个月
- 废弃功能会提前 3 个月通知
- 响应头包含 `X-API-Deprecated-At` 标识废弃时间

---

## 附录

### A. 完整字段映射表

| 领域字段 | API 字段 | 类型 | 说明 |
|----------|----------|------|------|
| ID | id | integer | 工单ID |
| Number | number | string | 工单号 |
| Title | title | string | 标题 |
| Description | description | string | 描述 |
| Category | category | string | 分类 |
| Priority | priority | string | 优先级 |
| Status | status | string | 状态 |
| CreatorID | creator_id | integer | 创建人ID |
| AssigneeID | assignee_id | integer | 处理人ID |
| Tags | tags | array | 标签 |
| Metadata | metadata | object | 元数据 |
| SLADueTime | sla_due_time | datetime | SLA到期时间 |
| ResponseTime | response_time | datetime | 首次响应时间 |
| ResolvedTime | resolved_time | datetime | 解决时间 |
| CreatedAt | created_at | datetime | 创建时间 |
| UpdatedAt | updated_at | datetime | 更新时间 |
| ClosedAt | closed_at | datetime | 关闭时间 |

### B. Postman Collection

导入以下 Postman Collection 快速开始测试：

[下载 Postman Collection](https://api.example.com/docs/postman/ticket-api.json)

### C. OpenAPI/Swagger 文档

访问交互式 API 文档：

```
https://api.example.com/docs/swagger-ui
```

---

**文档版本**: v1.0
**最后更新**: 2024-10-23
**维护者**: Orris Team
