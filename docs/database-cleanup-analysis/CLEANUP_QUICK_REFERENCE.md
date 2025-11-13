# 数据库清理快速参考指南

## 一、立即可删除的字段（零风险）⭐⭐⭐⭐⭐

### 1. subscription_histories 表（整表删除）
```sql
DROP TABLE IF EXISTS subscription_histories;
```
**原因**: 整个表从未实现，无任何代码引用

### 2. subscription_usages 表字段清理
```sql
ALTER TABLE subscription_usages 
  DROP COLUMN api_requests,
  DROP COLUMN api_data_out,
  DROP COLUMN api_data_in,
  DROP COLUMN webhook_calls,
  DROP COLUMN emails_sent,
  DROP COLUMN reports_generated,
  DROP COLUMN projects_count;
```
**原因**: 这些字段只有 Domain 层的 getter/setter，无实际业务使用

### 3. subscription_plans.custom_endpoint
```sql
ALTER TABLE subscription_plans DROP COLUMN custom_endpoint;
```
**原因**: 为未实现的功能预留，完全无使用

---

## 二、建议删除的字段（低风险）⭐⭐⭐⭐

### 1. users.locale
```sql
ALTER TABLE users DROP COLUMN locale;
```
**原因**: 仅在 OAuth 登录时设置，无任何业务使用
**影响**: 需修改 OAuth 集成代码（2 个文件）

### 2. announcements.view_count
```sql
ALTER TABLE announcements DROP COLUMN view_count;
```
**原因**: 
- 并发不安全
- 无统计分析功能使用
- 增加数据库写压力

**替代方案**: 使用 Redis 统计
```go
redis.INCR("announcement:view_count:{id}")
```

### 3. notifications.archived_at
```sql
ALTER TABLE notifications DROP COLUMN archived_at;
```
**原因**: 与 GORM 的 `deleted_at` 功能重复

---

## 三、需评估后删除（中风险）⭐⭐⭐

### 1. subscription_usages.storage_used
```sql
ALTER TABLE subscription_usages DROP COLUMN storage_used;
```
**评估点**: 在代理节点项目中，"存储使用量"的业务含义是什么？

### 2. users.avatar_url
```sql
ALTER TABLE users DROP COLUMN avatar_url;
```
**评估点**: 是否需要用户资料展示功能？

### 3. subscription_plans 的限制字段
**选项A: 合并到 JSON**
```sql
-- 将 max_users, max_projects, storage_limit 合并到 limits JSON
-- 需要大量代码重构
```

**选项B: 保持现状**
```sql
-- 不做改动，但删除未使用的 limits JSON 字段
ALTER TABLE subscription_plans DROP COLUMN limits;
```

---

## 四、性能权衡字段（建议保留）⭐

### 1. node_traffic.total 和 user_traffic.total
```sql
-- 不建议删除
-- total = upload + download
```
**原因**: 虽然是计算冗余，但可以提升查询性能

---

## 五、执行顺序建议

### Phase 1: 立即执行（本周）
1. 删除 `subscription_histories` 表
2. 清理 `subscription_usages` 7个字段
3. 删除 `subscription_plans.custom_endpoint`

**预计收益**:
- 数据库大小: -10%
- 代码行数: -500 行
- 时间成本: 2-3 小时

### Phase 2: 评估后执行（下周）
1. 删除 `users.locale`
2. 删除 `announcements.view_count`（实现 Redis 替代）
3. 删除 `notifications.archived_at`

**预计收益**:
- 数据库大小: -3%
- 代码行数: -200 行
- 时间成本: 4-5 小时

### Phase 3: 业务确认后执行（月内）
1. 评估并决定 `subscription_usages.storage_used`
2. 评估并决定 `users.avatar_url`
3. 统一 `subscription_plans` 限制字段设计

**预计收益**:
- 数据库大小: -2%
- 代码行数: -300 行
- 时间成本: 8-10 小时

---

## 六、迁移脚本使用说明

### 准备工作
```bash
# 1. 备份数据库
mysqldump -u root -p orris > backup_$(date +%Y%m%d_%H%M%S).sql

# 2. 创建测试环境
cp .env .env.test
# 修改 .env.test 使用测试数据库

# 3. 在测试环境验证
go test ./... -v
```

### 执行迁移
```bash
# Phase 1
goose -dir internal/infrastructure/migration/scripts mysql "user:pass@/orris" up

# 验证
mysql -u root -p orris -e "SHOW TABLES;"
mysql -u root -p orris -e "DESCRIBE subscription_usages;"
```

### 回滚（如果需要）
```bash
goose -dir internal/infrastructure/migration/scripts mysql "user:pass@/orris" down
```

---

## 七、代码清理检查清单

对于每个删除的字段，按以下顺序清理代码：

### 步骤1: Model 层
```bash
# 文件: internal/infrastructure/persistence/models/*model.go
- [ ] 删除字段定义
- [ ] 删除 GORM 标签
- [ ] 清理 BeforeCreate/Update hooks
```

### 步骤2: Mapper 层
```bash
# 文件: internal/infrastructure/persistence/mappers/*mapper.go
- [ ] 删除 ToEntity() 中的映射
- [ ] 删除 ToModel() 中的映射
```

### 步骤3: Domain 层
```bash
# 文件: internal/domain/*/*.go
- [ ] 删除实体字段
- [ ] 删除 Getter 方法
- [ ] 删除 Setter 方法
- [ ] 更新构造函数
- [ ] 更新 Reconstruct 函数
```

