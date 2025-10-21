# 订阅管理领域功能文档

## 概述

订阅管理领域（Subscription Domain）是 SaaS 业务的核心模块，负责管理订阅计划、用户订阅、访问令牌和权限控制。该领域遵循领域驱动设计（DDD）原则，与用户领域和权限领域深度集成，提供完整的 SaaS 订阅解决方案。

## 领域模型

### 聚合根

#### Subscription 聚合（用户订阅）

用户订阅聚合管理用户与订阅计划的关联关系，以及订阅的完整生命周期。

**核心属性：**
- `id`: 订阅唯一标识
- `userID`: 关联用户 ID
- `planID`: 关联订阅计划 ID
- `status`: 订阅状态（值对象）
- `startDate`: 订阅开始日期
- `endDate`: 订阅结束日期
- `autoRenew`: 是否自动续订
- `currentPeriodStart`: 当前计费周期开始时间
- `currentPeriodEnd`: 当前计费周期结束时间
- `cancelledAt`: 取消时间
- `cancelReason`: 取消原因
- `metadata`: 元数据（JSON，扩展字段）
- `version`: 乐观锁版本号
- `createdAt`: 创建时间
- `updatedAt`: 更新时间

**业务方法：**

```go
// 激活订阅
func (s *Subscription) Activate() error

// 取消订阅
func (s *Subscription) Cancel(reason string) error

// 续订
func (s *Subscription) Renew(endDate time.Time) error

// 升级计划
func (s *Subscription) UpgradePlan(newPlanID uint) error

// 降级计划
func (s *Subscription) DowngradePlan(newPlanID uint) error

// 检查是否过期
func (s *Subscription) IsExpired() bool

// 检查是否激活
func (s *Subscription) IsActive() bool

// 标记为过期
func (s *Subscription) MarkAsExpired() error

// 更新自动续订设置
func (s *Subscription) SetAutoRenew(autoRenew bool)
```

#### SubscriptionPlan 聚合（订阅计划）

订阅计划聚合定义不同的订阅套餐，包括价格、功能特性、访问限制等。

**核心属性：**
- `id`: 计划唯一标识
- `name`: 计划名称（如"专业版"）
- `slug`: 计划标识符（如"pro"）
- `description`: 计划描述
- `price`: 价格（单位：分）
- `currency`: 货币代码（如"CNY"、"USD"）
- `billingCycle`: 计费周期（值对象）
- `trialDays`: 试用天数
- `status`: 计划状态（active/inactive）
- `features`: 功能特性列表（JSON 或关联表）
- `limits`: 使用限制配置（JSON）
- `customEndpoint`: 自定义 API 端点路径
- `apiRateLimit`: API 速率限制（请求/分钟）
- `maxUsers`: 最大用户数
- `maxProjects`: 最大项目数
- `storageLimit`: 存储限制（MB）
- `isPublic`: 是否公开可见
- `sortOrder`: 排序顺序
- `metadata`: 元数据（JSON）
- `createdAt`: 创建时间
- `updatedAt`: 更新时间

**特性配置示例：**
```json
{
  "features": [
    "advanced_analytics",
    "priority_support",
    "custom_branding",
    "api_access",
    "webhook_integration"
  ],
  "limits": {
    "api_requests_per_day": 10000,
    "max_team_members": 50,
    "max_projects": 100,
    "storage_gb": 500,
    "concurrent_builds": 10
  }
}
```

**业务方法：**

```go
// 激活计划
func (p *SubscriptionPlan) Activate() error

// 停用计划
func (p *SubscriptionPlan) Deactivate() error

// 更新价格
func (p *SubscriptionPlan) UpdatePrice(price uint64, currency string) error

// 更新功能特性
func (p *SubscriptionPlan) UpdateFeatures(features []string) error

// 更新使用限制
func (p *SubscriptionPlan) UpdateLimits(limits map[string]interface{}) error

// 设置自定义端点
func (p *SubscriptionPlan) SetCustomEndpoint(endpoint string) error

// 设置 API 速率限制
func (p *SubscriptionPlan) SetAPIRateLimit(limit uint) error

// 检查是否包含特性
func (p *SubscriptionPlan) HasFeature(feature string) bool

// 获取限制值
func (p *SubscriptionPlan) GetLimit(key string) (interface{}, bool)
```

### 实体

#### SubscriptionToken（订阅访问令牌）

订阅令牌实体用于访问认证和 API 访问控制。

**属性：**
- `ID`: 令牌 ID
- `SubscriptionID`: 关联订阅 ID
- `Name`: 令牌名称（用于标识）
- `TokenHash`: 令牌哈希值（SHA256）
- `Prefix`: 令牌前缀（用于显示，如 "sk_live_xxx"）
- `Scope`: 令牌作用域（值对象）
- `ExpiresAt`: 过期时间（nil 表示永不过期）
- `LastUsedAt`: 最后使用时间
- `LastUsedIP`: 最后使用 IP
- `UsageCount`: 使用次数
- `IsActive`: 是否激活
- `CreatedAt`: 创建时间
- `RevokedAt`: 撤销时间

**业务方法：**

```go
// 验证令牌
func (t *SubscriptionToken) Verify(plainToken string) bool

// 检查是否过期
func (t *SubscriptionToken) IsExpired() bool

// 撤销令牌
func (t *SubscriptionToken) Revoke() error

// 记录使用
func (t *SubscriptionToken) RecordUsage(ipAddress string)

// 检查作用域权限
func (t *SubscriptionToken) HasScope(scope string) bool
```

#### SubscriptionHistory（订阅历史记录）

记录订阅的变更历史，用于审计和分析。

**属性：**
- `ID`: 记录 ID
- `SubscriptionID`: 关联订阅 ID
- `EventType`: 事件类型（created, activated, cancelled, renewed, plan_changed）
- `OldPlanID`: 旧计划 ID（计划变更时）
- `NewPlanID`: 新计划 ID（计划变更时）
- `Reason`: 原因说明
- `Metadata`: 额外信息（JSON）
- `CreatedAt`: 记录时间

### 值对象

#### BillingCycle（计费周期）

```go
type BillingCycle string

const (
    BillingCycleMonthly  BillingCycle = "monthly"   // 月付
    BillingCycleQuarterly BillingCycle = "quarterly" // 季付
    BillingCycleYearly   BillingCycle = "yearly"    // 年付
    BillingCycleLifetime BillingCycle = "lifetime"  // 终身
)

// 获取周期天数
func (bc BillingCycle) Days() int

// 获取下一个计费日期
func (bc BillingCycle) NextBillingDate(from time.Time) time.Time
```

#### SubscriptionStatus（订阅状态）

