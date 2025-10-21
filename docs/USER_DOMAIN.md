# 用户领域功能文档

## 概述

用户领域（User Domain）是系统的核心领域模块，负责管理用户的完整生命周期，包括注册、认证、授权、状态管理等功能。该领域严格遵循领域驱动设计（DDD）原则，具有清晰的聚合根、值对象、领域事件和仓储接口。

## 领域模型

### 聚合根

#### User 聚合
用户聚合是该领域的核心，封装了用户的所有业务逻辑和状态管理。

**核心属性：**
- `id`: 用户唯一标识
- `email`: 电子邮件地址（值对象）
- `name`: 用户姓名（值对象）
- `status`: 用户状态（值对象）
- `version`: 乐观锁版本号
- `createdAt`: 创建时间
- `updatedAt`: 更新时间

**认证相关属性：**
- `passwordHash`: 密码哈希值
- `emailVerified`: 邮箱验证状态
- `emailVerificationToken`: 邮箱验证令牌
- `emailVerificationExpiresAt`: 验证令牌过期时间
- `passwordResetToken`: 密码重置令牌
- `passwordResetExpiresAt`: 重置令牌过期时间
- `lastPasswordChangeAt`: 最后密码修改时间
- `failedLoginAttempts`: 失败登录次数
- `lockedUntil`: 账户锁定截止时间

### 值对象

#### Email
电子邮件地址值对象，提供格式验证和业务判断。

**验证规则：**
- 必须符合标准邮箱格式
- 最大长度限制
- 不可为空

**业务方法：**
- `IsBusinessEmail()`: 判断是否为企业邮箱

#### Name
用户姓名值对象，支持完整名称和显示名称。

**验证规则：**
- 长度限制（2-50字符）
- 不可为空

**业务方法：**
- `DisplayName()`: 获取显示名称
- `Initials()`: 获取姓名首字母缩写

#### Password
密码值对象，提供密码强度验证。

**验证规则：**
- 最小长度8字符
- 必须包含字母和数字
- 支持特殊字符

#### Status
用户状态值对象，管理用户的生命周期状态。

**状态类型：**
- `pending`: 待验证（新注册用户默认状态）
- `active`: 活跃（已验证，可正常使用）
- `inactive`: 未激活（暂时停用）
- `suspended`: 已暂停（违规等原因被暂停）
- `deleted`: 已删除（软删除状态）

**状态转换规则：**
```
pending    -> active, inactive, deleted
active     -> inactive, suspended, deleted
inactive   -> active, deleted
suspended  -> active, inactive, deleted
deleted    -> (终态，无法转换)
```

**业务方法：**
- `CanPerformActions()`: 仅 active 状态可执行操作
- `RequiresVerification()`: pending 状态需要验证
- `CanTransitionTo(target)`: 检查是否可转换到目标状态

#### Token
令牌值对象，用于邮箱验证和密码重置。

**特性：**
- 加密存储
- 自动过期机制
- 单次使用

### 实体

#### Session
会话实体，管理用户的登录会话。

**属性：**
- `ID`: 会话唯一标识
- `UserID`: 关联用户ID
- `DeviceName`: 设备名称
- `DeviceType`: 设备类型
- `IPAddress`: IP地址
- `UserAgent`: 浏览器代理
- `TokenHash`: 访问令牌哈希
- `RefreshTokenHash`: 刷新令牌哈希
- `ExpiresAt`: 过期时间
- `LastActivityAt`: 最后活跃时间

**业务方法：**
- `IsExpired()`: 检查是否过期
- `UpdateActivity()`: 更新活跃时间

#### OAuthAccount
OAuth账户实体，管理第三方登录账户绑定。

**支持的提供商：**
- Google
- GitHub

**属性：**
- `Provider`: 提供商标识
- `ProviderUserID`: 提供商用户ID
- `ProviderEmail`: 提供商邮箱
- `ProviderUsername`: 提供商用户名
- `ProviderAvatarURL`: 头像URL
- `LoginCount`: 登录次数
- `LastLoginAt`: 最后登录时间

