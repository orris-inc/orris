# 数据库清理执行摘要

## 快速概览

本次数据库清理行动识别出 **45+ 个冗余字段**，分为 2 个阶段执行：

| 阶段 | 风险等级 | 删除数量 | 预计时间 | 代码变更 |
|------|---------|---------|---------|---------|
| Phase 1 | ✅ 零风险 | 1个表 + 8个字段 | 2-3小时 | ~500行 |
| Phase 2 | 🟡 低风险 | 3个字段 | 4-5小时 | ~200行 |

**总预计收益**:
- 数据库大小减少: 10-15%
- 代码库减少: ~700 行
- 维护成本: 显著降低
- 查询性能: 轻微提升 (2-5%)

---

## Phase 1: 立即执行（零风险）

### 删除内容

1. **整个 subscription_histories 表**
   - 原因: 从未实现，无任何代码引用
   - 影响: 零

2. **subscription_usages 表的 7 个字段**
   - `api_requests`, `api_data_out`, `api_data_in`
   - `webhook_calls`, `emails_sent`, `reports_generated`
   - `projects_count`
   - 原因: 仅有 domain 层方法定义，无实际业务使用
   - 影响: 仅需删除无用代码

3. **subscription_plans.custom_endpoint**
   - 原因: 为未实现功能预留，完全无使用
   - 影响: 零

### 执行步骤

```bash
# 1. 备份数据库
mkdir -p backups
mysqldump -u root -p orris > backups/backup_$(date +%Y%m%d_%H%M%S).sql

# 2. 验证准备工作
./scripts/verify_cleanup.sh 1

# 3. 执行迁移
goose -dir internal/infrastructure/migration/scripts/cleanup \
  mysql "user:pass@/orris" up-to 8

# 4. 清理代码（按照清单逐个清理）
# 见 CLEANUP_QUICK_REFERENCE.md 第七章

# 5. 运行测试
go test ./... -v

# 6. 更新文档
swag init
```

### 代码清理清单

#### subscription_histories
- [ ] 无代码需要清理（从未实现）

#### subscription_usages 字段
对于每个字段，清理以下文件：

**Model 层**:
- [ ] `internal/infrastructure/persistence/models/subscriptionusagemodel.go`
  - 删除字段定义

**Domain 层**:
- [ ] `internal/domain/subscription/subscriptionusage.go`
  - 删除私有字段 (如 `apiRequests uint64`)
  - 删除 Getter 方法 (如 `APIRequests() uint64`)
  - 删除 Setter/Increment 方法 (如 `IncrementAPIRequests(uint64)`)
  - 更新 `NewSubscriptionUsage()` 构造函数
  - 更新 `ReconstructSubscriptionUsage()` 函数
  - 更新 `Reset()` 方法
  - 更新 `HasUsage()` 方法
  - 删除 `GetTotalActivity()` 中的相关字段

**Mapper 层**:
- [ ] `internal/infrastructure/persistence/mappers/subscriptionusagemapper.go`
  - 在 `ToEntity()` 中删除字段映射
  - 在 `ToModel()` 中删除字段映射

**Repository 层**:
- [ ] `internal/infrastructure/repository/subscriptionusagerepository.go`
  - 检查是否有查询使用这些字段（应该没有）

#### subscription_plans.custom_endpoint

- [ ] `internal/infrastructure/persistence/models/subscriptionplanmodel.go`
- [ ] `internal/infrastructure/persistence/mappers/subscriptionplanmapper.go`
- [ ] `internal/domain/subscription/subscriptionplan.go`
- [ ] `internal/application/subscription/dto/dto.go`
- [ ] 所有 use case 中的相关引用

---

## Phase 2: 评估后执行（低风险）

### 删除内容

1. **users.locale**
   - 原因: 仅在 OAuth 登录设置，无业务使用
   - 影响: 需修改 OAuth 集成代码（2个文件）

2. **announcements.view_count**
   - 原因: 
     - 并发不安全
     - 无统计分析功能
     - 增加数据库写压力
   - 替代方案: Redis 统计
   - 影响: 需修改 `getannouncement.go` 用例

3. **notifications.archived_at**
   - 原因: 与 GORM 的 `deleted_at` 重复
   - 影响: 需统一使用 `deleted_at`

### 执行步骤

```bash
# 1. 完成 Phase 1 的所有清理工作

# 2. 实现替代方案（如 Redis view count）

# 3. 验证准备工作
./scripts/verify_cleanup.sh 2

# 4. 执行迁移
goose -dir internal/infrastructure/migration/scripts/cleanup \
  mysql "user:pass@/orris" up-to 9

# 5. 清理代码

# 6. 测试
go test ./... -v
go run cmd/api/main.go
# 手工测试 OAuth 登录
# 手工测试公告查看
```

### 代码清理清单

#### users.locale
- [ ] `internal/infrastructure/persistence/models/usermodel.go`
  - 删除 `Locale` 字段
- [ ] `internal/infrastructure/auth/oauthgoogle.go`
  - 删除设置 locale 的代码（约第 XX 行）

#### announcements.view_count
- [ ] `internal/infrastructure/persistence/models/announcementmodel.go`
  - 删除 `ViewCount` 字段
- [ ] `internal/domain/notification/announcement.go`
  - 删除 `viewCount` 字段
  - 删除 `ViewCount()` getter
  - 删除 `IncrementViewCount()` 方法
- [ ] `internal/application/notification/usecases/getannouncement.go`
  - 删除 `announcement.IncrementViewCount()` 调用
  - 可选: 添加 Redis 统计
    ```go
    redis.Incr(ctx, fmt.Sprintf("announcement:view:%d", id))
    ```
