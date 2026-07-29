---
inclusion: always
---

# Observability Standards — TelemetryFlow

These rules apply to every spec, design, and task that touches telemetry.
Apply them without being asked. If a user request conflicts with a rule below,
flag the conflict explicitly rather than silently overriding it.

---

## 1. Naming

- Use OpenTelemetry semantic conventions. Never invent a metric name when a
  standard one exists.
- Latency histogram: `http.server.request.duration`, unit **seconds**.
- Request counter: `http.server.request.count`.
- Resource attributes are mandatory on every signal:
  `service.name`, `service.version`, `deployment.environment`.
- Metric names use dots, not underscores. Lowercase only.

## 2. Cardinality — hard constraints

**Allowed metric labels (allowlist — nothing else):**
`http.route`, `http.request.method`, `http.response.status_code`, `service.name`

**Forbidden as metric labels, without exception:**
`order_id`, `customer_id`, `user_id`, `session_id`, `trace_id`, `span_id`,
`request_id`, email addresses, IP addresses, free-text fields, timestamps,
and any UUID-shaped value.

**Rules:**

- High-cardinality values MAY be span attributes. They must NEVER become metric labels.
- Every design document MUST include a cardinality budget: the multiplication of
  each allowed label's expected distinct values, and the resulting series count.
- Any metric whose projected series count exceeds **1,000** requires an explicit
  justification in the design document.
- `http.route` means the route template (`/orders/{id}`), never the resolved path
  (`/orders/a3f9-...`). Resolved paths are unbounded.

## 3. Alerting

Every alert rule MUST specify all five of these. An alert missing any one is incomplete:

| Field                | Requirement                                    |
| -------------------- | ---------------------------------------------- |
| Threshold            | A concrete number with a unit                  |
| Sustained duration   | Minimum evaluation window before firing        |
| Severity             | `CRITICAL`, `WARNING`, or `INFO`               |
| Suppression window   | Default **30 minutes** unless stated otherwise |
| Resolution condition | How and when the alert clears                  |

**Rules:**

- No alert fires from a single sample. Require a minimum sample count.
- Every alert must be actionable: if the on-call engineer cannot do anything
  about it at 3am, it is a dashboard panel, not an alert.
- Each alert must carry a `summary` annotation and a reference to the
  requirement ID that produced it.

## 4. Requirements format

- Acceptance criteria use **EARS** notation:
  `WHEN <trigger> [FOR <duration>] THEN the system SHALL <behaviour>`
- Use `WHERE` for optional or conditional behaviour.
- Use `SHALL NOT` for explicit prohibitions.
- Every requirement has a stable ID (`R1`, `R2`, ...) and a one-line user story.
- Every acceptance criterion must be **testable**: it contains a number, a
  condition, or an observable state — never a subjective adjective.

**Reject phrasing like:** "the system should be fast", "reasonable latency",
"appropriate alerting". Replace with measurable values.

## 5. Requirements vs. design — keep the boundary

- **Requirements** describe behaviour and limits. No technology names.
  No ClickHouse, no OTLP, no histogram buckets, no query language.
- **Design** is where technology appears: schema, bucket bounds, TFQL queries,
  storage layout, retention.
- If a requirement mentions a specific database or protocol, move it to design.

## 6. Export and instrumentation

- Export via standard OTLP using `TELEMETRYFLOW_ENDPOINT`
  (gRPC `:4317`, HTTP `:4318`). Do not introduce a vendor-specific SDK.
- Configure through standard `OTEL_*` environment variables where they exist.
- Enable exemplars on latency histograms so metric data points carry the
  `trace_id` of a sampled request.
- Instrumentation overhead target: **< 3% p95 latency**. State this as a
  non-functional requirement and leave it verifiable.

## 7. Design document must include

Every observability design document contains, at minimum:

1. **Metric schema table** — name, type, unit, allowed labels, forbidden labels
2. **Cardinality budget** — the arithmetic, and the resulting series count
3. **Histogram bucket bounds** — chosen so alert thresholds land on a boundary
4. **TFQL query templates** — one per dashboard panel, one per alert evaluation
5. **Retention policy** — per signal type
6. **Alert rule definitions** — all five fields from section 3

## 8. Tasks

- Every task references the requirement ID it satisfies.
- Separate instrumentation tasks from verification tasks.
- Include at least one property-based test asserting that no metric label
  matches a UUID pattern.
- Tasks that cannot be completed within the current scope stay **unchecked**
  rather than being silently dropped.

## 9. Retention defaults

| Signal     | Retention |
| ---------- | --------- |
| Metrics    | 90 days   |
| Traces     | 30 days   |
| Logs       | 30 days   |
| Exemplars  | 7 days    |
| Audit logs | 365 days  |

## 10. Privacy

- Filter attributes at the collector, before storage — not after.
- Personally identifiable data must not appear in metric labels under any
  circumstance. In span attributes it requires explicit justification in design.