```go
type SubscriptionStatus string

const (
    SubscriptionStatusActive    SubscriptionStatus = "active"     // 激活
    SubscriptionStatusInactive  SubscriptionStatus = "inactive"   // 未激活
    SubscriptionStatusTrialing  SubscriptionStatus = "trialing"   // 试用中
    SubscriptionStatusPastDue   SubscriptionStatus = "past_due"   // 逾期
    SubscriptionStatusExpired   SubscriptionStatus = "expired"    // 已过期
    SubscriptionStatusCancelled SubscriptionStatus = "cancelled"  // 已取消
)

// 状态转换规则
var SubscriptionStatusTransitions = map[SubscriptionStatus][]SubscriptionStatus{
    SubscriptionStatusInactive: {
        SubscriptionStatusActive,
        SubscriptionStatusTrialing,
    },
    SubscriptionStatusTrialing: {
        SubscriptionStatusActive,
        SubscriptionStatusExpired,
        SubscriptionStatusCancelled,
    },
    SubscriptionStatusActive: {
        SubscriptionStatusPastDue,
        SubscriptionStatusExpired,
        SubscriptionStatusCancelled,
    },
    SubscriptionStatusPastDue: {
        SubscriptionStatusActive,
        SubscriptionStatusExpired,
        SubscriptionStatusCancelled,
    },
    SubscriptionStatusExpired: {
        SubscriptionStatusActive, // 可以续订重新激活
    },
    SubscriptionStatusCancelled: {
        // 已取消不可转换
    },
}

// 检查是否可以使用服务
func (s SubscriptionStatus) CanUseService() bool

// 检查是否可以续订
func (s SubscriptionStatus) CanRenew() bool

// 检查状态转换
func (s SubscriptionStatus) CanTransitionTo(target SubscriptionStatus) bool
```

#### TokenScope（令牌作用域）

```go
type TokenScope string

const (
    TokenScopeFull      TokenScope = "full"        // 完整访问
    TokenScopeReadOnly  TokenScope = "read_only"   // 只读访问
    TokenScopeAPI       TokenScope = "api"         // API 访问
    TokenScopeWebhook   TokenScope = "webhook"     // Webhook 回调
    TokenScopeAdmin     TokenScope = "admin"       // 管理员权限
)

// 检查是否有权限执行操作
func (ts TokenScope) CanPerform(action string) bool
```

#### PlanFeatures（计划特性）

```go
type PlanFeatures struct {
    Features []string               // 功能特性列表
    Limits   map[string]interface{} // 使用限制
}

// 检查是否包含特性
func (pf *PlanFeatures) HasFeature(feature string) bool

// 获取限制值
func (pf *PlanFeatures) GetLimit(key string) (interface{}, bool)

// 检查是否超出限制
func (pf *PlanFeatures) IsWithinLimit(key string, value interface{}) bool
```

## 核心功能

### 1. 订阅计划管理

#### 创建订阅计划

**用例：** `CreateSubscriptionPlanUseCase`

**流程：**
1. 验证计划名称和标识符唯一性
2. 验证价格和计费周期
3. 创建计划聚合
4. 设置功能特性和限制
5. 持久化计划
6. 触发计划创建事件

**输入：**
```go
type CreateSubscriptionPlanCommand struct {
    Name            string
    Slug            string
    Description     string
    Price           uint64
    Currency        string
    BillingCycle    string
    TrialDays       int
    Features        []string
    Limits          map[string]interface{}
    CustomEndpoint  string
    APIRateLimit    uint
    MaxUsers        uint
    MaxProjects     uint
    StorageLimit    uint64
    IsPublic        bool
}
```

#### 更新订阅计划

**用例：** `UpdateSubscriptionPlanUseCase`

**可更新内容：**
- 计划描述
- 价格（仅影响新订阅）
- 功能特性
- 使用限制
- API 速率限制
- 自定义端点

**注意事项：**
- 价格修改不影响现有订阅
- 限制收紧需要通知现有用户
- 功能移除需要兼容性处理

#### 列出订阅计划

**用例：** `ListSubscriptionPlansUseCase`

**支持功能：**
- 公开计划列表（供用户选择）
- 管理员查看所有计划
- 按价格排序
- 按受欢迎度排序
- 过滤激活/停用计划

### 2. 用户订阅管理

#### 创建订阅（购买）

**用例：** `CreateSubscriptionUseCase`

**流程：**
1. 验证用户身份和权限
2. 验证订阅计划是否存在且激活
3. 检查用户是否已有激活订阅
4. 创建订阅聚合
5. 设置订阅周期
6. 如果有试用期，状态设为 trialing
7. 生成默认访问令牌
8. 持久化订阅
9. 同步 RBAC 权限（基于计划特性）
10. 触发订阅创建事件
11. 发送确认邮件

**输入：**
```go
type CreateSubscriptionCommand struct {
    UserID      uint
    PlanID      uint
    StartDate   time.Time
    AutoRenew   bool
    PaymentInfo map[string]interface{} // 支付信息
}
```

#### 激活订阅

**用例：** `ActivateSubscriptionUseCase`

**触发条件：**
- 支付成功后
- 试用期转正式订阅
- 从过期状态续订

**流程：**
1. 验证订阅状态可以激活
2. 设置当前计费周期
3. 更新状态为 active
4. 同步 RBAC 权限
5. 触发激活事件
6. 发送激活通知

#### 取消订阅

**用例：** `CancelSubscriptionUseCase`

**取消策略：**
- **立即取消**：立即失效，可能退款
- **周期结束取消**：当前周期继续使用，到期后不续订

**流程：**
1. 验证用户权限（只能取消自己的订阅或管理员）
2. 验证订阅状态可以取消
3. 记录取消原因
4. 更新订阅状态
5. 如果立即取消，撤销 RBAC 权限
6. 撤销所有访问令牌
7. 触发取消事件
8. 发送取消确认邮件

**输入：**
```go
type CancelSubscriptionCommand struct {
    SubscriptionID uint
    UserID         uint   // 操作用户
    Reason         string
    Immediate      bool   // 是否立即取消
}
```

#### 续订订阅

**用例：** `RenewSubscriptionUseCase`

**续订类型：**
- **自动续订**：定时任务自动执行
- **手动续订**：用户主动续订

**流程：**
1. 验证订阅可以续订
2. 计算新的计费周期
3. 处理支付（如需）
4. 更新订阅结束时间
5. 重置状态为 active
6. 触发续订事件
7. 发送续订通知

**自动续订检查：**
```go
// 定时任务每天检查即将到期的订阅
func CheckExpiringSubscriptions() {
    // 查询 7 天内到期且自动续订的订阅
    subscriptions := repo.FindExpiringSubscriptions(7)

    for _, sub := range subscriptions {
        if sub.AutoRenew {
            renewUseCase.Execute(RenewSubscriptionCommand{
                SubscriptionID: sub.ID,
                IsAutoRenew:    true,
            })
        }
    }
}
```

#### 升级/降级订阅计划

**用例：** `ChangePlanUseCase`

**升级场景：**
- 立即生效
- 按比例退款或补差价
- 立即获得新计划权限

**降级场景：**
- 周期结束后生效（推荐）
- 或立即生效，按比例退款
- 权限变更需要兼容性处理

