# Auth Errors 使用指南

## 概述

`auth_errors.go` 提供了一套专门用于认证和授权场景的结构化错误类型，解决了以下问题：

1. ✅ 统一的认证错误格式
2. ✅ 智能日志记录策略（避免日志污染）
3. ✅ 安全事件追踪
4. ✅ 符合安全最佳实践（不泄露敏感信息）

## 核心设计理念

### 1. 错误分类

| 错误类型 | 使用场景 | HTTP状态码 | 是否记录日志 | 是否安全事件 |
|---------|---------|-----------|-------------|-------------|
| `InvalidCredentials` | 用户名/密码错误 | 401 | ❌ | ✅ |
| `AccountLocked` | 账户被锁定 | 403 | ✅ | ✅ |
| `AccountInactive` | 账户未激活 | 403 | ❌ | ❌ |
| `TokenExpired` | Token过期（正常） | 401 | ❌ | ❌ |
| `TokenInvalid` | Token无效（异常） | 401 | ✅ | ✅ |
| `SessionExpired` | Session过期（正常） | 401 | ❌ | ❌ |
| `PasswordNotSet` | OAuth账户尝试密码登录 | 400 | ❌ | ❌ |
| `OAuthError` | OAuth流程失败 | 502 | ✅ | ❌ |

### 2. 安全最佳实践

**原则1: 不泄露用户存在性**
```go
// ❌ 不好 - 泄露了邮箱是否存在
if user == nil {
    return fmt.Errorf("user not found")
}

// ✅ 好 - 统一的错误消息
if user == nil || !passwordMatches {
    return errors.NewInvalidCredentialsError()
}
```

**原则2: 区分预期错误和异常**
```go
// 预期的错误（用户输错密码）- 不应该记录Error级别日志
return errors.NewInvalidCredentialsError() // ShouldLog = false

// 异常情况（Token被篡改）- 应该记录并告警
return errors.NewTokenInvalidError("access token") // ShouldLog = true
```

## 使用示例

### 示例1: LoginWithPassword 用例重构

**重构前:**
```go
func (uc *LoginWithPasswordUseCase) Execute(ctx context.Context, cmd LoginWithPasswordCommand) (*LoginWithPasswordResult, error) {
    existingUser, err := uc.userRepo.GetByEmail(ctx, cmd.Email)
    if err != nil {
        uc.logger.Errorw("failed to get user by email", "error", err)
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    if existingUser == nil {
        return nil, fmt.Errorf("invalid email or password") // 🔴 不统一
    }

    if existingUser.IsLocked() {
        return nil, fmt.Errorf("account is temporarily locked...") // 🔴 格式不统一
    }

    if !existingUser.HasPassword() {
        return nil, fmt.Errorf("password login not available...") // 🔴 日志级别不明确
    }

    if err := existingUser.VerifyPassword(cmd.Password, uc.passwordHasher); err != nil {
        return nil, fmt.Errorf("invalid email or password") // 🔴 没有记录安全事件
    }
    
    // ... rest of code
}
```

**重构后:**
```go
func (uc *LoginWithPasswordUseCase) Execute(ctx context.Context, cmd LoginWithPasswordCommand) (*LoginWithPasswordResult, error) {
    existingUser, err := uc.userRepo.GetByEmail(ctx, cmd.Email)
    if err != nil {
        uc.logger.Errorw("failed to get user by email", "error", err)
        return nil, errors.NewInternalError("database error", err.Error())
    }
    
    // Use unified error for non-existent user
    if existingUser == nil {
        authErr := errors.NewInvalidCredentialsError()
        // Track security event without logging at Error level
        if authErr.SecurityEvent {
            uc.logger.Infow("login attempt with unknown email", "email", cmd.Email, "ip", cmd.IPAddress)
        }
        return nil, authErr
    }

    // Account locked check
    if existingUser.IsLocked() {
        authErr := errors.NewAccountLockedError()
        if authErr.ShouldLog {
            uc.logger.Warnw("login attempt on locked account", "user_id", existingUser.ID(), "ip", cmd.IPAddress)
        }
        return nil, authErr
    }

    // Password not set check
    if !existingUser.HasPassword() {
        return nil, errors.NewPasswordNotSetError() // No logging needed
    }

    // Password verification
    if err := existingUser.VerifyPassword(cmd.Password, uc.passwordHasher); err != nil {
        // Update failed attempts
        if updateErr := uc.userRepo.Update(ctx, existingUser); updateErr != nil {
            uc.logger.Errorw("failed to update user after failed login", "error", updateErr)
        }
        
        authErr := errors.NewInvalidCredentialsError()
        if authErr.SecurityEvent {
            uc.logger.Infow("failed password verification", "user_id", existingUser.ID(), "ip", cmd.IPAddress)
        }
        return nil, authErr
    }

    // Account inactive check
    if !existingUser.CanPerformActions() {
        return nil, errors.NewAccountInactiveError()
    }
    
    // ... rest of code
}
```

