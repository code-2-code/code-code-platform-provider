package providerorchestration

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type providerProbeMetrics struct {
	runs        otelmetric.Int64Counter
	lastRun     otelmetric.Float64Gauge
	nextAllowed otelmetric.Float64Gauge
	lastOutcome otelmetric.Float64Gauge
}

var (
	providerProbeMetricsOnce sync.Once
	providerProbeMetricsInst *providerProbeMetrics
	providerProbeMetricsErr  error
)

func getProviderProbeMetrics() (*providerProbeMetrics, error) {
	providerProbeMetricsOnce.Do(func() {
		meter := otel.Meter("platform-k8s/providerorchestration")
		runs, err := meter.Int64Counter(
			"gen_ai.provider.probe.runs.total",
			otelmetric.WithDescription("Count of provider probe tasks coordinated by provider orchestration."),
			otelmetric.WithUnit("1"),
		)
		if err != nil {
			providerProbeMetricsErr = err
			return
		}
		lastRun, err := meter.Float64Gauge(
			"gen_ai.provider.probe.last.run.timestamp.seconds",
			otelmetric.WithDescription("Unix timestamp of the last provider probe task."),
			otelmetric.WithUnit("s"),
		)
		if err != nil {
			providerProbeMetricsErr = err
			return
		}
		nextAllowed, err := meter.Float64Gauge(
			"gen_ai.provider.probe.next.allowed.timestamp.seconds",
			otelmetric.WithDescription("Unix timestamp when provider orchestration may run the next probe task."),
			otelmetric.WithUnit("s"),
		)
		if err != nil {
			providerProbeMetricsErr = err
			return
		}
		lastOutcome, err := meter.Float64Gauge(
			"gen_ai.provider.probe.last.outcome",
			otelmetric.WithDescription("Numeric code for the last provider probe task outcome."),
			otelmetric.WithUnit("1"),
		)
		if err != nil {
			providerProbeMetricsErr = err
			return
		}
		providerProbeMetricsInst = &providerProbeMetrics{
			runs:        runs,
			lastRun:     lastRun,
			nextAllowed: nextAllowed,
			lastOutcome: lastOutcome,
		}
	})
	return providerProbeMetricsInst, providerProbeMetricsErr
}

func recordProviderProbeMetric(input ProviderProbeStatusInput, lastRunAt time.Time, nextAllowedAt time.Time) {
	metrics, err := getProviderProbeMetrics()
	if err != nil || metrics == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider_id", strings.TrimSpace(input.ProviderID)),
		attribute.String("probe_kind", string(input.Kind)),
		attribute.String("trigger", input.Trigger.String()),
		attribute.String("outcome", strings.TrimSpace(input.Outcome)),
	}
	options := otelmetric.WithAttributes(attrs...)
	ctx := context.Background()
	metrics.runs.Add(ctx, 1, options)
	identity := otelmetric.WithAttributes(
		attribute.String("provider_id", strings.TrimSpace(input.ProviderID)),
		attribute.String("probe_kind", string(input.Kind)),
	)
	metrics.lastRun.Record(ctx, float64(lastRunAt.Unix()), identity)
	metrics.nextAllowed.Record(ctx, float64(nextAllowedAt.Unix()), identity)
	metrics.lastOutcome.Record(ctx, providerProbeOutcomeValue(input.Outcome), identity)
}

func providerProbeOutcomeValue(outcome string) float64 {
	switch strings.TrimSpace(outcome) {
	case "executed":
		return 1
	case "throttled":
		return 2
	case "auth_blocked":
		return 3
	case "unsupported":
		return 4
	case "failed":
		return 5
	default:
		return 0
	}
}
