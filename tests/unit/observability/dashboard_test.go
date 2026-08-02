package observability

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GrafanaDashboard represents the top-level structure of a Grafana dashboard JSON.
type GrafanaDashboard struct {
	Inputs     []DashboardInput `json:"__inputs"`
	Templating struct {
		List []TemplateVar `json:"list"`
	} `json:"templating"`
	Panels []Panel `json:"panels"`
}

// DashboardInput represents an input variable declaration in __inputs.
type DashboardInput struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Type       string `json:"type"`
	PluginID   string `json:"pluginId"`
	PluginName string `json:"pluginName"`
}

// TemplateVar represents a templating variable in templating.list.
type TemplateVar struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Query      interface{} `json:"query"`
	Definition string      `json:"definition"`
}

// Panel represents a Grafana panel with targets and field config.
type Panel struct {
	Targets     []PanelTarget `json:"targets"`
	FieldConfig struct {
		Defaults struct {
			Links []DataLink `json:"links"`
		} `json:"defaults"`
	} `json:"fieldConfig"`
}

// PanelTarget represents a query target within a panel.
type PanelTarget struct {
	Exemplar bool `json:"exemplar"`
}

// DataLink represents a data link or internal link on a panel.
type DataLink struct {
	Datasource interface{} `json:"datasource"`
	Field      string      `json:"field"`
	Internal   bool        `json:"internal"`
	Title      string      `json:"title"`
	Type       string      `json:"type"`
}

func loadDashboard(t *testing.T) GrafanaDashboard {
	t.Helper()

	data, err := os.ReadFile("../../../configs/grafana/dashboards/order_service_p95.json")
	require.NoError(t, err, "failed to read dashboard JSON file")

	var dashboard GrafanaDashboard
	err = json.Unmarshal(data, &dashboard)
	require.NoError(t, err, "failed to parse dashboard JSON")

	return dashboard
}

// TestDashboardDatasourceVariablePrometheus validates that the dashboard defines
// a DS_PROMETHEUS datasource variable in __inputs or templating.list.
// Validates: R5.1
func TestDashboardDatasourceVariablePrometheus(t *testing.T) {
	dashboard := loadDashboard(t)

	// Check __inputs for DS_PROMETHEUS
	foundInInputs := false
	for _, input := range dashboard.Inputs {
		if input.Name == "DS_PROMETHEUS" && input.Type == "datasource" &&
			input.PluginID == "prometheus" {
			foundInInputs = true
			break
		}
	}

	// Check templating.list for DS_PROMETHEUS
	foundInTemplating := false
	for _, tmpl := range dashboard.Templating.List {
		if tmpl.Name == "DS_PROMETHEUS" && tmpl.Type == "datasource" {
			queryStr := templateVarQueryString(tmpl.Query)
			if queryStr == "prometheus" {
				foundInTemplating = true
				break
			}
		}
	}

	assert.True(t, foundInInputs || foundInTemplating,
		"DS_PROMETHEUS datasource variable not found in __inputs or templating.list")
}

// TestDashboardDatasourceVariableTraces validates that the dashboard defines
// a DS_TRACES datasource variable in __inputs or templating.list supporting Jaeger or Tempo.
// Validates: R5.1
func TestDashboardDatasourceVariableTraces(t *testing.T) {
	dashboard := loadDashboard(t)

	// Check __inputs for DS_TRACES
	foundInInputs := false
	for _, input := range dashboard.Inputs {
		if input.Name == "DS_TRACES" && input.Type == "datasource" &&
			(input.PluginID == "jaeger" || input.PluginID == "tempo") {
			foundInInputs = true
			break
		}
	}

	// Check templating.list for DS_TRACES
	foundInTemplating := false
	for _, tmpl := range dashboard.Templating.List {
		if tmpl.Name == "DS_TRACES" && tmpl.Type == "datasource" {
			queryStr := templateVarQueryString(tmpl.Query)
			if queryStr == "jaeger" || queryStr == "tempo" {
				foundInTemplating = true
				break
			}
		}
	}

	assert.True(t, foundInInputs || foundInTemplating,
		"DS_TRACES datasource variable not found in __inputs or templating.list (must be jaeger or tempo)")
}

// TestDashboardHTTPRouteTemplateVariable validates that the http_route template
// variable is defined with the correct label_values query.
// Validates: R5.5
func TestDashboardHTTPRouteTemplateVariable(t *testing.T) {
	dashboard := loadDashboard(t)

	expectedQuery := "label_values(traces_duration_milliseconds_bucket, http_route)"

	found := false
	for _, tmpl := range dashboard.Templating.List {
		if tmpl.Name == "http_route" && tmpl.Type == "query" {
			// The query field can be a string or an object with a "query" field
			queryStr := templateVarQueryString(tmpl.Query)
			if strings.Contains(queryStr, expectedQuery) {
				found = true
				break
			}
			// Also check the definition field
			if strings.Contains(tmpl.Definition, expectedQuery) {
				found = true
				break
			}
		}
	}

	assert.True(t, found,
		"http_route template variable not found with expected query: %s", expectedQuery)
}

// TestDashboardExemplarEnabled validates that at least one panel target has
// exemplar: true configured.
// Validates: R5.3
func TestDashboardExemplarEnabled(t *testing.T) {
	dashboard := loadDashboard(t)

	found := false
	for _, panel := range dashboard.Panels {
		for _, target := range panel.Targets {
			if target.Exemplar {
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	assert.True(t, found,
		"no panel target has exemplar: true enabled")
}

// TestDashboardExemplarTraceLink validates that at least one panel has an internal
// link targeting the trace datasource with trace_id field.
// Validates: R5.4
func TestDashboardExemplarTraceLink(t *testing.T) {
	dashboard := loadDashboard(t)

	found := false
	for _, panel := range dashboard.Panels {
		for _, link := range panel.FieldConfig.Defaults.Links {
			if link.Field == "trace_id" && link.Internal {
				// Check if the datasource references DS_TRACES
				dsStr := datasourceString(link.Datasource)
				if strings.Contains(dsStr, "DS_TRACES") {
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	assert.True(t, found,
		"no panel has an internal link with trace_id field targeting DS_TRACES datasource")
}

// templateVarQueryString extracts the query string from a template variable's
// query field, which can be a plain string or an object with a "query" field.
func templateVarQueryString(query interface{}) string {
	switch v := query.(type) {
	case string:
		return v
	case map[string]interface{}:
		if q, ok := v["query"]; ok {
			if qs, ok := q.(string); ok {
				return qs
			}
		}
	}
	return ""
}

// datasourceString extracts a string representation from a datasource field,
// which can be a string or an object with uid/type fields.
func datasourceString(ds interface{}) string {
	switch v := ds.(type) {
	case string:
		return v
	case map[string]interface{}:
		result := ""
		if uid, ok := v["uid"]; ok {
			if uidStr, ok := uid.(string); ok {
				result += uidStr
			}
		}
		if dsType, ok := v["type"]; ok {
			if typeStr, ok := dsType.(string); ok {
				result += " " + typeStr
			}
		}
		return result
	}
	return ""
}
