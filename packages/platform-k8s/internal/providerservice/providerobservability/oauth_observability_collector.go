package providerobservability

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

const (
	oauthObservabilityMaxBodyReadSize = 1 << 14
)

// OAuthObservabilityCollector probes one CLI OAuth management surface.
type OAuthObservabilityCollector interface {
	CollectorID() string
	Collect(context.Context, OAuthObservabilityCollectInput) (*OAuthObservabilityCollectResult, error)
}

// OAuthObservabilityCollectInput carries one collector execution context.
type OAuthObservabilityCollectInput struct {
	ProviderSurfaceBindingID string
	CredentialID             string
	AccessToken              string
	HTTPClient               *http.Client
	MaterialValues           map[string]string
	CredentialBackfills      []CredentialBackfillRule
	ClientVersion            string
	ModelCatalogUserAgent    string
	ObservabilityUserAgent   string
}

// OAuthObservabilityCollectResult carries active operation metric values from one collector execution.
type OAuthObservabilityCollectResult struct {
	GaugeRows                []OAuthObservabilityMetricRow
	CredentialBackfillValues map[string]string
}

type OAuthObservabilityMetricRow struct {
	MetricName string
	Labels     map[string]string
	Value      float64
}

func gaugeRows(values map[string]float64) []OAuthObservabilityMetricRow {
	if len(values) == 0 {
		return nil
	}
	rows := make([]OAuthObservabilityMetricRow, 0, len(values))
	for metricName, value := range values {
		trimmedMetricName := strings.TrimSpace(metricName)
		if trimmedMetricName == "" {
			continue
		}
		rows = append(rows, OAuthObservabilityMetricRow{
			MetricName: trimmedMetricName,
			Value:      value,
		})
	}
	return rows
}

type oauthObservabilityUnauthorizedError struct {
	message string
}

func (e *oauthObservabilityUnauthorizedError) Error() string {
	return e.message
}

func unauthorizedOAuthObservabilityError(message string) error {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		trimmed = "oauth unauthorized"
	}
	return &oauthObservabilityUnauthorizedError{message: trimmed}
}

func isOAuthObservabilityUnauthorizedError(err error) bool {
	var target *oauthObservabilityUnauthorizedError
	return errors.As(err, &target)
}
