# Agent API 使用指南

## 📌 概述

Agent API 用于节点代理程序（如 XrayR）与面板后端通信。所有 Agent API 都需要使用 `X-Node-Token` header 进行身份认证。

**响应格式**: 所有 Agent API 使用标准 RESTful 响应格式（与管理端 API 统一）。

## 🔑 认证方式

Agent API 使用自定义 Header 认证：
- **Header 名称**: `X-Node-Token`
- **Token 类型**: Node Token（节点专用令牌）
- **安全定义**: `NodeToken` (在 Swagger 中定义)

---

## 🚀 快速开始

### 步骤 1: 生成 Node Token

首先需要通过管理端 API 为节点生成 token。

#### 使用 Swagger UI:

1. 访问 Swagger UI: `http://localhost:8080/swagger/index.html`
2. 找到 `nodes` 标签下的 `POST /nodes/{id}/token` 接口
3. 点击右上角 🔓 **Authorize** 按钮
4. 在 `Bearer` 输入框中输入管理员 JWT token:
   ```
   Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
   ```
5. 点击 **Authorize** → **Close**
6. 执行 `POST /nodes/{id}/token` 接口（将 `{id}` 替换为节点 ID，例如 `1`）
7. 响应示例：
   ```json
   {
     "success": true,
     "message": "Token generated successfully",
     "data": {
       "token": "node_abc123def456..."
     }
   }
   ```
8. **复制** 返回的 `data.token` 值

#### 使用 cURL:

```bash
# 1. 先登录获取管理员 token
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your_password"
  }'

# 2. 使用管理员 token 生成节点 token
curl -X POST http://localhost:8080/nodes/1/token \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN"
```

---

### 步骤 2: 在 Swagger 中配置 Node Token

1. 在 Swagger UI 页面，点击右上角 🔓 **Authorize** 按钮
2. 找到 **NodeToken (apiKey)** 部分
3. 在 **Value** 输入框中输入刚才生成的 node token:
   ```
   node_abc123def456...
   ```
   ⚠️ **注意**: 这里**不需要**加 `Bearer` 前缀，直接输入原始 token
4. 点击 **Authorize**
5. 点击 **Close**

---

### 步骤 3: 测试 Agent API

现在可以测试 Agent API 了！

#### 在 Swagger UI 中测试 `/agents/{id}/config`:

1. 找到 `agent-v1` 标签
2. 展开 `GET /agents/{id}/config`
3. 点击 **Try it out**
4. 输入参数:
   - `id`: 节点 ID（例如 `1`）
   - `node_type` (可选): `shadowsocks` 或 `trojan`
5. 点击 **Execute**
6. 查看响应结果

#### 使用 cURL 测试:

```bash
curl -X GET "http://localhost:8080/agents/1/config" \
  -H "X-Node-Token: node_abc123def456..."
```

---

## 📋 所有 Agent API 端点

| 方法 | 路径 | 描述 | 标签 |
|------|------|------|------|
| `GET` | `/agents/{id}/config` | 获取节点配置 | agent-v1 |
| `GET` | `/agents/{id}/users` | 获取授权用户列表 | agent-v1 |
| `POST` | `/agents/{id}/traffic` | 上报用户流量数据 | agent-v1 |
| `PUT` | `/agents/{id}/status` | 更新节点系统状态 | agent-v1 |
| `PUT` | `/agents/{id}/online-users` | 更新在线用户列表 | agent-v1 |

---

## 🔧 完整示例

### 1. 获取节点配置

**请求:**
```bash
curl -X GET "http://localhost:8080/agents/1/config?node_type=shadowsocks" \
  -H "X-Node-Token: node_abc123def456..." \
  -H "Content-Type: application/json"
```

**响应 (200 OK):**
```json
{
  "success": true,
  "message": "node configuration retrieved successfully",
  "data": {
    "node_id": 1,
    "server_port": 443,
    "encryption": "aes-256-gcm",
    "password": "your_password",
    ...
  }
}
```

