package providerobservability

import (
	"context"
	"testing"
	"time"

	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestOAuthObservabilityMetricsRecordAuthUsable(t *testing.T) {
	t.Parallel()

	metrics, reader := newTestOAuthObservabilityMetrics(t)
	now := time.Unix(1700000000, 0).UTC()

	metrics.record("gemini", "account-a", TriggerManual, ProbeOutcomeExecuted, "", now, now.Add(time.Minute))
	if got := observedMetricValue(t, reader, "oauth.auth.usable.test"); got != 1 {
		t.Fatalf("authUsable after executed = %v, want 1", got)
	}
	if got := observedMetricValue(t, reader, "oauth.probe.last.outcome.test"); got != 1 {
		t.Fatalf("lastOutcome after executed = %v, want 1", got)
	}
	if got, want := observedMetricValue(t, reader, "oauth.credential.last.used.test"), float64(now.Unix()); got != want {
		t.Fatalf("credentialLastUsed after executed = %v, want %v", got, want)
	}

	authBlockedAt := now.Add(time.Minute)
	metrics.record("gemini", "account-a", TriggerManual, ProbeOutcomeAuthBlocked, "INVALID_TOKEN", authBlockedAt, authBlockedAt.Add(time.Minute))
	if got := observedMetricValue(t, reader, "oauth.auth.usable.test"); got != 0 {
		t.Fatalf("authUsable after auth_blocked = %v, want 0", got)
	}
	if got := observedMetricValue(t, reader, "oauth.probe.last.outcome.test"); got != 3 {
		t.Fatalf("lastOutcome after auth_blocked = %v, want 3", got)
	}
	if got, want := observedMetricValue(t, reader, "oauth.credential.last.used.test"), float64(authBlockedAt.Unix()); got != want {
		t.Fatalf("credentialLastUsed after auth_blocked = %v, want %v", got, want)
	}

	failedAt := authBlockedAt.Add(time.Minute)
	metrics.record("gemini", "account-a", TriggerManual, ProbeOutcomeFailed, "KUBERNETES_API_UNAVAILABLE", failedAt, failedAt.Add(time.Minute))
	if got := observedMetricValue(t, reader, "oauth.auth.usable.test"); got != 0 {
		t.Fatalf("authUsable after failed = %v, want previous 0", got)
	}
	if got := observedMetricValue(t, reader, "oauth.probe.last.outcome.test"); got != 5 {
		t.Fatalf("lastOutcome after failed = %v, want 5", got)
	}
	if got, want := observedMetricValue(t, reader, "oauth.credential.last.used.test"), float64(authBlockedAt.Unix()); got != want {
		t.Fatalf("credentialLastUsed after failed = %v, want previous %v", got, want)
	}
}

func newTestOAuthObservabilityMetrics(t *testing.T) (*observabilityMetrics, *sdkmetric.ManualReader) {
	t.Helper()
	meter, reader := newTestMeter(t)
	return &observabilityMetrics{
		ownerLabel:         "cli_id",
		meter:              meter,
		probeRuns:          mustTestCounter(t, meter, "oauth.probe.runs.test"),
		probeLastRun:       mustTestGauge(t, meter, "oauth.probe.last.run.test"),
		probeLastOutcome:   mustTestGauge(t, meter, "oauth.probe.last.outcome.test"),
		probeLastReason:    mustTestGauge(t, meter, "oauth.probe.last.reason.test"),
		probeNextAllowed:   mustTestGauge(t, meter, "oauth.probe.next.allowed.test"),
		authUsable:         mustTestGauge(t, meter, "oauth.auth.usable.test"),
		credentialLastUsed: mustTestGauge(t, meter, "oauth.credential.last.used.test"),
		lastReasons:        map[string]string{},
		collectedGauges:    map[string]collectedGauge{},
	}, reader
}

func newTestMeter(t *testing.T) (otelmetric.Meter, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	return provider.Meter("credentials-test"), reader
}

func mustTestCounter(t *testing.T, meter otelmetric.Meter, name string) otelmetric.Int64Counter {
	t.Helper()
	counter, err := newCredentialsCounter(meter, name, "test")
	if err != nil {
		t.Fatalf("newCredentialsCounter() error = %v", err)
	}
	return counter
}

func mustTestGauge(t *testing.T, meter otelmetric.Meter, name string) otelmetric.Float64Gauge {
	t.Helper()
	gauge, err := newCredentialsGauge(meter, name, "test")
	if err != nil {
		t.Fatalf("newCredentialsGauge() error = %v", err)
	}
	return gauge
}

func observedMetricValue(t *testing.T, reader *sdkmetric.ManualReader, name string) float64 {
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
			return firstObservedValue(t, name, observed.Data)
		}
	}
	t.Fatalf("metric %q not found", name)
	return 0
}

func firstObservedValue(t *testing.T, name string, data metricdata.Aggregation) float64 {
	t.Helper()
	switch value := data.(type) {
	case metricdata.Gauge[float64]:
		if len(value.DataPoints) > 0 {
			return value.DataPoints[0].Value
		}
	case metricdata.Sum[int64]:
		if len(value.DataPoints) > 0 {
			return float64(value.DataPoints[0].Value)
		}
	case metricdata.Sum[float64]:
		if len(value.DataPoints) > 0 {
			return value.DataPoints[0].Value
		}
	}
	t.Fatalf("metric %q data = %T, want gauge or sum with one point", name, data)
	return 0
}

func TestSanitizeCollectorLabelsDropsInstanceLabels(t *testing.T) {
	t.Parallel()

	m := &observabilityMetrics{ownerLabel: "cli_id"}
	labels := m.sanitizeCollectorLabels(map[string]string{
		"provider_surface_binding_id": "instance-1",
		"instance_id":                 "instance-1",
		"model_id":                    "gemini-2.5-pro",
	})
	if _, ok := labels["provider_surface_binding_id"]; ok {
		t.Fatal("provider_surface_binding_id should be dropped")
	}
	if _, ok := labels["instance_id"]; ok {
		t.Fatal("instance_id should be dropped")
	}
	if got, want := labels["model_id"], "gemini-2.5-pro"; got != want {
		t.Fatalf("model_id = %q, want %q", got, want)
	}
}
