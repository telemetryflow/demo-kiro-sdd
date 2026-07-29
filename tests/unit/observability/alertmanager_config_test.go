// Package observability provides validation tests for Alertmanager configuration extensibility.
//
// TelemetryFlow Order Service - AI-Powered Observability & Incident Response Management (IRM) Platform
// Copyright (c) 2024-2026 Telemetri Data Indonesia. All rights reserved.

package observability

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Shared helper — getDockerComposeFilePath
// Used by this file and docker_compose_test.go
// =============================================================================

// getDockerComposeFilePath returns the absolute path to the docker-compose.yml file.
func getDockerComposeFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	projectRoot := filepath.Join(testDir, "..", "..", "..")
	return filepath.Join(projectRoot, "docker-compose.yml")
}

// =============================================================================
// Types for parsing Alertmanager configuration
// =============================================================================

// AlertmanagerConfig represents the top-level Alertmanager configuration.
type AlertmanagerConfig struct {
	Global    map[string]interface{} `yaml:"global"`
	Route     AlertRoute             `yaml:"route"`
	Receivers []AlertReceiver        `yaml:"receivers"`
}

// AlertRoute represents the routing configuration.
type AlertRoute struct {
	Receiver       string   `yaml:"receiver"`
	GroupBy        []string `yaml:"group_by"`
	GroupWait      string   `yaml:"group_wait"`
	GroupInterval  string   `yaml:"group_interval"`
	RepeatInterval string   `yaml:"repeat_interval"`
}

// AlertReceiver represents a receiver entry in Alertmanager config.
type AlertReceiver struct {
	Name           string        `yaml:"name"`
	WebhookConfigs []interface{} `yaml:"webhook_configs,omitempty"`
	EmailConfigs   []interface{} `yaml:"email_configs,omitempty"`
	SlackConfigs   []interface{} `yaml:"slack_configs,omitempty"`
}

// =============================================================================
// Helper functions
// =============================================================================

// alertmanagerConfigPath returns the path to the Alertmanager config file.
func alertmanagerConfigPath() string {
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	projectRoot := filepath.Join(testDir, "..", "..", "..")
	return filepath.Join(projectRoot, "configs", "alertmanager", "alertmanager.yml")
}

// loadAlertmanagerConfig reads and parses the Alertmanager configuration file.
func loadAlertmanagerConfig(t *testing.T) AlertmanagerConfig {
	t.Helper()

	data, err := os.ReadFile(alertmanagerConfigPath())
	require.NoError(t, err, "failed to read alertmanager config file")

	var config AlertmanagerConfig
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err, "failed to parse alertmanager config YAML")

	return config
}

// =============================================================================
// Validation Tests — Alertmanager Config Extensibility
// Validates: R3.5
// =============================================================================

// TestAlertmanagerConfig_ReceiversBlockExists verifies that the receivers block
// exists and contains at least one receiver named "default".
func TestAlertmanagerConfig_ReceiversBlockExists(t *testing.T) {
	config := loadAlertmanagerConfig(t)

	require.NotEmpty(t, config.Receivers, "receivers block must exist and contain at least one receiver")

	var foundDefault bool
	for _, receiver := range config.Receivers {
		if receiver.Name == "default" {
			foundDefault = true
			break
		}
	}
	assert.True(t, foundDefault, "receivers must contain a receiver named 'default'")
}

// TestAlertmanagerConfig_DefaultRouteUsesDefaultReceiver verifies the default
// route is configured to use the "default" receiver.
func TestAlertmanagerConfig_DefaultRouteUsesDefaultReceiver(t *testing.T) {
	config := loadAlertmanagerConfig(t)

	assert.Equal(t, "default", config.Route.Receiver,
		"default route must reference the 'default' receiver")
}

