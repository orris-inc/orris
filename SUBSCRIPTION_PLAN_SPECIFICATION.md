# 订阅计划(Subscription Plan)文档规范汇总

## 一、文档总览

本项目中关于订阅计划的文档包括：
1. **业务规范文档** - 流量限制配置指南
2. **数据库迁移文档** - 多价格支持
3. **代码层实现规范** - 领域驱动设计
4. **API规范** - RESTful接口定义
5. **DTO转换规范** - 数据传输层定义

---

## 二、关键文档位置

### A. 顶级文档(项目根目录)

| 文档 | 路径 | 内容摘要 |
|------|------|--------|
| **使用指南** | `/PLAN_FEATURES_USAGE_GUIDE.md` | 完整的PlanFeatures使用指南，包括流量限制、设备限制、速度限制的配置和验证 |
| **验证报告** | `/PLAN_FEATURES_VERIFICATION_REPORT.md` | 流量限制配置的验证和增强工作报告（中文） |
| **快速参考** | `/QUICK_REFERENCE_TRAFFIC_LIMITS.md` | 标准限制键常量、单位转换、快速创建示例 |
| **并发分析** | `/CONCURRENCY_ANALYSIS_REPORT.md` | 并发处理分析（包含订阅相关内容） |

### B. 迁移脚本文档

| 文档 | 路径 | 内容摘要 |
|------|------|--------|
| **定价迁移** | `/internal/infrastructure/migration/scripts/MIGRATION_PLAN_PRICING.md` | 多价格选项支持迁移说明 |
| **节点表迁移** | `/internal/infrastructure/migration/scripts/MIGRATION_NODE_TABLES.md` | 节点表相关迁移 |

### C. 工程规范

| 文档 | 路径 | 内容摘要 |
|------|------|--------|
| **核心规则** | `/CLAUDE.md` | 项目通用规则、代码规范 |

---

## 三、核心业务规则

### 3.1 流量限制标准

#### 标准限制键常量
```go
vo.LimitKeyTraffic         = "traffic_limit"      // 月流量限制(字节)，0=无限
vo.LimitKeyDeviceCount     = "device_limit"       // 并发设备数，0=无限
vo.LimitKeySpeedLimit      = "speed_limit"        // 速度限制(Mbps)，0=无限
vo.LimitKeyConnectionLimit = "connection_limit"   // 并发连接数，0=无限
vo.LimitKeyNodeAccess      = "node_access"        // 可访问的节点组ID列表
```

#### 零值语义
- `traffic_limit: 0` → 无限流量
- `device_limit: 0` → 无限设备
- `speed_limit: 0` → 无限速度
- `connection_limit: 0` → 无限连接
- `node_access: []` → 所有节点可访问

### 3.2 订阅计划状态管理

```go
type PlanStatus string
const (
    PlanStatusActive   PlanStatus = "active"   // 活跃（可用）
    PlanStatusInactive PlanStatus = "inactive" // 不活跃（不可用）
)
```

#### 状态转换规则
- 新建计划默认为 `active` 状态
- 支持激活/停用操作
- 不支持直接删除，应使用停用代替

### 3.3 支持的货币

```go
validCurrencies = map[string]bool{
    "CNY": true, "USD": true, "EUR": true, "GBP": true, "JPY": true
}
```

### 3.4 计费周期类型

```go
type BillingCycle string
const (
    BillingCycleWeekly      BillingCycle = "weekly"
    BillingCycleMonthly     BillingCycle = "monthly"
    BillingCycleQuarterly   BillingCycle = "quarterly"
    BillingCycleSemiAnnual  BillingCycle = "semi_annual"
    BillingCycleYearly      BillingCycle = "yearly"
    BillingCycleLifetime    BillingCycle = "lifetime"
)
```

### 3.5 API速率限制

- 最小值: 1（每分钟请求数）
- 默认值: 60
- 0表示不允许设置，必须 > 0

---

## 四、数据库设计

### 4.1 主表: subscription_plans

