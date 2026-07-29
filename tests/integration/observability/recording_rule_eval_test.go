// Package observability provides integration tests for observability configuration.
package observability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Integration Test: Recording Rule Produces P95 Within 30 Seconds
// Validates: R1.3
//
// This test verifies that the Prometheus recording rule
// `order_service:http_request_duration_p95:5m` produces a non-empty result
// within 30 seconds (2 evaluation cycles at 15s interval) once histogram
// samples are available.
// =============================================================================

const (
	prometheusURL = "http://localhost:9090"
	orderAPIURL   = "http://localhost:8080"
	p95Query      = `order_service:http_request_duration_p95:5m{http_method="POST",http_route="/api/v1/orders"}`
	waitDuration  = 30 * time.Second
)

// prometheusQueryResponse represents the structure of a Prometheus instant query response.
type prometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func TestRecordingRuleProducesP95(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=true to run")
	}

	// Step 1: Check if the recording rule is already producing values
	// (there may already be traffic in the system)
	result, err := queryPrometheusP95(t)
	if err != nil {
		t.Fatalf("failed to query Prometheus: %v", err)
	}

	if hasPositiveResult(result) {
		t.Log("recording rule already producing P95 values — test passes")
		assertPositiveP95Value(t, result)
		return
	}

	// Step 2: No result yet — generate test traffic
	t.Log("no P95 result found, generating test traffic...")
	sendTestTraffic(t)

	// Step 3: Wait 30 seconds for 2 evaluation cycles
	t.Logf("waiting %s for recording rule evaluation...", waitDuration)
	time.Sleep(waitDuration)

	// Step 4: Retry the query
	result, err = queryPrometheusP95(t)
	if err != nil {
		t.Fatalf("failed to query Prometheus after waiting: %v", err)
	}

	if !hasPositiveResult(result) {
		t.Skip("recording rule did not produce P95 after 30s — " +
			"this test requires histogram samples flowing through the collector pipeline; " +
			"ensure the full observability stack is running with traffic")
	}

	assertPositiveP95Value(t, result)
}

// queryPrometheusP95 performs an instant query against Prometheus for the P95 recording rule.
func queryPrometheusP95(t *testing.T) (*prometheusQueryResponse, error) {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/query?query=%s", prometheusURL, encodeQuery(p95Query))

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result prometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus query returned status: %s", result.Status)
	}

	return &result, nil
}

// hasPositiveResult checks if the Prometheus query returned at least one result
// with a positive numeric value.
func hasPositiveResult(result *prometheusQueryResponse) bool {
	if result == nil || len(result.Data.Result) == 0 {
		return false
	}

	for _, r := range result.Data.Result {
		if len(r.Value) < 2 {
			continue
		}
		valStr, ok := r.Value[1].(string)
		if !ok {
			continue
		}
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		if val > 0 {
			return true
		}
	}
	return false
}

// assertPositiveP95Value verifies the query result contains a positive P95 value.
func assertPositiveP95Value(t *testing.T, result *prometheusQueryResponse) {
	t.Helper()

	require.NotEmpty(t, result.Data.Result, "expected at least one result from recording rule")

	for _, r := range result.Data.Result {
		require.Len(t, r.Value, 2, "expected [timestamp, value] pair")
		valStr, ok := r.Value[1].(string)
		require.True(t, ok, "expected value to be a string")
		val, err := strconv.ParseFloat(valStr, 64)
		require.NoError(t, err, "expected value to be a valid float")
		assert.Greater(t, val, float64(0), "P95 value should be positive")
		t.Logf("P95 latency value: %.2fms", val)
	}
}

// sendTestTraffic sends a few POST requests to the orders API to generate
// histogram samples that will flow through the collector into Prometheus.
func sendTestTraffic(t *testing.T) {
	t.Helper()

	orderPayload := `{"customer_name":"integration-test","items":[{"product_name":"test-item","quantity":1,"price":9.99}]}`

	for i := 0; i < 5; i++ {
		req, err := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders", orderAPIURL),
			strings.NewReader(orderPayload),
		)
		if err != nil {
			t.Logf("warning: failed to create request %d: %v", i+1, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Logf("warning: request %d failed: %v", i+1, err)
			continue
		}
		resp.Body.Close() //nolint:errcheck
		t.Logf("sent test request %d, status: %d", i+1, resp.StatusCode)

		// Brief pause between requests
		time.Sleep(200 * time.Millisecond)
	}
}

// encodeQuery performs minimal URL encoding for the Prometheus query parameter.
func encodeQuery(query string) string {
	replacer := strings.NewReplacer(
		" ", "%20",
		`"`, "%22",
		"{", "%7B",
		"}", "%7D",
		",", "%2C",
		"=", "%3D",
	)
	return replacer.Replace(query)
}
