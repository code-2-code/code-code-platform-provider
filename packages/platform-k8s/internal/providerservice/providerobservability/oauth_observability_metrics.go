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

const (
	oauthObservabilityProbeRunsMetricName        = "gen_ai.provider.cli.oauth.active.operation.runs.total"
	oauthObservabilityProbeLastRunMetricName     = "gen_ai.provider.cli.oauth.active.operation.last.run.timestamp.seconds"
	oauthObservabilityProbeLastOutcomeMetricName = "gen_ai.provider.cli.oauth.active.operation.last.outcome"
	oauthObservabilityProbeLastReasonMetricName  = "gen_ai.provider.cli.oauth.active.operation.last.reason"
	oauthObservabilityProbeNextAllowedMetricName = "gen_ai.provider.cli.oauth.active.operation.next.allowed.timestamp.seconds"
	oauthObservabilityAuthUsableMetricName       = "gen_ai.provider.cli.oauth.active.operation.auth.usable"
	oauthCredentialLastUsedMetricName            = "gen_ai.provider.cli.oauth.credential.last.used.timestamp.seconds"
)

type oauthObservabilityMetrics struct {
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
	collectedGauges map[string]oauthCollectedGauge
}

type oauthCollectedGauge struct {
	gauge otelmetric.Float64Gauge
}

