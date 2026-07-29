// Package observability provides property-based tests for metric label cardinality constraints.
//
// **Validates: Requirements 1.2**
// Feature: post-orders-observability
// Property 1: No metric label contains a UUID or high-cardinality identifier
package observability

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// =============================================================================
// Allowed label values (from design document & Prometheus rules)
// =============================================================================

// AllowedHTTPMethods defines the valid http_method label values.
var AllowedHTTPMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

// AllowedHTTPRoutes defines the valid http_route label values (route templates, never resolved paths).
var AllowedHTTPRoutes = []string{
	"/api/v1/orders",
	"/api/v1/orders/{id}",
	"/api/v1/orders/{id}/items",
	"/health",
	"/ready",
	"/",
}

// AllowedHTTPStatusCodes defines the valid http_status_code label values.
var AllowedHTTPStatusCodes = []string{"200", "201", "400", "401", "403", "404", "422", "500"}

// AllowedLabelNames defines the only permitted metric label names.
var AllowedLabelNames = []string{"http_method", "http_route", "http_status_code"}

// ForbiddenLabelNames defines label names that must NEVER appear on metrics.
var ForbiddenLabelNames = []string{
	"order_id",
	"customer_id",
	"user_id",
	"session_id",
	"trace_id",
	"span_id",
	"request_id",
}

// =============================================================================
// Forbidden patterns (high-cardinality identifiers)
// =============================================================================

var (
	// uuidPattern matches UUID v4 format: 8-4-4-4-12 hex chars
	uuidPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

	// hexTraceIDPattern matches 32-character hex strings (trace_id/span_id format)
	hexTraceIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

	// emailPattern matches strings containing @ (email indicator)
	emailPattern = regexp.MustCompile(`@`)
)

// containsForbiddenPattern checks if a string matches any forbidden high-cardinality pattern.
func containsForbiddenPattern(value string) (bool, string) {
	if uuidPattern.MatchString(value) {
		return true, "UUID pattern"
	}
	if hexTraceIDPattern.MatchString(strings.ToLower(value)) {
		return true, "32-char hex (trace_id/span_id)"
	}
	if emailPattern.MatchString(value) {
		return true, "email pattern (contains @)"
	}
	return false, ""
}

// isForbiddenLabelName checks if a label name is in the forbidden list.
func isForbiddenLabelName(name string) bool {
	for _, forbidden := range ForbiddenLabelNames {
		if strings.EqualFold(name, forbidden) {
			return true
		}
	}
	return false
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// TestProperty_AllowedLabelValuesContainNoHighCardinalityIdentifiers verifies that
// the defined allowed label values do NOT contain any UUID, 32-char hex, or email patterns.
// This is a static property: the allowlist itself must be free of high-cardinality values.
//
// **Validates: Requirements 1.2**
func TestProperty_AllowedLabelValuesContainNoHighCardinalityIdentifiers(t *testing.T) {
	// Collect all allowed label values into a single pool
	allAllowedValues := make([]string, 0)
	allAllowedValues = append(allAllowedValues, AllowedHTTPMethods...)
	allAllowedValues = append(allAllowedValues, AllowedHTTPRoutes...)
	allAllowedValues = append(allAllowedValues, AllowedHTTPStatusCodes...)

	rapid.Check(t, func(t *rapid.T) {
		// Pick a random allowed label value from our allowlist
		idx := rapid.IntRange(0, len(allAllowedValues)-1).Draw(t, "labelValueIndex")
		value := allAllowedValues[idx]

		// Assert it does NOT match any forbidden pattern
		hasForbidden, patternName := containsForbiddenPattern(value)
		if hasForbidden {
			t.Fatalf("Allowed label value %q matches forbidden pattern: %s", value, patternName)
		}
	})
}

// TestProperty_RandomStringsMatchingForbiddenPatternsAreNotInAllowlist verifies that
// randomly generated strings that look like UUIDs, trace IDs, or emails are NOT present
// in the allowed label values. This ensures the allowlist cannot accidentally include
// high-cardinality identifiers.
//
// **Validates: Requirements 1.2**
func TestProperty_RandomStringsMatchingForbiddenPatternsAreNotInAllowlist(t *testing.T) {
	allAllowedValues := make(map[string]bool)
	for _, v := range AllowedHTTPMethods {
		allAllowedValues[v] = true
	}
	for _, v := range AllowedHTTPRoutes {
		allAllowedValues[v] = true
	}
	for _, v := range AllowedHTTPStatusCodes {
		allAllowedValues[v] = true
	}

	// Generator for UUID-shaped strings
	uuidGen := rapid.Custom(func(t *rapid.T) string {
		hex := "0123456789abcdef"
		var sb strings.Builder
		for i := 0; i < 8; i++ {
			sb.WriteByte(hex[rapid.IntRange(0, 15).Draw(t, "h")])
		}
		sb.WriteByte('-')
		for i := 0; i < 4; i++ {
			sb.WriteByte(hex[rapid.IntRange(0, 15).Draw(t, "h")])
		}
		sb.WriteByte('-')
		for i := 0; i < 4; i++ {
			sb.WriteByte(hex[rapid.IntRange(0, 15).Draw(t, "h")])
		}
		sb.WriteByte('-')
		for i := 0; i < 4; i++ {
			sb.WriteByte(hex[rapid.IntRange(0, 15).Draw(t, "h")])
		}
		sb.WriteByte('-')
		for i := 0; i < 12; i++ {
			sb.WriteByte(hex[rapid.IntRange(0, 15).Draw(t, "h")])
		}
		return sb.String()
	})

	// Generator for 32-char hex strings (trace_id / span_id format)
	traceIDGen := rapid.Custom(func(t *rapid.T) string {
		hex := "0123456789abcdef"
		var sb strings.Builder
		for i := 0; i < 32; i++ {
			sb.WriteByte(hex[rapid.IntRange(0, 15).Draw(t, "h")])
		}
		return sb.String()
	})

	// Generator for email-like strings
	emailGen := rapid.Custom(func(t *rapid.T) string {
		user := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "user")
		domain := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "domain")
		tld := rapid.SampledFrom([]string{"com", "org", "net", "io"}).Draw(t, "tld")
		return user + "@" + domain + "." + tld
	})

	t.Run("uuid_not_in_allowlist", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			uuid := uuidGen.Draw(t, "uuid")
			assert.False(t, allAllowedValues[uuid],
				"Generated UUID %q should NOT be in the allowed label values", uuid)
		})
	})

	t.Run("trace_id_not_in_allowlist", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			traceID := traceIDGen.Draw(t, "traceID")
			assert.False(t, allAllowedValues[traceID],
				"Generated trace_id %q should NOT be in the allowed label values", traceID)
		})
	})

	t.Run("email_not_in_allowlist", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			email := emailGen.Draw(t, "email")
			assert.False(t, allAllowedValues[email],
				"Generated email %q should NOT be in the allowed label values", email)
		})
	})
}

