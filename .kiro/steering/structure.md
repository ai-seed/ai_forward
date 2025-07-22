# Project Structure

## Clean Architecture Layout

The project follows Clean Architecture principles with strict layer separation:

```
ai-api-gateway/
├── cmd/                    # Application entry points
│   └── server/            # Main server startup
├── internal/              # Private application code (Clean Architecture layers)
│   ├── domain/           # Domain layer (business logic core)
│   │   ├── entities/     # Business entities
│   │   ├── repositories/ # Repository interfaces
│   │   ├── services/     # Domain services
│   │   └── values/       # Value objects
│   ├── application/      # Application layer (use cases)
│   │   ├── dto/          # Data transfer objects
│   │   ├── services/     # Application services
│   │   └── utils/        # Application utilities
│   ├── infrastructure/   # Infrastructure layer (external concerns)
│   │   ├── async/        # Async processing (quota consumers)
│   │   ├── cache/        # Redis caching implementation
│   │   ├── clients/      # External API clients
│   │   ├── config/       # Configuration management
│   │   ├── database/     # Database connections and migrations
│   │   ├── functioncall/ # Function calling integrations
│   │   ├── gateway/      # API gateway logic
│   │   ├── logger/       # Logging implementation
│   │   ├── redis/        # Redis client setup
│   │   ├── repositories/ # Repository implementations
│   │   └── session/      # Session management
│   └── presentation/     # Presentation layer (HTTP interface)
│       ├── handlers/     # HTTP request handlers
│       ├── middleware/   # HTTP middleware
│       ├── routes/       # Route definitions
│       └── utils/        # Presentation utilities
├── web/                  # Frontend React application
├── configs/              # Configuration files (YAML)
├── migrations/           # Database migration files
├── docs/                 # API documentation and guides
├── scripts/              # Build and utility scripts
├── test/                 # Integration tests
└── data/                 # Local database files (SQLite)
```

## Key Architectural Principles

### Layer Dependencies
- **Domain**: No external dependencies (pure business logic)
- **Application**: Depends only on Domain layer
- **Infrastructure**: Implements Domain interfaces, depends on Domain/Application
- **Presentation**: Depends on Application layer, orchestrates requests

### Database Design
- **No Foreign Key Constraints**: Relationships managed at application layer
- **GORM Models**: Located in `internal/infrastructure/database/models/`
- **Repository Pattern**: Interfaces in Domain, implementations in Infrastructure
- **Migration System**: Sequential numbered migrations in `/migrations`

### Configuration Management
- **Viper-based**: YAML configuration in `/configs`
- **Environment-specific**: Support for development/production configs
- **Hierarchical**: Nested configuration structures for different components

### Caching Strategy
- **Multi-layer**: Entity, query, and statistics caching
- **Redis-based**: Centralized cache with configurable TTLs
- **Auto-invalidation**: Intelligent cache invalidation on data changes

### API Design
- **RESTful**: Standard HTTP methods and status codes
- **Swagger Documentation**: Auto-generated from code annotations
- **Middleware Chain**: Authentication, logging, rate limiting, CORS
- **Error Handling**: Consistent error response format

## File Naming Conventions

- **Go Files**: Snake_case for packages, PascalCase for types
- **Database Models**: Singular nouns (User, ApiKey, Model)
- **Repository Files**: `{entity}_repository.go`
- **Service Files**: `{domain}_service.go`
- **Handler Files**: `{resource}_handler.go`
- **Migration Files**: `{number}_{description}.up/down.sql`

## Import Organization

```go
// Standard library
import (
    "context"
    "fmt"
)

// Third-party packages
import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

// Internal packages (domain first, then application, infrastructure, presentation)
import (
    "ai-api-gateway/internal/domain/entities"
    "ai-api-gateway/internal/application/services"
    "ai-api-gateway/internal/infrastructure/database"
)
```

## Testing Structure

- **Unit Tests**: Alongside source files with `_test.go` suffix
- **Integration Tests**: In `/test` directory
- **Mocks**: Generated or manual mocks for external dependencies
- **Test Data**: Fixtures and test databases in test directories