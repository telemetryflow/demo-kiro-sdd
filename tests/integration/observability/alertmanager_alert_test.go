// Package observability provides integration tests for observability configuration.
package observability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Alertmanager Alert Integration Test
// Validates: R2.2, R3.4
//
// This test verifies that when the HighP95Latency alert fires in Prometheus,
// it is routed to Alertmanager and received by the default receiver.
//
// Prerequisites:
// - INTEGRATION_TEST=true environment variable must be set
// - Full observability stack running (Prometheus on :9090, Alertmanager on :9093)
// - For alert verification: traffic generating P95 > 500ms for >2 minutes
// =============================================================================

const (
	alertmanagerURL  = "http://localhost:9093"
	alertHTTPTimeout = 10 * time.Second
)

// promAlertsAPIResponse represents the Prometheus /api/v1/alerts response.
type promAlertsAPIResponse struct {
	Status string `json:"status"`
	Data   struct {
		Alerts []promAlertEntry `json:"alerts"`
	} `json:"data"`
}

type promAlertEntry struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    string            `json:"activeAt"`
}

// alertmanagerAlertEntry represents an alert from the Alertmanager /api/v2/alerts response.
type alertmanagerAlertEntry struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Status      struct {
		State string `json:"state"`
	} `json:"status"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
}

func TestAlertmanagerReceivesFiredAlert(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test: set INTEGRATION_TEST=true to run")
	}

	client := &http.Client{Timeout: alertHTTPTimeout}

	// Step 1: Verify Prometheus has the HighP95Latency rule loaded
	t.Run("prometheus_has_alerting_rule_loaded", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/rules", prometheusURL))
		if err != nil {
			t.Skipf("Prometheus not reachable at %s: %v", prometheusURL, err)
		}
		defer resp.Body.Close() //nolint:errcheck

		require.Equal(t, http.StatusOK, resp.StatusCode,
			"Prometheus /api/v1/rules should return 200")

		var rulesResp prometheusRulesResponse
		err = json.NewDecoder(resp.Body).Decode(&rulesResp)
		require.NoError(t, err, "failed to decode Prometheus rules response")
		require.Equal(t, "success", rulesResp.Status)

		// Verify order_service_latency group exists
		found := false
		for _, group := range rulesResp.Data.Groups {
			if group.Name == "order_service_latency" {
				found = true
				break
			}
		}
		assert.True(t, found,
			"Prometheus must have order_service_latency rule group loaded")
	})

	// Step 2: Check if the HighP95Latency alert is currently firing in Prometheus
	t.Run("alert_fires_and_routes_to_alertmanager", func(t *testing.T) {
		// Query Prometheus for active alerts
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/alerts", prometheusURL))
		if err != nil {
			t.Skipf("Prometheus not reachable at %s: %v", prometheusURL, err)
		}
		defer resp.Body.Close() //nolint:errcheck

		require.Equal(t, http.StatusOK, resp.StatusCode,
			"Prometheus /api/v1/alerts should return 200")

		var alertsResp promAlertsAPIResponse
		err = json.NewDecoder(resp.Body).Decode(&alertsResp)
		require.NoError(t, err, "failed to decode Prometheus alerts response")
		require.Equal(t, "success", alertsResp.Status)

		// Find HighP95Latency alert in firing state
		var firingAlert *promAlertEntry
		for i := range alertsResp.Data.Alerts {
			a := &alertsResp.Data.Alerts[i]
			if a.Labels["alertname"] == "HighP95Latency" && a.State == "firing" {
				firingAlert = a
				break
			}
		}

		if firingAlert == nil {
			t.Skip("Skipping Alertmanager verification: HighP95Latency alert is not currently firing. " +
				"To trigger this alert, generate traffic with P95 latency > 500ms on POST /api/v1/orders " +
				"sustained for more than 2 minutes (8 consecutive 15s evaluation cycles).")
		}

		// Alert is firing — verify it was received by Alertmanager
		t.Logf("HighP95Latency alert is firing (state=%s, activeAt=%s)",
			firingAlert.State, firingAlert.ActiveAt)

		// Step 3: Query Alertmanager to verify the alert was received
		amResp, err := client.Get(fmt.Sprintf("%s/api/v2/alerts", alertmanagerURL))
		if err != nil {
			t.Fatalf("Alertmanager not reachable at %s: %v", alertmanagerURL, err)
		}
		defer amResp.Body.Close() //nolint:errcheck

		require.Equal(t, http.StatusOK, amResp.StatusCode,
			"Alertmanager /api/v2/alerts should return 200")

		var amAlerts []alertmanagerAlertEntry
		err = json.NewDecoder(amResp.Body).Decode(&amAlerts)
		require.NoError(t, err, "failed to decode Alertmanager alerts response")

		// Verify at least one alert is the HighP95Latency alert with correct labels
		var matchedAlert *alertmanagerAlertEntry
		for i := range amAlerts {
			a := &amAlerts[i]
			if a.Labels["alertname"] == "HighP95Latency" {
				matchedAlert = a
				break
			}
		}

		require.NotNil(t, matchedAlert,
			"Alertmanager must have received the HighP95Latency alert from Prometheus")

		// Verify severity label (R2.2)
		assert.Equal(t, "warning", matchedAlert.Labels["severity"],
			"HighP95Latency alert must have severity=warning label")

		// Verify routing labels are present (R3.4 - routed to default receiver)
		assert.NotEmpty(t, matchedAlert.Labels["http_method"],
			"HighP95Latency alert must include http_method label for routing")
		assert.NotEmpty(t, matchedAlert.Labels["http_route"],
			"HighP95Latency alert must include http_route label for routing")

		t.Logf("Alertmanager received HighP95Latency alert: severity=%s, http_method=%s, http_route=%s",
			matchedAlert.Labels["severity"],
			matchedAlert.Labels["http_method"],
			matchedAlert.Labels["http_route"])
	})
}