### 示例2: RefreshToken 用例重构

**重构前:**
```go
func (uc *RefreshTokenUseCase) Execute(cmd RefreshTokenCommand) (*RefreshTokenResult, error) {
    refreshTokenHash := uc.authHelper.HashToken(cmd.RefreshToken)
    
    session, err := uc.sessionRepo.GetByTokenHash(refreshTokenHash)
    if err != nil {
        uc.logger.Errorw("failed to get session", "error", err)
        return nil, fmt.Errorf("invalid or expired refresh token") // 🔴 混淆了两种情况
    }

    if session.IsExpired() {
        return nil, fmt.Errorf("session has expired") // 🔴 不统一
    }
    
    // ... rest of code
}
```

**重构后:**
```go
func (uc *RefreshTokenUseCase) Execute(cmd RefreshTokenCommand) (*RefreshTokenResult, error) {
    refreshTokenHash := uc.authHelper.HashToken(cmd.RefreshToken)
    
    session, err := uc.sessionRepo.GetByTokenHash(refreshTokenHash)
    if err != nil {
        // Database error vs invalid token
        if errors.Is(err, gorm.ErrRecordNotFound) {
            // Invalid token - potential security issue
            authErr := errors.NewTokenInvalidError("refresh token")
            if authErr.ShouldLog {
                uc.logger.Warnw("refresh token not found in database", "error", err)
            }
            return nil, authErr
        }
        // Database error
        uc.logger.Errorw("failed to get session", "error", err)
        return nil, errors.NewInternalError("database error", err.Error())
    }

    if session.IsExpired() {
        // Normal expiration - no need to log
        return nil, errors.NewSessionExpiredError()
    }
    
    // ... rest of code
}
```

### 示例3: OAuth Callback 用例重构

**重构前:**
```go
func (uc *HandleOAuthCallbackUseCase) Execute(ctx context.Context, cmd HandleOAuthCallbackCommand) (*HandleOAuthCallbackResult, error) {
    accessToken, err := client.ExchangeCode(ctx, cmd.Code)
    if err != nil {
        uc.logger.Errorw("failed to exchange code", "error", err, "provider", cmd.Provider)
        return nil, fmt.Errorf("failed to exchange authorization code: %w", err) // 🔴 不统一
    }

    userInfo, err := client.GetUserInfo(ctx, accessToken)
    if err != nil {
        uc.logger.Errorw("failed to get user info", "error", err, "provider", cmd.Provider)
        return nil, fmt.Errorf("failed to get user info: %w", err) // 🔴 不统一
    }
    
    // ... rest of code
}
```

**重构后:**
```go
func (uc *HandleOAuthCallbackUseCase) Execute(ctx context.Context, cmd HandleOAuthCallbackCommand) (*HandleOAuthCallbackResult, error) {
    accessToken, err := client.ExchangeCode(ctx, cmd.Code)
    if err != nil {
        authErr := errors.NewOAuthError(cmd.Provider, "code exchange", err.Error())
        if authErr.ShouldLog {
            uc.logger.Errorw("OAuth code exchange failed", "error", err, "provider", cmd.Provider)
        }
        return nil, authErr
    }

    userInfo, err := client.GetUserInfo(ctx, accessToken)
    if err != nil {
        authErr := errors.NewOAuthError(cmd.Provider, "user info retrieval", err.Error())
        if authErr.ShouldLog {
            uc.logger.Errorw("OAuth user info retrieval failed", "error", err, "provider", cmd.Provider)
        }
        return nil, authErr
    }
    
    // ... rest of code
}
```