关键字段：
- `id` - 计划ID (BIGINT UNSIGNED)
- `name` - 计划名称 (VARCHAR 100)
- `slug` - 唯一标识符 (VARCHAR 100, UNIQUE)
- `description` - 描述文本
- `price` - 价格（保留用于向后兼容）
- `currency` - 货币代码 (VARCHAR 3)
- `billing_cycle` - 计费周期
- `trial_days` - 试用天数
- `status` - 计划状态 (active/inactive)
- `features` - JSON格式特性列表
- `is_public` - 是否公开
- `sort_order` - 排序顺序
- `deleted_at` - 软删除时间戳

### 4.2 定价表: subscription_plan_pricing (新增)

用于支持多个计费周期的不同价格

关键字段：
- `id` - 定价ID
- `plan_id` - 关联的计划ID (FK→subscription_plans.id)
- `billing_cycle` - 计费周期 (weekly|monthly|quarterly|semi_annual|yearly|lifetime)
- `price` - 价格（最小货币单位，如分）
- `currency` - 货币代码
- `is_active` - 是否活跃
- `created_at`, `updated_at`, `deleted_at` - 时间戳

约束：
- UNIQUE(plan_id, billing_cycle) - 每个计划每个周期只能有一个价格
- FK plan_id CASCADE DELETE - 删除计划时自动删除定价

索引：
- idx_plan_id
- idx_billing_cycle
- idx_is_active
- idx_deleted_at

---

## 五、Domain层规范

### 5.1 SubscriptionPlan实体

位置: `/internal/domain/subscription/subscriptionplan.go`

#### 构造方法

```go
// 创建新计划
func NewSubscriptionPlan(
    name, slug, description string,
    price uint64,
    currency string,
    billingCycle vo.BillingCycle,
    trialDays int,
) (*SubscriptionPlan, error)

// 从持久层重构
func ReconstructSubscriptionPlan(
    id uint,
    ... // 其他字段
) (*SubscriptionPlan, error)
```

#### 验证规则

- `name`: 必需, 1-100字符
- `slug`: 必需, 1-100字符, 唯一
- `currency`: 必需, 5种支持的货币之一
- `billingCycle`: 必需, 有效的计费周期
- `trialDays`: >= 0

#### 核心业务方法

```go
// 状态管理
func (p *SubscriptionPlan) Activate() error      // 激活计划
func (p *SubscriptionPlan) Deactivate() error    // 停用计划
func (p *SubscriptionPlan) IsActive() bool       // 检查是否活跃

// 信息更新
func (p *SubscriptionPlan) UpdatePrice(price uint64, currency string) error
func (p *SubscriptionPlan) UpdateDescription(description string)
func (p *SubscriptionPlan) UpdateFeatures(features *vo.PlanFeatures) error

// 配置管理
func (p *SubscriptionPlan) SetAPIRateLimit(limit uint) error
func (p *SubscriptionPlan) SetMaxUsers(max uint)
func (p *SubscriptionPlan) SetMaxProjects(max uint)
func (p *SubscriptionPlan) SetSortOrder(order int)
func (p *SubscriptionPlan) SetPublic(isPublic bool)

// 流量限制查询（便捷方法）
func (p *SubscriptionPlan) GetTrafficLimit() (uint64, error)           // 获取月流量限制
func (p *SubscriptionPlan) IsUnlimitedTraffic() bool                   // 检查无限流量
func (p *SubscriptionPlan) HasTrafficRemaining(usedBytes uint64) (bool, error)  // 验证使用
```

### 5.2 PlanFeatures值对象

位置: `/internal/domain/subscription/value_objects/planfeatures.go`

#### 结构
```go
type PlanFeatures struct {
    Features []string               // 特性列表
    Limits   map[string]interface{} // 限制配置
}
```

#### 流量限制方法

```go
// 设置方法
func (pf *PlanFeatures) SetTrafficLimit(bytes uint64)
func (pf *PlanFeatures) SetDeviceLimit(count int) error
func (pf *PlanFeatures) SetSpeedLimit(mbps int) error
func (pf *PlanFeatures) SetConnectionLimit(count int) error
func (pf *PlanFeatures) SetNodeAccess(nodeGroupIDs []uint)

// 获取方法
func (pf *PlanFeatures) GetTrafficLimit() (uint64, error)
func (pf *PlanFeatures) GetDeviceLimit() (int, error)
func (pf *PlanFeatures) GetSpeedLimit() (int, error)
func (pf *PlanFeatures) GetConnectionLimit() (int, error)
func (pf *PlanFeatures) GetNodeAccess() ([]uint, error)

// 验证方法
func (pf *PlanFeatures) IsUnlimitedTraffic() bool
func (pf *PlanFeatures) HasTrafficRemaining(usedBytes uint64) (bool, error)

// 特性管理
func (pf *PlanFeatures) HasFeature(feature string) bool
func (pf *PlanFeatures) AddFeature(feature string)
```

