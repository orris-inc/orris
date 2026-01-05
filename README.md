# Orris

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash
```

Or specify domain and admin credentials:

```bash
curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | \
  DOMAIN=orris.example.com \
  ADMIN_EMAIL=admin@example.com \
  ADMIN_PASSWORD=your-password \
  bash
```

## Detailed Installation

See [INSTALL.md](./INSTALL.md) for detailed installation instructions.

## Update

Update to the latest version:

```bash
# From your Orris installation directory
./install.sh update

# Or remotely
curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash -s -- update
```

This will:
1. Pull the latest Docker images
2. Run database migrations
3. Restart all services

## Common Commands

```bash
docker compose ps        # Check status
docker compose logs -f   # View logs
docker compose down      # Stop services
docker compose up -d     # Start services
./install.sh update      # Update to latest version
./install.sh help        # Show help
```

## License

[MIT License](./LICENSE)
