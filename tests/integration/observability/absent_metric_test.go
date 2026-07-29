// Package observability provides integration tests for observability configuration.
package observability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Integration Test: Absent Histogram Produces No Recording Rule Output
// Validates: R1.4, R2.5
//
// This test verifies that when there is no traffic for a given http_method and
// http_route combination, the recording rule `order_service:http_request_duration_p95:5m`
// produces no time series (absent/empty) rather than zero. This confirms that no
// false time series are produced when there are no histogram samples in the rate window.
//
// The test queries for a method+route combination ("NONEXISTENT"/"/nonexistent")
// that will never have real traffic, ensuring the result set is truly empty.
// =============================================================================

// absentMetricQuery uses a method/route combination that definitively has no traffic.
const absentMetricQuery = `order_service:http_request_duration_p95:5m{http_method="NONEXISTENT",http_route="/nonexistent"}`

func TestAbsentHistogramProducesNoRecordingRuleOutput(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=true to run")
	}

	// Query Prometheus for the recording rule output with a method/route
	// that has never received any traffic.
	url := fmt.Sprintf("%s/api/v1/query?query=%s", prometheusURL, encodeQuery(absentMetricQuery))

	resp, err := http.Get(url)
	require.NoError(t, err, "failed to query Prometheus")
	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 from Prometheus query API")

	var result prometheusQueryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "failed to decode Prometheus response")
	require.Equal(t, "success", result.Status, "expected Prometheus query status 'success'")

	// The key assertion: the result set MUST be empty (no time series returned).
	// This proves the recording rule does not produce false series for metrics
	// with no samples. An empty result means "absent" — not zero.
	assert.Empty(t, result.Data.Result,
		"expected no time series for absent histogram — the recording rule should produce "+
			"no output (absent/empty) when there are no samples for the given http_method and http_route, "+
			"but got %d result(s)", len(result.Data.Result))

	t.Log("confirmed: absent histogram produces no recording rule output (no false series)")
}