**流程：**
1. 验证新旧计划有效性
2. 计算费用差异
3. 处理退款或补款
4. 更新订阅计划 ID
5. 更新计费周期（如计费周期不同）
6. 同步 RBAC 权限
7. 更新访问令牌作用域
8. 记录计划变更历史
9. 触发计划变更事件
10. 发送变更通知

**输入：**
```go
type ChangePlanCommand struct {
    SubscriptionID uint
    NewPlanID      uint
    ChangeType     string // "upgrade" or "downgrade"
    EffectiveDate  string // "immediate" or "period_end"
}
```

### 3. 访问令牌管理

#### 生成订阅令牌

**用例：** `GenerateSubscriptionTokenUseCase`

**流程：**
1. 验证用户拥有该订阅
2. 验证订阅状态为激活
3. 生成随机令牌（32字节）
4. 创建令牌前缀（如 `sk_live_`）
5. 对令牌进行 SHA256 哈希
6. 设置令牌作用域和过期时间
7. 持久化令牌（仅存储哈希）
8. 触发令牌生成事件
9. 返回明文令牌（仅此一次显示）

**令牌格式：**
```
token_prod_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
│     │    └─────────────────────────────┘
│     │              令牌主体（32字节随机）
│     └─── 环境（prod/test）
└────── 令牌类型（token）
```

**输入：**
```go
type GenerateSubscriptionTokenCommand struct {
    SubscriptionID uint
    UserID         uint
    Name           string      // 令牌名称，如 "Production API"
    Scope          TokenScope
    ExpiresAt      *time.Time  // nil 表示永不过期
}
```

**输出：**
```go
type GenerateSubscriptionTokenResult struct {
    TokenID    uint
    Token      string      // 明文令牌（仅返回一次）
    Prefix     string      // 令牌前缀，用于显示
    ExpiresAt  *time.Time
    CreatedAt  time.Time
}
```

#### 验证订阅令牌

**用例：** `ValidateSubscriptionTokenUseCase`

**验证流程：**
1. 从请求头提取令牌
2. 对令牌进行 SHA256 哈希
3. 查询令牌记录
4. 检查令牌是否激活
5. 检查令牌是否过期
6. 检查关联订阅状态
7. 检查令牌作用域权限
8. 检查 API 速率限制
9. 记录令牌使用
10. 返回订阅和用户信息

**API 中间件集成：**
```go
func SubscriptionTokenMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "missing token"})
            c.Abort()
            return
        }

        // 移除 "Bearer " 前缀
        token = strings.TrimPrefix(token, "Bearer ")

        result, err := validateTokenUseCase.Execute(ValidateTokenCommand{
            Token:      token,
            RequiredScope: TokenScopeAPI,
        })

        if err != nil {
            c.JSON(401, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        // 检查订阅状态
        if !result.Subscription.IsActive() {
            c.JSON(403, gin.H{"error": "subscription inactive"})
            c.Abort()
            return
        }

        // 注入上下文
        c.Set("user_id", result.Subscription.UserID)
        c.Set("subscription_id", result.Subscription.ID)
        c.Set("subscription_plan", result.Plan)

        c.Next()
    }
}
```

#### 撤销订阅令牌

**用例：** `RevokeSubscriptionTokenUseCase`

**流程：**
1. 验证用户权限
2. 查询令牌
3. 标记为已撤销
4. 记录撤销时间
5. 触发令牌撤销事件

**批量撤销：**
- 取消订阅时撤销所有令牌
- 订阅过期时撤销所有令牌
- 计划降级时撤销超出权限的令牌

#### 刷新订阅令牌

**用例：** `RefreshSubscriptionTokenUseCase`

**场景：**
- 令牌即将过期
- 定期轮换令牌（安全最佳实践）

**流程：**
1. 验证旧令牌
2. 生成新令牌
3. 保留旧令牌一段时间（宽限期）
4. 返回新令牌

### 4. 自定义 API 端点路径

#### 配置自定义端点

每个订阅计划可以配置自定义的 API 访问路径，实现多租户隔离。

**示例：**
```go
// 免费计划
plan.CustomEndpoint = "/api/v1/free"

// 专业计划
plan.CustomEndpoint = "/api/v1/pro"

// 企业计划（支持自定义子域名）
plan.CustomEndpoint = "https://{tenant}.api.example.com"
```

**路由注册：**
```go
func RegisterSubscriptionRoutes(router *gin.Engine, plans []*SubscriptionPlan) {
    for _, plan := range plans {
        group := router.Group(plan.CustomEndpoint)
        {
            group.Use(SubscriptionTokenMiddleware())
            group.Use(PlanFeatureMiddleware(plan))

            // 注册端点
            group.GET("/data", dataHandler.GetData)
            group.POST("/data", dataHandler.CreateData)
            group.GET("/analytics", analyticsHandler.GetAnalytics)
        }
    }
}
```

**动态路由解析：**
```go
func PlanFeatureMiddleware(plan *SubscriptionPlan) gin.HandlerFunc {
    return func(c *gin.Context) {
        subscription := c.MustGet("subscription").(*Subscription)

        // 验证订阅计划匹配
        if subscription.PlanID != plan.ID {
            c.JSON(403, gin.H{"error": "plan mismatch"})
            c.Abort()
            return
        }

        // 注入计划特性
        c.Set("plan_features", plan.Features)
        c.Set("plan_limits", plan.Limits)

        c.Next()
    }
}
```

### 5. 基于订阅的 RBAC 权限控制

#### 权限映射策略

订阅计划的功能特性自动映射到 RBAC 权限。

**映射配置：**
```go
var FeatureToPermissionMap = map[string][]string{
    "advanced_analytics": {
        "analytics:view_advanced",
        "analytics:export",
    },
    "api_access": {
        "api:access",
        "api:create_token",
    },
    "webhook_integration": {
        "webhook:create",
        "webhook:update",
        "webhook:delete",
    },
    "custom_branding": {
        "branding:customize",
        "branding:upload_logo",
    },
}
```

#### 权限同步用例

**用例：** `SyncSubscriptionPermissionsUseCase`

**触发时机：**
- 创建订阅时
- 激活订阅时
- 变更订阅计划时
- 订阅过期时

**流程：**
```go
func (uc *SyncSubscriptionPermissionsUseCase) Execute(ctx context.Context, subscriptionID uint) error {
    // 1. 获取订阅和计划
    subscription, err := uc.subscriptionRepo.GetByID(ctx, subscriptionID)
    if err != nil {
        return err
    }

    plan, err := uc.planRepo.GetByID(ctx, subscription.PlanID)
    if err != nil {
        return err
    }

    // 2. 移除所有订阅相关权限
    err = uc.permissionService.RemoveSubscriptionPermissions(ctx, subscription.UserID)
    if err != nil {
        return err
    }

    // 3. 如果订阅激活，添加新权限
    if subscription.IsActive() {
        permissions := uc.mapFeaturesToPermissions(plan.Features)
        err = uc.permissionService.GrantPermissions(ctx, subscription.UserID, permissions)
        if err != nil {
            return err
        }
    }

    return nil
}
```

