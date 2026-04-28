package providerobservability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMinimaxObservabilityCollectorCollectsTextQuota(t *testing.T) {
	previousCNURL := minimaxRemainsCNURL
	previousGlobalURL := minimaxRemainsGlobalURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodGet; got != want {
			t.Fatalf("method = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer test-key"; got != want {
			t.Fatalf("authorization = %q, want %q", got, want)
		}
		_, _ = w.Write([]byte(`{
			"base_resp":{"status_code":0,"status_msg":"ok"},
			"model_remains":[
				{"model_name":"MiniMax-M2.7","remaining_count":120,"total_count":200,"reset_time":"2026-04-18T00:00:00Z"},
				{"model_name":"image-01","remaining_count":2,"total_count":10}
			]
		}`))
	}))
	defer server.Close()
	minimaxRemainsCNURL = server.URL
	minimaxRemainsGlobalURL = server.URL
	defer func() {
		minimaxRemainsCNURL = previousCNURL
		minimaxRemainsGlobalURL = previousGlobalURL
	}()

	result, err := NewMinimaxObservabilityCollector().Collect(context.Background(), ObservabilityCollectInput{
		SurfaceBaseURL: "https://api.minimaxi.com/v1",
		APIKey:         "test-key",
		HTTPClient:     server.Client(),
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got, want := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextRemainingCountMetric, "model_id", "MiniMax-M2.7"), 120.0; got != want {
		t.Fatalf("remaining_count = %v, want %v", got, want)
	}
	if got, want := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextTotalCountMetric, "model_id", "MiniMax-M2.7"), 200.0; got != want {
		t.Fatalf("total_count = %v, want %v", got, want)
	}
	if got, want := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextRemainingPercentMetric, "model_id", "MiniMax-M2.7"), 60.0; got != want {
		t.Fatalf("remaining_percent = %v, want %v", got, want)
	}
	resetAt := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextResetTimestampMetric, "model_id", "MiniMax-M2.7")
	if resetAt != float64(time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC).Unix()) {
		t.Fatalf("reset_timestamp = %v, want %v", resetAt, float64(time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC).Unix()))
	}
	if got := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextRemainingCountMetric, "model_id", "image-01"); got != 0 {
		t.Fatalf("non-text model remaining_count = %v, want 0", got)
	}
}

func TestMinimaxObservabilityCollectorCollectsCurrentTokenPlanShape(t *testing.T) {
	previousCNURL := minimaxRemainsCNURL
	previousGlobalURL := minimaxRemainsGlobalURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"base_resp":{"status_code":0,"status_msg":"success"},
			"model_remains":[
				{
					"model_name":"MiniMax-M*",
					"current_interval_total_count":4500,
					"current_interval_usage_count":4377,
					"end_time":1776441600000
				},
				{
					"model_name":"coding-plan-vlm",
					"current_interval_total_count":450,
					"current_interval_usage_count":372,
					"end_time":1776441600000
				},
				{
					"model_name":"coding-plan-search",
					"current_interval_total_count":450,
					"current_interval_usage_count":401,
					"end_time":1776441600000
				},
				{
					"model_name":"MiniMax-Hailuo-2.3-Fast-6s-768p",
					"current_interval_total_count":2,
					"current_interval_usage_count":1,
					"end_time":1776441600000
				}
			]
		}`))
	}))
	defer server.Close()
	minimaxRemainsCNURL = server.URL
	minimaxRemainsGlobalURL = server.URL
	defer func() {
		minimaxRemainsCNURL = previousCNURL
		minimaxRemainsGlobalURL = previousGlobalURL
	}()

	result, err := NewMinimaxObservabilityCollector().Collect(context.Background(), ObservabilityCollectInput{
		SurfaceBaseURL: "https://api.minimaxi.com/v1",
		APIKey:         "test-key",
		HTTPClient:     server.Client(),
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got, want := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextRemainingCountMetric, "model_id", "MiniMax-M*"), 4377.0; got != want {
		t.Fatalf("remaining_count = %v, want %v", got, want)
	}
	if got, want := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextRemainingCountMetric, "model_id", "coding-plan-vlm"), 372.0; got != want {
		t.Fatalf("vlm remaining_count = %v, want %v", got, want)
	}
	if got, want := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextRemainingCountMetric, "model_id", "coding-plan-search"), 401.0; got != want {
		t.Fatalf("search remaining_count = %v, want %v", got, want)
	}
	if got, want := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextTotalCountMetric, "model_id", "MiniMax-M*"), 4500.0; got != want {
		t.Fatalf("total_count = %v, want %v", got, want)
	}
	if got, want := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextRemainingPercentMetric, "model_id", "MiniMax-M*"), 4377.0/4500.0*100.0; got != want {
		t.Fatalf("remaining_percent = %v, want %v", got, want)
	}
	resetAt := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextResetTimestampMetric, "model_id", "MiniMax-M*")
	if resetAt != float64(time.UnixMilli(1776441600000).UTC().Unix()) {
		t.Fatalf("reset_timestamp = %v, want %v", resetAt, float64(time.UnixMilli(1776441600000).UTC().Unix()))
	}
	if got := vendorMetricRowValueWithLabels(result.GaugeRows, minimaxTextRemainingCountMetric, "model_id", "MiniMax-Hailuo-2.3-Fast-6s-768p"); got != 0 {
		t.Fatalf("non-text minimax remaining_count = %v, want 0", got)
	}
}

func TestMinimaxObservabilityCollectorTreatsBaseRespUnauthorizedAsAuthBlocked(t *testing.T) {
	previousCNURL := minimaxRemainsCNURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"base_resp":{"status_code":1004,"status_msg":"cookie is missing, log in again"}}`))
	}))
	defer server.Close()
	minimaxRemainsCNURL = server.URL
	defer func() {
		minimaxRemainsCNURL = previousCNURL
	}()

	_, err := NewMinimaxObservabilityCollector().Collect(context.Background(), ObservabilityCollectInput{
		SurfaceBaseURL: "https://api.minimaxi.com/v1",
		APIKey:         "test-key",
		HTTPClient:     server.Client(),
	})
	if err == nil {
		t.Fatal("Collect() error = nil, want unauthorized error")
	}
	if !isObservabilityUnauthorizedError(err) {
		t.Fatalf("Collect() error = %v, want unauthorized error", err)
	}
}

func TestMinimaxRemainsURLRejectsUnsupportedHost(t *testing.T) {
	if _, err := minimaxRemainsURL("https://example.com/v1"); err == nil {
		t.Fatal("minimaxRemainsURL() error = nil, want unsupported host error")
	}
}

func vendorMetricRowValueWithLabels(rows []ObservabilityMetricRow, metricName string, labelName string, labelValue string) float64 {
	for _, row := range rows {
		if row.MetricName != metricName {
			continue
		}
		if row.Labels[labelName] == labelValue {
			return row.Value
		}
	}
	return 0
}
