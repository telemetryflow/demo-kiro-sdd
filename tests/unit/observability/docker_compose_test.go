// Package observability provides validation tests for Docker Compose volume mounts.
//
// TelemetryFlow Order Service - AI-Powered Observability & Incident Response Management (IRM) Platform
// Copyright (c) 2024-2026 Telemetri Data Indonesia. All rights reserved.

package observability

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Shared types for parsing Docker Compose YAML
// Used by this file and alertmanager_config_test.go
// =============================================================================

// ComposeFile represents the top-level structure of a docker-compose.yml file.
type ComposeFile struct {
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService represents a single service definition in docker-compose.yml.
type ComposeService struct {
	Image       string      `yaml:"image"`
	Command     interface{} `yaml:"command,omitempty"`
	Environment []string    `yaml:"environment,omitempty"`
	Volumes     []string    `yaml:"volumes,omitempty"`
	Ports       []string    `yaml:"ports,omitempty"`
}

// loadDockerCompose reads and parses the docker-compose.yml file.
func loadDockerCompose(t *testing.T) ComposeFile {
	t.Helper()

	data, err := os.ReadFile(getDockerComposeFilePath())
	require.NoError(t, err, "failed to read docker-compose.yml")

	var compose ComposeFile
	err = yaml.Unmarshal(data, &compose)
	require.NoError(t, err, "failed to parse docker-compose.yml")

	return compose
}

// =============================================================================
// Validation Tests — Docker Compose Volume Mounts
// Validates: R3.2, R6.1
// =============================================================================

// TestPrometheusRulesVolumeMount validates that the Prometheus service mounts
// the rules directory with a read-only (:ro) suffix.
// Validates: R6.1
func TestPrometheusRulesVolumeMount(t *testing.T) {
	compose := loadDockerCompose(t)

	promService, ok := compose.Services["prometheus"]
	require.True(t, ok, "prometheus service not found in docker-compose.yml")

	expectedMount := "./configs/prometheus/rules/:/etc/prometheus/rules/:ro"

	found := false
	for _, vol := range promService.Volumes {
		if strings.TrimSpace(vol) == expectedMount {
			found = true
			break
		}
	}

	assert.True(t, found,
		"prometheus service missing expected volume mount: %s\nActual volumes: %v",
		expectedMount, promService.Volumes)
}

// TestPrometheusRulesVolumeReadOnly validates that the Prometheus rules directory
// volume mount has the :ro suffix for read-only access.
// Validates: R6.1
func TestPrometheusRulesVolumeReadOnly(t *testing.T) {
	compose := loadDockerCompose(t)

	promService, ok := compose.Services["prometheus"]
	require.True(t, ok, "prometheus service not found in docker-compose.yml")

	found := false
	for _, vol := range promService.Volumes {
		if strings.Contains(vol, "/etc/prometheus/rules/") {
			assert.True(t, strings.HasSuffix(vol, ":ro"),
				"prometheus rules volume mount must have :ro suffix, got: %s", vol)
			found = true
			break
		}
	}

	assert.True(t, found,
		"no volume mount found targeting /etc/prometheus/rules/ in prometheus service")
}

// TestAlertmanagerConfigVolumeMount validates that the Alertmanager service mounts
// the configuration file with a read-only (:ro) suffix.
// Validates: R3.2
func TestAlertmanagerConfigVolumeMount(t *testing.T) {
	compose := loadDockerCompose(t)

	alertService, ok := compose.Services["alertmanager"]
	require.True(t, ok, "alertmanager service not found in docker-compose.yml")

	expectedMount := "./configs/alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro"

	found := false
	for _, vol := range alertService.Volumes {
		if strings.TrimSpace(vol) == expectedMount {
			found = true
			break
		}
	}

	assert.True(t, found,
		"alertmanager service missing expected volume mount: %s\nActual volumes: %v",
		expectedMount, alertService.Volumes)
}

// TestAlertmanagerConfigVolumeReadOnly validates that the Alertmanager config
// volume mount has the :ro suffix for read-only access.
// Validates: R3.2
func TestAlertmanagerConfigVolumeReadOnly(t *testing.T) {
	compose := loadDockerCompose(t)

	alertService, ok := compose.Services["alertmanager"]
	require.True(t, ok, "alertmanager service not found in docker-compose.yml")

	found := false
	for _, vol := range alertService.Volumes {
		if strings.Contains(vol, "/etc/alertmanager/alertmanager.yml") {
			assert.True(t, strings.HasSuffix(vol, ":ro"),
				"alertmanager config volume mount must have :ro suffix, got: %s", vol)
			found = true
			break
		}
	}

	assert.True(t, found,
		"no volume mount found targeting /etc/alertmanager/alertmanager.yml in alertmanager service")
}
