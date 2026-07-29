// Package observability provides shared test helpers for Prometheus rule validation tests.
//
// TelemetryFlow Order Service - AI-Powered Observability & Incident Response Management (IRM) Platform
// Copyright (c) 2024-2026 Telemetri Data Indonesia. All rights reserved.

package observability

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Shared types for parsing Prometheus rules YAML
// =============================================================================

// RulesFile represents the top-level Prometheus rules file structure.
type RulesFile struct {
	Groups []RuleGroup `yaml:"groups"`
}

// RuleGroup represents a Prometheus rule group.
type RuleGroup struct {
	Name     string `yaml:"name"`
	Interval string `yaml:"interval"`
	Rules    []Rule `yaml:"rules"`
}

// Rule represents a single Prometheus recording or alerting rule.
type Rule struct {
	Record      string            `yaml:"record,omitempty"`
	Alert       string            `yaml:"alert,omitempty"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// =============================================================================
// Shared helper functions
// =============================================================================

// rulesFilePath returns the absolute path to the Prometheus rules file.
func rulesFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	projectRoot := filepath.Join(testDir, "..", "..", "..")
	return filepath.Join(projectRoot, "configs", "prometheus", "rules", "order_service.yml")
}

// loadRulesFile reads and parses the Prometheus rules YAML file.
func loadRulesFile(t *testing.T) RulesFile {
	t.Helper()

	data, err := os.ReadFile(rulesFilePath())
	require.NoError(t, err, "failed to read rules file")

	var rules RulesFile
	err = yaml.Unmarshal(data, &rules)
	require.NoError(t, err, "failed to parse rules YAML")

	return rules
}

// findRecordingRule locates the P95 recording rule in the parsed rules file.
func findRecordingRule(t *testing.T, rules RulesFile) Rule {
	t.Helper()

	for _, group := range rules.Groups {
		for _, r := range group.Rules {
			if r.Record == "order_service:http_request_duration_p95:5m" {
				return r
			}
		}
	}

	t.Fatal("order_service:http_request_duration_p95:5m recording rule not found in rules file")
	return Rule{}
}

// findAlertRule locates the HighP95Latency alerting rule in the parsed rules file.
func findAlertRule(t *testing.T, rules RulesFile) Rule {
	t.Helper()

	for _, group := range rules.Groups {
		for _, r := range group.Rules {
			if r.Alert == "HighP95Latency" {
				return r
			}
		}
	}

	t.Fatal("HighP95Latency alert rule not found in rules file")
	return Rule{}
}

// =============================================================================
// NaN comparison helpers
//
// These wrap NaN comparisons in function calls to avoid staticcheck SA4012
// ("no value is equal to NaN") while still testing the IEEE 754 / PromQL
// semantic that NaN comparisons always return false.
// =============================================================================

// nanGreaterThan returns a > b. Used to test NaN comparison semantics without
// triggering staticcheck SA4012.
//
//nolint:unparam
func nanGreaterThan(a, b float64) bool {
	return a > b
}

// nanLessThan returns a < b. Used to test NaN comparison semantics without
// triggering staticcheck SA4012.
//
//nolint:unparam
func nanLessThan(a, b float64) bool {
	return a < b
}

// nanEqualTo returns a == b. Used to test NaN comparison semantics without
// triggering staticcheck SA4012.
//
//nolint:unparam
func nanEqualTo(a, b float64) bool {
	return a == b
}