## 核心功能

### 1. 用户注册

#### 密码注册
使用邮箱和密码注册新用户。

**流程：**
1. 验证邮箱格式和唯一性
2. 验证密码强度
3. 创建用户聚合（状态为 pending）
4. 对密码进行哈希加密
5. 生成邮箱验证令牌
6. 持久化用户数据
7. 如果是系统首个用户，自动分配管理员角色
8. 发送验证邮件

**用例：** `RegisterWithPasswordUseCase`

**输入：**
```go
RegisterWithPasswordCommand{
    Email:    "user@example.com",
    Name:     "张三",
    Password: "SecurePass123",
}
```

**输出：**
- 返回创建的用户聚合
- 发送验证邮件到用户邮箱

#### OAuth 注册
通过第三方OAuth提供商注册（Google/GitHub）。

**流程：**
1. 重定向到OAuth提供商授权页面
2. 用户授权后回调
3. 获取用户信息
4. 检查邮箱是否已存在
5. 创建用户或关联现有用户
6. 创建OAuth账户绑定
7. 自动激活用户（状态为 active）
8. 创建登录会话

**用例：**
- `InitiateOAuthLoginUseCase`（发起登录）
- `HandleOAuthCallbackUseCase`（处理回调）

### 2. 用户认证

#### 密码登录
使用邮箱和密码登录。

**流程：**
1. 验证用户是否存在
2. 检查账户锁定状态
3. 验证密码
4. 失败则记录失败次数（5次后锁定30分钟）
5. 成功则重置失败次数
6. 检查用户状态是否可登录
7. 创建会话
8. 生成JWT令牌对（访问令牌 + 刷新令牌）
9. 返回令牌

**用例：** `LoginWithPasswordUseCase`

**输入：**
```go
LoginWithPasswordCommand{
    Email:      "user@example.com",
    Password:   "SecurePass123",
    DeviceName: "Chrome Browser",
    DeviceType: "desktop",
    IPAddress:  "192.168.1.1",
    UserAgent:  "Mozilla/5.0...",
}
```

**输出：**
```go
LoginWithPasswordResult{
    User:         *User,
    AccessToken:  "eyJhbGc...",
    RefreshToken: "eyJhbGc...",
    ExpiresIn:    3600,
}
```

**安全特性：**
- 失败登录限制：5次失败后锁定30分钟
- 会话管理：支持多设备登录
- 令牌加密：SHA256哈希存储

#### OAuth 登录
通过第三方提供商登录。

**支持的提供商：**
- Google OAuth 2.0
- GitHub OAuth

**用例：**
- `InitiateOAuthLoginUseCase`
- `HandleOAuthCallbackUseCase`

### 3. 邮箱验证

**流程：**
1. 用户点击邮件中的验证链接
2. 验证令牌的有效性和过期时间
3. 标记邮箱为已验证
4. 清除验证令牌
5. 触发邮箱验证事件

**用例：** `VerifyEmailUseCase`

**令牌特性：**
- 有效期：24小时
- 单次使用
- 加密存储

### 4. 密码管理

#### 请求密码重置
用户忘记密码时请求重置。

**流程：**
1. 验证用户邮箱是否存在
2. 生成密码重置令牌
3. 设置令牌过期时间（30分钟）
4. 发送重置邮件
5. 触发密码重置请求事件

**用例：** `RequestPasswordResetUseCase`

#### 重置密码
使用重置令牌设置新密码。

**流程：**
1. 验证令牌有效性和过期时间
2. 验证新密码强度
3. 更新密码哈希
4. 清除重置令牌
5. 重置失败登录计数
6. 解锁账户
7. 发送密码修改通知邮件

**用例：** `ResetPasswordUseCase`

**安全特性：**
- 令牌有效期：30分钟
- 重置后自动解锁账户
- 发送通知邮件

### 5. 用户状态管理

#### 激活用户
将 pending 或 inactive 用户激活。

