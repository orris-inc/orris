# 节点管理迁移脚本说明

## 概述

本文档说明如何运行节点管理相关的数据库迁移脚本。

## 创建的数据库表

### 1. nodes - 节点配置表
存储代理服务器节点的配置信息，包括：
- 服务器地址和端口
- 加密方法和密码
- 协议类型（Shadowsocks/Trojan）
- 节点状态（active/inactive/maintenance）
- 流量统计和限制
- 认证令牌

**索引：**
- `idx_status`: 节点状态索引
- `idx_protocol`: 协议类型索引
- `idx_server`: 服务器地址和端口复合索引
- `idx_token_hash`: 令牌哈希唯一索引

### 2. node_groups - 节点组表
用于组织和管理节点分组：
- 节点组名称和描述
- 是否公开
- 排序顺序

**索引：**
- `idx_is_public`: 公开状态索引

### 3. node_group_nodes - 节点组与节点关联表
多对多关系表，关联节点组和节点：
- 节点组 ID
- 节点 ID

**约束：**
- 唯一约束：`(node_group_id, node_id)`
- 外键：级联删除

### 4. node_group_plans - 节点组与订阅计划关联表
多对多关系表，关联节点组和订阅计划：
- 节点组 ID
- 订阅计划 ID

**约束：**
- 唯一约束：`(node_group_id, subscription_plan_id)`
- 外键：级联删除

### 5. node_traffic - 节点流量统计表
记录节点级别的流量统计：
- 上传/下载/总流量
- 时间周期
- 关联的用户和订阅

**索引：**
- `idx_node_period`: 节点和时间周期复合索引
- `idx_user_period`: 用户和时间周期复合索引
- `idx_subscription`: 订阅索引

**外键：**
- `node_id` → nodes (ON DELETE CASCADE)
- `user_id` → users (ON DELETE SET NULL)
- `subscription_id` → subscriptions (ON DELETE SET NULL)

### 6. user_traffic - 用户流量统计表
记录用户级别的每个节点流量统计：
- 上传/下载/总流量
- 时间周期
- 关联的节点和订阅

**索引：**
- `idx_user_node_period`: 用户、节点和时间周期复合索引
- `idx_user_period`: 用户和时间周期复合索引
- `idx_node_period`: 节点和时间周期复合索引
- `idx_subscription`: 订阅索引

**唯一约束：**
- `idx_user_traffic_unique`: `(user_id, node_id, period)`

**外键：**
- `user_id` → users (ON DELETE CASCADE)
- `node_id` → nodes (ON DELETE CASCADE)
- `subscription_id` → subscriptions (ON DELETE SET NULL)

## 执行迁移

### 方法一：使用 CLI 命令（推荐）

```bash
# 运行所有待执行的迁移
./bin/orris migrate up

# 查看迁移状态
./bin/orris migrate status

# 回滚迁移（谨慎使用）
./bin/orris migrate down --steps 1
```

### 方法二：直接执行 SQL 脚本

如果你需要手动执行迁移脚本：

```bash
# 执行迁移
mysql -u orris -p orris_dev < internal/infrastructure/migration/scripts/004_node_tables.sql

# 或使用 goose 命令行工具
goose -dir internal/infrastructure/migration/scripts mysql "orris:password@/orris_dev" up
```

## 验证迁移

执行迁移后，验证表是否正确创建：

```sql
-- 检查表是否存在
SHOW TABLES LIKE 'node%';

-- 检查 nodes 表结构
DESCRIBE nodes;

-- 检查索引
SHOW INDEX FROM nodes;
SHOW INDEX FROM node_groups;
SHOW INDEX FROM node_group_nodes;
SHOW INDEX FROM node_group_plans;
SHOW INDEX FROM node_traffic;
SHOW INDEX FROM user_traffic;

-- 检查外键约束
SELECT
    CONSTRAINT_NAME,
    TABLE_NAME,
    REFERENCED_TABLE_NAME,
    DELETE_RULE
FROM
    information_schema.REFERENTIAL_CONSTRAINTS
WHERE
    TABLE_SCHEMA = 'orris_dev'
    AND TABLE_NAME IN ('node_group_nodes', 'node_group_plans', 'node_traffic', 'user_traffic');
```

## 回滚迁移

如果需要回滚此迁移（**注意：这将删除所有节点相关数据**）：

```bash
# 使用 CLI 回滚一个步骤
./bin/orris migrate down --steps 1

# 或使用 goose 直接回滚
goose -dir internal/infrastructure/migration/scripts mysql "orris:password@/orris_dev" down
```

## 注意事项

### 1. 数据完整性
- 所有关联表都设置了适当的外键约束
- `node_group_nodes` 和 `node_group_plans` 使用 `ON DELETE CASCADE`
- `node_traffic.user_id` 使用 `ON DELETE SET NULL` 以保留历史记录

### 2. 性能优化
- 所有频繁查询的字段都已添加索引
- 使用复合索引提高多条件查询性能
- `user_traffic` 表有唯一约束防止重复数据

### 3. 现有表依赖
此迁移依赖以下已存在的表：
- `users` - 用户表（来自 001_initial_schema.sql）
- `subscriptions` - 订阅表（来自 002_subscription_tables.sql）
- `subscription_plans` - 订阅计划表（来自 002_subscription_tables.sql）

**确保在运行此迁移之前，这些表已经存在！**

### 4. 字符集和排序
- 所有表使用 `utf8mb4` 字符集
- 使用 `utf8mb4_general_ci` 排序规则
- 支持存储 emoji 和特殊字符

## 下一步

迁移成功后，你可以：

1. **使用 GORM 模型**：
   - `NodeModel`
   - `NodeGroupModel`
   - `NodeGroupNodeModel`
   - `NodeGroupPlanModel`
   - `NodeTrafficModel`
   - `UserTrafficModel`

2. **实现业务逻辑**：
   - 节点管理 API
   - 流量统计和监控
   - 节点组权限控制

3. **添加初始数据**：
   ```sql
   -- 示例：添加初始节点组
   INSERT INTO node_groups (name, description, is_public, sort_order)
   VALUES ('Default Group', 'Default node group for all users', TRUE, 0);
   ```

## 疑难解答

### 迁移失败：外键约束错误

如果遇到外键约束错误，确保：
1. 依赖的表（users, subscriptions, subscription_plans）已存在
2. 按照正确的顺序运行迁移脚本

### 索引已存在错误

如果遇到 "Duplicate key name" 错误：
```sql
-- 删除重复的索引
DROP INDEX idx_name ON table_name;
```

### 表已存在错误

如果表已经存在，可以：
1. 删除现有表（**注意：会丢失数据**）
2. 或跳过此迁移

```sql
-- 删除所有节点相关表（谨慎使用）
DROP TABLE IF EXISTS user_traffic;
DROP TABLE IF EXISTS node_traffic;
DROP TABLE IF EXISTS node_group_plans;
DROP TABLE IF EXISTS node_group_nodes;
DROP TABLE IF EXISTS node_groups;
DROP TABLE IF EXISTS nodes;
```

## 参考资料

- [GORM 文档](https://gorm.io/docs/)
- [Goose 迁移工具](https://github.com/pressly/goose)
- [MySQL 外键约束](https://dev.mysql.com/doc/refman/8.0/en/create-table-foreign-keys.html)