#### 权限检查中间件

结合订阅和 RBAC 的双重检查：

```go
func RequireSubscriptionFeature(feature string) gin.HandlerFunc {
    return func(c *gin.Context) {
        subscription := c.MustGet("subscription").(*Subscription)
        plan := c.MustGet("subscription_plan").(*SubscriptionPlan)

        // 检查订阅计划是否包含该特性
        if !plan.HasFeature(feature) {
            c.JSON(403, gin.H{
                "error": "feature not available in your plan",
                "feature": feature,
                "upgrade_url": "/subscription/upgrade",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// 使用示例
router.GET("/analytics/advanced",
    SubscriptionTokenMiddleware(),
    RequireSubscriptionFeature("advanced_analytics"),
    analyticsHandler.GetAdvancedAnalytics,
)
```

### 6. API 速率限制

基于订阅计划的动态速率限制。

**限制策略：**
```go
type RateLimitConfig struct {
    RequestsPerMinute int
    RequestsPerHour   int
    RequestsPerDay    int
    BurstSize         int
}

// 不同计划的限制
var PlanRateLimits = map[string]RateLimitConfig{
    "free": {
        RequestsPerMinute: 10,
        RequestsPerHour:   100,
        RequestsPerDay:    1000,
        BurstSize:         5,
    },
    "basic": {
        RequestsPerMinute: 60,
        RequestsPerHour:   1000,
        RequestsPerDay:    10000,
        BurstSize:         20,
    },
    "pro": {
        RequestsPerMinute: 300,
        RequestsPerHour:   10000,
        RequestsPerDay:    100000,
        BurstSize:         100,
    },
    "enterprise": {
        RequestsPerMinute: 0, // 无限制
        RequestsPerHour:   0,
        RequestsPerDay:    0,
        BurstSize:         0,
    },
}
```

**限流中间件：**
```go
func SubscriptionRateLimitMiddleware(limiter RateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        subscription := c.MustGet("subscription").(*Subscription)
        plan := c.MustGet("subscription_plan").(*SubscriptionPlan)

        // 获取限制配置
        config := PlanRateLimits[plan.Slug]

        // 检查速率限制
        allowed, err := limiter.Allow(subscription.ID, config)
        if err != nil {
            c.JSON(500, gin.H{"error": "rate limit check failed"})
            c.Abort()
            return
        }

        if !allowed {
            c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerMinute))
            c.Header("X-RateLimit-Remaining", "0")
            c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

            c.JSON(429, gin.H{
                "error": "rate limit exceeded",
                "upgrade_url": "/subscription/upgrade",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 7. 使用量跟踪

跟踪订阅的各项使用指标，用于计费和限制。

**使用量指标：**
```go
type SubscriptionUsage struct {
    SubscriptionID uint
    Period         time.Time // 计费周期

    // API 使用量
    APIRequests    uint64
    APIDataOut     uint64 // 字节
    APIDataIn      uint64

    // 资源使用量
    StorageUsed    uint64 // MB
    UsersCount     uint
    ProjectsCount  uint

    // 功能使用量
    WebhookCalls   uint64
    EmailsSent     uint64
    ReportsGenerated uint

    UpdatedAt      time.Time
}
```

**使用量检查中间件：**
```go
func CheckUsageLimits() gin.HandlerFunc {
    return func(c *gin.Context) {
        subscription := c.MustGet("subscription").(*Subscription)
        plan := c.MustGet("subscription_plan").(*SubscriptionPlan)

        usage, err := usageRepo.GetCurrentUsage(subscription.ID)
        if err != nil {
            c.JSON(500, gin.H{"error": "failed to check usage"})
            c.Abort()
            return
        }

        // 检查各项限制
        if limit, ok := plan.GetLimit("max_projects"); ok {
            if usage.ProjectsCount >= limit.(uint) {
                c.JSON(403, gin.H{
                    "error": "project limit exceeded",
                    "limit": limit,
                    "current": usage.ProjectsCount,
                    "upgrade_url": "/subscription/upgrade",
                })
                c.Abort()
                return
            }
        }

        c.Next()
    }
}
```

## 领域事件

### 订阅事件

#### SubscriptionCreatedEvent
订阅创建事件。

**字段：**
```go
type SubscriptionCreatedEvent struct {
    SubscriptionID uint
    UserID         uint
    PlanID         uint
    Status         string
    StartDate      time.Time
    EndDate        time.Time
    Timestamp      time.Time
}
```

**触发时机：** 用户购买订阅时

**事件处理：**
- 发送欢迎邮件
- 初始化用户数据
- 记录订阅历史
- 同步 RBAC 权限

#### SubscriptionActivatedEvent
订阅激活事件。

**字段：**
```go
type SubscriptionActivatedEvent struct {
    SubscriptionID uint
    UserID         uint
    PlanID         uint
    ActivatedAt    time.Time
    Timestamp      time.Time
}
```

**触发时机：**
- 支付成功后
- 试用转正式
- 续订成功

**事件处理：**
- 发送激活通知
- 启用功能权限
- 记录激活日志

#### SubscriptionExpiredEvent
订阅过期事件。

**字段：**
```go
type SubscriptionExpiredEvent struct {
    SubscriptionID uint
    UserID         uint
    PlanID         uint
    ExpiredAt      time.Time
    Reason         string // auto_expire, payment_failed
    Timestamp      time.Time
}
```

**触发时机：**
- 到期未续订
- 支付失败

**事件处理：**
- 撤销访问权限
- 撤销所有令牌
- 发送过期通知
- 数据降级处理（如有）

#### SubscriptionCancelledEvent
订阅取消事件。

**字段：**
```go
type SubscriptionCancelledEvent struct {
    SubscriptionID uint
    UserID         uint
    PlanID         uint
    Reason         string
    CancelledBy    uint   // 操作用户 ID
    Immediate      bool
    Timestamp      time.Time
}
```

**触发时机：** 用户或管理员取消订阅

**事件处理：**
- 发送取消确认
- 收集反馈
- 数据保留处理
- 权限撤销（如立即取消）

#### SubscriptionRenewedEvent
订阅续订事件。

**字段：**
```go
type SubscriptionRenewedEvent struct {
    SubscriptionID uint
    UserID         uint
    PlanID         uint
    OldEndDate     time.Time
    NewEndDate     time.Time
    IsAutoRenew    bool
    Timestamp      time.Time
}
```

**触发时机：**
- 自动续订成功
- 手动续订

**事件处理：**
- 发送续订确认
- 生成发票
- 记录续订历史

#### SubscriptionPlanChangedEvent
订阅计划变更事件。

**字段：**
```go
type SubscriptionPlanChangedEvent struct {
    SubscriptionID uint
    UserID         uint
    OldPlanID      uint
    NewPlanID      uint
    ChangeType     string // upgrade, downgrade
    EffectiveDate  time.Time
    Timestamp      time.Time
}
```

**触发时机：** 升级或降级订阅计划

**事件处理：**
- 同步 RBAC 权限
- 更新令牌作用域
- 发送变更通知
- 记录变更历史
- 处理退款或补款

### 令牌事件

#### SubscriptionTokenGeneratedEvent
订阅令牌生成事件。

**字段：**
```go
type SubscriptionTokenGeneratedEvent struct {
    TokenID        uint
    SubscriptionID uint
    UserID         uint
    Scope          string
    ExpiresAt      *time.Time
    Timestamp      time.Time
}
```

**触发时机：** 生成新访问令牌

**事件处理：**
- 记录审计日志
- 发送安全通知

#### SubscriptionTokenRevokedEvent
订阅令牌撤销事件。

**字段：**
```go
type SubscriptionTokenRevokedEvent struct {
    TokenID        uint
    SubscriptionID uint
    Reason         string
    RevokedBy      uint
    Timestamp      time.Time
}
```

**触发时机：**
- 手动撤销
- 订阅取消/过期
- 安全事件

**事件处理：**
- 记录审计日志
- 通知相关集成

## 仓储接口

### SubscriptionRepository

```go
type SubscriptionRepository interface {
    // 基础操作
    Create(ctx context.Context, subscription *Subscription) error
    GetByID(ctx context.Context, id uint) (*Subscription, error)
    Update(ctx context.Context, subscription *Subscription) error
    Delete(ctx context.Context, id uint) error

    // 查询
    GetByUserID(ctx context.Context, userID uint) ([]*Subscription, error)
    GetActiveByUserID(ctx context.Context, userID uint) (*Subscription, error)
    FindExpiringSubscriptions(ctx context.Context, days int) ([]*Subscription, error)
    FindExpiredSubscriptions(ctx context.Context) ([]*Subscription, error)

    // 列表和分页
    List(ctx context.Context, filter SubscriptionFilter) ([]*Subscription, int64, error)

    // 统计
    CountByPlanID(ctx context.Context, planID uint) (int64, error)
    CountByStatus(ctx context.Context, status SubscriptionStatus) (int64, error)
}