### 5.3 PlanPricing值对象

位置: `/internal/domain/subscription/value_objects/planpricing.go`

```go
type PlanPricing struct {
    id           uint
    planID       uint
    billingCycle BillingCycle
    price        uint64
    currency     string
    isActive     bool
    createdAt    time.Time
    updatedAt    time.Time
}

// 验证规则
- planID: > 0
- price: > 0
- currency: 有效的货币代码
- billingCycle: 有效的计费周期

// 核心方法
func (p *PlanPricing) UpdatePrice(newPrice uint64) error
func (p *PlanPricing) Activate()
func (p *PlanPricing) Deactivate()
```

### 5.4 错误定义

位置: `/internal/domain/subscription/errors.go`

```go
ErrSubscriptionNotFound    // 订阅未找到
ErrPlanNotFound            // 计划未找到
ErrPlanInactive            // 计划不活跃
ErrPlanSlugExists          // 计划Slug已存在
ErrUsageLimitExceeded      // 使用限制超出
ErrInvalidBillingCycle     // 无效的计费周期
ErrInvalidPrice            // 无效的价格
ErrInvalidTrialDays        // 无效的试用天数
```

---

## 六、Application层规范

### 6.1 UseCase模式

所有订阅计划相关的UseCase位于:
`/internal/application/subscription/usecases/`

#### 创建计划: CreateSubscriptionPlanUseCase

```go
type CreateSubscriptionPlanCommand struct {
    Name         string
    Slug         string
    Description  string
    Price        uint64
    Currency     string
    BillingCycle string
    TrialDays    int
    Features     []string
    Limits       map[string]interface{}
    APIRateLimit uint
    MaxUsers     uint
    MaxProjects  uint
    IsPublic     bool
    SortOrder    int
}

// 执行: Execute(ctx context.Context, cmd) (*dto.SubscriptionPlanDTO, error)
```

执行流程:
1. 检查Slug唯一性
2. 验证BillingCycle
3. 创建SubscriptionPlan实体
4. 配置Features和Limits
5. 持久化到数据库

#### 更新计划: UpdateSubscriptionPlanUseCase

支持字段:
- Description (可选)
- Price + Currency (必须一起)
- Features + Limits (可选)
- APIRateLimit (可选)
- MaxUsers/MaxProjects (可选)
- SortOrder (可选)
- IsPublic (可选)

#### 获取计划: GetSubscriptionPlanUseCase

支持按ID或Slug获取

#### 列表计划: ListSubscriptionPlansUseCase

支持过滤:
- Status (active/inactive)
- IsPublic (true/false)
- BillingCycle
- 分页和排序

#### 公开计划: GetPublicPlansUseCase

- 返回所有active且public的计划
- 包含多个计费周期的价格选项（来自PlanPricingRepository）
- 优雅降级处理价格获取失败

#### 状态管理

- ActivateSubscriptionPlanUseCase - 激活计划
- DeactivateSubscriptionPlanUseCase - 停用计划

#### 价格管理

- GetPlanPricingsUseCase - 获取计划的所有价格选项

### 6.2 DTO定义

位置: `/internal/application/subscription/dto/dto.go`

