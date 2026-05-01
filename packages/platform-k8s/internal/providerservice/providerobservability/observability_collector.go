package providerobservability

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const (
	observabilityMaxBodyReadSize = 1 << 16
)

// ObservabilityCollector probes one vendor or CLI management surface.
type ObservabilityCollector interface {
	CollectorID() string
	Collect(context.Context, ObservabilityCollectInput) (*ObservabilityCollectResult, error)
}

// ObservabilityCollectInput carries one collector execution context.
// Both vendor and OAuth collectors receive the same shape; each collector
// reads only the fields relevant to its probe logic.
type ObservabilityCollectInput struct {
	// Common
	ProviderID   string
	SurfaceID    string
	CredentialID string
	Auth         ObservabilityAuthClient
	HTTPClient   *http.Client

	// Vendor-specific
	SchemaID       string
	SurfaceBaseURL string

	// OAuth-specific
	ClientVersion          string
	ModelCatalogUserAgent  string
	ObservabilityUserAgent string
}

// ObservabilityCollectResult carries metric values from one collector execution.
type ObservabilityCollectResult struct {
	GaugeRows []ObservabilityMetricRow
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

// parseRFC3339Timestamp parses an RFC3339 timestamp from a raw any value.
// Shared by multiple collectors for quota reset time extraction.
func parseRFC3339Timestamp(raw any) (time.Time, bool) {
	value, _ := raw.(string)
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return time.Time{}, false
		}
	}
	return parsed.UTC(), true
}

// clampPercent clamps a percentage value to [0, 100].
func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
