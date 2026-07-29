package observability

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Prometheus Hot-Reload Smoke Test
// Validates: R6.3, R6.4, R6.5
// =============================================================================

// prometheusRulesResponse represents the JSON response from /api/v1/rules.
type prometheusRulesResponse struct {
	Status string         `json:"status"`
	Data   rulesGroupData `json:"data"`
}

type rulesGroupData struct {
	Groups []rulesGroupEntry `json:"groups"`
}

type rulesGroupEntry struct {
	Name           string `json:"name"`
	File           string `json:"file"`
	LastEvaluation string `json:"lastEvaluation"`
}

func TestPrometheusHotReloadWithRules(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("skipping integration test: INTEGRATION_TEST env var not set to 'true'")
	}

	prometheusURL := "http://localhost:9090"

	t.Run("POST /-/reload returns 200", func(t *testing.T) {
		resp, err := http.Post(prometheusURL+"/-/reload", "", nil)
		require.NoError(t, err, "failed to POST /-/reload to Prometheus")
		defer resp.Body.Close() //nolint:errcheck

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"Prometheus /-/reload should return 200 OK")
	})

	t.Run("GET /api/v1/rules contains order_service_latency group", func(t *testing.T) {
		resp, err := http.Get(prometheusURL + "/api/v1/rules")
		require.NoError(t, err, "failed to GET /api/v1/rules from Prometheus")
		defer resp.Body.Close() //nolint:errcheck

		require.Equal(t, http.StatusOK, resp.StatusCode,
			"Prometheus /api/v1/rules should return 200 OK")

		var rulesResp prometheusRulesResponse
		err = json.NewDecoder(resp.Body).Decode(&rulesResp)
		require.NoError(t, err, "failed to decode /api/v1/rules JSON response")

		assert.Equal(t, "success", rulesResp.Status,
			"Prometheus /api/v1/rules response status must be 'success'")

		// Find the order_service_latency group
		var found bool
		var lastEval string
		for _, group := range rulesResp.Data.Groups {
			if group.Name == "order_service_latency" {
				found = true
				lastEval = group.LastEvaluation
				break
			}
		}

		assert.True(t, found,
			"rule group 'order_service_latency' must appear in /api/v1/rules response")
		assert.NotEmpty(t, lastEval,
			"order_service_latency group must have a non-empty lastEvaluation timestamp")
	})
}
