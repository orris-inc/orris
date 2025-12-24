# Orris Installation Guide

## Requirements

- Docker
- Docker Compose

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/orris-inc/orris.git
cd orris
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env to configure database, Redis, OAuth, etc.
```

### 3. Start Services

```bash
docker compose up -d
```

### 4. Run Database Migrations

```bash
docker exec -it orris_app /app/orris migrate up
```

Services included:
- **Caddy** - Reverse proxy (80, 443)
- **Backend** - API service (8080)
- **Frontend** - Web application
- **MySQL** - Database (3306)
- **Redis** - Cache (6379)

### 5. Check Status

```bash
docker compose ps
docker compose logs -f
```

## Update Services

```bash
docker compose pull
docker compose up -d
```

## Common Commands

| Command | Description |
|---------|-------------|
| `docker compose up -d` | Start services |
| `docker compose down` | Stop services |
| `docker compose pull` | Update images |
| `docker compose logs -f` | View logs |

## Configuration

Environment variable format: `ORRIS_<SECTION>_<KEY>`

Required:
- `ORRIS_SERVER_BASE_URL` - Backend service URL

Key settings:
- Database: `ORRIS_DATABASE_*`
- Redis: `ORRIS_REDIS_*`
- JWT: `ORRIS_AUTH_JWT_SECRET`
- OAuth: `ORRIS_OAUTH_GOOGLE_*`, `ORRIS_OAUTH_GITHUB_*`

See `.env.example` for details.
