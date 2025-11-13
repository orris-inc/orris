# API 迁移指南

## 📌 概述

本指南帮助你从旧版 `/api/node` API 迁移到新版 `/agents` RESTful API。

---

## 🚨 重要变更

### 已移除的 API

❌ **旧版 API 已完全移除**：
- `/api/node?act=config`
- `/api/node?act=user`
- `/api/node?act=submit`
- `/api/node?act=nodestatus`
- `/api/node?act=onlineusers`

✅ **请使用新版 RESTful API**：
- `GET /agents/{id}/config`
- `GET /agents/{id}/users`
- `POST /agents/{id}/traffic`
- `PUT /agents/{id}/status`
- `PUT /agents/{id}/online-users`

---

## 🔄 API 对照表

| 旧版 API | 新版 API | 说明 |
|---------|---------|------|
| `GET /api/node?act=config&node_id=1` | `GET /agents/1/config` | 获取节点配置 |
| `GET /api/node?act=user&node_id=1` | `GET /agents/1/users` | 获取用户列表 |
| `POST /api/node?act=submit` | `POST /agents/{id}/traffic` | 上报流量数据 |
| `POST /api/node?act=nodestatus` | `PUT /agents/{id}/status` | 上报节点状态 |
| `POST /api/node?act=onlineusers` | `PUT /agents/{id}/online-users` | 上报在线用户 |

---

## 📝 请求格式变化

### 1. URL 参数 → 路径参数

**旧版:**
```bash
GET /api/node?act=config&node_id=1&token=xxx
```

**新版:**
```bash
GET /agents/1/config
Header: X-Node-Token: xxx
```

### 2. Query 参数 → Header 认证

**旧版:**
```bash
GET /api/node?act=config&node_id=1&token=node_abc123
```

**新版:**
```bash
GET /agents/1/config
Header: X-Node-Token: node_abc123
```

### 3. POST 数据格式保持一致

流量上报的请求体格式保持不变：

```json
[
  {
    "user_id": 100,
    "upload": 1024000,
    "download": 2048000
  }
]
```

---

## 🔧 响应格式变化

### 旧版响应格式 (v2raysocks)

**成功:**
```json
{
  "data": {...}
}
```

**错误:**
```json
{
  "ret": 0,
  "msg": "error message"
}
```

### 新版响应格式 (RESTful)

**成功:**
```json
{
  "success": true,
  "message": "operation successful",
  "data": {...}
}
```

**错误:**
```json
{
  "success": false,
  "error": {
    "type": "validation_error",
    "message": "invalid parameter",
    "details": "..."
  }
}
```

---

## 💻 客户端代码迁移示例

### Go 客户端示例

**旧版代码:**
```go
type OldClient struct {
    BaseURL string
    Token   string
}

func (c *OldClient) GetConfig(nodeID int) (*Config, error) {
    url := fmt.Sprintf("%s/api/node?act=config&node_id=%d&token=%s",
        c.BaseURL, nodeID, c.Token)

    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data *Config `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    return result.Data, nil
}
```

**新版代码:**
```go
type NewClient struct {
    BaseURL string
    Token   string
}

func (c *NewClient) GetConfig(nodeID int) (*Config, error) {
    url := fmt.Sprintf("%s/agents/%d/config", c.BaseURL, nodeID)

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("X-Node-Token", c.Token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Success bool    `json:"success"`
        Message string  `json:"message"`
        Data    *Config `json:"data"`
        Error   *struct {
            Type    string `json:"type"`
            Message string `json:"message"`
            Details string `json:"details"`
        } `json:"error,omitempty"`
    }

    json.NewDecoder(resp.Body).Decode(&result)

    if !result.Success {
        return nil, fmt.Errorf("%s: %s", result.Error.Type, result.Error.Message)
    }

    return result.Data, nil
}
```

### Python 客户端示例

**旧版代码:**
```python
import requests

class OldClient:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.token = token

    def get_config(self, node_id):
        url = f"{self.base_url}/api/node"
        params = {
            "act": "config",
            "node_id": node_id,
            "token": self.token
        }
        resp = requests.get(url, params=params)
        return resp.json().get("data")
```

**新版代码:**
```python
import requests

class NewClient:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.token = token

    def get_config(self, node_id):
        url = f"{self.base_url}/agents/{node_id}/config"
        headers = {
            "X-Node-Token": self.token,
            "Content-Type": "application/json"
        }
        resp = requests.get(url, headers=headers)
        data = resp.json()

        if not data.get("success"):
            error = data.get("error", {})
            raise Exception(f"{error.get('type')}: {error.get('message')}")

        return data.get("data")
```

---

## 🔑 认证方式变化

### 旧版认证
```
Token 通过 URL 查询参数传递:
/api/node?token=xxx
```

### 新版认证
```
Token 通过 HTTP Header 传递:
X-Node-Token: xxx
```

**优势:**
- ✅ 更安全（不会出现在日志和浏览器历史中）
- ✅ 符合 RESTful 最佳实践
- ✅ 支持标准 HTTP 缓存策略

---

## 📊 迁移检查清单

- [ ] 更新客户端 URL：`/api/node` → `/agents`
- [ ] 更新认证方式：Query 参数 → HTTP Header
- [ ] 更新 HTTP 方法：GET/POST → GET/POST/PUT
- [ ] 更新响应解析：检查 `success` 字段
- [ ] 更新错误处理：解析 `error` 对象
- [ ] 测试所有 5 个 API 端点
- [ ] 更新配置文件中的 API 地址
- [ ] 更新文档和注释

---

## 🚀 分阶段迁移建议

### 阶段 1: 准备工作（1-2天）
1. 阅读本迁移指南
2. 了解新 API 的变化
3. 准备测试环境

### 阶段 2: 代码修改（2-3天）
1. 创建新版 API 客户端
2. 保留旧代码作为备份
3. 逐个替换 API 调用

### 阶段 3: 测试验证（2-3天）
1. 单元测试
2. 集成测试
3. 生产环境验证

### 阶段 4: 上线部署（1天）
1. 灰度发布
2. 监控日志
3. 回滚准备

---

## ⚠️ 常见问题

### Q1: 旧版 API 何时完全移除？
**A:** 旧版 `/api/node` API 已在当前版本完全移除。请尽快迁移到新版 `/agents` API。

### Q2: 能否同时支持两种格式？
**A:** 不支持。为了保持代码简洁和维护性，只保留新版 RESTful API。

### Q3: 如何测试新 API？
**A:**
1. 使用 Swagger UI: `http://localhost:8080/swagger/index.html`
2. 使用 Postman 导入 `docs/swagger.json`
3. 参考 `docs/AGENT_API_USAGE.md` 文档

### Q4: 遇到问题怎么办？
**A:**
1. 检查文档：`docs/AGENT_API_USAGE.md`
2. 查看示例：Swagger UI 中的示例请求
3. 提交 Issue：项目 GitHub 仓库

---

## 📚 相关文档

- [Agent API 使用指南](./AGENT_API_USAGE.md) - 完整的新 API 文档
- [API 变更记录](./API_CHANGES.md) - 详细变更列表
- [Swagger 文档](../docs/swagger.yaml) - OpenAPI 规范

---

**最后更新**: 2025-11-12
**适用版本**: v2.0+