### 步骤4: Repository 层
```bash
# 文件: internal/infrastructure/repository/*repository.go
- [ ] 删除相关查询逻辑
- [ ] 更新 WHERE 条件
```

### 步骤5: Use Case 层
```bash
# 文件: internal/application/*/usecases/*.go
- [ ] 删除业务逻辑
- [ ] 更新 Command/Query 结构
- [ ] 更新 Result 结构
```

### 步骤6: DTO 层
```bash
# 文件: internal/application/*/dto/*.go
# 文件: internal/interfaces/dto/*.go
- [ ] 删除 DTO 字段
- [ ] 删除 JSON 标签
- [ ] 更新 Converter
```

### 步骤7: Handler 层
```bash
# 文件: internal/interfaces/http/handlers/*/*.go
- [ ] 删除 HTTP 响应字段
- [ ] 更新 Swagger 注释
```

### 步骤8: 文档层
```bash
# 文件: docs/*
- [ ] 运行 swag init
- [ ] 检查生成的 docs/swagger.json
- [ ] 检查生成的 docs/swagger.yaml
```

---

## 八、自动化工具

### 字段使用情况检查
```bash
#!/bin/bash
# check_field_usage.sh

FIELD=$1
TABLE=$2

echo "=== Searching for: $FIELD in $TABLE ==="
echo ""

echo "1. Model Layer:"
grep -rn "\b${FIELD}\b" internal/infrastructure/persistence/models/ | grep -i "$TABLE"

echo ""
echo "2. Domain Layer:"
grep -rn "\b${FIELD}\b" internal/domain/

echo ""
echo "3. Use Case Layer:"
grep -rn "\b${FIELD}\b" internal/application/

echo ""
echo "4. Handler Layer:"
grep -rn "\b${FIELD}\b" internal/interfaces/
```

使用示例:
```bash
chmod +x check_field_usage.sh
./check_field_usage.sh "APIRequests" "subscription_usages"
```

### 迁移验证脚本
```bash
#!/bin/bash
# verify_migration.sh

echo "=== Pre-Migration Checklist ==="
echo "1. Database backup created? (y/n)"
read -r backup

echo "2. All tests passing? (y/n)"
read -r tests

echo "3. Code changes committed? (y/n)"
read -r commit

if [[ "$backup" == "y" && "$tests" == "y" && "$commit" == "y" ]]; then
    echo "✅ Ready to migrate!"
    echo "Run: goose up"
else
    echo "❌ Pre-requisites not met!"
    exit 1
fi
```

---

## 九、风险矩阵

| 字段/表 | 删除风险 | 代码影响 | 业务影响 | 建议 |
|---------|---------|---------|---------|------|
| subscription_histories | ✅ 零风险 | 无 | 无 | 立即删除 |
| subscription_usages.* (7个) | ✅ 零风险 | 仅 domain | 无 | 立即删除 |
| subscription_plans.custom_endpoint | ✅ 零风险 | 无 | 无 | 立即删除 |
| users.locale | 🟡 低风险 | OAuth (2文件) | 低 | 建议删除 |
| announcements.view_count | 🟡 低风险 | 中间件 | 低 | 建议删除 |
| notifications.archived_at | 🟡 低风险 | Mapper | 无 | 建议删除 |
| subscription_usages.storage_used | 🟠 中风险 | Middleware | 中 | 需评估 |
| users.avatar_url | 🟠 中风险 | OAuth | 中 | 需评估 |
| *_traffic.total | 🔴 高风险 | 多处查询 | 高 | 保留 |

---

## 十、FAQ

### Q1: 删除字段后数据会丢失吗？
**A**: 是的，执行 `DROP COLUMN` 后该字段的所有数据将永久删除。务必提前备份。

### Q2: 如果删除后发现需要怎么办？
**A**: 使用 goose down 回滚迁移，或从备份恢复数据。

### Q3: 为什么不一次性删除所有字段？
**A**: 分阶段删除可以：
1. 降低风险
2. 便于回滚
3. 逐步验证影响

### Q4: 删除字段后性能会提升吗？
**A**: 会有轻微提升：
- 表扫描更快（列数少）
- 备份更快
- 内存占用减少

但提升幅度不大（<5%），主要收益是维护成本降低。

### Q5: subscription_plan_pricing 表要删除吗？
**A**: 不建议删除整个表，但建议：
- 删除 subscription_plans 表中的 price/billing_cycle 字段
- 或删除 subscription_plan_pricing 表并保留 subscription_plans 的字段
- 不要让两个表同时存储定价信息

---

## 十一、成功标准

### Phase 1 完成标准
- [ ] 迁移脚本执行成功
- [ ] 所有单元测试通过
- [ ] 所有集成测试通过
- [ ] API 正常响应（手工测试 10+ 接口）
- [ ] Swagger 文档正确生成
- [ ] 无 console errors

### Phase 2 完成标准
- [ ] Phase 1 所有标准
- [ ] Redis view count 功能验证
- [ ] OAuth 登录功能正常

### Phase 3 完成标准
- [ ] Phase 2 所有标准  
- [ ] 业务方确认功能完整性
- [ ] 性能测试通过

---

## 十二、联系人

如有疑问，请联系：
- 技术负责人: [Name]
- DBA: [Name]
- 产品经理: [Name]

---

**最后更新**: 2025-11-12
**文档版本**: v1.0
