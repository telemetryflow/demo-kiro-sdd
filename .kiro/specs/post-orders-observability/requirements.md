# Requirements Document

## Introduction

Observability enhancement for the POST /api/v1/orders endpoint (extensible to all endpoints). This feature adds P95 latency monitoring with alerting and exemplar-based trace linkage. The existing infrastructure already provides: span_metrics connector with exemplars enabled, Prometheus with exemplar-storage and native-histograms features, and otelecho auto-instrumentation. This spec covers the missing pieces: Prometheus alerting rules, Alertmanager deployment, end-to-end exemplar verification, and Grafana dashboard/query examples for P95 with exemplar drill-down.

## Glossary

- **Collector**: The TFO Collector (OCB-based OpenTelemetry Collector) deployed as `tfo-collector` in Docker Compose, responsible for receiving traces and deriving span metrics with exemplars.
- **Prometheus**: The Prometheus v3.4.0 instance configured with `--enable-feature=exemplar-storage` and `--enable-feature=native-histograms`, scraping the Collector at port 8889.
- **Alertmanager**: The Prometheus Alertmanager component responsible for receiving firing alerts from Prometheus and routing notifications to configured receivers.
- **Span_Metrics**: The `span_metrics` connector in the Collector that derives histogram metrics (with exemplars containing trace_id) from incoming trace spans.
- **Exemplar**: A sample data point attached to a histogram bucket containing a `trace_id` label, enabling navigation from a metric observation to the originating distributed trace.
- **P95_Latency**: The 95th percentile of request duration, computed via `histogram_quantile(0.95, ...)` over the `traces_span_metrics_duration_milliseconds` histogram.
- **Alert_Rule**: A Prometheus recording or alerting rule defined in a YAML rules file, evaluated at a configured interval.
- **Grafana**: The visualization platform used to display P95 latency dashboards with exemplar overlay and trace drill-down links.
- **Order_Service**: The Go REST API application (`api` container) instrumented with otelecho middleware, exporting traces via OTLP gRPC to the Collector.

## Requirements

### Requirement 1: P95 Latency Recording Rule

**User Story:** As an SRE, I want a pre-computed P95 latency metric per endpoint, so that I can query endpoint performance without computing quantiles on every dashboard load.

#### Acceptance Criteria

1. THE Prometheus SHALL load a recording rule file that defines a rule group named `order_service_latency` with an evaluation interval of 15 seconds, containing a rule that computes `histogram_quantile(0.95, sum(rate(traces_span_metrics_duration_milliseconds_bucket[5m])) by (le, http_method, http_route))` and stores the result as `order_service:http_request_duration_p95:5m`.
2. WHEN the recording rule is evaluated, THE Prometheus SHALL produce a time series labeled with `http_method` and `http_route` dimensions, excluding the `le` label from the output series.
3. WHEN the endpoint `POST /api/v1/orders` receives traffic, THE Prometheus SHALL produce a P95 latency value in milliseconds for the label set `{http_method="POST", http_route="/api/v1/orders"}` within 30 seconds of the first matching sample being scraped (i.e., no more than 2 evaluation cycles at the 15-second evaluation interval).
4. IF the source histogram `traces_span_metrics_duration_milliseconds_bucket` contains no samples for a given `http_method` and `http_route` combination within the 5-minute rate window, THEN THE Prometheus SHALL produce no time series for that label set (the recording rule outputs NaN or absent series rather than zero).
5. IF the Prometheus configuration references the recording rules file but the file is not mounted or contains invalid YAML, THEN THE Prometheus SHALL fail to start or reload, reporting a configuration validation error in its logs.

### Requirement 2: P95 Latency Alerting Rule

**User Story:** As an SRE, I want to be alerted when P95 latency exceeds 500ms for a sustained period, so that I can investigate performance degradation before users are impacted.

#### Acceptance Criteria

1. THE Prometheus SHALL evaluate an alerting rule named `HighP95Latency` whose expression compares the recorded metric `order_service:http_request_duration_p95:5m` against a threshold of 500 (milliseconds), firing for any label combination where the value exceeds the threshold.
2. WHEN the `HighP95Latency` condition is true for 2 minutes continuously (8 consecutive evaluations at 15-second intervals), THE Prometheus SHALL transition the alert state from `pending` to `firing`.
3. WHEN the `HighP95Latency` alert fires, THE Prometheus SHALL include `http_method` and `http_route` as alert labels (available for Alertmanager routing and grouping) and include a `summary` annotation containing the observed P95 value and the affected method/route combination.
4. WHEN the `HighP95Latency` condition evaluates to false (P95 latency at or below 500 milliseconds) on a subsequent evaluation cycle, THE Prometheus SHALL resolve the `HighP95Latency` alert for the affected label set.
5. IF the metric `order_service:http_request_duration_p95:5m` is absent for a given label set (no traffic), THEN THE Prometheus SHALL treat the alerting expression as not satisfied and the alert SHALL remain inactive for that label set.

### Requirement 3: Alertmanager Deployment and Routing

**User Story:** As an SRE, I want alert notifications routed to a configurable receiver, so that the team is informed of latency degradation through the appropriate channel.

