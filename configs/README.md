# Configuration Guide

Orris 支持灵活的配置管理，可以使用配置文件、环境变量或两者混合使用。

## 📋 配置方式

### 优先级顺序（从高到低）

1. **环境变量** (最高优先级)
2. **config.yaml 文件**
3. **代码默认值** (最低优先级)

### 配置格式

#### 环境变量格式
```bash
ORRIS_<SECTION>_<KEY>=value
```

**示例**:
```bash
ORRIS_SERVER_PORT=8080
ORRIS_DATABASE_PASSWORD=my_password
ORRIS_LOGGER_LEVEL=debug
```

#### YAML 配置格式
```yaml
section:
  key: value
```

**示例**:
```yaml
server:
  port: 8080
database:
  password: "my_password"
logger:
  level: "debug"
```

## 🚀 快速开始

### 方式一：使用 config.yaml（推荐开发环境）

1. 复制示例配置文件：
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   ```

2. 编辑 `configs/config.yaml` 设置你的配置：
   ```yaml
   database:
     host: "localhost"
     username: "orris"
     database: "orris_dev"
     # 密码留空，使用环境变量
   ```

3. 设置敏感信息为环境变量：
   ```bash
   export ORRIS_DATABASE_PASSWORD=your_password
   ```

4. 运行应用：
   ```bash
   ./bin/orris server
   ```

### 方式二：仅使用环境变量（推荐生产环境）

1. 设置所有配置为环境变量：
   ```bash
   export ORRIS_SERVER_MODE=release
   export ORRIS_SERVER_PORT=8080
   export ORRIS_DATABASE_HOST=prod-db.example.com
   export ORRIS_DATABASE_USERNAME=prod_user
   export ORRIS_DATABASE_PASSWORD=secure_password
   export ORRIS_DATABASE_DATABASE=orris_production
   export ORRIS_LOGGER_LEVEL=warn
   export ORRIS_LOGGER_FORMAT=json
   ```

2. 运行应用：
   ```bash
   ./bin/orris server
   ```

### 方式三：混合使用

使用 `config.yaml` 作为基础配置，环境变量覆盖特定值：

```bash
# config.yaml 包含大部分配置
# 仅通过环境变量覆盖敏感或环境特定的配置
export ORRIS_DATABASE_PASSWORD=prod_password
export ORRIS_SERVER_MODE=release
./bin/orris server
```

## 📝 完整配置选项

### Server 配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `server.host` | `ORRIS_SERVER_HOST` | `0.0.0.0` | 服务器监听地址 |
| `server.port` | `ORRIS_SERVER_PORT` | `8081` | HTTP 服务端口 |
| `server.mode` | `ORRIS_SERVER_MODE` | `debug` | 运行模式: `debug`, `release`, `test` |

### Database 配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `database.host` | `ORRIS_DATABASE_HOST` | `localhost` | 数据库主机地址 |
| `database.port` | `ORRIS_DATABASE_PORT` | `3306` | MySQL 端口 |
| `database.username` | `ORRIS_DATABASE_USERNAME` | `root` | 数据库用户名 |
| `database.password` | `ORRIS_DATABASE_PASSWORD` | `password` | 数据库密码 ⚠️ |
| `database.database` | `ORRIS_DATABASE_DATABASE` | `orris_dev` | 数据库名称 |
| `database.max_idle_conns` | `ORRIS_DATABASE_MAX_IDLE_CONNS` | `10` | 最大空闲连接数 |
| `database.max_open_conns` | `ORRIS_DATABASE_MAX_OPEN_CONNS` | `100` | 最大打开连接数 |
| `database.conn_max_lifetime` | `ORRIS_DATABASE_CONN_MAX_LIFETIME` | `60` | 连接最大生存时间（分钟） |

### Logger 配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `logger.level` | `ORRIS_LOGGER_LEVEL` | `info` | 日志级别: `debug`, `info`, `warn`, `error` |
| `logger.format` | `ORRIS_LOGGER_FORMAT` | `console` | 日志格式: `console`, `json` |
| `logger.output_path` | `ORRIS_LOGGER_OUTPUT_PATH` | `stdout` | 日志输出: `stdout`, `stderr`, 文件路径 |

## 🔒 安全最佳实践

### ⚠️ 重要安全提示

1. **永远不要将敏感信息提交到 Git**
   - ✅ `config.example.yaml` 可以提交（示例文件）
   - ❌ `config.yaml` 不应提交（已在 `.gitignore` 中）
   - ❌ `.env` 文件不应提交（已在 `.gitignore` 中）

2. **数据库密码管理**
   ```bash
   # 开发环境
   export ORRIS_DATABASE_PASSWORD=dev_password

   # 生产环境（使用密钥管理服务）
   export ORRIS_DATABASE_PASSWORD=$(aws secretsmanager get-secret-value ...)
   ```

3. **生产环境配置**
   - 使用环境变量，不要使用配置文件
   - 使用密钥管理服务（AWS Secrets Manager, HashiCorp Vault, etc.）
   - 定期轮换敏感凭据

4. **权限控制**
   ```bash
   # 确保配置文件权限安全
   chmod 600 configs/config.yaml
   chmod 600 .env
   ```

## 🐳 容器化部署

### Docker Compose

```yaml
version: '3.8'
services:
  orris:
    image: orris:latest
    environment:
      - ORRIS_DATABASE_HOST=mysql
      - ORRIS_DATABASE_USERNAME=orris
      - ORRIS_DATABASE_PASSWORD=${DB_PASSWORD}  # 从 .env 文件读取
      - ORRIS_DATABASE_DATABASE=orris_production
      - ORRIS_SERVER_MODE=release
      - ORRIS_LOGGER_FORMAT=json
    ports:
      - "8080:8080"
