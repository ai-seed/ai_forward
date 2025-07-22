# Technology Stack

## Backend (Go)

- **Language**: Go 1.23+
- **Framework**: Gin (HTTP router/middleware)
- **Database**: PostgreSQL (production) / SQLite (development)
- **ORM**: GORM v2 with no foreign key constraints
- **Cache**: Redis with connection pooling
- **Authentication**: JWT with golang-jwt/jwt/v5
- **Documentation**: Swagger/OpenAPI with swaggo
- **Logging**: Logrus structured logging
- **Configuration**: Viper with YAML configs
- **Testing**: Testify framework

## Frontend (React)

- **Framework**: React 19+ with TypeScript
- **UI Library**: Material-UI (MUI) v7
- **Build Tool**: Vite
- **State Management**: React hooks
- **Routing**: React Router DOM v7
- **HTTP Client**: Axios
- **Internationalization**: i18next
- **Charts**: ApexCharts with react-apexcharts

## Infrastructure

- **Containerization**: Docker with multi-stage builds
- **Orchestration**: Docker Compose
- **Database Migrations**: Custom migration system in `/migrations`
- **Health Checks**: Built-in health endpoints
- **Monitoring**: Configurable logging levels and formats

## Build Commands

### Backend
```bash
# Install dependencies
go mod tidy

# Build application
make build
# or: go build -o bin/server cmd/server/main.go

# Run development server
make run
# or: go run cmd/server/main.go

# Run tests
make test
# or: go test -v ./...

# Generate Swagger docs
make swagger

# Clean build artifacts
make clean
```

### Frontend
```bash
# Install dependencies
cd web && npm install
# or: cd web && yarn install

# Development server
npm run dev

# Build for production
npm run build

# Lint and format
npm run lint:fix
npm run fm:fix
```

### Docker
```bash
# Build image
make docker-build

# Run container
make docker-run

# Full stack with compose
docker-compose up -d
```

## Key Dependencies

- **gin-gonic/gin**: HTTP web framework
- **gorm.io/gorm**: ORM with PostgreSQL driver
- **go-redis/redis/v8**: Redis client
- **swaggo/swag**: Swagger documentation generator
- **spf13/viper**: Configuration management
- **golang-jwt/jwt/v5**: JWT authentication
- **google/uuid**: UUID generation
- **golang.org/x/crypto**: Password hashing
- **golang.org/x/time**: Rate limiting utilities