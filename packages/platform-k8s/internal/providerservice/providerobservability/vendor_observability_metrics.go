package providerobservability

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	platformtelemetry "code-code.internal/platform-k8s/internal/platform/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	vendorObservabilityProbeRunsMetricName        = "gen_ai.provider.vendor.api_key.active.operation.runs.total"
	vendorObservabilityProbeLastRunMetricName     = "gen_ai.provider.vendor.api_key.active.operation.last.run.timestamp.seconds"
	vendorObservabilityProbeLastOutcomeMetricName = "gen_ai.provider.vendor.api_key.active.operation.last.outcome"
	vendorObservabilityProbeLastReasonMetricName  = "gen_ai.provider.vendor.api_key.active.operation.last.reason"
	vendorObservabilityProbeNextAllowedMetricName = "gen_ai.provider.vendor.api_key.active.operation.next.allowed.timestamp.seconds"
	vendorObservabilityAuthUsableMetricName       = "gen_ai.provider.vendor.api_key.active.operation.auth.usable"
	vendorCredentialLastUsedMetricName            = "gen_ai.provider.vendor.api_key.credential.last.used.timestamp.seconds"
)

type vendorObservabilityMetrics struct {
	meter              otelmetric.Meter
	probeRuns          otelmetric.Int64Counter
	probeLastRun       otelmetric.Float64Gauge
	probeLastOutcome   otelmetric.Float64Gauge
	probeLastReason    otelmetric.Float64Gauge
	probeNextAllowed   otelmetric.Float64Gauge
	authUsable         otelmetric.Float64Gauge
	credentialLastUsed otelmetric.Float64Gauge

	lastReasonMu sync.Mutex
	lastReasons  map[string]string

	collectedMu     sync.Mutex
	collectedGauges map[string]vendorCollectedGauge
}

type vendorCollectedGauge struct {
	gauge otelmetric.Float64Gauge
}

