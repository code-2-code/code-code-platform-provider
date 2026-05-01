package providerobservability

import (
	"context"
	"testing"

	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func newTestMeter(t *testing.T) (otelmetric.Meter, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	return provider.Meter("credentials-test"), reader
}

func mustTestGauge(t *testing.T, meter otelmetric.Meter, name string) otelmetric.Float64Gauge {
	t.Helper()
	gauge, err := newCredentialsGauge(meter, name, "test")
	if err != nil {
		t.Fatalf("newCredentialsGauge() error = %v", err)
	}
	return gauge
}

func TestSanitizeCollectorLabelsDropsInstanceLabels(t *testing.T) {
	t.Parallel()

	m := &observabilityMetrics{ownerLabel: "cli_id"}
	labels := m.sanitizeCollectorLabels(map[string]string{
		"surface_id":  "instance-1",
		"instance_id": "instance-1",
		"model_id":    "gemini-2.5-pro",
	})
	if _, ok := labels["surface_id"]; ok {
		t.Fatal("surface_id should be dropped")
	}
	if _, ok := labels["instance_id"]; ok {
		t.Fatal("instance_id should be dropped")
	}
	if got, want := labels["model_id"], "gemini-2.5-pro"; got != want {
		t.Fatalf("model_id = %q, want %q", got, want)
	}
}
