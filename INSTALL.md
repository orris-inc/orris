# Orris 安装指南

## 环境要求

- Docker
- Docker Compose

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/orris-inc/orris.git
cd orris
```

### 2. 配置环境

```bash
cp .env.example .env
# 编辑 .env 配置数据库、Redis、OAuth 等
```

### 3. 启动服务

```bash
docker compose up -d
```

### 4. 数据库迁移

```bash
docker exec -it orris_app /app/orris migrate up
```

服务包含：
- **Caddy** - 反向代理 (80, 443)
- **Backend** - API 服务 (8080)
- **Frontend** - 前端应用
- **MySQL** - 数据库 (3306)
- **Redis** - 缓存 (6379)

### 5. 查看状态

```bash
docker compose ps
docker compose logs -f
```

## 更新服务

```bash
docker compose pull
docker compose up -d
```

## 常用命令

| 命令 | 说明 |
|------|------|
| `docker compose up -d` | 启动服务 |
| `docker compose down` | 停止服务 |
| `docker compose pull` | 更新镜像 |
| `docker compose logs -f` | 查看日志 |

## 配置说明

环境变量格式: `ORRIS_<SECTION>_<KEY>`

必须配置:
- `ORRIS_SERVER_BASE_URL` - 服务后端访问地址

关键配置项:
- 数据库: `ORRIS_DATABASE_*`
- Redis: `ORRIS_REDIS_*`
- JWT: `ORRIS_AUTH_JWT_SECRET`
- OAuth: `ORRIS_OAUTH_GOOGLE_*`, `ORRIS_OAUTH_GITHUB_*`

详见 `.env.example`