var (
	registerVendorObservabilityMetricsOnce sync.Once
	registeredVendorObservabilityMetrics   *vendorObservabilityMetrics
	registerVendorObservabilityMetricsErr  error
	vendorObservabilityMetricNamePattern   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.\-/]{0,254}$`)
	vendorObservabilityLabelNamePattern    = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

func registerVendorObservabilityMetrics() (*vendorObservabilityMetrics, error) {
	registerVendorObservabilityMetricsOnce.Do(func() {
		meter := otel.Meter("platform-k8s/providerobservability")
		probeRuns, err := newCredentialsCounter(
			meter,
			vendorObservabilityProbeRunsMetricName,
			"Count of vendor API-key active observability probes.",
		)
		if err != nil {
			registerVendorObservabilityMetricsErr = err
			return
		}
		probeLastRun, err := newCredentialsGauge(
			meter,
			vendorObservabilityProbeLastRunMetricName,
			"Unix timestamp of the last vendor API-key active observability operation.",
		)
		if err != nil {
			registerVendorObservabilityMetricsErr = err
			return
		}
		probeLastOutcome, err := newCredentialsGauge(
			meter,
			vendorObservabilityProbeLastOutcomeMetricName,
			"Numeric code of the last vendor API-key active observability operation outcome.",
		)
		if err != nil {
			registerVendorObservabilityMetricsErr = err
			return
		}
		probeLastReason, err := newCredentialsGauge(
			meter,
			vendorObservabilityProbeLastReasonMetricName,
			"Machine-readable reason for the last decisive vendor API-key active observability operation.",
		)
		if err != nil {
			registerVendorObservabilityMetricsErr = err
			return
		}
		probeNextAllowed, err := newCredentialsGauge(
			meter,
			vendorObservabilityProbeNextAllowedMetricName,
			"Unix timestamp for the next allowed vendor API-key active observability operation.",
		)
		if err != nil {
			registerVendorObservabilityMetricsErr = err
			return
		}
		authUsable, err := newCredentialsGauge(
			meter,
			vendorObservabilityAuthUsableMetricName,
			"Whether the last decisive vendor API-key active operation confirmed usable management-plane auth (1) or auth_blocked (0).",
		)
		if err != nil {
			registerVendorObservabilityMetricsErr = err
			return
		}
		credentialLastUsed, err := newCredentialsGauge(
			meter,
			vendorCredentialLastUsedMetricName,
			"Unix timestamp of the last vendor API-key credential use against the provider.",
		)
		if err != nil {
			registerVendorObservabilityMetricsErr = err
			return
		}
		metrics := &vendorObservabilityMetrics{
			meter:              meter,
			probeRuns:          probeRuns,
			probeLastRun:       probeLastRun,
			probeLastOutcome:   probeLastOutcome,
			probeLastReason:    probeLastReason,
			probeNextAllowed:   probeNextAllowed,
			authUsable:         authUsable,
			credentialLastUsed: credentialLastUsed,
			lastReasons:        map[string]string{},
			collectedGauges:    map[string]vendorCollectedGauge{},
		}
		registeredVendorObservabilityMetrics = metrics
	})
	if registerVendorObservabilityMetricsErr != nil {
		return nil, registerVendorObservabilityMetricsErr
	}
	return registeredVendorObservabilityMetrics, nil
}

func (m *vendorObservabilityMetrics) record(
	vendorID string,
	providerID string,
	trigger VendorObservabilityProbeTrigger,
	outcome VendorObservabilityProbeOutcome,
	reason string,
	lastRunAt time.Time,
	nextAllowedAt time.Time,
) {
	if m == nil {
		return
	}
	if vendorID == "" || providerID == "" {
		return
	}
	ctx := context.Background()
	m.probeRuns.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String("vendor_id", vendorID),
		attribute.String("provider_id", providerID),
		attribute.String("trigger", string(trigger)),
		attribute.String("outcome", string(outcome)),
	))
	identityAttrs := otelmetric.WithAttributes(
		attribute.String("vendor_id", vendorID),
		attribute.String("provider_id", providerID),
	)
	m.probeLastRun.Record(ctx, float64(lastRunAt.Unix()), identityAttrs)
	m.probeLastOutcome.Record(ctx, vendorProbeOutcomeValue(outcome), identityAttrs)
	m.recordLastReason(vendorID, providerID, outcome, reason)
	m.probeNextAllowed.Record(ctx, float64(nextAllowedAt.Unix()), identityAttrs)
	if value, ok := vendorAuthUsableValue(outcome); ok {
		m.authUsable.Record(ctx, value, identityAttrs)
		m.credentialLastUsed.Record(ctx, float64(lastRunAt.Unix()), identityAttrs)
	}
}

func (m *vendorObservabilityMetrics) recordLastReason(
	vendorID string,
	providerID string,
	outcome VendorObservabilityProbeOutcome,
	reason string,
) {
	if m == nil {
		return
	}
	key := vendorID + "\x00" + providerID
	reason = strings.TrimSpace(reason)
	shouldSet := (outcome == VendorObservabilityProbeOutcomeAuthBlocked || outcome == VendorObservabilityProbeOutcomeFailed) && reason != ""
	ctx := context.Background()
	m.lastReasonMu.Lock()
	previous := m.lastReasons[key]
	if previous != "" && (!shouldSet || previous != reason) {
		m.probeLastReason.Record(ctx, 0, otelmetric.WithAttributes(
			attribute.String("vendor_id", vendorID),
			attribute.String("provider_id", providerID),
			attribute.String("reason", previous),
		))
		delete(m.lastReasons, key)
	}
	if shouldSet {
		m.probeLastReason.Record(ctx, 1, otelmetric.WithAttributes(
			attribute.String("vendor_id", vendorID),
			attribute.String("provider_id", providerID),
			attribute.String("reason", reason),
		))
		m.lastReasons[key] = reason
	}
	m.lastReasonMu.Unlock()
}

func vendorProbeOutcomeValue(outcome VendorObservabilityProbeOutcome) float64 {
	switch outcome {
	case VendorObservabilityProbeOutcomeExecuted:
		return 1
	case VendorObservabilityProbeOutcomeThrottled:
		return 2
	case VendorObservabilityProbeOutcomeAuthBlocked:
		return 3
	case VendorObservabilityProbeOutcomeUnsupported:
		return 4
	case VendorObservabilityProbeOutcomeFailed:
		return 5
	default:
		return 0
	}
}

func vendorAuthUsableValue(outcome VendorObservabilityProbeOutcome) (float64, bool) {
	switch outcome {
	case VendorObservabilityProbeOutcomeExecuted:
		return 1, true
	case VendorObservabilityProbeOutcomeAuthBlocked:
		return 0, true
	default:
		return 0, false
	}
}

func (m *vendorObservabilityMetrics) recordCollectorValues(
	vendorID string,
	providerID string,
	rows []VendorObservabilityMetricRow,
) {
	if m == nil || vendorID == "" || providerID == "" || len(rows) == 0 {
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
		gauge.gauge.Record(context.Background(), row.Value, otelmetric.WithAttributes(credentialsAttributes(vendorCollectorLabels(vendorID, providerID, row.Labels))...))
	}
}

func (m *vendorObservabilityMetrics) ensureCollectedGauge(metricName string) (vendorCollectedGauge, error) {
	if m == nil {
		return vendorCollectedGauge{}, nil
	}
	metricName = strings.TrimSpace(metricName)
	if !vendorObservabilityMetricNamePattern.MatchString(metricName) {
		return vendorCollectedGauge{}, fmt.Errorf("providerobservability: invalid vendor collector metric name %q", metricName)
	}
	m.collectedMu.Lock()
	defer m.collectedMu.Unlock()
	if existing, ok := m.collectedGauges[metricName]; ok {
		return existing, nil
	}
	gauge, err := newCredentialsGauge(m.meter, metricName, "Vendor API-key active operation collected gauge value.")
	if err != nil {
		return vendorCollectedGauge{}, err
	}
	collected := vendorCollectedGauge{
		gauge: gauge,
	}
	m.collectedGauges[metricName] = collected
	return collected, nil
}

func vendorCollectorLabels(vendorID string, providerID string, rowLabels map[string]string) map[string]string {
	labels := map[string]string{
		ownerKindLabel: ownerKindVendor,
		ownerIDLabel:   vendorID,
		"vendor_id":    vendorID,
		"provider_id":  providerID,
	}
	for key, value := range sanitizeVendorCollectorLabels(rowLabels) {
		labels[key] = value
	}
	return labels
}

func sanitizeVendorCollectorLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	sanitized := map[string]string{}
	for key, value := range labels {
		trimmedKey := platformtelemetry.StorageMetricName(strings.TrimSpace(key))
		if trimmedKey == "" ||
			trimmedKey == ownerKindLabel ||
			trimmedKey == ownerIDLabel ||
			trimmedKey == "vendor_id" ||
			trimmedKey == "provider_id" ||
			trimmedKey == "provider_surface_binding_id" ||
			trimmedKey == "instance_id" {
			continue
		}
		if !vendorObservabilityLabelNamePattern.MatchString(trimmedKey) {
			continue
		}
		sanitized[trimmedKey] = strings.TrimSpace(value)
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}
