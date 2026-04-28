package providerobservability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCodexOAuthObservabilityCollectorCollectUsage(t *testing.T) {
	previousURL := codexUsageProbeURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer access-token"; got != want {
			t.Fatalf("authorization = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("ChatGPT-Account-Id"), "account-123"; got != want {
			t.Fatalf("chatgpt-account-id = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("User-Agent"), "codex_cli_rs/0.121.0"; got != want {
			t.Fatalf("user-agent = %q, want %q", got, want)
		}
		_, _ = w.Write([]byte(`{"plan_type":"pro","rate_limit":{"allowed":true,"limit_reached":false,"primary_window":{"used_percent":42,"limit_window_seconds":18000,"reset_at":1735693200},"secondary_window":{"used_percent":7,"limit_window_seconds":604800,"reset_at":1736200000}}}`))
	}))
	defer server.Close()
	codexUsageProbeURL = server.URL
	defer func() { codexUsageProbeURL = previousURL }()

	collector := NewCodexOAuthObservabilityCollector()
	result, err := collector.Collect(context.Background(), OAuthObservabilityCollectInput{
		AccessToken:            "access-token",
		HTTPClient:             server.Client(),
		ObservabilityUserAgent: "codex_cli_rs/0.121.0",
		MaterialValues: map[string]string{
			materialKeyAccountID: "account-123",
		},
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got, want := metricRowValue(result.GaugeRows, codexLimitReachedMetric), 0.0; got != want {
		t.Fatalf("limit reached = %v, want %v", got, want)
	}
	if got, want := metricRowValue(result.GaugeRows, codexPrimaryWindowUsedPercentMetric), 42.0; got != want {
		t.Fatalf("primary used = %v, want %v", got, want)
	}
	if got, want := metricRowValue(result.GaugeRows, codexPrimaryWindowDurationMetric), 300.0; got != want {
		t.Fatalf("primary duration = %v, want %v", got, want)
	}
	if got, want := metricRowValue(result.GaugeRows, codexSecondaryWindowUsedPercentMetric), 7.0; got != want {
		t.Fatalf("secondary used = %v, want %v", got, want)
	}
	if got, want := metricRowValue(result.GaugeRows, codexSecondaryWindowDurationMetric), 10080.0; got != want {
		t.Fatalf("secondary duration = %v, want %v", got, want)
	}
	if got, want := metricRowValue(result.GaugeRows, codexPlanTypeCodeMetric), 5.0; got != want {
		t.Fatalf("plan type code = %v, want %v", got, want)
	}
}

func TestCodexUsageLimitGaugeValuesFallback429(t *testing.T) {
	now := time.Unix(1735692600, 0).UTC()
	values, ok := codexUsageLimitGaugeValues("", nil, now, []byte(`{"error":{"type":"usage_limit_reached","resets_in_seconds":300}}`))
	if !ok {
		t.Fatal("codexUsageLimitGaugeValues() ok = false, want true")
	}
	if got, want := values[codexLimitReachedMetric], 1.0; got != want {
		t.Fatalf("limit reached = %v, want %v", got, want)
	}
	if got, want := values[codexPrimaryWindowUsedPercentMetric], 100.0; got != want {
		t.Fatalf("primary used = %v, want %v", got, want)
	}
	if got, want := values[codexPrimaryWindowResetTimestampMetric], float64(now.Add(5*time.Minute).Unix()); got != want {
		t.Fatalf("primary reset = %v, want %v", got, want)
	}
	if got, want := values[codexPlanTypeCodeMetric], 0.0; got != want {
		t.Fatalf("plan type code = %v, want %v", got, want)
	}
}

func TestCodexOAuthObservabilityCollectorRequiresAccountID(t *testing.T) {
	collector := NewCodexOAuthObservabilityCollector()
	_, err := collector.Collect(context.Background(), OAuthObservabilityCollectInput{
		AccessToken: "access-token",
		HTTPClient:  http.DefaultClient,
	})
	if err == nil {
		t.Fatal("Collect() error = nil, want error")
	}
	if !isOAuthObservabilityUnauthorizedError(err) {
		t.Fatalf("Collect() error = %T, want unauthorized error", err)
	}
}

func metricRowValue(rows []OAuthObservabilityMetricRow, metricName string) float64 {
	for _, row := range rows {
		if row.MetricName == metricName {
			return row.Value
		}
	}
	return 0
}