```go
type SubscriptionPlanDTO struct {
    ID           uint
    Name         string
    Slug         string
    Description  string
    Price        uint64  // 已弃用：使用Pricings数组，保留用于向后兼容
    Currency     string
    BillingCycle string  // 已弃用：使用Pricings数组，保留用于向后兼容
    TrialDays    int
    Status       string
    Features     []string
    Limits       map[string]interface{}
    APIRateLimit uint
    MaxUsers     uint
    MaxProjects  uint
    IsPublic     bool
    SortOrder    int
    Pricings     []*PricingOptionDTO `json:"pricings,omitempty"` // 新字段：多价格选项
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type PricingOptionDTO struct {
    BillingCycle string `json:"billing_cycle"` // weekly, monthly, quarterly等
    Price        uint64 `json:"price"`         // 最小货币单位（如分）
    Currency     string `json:"currency"`      // CNY, USD等
    IsActive     bool   `json:"is_active"`     // 当前是否可用
}
```

### 6.3 DTO转换

位置: `/internal/application/subscription/dto/converters.go`

```go
// 基础转换
func ToSubscriptionPlanDTO(plan *subscription.SubscriptionPlan) *SubscriptionPlanDTO

// 转换为包含价格的DTO
func ToSubscriptionPlanDTOWithPricings(
    plan *subscription.SubscriptionPlan,
    pricings []*vo.PlanPricing,
) *SubscriptionPlanDTO

// 价格选项转换
func ToPricingOptionDTO(pricing *vo.PlanPricing) *PricingOptionDTO
func ToPricingOptionDTOList(pricings []*vo.PlanPricing) []*PricingOptionDTO
```

---

## 七、API规范

### 7.1 HTTP Handler

位置: `/internal/interfaces/http/handlers/subscriptionplanhandler.go`

#### 请求体定义

```go
type CreatePlanRequest struct {
    Name         string                 `json:"name" binding:"required"`
    Slug         string                 `json:"slug" binding:"required"`
    Description  string                 `json:"description"`
    Price        uint64                 `json:"price" binding:"required"`
    Currency     string                 `json:"currency" binding:"required"`
    BillingCycle string                 `json:"billing_cycle" binding:"required"`
    TrialDays    int                    `json:"trial_days"`
    Features     []string               `json:"features"`
    Limits       map[string]interface{} `json:"limits"`
    APIRateLimit uint                   `json:"api_rate_limit"`
    MaxUsers     uint                   `json:"max_users"`
    MaxProjects  uint                   `json:"max_projects"`
    IsPublic     bool                   `json:"is_public"`
    SortOrder    int                    `json:"sort_order"`
}

type UpdatePlanRequest struct {
    Description  *string                `json:"description"`
    Price        *uint64                `json:"price"`
    Currency     *string                `json:"currency"`
    Features     []string               `json:"features"`
    Limits       map[string]interface{} `json:"limits"`
    APIRateLimit *uint                  `json:"api_rate_limit"`
    MaxUsers     *uint                  `json:"max_users"`
    MaxProjects  *uint                  `json:"max_projects"`
    IsPublic     *bool                  `json:"is_public"`
    SortOrder    *int                   `json:"sort_order"`
}
```

### 7.2 API端点

| 方法 | 端点 | 功能 | 权限 |
|------|------|------|------|
| POST | `/subscription-plans` | 创建计划 | Bearer Token |
| GET | `/subscription-plans/:id` | 获取计划 | Public |
| GET | `/subscription-plans` | 列表计划 | Public |
| GET | `/subscription-plans/public` | 获取公开计划 | Public |
| PUT | `/subscription-plans/:id` | 更新计划 | Bearer Token (Admin) |
| PATCH | `/subscription-plans/:id/activate` | 激活计划 | Bearer Token (Admin) |
| PATCH | `/subscription-plans/:id/deactivate` | 停用计划 | Bearer Token (Admin) |
| GET | `/subscription-plans/:id/pricings` | 获取计划价格 | Public |

### 7.3 完整API示例

#### 创建计划请求

```bash
POST /api/v1/subscription-plans
Content-Type: application/json
Authorization: Bearer {token}

{
  "name": "Standard Plan",
  "slug": "standard",
  "description": "Perfect for small teams",
  "price": 2999,
  "currency": "USD",
  "billing_cycle": "monthly",
  "trial_days": 14,
  "features": ["priority_support", "advanced_analytics"],
  "limits": {
    "traffic_limit": 536870912000,    # 500GB in bytes
    "device_limit": 5,
    "speed_limit": 500,               # 500 Mbps
    "connection_limit": 100,
    "node_access": [1, 2, 3]
  },
  "api_rate_limit": 1000,
  "max_users": 50,
  "max_projects": 10,
  "is_public": true,
  "sort_order": 2
}
```

