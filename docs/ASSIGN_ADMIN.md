# 手动分配Admin角色

## 方法1：直接SQL操作（推荐）

### 步骤1：注册第一个用户

```bash
curl -X POST http://localhost:8081/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "name": "Admin User",
    "password": "Admin@123456"
  }'
```

### 步骤2：查看用户ID

```bash
# 登录获取用户信息
curl -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin@123456"
  }'

# 返回结果中会包含user_id
```

或者直接在数据库查询：

```sql
-- 连接数据库
mysql -u root -p123456 orris-dev

-- 查看所有用户
SELECT id, email, name, status FROM users;
```

### 步骤3：分配admin角色

```sql
-- 查看角色列表
SELECT id, name, slug FROM roles;

-- 输出示例:
-- +----+---------------+-------+
-- | id | name          | slug  |
-- +----+---------------+-------+
-- |  1 | Administrator | admin |
-- |  2 | User          | user  |
-- +----+---------------+-------+

-- 给用户ID为1的用户分配admin角色（角色ID为1）
INSERT INTO user_roles (user_id, role_id, created_at)
VALUES (1, 1, NOW());

-- 验证分配成功
SELECT ur.*, u.email, r.name as role_name
FROM user_roles ur
JOIN users u ON ur.user_id = u.id
JOIN roles r ON ur.role_id = r.id;
```

### 步骤4：同步Casbin策略

```sql
-- 添加Casbin策略，使用户与admin角色关联
INSERT INTO casbin_rule (ptype, v0, v1, v2)
VALUES ('g', '1', 'admin', '');

-- 验证Casbin规则
SELECT * FROM casbin_rule WHERE v0 = '1';
```

### 步骤5：重启应用并测试

```bash
# 重启服务
# Ctrl+C 停止当前服务，然后重新启动
./bin/orris server start

# 测试admin权限
curl -X GET http://localhost:8081/auth/roles \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# 应该返回admin角色
```

## 方法2：使用SQL脚本一次完成

创建文件 `scripts/assign_first_admin.sql`:

```sql
-- 查找第一个注册的用户
SET @first_user_id = (SELECT id FROM users ORDER BY created_at LIMIT 1);
SET @admin_role_id = (SELECT id FROM roles WHERE slug = 'admin');

-- 分配admin角色
INSERT INTO user_roles (user_id, role_id, created_at)
SELECT @first_user_id, @admin_role_id, NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM user_roles
    WHERE user_id = @first_user_id AND role_id = @admin_role_id
);

-- 同步到Casbin
INSERT INTO casbin_rule (ptype, v0, v1, v2)
SELECT 'g', @first_user_id, 'admin', ''
WHERE NOT EXISTS (
    SELECT 1 FROM casbin_rule
    WHERE ptype = 'g' AND v0 = @first_user_id AND v1 = 'admin'
);

-- 显示结果
SELECT
    u.id as user_id,
    u.email,
    u.name,
    r.name as role_name
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.role_id
WHERE u.id = @first_user_id;
```

执行脚本：

```bash
mysql -u root -p123456 orris-dev < scripts/assign_first_admin.sql
```

## 方法3：指定邮箱分配admin

如果知道用户邮箱：

```sql
-- 设置要提升为admin的用户邮箱
SET @admin_email = 'admin@example.com';

-- 获取用户ID和角色ID
SET @user_id = (SELECT id FROM users WHERE email = @admin_email);
SET @admin_role_id = (SELECT id FROM roles WHERE slug = 'admin');

-- 分配admin角色
INSERT INTO user_roles (user_id, role_id, created_at)
VALUES (@user_id, @admin_role_id, NOW())
ON DUPLICATE KEY UPDATE created_at = NOW();

-- 同步到Casbin
INSERT INTO casbin_rule (ptype, v0, v1, v2)
VALUES ('g', @user_id, 'admin', '')
ON DUPLICATE KEY UPDATE v1 = 'admin';

-- 验证
SELECT
    u.email,
    r.name as role
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.role_id
WHERE u.email = @admin_email;
```

## 验证Admin权限

分配完成后，登录并测试：

```bash
# 1. 登录获取token
TOKEN=$(curl -s -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin@123456"
  }' | jq -r '.data.access_token')

# 2. 查看我的角色
curl -X GET http://localhost:8081/auth/roles \
  -H "Authorization: Bearer $TOKEN" | jq

# 应该返回包含admin角色的响应

# 3. 查看我的权限
curl -X GET http://localhost:8081/auth/permissions \
  -H "Authorization: Bearer $TOKEN" | jq

# 应该返回所有权限

# 4. 测试admin权限 - 尝试给其他用户分配角色
curl -X POST http://localhost:8081/users/2/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_ids": [2]
  }'

# 应该成功（只有admin可以分配角色）
```

## 常见问题

### Q: 为什么需要同步到casbin_rule表？
A: Casbin通过casbin_rule表来执行权限检查。虽然user_roles表记录了用户-角色关系，但Casbin策略执行器需要在casbin_rule表中有对应的`g`（grouping）记录。

### Q: 分配后还是没有权限？
A: 检查以下几点：
1. 确认user_roles表中有记录
2. 确认casbin_rule表中有记录（ptype='g'）
3. 重启应用，让Casbin重新加载策略
4. 检查JWT token是否是最新的（重新登录获取新token）

### Q: 可以一个用户分配多个角色吗？
A: 可以！只需要在user_roles表中添加多条记录：

```sql
-- 同时分配admin和user角色
INSERT INTO user_roles (user_id, role_id, created_at) VALUES
(1, 1, NOW()),  -- admin
(1, 2, NOW());  -- user

-- 同步到Casbin
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
('g', '1', 'admin', ''),
('g', '1', 'user', '');
```

### Q: 如何撤销admin角色？
A: 删除对应的记录：

```sql
-- 撤销用户ID为1的admin角色
DELETE FROM user_roles
WHERE user_id = 1 AND role_id = (SELECT id FROM roles WHERE slug = 'admin');

-- 从Casbin中删除
DELETE FROM casbin_rule
WHERE ptype = 'g' AND v0 = '1' AND v1 = 'admin';
```

## 快捷命令

保存为 `assign-admin.sh`:

```bash
#!/bin/bash

# 配置
DB_USER="root"
DB_PASS="123456"
DB_NAME="orris-dev"
ADMIN_EMAIL="${1:-admin@example.com}"

# 执行SQL
mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" <<EOF
SET @user_id = (SELECT id FROM users WHERE email = '$ADMIN_EMAIL');
SET @admin_role_id = (SELECT id FROM roles WHERE slug = 'admin');

INSERT INTO user_roles (user_id, role_id, created_at)
VALUES (@user_id, @admin_role_id, NOW())
ON DUPLICATE KEY UPDATE created_at = NOW();

INSERT INTO casbin_rule (ptype, v0, v1, v2)
VALUES ('g', @user_id, 'admin', '')
ON DUPLICATE KEY UPDATE v1 = 'admin';

SELECT
    u.id,
    u.email,
    u.name,
    r.name as role
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.role_id
WHERE u.email = '$ADMIN_EMAIL';
EOF

echo "✅ Admin role assigned to $ADMIN_EMAIL"
echo "🔄 Please restart the application for changes to take effect"
```

使用：

```bash
chmod +x assign-admin.sh
./assign-admin.sh admin@example.com
```