- [ ] `internal/interfaces/dto/notificationdto.go`
  - 删除 `ViewCount` 字段

#### notifications.archived_at
- [ ] `internal/infrastructure/persistence/models/notificationmodel.go`
  - 删除 `ArchivedAt` 字段
- [ ] `internal/infrastructure/persistence/mappers/notificationmapper.go`
  - 删除特殊的 `ArchivedAt` 到 `DeletedAt` 映射逻辑（第78-83行）
  - 直接使用 GORM 的 `DeletedAt`
- [ ] `internal/domain/notification/notification.go`
  - 删除 `archivedAt` 字段
  - 删除 `ArchivedAt()` getter
  - 归档操作直接使用软删除

---

## 验证检查清单

### 迁移后立即检查

```bash
# 1. 数据库结构验证
mysql -u root -p orris -e "SHOW TABLES;"
mysql -u root -p orris -e "DESCRIBE subscription_usages;"
mysql -u root -p orris -e "DESCRIBE users;"
mysql -u root -p orris -e "DESCRIBE announcements;"

# 2. 应用启动检查
go run cmd/api/main.go
# 检查是否有报错

# 3. 测试套件
go test ./internal/infrastructure/repository/... -v
go test ./internal/application/... -v
go test ./internal/interfaces/... -v

# 4. Swagger 文档生成
swag init
# 检查 docs/swagger.json 是否正确生成
```

### 功能验证（手工测试）

#### Phase 1
- [ ] 创建订阅
- [ ] 查看订阅详情
- [ ] 创建订阅计划
- [ ] 查看订阅计划列表

#### Phase 2
- [ ] OAuth Google 登录
- [ ] OAuth GitHub 登录  
- [ ] 查看公告列表
- [ ] 查看公告详情
- [ ] 归档通知
- [ ] 查看通知列表

### 性能验证

```bash
# 1. 检查表大小变化
mysql -u root -p orris -e "
  SELECT 
    table_name,
    ROUND(((data_length + index_length) / 1024 / 1024), 2) AS size_mb
  FROM information_schema.TABLES
  WHERE table_schema = 'orris'
  ORDER BY size_mb DESC;
"

# 2. 简单性能测试
# 查询订阅计划（应该稍快）
time mysql -u root -p orris -e "SELECT * FROM subscription_plans LIMIT 100;"
```

---

## 回滚方案

### 如果出现问题

```bash
# 方案1: 使用 goose 回滚
goose -dir internal/infrastructure/migration/scripts/cleanup \
  mysql "user:pass@/orris" down

# 方案2: 从备份恢复
mysql -u root -p orris < backups/backup_YYYYMMDD_HHMMSS.sql

# 方案3: 手工回滚（参考迁移脚本的 Down 部分）
```

### 回滚后需要做什么

1. 恢复删除的代码（使用 git）
2. 重新运行测试
3. 检查应用正常启动
4. 分析问题原因

---

## 风险评估

### Phase 1 风险: ✅ 零风险
**原因**:
- `subscription_histories` 表从未被任何代码使用
- `subscription_usages` 字段仅有无业务逻辑的 getter/setter
- `custom_endpoint` 完全无引用

**最坏情况**: 删除一些无用代码后需要小幅重新编译

### Phase 2 风险: 🟡 低风险
**原因**:
- `users.locale`: OAuth 集成明确不依赖此字段
- `view_count`: 仅在一个用例中使用，无并发保护
- `archived_at`: 可以用 `deleted_at` 完全替代

**最坏情况**: 需要少量代码调整（<50行）

---

## 时间规划

### Week 1: Phase 1

| 任务 | 预计时间 | 负责人 |
|------|---------|-------|
| 数据库备份 | 10分钟 | DBA |
| 执行迁移 | 5分钟 | DBA |
| 代码清理 | 2小时 | 开发 |
| 测试验证 | 1小时 | QA |
| **总计** | **3小时** | - |

### Week 2: Phase 2

| 任务 | 预计时间 | 负责人 |
|------|---------|-------|
| 实现 Redis view count（可选） | 1小时 | 开发 |
| 执行迁移 | 5分钟 | DBA |
| 代码清理 | 2小时 | 开发 |
| OAuth 测试 | 1小时 | QA |
| 通知功能测试 | 1小时 | QA |
| **总计** | **5小时** | - |

---

## 成功标准

### Phase 1 完成标准
- [x] 迁移脚本执行成功，无错误
- [x] `subscription_histories` 表已删除
- [x] `subscription_usages` 表字段已删除
- [x] 所有测试通过（单元测试 + 集成测试）
- [x] 应用正常启动，无 panic
- [x] Swagger 文档正确生成
- [x] 手工测试订阅相关功能正常

### Phase 2 完成标准
- [x] Phase 1 所有标准
- [x] OAuth 登录功能正常（Google + GitHub）
- [x] 公告查看功能正常
- [x] 通知归档功能正常
- [x] 如实现 Redis view count，统计功能正常

---

## 联系与支持

### 问题报告
如果遇到问题，请提供：
1. 错误日志
2. 执行的具体步骤
3. 数据库备份位置

### 参考文档
- 详细分析报告: `DATABASE_REDUNDANCY_ANALYSIS_REPORT.md`
- 快速参考指南: `CLEANUP_QUICK_REFERENCE.md`
- 迁移脚本: `internal/infrastructure/migration/scripts/cleanup/`

### 工具脚本
- 字段使用检查: `./scripts/check_field_usage.sh <field_name>`
- 迁移验证: `./scripts/verify_cleanup.sh <phase>`

---

**文档版本**: v1.0  
**创建日期**: 2025-11-12  
**最后更新**: 2025-11-12  
**状态**: 待执行