### 2. 获取用户列表

**请求:**
```bash
curl -X GET "http://localhost:8080/agents/1/users" \
  -H "X-Node-Token: node_abc123def456..." \
  -H "Content-Type: application/json"
```

**响应 (200 OK):**
```json
{
  "success": true,
  "message": "user list retrieved successfully",
  "data": [
    {
      "user_id": 100,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com"
    },
    {
      "user_id": 101,
      "uuid": "660e8400-e29b-41d4-a716-446655440001",
      "email": "user2@example.com"
    }
  ]
}
```

### 3. 上报流量数据

**请求:**
```bash
curl -X POST "http://localhost:8080/agents/1/traffic" \
  -H "X-Node-Token: node_abc123def456..." \
  -H "Content-Type: application/json" \
  -d '[
    {
      "user_id": 100,
      "upload": 1024000,
      "download": 2048000
    },
    {
      "user_id": 101,
      "upload": 512000,
      "download": 1024000
    }
  ]'
```

**响应 (200 OK):**
```json
{
  "success": true,
  "message": "traffic reported successfully",
  "data": {
    "users_updated": 2
  }
}
```

### 4. 上报节点状态

**请求:**
```bash
curl -X PUT "http://localhost:8080/agents/1/status" \
  -H "X-Node-Token: node_abc123def456..." \
  -H "Content-Type: application/json" \
  -d '{
    "cpu": 45.5,
    "mem": 60.2,
    "disk": 75.0,
    "net_speed_in": "100MB/s",
    "net_speed_out": "80MB/s",
    "uptime": 86400
  }'
```

**响应 (200 OK):**
```json
{
  "success": true,
  "message": "status updated successfully",
  "data": {
    "status": "ok"
  }
}
```

### 5. 上报在线用户

**请求:**
```bash
curl -X PUT "http://localhost:8080/agents/1/online-users" \
  -H "X-Node-Token: node_abc123def456..." \
  -H "Content-Type: application/json" \
  -d '{
    "users": [
      {
        "user_id": 100,
        "ip": "192.168.1.100"
      },
      {
        "user_id": 101,
        "ip": "192.168.1.101"
      }
    ]
  }'
```

**响应 (200 OK):**
```json
{
  "success": true,
  "message": "online users updated successfully",
  "data": {
    "online_count": 2
  }
}
```

---

## 📋 标准 RESTful 响应格式

所有 Agent API 使用统一的 RESTful 响应格式，与管理端 API 保持一致。

### ✅ 成功响应格式

```json
{
  "success": true,
  "message": "操作成功的描述信息",
  "data": {
    // 实际返回的数据
  }
}
```

**字段说明:**
- `success` (boolean): 请求是否成功
- `message` (string): 人类可读的操作描述
- `data` (object/array): 实际的业务数据

### ❌ 错误响应格式

```json
{
  "success": false,
  "error": {
    "type": "validation_error",
    "message": "invalid node_id parameter",
    "details": "node_id must be a valid integer"
  }
}
```

**字段说明:**
- `success` (boolean): 固定为 `false`
- `error.type` (string): 错误类型（如 `validation_error`, `not_found`, `internal_error`）
- `error.message` (string): 错误消息
- `error.details` (string, 可选): 详细错误信息

### 📊 HTTP 状态码

| 状态码 | 说明 | 示例场景 |
|--------|------|---------|
| `200 OK` | 请求成功 | 获取配置、上报成功 |
| `400 Bad Request` | 请求参数错误 | 无效的 node_id、JSON 格式错误 |
| `401 Unauthorized` | 未认证 | 缺少或无效的 X-Node-Token |
| `404 Not Found` | 资源不存在 | 节点不存在 |
| `500 Internal Server Error` | 服务器内部错误 | 数据库错误、未知异常 |

---

## 🛡️ 安全注意事项