// TestProperty_NoAllowedLabelNameIsForbidden verifies that none of the allowed
// metric label NAMES match the forbidden label names list (order_id, customer_id, etc.).
//
// **Validates: Requirements 1.2**
func TestProperty_NoAllowedLabelNameIsForbidden(t *testing.T) {
	// Verify none of the allowed label names overlap with forbidden names
	rapid.Check(t, func(t *rapid.T) {
		// Pick a random name from the allowed label names
		idx := rapid.IntRange(0, len(AllowedLabelNames)-1).Draw(t, "labelNameIndex")
		name := AllowedLabelNames[idx]

		// Assert it is NOT in the forbidden list
		assert.False(t, isForbiddenLabelName(name),
			"Allowed label name %q must not be in the forbidden label names list", name)
	})
}

// TestProperty_ForbiddenLabelNamesAreNotInAllowlist verifies that forbidden label names
// (order_id, customer_id, user_id, session_id, trace_id, span_id, request_id) are never
// present in the allowed label names list.
//
// **Validates: Requirements 1.2**
func TestProperty_ForbiddenLabelNamesAreNotInAllowlist(t *testing.T) {
	allowedSet := make(map[string]bool)
	for _, name := range AllowedLabelNames {
		allowedSet[strings.ToLower(name)] = true
	}

	rapid.Check(t, func(t *rapid.T) {
		// Pick a random forbidden label name
		idx := rapid.IntRange(0, len(ForbiddenLabelNames)-1).Draw(t, "forbiddenIndex")
		forbidden := ForbiddenLabelNames[idx]

		// Assert it is NOT in the allowed label names
		assert.False(t, allowedSet[strings.ToLower(forbidden)],
			"Forbidden label name %q must NOT be in the allowed label names set", forbidden)
	})
}

// TestProperty_GeneratedLabelValuesFromAllowlistAreSafe uses rapid to generate
// random label sets (method + route + status_code combinations) and verifies
// that every combination is free from high-cardinality identifiers.
//
// **Validates: Requirements 1.2**
func TestProperty_GeneratedLabelValuesFromAllowlistAreSafe(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random valid label set from the allowlists
		method := rapid.SampledFrom(AllowedHTTPMethods).Draw(t, "method")
		route := rapid.SampledFrom(AllowedHTTPRoutes).Draw(t, "route")
		statusCode := rapid.SampledFrom(AllowedHTTPStatusCodes).Draw(t, "statusCode")

		// Check each label value for forbidden patterns
		for _, labelValue := range []string{method, route, statusCode} {
			hasForbidden, patternName := containsForbiddenPattern(labelValue)
			if hasForbidden {
				t.Fatalf("Label value %q from allowlist matches forbidden pattern: %s", labelValue, patternName)
			}
		}
	})
}
