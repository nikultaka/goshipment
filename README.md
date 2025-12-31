# Go API System - Phase 1

A fully-featured Go API system with OAuth 2.1, modular architecture, and comprehensive modules for a complete backend solution.


## Features

### Core Infrastructure
- **Modular Monolith Architecture**: Clean separation of concerns with module-based structure
- **PostgreSQL 16**: Primary data store with connection pooling
- **Redis 7 + Sentinel**: Caching and session management with high availability
- **NATS JetStream**: Async message queue for notifications
- **Comprehensive Middleware**: Logging, CORS, rate limiting, error handling, request ID tracking

### OAuth 2.1 Server
- **OAuth 2.1 + OIDC** compliant Authorization Server
- Authorization Code Flow with PKCE
- Client Credentials Flow
- Refresh Token Flow
- Token Revocation
- Token Introspection
- JWT-based access tokens


### Modules
- **Users**: Registration, login, profile management, anonymization
- **API Clients**: Admin-managed OAuth client applications
- **Audit Logs**: Immutable audit trail for all actions
- **Notifications**: Email (SendGrid) and SMS (Twilio) via NATS queue
- **Settings**: System and user-level configuration
- **Storage**: File upload/download (local or S3)
- **Reviews**: User reviews with admin moderation
- **Support**: Ticket system with admin replies
- **Dashboard**: Role-based analytics
- **Analytics**: GA4 server-side event tracking
- **Team Members**: Team directory with visibility controls
- **PDF Generator**: Generate invoices and reports

## Project Structure

```
gogin/
├── cmd/
│   ├── api/              # Main API server
│   └── worker/           # Background worker
├── internal/
│   ├── config/           # Configuration management
│   ├── clients/          # Database, Redis, NATS clients
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Database models
│   ├── response/         # Standard response wrapper
│   └── modules/          # Business logic modules
│       ├── core/         # Health & status endpoints
│       ├── oauth2/       # OAuth2 server
│       ├── users/        # User management
│       ├── apiclient/    # API client management
│       ├── logs/         # Audit logging
│       ├── notifications/ # Notification system
│       ├── settings/     # Settings management
│       ├── storage/      # File storage
│       ├── reviews/      # Review system
│       ├── support/      # Support tickets
│       ├── dashboard/    # Analytics dashboard
│       ├── analytics/    # GA4 tracking
│       ├── team/         # Team directory
│       ├── sendgrid/     # SendGrid wrapper
│       ├── twilio/       # Twilio wrapper
│       ├── pdf/          # PDF generation
│       └── redishelper/  # Redis utilities
├── migrations/           # Database migrations
├── docs/                # API documentation
├── scripts/             # Build scripts
├── .env.example         # Example environment variables
└── go.mod              # Go modules

```

## Getting Started

### Prerequisites
- Go 1.21 or higher
- PostgreSQL 16
- Redis 7
- NATS Server with JetStream

### Installation

1. **Clone the repository**
   ```bash
   cd /Applications/MAMP/htdocs/gogin
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Set up database**
   ```bash
   # Create database
   createdb goapi

   # Run migrations
   psql -U postgres -d goapi -f migrations/001_create_users_table.sql
   psql -U postgres -d goapi -f migrations/002_create_oauth_tables.sql
   psql -U postgres -d goapi -f migrations/003_create_audit_logs_table.sql
   psql -U postgres -d goapi -f migrations/004_create_notifications_table.sql
   psql -U postgres -d goapi -f migrations/005_create_settings_table.sql
   psql -U postgres -d goapi -f migrations/006_create_files_table.sql
   psql -U postgres -d goapi -f migrations/007_create_reviews_table.sql
   psql -U postgres -d goapi -f migrations/008_create_support_tickets_table.sql
   psql -U postgres -d goapi -f migrations/009_create_team_members_table.sql
   ```

5. **Run the server**
   ```bash
   go run cmd/api/main.go
   ```

The server will start on `http://localhost:8080` (or your configured port).

## API Endpoints

### Core
- `GET /` - Root endpoint
- `GET /api/v1/health` - Health check
- `GET /api/v1/status` - Detailed system status

### Response Format

All API responses follow this standard format:

```json
{
  "success": true,
  "message": "Success message",
  "data": {},
  "meta": {
    "timestamp": "2025-01-15T10:30:00Z",
    "request_id": "uuid",
    "version": "v1",
    "actor": {
      "user_id": "uuid",
      "client_id": "client_id",
      "role": "user"
    }
  },
  "errors": []
}
```

## Configuration

Key configuration options in `.env`:

- **App**: Name, environment, port, version
- **Database**: PostgreSQL connection settings
- **Redis**: Redis/Sentinel configuration
- **NATS**: JetStream connection
- **OAuth**: Token expiry, JWT secret
- **SendGrid**: Email service
- **Twilio**: SMS service
- **Storage**: Local or S3 storage
- **GA4**: Analytics tracking

## Development

### Adding a New Module

1. Create module directory: `internal/modules/mymodule/`
2. Create module.go with route registration
3. Create handlers.go with HTTP handlers
4. Create service.go with business logic
5. Register in main.go

### Running Tests

```bash
go test ./...
```

### Building for Production

```bash
go build -o bin/api cmd/api/main.go
```

## Security Features

- JWT-based authentication
- OAuth 2.0 with PKCE
- Rate limiting per IP/user
- CORS protection
- Request ID tracking
- Comprehensive audit logging
- Password hashing (bcrypt)
- Token revocation support

## Monitoring

- Health check endpoint: `/api/v1/health`
- Status endpoint: `/api/v1/status`
- Audit logs in database
- Colored console logging

## Next Steps

- [ ] Complete OAuth2 implementation
- [ ] Implement all module endpoints
- [ ] Add Swagger/OpenAPI documentation
- [ ] Write comprehensive tests
- [ ] Add CI/CD pipeline
- [ ] Docker containerization

## License

Proprietary - All rights reserved
