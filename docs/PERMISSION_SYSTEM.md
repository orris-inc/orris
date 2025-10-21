# 权限控制系统使用文档

## 概述

本系统实现了基于RBAC（Role-Based Access Control）的权限控制，结合Casbin实现灵活的权限策略管理。

## 架构设计

### 领域层 (Domain Layer)
- **Role 聚合根**: `internal/domain/permission/role.go`
- **Permission 聚合根**: `internal/domain/permission/permission.go`
- **值对象**: Resource, Action

### 基础设施层 (Infrastructure Layer)
- **仓储实现**: Role Repository, Permission Repository
- **Casbin集成**: `internal/infrastructure/permission/enforcer.go`

### 应用层 (Application Layer)
- **权限服务**: `internal/application/permission/service.go`

### 接口层 (Interface Layer)
- **权限中间件**: `internal/interfaces/http/middleware/permission.go`

## 数据库表结构

- `roles`: 角色表
- `permissions`: 权限表
- `role_permissions`: 角色-权限关联表
- `user_roles`: 用户-角色关联表
- `casbin_rule`: Casbin策略表（自动创建）

## 使用示例

### 1. 在路由中使用权限控制

```go
// 要求用户有user:read权限
router.GET("/users/:id",
    authMiddleware.RequireAuth(),
    permissionMiddleware.RequirePermission("user", "read"),
    userHandler.GetUser,
)

// 要求用户有admin角色
router.DELETE("/users/:id",
    authMiddleware.RequireAuth(),
    permissionMiddleware.RequireRole("admin"),
    userHandler.DeleteUser,
)
```

### 2. 分配角色给用户

```go
// 使用Permission Service
roleIDs := []uint{1, 2} // admin和user角色ID
err := permissionService.AssignRoleToUser(ctx, userID, roleIDs)
```

### 3. 检查用户权限

```go
// 检查用户是否有特定权限
allowed, err := permissionService.CheckPermission(ctx, userID, "user", "delete")
if !allowed {
    return fmt.Errorf("permission denied")
}
```

### 4. 获取用户的所有权限

```go
permissions, err := permissionService.GetUserPermissions(ctx, userID)
for _, perm := range permissions {
    fmt.Printf("%s:%s\n", perm.Resource(), perm.Action())
}
```

## 权限粒度支持

### 1. 功能级别（API端点）
使用中间件 `RequirePermission(resource, action)`:
```go
router.POST("/articles",
    permissionMiddleware.RequirePermission("article", "create"),
    articleHandler.Create,
)
```

### 2. 操作级别
权限定义为 `resource:action`:
- `user:create` - 创建用户
- `user:read` - 读取用户
- `user:update` - 更新用户
- `user:delete` - 删除用户

### 3. 数据级别（资源所有权）
在Handler中检查:
```go
func (h *ArticleHandler) Update(c *gin.Context) {
    userID := c.Get("user_id").(uint)
    article := h.getArticle(articleID)

    // 检查是否是文章所有者
    if article.AuthorID != userID {
        // 检查是否有admin权限
        allowed, _ := h.permissionService.CheckPermission(ctx, userID, "article", "update_any")
        if !allowed {
            c.JSON(403, gin.H{"error": "forbidden"})
            return
        }
    }

    // 继续更新逻辑
}
```

### 4. 字段级别
在应用服务层过滤敏感字段:
```go
func (s *UserService) GetUser(ctx context.Context, userID, targetUserID uint) (*UserDTO, error) {
    user := s.userRepo.GetByID(ctx, targetUserID)

    // 检查权限决定返回哪些字段
    canViewSensitive, _ := s.permissionService.CheckPermission(ctx, userID, "user", "view_sensitive")

    dto := &UserDTO{
        ID:    user.ID,
        Name:  user.Name,
        Email: user.Email,
    }

    if !canViewSensitive {
        dto.Email = "" // 隐藏敏感字段
    }

    return dto, nil
}
```

## 默认角色和权限

系统初始化时创建了以下默认角色：

### Admin 角色 (slug: admin)
拥有所有权限

### User 角色 (slug: user)
拥有基础权限：
- `user:read` - 查看用户信息

## 扩展权限

### 添加新权限

1. 在数据库中添加权限记录:
```sql
INSERT INTO permissions (resource, action, description) VALUES
('article', 'create', 'Create articles'),
('article', 'read', 'View articles'),
('article', 'update', 'Update articles');
```

2. 将权限分配给角色:
```sql
INSERT INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE resource = 'article';
```

3. Casbin会自动同步策略到`casbin_rule`表

## 配置文件

Casbin RBAC模型配置: `configs/rbac_model.conf`

```
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

## 注意事项

1. **性能考虑**: Casbin的策略会缓存在内存中，权限检查非常快速
2. **策略更新**: 修改权限后会自动保存到数据库，无需手动刷新
3. **角色继承**: Casbin支持角色继承，可以实现更复杂的权限模型
4. **动态配置**: 所有权限都存储在数据库中，可以在运行时动态修改

## 未来扩展

1. **权限管理API**: 创建管理界面用于动态管理角色和权限
2. **权限审计**: 记录权限检查日志用于审计
3. **条件权限**: 基于时间、IP等条件的权限控制
4. **数据权限**: 更细粒度的数据行级权限控制
