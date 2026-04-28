package providerobservability

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestVendorObservabilityMetricsRecordAuthUsable(t *testing.T) {
	t.Parallel()

	metrics, reader := newTestVendorObservabilityMetrics(t)
	now := time.Unix(1700000000, 0).UTC()

	metrics.record("cerebras", "account-a", TriggerManual, ProbeOutcomeExecuted, "", now, now.Add(time.Minute))
	if got := observedMetricValue(t, reader, "vendor.auth.usable.test"); got != 1 {
		t.Fatalf("authUsable after executed = %v, want 1", got)
	}
	if got := observedMetricValue(t, reader, "vendor.probe.last.outcome.test"); got != 1 {
		t.Fatalf("lastOutcome after executed = %v, want 1", got)
	}
	if got, want := observedMetricValue(t, reader, "vendor.credential.last.used.test"), float64(now.Unix()); got != want {
		t.Fatalf("credentialLastUsed after executed = %v, want %v", got, want)
	}

	authBlockedAt := now.Add(time.Minute)
	metrics.record("cerebras", "account-a", TriggerManual, ProbeOutcomeAuthBlocked, "SESSION_COOKIE_INVALID", authBlockedAt, authBlockedAt.Add(time.Minute))
	if got := observedMetricValue(t, reader, "vendor.auth.usable.test"); got != 0 {
		t.Fatalf("authUsable after auth_blocked = %v, want 0", got)
	}
	if got := observedMetricValue(t, reader, "vendor.probe.last.outcome.test"); got != 3 {
		t.Fatalf("lastOutcome after auth_blocked = %v, want 3", got)
	}
	if got, want := observedMetricValue(t, reader, "vendor.credential.last.used.test"), float64(authBlockedAt.Unix()); got != want {
		t.Fatalf("credentialLastUsed after auth_blocked = %v, want %v", got, want)
	}

	failedAt := authBlockedAt.Add(time.Minute)
	metrics.record("cerebras", "account-a", TriggerManual, ProbeOutcomeFailed, "", failedAt, failedAt.Add(time.Minute))
	if got := observedMetricValue(t, reader, "vendor.auth.usable.test"); got != 0 {
		t.Fatalf("authUsable after failed = %v, want previous 0", got)
	}
	if got := observedMetricValue(t, reader, "vendor.probe.last.outcome.test"); got != 5 {
		t.Fatalf("lastOutcome after failed = %v, want 5", got)
	}
	if got, want := observedMetricValue(t, reader, "vendor.credential.last.used.test"), float64(authBlockedAt.Unix()); got != want {
		t.Fatalf("credentialLastUsed after failed = %v, want previous %v", got, want)
	}
}

func TestVendorCollectedGaugeRecordsOTelAttributeSets(t *testing.T) {
	t.Parallel()

	metrics, reader := newTestVendorObservabilityMetrics(t)
	metrics.recordCollectorValues("cerebras", "account-a", []ObservabilityMetricRow{{
		MetricName: "gen_ai.provider.quota.limit.test",
		Value:      100,
		Labels: map[string]string{
			"model_id": "llama",
			"org_id":   "org-personal",
			"resource": "tokens",
			"window":   "day",
		},
	}})
	metrics.recordCollectorValues("minimax", "account-b", []ObservabilityMetricRow{{
		MetricName: "gen_ai.provider.quota.limit.test",
		Value:      10,
		Labels: map[string]string{
			"model_id": "abab",
			"resource": "requests",
			"window":   "day",
		},
	}})

	points := observedGaugePoints(t, reader, "gen_ai.provider.quota.limit.test")
	if len(points) != 2 {
		t.Fatalf("points = %d, want 2", len(points))
	}
	var foundCerebrasOrg bool
	var foundMinimaxWithoutOrg bool
	for _, point := range points {
		labels := attributeSet(point.Attributes.ToSlice())
		switch labels["vendor_id"] {
		case "cerebras":
			foundCerebrasOrg = labels["org_id"] == "org-personal"
		case "minimax":
			_, hasOrgID := labels["org_id"]
			foundMinimaxWithoutOrg = !hasOrgID
		}
	}
	if !foundCerebrasOrg {
		t.Fatal("cerebras point did not keep org_id label")
	}
	if !foundMinimaxWithoutOrg {
		t.Fatal("minimax point unexpectedly included org_id label")
	}
}

func newTestVendorObservabilityMetrics(t *testing.T) (*observabilityMetrics, *sdkmetric.ManualReader) {
	t.Helper()
	meter, reader := newTestMeter(t)
	return &observabilityMetrics{
		ownerLabel:         "vendor_id",
		meter:              meter,
		probeRuns:          mustTestCounter(t, meter, "vendor.probe.runs.test"),
		probeLastRun:       mustTestGauge(t, meter, "vendor.probe.last.run.test"),
		probeLastOutcome:   mustTestGauge(t, meter, "vendor.probe.last.outcome.test"),
		probeLastReason:    mustTestGauge(t, meter, "vendor.probe.last.reason.test"),
		probeNextAllowed:   mustTestGauge(t, meter, "vendor.probe.next.allowed.test"),
		authUsable:         mustTestGauge(t, meter, "vendor.auth.usable.test"),
		credentialLastUsed: mustTestGauge(t, meter, "vendor.credential.last.used.test"),
		lastReasons:        map[string]string{},
		collectedGauges:    map[string]collectedGauge{},
	}, reader
}

func observedGaugePoints(t *testing.T, reader *sdkmetric.ManualReader, name string) []metricdata.DataPoint[float64] {
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
			gauge, ok := observed.Data.(metricdata.Gauge[float64])
			if !ok {
				t.Fatalf("metric %q data = %T, want float64 gauge", name, observed.Data)
			}
			return gauge.DataPoints
		}
	}
	t.Fatalf("metric %q not found", name)
	return nil
}

func attributeSet(attrs []attribute.KeyValue) map[string]string {
	labels := map[string]string{}
	for _, attr := range attrs {
		labels[string(attr.Key)] = attr.Value.AsString()
	}
	return labels
}

func TestSanitizeVendorCollectorLabelsDropsInstanceLabels(t *testing.T) {
	t.Parallel()

	m := &observabilityMetrics{ownerLabel: "vendor_id"}
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
