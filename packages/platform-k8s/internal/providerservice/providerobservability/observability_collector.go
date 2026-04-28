package providerobservability

import (
	"context"
	"net/http"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
)

const (
	observabilityMaxBodyReadSize = 1 << 16
)

// ObservabilityCollector probes one vendor or CLI management surface.
type ObservabilityCollector interface {
	CollectorID() string
	Collect(context.Context, ObservabilityCollectInput) (*ObservabilityCollectResult, error)
}

// ObservabilityAuthAdapter is an optional interface for collectors that
// need a specific egress auth adapter.
type ObservabilityAuthAdapter interface {
	AuthAdapterID() string
}

// ObservabilityCollectInput carries one collector execution context.
// Both vendor and OAuth collectors receive the same shape; each collector
// reads only the fields relevant to its probe logic.
type ObservabilityCollectInput struct {
	// Common
	ProviderID               string
	ProviderSurfaceBindingID string
	CredentialID             string
	HTTPClient               *http.Client
	CredentialBackfills      []CredentialBackfillRule
	MaterialValues           map[string]string

	// Vendor-specific
	OwnerID                 string
	SurfaceBaseURL          string
	APIKey                  string
	ObservabilityCredential *credentialv1.ResolvedCredential

	// OAuth-specific
	AccessToken            string
	ClientVersion          string
	ModelCatalogUserAgent  string
	ObservabilityUserAgent string
}

// ObservabilityCollectResult carries metric values from one collector execution.
type ObservabilityCollectResult struct {
	GaugeRows                []ObservabilityMetricRow
	CredentialBackfillValues map[string]string
}

type ObservabilityMetricRow struct {
	MetricName string
	Labels     map[string]string
	Value      float64
}

func gaugeRows(values map[string]float64) []ObservabilityMetricRow {
	if len(values) == 0 {
		return nil
	}
	rows := make([]ObservabilityMetricRow, 0, len(values))
	for metricName, value := range values {
		trimmedMetricName := strings.TrimSpace(metricName)
		if trimmedMetricName == "" {
			continue
		}
		rows = append(rows, ObservabilityMetricRow{
			MetricName: trimmedMetricName,
			Value:      value,
		})
	}
	return rows
}

func observabilityCredentialToken(resolved *credentialv1.ResolvedCredential) string {
	if resolved == nil {
		return ""
	}
	switch resolved.GetKind() {
	case credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY:
		return strings.TrimSpace(resolved.GetApiKey().GetApiKey())
	case credentialv1.CredentialKind_CREDENTIAL_KIND_OAUTH:
		return strings.TrimSpace(resolved.GetOauth().GetAccessToken())
	case credentialv1.CredentialKind_CREDENTIAL_KIND_SESSION:
		return observabilitySessionValue(resolved, "token", "access_token", "api_key", "session_token", "authjs_session_token", "authjs.session-token")
	default:
		return ""
	}
}

func observabilitySessionValue(resolved *credentialv1.ResolvedCredential, keys ...string) string {
	if resolved == nil || resolved.GetKind() != credentialv1.CredentialKind_CREDENTIAL_KIND_SESSION || resolved.GetSession() == nil {
		return ""
	}
	values := resolved.GetSession().GetValues()
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	return ""
}
