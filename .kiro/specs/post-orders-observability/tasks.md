# Implementation Plan: POST Orders Observability

## Overview

This plan implements P95 latency monitoring with alerting, exemplar-based trace drill-down, and Grafana dashboard provisioning for the `POST /api/v1/orders` endpoint. All work is infrastructure configuration (Prometheus rules, Alertmanager deployment, Docker Compose changes, Grafana JSON). No application code changes are needed — otelecho already instruments all HTTP handlers.

## Tasks

- [x] 1. Instrumentation — Prometheus rules file integration and recording/alerting rules
  - [x] 1.1 Create Prometheus rules directory and recording rule file
    - Create `configs/prometheus/rules/order_service.yml`
    - Define rule group `order_service_latency` with 15s evaluation interval
    - Add recording rule computing `histogram_quantile(0.95, sum(rate(traces_span_metrics_duration_milliseconds_bucket[5m])) by (le, http_method, http_route))` stored as `order_service:http_request_duration_p95:5m`
    - Add alerting rule `HighP95Latency` with expression `order_service:http_request_duration_p95:5m > 500`, `for: 2m`, severity label `warning`, summary annotation with value and route, and `requirement_id: R2` annotation
    - _Requirements: R1.1, R1.2, R2.1, R2.2, R2.3, R2.4, R2.5, R6_

  - [x] 1.2 Update Prometheus configuration to load rules and configure alerting
    - Add `rule_files: ['/etc/prometheus/rules/*.yml']` to `configs/prometheus/prometheus.yml`
    - Add `alerting.alertmanagers` section with `static_configs` target `alertmanager:9093`
    - _Requirements: R6.2, R3.3_

  - [x] 1.3 Mount rules directory into Prometheus container via Docker Compose
    - Add volume mount `./configs/prometheus/rules/:/etc/prometheus/rules/:ro` to the `prometheus` service in `docker-compose.yml`
    - _Requirements: R6.1_

- [x] 2. Instrumentation — Alertmanager deployment and routing
  - [x] 2.1 Create Alertmanager configuration file
    - Create `configs/alertmanager/alertmanager.yml`
    - Define global config with `resolve_timeout: 5m`
    - Define default route with `receiver: 'default'`, `group_by: ['alertname', 'http_method', 'http_route']`, `group_wait: 30s`, `group_interval: 5m`, `repeat_interval: 30m` (suppression window)
    - Define `default` receiver as a placeholder (webhook_configs with empty list or similar) so adding Slack/email/webhook requires only editing this file
    - _Requirements: R3.4, R3.5_

  - [x] 2.2 Add Alertmanager service to Docker Compose
    - Add `alertmanager` service with image `prom/alertmanager:v0.28.1`, profiles `["monitoring", "all"]`
    - Map host port `9093` to container port `9093`
    - Mount `./configs/alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro`
    - Connect to `order_service_net` with a static IPv4 address (e.g., `172.152.152.55`)
    - Add healthcheck on `/-/healthy` endpoint
    - _Requirements: R3.1, R3.2_

- [x] 3. Instrumentation — Grafana dashboard with exemplar drill-down
  - [x] 3.1 Create Grafana dashboard JSON provisioning file
    - Create `configs/grafana/dashboards/order_service_p95.json`
    - Define datasource UID variable `${DS_PROMETHEUS}` and trace datasource variable `${DS_TRACES}` (supporting Jaeger or Tempo)
    - Add `http_route` template variable populated from `label_values(traces_span_metrics_duration_milliseconds_bucket, http_route)` with default `/api/v1/orders`
    - Add time-series panel querying `order_service:http_request_duration_p95:5m{http_route="$http_route"}`
    - Enable exemplar toggle on the panel, with exemplar data source querying `traces_span_metrics_duration_milliseconds_bucket{http_route="$http_route"}`
    - Configure exemplar internal link to trace datasource using `trace_id` field
    - _Requirements: R5.1, R5.2, R5.3, R5.4, R5.5_