响应:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "Standard Plan",
    "slug": "standard",
    "description": "Perfect for small teams",
    "price": 2999,
    "currency": "USD",
    "billing_cycle": "monthly",
    "trial_days": 14,
    "status": "active",
    "features": ["priority_support", "advanced_analytics"],
    "limits": {
      "traffic_limit": 536870912000,
      "device_limit": 5,
      "speed_limit": 500,
      "connection_limit": 100,
      "node_access": [1, 2, 3]
    },
    "api_rate_limit": 1000,
    "max_users": 50,
    "max_projects": 10,
    "is_public": true,
    "sort_order": 2,
    "created_at": "2025-11-12T10:00:00Z",
    "updated_at": "2025-11-12T10:00:00Z"
  }
}
```

#### 获取公开计划（含多价格）

```bash
GET /api/v1/subscription-plans/public
```

响应:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "name": "Standard Plan",
      "slug": "standard",
      "...": "...",
      "pricings": [
        {
          "billing_cycle": "monthly",
          "price": 2999,
          "currency": "USD",
          "is_active": true
        },
        {
          "billing_cycle": "yearly",
          "price": 29990,
          "currency": "USD",
          "is_active": true
        }
      ]
    }
  ]
}
```

---

## 八、使用示例

### 8.1 创建带流量限制的计划

```go
package main

import (
    "context"
    "orris/internal/domain/subscription"
    vo "orris/internal/domain/subscription/value_objects"
)

func createBasicPlan() (*subscription.SubscriptionPlan, error) {
    // 1. 创建基础计划
    plan, err := subscription.NewSubscriptionPlan(
        "Basic Plan",                   // 名称
        "basic",                        // Slug
        "Perfect for individual users", // 描述
        999,                            // 价格（9.99 USD）
        "USD",                          // 货币
        vo.BillingCycleMonthly,        // 计费周期
        7,                              // 试用天数
    )
    if err != nil {
        return nil, err
    }

    // 2. 配置流量限制和特性
    features := vo.NewPlanFeatures(
        []string{"basic_support", "email_notifications"},
        map[string]interface{}{
            vo.LimitKeyTraffic:     uint64(100 * 1024 * 1024 * 1024), // 100GB
            vo.LimitKeyDeviceCount: 3,
            vo.LimitKeySpeedLimit:  100, // 100 Mbps
        },
    )

    // 3. 应用特性
    if err := plan.UpdateFeatures(features); err != nil {
        return nil, err
    }

    return plan, nil
}
```

### 8.2 验证用户流量使用

```go
func checkUserTraffic(
    plan *subscription.SubscriptionPlan,
    usedBytes uint64,
) error {
    // 检查是否无限流量
    if plan.IsUnlimitedTraffic() {
        return nil // 无限流量，允许继续
    }

    // 检查是否超出限制
    hasRemaining, err := plan.HasTrafficRemaining(usedBytes)
    if err != nil {
        return err
    }

    if !hasRemaining {
        return fmt.Errorf("traffic limit exceeded")
    }

    return nil
}
```

### 8.3 流量单位转换

```go
const (
    KB = 1024
    MB = 1024 * KB
    GB = 1024 * MB
    TB = 1024 * GB
)

// 示例
traffic100GB := uint64(100 * GB)    // 107,374,182,400 bytes
traffic500GB := uint64(500 * GB)    // 536,870,912,000 bytes
traffic1TB := uint64(1 * TB)        // 1,099,511,627,776 bytes
```

---

## 九、最佳实践

### 9.1 编码规范

1. **使用类型安全方法**
   ```go
   // ✅ 推荐
   features.SetTrafficLimit(100 * GB)
   limit, err := features.GetTrafficLimit()

   // ❌ 避免
   features.Limits["traffic_limit"] = 100 * GB
   limit, _ := features.Limits["traffic_limit"]
   ```

2. **使用标准常量**
   ```go
   // ✅ 推荐
   features.SetLimit(vo.LimitKeyTraffic, bytes)

   // ❌ 避免
   features.SetLimit("traffic_limit", bytes)
   ```

