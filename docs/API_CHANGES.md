# API Breaking Changes - Node Password Removal

## 概述

从本版本开始，节点（Node）不再在创建和更新时要求密码字段。密码认证已迁移到订阅（Subscription）层面，每个订阅使用其唯一的UUID作为认证凭证。

## 修改原因

- **流量独立计量**: 通过订阅UUID，节点后端可以区分不同用户的不同订阅
- **更好的安全性**: 每个订阅使用独立的UUID凭证
- **支持多订阅**: 一个用户可以有多个订阅，每个订阅独立计量
- **符合行业标准**: 遵循主流代理面板的设计模式

## API 变更详情

### 1. POST /nodes - 创建节点

#### 变更前
```json
{
  "name": "US-Node-01",
  "server_address": "1.2.3.4",
  "server_port": 8388,
  "method": "aes-256-gcm",
  "password": "your-password",  // ❌ 必需字段
  "country": "US"
}
```

#### 变更后
```json
{
  "name": "US-Node-01",
  "server_address": "1.2.3.4",
  "server_port": 8388,
  "method": "aes-256-gcm",      // ✅ 只需加密方法
  "country": "US"
}
```

**说明**:
- `password` 字段已从请求体中**移除**
- `method` 字段现在只表示加密方法（如 `aes-256-gcm`, `chacha20-ietf-poly1305`）
- 实际的认证密码由用户的订阅UUID提供

### 2. PUT /nodes/{id} - 更新节点

#### 变更前
```json
{
  "method": "chacha20-ietf-poly1305",
  "password": "new-password"     // ❌ 可选字段
}
```

#### 变更后
```json
{
  "method": "chacha20-ietf-poly1305"  // ✅ 只更新加密方法
}
```

**说明**:
- `password` 字段已被**废弃**，不再接受
- 只能更新加密方法，不能更新密码

### 3. 订阅生成 - 自动使用订阅UUID

当用户获取订阅链接时，系统会自动使用订阅的UUID作为节点的认证密码。

**示例流程**:
1. 用户创建订阅 → 系统生成订阅UUID（如 `a1b2c3d4-e5f6-7890-abcd-ef1234567890`）
2. 订阅关联到节点组 → 节点组包含多个节点
3. 生成订阅链接 → 每个节点配置中使用订阅UUID作为密码

**生成的Shadowsocks链接示例**:
```
ss://YWVzLTI1Ni1nY206YTFiMmMzZDQtZTVmNi03ODkwLWFiY2QtZWYxMjM0NTY3ODkw@1.2.3.4:8388#US-Node-01
```
其中密码部分是订阅UUID（`a1b2c3d4-e5f6-7890-abcd-ef1234567890`）

## 数据库迁移

### Subscriptions 表
添加了 `uuid` 字段：
```sql
ALTER TABLE subscriptions ADD COLUMN uuid VARCHAR(36) NOT NULL UNIQUE;
CREATE UNIQUE INDEX idx_subscription_uuid ON subscriptions(uuid);
```

### Nodes 表
移除了 `encryption_password` 字段：
```sql
ALTER TABLE nodes DROP COLUMN encryption_password;
```

## 兼容性说明

### ⚠️ Breaking Changes

1. **API请求**: 所有创建和更新节点的API调用必须**移除** `password` 字段
2. **现有节点**: 数据库迁移会移除现有节点的密码字段
3. **现有订阅**: 数据库迁移会为现有订阅生成UUID

### ✅ 向后兼容

- GET请求（查询节点）不受影响
- 订阅链接格式保持不变（只是密码来源改变）
- 节点后端认证机制不变（仍使用method+password）

## 客户端影响

**无影响** - 对于使用订阅链接的客户端来说，这是透明的变更。客户端仍然会获得包含密码的完整配置，只是密码现在来自订阅UUID而非节点固定密码。

## 示例代码

### 创建节点（Go）
```go
req := CreateNodeRequest{
    Name:          "US-Node-01",
    ServerAddress: "1.2.3.4",
    ServerPort:    8388,
    Method:        "aes-256-gcm", // 只需要加密方法
    Country:       "US",
}
```

### 创建节点（cURL）
```bash
curl -X POST https://api.example.com/nodes \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "US-Node-01",
    "server_address": "1.2.3.4",
    "server_port": 8388,
    "method": "aes-256-gcm",
    "country": "US"
  }'
```

## 常见问题

### Q: 如何为不同用户设置不同的密码？
A: 不需要手动设置。每个订阅自动获得唯一的UUID，这个UUID就是该订阅在所有节点上的认证凭证。

### Q: 现有节点会受影响吗？
A: 数据库迁移会自动处理。但需要确保所有订阅都有UUID（迁移脚本会自动生成）。

### Q: 节点后端需要修改吗？
A: 节点后端仍然使用 method + password 的方式认证，只是密码现在来自订阅UUID。后端可通过UUID识别不同订阅并分别计量。

### Q: 可以手动指定订阅UUID吗？
A: 不可以。UUID由系统自动生成，确保全局唯一性。

## 相关文档

- [Subscription API Documentation](./swagger.yaml)
- [Database Migration Guide](../internal/infrastructure/persistence/migrations/subscriptionuuidmigration.go)
- [Architecture Decision Record](../CLAUDE.md)

## 更新日期

2025-11-11
