package providerobservability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code-code.internal/platform-k8s/internal/supportservice/clidefinitions/codeassist"
)

func TestGeminiObservabilityCollectorCollectQuota(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer access-token"; got != want {
			t.Fatalf("authorization = %q, want %q", got, want)
		}
		switch r.URL.Path {
		case "/load":
			_, _ = w.Write([]byte(`{"cloudaicompanionProject":{"id":"workspacecli-489315"},"paidTier":{"name":"Google AI Pro"}}`))
		case "/quota":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if got, want := body["project"], "workspacecli-489315"; got != want {
				t.Fatalf("project = %v, want %v", got, want)
			}
			_, _ = w.Write([]byte(`{"buckets":[{"modelId":"gemini-2.5-pro","remainingAmount":"12","remainingFraction":0.6,"resetTime":"2026-04-17T06:00:00Z"},{"modelId":"gemini-2.5-flash","remainingAmount":"240","remainingFraction":0.8,"resetTime":"2026-04-17T07:00:00Z"},{"modelId":"gemini-2.5-flash-lite","remainingFraction":0.25,"resetTime":"2026-04-17T08:00:00Z"}]}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	defer codeassist.SetGeminiURLsForTest(server.URL+"/load", server.URL+"/quota")()

	collector := NewGeminiObservabilityCollector()
	result, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		AccessToken: "access-token",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got, want := metricRowValue(result.GaugeRows, geminiProRemainingAmountMetric), 12.0; got != want {
		t.Fatalf("pro amount = %v, want %v", got, want)
	}
	if got, want := metricRowValue(result.GaugeRows, geminiProRemainingPercentMetric), 60.0; got != want {
		t.Fatalf("pro percent = %v, want %v", got, want)
	}
	if got, want := metricRowValue(result.GaugeRows, geminiFlashRemainingAmountMetric), 240.0; got != want {
		t.Fatalf("flash amount = %v, want %v", got, want)
	}
	if got, want := metricRowValue(result.GaugeRows, geminiFlashLiteRemainingPercentMetric), 25.0; got != want {
		t.Fatalf("flash-lite percent = %v, want %v", got, want)
	}
	if got, want := result.CredentialBackfillValues[materialKeyProjectID], "workspacecli-489315"; got != want {
		t.Fatalf("secret project_id = %q, want %q", got, want)
	}
	if got, want := result.CredentialBackfillValues[materialKeyTierName], "Google AI Pro"; got != want {
		t.Fatalf("secret tier_name = %q, want %q", got, want)
	}
}

func TestGeminiQuotaGaugeValuesGroupsLowestRemainingBucket(t *testing.T) {
	values := geminiQuotaGaugeValues(map[string]any{
		"buckets": []any{
			map[string]any{"modelId": "gemini-3-pro-preview", "remainingAmount": "30", "remainingFraction": 0.9, "resetTime": "2026-04-17T06:00:00Z"},
			map[string]any{"modelId": "gemini-2.5-pro", "remainingAmount": "10", "remainingFraction": 0.2, "resetTime": "2026-04-17T05:30:00Z"},
		},
	})
	if got, want := values[geminiProRemainingAmountMetric], 10.0; got != want {
		t.Fatalf("pro amount = %v, want %v", got, want)
	}
	if got, want := values[geminiProRemainingPercentMetric], 20.0; got != want {
		t.Fatalf("pro percent = %v, want %v", got, want)
	}
}