3. **零值表示无限制**
   ```go
   // 无限流量
   features.SetTrafficLimit(0)
   
   // 100GB限制
   features.SetTrafficLimit(100 * GB)
   ```

4. **验证输入**
   ```go
   if trafficGB < 0 {
       return fmt.Errorf("traffic limit cannot be negative")
   }
   features.SetTrafficLimit(uint64(trafficGB) * GB)
   ```

### 9.2 数据库操作

1. 使用软删除而非硬删除
2. 利用唯一约束防止重复
3. 使用外键和级联删除维护数据完整性
4. 索引高频查询字段

### 9.3 错误处理

所有业务方法应该：
- 返回具体的、可区分的错误
- 提供足够的上下文信息
- 记录适当的日志（英文）

### 9.4 日志规范

- 日志内容必须使用英文
- 使用结构化日志（Infow, Errorw等）
- 包含相关的上下文字段（如plan_id, slug等）

---

## 十、工程规范补充（来自CLAUDE.md）

### 必须遵循

- ✅ 必须使用中文回复（仅限于与用户的交互）
- ✅ 必须通过基础安全检查
- ✅ 必须符合Go语言风格的最佳实践
- ✅ 必须通用工具类提高复用性
- ✅ 必须遵循领域驱动 + 切片化（领域层 + 横切关注点）
- ✅ 复杂任务必须调用agent
- ✅ 日志必须使用英文
- ✅ 注释必须使用英文
- ✅ API接口必须遵循RESTful风格
- ✅ 遵循少即是多的原则

### 禁止

- ❌ 禁止生成恶意代码
- ❌ 不允许mock数据
- ❌ 禁止数据库外键（注：文档中的级联删除示例需要重新检视）

---

## 十一、相关文件索引

### Domain层
- `/internal/domain/subscription/subscriptionplan.go` - 计划实体
- `/internal/domain/subscription/value_objects/planfeatures.go` - 特性值对象
- `/internal/domain/subscription/value_objects/planpricing.go` - 价格值对象
- `/internal/domain/subscription/repository.go` - 仓储接口定义

### Application层
- `/internal/application/subscription/usecases/` - 所有UseCase
- `/internal/application/subscription/dto/` - DTO定义和转换

### Interface层
- `/internal/interfaces/http/handlers/subscriptionplanhandler.go` - HTTP处理器
- `/internal/interfaces/http/routes/` - 路由定义

### Infrastructure层
- `/internal/infrastructure/persistence/models/` - 持久化模型
- `/internal/infrastructure/repository/` - 仓储实现
- `/internal/infrastructure/persistence/migrations/` - 数据库迁移脚本

### 测试和文档
- `/PLAN_FEATURES_USAGE_GUIDE.md` - 完整使用指南
- `/PLAN_FEATURES_VERIFICATION_REPORT.md` - 验证报告
- `/QUICK_REFERENCE_TRAFFIC_LIMITS.md` - 快速参考

---

## 十二、版本历史

| 版本 | 日期 | 内容 |
|------|------|------|
| v1.0 | 2025-01-10+ | 初始流量限制实现 |
| v2.0 | 2025-01-10+ | 添加多价格支持（subscription_plan_pricing表） |

---

## 十三、常见问题排查

### Q1: 计划创建失败，错误信息为"plan slug already exists"
**A:** Slug已被占用。解决方案：
- 检查数据库中是否存在相同Slug
- 使用不同的Slug重试
- 如果是测试数据，清理后重试

### Q2: 如何检查用户是否超出流量限制？
**A:** 
```go
hasRemaining, err := plan.HasTrafficRemaining(usedBytes)
if !hasRemaining {
    // 用户已超出限制
}
```

### Q3: 计划的Price和BillingCycle字段已弃用，如何使用新的多价格系统？
**A:** 
- 使用GetPublicPlansUseCase获取计划
- 从返回的SubscriptionPlanDTO中读取Pricings数组
- 每个元素包含不同计费周期的价格信息

### Q4: 如何创建无限流量的计划？
**A:**
```go
features.SetTrafficLimit(0) // 0 = 无限流量
```

---

**文档编制完成**
