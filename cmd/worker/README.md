# Traffic Sync Worker

## Overview

This worker synchronizes node traffic statistics from Redis cache to MySQL database periodically.

## Features

- **Periodic Sync**: Automatically flushes Redis traffic data to MySQL every 5 minutes
- **Atomic Operations**: Uses atomic increment operations to prevent race conditions
- **Graceful Shutdown**: Performs final flush before shutdown on SIGINT/SIGTERM
- **Error Handling**: Continues operation even if individual node sync fails
- **Memory Safety**: Redis keys expire after 24 hours to prevent memory leaks

## Usage

### Build

```bash
go build -o worker ./cmd/worker/main.go
```

### Run

```bash
# Development environment
./worker development

# Production environment
./worker production

# Using environment variable
ENV=production ./worker
```

### Configuration

The worker uses the same configuration file as the main application (`configs/config.yaml`):

```yaml
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

database:
  host: localhost
  port: 3306
  username: root
  password: password
  database: orris_dev
```

## Architecture

### Redis Key Format

- **Key Pattern**: `node:{node_id}:traffic`
- **Fields**:
  - `upload`: Total upload bytes
  - `download`: Total download bytes
- **Expiration**: 24 hours

### Sync Process

1. Scan all `node:*:traffic` keys in Redis
2. For each key:
   - Parse node ID
   - Get upload and download values
   - Calculate total traffic
   - Atomically increment MySQL `traffic_used` field
   - Delete Redis key after successful sync
3. Log sync statistics (flushed count, error count)

## Deployment

### Systemd Service (Linux)

Create `/etc/systemd/system/orris-traffic-worker.service`:

```ini
[Unit]
Description=Orris Traffic Sync Worker
After=network.target mysql.service redis.service

[Service]
Type=simple
User=orris
WorkingDirectory=/opt/orris
Environment=ENV=production
ExecStart=/opt/orris/worker production
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable orris-traffic-worker
sudo systemctl start orris-traffic-worker
sudo systemctl status orris-traffic-worker
```

### Docker

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o worker ./cmd/worker/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/worker .
COPY --from=builder /app/configs ./configs
CMD ["./worker", "production"]
```

### Supervisor (Unix-like systems)

Create `/etc/supervisor/conf.d/orris-traffic-worker.conf`:

```ini
[program:orris-traffic-worker]
command=/opt/orris/worker production
directory=/opt/orris
user=orris
autostart=true
autorestart=true
stderr_logfile=/var/log/orris/worker.err.log
stdout_logfile=/var/log/orris/worker.out.log
```

## Monitoring

### Logs

The worker logs important events:

- **INFO**: Startup, scheduled syncs, shutdown, sync statistics
- **ERROR**: Failed syncs, connection errors
- **DEBUG**: Individual node sync details

### Metrics to Monitor

- Flushed count per sync cycle
- Error count per sync cycle
- Sync duration
- Redis connection health
- MySQL connection health

## Troubleshooting

### Worker not syncing

1. Check Redis connection: `redis-cli ping`
2. Check MySQL connection: `mysql -u username -p`
3. Verify configuration file path
4. Check worker logs for errors

### High error count

1. Check MySQL disk space
2. Verify node IDs in Redis exist in MySQL
3. Check MySQL connection pool settings
4. Review MySQL slow query log

### Memory issues

- Redis keys expire after 24 hours automatically
- Ensure sync interval is reasonable (default: 5 minutes)
- Monitor Redis memory usage

## Development

### Testing

```bash
# Build
go build ./cmd/worker/main.go

# Run with development config
./worker development
```

### Code Structure

- **Main**: `cmd/worker/main.go`
- **Cache Interface**: `internal/infrastructure/cache/trafficcache.go`
- **Redis Implementation**: `internal/infrastructure/cache/redistrafficcache.go`
- **Repository**: `internal/infrastructure/repository/noderepository.go`