1. **Token 保密**: Node Token 应该被视为敏感信息，不要暴露在公共代码库或日志中
2. **HTTPS**: 生产环境必须使用 HTTPS 传输 token
3. **Token 轮换**: 定期重新生成 node token 以提高安全性
4. **访问控制**: 确保只有授权的节点程序能访问 Agent API

---

## 🔍 故障排查

### 错误: 401 Unauthorized

**原因**: Token 未提供或无效

**解决方案**:
1. 确认 header 名称是 `X-Node-Token`（不是 `Authorization`）
2. 确认 token 没有过期
3. 重新生成 token

### 错误: 400 Invalid node_id parameter

**原因**: 节点 ID 格式错误

**解决方案**:
- 确保 `{id}` 是有效的数字（如 `1`, `2`, `100`）
- 不要使用字母或特殊字符

### 错误: 404 Node not found

**原因**: 节点不存在或已被删除

**解决方案**:
1. 通过 `GET /nodes` 查看可用节点列表
2. 确认节点 ID 是否正确
3. 检查节点是否被删除

---

## 📚 相关文档

- [API 变更文档](./API_CHANGES.md)
- [RESTful 设计文档](./API_REDESIGN_RESTFUL.md)
- [Swagger 规范](./swagger.yaml)

---

## 💡 提示

### Swagger UI 中的两种认证方式

系统支持两种认证方式，用于不同的 API：

| 认证方式 | Header | 用途 | API 标签 |
|---------|--------|------|---------|
| **Bearer** | `Authorization: Bearer <JWT>` | 管理端、用户端 API | `nodes`, `users`, `subscriptions` 等 |
| **NodeToken** | `X-Node-Token: <token>` | Agent API（节点对接） | `agent-v1` |

### Postman 配置

如果使用 Postman:

1. 创建新请求
2. URL: `http://localhost:8080/agents/1/config`
3. Headers 标签页:
   - Key: `X-Node-Token`
   - Value: `node_abc123def456...`
4. Send

### XrayR 配置示例

⚠️ **重要提示**: XrayR 等现有客户端可能需要适配新的 RESTful 响应格式。

如果使用标准 XrayR，可能需要编写适配器来转换响应格式，或者使用支持标准 RESTful 格式的客户端。

**标准配置示例:**
```yaml
ApiConfig:
  ApiHost: "http://localhost:8080"
  ApiKey: "node_abc123def456..."  # 这里填写生成的 node token
  NodeID: 1
  NodeType: shadowsocks
  # 新增: 响应格式类型
  ResponseFormat: "restful"  # 或 "v2raysocks" (取决于客户端支持)
```

**响应格式对比:**

```diff
# 旧格式 (v2raysocks)
- {"data": {...}}
- {"ret": 0, "msg": "error"}

# 新格式 (RESTful)
+ {"success": true, "message": "...", "data": {...}}
+ {"success": false, "error": {"type": "...", "message": "..."}}
```

---

## ✅ 测试清单

- [ ] 能够成功生成 node token
- [ ] 能够在 Swagger UI 中配置 NodeToken 认证
- [ ] 能够成功调用 `GET /agents/{id}/config`
- [ ] 能够成功调用 `GET /agents/{id}/users`
- [ ] 能够成功上报流量数据
- [ ] 能够成功上报节点状态
- [ ] 能够处理认证失败的情况

---

**最后更新**: 2025-11-12
**版本**: v2.0 (RESTful 格式)

---

## 🔄 版本历史

### v2.0 (2025-11-12)
- ✅ 改为标准 RESTful 响应格式
- ✅ 与管理端 API 格式统一
- ✅ 更丰富的错误信息（type、message、details）
- ✅ 更清晰的成功/失败语义
- ✅ **完全移除旧版 `/api/node` API**

### v1.0 (2025-11-12)
- ✅ 初始版本，使用 v2raysocks 兼容格式
- ✅ 支持 XrayR 直接对接
- ❌ 已废弃并移除

---

## ⚠️ 重要提示

**旧版 `/api/node` API 已完全移除**，请参考 [迁移指南](./API_MIGRATION_GUIDE.md) 进行升级。