**方法：** `User.Activate()`

**状态转换：**
- `pending -> active`
- `inactive -> active`

**触发事件：** `UserStatusChangedEvent`

#### 停用用户
将用户设置为未激活状态。

**方法：** `User.Deactivate(reason)`

**状态转换：**
- `active -> inactive`

**参数：**
- `reason`: 停用原因（必填）

#### 暂停用户
暂停用户账户（通常用于违规处理）。

**方法：** `User.Suspend(reason)`

**状态转换：**
- `active -> suspended`

**参数：**
- `reason`: 暂停原因（必填）

#### 删除用户
软删除用户（数据保留，状态标记为已删除）。

**方法：** `User.Delete()`

**状态转换：**
- `任意状态 -> deleted`

**特性：**
- 软删除，数据不实际删除
- 删除后无法恢复
- 触发用户删除事件

### 6. 用户信息管理

#### 更新邮箱
修改用户邮箱地址。

**方法：** `User.UpdateEmail(newEmail)`

**流程：**
1. 验证新邮箱格式
2. 检查是否与当前邮箱相同
3. 更新邮箱
4. 递增版本号
5. 触发邮箱变更事件

**用例：** `UpdateUserUseCase`

#### 更新姓名
修改用户姓名。

**方法：** `User.UpdateName(newName)`

**流程：**
1. 验证姓名格式
2. 检查是否与当前姓名相同
3. 更新姓名
4. 递增版本号
5. 触发姓名变更事件

### 7. 会话管理

#### 刷新令牌
使用刷新令牌获取新的访问令牌。

**用例：** `RefreshTokenUseCase`

**流程：**
1. 验证刷新令牌有效性
2. 检查会话是否存在和过期
3. 生成新的访问令牌
4. 更新会话活跃时间
5. 返回新令牌

#### 登出
结束用户会话。

**用例：** `LogoutUseCase`

**流程：**
1. 验证会话存在性
2. 删除会话记录
3. 使令牌失效

**支持：**
- 单设备登出
- 全设备登出（删除用户所有会话）

### 8. OAuth 账户管理

#### 绑定OAuth账户
将第三方OAuth账户绑定到现有用户。

**流程：**
1. 验证OAuth提供商返回的用户信息
2. 检查该OAuth账户是否已绑定其他用户
3. 创建OAuth账户绑定记录
4. 记录首次登录时间

#### OAuth登录
使用已绑定的OAuth账户登录。

**流程：**
1. 根据提供商和提供商用户ID查找绑定
2. 加载对应的用户
3. 更新登录计数和最后登录时间
4. 创建会话
5. 生成JWT令牌

## 领域事件

领域事件用于解耦和异步处理，所有事件由聚合根记录，持久化后发布。

### 用户生命周期事件

#### UserCreatedEvent
用户创建事件。

**字段：**
- `UserID`: 用户ID
- `Email`: 邮箱地址
- `Name`: 姓名
- `Status`: 初始状态
- `Timestamp`: 事件时间

**触发时机：** 创建新用户时

#### UserStatusChangedEvent
用户状态变更事件。

**字段：**
- `UserID`: 用户ID
- `OldStatus`: 原状态
- `NewStatus`: 新状态
- `Reason`: 变更原因
- `Timestamp`: 事件时间

**触发时机：**
- 激活用户
- 停用用户
- 暂停用户
- 删除用户

#### UserDeletedEvent
用户删除事件。

**字段：**
- `UserID`: 用户ID
- `PreviousStatus`: 删除前状态
- `Timestamp`: 事件时间

**触发时机：** 删除用户时

### 用户信息变更事件

#### UserEmailChangedEvent
邮箱变更事件。

**字段：**
- `UserID`: 用户ID
- `OldEmail`: 旧邮箱
- `NewEmail`: 新邮箱
- `Timestamp`: 事件时间

**触发时机：** 更新邮箱时

#### UserNameChangedEvent
姓名变更事件。

