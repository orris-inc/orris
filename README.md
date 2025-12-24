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

## Common Commands

```bash
docker compose ps        # Check status
docker compose logs -f   # View logs
docker compose down      # Stop services
docker compose pull      # Update images
docker compose up -d     # Start services
```

## License

[MIT License](./LICENSE)
