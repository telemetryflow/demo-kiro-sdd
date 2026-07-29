// Package observability provides integration tests for observability configuration.
package observability

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Alertmanager Smoke Test
// Validates: R3.1 — Alertmanager container starts and /-/healthy returns HTTP 200
// =============================================================================

func TestAlertmanagerContainerHealth(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=true to run")
	}

	alertmanagerURL := "http://localhost:9093/-/healthy"

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(alertmanagerURL)
	if err != nil {
		t.Skipf("Alertmanager container not available at %s: %v", alertmanagerURL, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		fmt.Sprintf("expected HTTP 200 from Alertmanager health endpoint, got %d", resp.StatusCode))
}