```

### Kubernetes

#### ConfigMap（非敏感配置）
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: orris-config
data:
  ORRIS_SERVER_MODE: "release"
  ORRIS_DATABASE_HOST: "mysql-service"
  ORRIS_DATABASE_DATABASE: "orris_production"
  ORRIS_LOGGER_LEVEL: "warn"
  ORRIS_LOGGER_FORMAT: "json"
```

#### Secret（敏感配置）
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: orris-secret
type: Opaque
stringData:
  ORRIS_DATABASE_PASSWORD: "your-secure-password"
```

#### Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: orris
spec:
  template:
    spec:
      containers:
      - name: orris
        image: orris:latest
        envFrom:
        - configMapRef:
            name: orris-config
        - secretRef:
            name: orris-secret
```

## 🧪 环境特定配置

### 开发环境
```bash
export ORRIS_SERVER_MODE=debug
export ORRIS_LOGGER_LEVEL=debug
export ORRIS_DATABASE_HOST=localhost
export ORRIS_DATABASE_PASSWORD=dev_password
```

### 测试环境
```bash
export ORRIS_SERVER_MODE=test
export ORRIS_LOGGER_LEVEL=info
export ORRIS_DATABASE_HOST=test-db
export ORRIS_DATABASE_PASSWORD=test_password
export ORRIS_DATABASE_DATABASE=orris_test
```

### 生产环境
```bash
export ORRIS_SERVER_MODE=release
export ORRIS_SERVER_PORT=80
export ORRIS_LOGGER_LEVEL=warn
export ORRIS_LOGGER_FORMAT=json
export ORRIS_DATABASE_HOST=prod-db.internal
export ORRIS_DATABASE_PASSWORD=$(get-from-vault)
export ORRIS_DATABASE_DATABASE=orris_production
```

## 🔍 故障排查

### 检查配置加载

运行应用时会在日志中显示配置来源：

```bash
./bin/orris server
```

输出示例：
```
INFO  Loading configuration from: configs/config.yaml
INFO  Environment variables override: ORRIS_DATABASE_PASSWORD (set)
INFO  Starting server address=0.0.0.0:8081 mode=debug
```

### 常见问题

1. **数据库连接失败**
   - 检查 `ORRIS_DATABASE_PASSWORD` 是否设置
   - 验证数据库主机和端口可访问

2. **配置文件未找到**
   - 确保在项目根目录运行
   - 检查 `configs/config.yaml` 是否存在

3. **环境变量未生效**
   - 确认环境变量名称格式正确（`ORRIS_` 前缀）
   - 检查变量拼写和大小写

## 📚 更多资源

- [Viper 文档](https://github.com/spf13/viper) - 配置管理库
- [12-Factor App](https://12factor.net/config) - 配置最佳实践
- [环境变量安全](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html) - OWASP 安全指南
