# 数据库清理文档索引

本目录包含完整的数据库冗余字段分析和清理方案。

---

## 📚 文档结构

### 1️⃣ 核心报告
**[DATABASE_REDUNDANCY_ANALYSIS_REPORT.md](./DATABASE_REDUNDANCY_ANALYSIS_REPORT.md)** (24KB)
- **用途**: 完整的深度分析报告
- **适合**: 技术负责人、架构师、DBA
- **内容**:
  - 所有表的详细字段分析
  - 每个字段的使用情况统计
  - 冗余字段的识别依据
  - 风险评估和影响分析
  - 详细的删除建议和迁移脚本

**阅读时间**: 30-45 分钟

### 2️⃣ 快速参考
**[CLEANUP_QUICK_REFERENCE.md](./CLEANUP_QUICK_REFERENCE.md)** (8.7KB)
- **用途**: 快速查找删除建议
- **适合**: 开发人员、DBA
- **内容**:
  - 按风险等级分类的字段列表
  - 简洁的删除SQL语句
  - 执行顺序建议
  - 代码清理检查清单
  - 常见问题解答

**阅读时间**: 10-15 分钟

### 3️⃣ 执行摘要
**[CLEANUP_EXECUTION_SUMMARY.md](./CLEANUP_EXECUTION_SUMMARY.md)** (9.2KB)
- **用途**: 实际执行指南
- **适合**: 项目经理、执行人员
- **内容**:
  - 分阶段执行计划
  - 详细的执行步骤
  - 验证检查清单
  - 时间规划
  - 成功标准

**阅读时间**: 15-20 分钟

---

## 📁 迁移脚本

### 目录
`internal/infrastructure/migration/scripts/cleanup/`

### 文件列表
1. **README.md** - 迁移脚本使用说明
2. **008_phase1_remove_unused_fields.sql** - Phase 1 迁移（零风险）
3. **009_phase2_remove_low_usage_fields.sql** - Phase 2 迁移（低风险）

---

## 🛠️ 辅助工具

### 脚本目录
`scripts/`

### 工具列表

#### 1. `check_field_usage.sh`
检查字段在代码库中的使用情况

**使用方法**:
```bash
./scripts/check_field_usage.sh <field_name> [table_name]
```

**示例**:
```bash
./scripts/check_field_usage.sh APIRequests subscription_usages
./scripts/check_field_usage.sh CustomEndpoint
```

**输出**: 按层级展示字段的引用情况

#### 2. `verify_cleanup.sh`
验证数据库清理的准备工作

**使用方法**:
```bash
./scripts/verify_cleanup.sh <phase>
```

**示例**:
```bash
./scripts/verify_cleanup.sh 1  # 验证 Phase 1 准备
./scripts/verify_cleanup.sh 2  # 验证 Phase 2 准备
```

**输出**: 前置检查结果和安全提示

---

## 🎯 快速开始指南

### 情景1: 我想了解整体情况
1. 阅读 **CLEANUP_EXECUTION_SUMMARY.md** 的"快速概览"部分
2. 浏览 **DATABASE_REDUNDANCY_ANALYSIS_REPORT.md** 的"执行摘要"

### 情景2: 我要执行清理工作
1. 详读 **CLEANUP_EXECUTION_SUMMARY.md**
2. 参考 **CLEANUP_QUICK_REFERENCE.md** 的"执行顺序建议"
3. 使用 `verify_cleanup.sh` 验证准备工作
4. 执行迁移脚本

### 情景3: 我想确认某个字段是否可以删除
1. 使用 `check_field_usage.sh <field_name>` 工具
2. 查阅 **DATABASE_REDUNDANCY_ANALYSIS_REPORT.md** 对应表的分析
3. 参考 **CLEANUP_QUICK_REFERENCE.md** 的风险矩阵

### 情景4: 我需要技术依据说服团队
1. 展示 **DATABASE_REDUNDANCY_ANALYSIS_REPORT.md**
2. 重点: "执行摘要"、"汇总统计"、"风险评估"章节

---

## 📊 核心发现摘要

### 统计数据
- **识别冗余字段总数**: 45+
- **涉及表数**: 15个
- **完全未使用字段**: 15个
- **计算冗余字段**: 3个
- **业务废弃字段**: 8个
- **设计过度字段**: 19个

### 重点问题
1. **subscription_histories 表** - 整表未实现 ⭐⭐⭐⭐⭐
2. **subscription_usages 表** - 7-9个字段未使用 ⭐⭐⭐⭐⭐
3. **subscription_plans 表** - 多个限制字段设计混乱 ⭐⭐⭐⭐

### 预期收益
- 数据库大小: **-10-15%**
- 代码行数: **-700行**
- 查询性能: **+2-5%**
- 维护成本: **显著降低**

---

## 🗺️ 执行路线图

### Phase 1: 立即执行（本周）✅
**风险**: 零风险  
**时间**: 2-3小时  
**内容**: 
- 删除 `subscription_histories` 表
- 清理 `subscription_usages` 7个字段
- 删除 `subscription_plans.custom_endpoint`

### Phase 2: 评估后执行（下周）🟡
**风险**: 低风险  
**时间**: 4-5小时  
**内容**:
- 删除 `users.locale`
- 删除 `announcements.view_count`
- 删除 `notifications.archived_at`

### Phase 3: 业务确认后执行（月内）🟠
**风险**: 中风险  
**时间**: 8-10小时  
**内容**: 
- 评估 `subscription_usages.storage_used`
- 评估 `users.avatar_url`
- 统一 subscription_plans 限制字段设计

---

## ✅ 执行检查清单

### 准备阶段
- [ ] 阅读完整的分析报告
- [ ] 团队评审和讨论
- [ ] 确认执行时间窗口
- [ ] 准备回滚方案

### 执行阶段
- [ ] 数据库完整备份
- [ ] 运行 `verify_cleanup.sh`
- [ ] 执行迁移脚本
- [ ] 清理相关代码
- [ ] 运行测试套件
- [ ] 更新 Swagger 文档

### 验证阶段
- [ ] 数据库结构检查
- [ ] 应用启动检查
- [ ] 功能手工测试
- [ ] 性能对比测试
- [ ] 文档更新确认

---

## 📞 支持与反馈

### 问题反馈
如果在执行过程中遇到问题：
1. 检查相关文档的 FAQ 部分
2. 使用 `check_field_usage.sh` 工具再次确认
3. 查看迁移脚本的注释说明
4. 联系技术负责人

### 文档贡献
如果发现文档错误或需要补充：
1. 提交 Issue 或 PR
2. 注明文档名称和章节
3. 提供建议的改进内容

---

## 📈 版本历史

| 版本 | 日期 | 变更内容 |
|------|------|---------|
| v1.0 | 2025-11-12 | 初始版本 - 完整的分析报告和清理方案 |

---

## 📖 相关资源

### 内部文档
- [API Migration Guide](./docs/API_MIGRATION_GUIDE.md)
- [Plan Features Usage Guide](./PLAN_FEATURES_USAGE_GUIDE.md)

### 外部参考
- [Goose Migration Tool](https://github.com/pressly/goose)
- [GORM Documentation](https://gorm.io/docs/)
- [Database Design Best Practices](https://www.postgresql.org/docs/current/ddl.html)

---

**创建日期**: 2025-11-12  
**最后更新**: 2025-11-12  
**维护者**: Database Cleanup Team  
**状态**: ✅ 就绪