type SubscriptionFilter struct {
    UserID   *uint
    PlanID   *uint
    Status   *SubscriptionStatus
    Page     int
    PageSize int
    SortBy   string
    SortDesc bool
}
```

### SubscriptionPlanRepository

```go
type SubscriptionPlanRepository interface {
    // 基础操作
    Create(ctx context.Context, plan *SubscriptionPlan) error
    GetByID(ctx context.Context, id uint) (*SubscriptionPlan, error)
    GetBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error)
    Update(ctx context.Context, plan *SubscriptionPlan) error
    Delete(ctx context.Context, id uint) error

    // 查询
    GetActivePublicPlans(ctx context.Context) ([]*SubscriptionPlan, error)
    GetAllActive(ctx context.Context) ([]*SubscriptionPlan, error)
    List(ctx context.Context, filter PlanFilter) ([]*SubscriptionPlan, int64, error)

    // 验证
    ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

type PlanFilter struct {
    Status       *string
    IsPublic     *bool
    BillingCycle *BillingCycle
    Page         int
    PageSize     int
    SortBy       string
}
```

### SubscriptionTokenRepository

```go
type SubscriptionTokenRepository interface {
    // 基础操作
    Create(token *SubscriptionToken) error
    GetByID(id uint) (*SubscriptionToken, error)
    GetByTokenHash(tokenHash string) (*SubscriptionToken, error)
    Update(token *SubscriptionToken) error
    Delete(id uint) error

    // 查询
    GetBySubscriptionID(subscriptionID uint) ([]*SubscriptionToken, error)
    GetActiveBySubscriptionID(subscriptionID uint) ([]*SubscriptionToken, error)

    // 批量操作
    RevokeAllBySubscriptionID(subscriptionID uint) error
    DeleteExpiredTokens() error
}
```

### SubscriptionUsageRepository

```go
type SubscriptionUsageRepository interface {
    GetCurrentUsage(subscriptionID uint) (*SubscriptionUsage, error)
    IncrementAPIRequests(subscriptionID uint, count uint64) error
    IncrementStorageUsed(subscriptionID uint, bytes uint64) error
    GetUsageHistory(subscriptionID uint, from, to time.Time) ([]*SubscriptionUsage, error)
    ResetUsage(subscriptionID uint) error
}
```

## 应用层用例

### 订阅管理用例

| 用例 | 描述 | 命令/查询 |
|------|------|----------|
| CreateSubscriptionUseCase | 创建订阅 | CreateSubscriptionCommand |
| ActivateSubscriptionUseCase | 激活订阅 | ActivateSubscriptionCommand |
| CancelSubscriptionUseCase | 取消订阅 | CancelSubscriptionCommand |
| RenewSubscriptionUseCase | 续订订阅 | RenewSubscriptionCommand |
| ChangePlanUseCase | 变更计划 | ChangePlanCommand |
| GetSubscriptionUseCase | 获取订阅详情 | GetSubscriptionQuery |
| ListUserSubscriptionsUseCase | 列出用户订阅 | ListUserSubscriptionsQuery |
| CheckSubscriptionStatusUseCase | 检查订阅状态 | CheckSubscriptionStatusQuery |

### 订阅计划用例

| 用例 | 描述 | 命令/查询 |
|------|------|----------|
| CreateSubscriptionPlanUseCase | 创建订阅计划 | CreateSubscriptionPlanCommand |
| UpdateSubscriptionPlanUseCase | 更新订阅计划 | UpdateSubscriptionPlanCommand |
| ActivatePlanUseCase | 激活计划 | ActivatePlanCommand |
| DeactivatePlanUseCase | 停用计划 | DeactivatePlanCommand |
| GetSubscriptionPlanUseCase | 获取计划详情 | GetSubscriptionPlanQuery |
| ListSubscriptionPlansUseCase | 列出订阅计划 | ListSubscriptionPlansQuery |
| GetPublicPlansUseCase | 获取公开计划 | GetPublicPlansQuery |

### 令牌管理用例

| 用例 | 描述 | 命令/查询 |
|------|------|----------|
| GenerateSubscriptionTokenUseCase | 生成令牌 | GenerateSubscriptionTokenCommand |
| ValidateSubscriptionTokenUseCase | 验证令牌 | ValidateSubscriptionTokenCommand |
| RevokeSubscriptionTokenUseCase | 撤销令牌 | RevokeSubscriptionTokenCommand |
| RefreshSubscriptionTokenUseCase | 刷新令牌 | RefreshSubscriptionTokenCommand |
| ListSubscriptionTokensUseCase | 列出令牌 | ListSubscriptionTokensQuery |

### 权限同步用例

| 用例 | 描述 | 命令 |
|------|------|------|
| SyncSubscriptionPermissionsUseCase | 同步订阅权限 | SyncSubscriptionPermissionsCommand |
| GrantSubscriptionFeaturesUseCase | 授予订阅特性权限 | GrantSubscriptionFeaturesCommand |
| RevokeSubscriptionFeaturesUseCase | 撤销订阅特性权限 | RevokeSubscriptionFeaturesCommand |

## RBAC 权限定义

### 订阅资源（subscription）

| 权限 | 描述 | 默认角色 |
|------|------|---------|
| subscription:create | 创建订阅 | user, admin |
| subscription:read | 查看自己的订阅 | user, admin |
| subscription:update | 更新自己的订阅 | user, admin |
| subscription:delete | 删除自己的订阅 | user, admin |
| subscription:manage_all | 管理所有用户订阅 | admin |
| subscription:configure | 配置订阅高级设置 | admin |
| subscription:access_data | 访问订阅数据 | user, admin |
| subscription:view_usage | 查看使用统计 | user, admin |

### 订阅计划资源（subscription_plan）

| 权限 | 描述 | 默认角色 |
|------|------|---------|
| subscription_plan:create | 创建订阅计划 | admin |
| subscription_plan:read | 查看订阅计划 | user, admin |
| subscription_plan:update | 更新订阅计划 | admin |
| subscription_plan:delete | 删除订阅计划 | admin |
| subscription_plan:activate | 激活计划 | admin |
| subscription_plan:deactivate | 停用计划 | admin |

### 订阅令牌资源（subscription_token）

| 权限 | 描述 | 默认角色 |
|------|------|---------|
| subscription_token:generate | 生成令牌 | user, admin |
| subscription_token:read | 查看自己的令牌 | user, admin |
| subscription_token:revoke | 撤销自己的令牌 | user, admin |
| subscription_token:manage_all | 管理所有令牌 | admin |

## 使用示例

### 1. 创建订阅计划

```go
// 管理员创建订阅计划
createPlanUC := usecases.NewCreateSubscriptionPlanUseCase(
    planRepo,
    logger,
)

