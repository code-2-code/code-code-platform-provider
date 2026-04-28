package providercatalogs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"code-code.internal/go-contract/domainerror"
	modelcatalogdiscoveryv1 "code-code.internal/go-contract/model_catalog_discovery/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	catalogProbeRunsMetricName        = "gen_ai.model_catalog.probe.runs.total"
	catalogProbeDurationMetricName    = "gen_ai.model_catalog.probe.duration.seconds"
	catalogProbeModelsMetricName      = "gen_ai.model_catalog.probe.models.count"
	catalogProbeLastRunMetricName     = "gen_ai.model_catalog.probe.last.run.timestamp.seconds"
	catalogProbeLastOutcomeMetricName = "gen_ai.model_catalog.probe.last.outcome"
)

type catalogProbeMetrics struct {
	runs        otelmetric.Int64Counter
	duration    otelmetric.Float64Histogram
	models      otelmetric.Int64Histogram
	lastRun     otelmetric.Float64Gauge
	lastOutcome otelmetric.Float64Gauge
}

var (
	registerCatalogProbeMetricsOnce sync.Once
	registeredCatalogProbeMetrics   *catalogProbeMetrics
	registerCatalogProbeMetricsErr  error
)

func registerCatalogProbeMetrics() (*catalogProbeMetrics, error) {
	registerCatalogProbeMetricsOnce.Do(func() {
		meter := otel.Meter("platform-k8s/providercatalogs")
		runs, err := meter.Int64Counter(
			catalogProbeRunsMetricName,
			otelmetric.WithDescription("Count of model catalog probe executions."),
		)
		if err != nil {
			registerCatalogProbeMetricsErr = err
			return
		}
		duration, err := meter.Float64Histogram(
			catalogProbeDurationMetricName,
			otelmetric.WithUnit("s"),
			otelmetric.WithDescription("Duration of model catalog probe executions."),
		)
		if err != nil {
			registerCatalogProbeMetricsErr = err
			return
		}
		models, err := meter.Int64Histogram(
			catalogProbeModelsMetricName,
			otelmetric.WithUnit("{model}"),
			otelmetric.WithDescription("Number of model IDs discovered by successful model catalog probes."),
		)
		if err != nil {
			registerCatalogProbeMetricsErr = err
			return
		}
		lastRun, err := meter.Float64Gauge(
			catalogProbeLastRunMetricName,
			otelmetric.WithUnit("s"),
			otelmetric.WithDescription("Unix timestamp of the last model catalog probe execution."),
		)
		if err != nil {
			registerCatalogProbeMetricsErr = err
			return
		}
		lastOutcome, err := meter.Float64Gauge(
			catalogProbeLastOutcomeMetricName,
			otelmetric.WithDescription("Numeric code for the last model catalog probe outcome."),
		)
		if err != nil {
			registerCatalogProbeMetricsErr = err
			return
		}
		registeredCatalogProbeMetrics = &catalogProbeMetrics{
			runs:        runs,
			duration:    duration,
			models:      models,
			lastRun:     lastRun,
			lastOutcome: lastOutcome,
		}
	})
	if registerCatalogProbeMetricsErr != nil {
		return nil, registerCatalogProbeMetricsErr
	}
	return registeredCatalogProbeMetrics, nil
}

func (m *catalogProbeMetrics) record(
	request CatalogProbeRequest,
	operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation,
	modelCount int,
	started time.Time,
	err error,
) {
	if m == nil {
		return
	}
	outcome := "success"
	if err != nil {
		outcome = "failed"
	}
	attrs := catalogProbeAttributes(request, operation, outcome, err)
	options := otelmetric.WithAttributes(attrs...)
	ctx := context.Background()
	m.runs.Add(ctx, 1, options)
	m.duration.Record(ctx, time.Since(started).Seconds(), options)
	if err == nil {
		m.models.Record(ctx, int64(modelCount), options)
	}
	identityOptions := otelmetric.WithAttributes(catalogProbeIdentityAttributes(request, operation)...)
	m.lastRun.Record(ctx, float64(time.Now().Unix()), identityOptions)
	m.lastOutcome.Record(ctx, catalogProbeOutcomeValue(outcome), identityOptions)
}

func catalogProbeAttributes(
	request CatalogProbeRequest,
	operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation,
	outcome string,
	err error,
) []attribute.KeyValue {
	attrs := append(catalogProbeIdentityAttributes(request, operation),
		attribute.String("outcome", outcome),
	)
	if kind := catalogProbeErrorKind(err); kind != "" {
		attrs = append(attrs, attribute.String("error_kind", kind))
	}
	return attrs
}

func catalogProbeIdentityAttributes(
	request CatalogProbeRequest,
	operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation,
) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("probe_id", catalogProbeID(request)),
		attribute.String("protocol", catalogProbeProtocol(request)),
		attribute.String("auth", catalogProbeAuthKind(request, operation)),
		attribute.String("response_kind", operation.GetResponseKind().String()),
	}
}

func catalogProbeID(request CatalogProbeRequest) string {
	if probeID := strings.TrimSpace(request.ProbeID); probeID != "" {
		return probeID
	}
	if key := strings.TrimSpace(request.ConcurrencyKey); key != "" {
		return key
	}
	return "unknown"
}

func catalogProbeProtocol(request CatalogProbeRequest) string {
	protocol := strings.TrimSpace(request.Protocol.String())
	if protocol == "" {
		return "PROTOCOL_UNSPECIFIED"
	}
	return protocol
}

func catalogProbeAuthKind(
	request CatalogProbeRequest,
	operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation,
) string {
	if operationUsesCredential(operation, request) {
		return "credential"
	}
	return "anonymous"
}

func catalogProbeErrorKind(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	}
	var validationErr *domainerror.ValidationError
	if errors.As(err, &validationErr) {
		return "validation"
	}
	return "error"
}

func catalogProbeOutcomeValue(outcome string) float64 {
	if outcome == "success" {
		return 1
	}
	return 2
}
