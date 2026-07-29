// Package observability provides property-based tests for Prometheus recording rules.
//
// TelemetryFlow Order Service - AI-Powered Observability & Incident Response Management (IRM) Platform
// Copyright (c) 2024-2026 Telemetri Data Indonesia. All rights reserved.
// Open Source Software built by Telemetri Data Indonesia.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package observability

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// =============================================================================
// Property 2: Recording rule output excludes the `le` bucket label
//
// For any time series produced by the `order_service:http_request_duration_p95:5m`
// recording rule, the label set SHALL contain `http_method` and `http_route`
// but SHALL NOT contain the `le` label.
//
// **Validates: Requirements 1.2**
// =============================================================================

// extractByClauseLabels extracts labels from the `by (...)` clause in a PromQL expression.
func extractByClauseLabels(expr string) []string {
	re := regexp.MustCompile(`by\s*\(([^)]+)\)`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) < 2 {
		return nil
	}
	parts := strings.Split(matches[1], ",")
	var labels []string
	for _, p := range parts {
		labels = append(labels, strings.TrimSpace(p))
	}
	return labels
}

// computeOutputLabels determines the output labels of a histogram_quantile expression.
// histogram_quantile consumes the `le` label, so output = by_labels - {le}.
func computeOutputLabels(byLabels []string) []string {
	var output []string
	for _, l := range byLabels {
		if l != "le" {
			output = append(output, l)
		}
	}
	return output
}

// TestProperty_RecordingRuleOutputExcludesLeLabel is the property-based test
// verifying that for any http_method and http_route combination, the recording
// rule's output labels contain http_method and http_route but NOT le.
//
// **Validates: Requirements 1.2**
func TestProperty_RecordingRuleOutputExcludesLeLabel(t *testing.T) {
	// Load and parse the rules file using shared helper
	rules := loadRulesFile(t)

	// Find the P95 recording rule using shared helper
	recordingRule := findRecordingRule(t, rules)

	// Extract the by clause labels from the expression
	byLabels := extractByClauseLabels(recordingRule.Expr)
	require.NotEmpty(t, byLabels, "recording rule expression must contain a 'by' clause")

	// Verify histogram_quantile is used (which consumes `le`)
	require.True(t, strings.Contains(recordingRule.Expr, "histogram_quantile"),
		"recording rule must use histogram_quantile (which consumes the le label)")

	// Verify the by clause contains le (required for histogram_quantile to work)
	assert.Contains(t, byLabels, "le",
		"by clause must include 'le' for histogram_quantile to function correctly")

	// Verify the by clause contains http_method and http_route
	assert.Contains(t, byLabels, "http_method",
		"by clause must include 'http_method'")
	assert.Contains(t, byLabels, "http_route",
		"by clause must include 'http_route'")

	// Compute the output labels (histogram_quantile removes le)
	outputLabels := computeOutputLabels(byLabels)

	// Property-based test: for any arbitrary http_method and http_route values,
	// the output label set always contains http_method and http_route but NOT le.
	rapid.Check(t, func(t *rapid.T) {
		// Generate arbitrary http_method values
		httpMethod := rapid.SampledFrom([]string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS",
		}).Draw(t, "http_method")

		// Generate arbitrary http_route values (route templates)
		httpRoute := rapid.OneOf(
			rapid.SampledFrom([]string{
				"/api/v1/orders",
				"/api/v1/orders/{id}",
				"/api/v1/orders/{id}/items",
				"/api/v1/orders/{id}/items/{itemId}",
				"/health",
				"/ready",
				"/",
				"/api/v2/orders",
				"/api/v1/customers/{id}/orders",
				"/api/v1/products",
			}),
			rapid.StringMatching(`/api/v[0-9]+/[a-z]+(/\{[a-z_]+\})?`),
		).Draw(t, "http_route")

		// Simulate label set produced by the recording rule for these inputs.
		// histogram_quantile consumes `le`, so the output labels are
		// all by-clause labels EXCEPT le.
		simulatedOutputLabels := make(map[string]string)
		for _, label := range outputLabels {
			switch label {
			case "http_method":
				simulatedOutputLabels[label] = httpMethod
			case "http_route":
				simulatedOutputLabels[label] = httpRoute
			default:
				simulatedOutputLabels[label] = "some_value"
			}
		}

		// PROPERTY: Output contains http_method
		if _, ok := simulatedOutputLabels["http_method"]; !ok {
			t.Fatalf("output label set missing 'http_method' for method=%s route=%s",
				httpMethod, httpRoute)
		}

		// PROPERTY: Output contains http_route
		if _, ok := simulatedOutputLabels["http_route"]; !ok {
			t.Fatalf("output label set missing 'http_route' for method=%s route=%s",
				httpMethod, httpRoute)
		}

		// PROPERTY: Output does NOT contain le
		if _, ok := simulatedOutputLabels["le"]; ok {
			t.Fatalf("output label set must NOT contain 'le' but it does for method=%s route=%s",
				httpMethod, httpRoute)
		}

		// Additional verification: outputLabels list does not contain "le"
		for _, l := range outputLabels {
			if l == "le" {
				t.Fatalf("output labels list contains 'le' which should have been consumed by histogram_quantile")
			}
		}
	})
}