**字段：**
- `UserID`: 用户ID
- `OldName`: 旧姓名
- `NewName`: 新姓名
- `Timestamp`: 事件时间

**触发时机：** 更新姓名时

### 认证相关事件

#### UserEmailVerifiedEvent
邮箱验证事件。

**字段：**
- `UserID`: 用户ID
- `Email`: 已验证的邮箱
- `Timestamp`: 事件时间

**触发时机：** 成功验证邮箱时

#### UserPasswordChangedEvent
密码变更事件。

**字段：**
- `UserID`: 用户ID
- `Timestamp`: 事件时间

**触发时机：**
- 设置密码
- 重置密码

#### UserPasswordResetRequestedEvent
密码重置请求事件。

**字段：**
- `UserID`: 用户ID
- `Email`: 邮箱地址
- `Timestamp`: 事件时间

**触发时机：** 请求密码重置时

#### UserAccountLockedEvent
账户锁定事件。

**字段：**
- `UserID`: 用户ID
- `FailedAttempts`: 失败次数
- `LockDuration`: 锁定时长
- `Timestamp`: 事件时间

**触发时机：** 登录失败次数达到阈值（5次）

## 仓储接口

### UserRepository
用户聚合仓储接口。

**方法：**
```go
type Repository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id uint) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id uint) error
    List(ctx context.Context, filter ListFilter) ([]*User, int64, error)
    ExistsByEmail(ctx context.Context, email string) (bool, error)
}
```

### SessionRepository
会话仓储接口。

**方法：**
```go
type SessionRepository interface {
    Create(session *Session) error
    GetByID(sessionID string) (*Session, error)
    GetByUserID(userID uint) ([]*Session, error)
    GetByTokenHash(tokenHash string) (*Session, error)
    Update(session *Session) error
    Delete(sessionID string) error
    DeleteByUserID(userID uint) error
    DeleteExpired() error
}
```

### OAuthAccountRepository
OAuth账户仓储接口。

**方法：**
```go
type OAuthAccountRepository interface {
    Create(account *OAuthAccount) error
    GetByID(id uint) (*OAuthAccount, error)
    GetByProviderAndUserID(provider, providerUserID string) (*OAuthAccount, error)
    GetByUserID(userID uint) ([]*OAuthAccount, error)
    Update(account *OAuthAccount) error
    Delete(id uint) error
}
```

## 领域服务

### PasswordHasher
密码哈希服务接口。

**方法：**
```go
type PasswordHasher interface {
    Hash(password string) (string, error)
    Verify(password, hash string) error
}
```

**实现：** 使用 bcrypt 算法

### JWTService
JWT令牌服务接口。

**方法：**
```go
type JWTService interface {
    Generate(userID uint, sessionID string) (*TokenPair, error)
    Refresh(refreshToken string) (string, error)
}
```

**令牌类型：**
- **AccessToken**: 访问令牌，有效期短（建议1小时）
- **RefreshToken**: 刷新令牌，有效期长（建议7天）

### EmailService
邮件发送服务接口。

**方法：**
```go
type EmailService interface {
    SendVerificationEmail(to, token string) error
    SendPasswordResetEmail(to, token string) error
    SendPasswordChangedEmail(to string) error
}
```

## 应用层用例

### 用户注册和认证

| 用例 | 描述 | 命令 |
|------|------|------|
| RegisterWithPasswordUseCase | 密码注册 | RegisterWithPasswordCommand |
| LoginWithPasswordUseCase | 密码登录 | LoginWithPasswordCommand |
| InitiateOAuthLoginUseCase | 发起OAuth登录 | InitiateOAuthLoginCommand |
| HandleOAuthCallbackUseCase | 处理OAuth回调 | HandleOAuthCallbackCommand |
| VerifyEmailUseCase | 验证邮箱 | VerifyEmailCommand |
| LogoutUseCase | 登出 | LogoutCommand |
| RefreshTokenUseCase | 刷新令牌 | RefreshTokenCommand |

### 密码管理

