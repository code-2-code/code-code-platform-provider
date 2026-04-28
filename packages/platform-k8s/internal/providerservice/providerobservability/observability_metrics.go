package providerobservability

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	platformtelemetry "code-code.internal/platform-k8s/internal/platform/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// observabilityMetrics holds OTel instruments for one observability runner family.
// The ownerLabel differentiates "cli_id" (OAuth) from "vendor_id" (vendor API-key).
type observabilityMetrics struct {
	ownerLabel     string // "cli_id" or "vendor_id"
	meter          otelmetric.Meter
	probeRuns      otelmetric.Int64Counter
	probeLastRun   otelmetric.Float64Gauge
	probeLastOutcome otelmetric.Float64Gauge
	probeLastReason  otelmetric.Float64Gauge
	probeNextAllowed otelmetric.Float64Gauge
	authUsable       otelmetric.Float64Gauge
	credentialLastUsed otelmetric.Float64Gauge

	lastReasonMu sync.Mutex
	lastReasons  map[string]string

	collectedMu     sync.Mutex
	collectedGauges map[string]collectedGauge
}

type collectedGauge struct {
	gauge otelmetric.Float64Gauge
}

var (
	observabilityMetricNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.\-/]{0,254}$`)
	observabilityLabelNamePattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// newObservabilityMetrics creates an observabilityMetrics instance with the given metric
// name prefix (e.g. "gen_ai.provider.cli.oauth" or "gen_ai.provider.vendor.api_key")
// and owner label (e.g. "cli_id" or "vendor_id").
func newObservabilityMetrics(metricPrefix string, ownerLabel string) (*observabilityMetrics, error) {
	meter := otel.Meter("platform-k8s/providerobservability")
	probeRuns, err := newCredentialsCounter(
		meter,
		metricPrefix+".active.operation.runs.total",
		"Count of active observability probes.",
	)
	if err != nil {
		return nil, err
	}
	probeLastRun, err := newCredentialsGauge(
		meter,
		metricPrefix+".active.operation.last.run.timestamp.seconds",
		"Unix timestamp of the last active observability operation.",
	)
	if err != nil {
		return nil, err
	}
	probeLastOutcome, err := newCredentialsGauge(
		meter,
		metricPrefix+".active.operation.last.outcome",
		"Numeric code of the last active observability operation outcome.",
	)
	if err != nil {
		return nil, err
	}
	probeLastReason, err := newCredentialsGauge(
		meter,
		metricPrefix+".active.operation.last.reason",
		"Machine-readable reason for the last decisive active observability operation.",
	)
	if err != nil {
		return nil, err
	}
	probeNextAllowed, err := newCredentialsGauge(
		meter,
		metricPrefix+".active.operation.next.allowed.timestamp.seconds",
		"Unix timestamp for the next allowed active observability operation.",
	)
	if err != nil {
		return nil, err
	}
	authUsable, err := newCredentialsGauge(
		meter,
		metricPrefix+".active.operation.auth.usable",
		"Whether the last decisive active operation confirmed usable auth (1) or auth_blocked (0).",
	)
	if err != nil {
		return nil, err
	}
	credentialLastUsed, err := newCredentialsGauge(
		meter,
		metricPrefix+".credential.last.used.timestamp.seconds",
		"Unix timestamp of the last credential use against the provider.",
	)
	if err != nil {
		return nil, err
	}
	return &observabilityMetrics{
		ownerLabel:         ownerLabel,
		meter:              meter,
		probeRuns:          probeRuns,
		probeLastRun:       probeLastRun,
		probeLastOutcome:   probeLastOutcome,
		probeLastReason:    probeLastReason,
		probeNextAllowed:   probeNextAllowed,
		authUsable:         authUsable,
		credentialLastUsed: credentialLastUsed,
		lastReasons:        map[string]string{},
		collectedGauges:    map[string]collectedGauge{},
	}, nil
}

func newCredentialsCounter(meter otelmetric.Meter, name string, description string) (otelmetric.Int64Counter, error) {
	counter, err := meter.Int64Counter(name, otelmetric.WithDescription(description), otelmetric.WithUnit("1"))
	if err != nil {
		return nil, fmt.Errorf("providerobservability: create counter %q: %w", name, err)
	}
	return counter, nil
}

func newCredentialsGauge(meter otelmetric.Meter, name string, description string) (otelmetric.Float64Gauge, error) {
	gauge, err := meter.Float64Gauge(name, otelmetric.WithDescription(description))
	if err != nil {
		return nil, fmt.Errorf("providerobservability: create gauge %q: %w", name, err)
	}
	return gauge, nil
}

func (m *observabilityMetrics) record(
	ownerID string,
	providerID string,
	trigger Trigger,
	outcome ProbeOutcome,
	reason string,
	lastRunAt time.Time,
	nextAllowedAt time.Time,
) {
	if m == nil {
		return
	}
	if ownerID == "" || providerID == "" {
		return
	}
	ctx := context.Background()
	m.probeRuns.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String(m.ownerLabel, ownerID),
		attribute.String("provider_id", providerID),
		attribute.String("trigger", string(trigger)),
		attribute.String("outcome", string(outcome)),
	))
	identityAttrs := otelmetric.WithAttributes(
		attribute.String(m.ownerLabel, ownerID),
		attribute.String("provider_id", providerID),
	)
	m.probeLastRun.Record(ctx, float64(lastRunAt.Unix()), identityAttrs)
	m.probeLastOutcome.Record(ctx, probeOutcomeValue(outcome), identityAttrs)
	m.recordLastReason(ownerID, providerID, outcome, reason)
	m.probeNextAllowed.Record(ctx, float64(nextAllowedAt.Unix()), identityAttrs)
	if value, ok := authUsableValue(outcome); ok {
		m.authUsable.Record(ctx, value, identityAttrs)
		m.credentialLastUsed.Record(ctx, float64(lastRunAt.Unix()), identityAttrs)
	}
}

func (m *observabilityMetrics) recordLastReason(
	ownerID string,
	providerID string,
	outcome ProbeOutcome,
	reason string,
) {
	if m == nil {
		return
	}
	key := ownerID + "\x00" + providerID
	reason = strings.TrimSpace(reason)
	shouldSet := (outcome == ProbeOutcomeAuthBlocked || outcome == ProbeOutcomeFailed) && reason != ""
	ctx := context.Background()
	m.lastReasonMu.Lock()
	previous := m.lastReasons[key]
	if previous != "" && (!shouldSet || previous != reason) {
		m.probeLastReason.Record(ctx, 0, otelmetric.WithAttributes(
			attribute.String(m.ownerLabel, ownerID),
			attribute.String("provider_id", providerID),
			attribute.String("reason", previous),
		))
		delete(m.lastReasons, key)
	}
	if shouldSet {
		m.probeLastReason.Record(ctx, 1, otelmetric.WithAttributes(
			attribute.String(m.ownerLabel, ownerID),
			attribute.String("provider_id", providerID),
			attribute.String("reason", reason),
		))
		m.lastReasons[key] = reason
	}
	m.lastReasonMu.Unlock()
}

func probeOutcomeValue(outcome ProbeOutcome) float64 {
	switch outcome {
	case ProbeOutcomeExecuted:
		return 1
	case ProbeOutcomeThrottled:
		return 2
	case ProbeOutcomeAuthBlocked:
		return 3
	case ProbeOutcomeUnsupported:
		return 4
	case ProbeOutcomeFailed:
		return 5
	default:
		return 0
	}
}

func authUsableValue(outcome ProbeOutcome) (float64, bool) {
	switch outcome {
	case ProbeOutcomeExecuted:
		return 1, true
	case ProbeOutcomeAuthBlocked:
		return 0, true
	default:
		return 0, false
	}
}

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
		ownerKindLabel:  m.ownerKindValue(),
		ownerIDLabel:    ownerID,
		m.ownerLabel:    ownerID,
		"provider_id":   providerID,
	}
	for key, value := range m.sanitizeCollectorLabels(rowLabels) {
		labels[key] = value
	}
	return labels
}

func (m *observabilityMetrics) ownerKindValue() string {
	switch m.ownerLabel {
	case "cli_id":
		return ownerKindCLI
	default:
		return ownerKindVendor
	}
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
			trimmedKey == "provider_surface_binding_id" ||
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
