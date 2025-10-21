# 权限系统快速入门

## 概述

系统已成功集成基于RBAC的权限控制，所有用户管理接口都需要相应权限。

## 默认角色

系统初始化时创建了两个默认角色：

### Admin (slug: admin)
- 拥有所有权限
- 可以管理用户、角色和权限

### User (slug: user)
- 拥有基础读取权限
- 可以查看用户信息

## 权限控制的API

### 已添加权限控制的接口

| 接口 | 方法 | 路径 | 所需权限 |
|------|------|------|---------|
| 创建用户 | POST | /users | user:create |
| 列出用户 | GET | /users | user:list |
| 查看用户 | GET | /users/:id | user:read |
| 更新用户 | PUT | /users/:id | user:update |
| 删除用户 | DELETE | /users/:id | user:delete |
| 分配角色 | POST | /users/:id/roles | admin角色 |
| 查看用户角色 | GET | /users/:id/roles | user:read |
| 查看用户权限 | GET | /users/:id/permissions | user:read |

### 新增的权限相关接口

| 接口 | 方法 | 路径 | 说明 | 需要认证 |
|------|------|------|------|---------|
| 获取我的权限 | GET | /auth/permissions | 获取当前用户的所有权限 | ✅ |
| 获取我的角色 | GET | /auth/roles | 获取当前用户的所有角色 | ✅ |
| 检查权限 | GET | /auth/check-permission | 检查是否有特定权限 | ✅ |

## 快速测试

### 1. 注册并登录

```bash
# 注册用户
curl -X POST http://localhost:8081/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "name": "Admin User",
    "password": "password123"
  }'

# 登录获取token
curl -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "password123"
  }'
```

### 2. 分配角色（需要admin权限）

```bash
# 需要先手动在数据库中给用户分配admin角色
# 或者使用已有admin用户的token

curl -X POST http://localhost:8081/users/1/roles \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_ids": [1]
  }'
```

### 3. 查看我的权限

```bash
# 查看当前用户的所有权限
curl -X GET http://localhost:8081/auth/permissions \
  -H "Authorization: Bearer YOUR_TOKEN"

# 查看当前用户的角色
curl -X GET http://localhost:8081/auth/roles \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 4. 检查特定权限

```bash
# 检查是否有user:create权限
curl -X GET "http://localhost:8081/auth/check-permission?resource=user&action=create" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 5. 测试权限控制

```bash
# 尝试创建用户（需要user:create权限）
curl -X POST http://localhost:8081/users \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "name": "New User"
  }'

# 没有权限会返回403 Forbidden
```

## 手动分配admin角色

由于第一个用户需要admin权限来分配角色，可以手动在数据库中分配：

```sql
-- 查看用户ID
SELECT id, email FROM users;

-- 查看角色ID
SELECT id, name, slug FROM roles;

-- 分配admin角色给用户ID为1的用户
INSERT INTO user_roles (user_id, role_id, created_at)
VALUES (1, 1, NOW());

-- 同步到Casbin（需要在应用启动后自动加载）
-- 或重启应用让Casbin重新加载策略
```

## 权限代码示例

### 在代码中检查权限

```go
// 在Handler中使用
allowed, err := permissionService.CheckPermission(ctx, userID, "article", "delete")
if !allowed {
    return c.JSON(403, gin.H{"error": "insufficient permissions"})
}
```

### 在路由中使用中间件

```go
// 要求特定权限
router.DELETE("/articles/:id",
    authMiddleware.RequireAuth(),
    permissionMiddleware.RequirePermission("article", "delete"),
    articleHandler.Delete,
)

// 要求特定角色
router.GET("/admin/dashboard",
    authMiddleware.RequireAuth(),
    permissionMiddleware.RequireRole("admin"),
    adminHandler.Dashboard,
)
```

## 常见问题

### Q: 新注册的用户有权限吗？
A: 新注册用户默认**没有分配任何角色**，需要admin用户手动分配。

### Q: 如何添加新的权限？
A: 在数据库的`permissions`表中添加记录，然后通过`role_permissions`表分配给角色。

### Q: 权限修改后需要重启服务吗？
A: 不需要，Casbin会自动同步数据库的变更。

### Q: 如何实现数据级权限（只能操作自己的数据）？
A: 在Handler中额外检查资源所有权：

```go
func (h *ArticleHandler) Update(c *gin.Context) {
    userID := c.Get("user_id").(uint)
    article := h.getArticle(articleID)

    // 检查是否是所有者
    if article.AuthorID != userID {
        // 检查是否有管理员权限
        allowed, _ := h.permissionService.CheckPermission(
            ctx, userID, "article", "update_any",
        )
        if !allowed {
            return c.JSON(403, gin.H{"error": "forbidden"})
        }
    }
    // 继续更新逻辑
}
```

## 后续扩展

1. 创建角色管理API（CRUD操作）
2. 创建权限管理API
3. 实现权限审计日志
4. 添加前端权限管理界面
5. 支持动态权限策略（基于时间、IP等条件）

## 相关文档

- 详细文档: [PERMISSION_SYSTEM.md](./PERMISSION_SYSTEM.md)
- API文档: http://localhost:8081/swagger/index.html
