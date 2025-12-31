### Quick Start Guide v2

Run the API stack through Docker.  
Only **Nginx** should run directly on the host (bare metal).

---

### Nginx Configuration

Update your main Nginx configuration file (usually at `/etc/nginx/nginx.conf`)  
so it loads both the default sites and your Go API configs.

```nginx
http {
    ...
    ...

    # Existing includes
    include /etc/nginx/sites-enabled/*;

    # Add this line to include API configs
    include /srv/api/go-skeleton/nginx/*.conf;

}
```

### Docker Usage

You can run the full stack (API, Worker, Database, Redis, NATS) using Docker Compose.

### Build and Start All Services
```bash
cd infra
docker compose up -d --build
```

### Build and Run Only the API
```bash
cd infra
docker compose up -d --build api
```

### Build and Run Only the Worker
```bash
cd infra
docker compose up -d --build worker
```

### Stop and Remove All Containers
```bash
cd infra
docker compose down
```

### Remove All Containers and Volumes (Clean Reset)
```bash
cd infra
docker compose down -v
```

### View Logs
```bash
docker compose logs -f
# or for a specific service
docker compose logs -f api
docker compose logs -f worker
```

### End of Quickstart v2

---

# Quick Start Guide

Get your Go API System up and running in 5 minutes!

## Prerequisites Check

Ensure you have these installed:
- Go 1.21+ → `go version`
- PostgreSQL 16 → `psql --version`
- Redis 7 → `redis-cli --version`
- NATS Server → `nats-server --version`

## Setup Steps

### 1. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` and set at minimum:
```bash
# Critical settings
DB_PASSWORD=your_secure_password
JWT_SECRET=your_very_secure_jwt_secret_key_here_min_32_chars
```

### 2. Start Services

```bash
# Start PostgreSQL (if not running)
# macOS: brew services start postgresql@16
# Linux: sudo systemctl start postgresql

# Start Redis (if not running)
redis-server

# Start NATS (if not running)
nats-server -js  # -js enables JetStream
```

### 3. Setup Database

```bash
# Create database
createdb goapi

# Run migrations
make migrate
# OR manually:
# ./scripts/migrate.sh up
```

### 4. Install Dependencies

```bash
make install
```

### 5. Run the Server

```bash
make run
```

The server will start at `http://localhost:8080`

## Verify Installation

### 1. Check Health
```bash
curl http://localhost:8080/api/v1/health
```

Expected response:
```json
{
  "success": true,
  "message": "OK",
  "data": {
    "status": "healthy",
    "time": "2025-01-15T10:30:00Z"
  },
  "meta": {
    "timestamp": "2025-01-15T10:30:00Z",
    "request_id": "uuid-here",
    "version": "v1",
    "actor": {
      "user_id": "",
      "client_id": "",
      "role": ""
    }
  }
}
```

### 2. Check Status
```bash
curl http://localhost:8080/api/v1/status
```

This returns detailed status of all services (database, Redis, NATS).

## Common Commands

```bash
make help           # Show all available commands
make run            # Run the server
make build          # Build binary
make test           # Run tests
make migrate        # Run database migrations
make migrate-reset  # Reset database
make clean          # Clean build artifacts
make fmt            # Format code
```

## Development Workflow

### With Live Reload (Recommended)

```bash
# Install air for live reload
go install github.com/air-verse/air@latest

# Run with live reload
make dev
```

### Manual Testing

```bash
# Terminal 1: Run server
make run

# Terminal 2: Test endpoints
curl http://localhost:8080/api/v1/health
```

## What's Next?

Phase 1 foundation is complete! Next steps:

1. **Implement OAuth2 Server** - Token generation, validation, revocation
2. **Build User Module** - Registration, login, profile management
3. **Add RBAC System** - Fine-grained permission control
4. **Create Additional Modules** - Notifications, Storage, Reviews, etc.
5. **Add Swagger Documentation** - OpenAPI 3.1 with interactive docs
6. **Write Tests** - Comprehensive test coverage

See the main [README.md](README.md) for detailed documentation.

## Troubleshooting

### Database Connection Error
- Check PostgreSQL is running: `pg_isready`
- Verify credentials in `.env`
- Ensure database exists: `psql -l | grep goapi`

### Redis Connection Error
- Check Redis is running: `redis-cli ping`
- Verify Redis address in `.env`

### NATS Connection Error
- Check NATS is running: `nats-server --version`
- Ensure JetStream is enabled: `nats-server -js`
- Verify NATS URLs in `.env`

### Port Already in Use
- Change `APP_PORT` in `.env`
- Or kill the process: `lsof -ti:8080 | xargs kill -9`

## Project Structure Overview

```
gogin/
├── cmd/api/              # Main application entry
├── internal/
│   ├── clients/          # Database, Redis, NATS clients
│   ├── config/           # Configuration management
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── modules/          # Business logic
│   └── response/         # Standard responses
├── migrations/           # Database migrations
├── scripts/             # Helper scripts
├── .env.example         # Environment template
├── Makefile            # Build automation
└── README.md           # Full documentation
```

## Need Help?

- Read the full [README.md](README.md)
- Check existing issues
- Review code comments
