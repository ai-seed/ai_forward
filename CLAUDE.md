# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an AI API Gateway project consisting of:
- **Backend**: Go-based API gateway (`ai-api-gateway`) providing unified access to multiple AI services
- **Frontend**: React-based dashboard (`web/`) for user management and analytics

The gateway provides OpenAI-compatible endpoints and supports multiple AI providers including Midjourney, Stability.ai, and others.

## Common Development Commands

### Backend (Go)

**Development:**
```bash
# Build the application
make build

# Run the application
make run

# Run with custom config
go run cmd/server/main.go -config=configs/config.yaml

# Run tests
make test

# Install dependencies
make deps
```

**Code Quality:**
```bash
# Format code
make fmt

# Lint code (requires golangci-lint)
make lint
```

**Documentation:**
```bash
# Generate/update Swagger documentation
make swagger

# Clean Swagger docs
make swagger-clean

# Verify Swagger docs
make swagger-verify
```

**Docker:**
```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

### Frontend (React)

Located in `web/` directory:
```bash
cd web

# Development server
npm run dev
# or
yarn dev

# Build for production
npm run build
# or
yarn build

# Type checking
npm run tsc:watch

# Linting
npm run lint
npm run lint:fix

# Code formatting
npm run fm:fix
```

## Architecture

### Backend Structure

The Go backend follows Clean Architecture with these layers:

- **`cmd/server/`** - Application entry point and configuration
- **`internal/domain/`** - Core business entities and interfaces
  - `entities/` - Domain models (User, APIKey, Model, etc.)
  - `repositories/` - Data access interfaces  
  - `services/` - Business logic interfaces
- **`internal/application/`** - Application layer
  - `services/` - Service implementations
  - `dto/` - Data transfer objects
- **`internal/infrastructure/`** - External concerns
  - `database/` - GORM database setup and migrations
  - `redis/` - Redis caching and distributed locks
  - `repositories/` - Repository implementations
  - `clients/` - External API clients
  - `gateway/` - Load balancing and request routing
- **`internal/presentation/`** - HTTP layer
  - `handlers/` - HTTP request handlers
  - `middleware/` - HTTP middleware
  - `routes/` - Route configuration

### Key Components

1. **Gateway Service** - Main orchestrator handling requests, authentication, quota management
2. **Load Balancer** - Routes requests to different AI providers with health checks
3. **Cache System** - Redis-based caching for users, API keys, models, and quotas
4. **Quota System** - Token-based usage tracking and limits
5. **Billing System** - Usage tracking and cost calculation

### Frontend Structure

React SPA using Material-UI with:
- **`src/pages/`** - Main application pages
- **`src/components/`** - Reusable UI components
- **`src/sections/`** - Page-specific sections
- **`src/services/`** - API communication
- **`src/contexts/`** - React context providers

## Configuration

Main configuration file: `configs/config.yaml` (use `config.yaml.example` as template)

Key configuration sections:
- **Server** - Port, timeouts
- **Database** - PostgreSQL connection settings
- **Redis** - Cache and distributed lock settings
- **JWT** - Authentication token settings
- **OAuth** - Google/GitHub OAuth settings
- **AI Providers** - API keys and endpoints for various AI services

## Database

Uses GORM with PostgreSQL. Main entities:
- Users, APIKeys, Quotas
- Models, Providers, ProviderModelSupport
- UsageLogs, BillingRecords
- Tools, MidjourneyJobs

Auto-migration runs on startup.

## API Documentation

Swagger docs available at: `http://localhost:8080/swagger/index.html`

Generate docs after adding new endpoints:
```bash
make swagger
```

## Development Workflow

1. **Configuration**: Copy `configs/config.yaml.example` to `configs/config.yaml` and update settings
2. **Database**: Ensure PostgreSQL is running and accessible
3. **Redis**: Optional but recommended for caching and distributed locks
4. **Backend**: Use `make run` to start the Go server
5. **Frontend**: Use `npm run dev` in the `web/` directory

## Testing

Backend tests use the standard Go testing framework with testify:
```bash
# Run all tests
make test

# Run tests with coverage
go test -v -cover ./...
```

## API Authentication

The system supports:
1. **JWT Authentication** - For user dashboard access
2. **API Key Authentication** - For programmatic API access (Bearer tokens)
3. **OAuth** - Google and GitHub social login

## Deployment

Docker support included:
- Backend Dockerfile in root
- Frontend Dockerfile in `web/`
- Docker Compose configuration available

Environment-specific configurations should override the base config file.