| 用例 | 描述 | 命令 |
|------|------|------|
| RequestPasswordResetUseCase | 请求密码重置 | RequestPasswordResetCommand |
| ResetPasswordUseCase | 重置密码 | ResetPasswordCommand |

### 用户管理

| 用例 | 描述 | 命令 |
|------|------|------|
| CreateUserUseCase | 创建用户 | CreateUserCommand |
| GetUserUseCase | 获取用户 | GetUserQuery |
| UpdateUserUseCase | 更新用户 | UpdateUserCommand |

## 使用示例

### 注册新用户

```go
// 1. 创建用例
registerUC := usecases.NewRegisterWithPasswordUseCase(
    userRepo,
    roleRepo,
    passwordHasher,
    emailService,
    permissionService,
    logger,
)

// 2. 执行注册
cmd := usecases.RegisterWithPasswordCommand{
    Email:    "user@example.com",
    Name:     "张三",
    Password: "SecurePass123",
}

user, err := registerUC.Execute(ctx, cmd)
if err != nil {
    // 处理错误
}

// 3. 用户创建成功，状态为 pending
// 验证邮件已发送到用户邮箱
```

### 用户登录

```go
// 1. 创建用例
loginUC := usecases.NewLoginWithPasswordUseCase(
    userRepo,
    sessionRepo,
    passwordHasher,
    jwtService,
    logger,
)

// 2. 执行登录
cmd := usecases.LoginWithPasswordCommand{
    Email:      "user@example.com",
    Password:   "SecurePass123",
    DeviceName: "Chrome Browser",
    DeviceType: "desktop",
    IPAddress:  "192.168.1.1",
    UserAgent:  "Mozilla/5.0...",
}

result, err := loginUC.Execute(ctx, cmd)
if err != nil {
    // 处理错误（可能是密码错误、账户锁定等）
}

// 3. 登录成功，获取令牌
accessToken := result.AccessToken
refreshToken := result.RefreshToken
```

### 验证邮箱

```go
// 用户点击邮件中的链接，传入验证令牌
verifyUC := usecases.NewVerifyEmailUseCase(userRepo, logger)

err := verifyUC.Execute(ctx, token)
if err != nil {
    // 处理错误（令牌无效或过期）
}

// 邮箱验证成功，用户可以正常使用系统
```

### 重置密码

```go
// 1. 请求重置
requestUC := usecases.NewRequestPasswordResetUseCase(
    userRepo,
    emailService,
    logger,
)

err := requestUC.Execute(ctx, "user@example.com")
// 重置邮件已发送

// 2. 使用令牌重置密码
resetUC := usecases.NewResetPasswordUseCase(
    userRepo,
    passwordHasher,
    emailService,
    logger,
)

cmd := usecases.ResetPasswordCommand{
    Token:       "reset_token_from_email",
    NewPassword: "NewSecurePass456",
}

err = resetUC.Execute(ctx, cmd)
// 密码重置成功，账户自动解锁
```

### 管理用户状态

```go
// 获取用户
user, err := userRepo.GetByID(ctx, userID)

// 激活用户
err = user.Activate()
userRepo.Update(ctx, user)

// 暂停用户
err = user.Suspend("违反使用条款")
userRepo.Update(ctx, user)

// 删除用户（软删除）
err = user.Delete()
userRepo.Update(ctx, user)
```

### OAuth 登录流程

```go
// 1. 发起OAuth登录
initiateUC := usecases.NewInitiateOAuthLoginUseCase(
    googleClient,
    githubClient,
    logger,
)

cmd := usecases.InitiateOAuthLoginCommand{
    Provider: "google",
}

result, err := initiateUC.Execute(cmd)
// 重定向到 result.AuthURL

// 2. 处理OAuth回调
callbackUC := usecases.NewHandleOAuthCallbackUseCase(
    userRepo,
    oauthAccountRepo,
    sessionRepo,
    googleClient,
    githubClient,
    jwtService,
    logger,
)

callbackCmd := usecases.HandleOAuthCallbackCommand{
    Provider:   "google",
    Code:       "auth_code_from_google",
    State:      result.State,
    DeviceName: "Chrome Browser",
    DeviceType: "desktop",
    IPAddress:  "192.168.1.1",
    UserAgent:  "Mozilla/5.0...",
}

loginResult, err := callbackUC.Execute(ctx, callbackCmd)
// 登录成功，获取令牌
```