## 日志记录规范

### 日志级别使用指南

#### Error 级别
**何时使用:**
- 数据库操作失败
- 关键业务逻辑错误
- 系统内部错误
- 第三方服务调用失败

**示例:**
```go
// Database errors
uc.logger.Errorw("failed to create user in database", "error", err, "email", user.Email())

// Critical business logic errors
uc.logger.Errorw("failed to generate JWT tokens", "error", err, "user_id", userID)

// Internal errors
uc.logger.Errorw("password hasher failed", "error", err)
```

#### Warn 级别
**何时使用:**
- 非关键操作失败（可恢复）
- 账户安全事件（锁定、可疑登录）
- 配置问题
- 降级操作

**示例:**
```go
// Non-critical failures
uc.logger.Warnw("failed to send verification email", "error", err, "email", email)

// Security events
uc.logger.Warnw("login attempt on locked account", "user_id", userID, "ip", ipAddress)

// Token tampering
uc.logger.Warnw("refresh token not found in database", "error", err)
```

#### Info 级别
**何时使用:**
- 成功的业务操作
- 重要的业务流程节点
- 安全事件追踪（不是错误）
- 审计日志

**示例:**
```go
// Successful operations
uc.logger.Infow("user logged in successfully", "user_id", userID, "session_id", sessionID)

// Business milestones
uc.logger.Infow("first user detected, admin role granted", "user_id", userID)

// Security tracking (not errors)
uc.logger.Infow("failed password verification", "user_id", userID, "ip", ipAddress)
```

#### Debug 级别
**何时使用:**
- 开发调试信息
- 详细的流程追踪
- 性能监控数据
- 临时诊断信息

**示例:**
```go
// Development debugging
uc.logger.Debugw("token hash generated", "user_id", userID, "hash_length", len(hash))

// Flow tracking
uc.logger.Debugw("entering password verification", "user_id", userID)

// Performance monitoring
uc.logger.Debugw("database query completed", "duration_ms", duration.Milliseconds())
```

### 日志消息格式规范

**规则:**
1. 使用英文
2. 使用小写字母开头（除非是专有名词）
3. 使用过去时态描述已发生的事件
4. 包含关键上下文（user_id, session_id, error等）

**好的示例:**
```go
✅ logger.Errorw("failed to create user in database", "error", err, "email", email)
✅ logger.Infow("user registered successfully", "user_id", userID, "email", email)
✅ logger.Warnw("OAuth account update failed", "error", err, "provider", provider)
```

**不好的示例:**
```go
❌ logger.Errorw("Error", "error", err) // Too vague
❌ logger.Errorw("Failed to create user", "error", err) // Missing context
❌ logger.Errorw("创建用户失败", "error", err) // Not in English
❌ logger.Errorw("Creating user...", "email", email) // Using present continuous
```

### 结构化日志字段命名规范

**通用字段:**
- `error`: 错误对象
- `user_id`: 用户ID
- `session_id`: 会话ID
- `email`: 邮箱地址
- `ip`: IP地址
- `provider`: OAuth提供商

**特定场景字段:**
- `token_type`: Token类型（"access", "refresh", "reset"）
- `duration_ms`: 持续时间（毫秒）
- `attempt_count`: 尝试次数
- `is_new_user`: 是否新用户

## 错误处理助手（可选增强）

如果需要进一步简化错误处理，可以创建一个 ErrorHandler helper：

