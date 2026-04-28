package providercatalogs

import (
	"context"
	"testing"
	"time"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	"code-code.internal/go-contract/domainerror"
	modelcatalogdiscoveryv1 "code-code.internal/go-contract/model_catalog_discovery/v1"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestCatalogProbeMetricsRecordLowCardinalityAttributes(t *testing.T) {
	t.Parallel()

	metrics, reader := newTestCatalogProbeMetrics(t)
	metrics.record(CatalogProbeRequest{
		ProbeID:                  "vendor.openai",
		Protocol:                 apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
		BaseURL:                  "https://api.openai.com/v1",
		ProviderSurfaceBindingID: "surface-1",
	}, &modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation{
		ResponseKind: modelcatalogdiscoveryv1.ModelCatalogDiscoveryResponseKind_MODEL_CATALOG_DISCOVERY_RESPONSE_KIND_OPENAI_MODELS,
		Security: []*modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityRequirement{{
			Schemes: []modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme{
				modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme_MODEL_CATALOG_DISCOVERY_SECURITY_SCHEME_API_KEY,
			},
		}},
	}, 3, time.Now().Add(-50*time.Millisecond), nil)

	points := observedInt64SumPoints(t, reader, "model.probe.runs.test")
	if len(points) != 1 {
		t.Fatalf("run points = %d, want 1", len(points))
	}
	labels := metricAttributeSet(points[0].Attributes.ToSlice())
	if got, want := labels["probe_id"], "vendor.openai"; got != want {
		t.Fatalf("probe_id = %q, want %q", got, want)
	}
	if got, want := labels["auth"], "credential"; got != want {
		t.Fatalf("auth = %q, want %q", got, want)
	}
	if got, want := labels["outcome"], "success"; got != want {
		t.Fatalf("outcome = %q, want %q", got, want)
	}
	if _, exists := labels["base_url"]; exists {
		t.Fatal("base_url label should not be exported")
	}
	if _, exists := labels["credential_id"]; exists {
		t.Fatal("credential_id label should not be exported")
	}
}

func TestCatalogProbeMetricsRecordValidationFailures(t *testing.T) {
	t.Parallel()

	metrics, reader := newTestCatalogProbeMetrics(t)
	metrics.record(CatalogProbeRequest{
		ProbeID: "cli.codex.oauth",
	}, &modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation{
		ResponseKind: modelcatalogdiscoveryv1.ModelCatalogDiscoveryResponseKind_MODEL_CATALOG_DISCOVERY_RESPONSE_KIND_CODEX_MODELS,
	}, 0, time.Now().Add(-10*time.Millisecond), domainerror.NewValidation("bad response"))

	points := observedInt64SumPoints(t, reader, "model.probe.runs.test")
	if len(points) != 1 {
		t.Fatalf("run points = %d, want 1", len(points))
	}
	labels := metricAttributeSet(points[0].Attributes.ToSlice())
	if got, want := labels["outcome"], "failed"; got != want {
		t.Fatalf("outcome = %q, want %q", got, want)
	}
	if got, want := labels["error_kind"], "validation"; got != want {
		t.Fatalf("error_kind = %q, want %q", got, want)
	}
}

func newTestCatalogProbeMetrics(t *testing.T) (*catalogProbeMetrics, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter("model-probe-test")
	return &catalogProbeMetrics{
		runs:        mustTestInt64Counter(t, meter, "model.probe.runs.test"),
		duration:    mustTestFloat64Histogram(t, meter, "model.probe.duration.test"),
		models:      mustTestInt64Histogram(t, meter, "model.probe.models.test"),
		lastRun:     mustTestFloat64Gauge(t, meter, "model.probe.last.run.test"),
		lastOutcome: mustTestFloat64Gauge(t, meter, "model.probe.last.outcome.test"),
	}, reader
}

func mustTestInt64Counter(t *testing.T, meter otelmetric.Meter, name string) otelmetric.Int64Counter {
	t.Helper()
	counter, err := meter.Int64Counter(name)
	if err != nil {
		t.Fatalf("Int64Counter(%q) error = %v", name, err)
	}
	return counter
}

func mustTestFloat64Histogram(t *testing.T, meter otelmetric.Meter, name string) otelmetric.Float64Histogram {
	t.Helper()
	histogram, err := meter.Float64Histogram(name)
	if err != nil {
		t.Fatalf("Float64Histogram(%q) error = %v", name, err)
	}
	return histogram
}

func mustTestInt64Histogram(t *testing.T, meter otelmetric.Meter, name string) otelmetric.Int64Histogram {
	t.Helper()
	histogram, err := meter.Int64Histogram(name)
	if err != nil {
		t.Fatalf("Int64Histogram(%q) error = %v", name, err)
	}
	return histogram
}

func mustTestFloat64Gauge(t *testing.T, meter otelmetric.Meter, name string) otelmetric.Float64Gauge {
	t.Helper()
	gauge, err := meter.Float64Gauge(name)
	if err != nil {
		t.Fatalf("Float64Gauge(%q) error = %v", name, err)
	}
	return gauge
}

func observedInt64SumPoints(t *testing.T, reader *sdkmetric.ManualReader, name string) []metricdata.DataPoint[int64] {
	t.Helper()
	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &resourceMetrics); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		for _, observed := range scopeMetrics.Metrics {
			if observed.Name != name {
				continue
			}
			sum, ok := observed.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("metric %q data = %T, want int64 sum", name, observed.Data)
			}
			return sum.DataPoints
		}
	}
	t.Fatalf("metric %q not found", name)
	return nil
}

func metricAttributeSet(attrs []attribute.KeyValue) map[string]string {
	labels := map[string]string{}
	for _, attr := range attrs {
		labels[string(attr.Key)] = attr.Value.AsString()
	}
	return labels
}
