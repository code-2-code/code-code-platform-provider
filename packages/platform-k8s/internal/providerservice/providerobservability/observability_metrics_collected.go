package providerobservability

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	platformtelemetry "code-code.internal/platform-k8s/internal/platform/telemetry"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

func (m *observabilityMetrics) recordCollectorValues(ownerID, providerID string, rows []ObservabilityMetricRow) {
	if m == nil || strings.TrimSpace(ownerID) == "" || strings.TrimSpace(providerID) == "" || len(rows) == 0 {
		return
	}
	for _, row := range rows {
		metricName := strings.TrimSpace(row.MetricName)
		if metricName == "" {
			continue
		}
		gauge, err := m.ensureCollectedGauge(metricName)
		if err != nil || gauge.gauge == nil {
			continue
		}
		labels := m.collectorLabels(ownerID, providerID, row.Labels)
		gauge.gauge.Record(context.Background(), row.Value, otelmetric.WithAttributes(credentialsAttributes(labels)...))
	}
}

func (m *observabilityMetrics) ensureCollectedGauge(metricName string) (collectedGauge, error) {
	if m == nil {
		return collectedGauge{}, nil
	}
	metricName = strings.TrimSpace(metricName)
	if !observabilityMetricNamePattern.MatchString(metricName) {
		return collectedGauge{}, fmt.Errorf("providerobservability: invalid collector metric name %q", metricName)
	}
	m.collectedMu.Lock()
	defer m.collectedMu.Unlock()
	if existing, ok := m.collectedGauges[metricName]; ok {
		return existing, nil
	}
	gauge, err := newCredentialsGauge(m.meter, metricName, "Active operation collected gauge value.")
	if err != nil {
		return collectedGauge{}, err
	}
	collected := collectedGauge{gauge: gauge}
	m.collectedGauges[metricName] = collected
	return collected, nil
}

func (m *observabilityMetrics) collectorLabels(ownerID string, providerID string, rowLabels map[string]string) map[string]string {
	labels := map[string]string{
		ownerKindLabel: m.ownerKindValue(),
		ownerIDLabel:   ownerID,
		m.ownerLabel:   ownerID,
		"provider_id":  providerID,
	}
	for key, value := range m.sanitizeCollectorLabels(rowLabels) {
		labels[key] = value
	}
	return labels
}

func (m *observabilityMetrics) ownerKindValue() string {
	return string(OwnerKindSurface)
}

func (m *observabilityMetrics) sanitizeCollectorLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	sanitized := map[string]string{}
	for key, value := range labels {
		trimmedKey := platformtelemetry.StorageMetricName(strings.TrimSpace(key))
		if trimmedKey == "" ||
			trimmedKey == ownerKindLabel ||
			trimmedKey == ownerIDLabel ||
			trimmedKey == m.ownerLabel ||
			trimmedKey == "provider_id" ||
			trimmedKey == "surface_id" ||
			trimmedKey == "instance_id" {
			continue
		}
		if !observabilityLabelNamePattern.MatchString(trimmedKey) {
			continue
		}
		sanitized[trimmedKey] = strings.TrimSpace(value)
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}

func credentialsAttributes(labels map[string]string) []attribute.KeyValue {
	if len(labels) == 0 {
		return nil
	}
	names := slices.Sorted(maps.Keys(labels))
	attrs := make([]attribute.KeyValue, 0, len(names))
	for _, name := range names {
		attrs = append(attrs, attribute.String(name, labels[name]))
	}
	return attrs
}