cmd := usecases.CreateSubscriptionPlanCommand{
    Name:         "专业版",
    Slug:         "pro",
    Description:  "适合专业用户和小团队",
    Price:        9900, // 99.00 元
    Currency:     "CNY",
    BillingCycle: "monthly",
    TrialDays:    14,
    Features: []string{
        "advanced_analytics",
        "api_access",
        "priority_support",
        "custom_branding",
    },
    Limits: map[string]interface{}{
        "max_projects":       100,
        "max_team_members":   10,
        "storage_gb":         100,
        "api_requests_day":   10000,
    },
    CustomEndpoint: "/api/v1/pro",
    APIRateLimit:   300,
    MaxUsers:       10,
    IsPublic:       true,
}

plan, err := createPlanUC.Execute(ctx, cmd)
```

### 2. 用户购买订阅

```go
// 用户购买订阅
createSubUC := usecases.NewCreateSubscriptionUseCase(
    subscriptionRepo,
    planRepo,
    permissionService,
    emailService,
    logger,
)

cmd := usecases.CreateSubscriptionCommand{
    UserID:    userID,
    PlanID:    planID,
    StartDate: time.Now(),
    AutoRenew: true,
    PaymentInfo: map[string]interface{}{
        "payment_method": "alipay",
        "transaction_id": "2024012012345678",
    },
}

subscription, err := createSubUC.Execute(ctx, cmd)
// 订阅创建成功，自动同步 RBAC 权限
```

### 3. 生成访问令牌

```go
// 用户生成 API 访问令牌
generateTokenUC := usecases.NewGenerateSubscriptionTokenUseCase(
    subscriptionRepo,
    tokenRepo,
    logger,
)

cmd := usecases.GenerateSubscriptionTokenCommand{
    SubscriptionID: subscriptionID,
    UserID:         userID,
    Name:           "Production API Key",
    Scope:          TokenScopeAPI,
    ExpiresAt:      nil, // 永不过期
}

result, err := generateTokenUC.Execute(ctx, cmd)

// 返回明文令牌（仅此一次）
fmt.Println("请妥善保存您的 API 密钥：")
fmt.Println(result.Token)
// token_prod_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 4. 使用令牌访问 API

```go
// 客户端调用 API
client := &http.Client{}
req, _ := http.NewRequest("GET", "https://api.example.com/api/v1/pro/data", nil)
req.Header.Set("Authorization", "Bearer token_prod_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")

resp, err := client.Do(req)
```

**服务端验证：**
```go
// API 路由配置
router := gin.Default()

proGroup := router.Group("/api/v1/pro")
proGroup.Use(SubscriptionTokenMiddleware())
proGroup.Use(SubscriptionRateLimitMiddleware(limiter))
{
    proGroup.GET("/data", dataHandler.GetData)
    proGroup.POST("/data", dataHandler.CreateData)

    // 需要特定功能的端点
    proGroup.GET("/analytics",
        RequireSubscriptionFeature("advanced_analytics"),
        analyticsHandler.GetAdvancedAnalytics,
    )
}
```

### 5. 升级订阅计划

```go
// 用户升级到企业版
changePlanUC := usecases.NewChangePlanUseCase(
    subscriptionRepo,
    planRepo,
    permissionService,
    paymentService,
    emailService,
    logger,
)

cmd := usecases.ChangePlanCommand{
    SubscriptionID: subscriptionID,
    NewPlanID:      enterprisePlanID,
    ChangeType:     "upgrade",
    EffectiveDate:  "immediate",
}

err := changePlanUC.Execute(ctx, cmd)
// 自动计算补差价、同步权限、更新令牌作用域
```

### 6. 管理员管理用户订阅

```go
// 管理员查看所有订阅
listSubUC := usecases.NewListSubscriptionsUseCase(
    subscriptionRepo,
    permissionService,
    logger,
)

// 检查管理员权限
allowed, err := permissionService.CheckPermission(
    ctx,
    adminUserID,
    "subscription",
    "manage_all",
)

if allowed {
    query := usecases.ListSubscriptionsQuery{
        Page:     1,
        PageSize: 50,
        Status:   SubscriptionStatusActive,
        SortBy:   "created_at",
        SortDesc: true,
    }

    subscriptions, total, err := listSubUC.Execute(ctx, query)
}
```

### 7. 定时任务：处理过期订阅

```go
// 定时任务（每天执行）
func ProcessExpiringSubscriptions() {
    ctx := context.Background()

    // 查找即将到期的订阅（7天内）
    subscriptions, err := subscriptionRepo.FindExpiringSubscriptions(ctx, 7)

    for _, sub := range subscriptions {
        if sub.AutoRenew {
            // 自动续订
            renewUseCase.Execute(ctx, RenewSubscriptionCommand{
                SubscriptionID: sub.ID,
                IsAutoRenew:    true,
            })
        } else {
            // 发送到期提醒
            emailService.SendExpirationReminder(sub)
        }
    }

    // 标记已过期的订阅
    expired, err := subscriptionRepo.FindExpiredSubscriptions(ctx)
    for _, sub := range expired {
        sub.MarkAsExpired()
        subscriptionRepo.Update(ctx, sub)

        // 撤销权限和令牌
        syncPermissionsUseCase.Execute(ctx, sub.ID)
        tokenRepo.RevokeAllBySubscriptionID(sub.ID)
    }
}
```

