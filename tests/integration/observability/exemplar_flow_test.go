// Package observability provides integration tests for observability configuration.
//
// Validates: R4.1, R4.2, R4.3, R4.4, R4.5
package observability

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// =============================================================================
// Exemplar End-to-End Flow Integration Test
// Validates: R4.1, R4.2, R4.3, R4.4, R4.5
//
// This test verifies the complete exemplar pipeline:
// 1. Send a trace span with a known trace_id to the OTLP collector
// 2. Wait for scrape interval + propagation
// 3. Query Prometheus /api/v1/query_exemplars
// 4. Assert the response contains an exemplar with a 32-char hex trace_id
// =============================================================================

const (
	collectorEndpoint     = "localhost:4317"
	prometheusExemplarURL = "http://localhost:9090"
	exemplarQuery         = `traces_span_metrics_duration_milliseconds_bucket{http_method="POST",http_route="/api/v1/orders"}`
)

// hexTraceIDRegex matches a 32-character lowercase hex string (trace_id format).
var hexTraceIDRegex = regexp.MustCompile(`^[0-9a-f]{32}$`)

// prometheusExemplarResponse represents the response from /api/v1/query_exemplars.
type prometheusExemplarResponse struct {
	Status string         `json:"status"`
	Data   []exemplarData `json:"data"`
}

type exemplarData struct {
	SeriesLabels map[string]string `json:"seriesLabels"`
	Exemplars    []exemplarEntry   `json:"exemplars"`
}

type exemplarEntry struct {
	Labels    map[string]string `json:"labels"`
	Value     string            `json:"value"`
	Timestamp float64           `json:"timestamp"`
}

func TestExemplarEndToEndFlow(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test: INTEGRATION_TEST=true not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Generate a known trace_id
	knownTraceID := generateKnownTraceID(t)
	t.Logf("Using trace_id: %s", knownTraceID)

	// Step 2: Send an OTLP trace span to the collector with the known trace_id
	sendTraceSpan(ctx, t, knownTraceID)

	// Step 3: Wait for scrape interval (10s) + propagation time
	t.Log("Waiting 15 seconds for scrape interval + propagation...")
	select {
	case <-time.After(15 * time.Second):
	case <-ctx.Done():
		t.Fatal("context cancelled while waiting for propagation")
	}

	// Step 4: Query Prometheus /api/v1/query_exemplars
	exemplars := queryExemplars(ctx, t)

	// Step 5: Assert response contains at least one exemplar with a valid 32-char hex trace_id
	assertExemplarWithTraceID(t, exemplars)
}

// generateKnownTraceID generates a deterministic 32-char hex trace ID for this test.
func generateKnownTraceID(t *testing.T) string {
	t.Helper()
	// Use a fixed but unique-per-run trace_id based on current time
	traceIDBytes := [16]byte{}
	now := time.Now().UnixNano()
	// Fill first 8 bytes with timestamp for uniqueness
	for i := 0; i < 8; i++ {
		traceIDBytes[i] = byte(now >> (56 - 8*i))
	}
	// Fill remaining bytes with a recognizable pattern
	traceIDBytes[8] = 0xde
	traceIDBytes[9] = 0xad
	traceIDBytes[10] = 0xbe
	traceIDBytes[11] = 0xef
	traceIDBytes[12] = 0xca
	traceIDBytes[13] = 0xfe
	traceIDBytes[14] = 0xba
	traceIDBytes[15] = 0xbe

	traceID := hex.EncodeToString(traceIDBytes[:])
	require.Len(t, traceID, 32, "trace_id must be exactly 32 hex chars")
	require.Regexp(t, hexTraceIDRegex, traceID)
	return traceID
}

// sendTraceSpan sends a single OTLP trace span to the collector via gRPC.
func sendTraceSpan(ctx context.Context, t *testing.T, traceIDHex string) {
	t.Helper()

	// Create OTLP gRPC exporter
	conn, err := grpc.NewClient(
		collectorEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "failed to create gRPC connection")
	defer func() { _ = conn.Close() }()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn),
	)
	require.NoError(t, err, "failed to create OTLP trace exporter")
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		require.NoError(t, exporter.Shutdown(shutdownCtx))
	}()

	// Create a tracer provider with the exporter
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("order-service"),
			semconv.ServiceVersion("1.0.0-test"),
		),
	)
	require.NoError(t, err, "failed to create resource")

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxExportBatchSize(1),
			sdktrace.WithBatchTimeout(1*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		require.NoError(t, tp.Shutdown(shutdownCtx))
	}()

	// Parse the known trace ID
	traceIDBytes, err := hex.DecodeString(traceIDHex)
	require.NoError(t, err, "failed to decode trace_id hex")
	var traceID trace.TraceID
	copy(traceID[:], traceIDBytes)

	// Create a span with the known trace_id using a remote span context
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, spanCtx)

	// Create a span simulating "POST /api/v1/orders"
	tracer := tp.Tracer("integration-test")
	_, span := tracer.Start(ctx, "POST /api/v1/orders",
		trace.WithAttributes(
			attribute.String("http.method", "POST"),
			attribute.String("http.route", "/api/v1/orders"),
			attribute.Int("http.status_code", 201),
		),
		trace.WithSpanKind(trace.SpanKindServer),
	)

	// Simulate a 100ms duration
	time.Sleep(100 * time.Millisecond)
	span.End()

	// Force flush to ensure the span is exported
	err = tp.ForceFlush(ctx)
	require.NoError(t, err, "failed to flush trace provider")

	t.Log("Trace span sent successfully to collector")
}

// queryExemplars queries the Prometheus /api/v1/query_exemplars endpoint.
func queryExemplars(ctx context.Context, t *testing.T) *prometheusExemplarResponse {
	t.Helper()

	now := time.Now()
	start := now.Add(-60 * time.Second)

	url := fmt.Sprintf(
		"%s/api/v1/query_exemplars?query=%s&start=%s&end=%s",
		prometheusExemplarURL,
		exemplarQuery,
		fmt.Sprintf("%d", start.Unix()),
		fmt.Sprintf("%d", now.Unix()),
	)

	t.Logf("Querying Prometheus exemplars: %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err, "failed to create HTTP request")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "failed to query Prometheus exemplars endpoint")
	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusOK, resp.StatusCode,
		"Prometheus exemplars endpoint returned non-200 status")

	var exemplarResp prometheusExemplarResponse
	err = json.NewDecoder(resp.Body).Decode(&exemplarResp)
	require.NoError(t, err, "failed to decode exemplars response")

	require.Equal(t, "success", exemplarResp.Status,
		"Prometheus exemplars query status is not 'success'")

	t.Logf("Received %d exemplar series from Prometheus", len(exemplarResp.Data))
	return &exemplarResp
}

// assertExemplarWithTraceID asserts that at least one exemplar has a valid 32-char hex trace_id.
func assertExemplarWithTraceID(t *testing.T, resp *prometheusExemplarResponse) {
	t.Helper()

	require.NotEmpty(t, resp.Data,
		"expected at least one exemplar series in the response, got none")

	found := false
	for _, series := range resp.Data {
		for _, exemplar := range series.Exemplars {
			traceID, exists := exemplar.Labels["trace_id"]
			if exists && hexTraceIDRegex.MatchString(traceID) {
				t.Logf("Found exemplar with valid trace_id: %s", traceID)
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	assert.True(t, found,
		"expected at least one exemplar with a 32-char hex trace_id label")
}
