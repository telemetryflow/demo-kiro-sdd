# Order Service — Wiki

Welcome to the **Order Service** technical documentation. This wiki provides comprehensive guides for developers, SREs, and operators working with the service.

---

## Quick Navigation

| Section                                     | Description                                                                   |
| ------------------------------------------- | ----------------------------------------------------------------------------- |
| [Getting Started](Getting-Started.md)       | Prerequisites, setup, and running the service locally                         |
| [Architecture](Architecture.md)             | System design, DDD layers, CQRS flow, and infrastructure topology             |
| [Observability](Observability.md)           | Telemetry pipeline, metrics, traces, exemplars, and instrumentation           |
| [Alerting](Alerting.md)                     | Alert rules, Alertmanager configuration, and on-call runbooks                 |
| [Grafana Dashboards](Grafana-Dashboards.md) | Unified metrics + logs + traces correlation dashboard and exemplar drill-down |
| [Docker Compose](Docker-Compose.md)         | Service deployment, profiles, networking, and environment variables           |
| [Testing](Testing.md)                       | Test suite structure, conventions, and coverage strategy                      |
| [Running Tests](Running-Tests.md)           | Detailed test execution guide — unit, integration, E2E, CI                    |
| [Troubleshooting](Troubleshooting.md)       | Common issues, diagnostics, and resolution steps                              |

---

## About This Service

**Order Service** is a Go REST API implementing Domain-Driven Design (DDD) with Command Query Responsibility Segregation (CQRS). It manages customer orders and order items, with full distributed tracing and metrics observability via OpenTelemetry.

### Key Facts

| Property          | Value                                                                 |
| ----------------- | --------------------------------------------------------------------- |
| Language          | Go 1.26+                                                              |
| Framework         | Echo (HTTP) with otelecho auto-instrumentation                        |
| Database          | PostgreSQL 18                                                         |
| Telemetry         | OTLP gRPC → TFO Collector → Prometheus / Jaeger / Loki / TFO Platform |
| Monitoring        | Prometheus + Alertmanager + Grafana + Jaeger + Loki                   |
| Container Runtime | Docker Compose with profile-based deployment                          |
| License           | Apache 2.0                                                            |

### API Endpoints

| Method   | Path                       | Description             |
| -------- | -------------------------- | ----------------------- |
| `GET`    | `/health`                  | Health check            |
| `GET`    | `/ready`                   | Readiness check         |
| `GET`    | `/api/v1/orders`           | List orders (paginated) |
| `POST`   | `/api/v1/orders`           | Create order            |
| `GET`    | `/api/v1/orders/{id}`      | Get order by ID         |
| `PUT`    | `/api/v1/orders/{id}`      | Update order            |
| `DELETE` | `/api/v1/orders/{id}`      | Delete order            |
| `GET`    | `/api/v1/order-items`      | List order items        |
| `POST`   | `/api/v1/order-items`      | Create order item       |
| `GET`    | `/api/v1/order-items/{id}` | Get order item by ID    |
| `PUT`    | `/api/v1/order-items/{id}` | Update order item       |
| `DELETE` | `/api/v1/order-items/{id}` | Delete order item       |

Full OpenAPI specification: [`docs/api/openapi.yaml`](../api/openapi.yaml)

---

## Repository Layout

```
order-service/
├── cmd/api/                        Application entry point
├── internal/
│   ├── domain/                     Domain layer (entities, repository interfaces)
│   ├── application/                Application layer (commands, queries, handlers, DTOs)
│   └── infrastructure/             Infrastructure layer (persistence, HTTP, config)
├── configs/
│   ├── otel/                       TFO Collector configuration
│   ├── prometheus/                 Prometheus config + recording/alerting rules
│   ├── alertmanager/               Alertmanager routing configuration
│   ├── grafana/                    Dashboards + datasource/dashboard provisioning
│   ├── loki/                       Loki OTLP ingestion + retention config
│   └── jaeger/                     Jaeger sampling strategies
├── docs/
│   ├── api/                        OpenAPI spec + Swagger JSON
│   ├── diagrams/                   ERD, DFD (Mermaid)
│   ├── postman/                    Postman collection + environment
│   └── wiki/                       This documentation
├── tests/
│   ├── unit/                       Unit + property-based tests
│   ├── integration/                Integration + smoke tests
│   └── e2e/                        End-to-end tests
├── docker-compose.yml              Multi-profile service definitions
├── Dockerfile                      Production container image
├── Makefile                        Build, test, and dev commands
└── go.mod / go.sum                 Go module dependencies
```

---

## Getting Started

```bash
# 1. Clone and configure
cp .env.example .env
# Edit .env with your settings

# 2. Start the full stack
docker compose --profile all up -d

# 3. Verify services
curl http://localhost:8080/health
curl http://localhost:9090/-/healthy
curl http://localhost:9093/-/healthy

# 4. Run tests
make test
```

---

## Related Resources

- [Data Flow Diagram](../diagrams/DFD.md) — Request and telemetry data flows
- [Entity Relationship Diagram](../diagrams/ERD.md) — Database schema
- [Postman Collection](../postman/collection.json) — Ready-to-use API requests
- [Changelog](../../CHANGELOG.md) — Version history