### 8. 使用量检查示例

```go
// 检查项目创建限制
func (h *ProjectHandler) CreateProject(c *gin.Context) {
    subscription := c.MustGet("subscription").(*Subscription)
    plan := c.MustGet("subscription_plan").(*SubscriptionPlan)

    // 获取当前使用量
    usage, err := usageRepo.GetCurrentUsage(subscription.ID)
    if err != nil {
        c.JSON(500, gin.H{"error": "failed to check usage"})
        return
    }

    // 检查项目数限制
    maxProjects, ok := plan.GetLimit("max_projects")
    if ok && usage.ProjectsCount >= maxProjects.(uint) {
        c.JSON(403, gin.H{
            "error": "project limit exceeded",
            "current_plan": plan.Name,
            "current_count": usage.ProjectsCount,
            "max_allowed": maxProjects,
            "upgrade_url": "/subscription/upgrade",
        })
        return
    }

    // 创建项目...
    project := createProject(c)

    // 递增使用量
    usage.ProjectsCount++
    usageRepo.Update(usage)

    c.JSON(201, project)
}
```

## 集成说明

### 1. 与 User 领域集成

**订阅关联用户：**
```go
type Subscription struct {
    UserID uint
    // ...
}

// 获取用户的订阅
func (s *SubscriptionService) GetUserSubscription(ctx context.Context, userID uint) (*Subscription, error) {
    return s.subscriptionRepo.GetActiveByUserID(ctx, userID)
}
```

**用户删除时处理订阅：**
```go
// 监听 UserDeletedEvent
func (h *UserDeletedEventHandler) Handle(event UserDeletedEvent) {
    // 取消用户的所有订阅
    subscriptions, _ := subscriptionRepo.GetByUserID(ctx, event.UserID)
    for _, sub := range subscriptions {
        cancelUseCase.Execute(ctx, CancelSubscriptionCommand{
            SubscriptionID: sub.ID,
            Reason:         "user account deleted",
            Immediate:      true,
        })
    }
}
```

### 2. 与 Permission 领域集成

**特性到权限映射：**
```go
func (s *SubscriptionService) SyncPermissions(ctx context.Context, subscription *Subscription) error {
    plan, _ := s.planRepo.GetByID(ctx, subscription.PlanID)

    // 移除旧权限
    s.permissionService.RemoveSubscriptionPermissions(ctx, subscription.UserID)

    // 如果订阅激活，添加新权限
    if subscription.IsActive() {
        for _, feature := range plan.Features {
            permissions := FeatureToPermissionMap[feature]
            for _, perm := range permissions {
                s.permissionService.GrantPermission(ctx, subscription.UserID, perm)
            }
        }
    }

    return nil
}
```

**权限检查中间件：**
```go
func RequireActiveSubscription() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetUint("user_id")

        subscription, err := subscriptionService.GetUserSubscription(c, userID)
        if err != nil || subscription == nil {
            c.JSON(403, gin.H{"error": "no active subscription"})
            c.Abort()
            return
        }

        if !subscription.IsActive() {
            c.JSON(403, gin.H{"error": "subscription inactive or expired"})
            c.Abort()
            return
        }

        c.Set("subscription", subscription)
        c.Next()
    }
}
```

### 3. 支付系统集成

**支付接口：**
```go
type PaymentService interface {
    CreatePayment(amount uint64, currency string, metadata map[string]interface{}) (*Payment, error)
    ProcessPayment(paymentID string) error
    RefundPayment(paymentID string, amount uint64) error
    GetPayment(paymentID string) (*Payment, error)
}

type Payment struct {
    ID            string
    Amount        uint64
    Currency      string
    Status        string
    TransactionID string
    CreatedAt     time.Time
}
```

**订阅创建时处理支付：**
```go
func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, cmd CreateSubscriptionCommand) (*Subscription, error) {
    plan, _ := uc.planRepo.GetByID(ctx, cmd.PlanID)

    // 创建支付
    if plan.Price > 0 {
        payment, err := uc.paymentService.CreatePayment(
            plan.Price,
            plan.Currency,
            map[string]interface{}{
                "user_id": cmd.UserID,
                "plan_id": cmd.PlanID,
            },
        )
        if err != nil {
            return nil, err
        }

        // 处理支付
        if err := uc.paymentService.ProcessPayment(payment.ID); err != nil {
            return nil, fmt.Errorf("payment failed: %w", err)
        }
    }

    // 创建订阅...
}
```

### 4. 邮件通知集成

**邮件服务接口：**
```go
type SubscriptionEmailService interface {
    SendSubscriptionConfirmation(subscription *Subscription, plan *SubscriptionPlan) error
    SendActivationNotification(subscription *Subscription) error
    SendExpirationReminder(subscription *Subscription, daysRemaining int) error
    SendCancellationConfirmation(subscription *Subscription, reason string) error
    SendRenewalConfirmation(subscription *Subscription) error
    SendPlanChangeNotification(subscription *Subscription, oldPlan, newPlan *SubscriptionPlan) error
}
```

### 5. Webhook 集成

**Webhook 配置：**
```go
type WebhookConfig struct {
    URL     string
    Secret  string
    Events  []string
}

// 订阅事件 Webhook
func (s *SubscriptionService) SendWebhook(event interface{}, config WebhookConfig) error {
    payload, _ := json.Marshal(event)

    // 生成签名
    signature := generateSignature(payload, config.Secret)

    // 发送请求
    req, _ := http.NewRequest("POST", config.URL, bytes.NewBuffer(payload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Webhook-Signature", signature)

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)

    return err
}
```

## 安全特性

### 1. 令牌安全

**令牌生成：**
- 使用加密安全的随机数生成器（crypto/rand）
- 令牌长度至少 32 字节
- 存储 SHA256 哈希，不存储明文

**令牌前缀：**
```
sk_live_xxx   - 生产环境密钥
sk_test_xxx   - 测试环境密钥
pk_live_xxx   - 公开密钥（如有）
```

**令牌验证：**
- 恒定时间比较，防止时序攻击
- 检查令牌是否激活和过期
- 验证订阅状态
- 记录使用日志

### 2. 访问控制

**多层权限验证：**
1. 令牌验证（身份认证）
2. 订阅状态检查（订阅是否激活）
3. 计划特性检查（是否有该功能）
4. RBAC 权限检查（是否有操作权限）
5. 使用量限制检查（是否超限）

