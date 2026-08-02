# Order-Service

<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://github.com/telemetryflow/.github/raw/main/docs/assets/tfo-logo-sdk-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="https://github.com/telemetryflow/.github/raw/main/docs/assets/tfo-logo-sdk-light.svg">
    <img src="https://github.com/telemetryflow/.github/raw/main/docs/assets/tfo-logo-sdk-light.svg" alt="TelemetryFlow Logo" width="80%">
  </picture>

[![Version](https://img.shields.io/badge/Version-1.4.4-orange.svg)](CHANGELOG.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org/)
[![OpenTelemetry](https://img.shields.io/badge/OTLP-100%25%20Compliant-success?logo=opentelemetry)](https://opentelemetry.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://hub.docker.com/r/telemetryflow/telemetryflow-sdk)

</div>

<p align="center">
<strong>[GENERATED TelemetryFlow SDK]</strong> Order-Service - RESTful API with DDD + CQRS Pattern
</p>

---

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.4] - 2026-08-02

### Added

- **Request Validator**: Registered Echo validator on server instance, enabling struct validation on all request handlers
- **Auth Token Endpoint**: Public `POST /api/v1/auth/token` endpoint with auto-generated `user_id` (UUID) — only `email` and `role` required in request body
- **Unified Observability Stack**: Added Grafana, Jaeger, and Loki services under the `monitoring` profile, providing a full metrics + logs + traces visualization layer alongside the TFO Platform
- **Unified Correlation Dashboard**: New Grafana dashboard `order_service_overview` correlating metrics (Prometheus exemplars), traces (Jaeger), and logs (Loki `traceId`) on a single view — RED health stats, p50/p95/p99 latency, top slow spans, service dependency graph, and live correlated logs
- **Grafana Provisioning**: Auto-provisioned datasources (Prometheus, Jaeger, Loki) with cross-signal correlation links (`exemplarTraceIdDestinations`, `tracesToLogsV2`, Loki `derivedFields`) and a dashboard provider
- **Loki OTLP Ingestion**: Loki config with TSDB schema, structured metadata, and OTLP resource attributes promoted to stream labels via `distributor.otlp_config` for `service_name` filtering
- **Collector Fan-out**: TFO-Collector now exports traces → Jaeger, logs → Loki (OTLP native), and span_metrics/service_graph → a Prometheus exporter on `:8889` with exemplars

### Fixed

- **Router Middleware Bleed**: Fixed auth middleware being applied to public routes due to Echo sub-group with empty prefix. Split into independent `v1Public` and `v1Protected` groups
- **Lint Issues**: Resolved all linter warnings (errcheck, staticcheck SA4012, ineffassign) across observability test files
- **Loki Config Parse Error**: Removed invalid `limits_config.otlp_resource_attributes` (rejected by Loki 3.7.x); moved resource-attribute label promotion to the valid `distributor.otlp_config.default_resource_attributes_as_index_labels` field
- **Jaeger Badger Permissions**: `mkdir /badger/key: permission denied` (named volume owned by root, container runs non-root) — switched span storage to in-memory for the demo stack
- **PostgreSQL Major-Version Incompatibility**: PG 18 cannot read a data directory initialized by PG 16; documented the upgrade requirement and wiped the stale dev data directory so PG 18 initializes fresh
- **Loki Healthcheck on Distroless Image**: The `grafana/loki` image has no shell/`wget`/`curl`, so the `CMD-SHELL` healthcheck could never succeed (permanently "unhealthy"). Removed the healthcheck and switched Grafana's Loki dependency to `service_started`
- **Collector Exporter Type Names**: TFO Collector's OCB build registers the OTLP HTTP exporter as `otlp_http` (underscored), not the standard `otlphttp`; corrected the Jaeger and Loki exporter IDs
- **Spanmetric Name Mismatch**: The spanmetrics connector (v0.152.0) emits `traces_calls_total` / `traces_duration_milliseconds_bucket` (no `span_metrics` infix); updated all Grafana dashboards, the Prometheus recording rule, datasource links, docs, and tests from the obsolete `traces_span_metrics_*` names
- **Port Collision (API vs TFO-Viz)**: Both the API and `tfo-viz` defaulted to host port 8080; moved `PORT_FRONTEND` to 8088
- **Duplicate Container Name**: `CONTAINER_TFO_POSTGRES` was set to the same name as the app postgres; gave the platform postgres a distinct container name

### Changed

- **OpenAPI Spec**: Added Auth tag and `/api/v1/auth/token` endpoint with `TokenRequest`, `TokenResponse`, `TokenSuccessResponse` schemas
- **Swagger JSON**: Synchronized with OpenAPI spec changes
- **Postman Collection**: Replaced legacy "Login" request with "Generate Token" pointing to correct endpoint
- **Postman Environment**: Replaced `testUserEmail`/`testUserPassword` with `tokenEmail`/`tokenRole` variables
- **Docker Compose Docs**: Added rebuild command (`--build api`) to wiki and README
- **Cache Store**: Replaced Redis with Valkey (`valkey/valkey:8-alpine`), a Redis-compatible drop-in. Service renamed `tfo-demo-redis` → `tfo-demo-valkey`; the `tfo-backend` image still receives `REDIS_*` env-var keys (app contract) sourced from `VALKEY_*` values
- **Container Versions**: Bumped all third-party images to current latest
  - postgres `16-alpine` → `18-alpine`
  - prometheus `v3.4.0` → `v3.13.2`
  - alertmanager `v0.28.1` → `v0.33.1` (externalized to `ALERTMANAGER_VERSION`)
  - loki `3.3.2` → `3.7.4`
  - jaeger `1.62.0` → `1.76.0`
  - grafana `11.4.0` → `13.1.1`
  - clickhouse `24.12-alpine` → `26.7-alpine`
  - nats `2-alpine` → `2.14-alpine`
- **Prometheus Scrape**: Fixed scrape targets to use the `tfo-collector` DNS name (previously referenced the non-existent `otel-collector` host) and documented the exemplar bridge
- **Documentation**: Rewrote Grafana-Dashboards wiki for the unified stack; updated Architecture, Docker-Compose, Getting-Started, Observability, Home, DFD, and README to reflect Valkey and the new services/versions

---

## [1.2.0] - 2026-06-23

### Added

- **Dependabot Configuration**: Added `.github/dependabot.yml` for automated dependency updates (gomod, docker, github-actions ecosystems)
- **TFO Platform Profile**: Added `platform` profile to docker-compose with TFO-Backend, TFO-Viz, and infrastructure services (PostgreSQL, ClickHouse, Redis, NATS) for end-to-end local observability

### Changed

- **Go Version**: Upgraded from Go 1.24 to Go 1.26
  - `go.mod`: `go 1.24.0` → `go 1.26.0`, `toolchain go1.24.11` → `toolchain go1.26.3`
  - `Dockerfile`: `golang:1.24-alpine` → `golang:1.26-alpine`
- **TelemetryFlow Go SDK**: Upgraded from v1.1.2 to v1.2.0
- **OpenTelemetry SDK**: Upgraded from v1.39.0 to v1.43.0
- **TFO-Collector**: Updated from v1.1.2 to v1.2.1 in docker-compose
- **GitHub Actions CI/CD**: Upgraded all workflow action versions
  - `actions/checkout` v4 → v6
  - `actions/setup-go` v5 → v6
  - `actions/upload-artifact` v4 → v7
  - `actions/download-artifact` v4 → v8
  - `docker/metadata-action` v5 → v6
  - `docker/setup-qemu-action` v3 → v4
  - `docker/setup-buildx-action` v3 → v4
  - `docker/login-action` v3 → v4
  - `docker/build-push-action` v6 → v7
  - `softprops/action-gh-release` v2 → v3
  - `golangci/golangci-lint-action` v7 → v9
- **Docker Compose**: Removed Grafana and Jaeger services; telemetry visualization now handled by TFO-Viz (via `platform` profile)
- **golangci-lint**: Added `-ST1000` to staticcheck exclusions (package comments present but not recognized through license header blocks)
- **File Headers**: Refactored all 65 Go source files to standard Apache 2.0 license header format
- **Version**: Updated from 1.1.2 to 1.2.0

### Removed

- **Grafana**: Removed from docker-compose (replaced by TFO-Viz via `platform` profile)
- **Jaeger**: Removed from docker-compose (replaced by TFO-Backend distributed tracing via `platform` profile)