#### Acceptance Criteria

1. THE Docker_Compose SHALL deploy an Alertmanager container with profiles `["monitoring", "all"]`, accessible on host port 9093, connected to the `order_service_net` network with a static IPv4 address, and with a healthcheck verifying the `/-/healthy` endpoint.
2. THE Docker_Compose SHALL mount the Alertmanager configuration from `configs/alertmanager/alertmanager.yml` into the container as a read-only volume.
3. THE Prometheus configuration SHALL include an `alerting.alertmanagers` section with a `static_configs` target specifying the Alertmanager container hostname and port 9093.
4. WHEN the `HighP95Latency` alert fires, THE Alertmanager SHALL route the notification to the default receiver defined in the `route.receiver` field of the Alertmanager configuration file.
5. THE Alertmanager configuration file SHALL define receiver blocks such that adding a webhook, email, or Slack receiver requires only editing `configs/alertmanager/alertmanager.yml` and reloading, without modifying the Docker Compose definition or restarting the container.

### Requirement 4: Exemplar End-to-End Flow Verification

**User Story:** As a developer, I want to verify that exemplars containing trace_id flow from the application span through the Collector into Prometheus, so that I can trust the metric-to-trace linkage.

#### Acceptance Criteria

1. WHEN the Order_Service processes a POST /api/v1/orders request, THE Collector SHALL receive a trace span with a `trace_id` formatted as a 32-character lowercase hexadecimal string and span attributes `http.method=POST` and `http.route=/api/v1/orders`.
2. WHEN the Span_Metrics connector processes the span, THE Collector SHALL emit a histogram data point for `traces_span_metrics_duration_milliseconds` on the Prometheus exporter endpoint (port 8889) with an exemplar containing a `trace_id` label whose value matches the originating span's `trace_id`.
3. WHEN Prometheus scrapes the Collector metrics endpoint at its configured 10-second scrape interval, THE Prometheus SHALL store the exemplar with its `trace_id` value alongside the histogram bucket observation such that the exemplar is queryable via the Prometheus API.
4. WHEN querying the Prometheus `/api/v1/query_exemplars` endpoint with the series selector `{__name__="traces_span_metrics_duration_milliseconds_bucket"}` and a time range covering the request, THE Prometheus API SHALL return at least one exemplar object containing a `trace_id` label with a 32-character hexadecimal value and a numeric timestamp in Unix epoch seconds.
5. WHEN a POST /api/v1/orders request is completed, THE exemplar containing its `trace_id` SHALL be queryable from the Prometheus API within 60 seconds of the request completion.

### Requirement 5: Grafana Dashboard with Exemplar Drill-Down

**User Story:** As a developer, I want a Grafana dashboard showing P95 latency with exemplar markers, so that I can click an exemplar dot to jump directly to the corresponding trace.

#### Acceptance Criteria

1. THE Grafana dashboard SHALL be provided as a JSON provisioning file that references the Prometheus datasource by a UID variable and defines a configurable trace datasource variable (supporting Jaeger or Tempo).
2. THE Grafana SHALL display a time-series panel showing `order_service:http_request_duration_p95:5m` filtered by the selected `http_route` variable, with a default value of `/api/v1/orders`.
3. THE Grafana panel SHALL enable the exemplar toggle and query exemplar data from the raw histogram metric `traces_span_metrics_duration_milliseconds_bucket` (not the recording rule), rendering exemplar data points as overlay dots on the time-series graph.
4. WHEN a user clicks an exemplar dot, THE Grafana SHALL construct a link to the trace view using the `trace_id` value from the exemplar, targeting the trace datasource variable selected in the dashboard (Jaeger or Tempo).
5. THE Grafana dashboard SHALL provide an `http_route` variable selector populated by querying distinct `http_route` label values from the `traces_span_metrics_duration_milliseconds_bucket` metric, so that users can view P95 latency for any instrumented endpoint.

### Requirement 6: Prometheus Rules File Integration

**User Story:** As a DevOps engineer, I want alerting and recording rules loaded from a dedicated rules file mounted into Prometheus, so that rules are version-controlled and independently deployable.

#### Acceptance Criteria

1. THE Docker_Compose SHALL mount the host path `configs/prometheus/rules/` into the Prometheus container at `/etc/prometheus/rules/` as a read-only bind mount (`:ro`), using profiles `["monitoring", "all"]`.
2. THE Prometheus configuration SHALL include a `rule_files` directive referencing `/etc/prometheus/rules/*.yml`.
3. WHEN Prometheus receives an HTTP POST to `/-/reload` after a rules file modification, THE Prometheus SHALL load the updated rules within 10 seconds and reflect the changes in the `/api/v1/rules` endpoint without container restart.
4. IF the rules file contains a syntax error, THEN THE Prometheus SHALL reject the reload, log an error message indicating the parse failure, continue operating with the previously loaded rules, and report the failure via the `/-/reload` HTTP response with a non-2xx status code.
5. WHEN a valid rules file is loaded successfully, THE Prometheus SHALL expose the rule group in the `/api/v1/rules` API response with a `lastEvaluation` timestamp that updates on each evaluation cycle.
