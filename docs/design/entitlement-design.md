# Entitlement 核心权限层设计方案

## 1. 整体架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│                        ┌──────────────────┐                             │
│                        │   Entitlement    │                             │
│                        │   (权限中心)      │                             │
│                        └────────┬─────────┘                             │
│                                 │                                       │
│         ┌───────────────────────┼───────────────────────┐               │
│         │                       │                       │               │
│    ┌────┴────┐            ┌─────┴─────┐           ┌─────┴─────┐         │
│    │ Subject │            │ Resource  │           │  Source   │         │
│    │  主体   │            │   资源    │           │   来源    │         │
│    └────┬────┘            └─────┬─────┘           └─────┴─────┘         │
│         │                       │                       │               │
│    ┌────┴────┐            ┌─────┴─────┐           ┌─────┴─────┐         │
│    │  user   │            │   node    │           │   plan    │         │
│    │  group  │            │   agent   │           │  direct   │         │
│    └─────────┘            │  feature  │           │ promotion │         │
│                           └───────────┘           └───────────┘         │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## 2. 领域模型设计

### 2.1 Entitlement 聚合根

```go
// internal/domain/entitlement/entitlement.go
type Entitlement struct {
    id            uint

    // 权限主体（谁拥有这个权限）
    subjectType   SubjectType     // user | user_group
    subjectID     uint

    // 资源（权限作用于什么）
    resourceType  ResourceType    // node | forward_agent | feature
    resourceID    uint            // 具体资源ID，feature类型可为0

    // 权限来源（权限从哪来）
    sourceType    SourceType      // subscription | direct | promotion
    sourceID      uint            // 来源ID（subscription_id / promotion_id）

    // 权限属性
    status        EntitlementStatus  // active | expired | revoked
    expiresAt     *time.Time         // 过期时间（nil = 永不过期）
    metadata      map[string]any     // 扩展元数据

    createdAt     time.Time
    updatedAt     time.Time
    version       int
}
```

### 2.2 值对象

```go
// SubjectType - 权限主体类型
type SubjectType string
const (
    SubjectTypeUser      SubjectType = "user"
    SubjectTypeUserGroup SubjectType = "user_group"
)

// ResourceType - 资源类型
type ResourceType string
const (
    ResourceTypeNode         ResourceType = "node"
    ResourceTypeForwardAgent ResourceType = "forward_agent"
    ResourceTypeFeature      ResourceType = "feature"      // 功能特性
)

// SourceType - 权限来源类型
type SourceType string
const (
    SourceTypeSubscription SourceType = "subscription"  // 来自订阅
    SourceTypeDirect       SourceType = "direct"        // 直接授权（管理员）
    SourceTypePromotion    SourceType = "promotion"     // 促销活动
    SourceTypeTrial        SourceType = "trial"         // 试用
)

// EntitlementStatus - 权限状态
type EntitlementStatus string
const (
    EntitlementStatusActive  EntitlementStatus = "active"
    EntitlementStatusExpired EntitlementStatus = "expired"
    EntitlementStatusRevoked EntitlementStatus = "revoked"
)
```

### 2.3 UserGroup 聚合根

```go
// internal/domain/user/usergroup.go
type UserGroup struct {
    id          uint
    name        string
    slug        string              // URL友好标识
    ownerID     uint                // 群组所有者（User ID）
    status      UserGroupStatus     // active | suspended | deleted
    maxMembers  uint                // 最大成员数
    metadata    map[string]any
    createdAt   time.Time
    updatedAt   time.Time
    version     int
}

// UserGroupMember - 成员关系（独立实体）
type UserGroupMember struct {
    id        uint
    groupID   uint
    userID    uint
    role      MemberRole          // owner | admin | member
    joinedAt  time.Time
    invitedBy *uint               // 邀请人
}

type MemberRole string
const (
    MemberRoleOwner  MemberRole = "owner"
    MemberRoleAdmin  MemberRole = "admin"
    MemberRoleMember MemberRole = "member"
)
```

### 2.4 Subscription 调整

```go
// internal/domain/subscription/subscription.go
type Subscription struct {
    id                 uint
    uuid               string

    // 订阅主体（与 Entitlement 保持一致）
    subjectType        SubjectType    // user | user_group
    subjectID          uint

    planID             uint
    status             SubscriptionStatus
    // ... 其余字段保持不变
}
```

### 2.5 Plan 简化

```go
// Plan 只定义"模板"，不再直接关联资源
type Plan struct {
    id           uint
    name         string
    slug         string
    description  string
    planType     PlanType           // node | forward
    status       PlanStatus

    // 功能限制模板
    features     *PlanFeatures

    // 资源模板（订阅时根据此模板生成 Entitlement）
    resourceTemplate  *ResourceTemplate

    trialDays    int
    isPublic     bool
    sortOrder    int
    metadata     map[string]any
    version      int
}

// ResourceTemplate - 资源模板
type ResourceTemplate struct {
    // 订阅此计划时，自动授权的资源
    NodeIDs         []uint   `json:"node_ids,omitempty"`
    ForwardAgentIDs []uint   `json:"forward_agent_ids,omitempty"`
    Features        []string `json:"features,omitempty"`  // 功能列表
}
```

