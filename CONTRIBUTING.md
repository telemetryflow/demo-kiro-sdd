<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://github.com/telemetryflow/.github/raw/main/docs/assets/tfo-logo-sdk-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="https://github.com/telemetryflow/.github/raw/main/docs/assets/tfo-logo-sdk-light.svg">
    <img src="https://github.com/telemetryflow/.github/raw/main/docs/assets/tfo-logo-sdk-light.svg" alt="TelemetryFlow Logo" width="80%">
  </picture>

  <h3>Order Service - RESTful API with DDD + CQRS Pattern</h3>

[![Version](https://img.shields.io/badge/Version-1.4.4-orange.svg)](CHANGELOG.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org/)
[![OpenTelemetry](https://img.shields.io/badge/OTLP-100%25%20Compliant-success?logo=opentelemetry)](https://opentelemetry.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://hub.docker.com/r/telemetryflow/telemetryflow-sdk)

</div>

---

# Contributing to Order Service

Thank you for your interest in contributing to the Order Service. This document provides guidelines for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Style Guide](#style-guide)
- [Architecture Guidelines](#architecture-guidelines)
- [Release Process](#release-process)

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment. We expect all contributors to:

- Be respectful and constructive in discussions
- Welcome newcomers and help them learn
- Focus on what is best for the community
- Show empathy towards other community members

## Getting Started

### Prerequisites

| Tool           | Version | Purpose                       |
| -------------- | ------- | ----------------------------- |
| Go             | 1.26+   | Application build and test    |
| Docker         | 24+     | Container runtime             |
| Docker Compose | v2+     | Multi-container orchestration |
| Make           | any     | Build automation              |
| Git            | any     | Source control                |
| golangci-lint  | latest  | Code linting                  |

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/YOUR_USERNAME/order-service.git
cd order-service
```

3. Add the upstream remote:

```bash
git remote add upstream https://github.com/telemetryflow/order-service.git
```

## Development Setup

### Install Dependencies

```bash
# Download Go modules
go mod download

# Configure environment
cp .env.example .env
# Edit .env — set at minimum: DB_PASSWORD, JWT_SECRET

# Start database
docker compose --profile db up -d

# Run migrations
make migrate-up

# Build and run
make run
```

### Verify Setup

```bash
# Health check
curl http://localhost:8080/health

# Generate a token
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"email": "dev@example.com", "role": "admin"}'

# Run tests
make test

# Run linter
make lint

# Build binary
make build
```

### IDE Setup

Recommended setup:

- **VS Code / Kiro** with Go extension (`golang.go`)
- **GoLand** by JetBrains
- **Vim/Neovim** with `gopls` LSP

Useful VS Code settings:

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.formatTool": "goimports",
  "editor.formatOnSave": true
}
```

## Project Structure

```
order-service/
├── cmd/
│   └── api/
│       └── main.go               # Application entry point
├── internal/
│   ├── domain/                   # Domain Layer
│   │   ├── domain.go             # Domain types and interfaces
│   │   ├── entity/               # Domain entities (Order, OrderItem)
│   │   └── repository/           # Repository interfaces (contracts)
│   ├── application/              # Application Layer (CQRS)
│   │   ├── command/              # Write operations (Create, Update, Delete)
│   │   ├── query/                # Read operations (Get, List)
│   │   ├── handler/              # Command & Query handlers
│   │   └── dto/                  # Data Transfer Objects
│   └── infrastructure/           # Infrastructure Layer
│       ├── config/               # Configuration loading (env, YAML)
│       ├── http/                 # HTTP transport
│       │   ├── server.go         # Echo server setup
│       │   ├── router.go         # Route registration
│       │   ├── handler/          # HTTP handlers (auth, order, swagger, health)
│       │   └── middleware/       # Middleware (JWT auth, CORS, rate limit, logger)
│       └── persistence/          # Database implementations (GORM/PostgreSQL)
├── pkg/                          # Shared packages
│   ├── logger/                   # Structured logging
│   ├── response/                 # HTTP response helpers
│   ├── safefile/                 # Safe file operations
│   └── validator/                # Request validation (Echo-compatible)
├── telemetry/                    # TelemetryFlow SDK integration
│   ├── init.go                   # SDK initialization and shutdown
│   ├── logs/                     # Log signal configuration
│   ├── metrics/                  # Metric signal configuration
│   └── traces/                   # Trace signal configuration
├── configs/                      # Service configurations
│   ├── config.yaml               # Application defaults
│   ├── otel/                     # TFO Collector pipeline config
│   ├── prometheus/               # Prometheus scrape config + alerting rules
│   ├── alertmanager/             # Alertmanager routing config
│   ├── grafana/                  # Dashboard JSON models
│   └── jaeger/                   # Sampling strategies
├── docs/                         # Documentation
│   ├── api/                      # OpenAPI spec (openapi.yaml, swagger.json)
│   ├── diagrams/                 # ERD, DFD (Mermaid)
│   ├── postman/                  # Postman collection + environment
│   ├── githooks/                 # Git hook scripts
│   └── wiki/                     # Wiki pages (Getting Started, Architecture, etc.)
├── tests/                        # Tests (external test packages)
│   ├── unit/                     # Unit tests (no I/O)
│   │   ├── application/          # Command/query handler tests
│   │   ├── domain/               # Entity and domain logic tests
│   │   ├── infrastructure/       # Middleware, config, HTTP handler tests
│   │   ├── observability/        # Prometheus rule validation tests
│   │   ├── pkg/                  # Validator, response helper tests
│   │   └── telemetry/            # SDK initialization tests
│   ├── integration/              # Integration tests (require Docker)
│   │   ├── observability/        # Live Prometheus/Alertmanager queries
│   │   ├── api_test.go           # API endpoint integration tests
│   │   └── order_api_test.go     # Order CRUD integration tests
│   ├── e2e/                      # End-to-end tests
│   ├── mocks/                    # Shared mock implementations
│   └── fixtures/                 # Test data fixtures
├── migrations/                   # Database migration files
├── scripts/                      # Utility scripts (hooks, run, test)
├── .github/                      # GitHub Actions (CI, Docker, Release)
├── Dockerfile                    # Multi-stage Docker build
├── docker-compose.yml            # Profile-based orchestration (db, app, monitoring, platform, all)
├── docker-compose.prometheus.yml # Prometheus-only compose override
├── Makefile                      # Build, test, lint, migrate automation
├── .golangci.yml                 # Linter configuration
├── go.mod                        # Go module definition
└── go.sum                        # Dependency checksums
```

## Making Changes

### Branch Naming

Use descriptive branch names:

- `feature/add-order-status-history`
- `fix/auth-middleware-public-routes`
- `docs/update-api-documentation`
- `refactor/simplify-order-handler`
- `test/add-integration-coverage`

### Creating a Branch

```bash
# Sync with upstream
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name
```

### Commit Messages

Follow conventional commits format:

```
type(scope): short description

Longer description if needed.

Fixes #123
```

**Types:**

| Type       | Purpose                  |
| ---------- | ------------------------ |
| `feat`     | New feature              |
| `fix`      | Bug fix                  |
| `docs`     | Documentation changes    |
| `test`     | Adding or updating tests |
| `refactor` | Code refactoring         |
| `chore`    | Maintenance tasks        |
| `ci`       | CI/CD changes            |

**Examples:**

```
feat(auth): add auto-generated user_id to token endpoint

Remove user_id from request body and auto-generate a UUID
for the JWT claims. Only email and role are now required.

Fixes #45
```

```
fix(router): prevent auth middleware from intercepting public routes

Split v1 group into independent v1Public and v1Protected groups
to avoid Echo sub-group middleware bleed.
```

## Testing

### Running Tests

```bash
# Run all unit tests
make test

# Run with verbose output
go test -v ./...

# Run specific package
go test -v ./internal/application/handler/...

# Run integration tests (requires docker stack)
docker compose --profile monitoring up -d
INTEGRATION_TEST=true go test -v -timeout 5m ./tests/integration/...

# Run with coverage
make test-coverage

# Run short tests only (skip integration)
make test-short
```

### Writing Tests

Guidelines:

1. Use external test packages (`package foo_test`) to test the public API surface
2. Use `testify/assert` and `testify/require` for assertions
3. Use table-driven tests for multiple cases
4. Use `pgregory.net/rapid` for property-based tests
5. Mock external dependencies via interfaces

Example:

```go
func TestCreateOrder(t *testing.T) {
    tests := []struct {
        name    string
        input   command.CreateOrderCommand
        wantErr bool
    }{
        {
            name:    "valid order",
            input:   command.CreateOrderCommand{CustomerID: uuid.New(), Total: 99.99},
            wantErr: false,
        },
        {
            name:    "zero total",
            input:   command.CreateOrderCommand{CustomerID: uuid.New(), Total: 0},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // arrange, act, assert
        })
    }
}
```

### Test Organization

| Directory                          | Scope                      | Requires Infrastructure |
| ---------------------------------- | -------------------------- | ----------------------- |
| `tests/unit/`                      | Pure logic, no I/O         | No                      |
| `tests/integration/`               | Database, HTTP, Prometheus | Yes (Docker)            |
| `tests/e2e/`                       | Full API workflows         | Yes (full stack)        |
| `tests/unit/observability/`        | Prometheus rule validation | No (reads YAML files)   |
| `tests/integration/observability/` | Live Prometheus queries    | Yes (monitoring stack)  |

## Submitting Changes

### Before Submitting

```bash
# Format code
make fmt

# Run vet
go vet ./...

# Run linter (must pass with 0 issues)
make lint

# Run tests
make test

# Build successfully
make build
```

### Pull Request Process

1. Push your branch to your fork:

```bash
git push origin feature/your-feature-name
```

2. Create a Pull Request on GitHub

3. Fill in the PR template with:
   - Summary of changes
   - What was tested
   - Related issue numbers

4. Wait for CI to pass and address review feedback

### PR Title Format

Same format as commit messages:

```
feat(orders): add order status history tracking
fix(middleware): handle rate limit edge case
docs(wiki): update observability page
```

## Style Guide

### Go Style

- Follow standard Go conventions (`gofmt`, `goimports`)
- Use `golangci-lint` with the project's `.golangci.yml` config
- Keep functions short and focused (< 40 lines preferred)
- Return errors, don't panic

### Naming Conventions

| Type       | Convention    | Example                            |
| ---------- | ------------- | ---------------------------------- |
| Packages   | lowercase     | `handler`, `middleware`, `config`  |
| Interfaces | -er suffix    | `OrderRepository`, `Logger`        |
| Structs    | PascalCase    | `OrderHandler`, `JWTClaims`        |
| Functions  | PascalCase    | `NewOrderHandler`, `CreateToken`   |
| Variables  | camelCase     | `orderID`, `httpClient`            |
| Constants  | PascalCase    | `DefaultTimeout`, `MaxRetries`     |
| Files      | snake_case    | `order_handler.go`, `auth_test.go` |
| Test files | \_test suffix | `order_handler_test.go`            |

### Error Handling

- Always handle errors explicitly
- Wrap errors with context using `fmt.Errorf("...: %w", err)`
- Use custom error types for domain errors
- Return appropriate HTTP status codes

```go
// Good
if err := repo.Create(ctx, order); err != nil {
    return fmt.Errorf("creating order: %w", err)
}

// Bad
repo.Create(ctx, order) // error ignored
```

### Documentation

- Document all exported types, functions, and methods
- Use godoc conventions (comment starts with the name)
- Keep comments concise and useful

```go
// OrderHandler handles HTTP requests for order operations.
type OrderHandler struct {
    repo repository.OrderRepository
}

// Create handles POST /api/v1/orders.
func (h *OrderHandler) Create(c echo.Context) error {
    // ...
}
```

## Architecture Guidelines

### Domain-Driven Design

1. **Entities** (`internal/domain/entity/`): Core business objects with identity
2. **Repository Interfaces** (`internal/domain/repository/`): Define persistence contracts
3. **Value Objects**: Immutable types that validate on creation

### CQRS Pattern

1. **Commands** (`internal/application/command/`): Write operations that change state
2. **Queries** (`internal/application/query/`): Read operations that return data
3. **Handlers** (`internal/application/handler/`): Execute commands and queries
4. **DTOs** (`internal/application/dto/`): Data transfer between layers

### Dependency Direction

```
HTTP Handler → Application Handler → Domain Entity
                                    → Repository Interface
                                              ↑
                              Infrastructure (implements)
```

Dependencies always point inward. Infrastructure implements domain interfaces.

### Adding New Features

1. **Define the entity** in `internal/domain/entity/`
2. **Define repository interface** in `internal/domain/repository/`
3. **Create commands/queries** in `internal/application/command/` and `query/`
4. **Implement handlers** in `internal/application/handler/`
5. **Implement persistence** in `internal/infrastructure/persistence/`
6. **Add HTTP handler** in `internal/infrastructure/http/handler/`
7. **Register routes** in `internal/infrastructure/http/router.go`
8. **Write tests** in `tests/unit/` and `tests/integration/`
9. **Update OpenAPI spec** in `docs/api/openapi.yaml` and `swagger.json`

## Release Process

Releases follow semantic versioning (SemVer):

- **MAJOR**: Breaking API changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### CI/CD Workflows

| Workflow      | Trigger           | Purpose                        |
| ------------- | ----------------- | ------------------------------ |
| `ci.yml`      | Push/PR           | Lint, test, build verification |
| `docker.yml`  | Push to main/tags | Build Docker images            |
| `release.yml` | Tags (v*.*.\*)    | Create GitHub release          |

### Creating a Release

1. Update `VERSION` in `Makefile`
2. Update version badges in `README.md`, `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, `CONTRIBUTING.md`
3. Add release entry to `CHANGELOG.md`
4. Update `TELEMETRYFLOW_SERVICE_VERSION` in `docker-compose.yml`, `.env.example`, `configs/config.yaml`
5. Create and push tag:

```bash
git tag v1.4.4
git push origin v1.4.4
```

GitHub Actions will automatically:

- Run tests and linter
- Build Docker images
- Create GitHub release with binaries

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Open a GitHub Issue with the bug template
- **Features**: Open a GitHub Issue with the feature request template
- **Security**: Email security@telemetryflow.id (do not open public issues)

## Recognition

Contributors are recognized in:

- GitHub Contributors page
- CHANGELOG.md for significant contributions

Thank you for contributing to the Order Service!

---

Built with care by the **Telemetri Data Indonesia** community