## 安全特性

### 1. 密码安全
- **加密算法**: bcrypt（自适应哈希）
- **密码强度**: 最小8字符，必须包含字母和数字
- **密码历史**: 记录最后修改时间

### 2. 登录保护
- **失败限制**: 5次失败后锁定
- **锁定时长**: 30分钟
- **自动解锁**: 锁定时间到期自动解锁

### 3. 令牌安全
- **验证令牌**: 24小时有效期
- **重置令牌**: 30分钟有效期
- **访问令牌**: 建议1小时有效期
- **刷新令牌**: 建议7天有效期
- **令牌存储**: SHA256哈希存储

### 4. 会话管理
- **会话隔离**: 每个设备独立会话
- **会话过期**: 7天无活动自动过期
- **活跃追踪**: 记录最后活跃时间

### 5. OAuth安全
- **State参数**: 防止CSRF攻击
- **State过期**: 10分钟有效期
- **一次性使用**: State验证后立即删除

## 集成说明

### 权限系统集成
用户领域与权限（Permission）领域集成：
- **首个用户**: 自动分配 `admin` 角色
- **角色分配**: 通过 `PermissionService.AssignRoleToUser()`
- **权限检查**: 在应用层或接口层进行

### 邮件服务集成
邮件服务由基础设施层实现：
- 验证邮件
- 密码重置邮件
- 密码修改通知邮件

### 日志集成
所有用例集成结构化日志：
- 成功操作记录 Info 级别
- 业务错误记录 Warn 级别
- 系统错误记录 Error 级别

## 最佳实践

### 1. 聚合一致性
- 使用乐观锁（version字段）防止并发冲突
- 所有状态变更必须通过聚合根方法
- 验证逻辑封装在聚合根内部

### 2. 事件驱动
- 重要业务操作记录领域事件
- 事件在事务提交后发布
- 事件用于解耦和异步处理

### 3. 值对象使用
- 邮箱、姓名、密码等使用值对象封装
- 值对象不可变
- 验证逻辑内聚在值对象内

### 4. 仓储模式
- 仓储接口定义在领域层
- 实现在基础设施层
- 通过接口依赖倒置

### 5. 用例编排
- 每个用例对应一个业务场景
- 用例负责编排多个聚合和服务
- 用例不包含领域逻辑，仅负责协调

## 扩展点

### 1. 多因素认证（MFA）
可扩展支持：
- TOTP（基于时间的一次性密码）
- SMS验证码
- 邮箱验证码

### 2. 社交登录扩展
可添加更多OAuth提供商：
- Facebook
- Twitter/X
- Microsoft
- Apple

### 3. 用户属性扩展
可添加更多用户属性：
- 头像
- 电话号码
- 地址
- 偏好设置

### 4. 审计日志
可添加详细审计：
- 登录历史
- 操作日志
- IP白名单

## 相关文档

- [权限系统文档](PERMISSION_SYSTEM.md)
- [管理员分配指南](ASSIGN_ADMIN.md)
- [权限快速开始](PERMISSION_QUICKSTART.md)

## 总结

用户领域是系统的核心领域，提供了完整的用户生命周期管理功能。通过领域驱动设计，实现了高内聚、低耦合的架构，易于维护和扩展。关键特性包括：

- ✅ 多种注册方式（密码、OAuth）
- ✅ 安全的认证机制（密码加密、登录保护）
- ✅ 完整的状态管理（生命周期状态转换）
- ✅ 会话和令牌管理
- ✅ 邮箱验证和密码重置
- ✅ OAuth第三方登录
- ✅ 领域事件支持
- ✅ 与权限系统集成