## 3. 完整关系图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  ┌─────────┐         ┌─────────────┐         ┌──────────────────────────┐   │
│  │  User   │◄───N:N──│UserGroupMember│───N:1─►│      UserGroup          │   │
│  └────┬────┘         └─────────────┘         └───────────┬──────────────┘   │
│       │                                                   │                 │
│       │ subjectType=user                    subjectType=user_group          │
│       │                                                   │                 │
│       └───────────────────┬───────────────────────────────┘                 │
│                           │                                                 │
│                           ▼                                                 │
│                    ┌──────────────┐                                         │
│                    │ Subscription │──────N:1────►┌──────┐                   │
│                    │              │              │ Plan │                   │
│                    └──────┬───────┘              └──────┘                   │
│                           │                                                 │
│                           │ sourceType=subscription                         │
│                           ▼                                                 │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                        Entitlement                                    │   │
│  │  ┌─────────────────────────────────────────────────────────────────┐ │   │
│  │  │ subject (user/group) + resource (node/agent/feature) + source  │ │   │
│  │  └─────────────────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                           │                                                 │
│           ┌───────────────┼───────────────┐                                 │
│           ▼               ▼               ▼                                 │
│       ┌──────┐     ┌────────────┐    ┌─────────┐                            │
│       │ Node │     │ForwardAgent│    │ Feature │                            │
│       └──────┘     └────────────┘    └─────────┘                            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 4. 权限查询服务

```go
// EntitlementService - 权限查询服务
type EntitlementService interface {
    // 检查用户是否有资源权限
    HasAccess(ctx context.Context, userID uint, resourceType ResourceType, resourceID uint) (bool, error)

    // 获取用户所有有效权限
    GetUserEntitlements(ctx context.Context, userID uint) ([]*Entitlement, error)

    // 获取用户可访问的资源ID列表
    GetAccessibleResources(ctx context.Context, userID uint, resourceType ResourceType) ([]uint, error)
}

// 查询逻辑：
// 1. 查询 user 直接拥有的 entitlements
// 2. 查询 user 所属的所有 user_groups
// 3. 查询这些 groups 拥有的 entitlements
// 4. 合并去重，过滤有效状态
```

## 5. 订阅生命周期与 Entitlement

```
订阅创建时：
┌─────────────┐     ┌────────────────────────────────────────┐
│ Subscription│────►│ 根据 Plan.resourceTemplate 生成       │
│   Created   │     │ 多条 Entitlement (sourceType=subscription)│
└─────────────┘     └────────────────────────────────────────┘

订阅过期/取消时：
┌─────────────┐     ┌────────────────────────────────────────┐
│ Subscription│────►│ 将相关 Entitlement 标记为 expired/revoked │
│   Expired   │     │ 或设置 expiresAt                        │
└─────────────┘     └────────────────────────────────────────┘

订阅续期时：
┌─────────────┐     ┌────────────────────────────────────────┐
│ Subscription│────►│ 更新相关 Entitlement 的 expiresAt       │
│   Renewed   │     └────────────────────────────────────────┘
└─────────────┘
```

## 6. 数据库表设计

```sql
-- entitlements 表
CREATE TABLE entitlements (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,

    -- 主体
    subject_type    VARCHAR(20) NOT NULL,  -- user | user_group
    subject_id      BIGINT UNSIGNED NOT NULL,

    -- 资源
    resource_type   VARCHAR(30) NOT NULL,  -- node | forward_agent | feature
    resource_id     BIGINT UNSIGNED NOT NULL,

    -- 来源
    source_type     VARCHAR(20) NOT NULL,  -- subscription | direct | promotion
    source_id       BIGINT UNSIGNED NOT NULL,

    -- 属性
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at      TIMESTAMP NULL,
    metadata        JSON,

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    version         INT NOT NULL DEFAULT 1,

    -- 索引
    UNIQUE INDEX idx_unique_entitlement (subject_type, subject_id, resource_type, resource_id, source_type, source_id),
    INDEX idx_subject (subject_type, subject_id),
    INDEX idx_resource (resource_type, resource_id),
    INDEX idx_source (source_type, source_id),
    INDEX idx_status_expires (status, expires_at)
);

-- user_groups 表
CREATE TABLE user_groups (
    id          BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    owner_id    BIGINT UNSIGNED NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    max_members INT UNSIGNED NOT NULL DEFAULT 10,
    metadata    JSON,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    version     INT NOT NULL DEFAULT 1,

    INDEX idx_owner (owner_id),
    INDEX idx_status (status)
);

-- user_group_members 表
CREATE TABLE user_group_members (
    id          BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    group_id    BIGINT UNSIGNED NOT NULL,
    user_id     BIGINT UNSIGNED NOT NULL,
    role        VARCHAR(20) NOT NULL DEFAULT 'member',
    joined_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    invited_by  BIGINT UNSIGNED,

    UNIQUE INDEX idx_group_user (group_id, user_id),
    INDEX idx_user (user_id)
);
```

## 7. 迁移策略

| 步骤 | 内容 | 影响 |
|------|------|------|
| 1 | 创建新的 `entitlements` 表（新结构） | 无 |
| 2 | 创建 `user_groups` 和 `user_group_members` 表 | 无 |
| 3 | 迁移现有 entitlements 数据到新表 | 数据迁移脚本 |
| 4 | 修改 Subscription 支持 subjectType/subjectID | 代码改动 |
| 5 | 实现 EntitlementService | 新增服务 |
| 6 | 切换业务逻辑使用新权限模型 | 逐步替换 |

## 8. 设计优势

1. **解耦权限来源与权限本身**：Entitlement 独立存在，不依赖 Plan 直接关联
2. **支持多种授权方式**：订阅、直接授权、促销、试用等
3. **支持 User 和 UserGroup**：灵活的权限主体
4. **支持多种资源类型**：Node、ForwardAgent、Feature 等
5. **支持过期时间**：精细的权限有效期控制
6. **便于权限查询**：统一的权限查询入口
