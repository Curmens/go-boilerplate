# Go API Boilerplate

A production-ready Go boilerplate for building REST APIs with Gin, GORM, and PostgreSQL/MySQL.

## Features

- 🚀 **Gin** - Fast HTTP web framework
- 🗄️ **GORM** - ORM with PostgreSQL and MySQL support
- 🔐 **JWT Authentication** - Secure token-based authentication
- 👮 **RBAC** - Role-based access control with permissions
- 📝 **Structured Logging** - JSON logging with rotation
- ✅ **Request Validation** - Input validation with detailed errors
- 🔄 **Database Migrations** - Auto-migration and seeding
- 🐳 **Docker** - Development and production containerization
- ♻️ **Hot Reload** - Development with Air
- 🛡️ **Security** - CORS, secure headers, rate limiting ready
- 📊 **Health Checks** - Kubernetes-ready liveness/readiness probes

## Project Structure

```
.
├── cmd/
│   └── migrate/          # Database migration CLI
├── config/               # Configuration management
├── controllers/          # HTTP handlers
├── database/             # Database connection and migrations
├── middleware/           # HTTP middleware
├── models/               # GORM models
├── routes/               # Route definitions
├── seeder/               # Database seeders
├── utils/                # Utility functions
├── logs/                 # Application logs (gitignored)
├── uploads/              # File uploads (gitignored)
├── main.go               # Application entry point
├── Makefile              # Development commands
├── docker-compose.yml    # Docker compose configuration
├── Dockerfile.dev        # Development Dockerfile
├── .air.toml             # Hot reload configuration
├── .env.example          # Environment variables template
└── go.mod                # Go module definition
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL or MySQL
- Docker & Docker Compose (optional)

### Setup

1. **Clone and install dependencies**

```bash
git clone <repository>
cd go-boilerplate
make deps
```

2. **Configure environment**

```bash
cp .env.example .env
# Edit .env with your database credentials
```

3. **Run with Docker**

```bash
docker-compose up -d
```

Or run locally:

```bash
make run
```

4. **Run migrations and seed data**

```bash
make migrate
make seed
```

## Development

### Available Commands

```bash
make help           # Show all available commands
make run            # Run the application
make dev            # Run with hot reload (requires air)
make test           # Run tests
make lint           # Run linter
make fmt            # Format code
make build          # Build binary
make migrate        # Run database migrations
make seed           # Run database seeders
make docker-up      # Start Docker containers
make docker-down    # Stop Docker containers
```

### Hot Reload

Install Air for hot reload:

```bash
go install github.com/cosmtrek/air@latest
```

Then run:

```bash
make dev
# or
air
```

## API Endpoints

### Health Checks

| Method | Endpoint   | Description           |
|--------|------------|-----------------------|
| GET    | /health    | Full health status    |
| GET    | /healthz   | Kubernetes liveness   |
| GET    | /ready     | Kubernetes readiness  |
| GET    | /metrics   | Application metrics   |
| GET    | /ping      | Simple ping/pong      |

### Authentication

| Method | Endpoint                    | Description              |
|--------|-----------------------------|--------------------------| 
| POST   | /api/v1/auth/register       | Register new user        |
| POST   | /api/v1/auth/login          | Login user               |
| POST   | /api/v1/auth/refresh        | Refresh access token     |
| POST   | /api/v1/auth/forgot-password| Request password reset   |
| POST   | /api/v1/auth/reset-password | Reset password           |

### User (Authenticated)

| Method | Endpoint              | Description          |
|--------|-----------------------|----------------------|
| GET    | /api/v1/users/me      | Get current user     |
| PUT    | /api/v1/users/me      | Update current user  |
| DELETE | /api/v1/users/me      | Delete current user  |
| PUT    | /api/v1/users/me/password | Change password  |

### Admin (Admin Role Required)

| Method | Endpoint              | Description          |
|--------|-----------------------|----------------------|
| GET    | /api/v1/admin/users   | List all users       |
| GET    | /api/v1/admin/users/:id | Get user           |
| POST   | /api/v1/admin/users   | Create user          |
| PUT    | /api/v1/admin/users/:id | Update user        |
| DELETE | /api/v1/admin/users/:id | Delete user        |

## Configuration

Configuration is loaded from environment variables. See `.env.example` for all available options.

### Key Configuration Options

| Variable          | Description                    | Default        |
|-------------------|--------------------------------|----------------|
| SERVER_PORT       | Server port                    | 8080           |
| GIN_MODE          | Gin mode (debug/release/test)  | debug          |
| DB_DRIVER         | Database driver                | postgres       |
| DB_HOST           | Database host                  | localhost      |
| DB_PORT           | Database port                  | 5432           |
| DB_NAME           | Database name                  | myapp          |
| JWT_SECRET        | JWT signing secret             | (required)     |
| JWT_ACCESS_TOKEN_TTL | Access token duration       | 15m            |
| LOG_LEVEL         | Log level                      | debug          |

## Authentication

This boilerplate uses JWT-based authentication:

1. **Access Token**: Short-lived token (15 min default) for API access
2. **Refresh Token**: Long-lived token (7 days default) for obtaining new access tokens

### Usage

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "User123!"}'

# Use access token
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"

# Refresh token
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

## Database

### Migrations

```bash
# Run migrations
make migrate

# Rollback migrations (DANGEROUS)
make migrate-down

# Reset database (DANGEROUS)
make migrate-reset
```

### Seeders

```bash
# Run all seeders
make seed

# Run specific seeders
make seed-only SEEDERS=roles,users

# List available seeders
make list-seeders
```

### Default Seeded Users

| Email              | Password   | Role        |
|--------------------|------------|-------------|
| admin@example.com  | Admin123!  | super_admin |
| user@example.com   | User123!   | user        |

**⚠️ Change these passwords in production!**

## Middleware

- **RequestID**: Adds unique request ID to each request
- **CORS**: Cross-Origin Resource Sharing
- **Recovery**: Panic recovery with logging
- **Logger**: Request/response logging
- **Auth**: JWT authentication
- **RoleCheck**: Role-based authorization

## Models

### User
- Email, Password, FirstName, LastName
- Role (many-to-one with Role)
- IsActive, IsVerified, LastLoginAt

### Role
- Name, Description, IsActive
- Permissions (many-to-many)

### Permission
- Name, Description, Resource, Action

## Response Format

### Success Response

```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "page": 1,
    "per_page": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": {
      "email": "Invalid email address",
      "password": "Must be at least 8 characters"
    }
  }
}
```

## Testing

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run with race detector
make test-race
```

## Production Deployment

1. Build the binary:

```bash
make build-linux
```

2. Set production environment variables:

```bash
export GIN_MODE=release
export JWT_SECRET=<secure-secret-at-least-32-chars>
export DB_PASSWORD=<secure-password>
```

3. Run migrations:

```bash
./build/api -migrate
```

4. Start the server:

```bash
./build/api
```

### Docker Production Build

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api .

FROM alpine:latest
COPY --from=builder /app/api /api
EXPOSE 8080
CMD ["/api"]
```

## Security Considerations

- ✅ JWT tokens with proper expiration
- ✅ Password hashing with bcrypt
- ✅ Input validation on all endpoints
- ✅ CORS configuration
- ✅ SQL injection prevention (GORM)
- ✅ Request rate limiting (configurable)
- ⚠️ Set strong JWT_SECRET in production
- ⚠️ Use HTTPS in production
- ⚠️ Change default seeded passwords

## Contributing

1. Fork the repository
2. Create your feature branch
3. Run tests and linter
4. Submit a pull request

## License

MIT License