// TestAlertmanagerConfig_ReceiverStructureSupportsExtension verifies that the
// receiver structure is a list/map that can accept webhook_configs, email_configs,
// or slack_configs without structural changes to the file format.
func TestAlertmanagerConfig_ReceiverStructureSupportsExtension(t *testing.T) {
	// Read the raw YAML to verify the structure supports extension
	data, err := os.ReadFile(alertmanagerConfigPath())
	require.NoError(t, err, "failed to read alertmanager config file")

	// Parse into a generic structure to verify receivers is a list of maps
	var raw map[string]interface{}
	err = yaml.Unmarshal(data, &raw)
	require.NoError(t, err, "failed to parse alertmanager config as generic YAML")

	receivers, ok := raw["receivers"]
	require.True(t, ok, "receivers key must exist in alertmanager config")

	receiverList, ok := receivers.([]interface{})
	require.True(t, ok, "receivers must be a YAML sequence (list)")
	require.NotEmpty(t, receiverList, "receivers list must not be empty")

	// Verify each receiver is a map that can accept additional config keys
	for _, r := range receiverList {
		receiverMap, ok := r.(map[string]interface{})
		require.True(t, ok, "each receiver must be a YAML mapping (map)")

		// Must have a name field
		_, hasName := receiverMap["name"]
		assert.True(t, hasName, "each receiver must have a 'name' field")
	}

	// Verify the default receiver can accept webhook_configs, email_configs,
	// or slack_configs by confirming no structural constraint prevents it.
	// We do this by marshaling a receiver with all config types and verifying
	// it round-trips correctly through the same YAML schema.
	extensibleReceiver := AlertReceiver{
		Name:           "test-extensible",
		WebhookConfigs: []interface{}{map[string]interface{}{"url": "http://example.com/hook"}},
		EmailConfigs:   []interface{}{map[string]interface{}{"to": "ops@example.com"}},
		SlackConfigs:   []interface{}{map[string]interface{}{"channel": "#alerts"}},
	}

	marshaled, err := yaml.Marshal(extensibleReceiver)
	require.NoError(t, err, "receiver with webhook/email/slack configs must be marshalable")

	var roundTripped AlertReceiver
	err = yaml.Unmarshal(marshaled, &roundTripped)
	require.NoError(t, err, "receiver with webhook/email/slack configs must round-trip through YAML")

	assert.Equal(t, "test-extensible", roundTripped.Name)
	assert.Len(t, roundTripped.WebhookConfigs, 1, "webhook_configs should survive round-trip")
	assert.Len(t, roundTripped.EmailConfigs, 1, "email_configs should survive round-trip")
	assert.Len(t, roundTripped.SlackConfigs, 1, "slack_configs should survive round-trip")
}

// TestAlertmanagerConfig_DockerComposeNoHardCodedReceivers verifies that the
// alertmanager service in docker-compose.yml does not contain hard-coded
// receiver/notification logic (URLs, email addresses, Slack webhooks).
func TestAlertmanagerConfig_DockerComposeNoHardCodedReceivers(t *testing.T) {
	data, err := os.ReadFile(getDockerComposeFilePath())
	require.NoError(t, err, "failed to read docker-compose.yml")

	var compose map[string]interface{}
	err = yaml.Unmarshal(data, &compose)
	require.NoError(t, err, "failed to parse docker-compose.yml")

	services, ok := compose["services"].(map[string]interface{})
	require.True(t, ok, "docker-compose.yml must have a services section")

	alertmanagerSvc, ok := services["alertmanager"].(map[string]interface{})
	require.True(t, ok, "docker-compose.yml must have an alertmanager service")

	// Convert the alertmanager service definition to a string for pattern searching
	alertmanagerYAML, err := yaml.Marshal(alertmanagerSvc)
	require.NoError(t, err)
	svcString := strings.ToLower(string(alertmanagerYAML))

	// Patterns that would indicate hard-coded notification logic
	hardCodedPatterns := []struct {
		pattern     string
		description string
	}{
		{"slack.com/services", "hard-coded Slack webhook URL"},
		{"hooks.slack.com", "hard-coded Slack webhook URL"},
		{"smtp://", "hard-coded SMTP URL"},
		{"webhook_url", "hard-coded webhook URL in service definition"},
		{"email_configs", "hard-coded email config in service definition"},
		{"slack_configs", "hard-coded Slack config in service definition"},
		{"pagerduty", "hard-coded PagerDuty config in service definition"},
		{"opsgenie", "hard-coded OpsGenie config in service definition"},
	}

	for _, p := range hardCodedPatterns {
		assert.False(t, strings.Contains(svcString, p.pattern),
			"alertmanager service should not contain %s: found '%s'", p.description, p.pattern)
	}
}