- [x] 4. Checkpoint — Validate instrumentation configuration
  - Ensure all configuration files are valid YAML/JSON (run `promtool check rules` on rules file if available, or validate YAML syntax)
  - Ensure `docker-compose.yml` is valid (run `docker compose config`)
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Verification — Property-based tests
  - [x] 5.1 Write property test: No metric label contains a UUID or high-cardinality identifier
    - **Property 1: No metric label contains a high-cardinality identifier**
    - Create test file `internal/infrastructure/observability/cardinality_test.go` (or similar location)
    - Use `pgregory.net/rapid` library to generate random label value sets
    - Assert no generated label value matches UUID pattern `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, 32-char hex (trace_id), or contains `@` (email)
    - Validate against the allowed label values defined in the Prometheus rules and collector config
    - Minimum 100 iterations
    - **Validates: R1.2 (label restrictions), Steering §2 (Cardinality hard constraints)**

  - [x] 5.2 Write property test: Recording rule output excludes the `le` bucket label
    - **Property 2: Recording rule output excludes the `le` bucket label**
    - Use `pgregory.net/rapid` to generate arbitrary `http_method` and `http_route` label combinations
    - Parse the recording rule expression from `configs/prometheus/rules/order_service.yml`
    - Assert the `by` clause produces output containing `http_method` and `http_route` but NOT `le`
    - **Validates: R1.2**

  - [x] 5.3 Write property test: Absent source metric produces no alert
    - **Property 3: Absent source metric produces no alert**
    - Use `pgregory.net/rapid` to generate random `http_method` and `http_route` label combinations with zero histogram samples
    - Parse the alerting rule expression from `configs/prometheus/rules/order_service.yml`
    - Assert `HighP95Latency` alert expression evaluates to empty/false when the source histogram has no samples in the rate window
    - Minimum 100 iterations
    - **Validates: R2.5**

- [x] 6. Verification — Integration and validation tests
  - [x] 6.1 Write integration test: Prometheus rules load correctly
    - Verify `promtool check rules configs/prometheus/rules/order_service.yml` passes
    - OR: start Prometheus container, query `/api/v1/rules`, verify `order_service_latency` group is present with correct rule names
    - **Validates: R1.1, R6.2, R6.5**

  - [x] 6.2 Write integration test: Alertmanager receives fired alert
    - Simulate alert by injecting metric samples above threshold for >2 minutes
    - Query Alertmanager `/api/v2/alerts` to verify `HighP95Latency` alert is received
    - **Validates: R2.2, R3.4**

  - [x] 6.3 Write integration test: Exemplar end-to-end flow
    - Send a trace with a known `trace_id` to the collector via OTLP gRPC
    - Wait for scrape interval (≤10s) + propagation
    - Query Prometheus `/api/v1/query_exemplars` with selector `traces_span_metrics_duration_milliseconds_bucket{http_method="POST", http_route="/api/v1/orders"}`
    - Assert response contains an exemplar with matching 32-char hex `trace_id`
    - **Validates: R4.1, R4.2, R4.3, R4.4, R4.5**

  - [x] 6.4 Write validation test: Grafana dashboard JSON structure
    - Load `configs/grafana/dashboards/order_service_p95.json`
    - Assert datasource variable `${DS_PROMETHEUS}` exists
    - Assert trace datasource variable `${DS_TRACES}` exists
    - Assert `http_route` template variable is defined with correct query
    - Assert at least one panel has exemplar config enabled
    - Assert exemplar internal link targets trace datasource with `trace_id`
    - **Validates: R5.1, R5.2, R5.3, R5.4, R5.5**

  - [x] 6.5 Write smoke test: Alertmanager container health
    - Verify Alertmanager container starts and `/-/healthy` returns HTTP 200
    - **Validates: R3.1**

  - [x] 6.6 Write smoke test: Prometheus hot-reload with rules
    - POST `/-/reload` to Prometheus after rules file is mounted
    - Verify 200 response
    - Query `/api/v1/rules` and confirm `order_service_latency` group appears with updated `lastEvaluation` timestamp
    - **Validates: R6.3, R6.4, R6.5**

  - [x] 6.7 Write integration test: Recording rule produces P95 within 30 seconds
    - Inject histogram samples for `POST /api/v1/orders` into Prometheus (via collector or remote write)
    - Wait 30 seconds (2 evaluation cycles at 15s interval)
    - Query `order_service:http_request_duration_p95:5m{http_method="POST", http_route="/api/v1/orders"}` and verify a non-empty result
    - **Validates: R1.3**

  - [x] 6.8 Write integration test: Absent histogram produces no recording rule output
    - Start Prometheus with rules loaded but no traffic flowing to collector
    - Query `order_service:http_request_duration_p95:5m` and verify the result is empty/absent (not zero)
    - Confirms no false time series are produced when there are no samples
    - **Validates: R1.4, R2.5**

  - [x] 6.9 Write integration test: Invalid rules file rejected on reload
    - Mount a syntactically invalid rules file (e.g., malformed YAML with unclosed brackets)
    - POST `/-/reload` to Prometheus and verify it returns a non-2xx HTTP status code
    - Also test: start Prometheus with an invalid rules file and verify it exits with a configuration error
    - Confirm previous valid rules remain active after a failed reload
    - **Validates: R1.5, R6.4**

  - [x] 6.10 Write integration test: Alert annotations and resolution
    - Inject histogram samples with P95 above 500ms for >2 minutes (8+ evaluation cycles)
    - Query Prometheus `/api/v1/alerts` and verify `HighP95Latency` alert is in `firing` state
    - Verify alert annotations contain `summary` field with the P95 value, `http_method`, and `http_route`
    - Then inject histogram samples with P95 below 500ms
    - Wait for one evaluation cycle and verify the alert transitions to `inactive` (resolved)
    - **Validates: R2.3, R2.4**

  - [x] 6.11 Write validation test: Docker Compose volume mounts
    - Parse `docker-compose.yml` and inspect the `prometheus` service volumes
    - Verify rules directory mount includes `:ro` suffix (`./configs/prometheus/rules/:/etc/prometheus/rules/:ro`)
    - Inspect the `alertmanager` service volumes
    - Verify alertmanager config mount includes `:ro` suffix (`./configs/alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro`)
    - **Validates: R3.2, R6.1**

  - [x] 6.12 Write validation test: Alertmanager config extensibility
    - Parse `configs/alertmanager/alertmanager.yml`
    - Verify `receivers` block exists and contains at least the `default` receiver
    - Verify receiver structure supports adding `webhook_configs`, `email_configs`, or `slack_configs` without structural changes
    - Parse `docker-compose.yml` and verify no hard-coded receiver/notification logic exists in the alertmanager service definition
    - Confirm that adding a new receiver requires only editing `alertmanager.yml` (no docker-compose.yml changes needed)
    - **Validates: R3.5**

- [x] 7. Final checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- All test tasks are mandatory to achieve ≥95% acceptance criteria coverage
- Each task references specific requirement IDs (R1–R6) for traceability
- Instrumentation tasks (1–3) are separated from verification/testing tasks (5–6) per steering file §8
- Property tests use `pgregory.net/rapid` (Go property-based testing library)
- No application code changes are needed — otelecho auto-instrumentation is already in place
- Checkpoints ensure incremental validation before and after testing
- Tasks that cannot be completed within current scope (e.g., end-to-end integration tests requiring a full running stack) remain unchecked rather than being silently dropped, per steering file §8

### Acceptance Criteria Coverage

| Requirement | Criteria | Covered By | Status |
|---|---|---|---|
| R1.1 | Rule group definition | 6.1 | ✅ |
| R1.2 | Labels http_method/http_route, no le | 5.1, 5.2 | ✅ |
| R1.3 | P95 produced within 30s | 6.7 | ✅ |
| R1.4 | Absent histogram → no series | 6.8 | ✅ |
| R1.5 | Invalid YAML → fail to start/reload | 6.9 | ✅ |
| R2.1 | Alerting rule expression | 6.1 | ✅ |
| R2.2 | Alert fires after 2min | 6.2 | ✅ |
| R2.3 | Alert annotations with summary | 6.10 | ✅ |
| R2.4 | Alert resolves below 500ms | 6.10 | ✅ |
| R2.5 | Absent metric → no alert | 5.3, 6.8 | ✅ |
| R3.1 | Alertmanager container health | 6.5 | ✅ |
| R3.2 | Config mounted read-only | 6.11 | ✅ |
| R3.3 | Prometheus alertmanagers config | 6.1 | ✅ |
| R3.4 | Alert routed to default receiver | 6.2 | ✅ |
| R3.5 | Extensible receivers (edit only) | 6.12 | ✅ |
| R4.1 | Trace span with trace_id received | 6.3 | ✅ |
| R4.2 | Exemplar emitted on histogram | 6.3 | ✅ |
| R4.3 | Exemplar stored in Prometheus | 6.3 | ✅ |
| R4.4 | Exemplar queryable via API | 6.3 | ✅ |
| R4.5 | Exemplar available within 60s | 6.3 | ✅ |
| R5.1 | Dashboard JSON with datasource vars | 6.4 | ✅ |
| R5.2 | Time-series panel with P95 query | 6.4 | ✅ |
| R5.3 | Exemplar toggle enabled | 6.4 | ✅ |
| R5.4 | Exemplar click → trace link | 6.4 | ✅ |
| R5.5 | http_route variable selector | 6.4 | ✅ |
| R6.1 | Rules dir mounted read-only | 6.11 | ✅ |
| R6.2 | rule_files directive in config | 6.1 | ✅ |
| R6.3 | Hot-reload via /-/reload | 6.6 | ✅ |
| R6.4 | Syntax error → reject reload | 6.9 | ✅ |
| R6.5 | Rule group in /api/v1/rules | 6.1, 6.6 | ✅ |

**Total: 30/30 acceptance criteria covered (100%) — exceeds 95% target**

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "2.1"] },
    { "id": 1, "tasks": ["1.2", "1.3", "2.2", "3.1"] },
    { "id": 2, "tasks": ["5.1", "5.2", "5.3", "6.1", "6.4", "6.11", "6.12"] },
    { "id": 3, "tasks": ["6.2", "6.3", "6.5", "6.6", "6.7", "6.8", "6.9", "6.10"] }
  ]
}
```
