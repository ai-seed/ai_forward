# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an AI API Gateway written in Go that provides a unified interface for multiple AI service providers (OpenAI, Anthropic, etc.). It features load balancing, quota management, billing, and user authentication. The project uses Clean Architecture with domain-driven design principles.

## Development Commands

### Building and Running
```bash
# Build the application
make build

# Run the application
make run

# Run tests
make test

# Install dependencies
make deps

# Format code
make fmt

# Lint code  
make lint
```

### Database Operations
```bash
# Run database migrations (check migrations/ directory for available migrations)
# Note: No specific migration command found in Makefile - check if manual migration is needed
```

### Swagger Documentation
```bash
# Generate Swagger documentation
make swagger

# Clean Swagger documentation
make swagger-clean

# Verify Swagger documentation
make swagger-verify
```

### Docker Operations
```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Use Docker Compose for full stack
docker-compose up -d
```

## Architecture Overview

This project follows Clean Architecture with these layers:

### Domain Layer (`internal/domain/`)
- **Entities**: Core business objects (User, APIKey, Model, Provider, Quota, UsageLog)
- **Repositories**: Data access interfaces
- **Services**: Core business logic interfaces
- **Values**: Value objects and generators

### Application Layer (`internal/application/`)  
- **DTOs**: Data Transfer Objects for API communication
- **Services**: Business logic implementations
- **Utils**: Application utilities like pagination

### Infrastructure Layer (`internal/infrastructure/`)
- **Clients**: External API clients (OpenAI, Anthropic, etc.)
- **Database**: GORM-based database connections
- **Gateway**: Load balancing and request routing logic
- **Redis**: Caching and distributed locking
- **Repositories**: Data access implementations

### Presentation Layer (`internal/presentation/`)
- **Handlers**: HTTP request handlers
- **Middleware**: Authentication, rate limiting, quota checks
- **Routes**: Route definitions and middleware setup

## Key Technical Details

### Database Design
- Uses "no foreign key" design - relationships managed at application level
- Supports both SQLite (development) and PostgreSQL (production)
- Database migrations in `migrations/` directory

### Authentication & Authorization
- JWT-based authentication with configurable TTL
- API key-based authentication for AI endpoints
- Middleware-based authorization checks

### Load Balancing & Routing
- Multiple strategies: round_robin, weighted, least_connections
- Automatic failover and health checks
- Provider-specific routing logic

### Caching Strategy
- Redis-based caching with configurable TTL per entity type
- Automatic cache invalidation on entity updates
- Distributed locking for concurrent operations

### Quota Management
- Multi-dimensional quotas (requests, tokens, cost)
- Real-time quota consumption tracking
- Async quota processing for performance

### Billing System
- Precise cost calculation based on model pricing
- Automatic balance deduction (allows negative balance)
- Detailed billing records and usage logs

## Configuration

The main configuration is in `configs/config.yaml` with sections for:
- Server settings (host, port, timeouts)
- Database connection (PostgreSQL/SQLite)
- Redis configuration for caching
- JWT authentication settings
- Rate limiting and quota defaults
- Cache TTL settings per entity type
- Function call and search service settings

## Important Development Notes

### Code Standards
- Follow the existing "reuse first" principle - check for existing components before creating new ones
- Use existing dependencies rather than adding new ones
- Maintain Clean Architecture separation of concerns
- No foreign keys in database - handle relationships in application code

### Testing
- Test files should be in `test/` directory
- Use the existing testing patterns and frameworks
- Run tests with `make test`

### Error Handling
- Use structured logging with logrus
- Errors are defined in `internal/domain/entities/errors.go`
- Return appropriate HTTP status codes and error messages

### API Endpoints
The gateway supports:
- OpenAI-compatible endpoints (`/v1/chat/completions`, `/v1/models`)
- Midjourney endpoints (`/mj/submit/*`)
- Authentication endpoints (`/auth/*`)
- Admin endpoints for user/key management
- Health check endpoint (`/health`)

### Frontend Integration
- React-based frontend in `web/` directory
- TypeScript with Material-UI components
- Separate build process with `package.json`

## Common Patterns

### Service Layer Pattern
Services are injected through constructor dependency injection. Check `internal/application/services/service_factory.go` for service creation patterns.

### Repository Pattern
All data access goes through repository interfaces. GORM implementations are in `internal/infrastructure/repositories/`.

### Middleware Chain
Authentication → Rate Limiting → Quota Check → Request Processing

### Request Flow
1. Authentication middleware validates API key/JWT
2. Rate limiting checks request frequency
3. Quota middleware verifies available quota
4. Gateway service routes to appropriate AI provider
5. Response processing and usage logging
6. Billing and quota consumption

## Development Workflow

1. Make changes following existing patterns
2. Run tests: `make test`
3. Format code: `make fmt`
4. Lint code: `make lint`
5. Update Swagger docs: `make swagger`
6. Build and test: `make build && make run`

## Environment Setup

The project uses:
- Go 1.23+
- PostgreSQL/SQLite for data storage
- Redis for caching and distributed locking
- Docker for containerization
- Make for build automation