var (
	registerOAuthObservabilityMetricsOnce sync.Once
	registeredOAuthObservabilityMetrics   *oauthObservabilityMetrics
	registerOAuthObservabilityMetricsErr  error
	oauthObservabilityMetricNamePattern   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.\-/]{0,254}$`)
	oauthObservabilityLabelNamePattern    = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

func registerOAuthObservabilityMetrics() (*oauthObservabilityMetrics, error) {
	registerOAuthObservabilityMetricsOnce.Do(func() {
		meter := otel.Meter("platform-k8s/providerobservability")
		probeRuns, err := newCredentialsCounter(
			meter,
			oauthObservabilityProbeRunsMetricName,
			"Count of CLI OAuth active observability probes.",
		)
		if err != nil {
			registerOAuthObservabilityMetricsErr = err
			return
		}
		probeLastRun, err := newCredentialsGauge(
			meter,
			oauthObservabilityProbeLastRunMetricName,
			"Unix timestamp of the last CLI OAuth active observability operation.",
		)
		if err != nil {
			registerOAuthObservabilityMetricsErr = err
			return
		}
		probeLastOutcome, err := newCredentialsGauge(
			meter,
			oauthObservabilityProbeLastOutcomeMetricName,
			"Numeric code of the last CLI OAuth active observability operation outcome.",
		)
		if err != nil {
			registerOAuthObservabilityMetricsErr = err
			return
		}
		probeLastReason, err := newCredentialsGauge(
			meter,
			oauthObservabilityProbeLastReasonMetricName,
			"Machine-readable reason for the last decisive CLI OAuth active observability operation.",
		)
		if err != nil {
			registerOAuthObservabilityMetricsErr = err
			return
		}
		probeNextAllowed, err := newCredentialsGauge(
			meter,
			oauthObservabilityProbeNextAllowedMetricName,
			"Unix timestamp for the next allowed CLI OAuth active observability operation.",
		)
		if err != nil {
			registerOAuthObservabilityMetricsErr = err
			return
		}
		authUsable, err := newCredentialsGauge(
			meter,
			oauthObservabilityAuthUsableMetricName,
			"Whether the last decisive CLI OAuth active operation confirmed usable auth (1) or auth_blocked (0).",
		)
		if err != nil {
			registerOAuthObservabilityMetricsErr = err
			return
		}
		credentialLastUsed, err := newCredentialsGauge(
			meter,
			oauthCredentialLastUsedMetricName,
			"Unix timestamp of the last CLI OAuth credential use against the provider.",
		)
		if err != nil {
			registerOAuthObservabilityMetricsErr = err
			return
		}
		metrics := &oauthObservabilityMetrics{
			meter:              meter,
			probeRuns:          probeRuns,
			probeLastRun:       probeLastRun,
			probeLastOutcome:   probeLastOutcome,
			probeLastReason:    probeLastReason,
			probeNextAllowed:   probeNextAllowed,
			authUsable:         authUsable,
			credentialLastUsed: credentialLastUsed,
			lastReasons:        map[string]string{},
			collectedGauges:    map[string]oauthCollectedGauge{},
		}
		registeredOAuthObservabilityMetrics = metrics
	})
	if registerOAuthObservabilityMetricsErr != nil {
		return nil, registerOAuthObservabilityMetricsErr
	}
	return registeredOAuthObservabilityMetrics, nil
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

func (m *oauthObservabilityMetrics) record(
	cliID string,
	providerID string,
	trigger OAuthObservabilityProbeTrigger,
	outcome OAuthObservabilityProbeOutcome,
	reason string,
	lastRunAt time.Time,
	nextAllowedAt time.Time,
) {
	if m == nil {
		return
	}
	if cliID == "" || providerID == "" {
		return
	}
	ctx := context.Background()
	m.probeRuns.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String("cli_id", cliID),
		attribute.String("provider_id", providerID),
		attribute.String("trigger", string(trigger)),
		attribute.String("outcome", string(outcome)),
	))
	identityAttrs := otelmetric.WithAttributes(
		attribute.String("cli_id", cliID),
		attribute.String("provider_id", providerID),
	)
	m.probeLastRun.Record(ctx, float64(lastRunAt.Unix()), identityAttrs)
	m.probeLastOutcome.Record(ctx, oauthProbeOutcomeValue(outcome), identityAttrs)
	m.recordLastReason(cliID, providerID, outcome, reason)
	m.probeNextAllowed.Record(ctx, float64(nextAllowedAt.Unix()), identityAttrs)
	if value, ok := oauthAuthUsableValue(outcome); ok {
		m.authUsable.Record(ctx, value, identityAttrs)
		m.credentialLastUsed.Record(ctx, float64(lastRunAt.Unix()), identityAttrs)
	}
}

func (m *oauthObservabilityMetrics) recordLastReason(
	cliID string,
	providerID string,
	outcome OAuthObservabilityProbeOutcome,
	reason string,
) {
	if m == nil {
		return
	}
	key := cliID + "\x00" + providerID
	reason = strings.TrimSpace(reason)
	shouldSet := (outcome == OAuthObservabilityProbeOutcomeAuthBlocked || outcome == OAuthObservabilityProbeOutcomeFailed) && reason != ""
	ctx := context.Background()
	m.lastReasonMu.Lock()
	previous := m.lastReasons[key]
	if previous != "" && (!shouldSet || previous != reason) {
		m.probeLastReason.Record(ctx, 0, otelmetric.WithAttributes(
			attribute.String("cli_id", cliID),
			attribute.String("provider_id", providerID),
			attribute.String("reason", previous),
		))
		delete(m.lastReasons, key)
	}
	if shouldSet {
		m.probeLastReason.Record(ctx, 1, otelmetric.WithAttributes(
			attribute.String("cli_id", cliID),
			attribute.String("provider_id", providerID),
			attribute.String("reason", reason),
		))
		m.lastReasons[key] = reason
	}
	m.lastReasonMu.Unlock()
}

func oauthProbeOutcomeValue(outcome OAuthObservabilityProbeOutcome) float64 {
	switch outcome {
	case OAuthObservabilityProbeOutcomeExecuted:
		return 1
	case OAuthObservabilityProbeOutcomeThrottled:
		return 2
	case OAuthObservabilityProbeOutcomeAuthBlocked:
		return 3
	case OAuthObservabilityProbeOutcomeUnsupported:
		return 4
	case OAuthObservabilityProbeOutcomeFailed:
		return 5
	default:
		return 0
	}
}

func oauthAuthUsableValue(outcome OAuthObservabilityProbeOutcome) (float64, bool) {
	switch outcome {
	case OAuthObservabilityProbeOutcomeExecuted:
		return 1, true
	case OAuthObservabilityProbeOutcomeAuthBlocked:
		return 0, true
	default:
		return 0, false
	}
}

func (m *oauthObservabilityMetrics) recordCollectorValues(cliID, providerID string, rows []OAuthObservabilityMetricRow) {
	if m == nil || strings.TrimSpace(cliID) == "" || strings.TrimSpace(providerID) == "" || len(rows) == 0 {
		return
	}
	for _, row := range rows {
		metricName := strings.TrimSpace(row.MetricName)
		if metricName == "" {
			continue
		}
		labels := sanitizeCollectorLabels(row.Labels)
		gauge, err := m.ensureCollectedGauge(metricName)
		if err != nil || gauge.gauge == nil {
			continue
		}
		if labels == nil {
			labels = map[string]string{}
		}
		labels[ownerKindLabel] = ownerKindCLI
		labels[ownerIDLabel] = cliID
		labels["cli_id"] = cliID
		labels["provider_id"] = providerID
		gauge.gauge.Record(context.Background(), row.Value, otelmetric.WithAttributes(credentialsAttributes(labels)...))
	}
}

func (m *oauthObservabilityMetrics) ensureCollectedGauge(metricName string) (oauthCollectedGauge, error) {
	if m == nil {
		return oauthCollectedGauge{}, nil
	}
	metricName = strings.TrimSpace(metricName)
	if !oauthObservabilityMetricNamePattern.MatchString(metricName) {
		return oauthCollectedGauge{}, fmt.Errorf("providerobservability: invalid collector metric name %q", metricName)
	}
	m.collectedMu.Lock()
	defer m.collectedMu.Unlock()
	if existing, ok := m.collectedGauges[metricName]; ok {
		return existing, nil
	}
	gauge, err := newCredentialsGauge(m.meter, metricName, "CLI OAuth active operation collected gauge value.")
	if err != nil {
		return oauthCollectedGauge{}, err
	}
	collected := oauthCollectedGauge{
		gauge: gauge,
	}
	m.collectedGauges[metricName] = collected
	return collected, nil
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

func sanitizeCollectorLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	sanitized := map[string]string{}
	for key, value := range labels {
		trimmedKey := platformtelemetry.StorageMetricName(strings.TrimSpace(key))
		if trimmedKey == "" ||
			trimmedKey == ownerKindLabel ||
			trimmedKey == ownerIDLabel ||
			trimmedKey == "cli_id" ||
			trimmedKey == "provider_id" ||
			trimmedKey == "provider_surface_binding_id" ||
			trimmedKey == "instance_id" {
			continue
		}
		if !oauthObservabilityLabelNamePattern.MatchString(trimmedKey) {
			continue
		}
		sanitized[trimmedKey] = strings.TrimSpace(value)
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}