```go
// internal/application/user/helpers/error_handler.go
package helpers

import (
    "orris/internal/shared/errors"
    "orris/internal/shared/logger"
)

type ErrorHandler struct {
    logger logger.Interface
}

func NewErrorHandler(logger logger.Interface) *ErrorHandler {
    return &ErrorHandler{logger: logger}
}

// HandleAuthError handles authentication errors with smart logging
func (h *ErrorHandler) HandleAuthError(err error, context ...interface{}) error {
    if err == nil {
        return nil
    }

    authErr := errors.GetAuthError(err)
    if authErr == nil {
        // Not an auth error, log as regular error
        h.logger.Errorw("unexpected error", append([]interface{}{"error", err}, context...)...)
        return err
    }

    // Handle based on error properties
    if authErr.ShouldLog {
        switch authErr.AppError.Code {
        case 500, 502, 503:
            h.logger.Errorw(authErr.AppError.Message, append([]interface{}{"error", err}, context...)...)
        default:
            h.logger.Warnw(authErr.AppError.Message, append([]interface{}{"error", err}, context...)...)
        }
    }

    if authErr.SecurityEvent {
        h.logger.Infow("security event detected", append([]interface{}{"error_type", authErr.AppError.Type}, context...)...)
    }

    return authErr
}
```

**使用示例:**
```go
if err := existingUser.VerifyPassword(cmd.Password, uc.passwordHasher); err != nil {
    authErr := errors.NewInvalidCredentialsError()
    return nil, uc.errorHandler.HandleAuthError(authErr, "user_id", existingUser.ID(), "ip", cmd.IPAddress)
}
```

## 迁移检查清单

重构现有代码时，使用以下检查清单：

- [ ] 将 `fmt.Errorf("invalid email or password")` 替换为 `errors.NewInvalidCredentialsError()`
- [ ] 将 `fmt.Errorf("account is locked...")` 替换为 `errors.NewAccountLockedError()`
- [ ] 将 `fmt.Errorf("account is not active")` 替换为 `errors.NewAccountInactiveError()`
- [ ] 将 `fmt.Errorf("invalid or expired refresh token")` 区分为 `NewTokenExpiredError()` 或 `NewTokenInvalidError()`
- [ ] 将 `fmt.Errorf("session has expired")` 替换为 `errors.NewSessionExpiredError()`
- [ ] 将 `fmt.Errorf("password login not available...")` 替换为 `errors.NewPasswordNotSetError()`
- [ ] OAuth错误使用 `errors.NewOAuthError(provider, stage, details)`
- [ ] 使用 `authErr.ShouldLog` 决定是否记录日志
- [ ] 使用 `authErr.SecurityEvent` 追踪安全事件
- [ ] 确保日志消息使用英文
- [ ] 确保日志级别正确（Error/Warn/Info/Debug）
- [ ] 包含足够的上下文信息（user_id, ip, provider等）

## 测试建议

```go
func TestLoginWithPassword_InvalidCredentials(t *testing.T) {
    // Test that invalid credentials return proper AuthError
    _, err := useCase.Execute(ctx, cmd)
    
    assert.Error(t, err)
    assert.True(t, errors.IsAuthError(err))
    
    authErr := errors.GetAuthError(err)
    assert.Equal(t, errors.ErrorTypeInvalidCredentials, authErr.Type)
    assert.False(t, authErr.ShouldLog)
    assert.True(t, authErr.SecurityEvent)
}

func TestRefreshToken_Expired(t *testing.T) {
    // Test that expired sessions return proper error
    _, err := useCase.Execute(cmd)
    
    authErr := errors.GetAuthError(err)
    assert.Equal(t, errors.ErrorTypeSessionExpired, authErr.Type)
    assert.False(t, authErr.ShouldLog)
    assert.False(t, authErr.SecurityEvent)
}
```

## 总结

通过使用 `auth_errors.go`，您可以获得：

1. **一致性**: 所有认证错误使用统一格式
2. **安全性**: 不泄露敏感信息，符合安全最佳实践
3. **可维护性**: 中心化的错误定义，易于修改和扩展
4. **可观测性**: 智能日志记录，区分预期错误和异常
5. **可追踪性**: 内置安全事件标记，便于审计和监控

开始重构时，建议先从核心用例（LoginWithPassword, RefreshToken）开始，然后逐步推广到其他用例。