**示例：**
```go
router.POST("/projects",
    SubscriptionTokenMiddleware(),           // 1. 令牌验证
    RequireActiveSubscription(),            // 2. 订阅状态
    RequireSubscriptionFeature("projects"), // 3. 特性检查
    permissionMiddleware.RequirePermission("project", "create"), // 4. RBAC
    CheckUsageLimits(),                     // 5. 使用量
    projectHandler.Create,
)
```

### 3. 速率限制

**基于令牌的限流：**
```go
// 使用 Redis 实现分布式限流
type RedisRateLimiter struct {
    client *redis.Client
}

func (l *RedisRateLimiter) Allow(tokenID uint, config RateLimitConfig) (bool, error) {
    key := fmt.Sprintf("ratelimit:token:%d", tokenID)

    // 使用滑动窗口算法
    now := time.Now().Unix()
    windowStart := now - 60 // 1分钟窗口

    pipe := l.client.Pipeline()

    // 移除过期记录
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

    // 统计当前窗口请求数
    pipe.ZCard(ctx, key)

    // 添加当前请求
    pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: now})

    // 设置过期时间
    pipe.Expire(ctx, key, time.Minute)

    cmds, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }

    count := cmds[1].(*redis.IntCmd).Val()

    return count <= int64(config.RequestsPerMinute), nil
}
```

### 4. 审计日志

**记录关键操作：**
```go
type SubscriptionAuditLog struct {
    ID             uint
    SubscriptionID uint
    UserID         uint
    Action         string // create, activate, cancel, renew, change_plan
    IPAddress      string
    UserAgent      string
    Details        string // JSON
    CreatedAt      time.Time
}

// 记录审计日志
func (s *SubscriptionService) logAudit(ctx context.Context, action string, details map[string]interface{}) {
    log := &SubscriptionAuditLog{
        SubscriptionID: ctx.Value("subscription_id").(uint),
        UserID:         ctx.Value("user_id").(uint),
        Action:         action,
        IPAddress:      ctx.Value("ip_address").(string),
        UserAgent:      ctx.Value("user_agent").(string),
        Details:        toJSON(details),
        CreatedAt:      time.Now(),
    }

    s.auditRepo.Create(log)
}
```

### 5. 数据隐私

**敏感信息保护：**
- 令牌哈希存储
- 支付信息加密
- 个人信息脱敏
- GDPR 合规（数据导出、删除权）

## 扩展点

### 1. 多货币支持

```go
type CurrencyConverter interface {
    Convert(amount uint64, from, to string) (uint64, error)
    GetExchangeRate(from, to string) (float64, error)
}

// 计划支持多货币定价
type SubscriptionPlan struct {
    Prices map[string]uint64 // {"CNY": 9900, "USD": 1500}
}
```

### 2. 优惠券系统

```go
type Coupon struct {
    Code            string
    DiscountType    string // percentage, fixed_amount
    DiscountValue   uint64
    ValidFrom       time.Time
    ValidUntil      time.Time
    MaxRedemptions  int
    UsedCount       int
    ApplicablePlans []uint
}

func (uc *CreateSubscriptionUseCase) ApplyCoupon(planPrice uint64, coupon *Coupon) uint64 {
    if coupon.DiscountType == "percentage" {
        return planPrice * (100 - coupon.DiscountValue) / 100
    }
    return planPrice - coupon.DiscountValue
}
```

### 3. 免费试用

```go
type SubscriptionPlan struct {
    TrialDays int // 试用天数
}

func (s *Subscription) StartTrial(plan *SubscriptionPlan) {
    s.Status = SubscriptionStatusTrialing
    s.StartDate = time.Now()
    s.EndDate = time.Now().AddDate(0, 0, plan.TrialDays)
}

// 试用转正式订阅
func (s *Subscription) ConvertTrialToActive() error {
    if s.Status != SubscriptionStatusTrialing {
        return fmt.Errorf("not in trial status")
    }

    s.Status = SubscriptionStatusActive
    s.EndDate = s.calculateEndDate()
    return nil
}
```

### 4. 推荐奖励

```go
type Referral struct {
    ReferrerID uint
    ReferredID uint
    RewardType string // discount, credit, free_month
    RewardValue uint64
    Status string // pending, rewarded
}

func (uc *CreateSubscriptionUseCase) ProcessReferral(referralCode string, newUserID uint) {
    referral, _ := uc.referralRepo.GetByCode(referralCode)

    // 给推荐人奖励
    uc.rewardService.GrantReward(referral.ReferrerID, referral.RewardType, referral.RewardValue)

    // 给被推荐人优惠
    uc.discountService.ApplyDiscount(newUserID, "referral_discount")
}
```

### 5. 使用量计费

```go
type UsageBasedBilling struct {
    SubscriptionID uint
    MetricName     string // api_requests, storage_gb, users
    UnitPrice      uint64
    IncludedUnits  uint64
    UsedUnits      uint64
}

func (uc *CalculateBillingUseCase) CalculateUsageCost(subscription *Subscription) uint64 {
    usages, _ := uc.usageRepo.GetCurrentUsage(subscription.ID)

    totalCost := subscription.Plan.BasePrice

    for metric, used := range usages.Metrics {
        billing := uc.getBillingConfig(subscription.PlanID, metric)

        if used > billing.IncludedUnits {
            overageUnits := used - billing.IncludedUnits
            totalCost += overageUnits * billing.UnitPrice
        }
    }

    return totalCost
}
```

## 最佳实践

### 1. 订阅状态管理

- 使用状态机模式管理状态转换
- 记录所有状态变更历史
- 定时任务自动处理过期订阅
- 提前通知即将到期的订阅

### 2. 权限同步

- 订阅变更时立即同步 RBAC 权限
- 使用事件驱动异步同步
- 定期校验权限一致性
- 记录权限变更日志

### 3. 令牌管理

- 令牌明文仅返回一次
- 提供令牌前缀用于显示
- 支持令牌轮换
- 定期清理过期令牌

### 4. 性能优化

- 订阅信息缓存
- 令牌验证缓存
- 使用量统计异步更新
- 批量处理定时任务

### 5. 用户体验

- 清晰的升级/降级提示
- 平滑的功能降级处理
- 友好的限制提示
- 便捷的自助管理

## 相关文档

- [用户领域文档](USER_DOMAIN.md)
- [权限系统文档](PERMISSION_SYSTEM.md)
- [管理员分配指南](ASSIGN_ADMIN.md)
- [权限快速开始](PERMISSION_QUICKSTART.md)

## 总结

订阅管理领域提供了完整的 SaaS 订阅解决方案，支持：

- ✅ 灵活的订阅计划配置
- ✅ 完整的订阅生命周期管理
- ✅ 安全的访问令牌系统
- ✅ 自定义 API 端点路径
- ✅ 基于计划的 RBAC 权限控制
- ✅ API 速率限制
- ✅ 使用量跟踪
- ✅ 与 User 和 Permission 领域集成
- ✅ 支付和邮件集成
- ✅ 丰富的扩展点

该领域遵循 DDD 设计原则，具有高内聚、低耦合的特点，易于维护和扩展。
