// Package observability provides integration tests for observability configuration.
package observability

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Prometheus Rules Validation Tests
// Validates: R1.1, R6.2, R6.5
// =============================================================================

// rulesFile is a representation of a Prometheus rules YAML file.
type rulesFile struct {
	Groups []ruleGroup `yaml:"groups"`
}

type ruleGroup struct {
	Name     string `yaml:"name"`
	Interval string `yaml:"interval"`
	Rules    []rule `yaml:"rules"`
}

type rule struct {
	Record      string            `yaml:"record,omitempty"`
	Alert       string            `yaml:"alert,omitempty"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// prometheusConfig is a partial representation of prometheus.yml.
type prometheusConfig struct {
	RuleFiles []string       `yaml:"rule_files"`
	Alerting  alertingConfig `yaml:"alerting"`
}

type alertingConfig struct {
	Alertmanagers []alertmanagerEntry `yaml:"alertmanagers"`
}

type alertmanagerEntry struct {
	StaticConfigs []staticConfig `yaml:"static_configs"`
}

type staticConfig struct {
	Targets []string `yaml:"targets"`
}

// projectRoot returns the project root by walking up from this test file.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")
	// tests/integration/observability/rules_validation_test.go -> project root is 3 levels up
	return filepath.Join(filepath.Dir(filename), "..", "..", "..")
}

func TestPromtoolCheckRules(t *testing.T) {
	root := projectRoot(t)
	rulesPath := filepath.Join(root, "configs", "prometheus", "rules", "order_service.yml")

	// Skip if promtool is not available on this machine
	promtool, err := exec.LookPath("promtool")
	if err != nil {
		t.Skip("promtool not found in PATH, skipping promtool validation")
	}

	cmd := exec.Command(promtool, "check", "rules", rulesPath)
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "promtool check rules failed: %s", string(output))
}

func TestRulesFileStructure(t *testing.T) {
	root := projectRoot(t)
	rulesPath := filepath.Join(root, "configs", "prometheus", "rules", "order_service.yml")

	data, err := os.ReadFile(rulesPath)
	require.NoError(t, err, "failed to read rules file")

	var rf rulesFile
	err = yaml.Unmarshal(data, &rf)
	require.NoError(t, err, "failed to parse rules YAML")

	require.NotEmpty(t, rf.Groups, "rules file must contain at least one group")

	// Find the order_service_latency group
	var latencyGroup *ruleGroup
	for i := range rf.Groups {
		if rf.Groups[i].Name == "order_service_latency" {
			latencyGroup = &rf.Groups[i]
			break
		}
	}
	require.NotNil(t, latencyGroup, "rule group 'order_service_latency' not found")

	t.Run("group has correct evaluation interval", func(t *testing.T) {
		assert.Equal(t, "15s", latencyGroup.Interval, "evaluation interval must be 15s")
	})

	t.Run("contains recording rule order_service:http_request_duration_p95:5m", func(t *testing.T) {
		var recordingRule *rule
		for i := range latencyGroup.Rules {
			if latencyGroup.Rules[i].Record == "order_service:http_request_duration_p95:5m" {
				recordingRule = &latencyGroup.Rules[i]
				break
			}
		}
		require.NotNil(t, recordingRule, "recording rule 'order_service:http_request_duration_p95:5m' not found")
		assert.Contains(t, recordingRule.Expr, "histogram_quantile",
			"recording rule expression must use histogram_quantile")
	})

	t.Run("contains alerting rule HighP95Latency", func(t *testing.T) {
		var alertRule *rule
		for i := range latencyGroup.Rules {
			if latencyGroup.Rules[i].Alert == "HighP95Latency" {
				alertRule = &latencyGroup.Rules[i]
				break
			}
		}
		require.NotNil(t, alertRule, "alerting rule 'HighP95Latency' not found")
		assert.Equal(t, "2m", alertRule.For, "alerting rule 'for' duration must be 2m")
		assert.Equal(t, "warning", alertRule.Labels["severity"],
			"alerting rule must have severity: warning label")
	})
}

func TestPrometheusConfigRulesAndAlerting(t *testing.T) {
	root := projectRoot(t)
	promConfigPath := filepath.Join(root, "configs", "prometheus", "prometheus.yml")

	data, err := os.ReadFile(promConfigPath)
	require.NoError(t, err, "failed to read prometheus.yml")

	var cfg prometheusConfig
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err, "failed to parse prometheus.yml")

	t.Run("has rule_files directive", func(t *testing.T) {
		require.NotEmpty(t, cfg.RuleFiles, "prometheus.yml must have a rule_files directive")
		found := false
		for _, rf := range cfg.RuleFiles {
			if rf == "/etc/prometheus/rules/*.yml" {
				found = true
				break
			}
		}
		assert.True(t, found,
			"rule_files must include '/etc/prometheus/rules/*.yml', got: %v", cfg.RuleFiles)
	})

	t.Run("has alerting.alertmanagers section", func(t *testing.T) {
		require.NotEmpty(t, cfg.Alerting.Alertmanagers,
			"prometheus.yml must have an alerting.alertmanagers section")

		// Verify at least one alertmanager target is configured
		hasTarget := false
		for _, am := range cfg.Alerting.Alertmanagers {
			for _, sc := range am.StaticConfigs {
				if len(sc.Targets) > 0 {
					hasTarget = true
					break
				}
			}
		}
		assert.True(t, hasTarget, "alertmanagers section must have at least one target configured")
	})
}