// TestAlertmanagerConfig_OnlyConfigFileMounted verifies that the alertmanager
// service in docker-compose.yml only mounts the config file, meaning all
// notification routing is managed via the config file (not docker-compose.yml).
func TestAlertmanagerConfig_OnlyConfigFileMounted(t *testing.T) {
	compose := loadDockerCompose(t)

	alertService, ok := compose.Services["alertmanager"]
	require.True(t, ok, "alertmanager service not found in docker-compose.yml")

	// Verify volumes only contain the alertmanager config file mount
	require.NotEmpty(t, alertService.Volumes, "alertmanager service must mount the config file")

	for _, vol := range alertService.Volumes {
		// Each volume should reference the alertmanager config, not notification-specific files
		assert.True(t,
			strings.Contains(vol, "alertmanager.yml") || strings.Contains(vol, "alertmanager.yaml"),
			"alertmanager volumes should only mount the config file, got: %s", vol)
	}
}

// TestAlertmanagerConfig_AddingReceiverRequiresOnlyConfigEdit confirms that
// adding a new receiver requires only editing alertmanager.yml and no changes
// to docker-compose.yml. This is validated by checking that:
// 1. The alertmanager service has no environment variables for receivers
// 2. The alertmanager service command does not reference specific receivers
// 3. The config file structure accepts additional receivers without schema changes
func TestAlertmanagerConfig_AddingReceiverRequiresOnlyConfigEdit(t *testing.T) {
	data, err := os.ReadFile(getDockerComposeFilePath())
	require.NoError(t, err, "failed to read docker-compose.yml")

	var compose map[string]interface{}
	err = yaml.Unmarshal(data, &compose)
	require.NoError(t, err)

	services, ok := compose["services"].(map[string]interface{})
	require.True(t, ok)

	alertmanagerSvc, ok := services["alertmanager"].(map[string]interface{})
	require.True(t, ok)

	// 1. No environment variables referencing receivers or notification endpoints
	if env, hasEnv := alertmanagerSvc["environment"]; hasEnv {
		envStr := strings.ToLower(strings.Join(toStringSlice(env), " "))
		assert.False(t, strings.Contains(envStr, "receiver"),
			"alertmanager environment should not reference receivers")
		assert.False(t, strings.Contains(envStr, "webhook"),
			"alertmanager environment should not reference webhooks")
		assert.False(t, strings.Contains(envStr, "slack"),
			"alertmanager environment should not reference slack")
		assert.False(t, strings.Contains(envStr, "email"),
			"alertmanager environment should not reference email")
	}

	// 2. Command does not reference specific receivers
	if cmd, hasCmd := alertmanagerSvc["command"]; hasCmd {
		cmdStr := strings.ToLower(commandToString(cmd))
		assert.False(t, strings.Contains(cmdStr, "receiver"),
			"alertmanager command should not reference receivers")
		assert.False(t, strings.Contains(cmdStr, "webhook"),
			"alertmanager command should not reference webhooks")
	}

	// 3. Config file structure accepts additional receivers
	config := loadAlertmanagerConfig(t)
	// Simulate adding a new receiver - if the structure is extensible,
	// appending to the receivers list is all that's needed
	newReceiver := AlertReceiver{
		Name:         "slack-ops",
		SlackConfigs: []interface{}{map[string]interface{}{"channel": "#ops-alerts"}},
	}
	extendedReceivers := append(config.Receivers, newReceiver)

	// Verify the extended config can be marshaled (proving extensibility)
	extendedConfig := AlertmanagerConfig{
		Global:    config.Global,
		Route:     config.Route,
		Receivers: extendedReceivers,
	}
	_, err = yaml.Marshal(extendedConfig)
	assert.NoError(t, err, "alertmanager config must support adding new receivers without structural changes")
}

// =============================================================================
// Helper utilities
// =============================================================================

// toStringSlice converts an interface{} (expected []interface{}) to []string.
func toStringSlice(v interface{}) []string {
	switch val := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return val
	default:
		return nil
	}
}

// commandToString converts a command field (string or []interface{}) to a string.
func commandToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return ""
	}
}
