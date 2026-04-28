package providerobservability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code-code.internal/platform-k8s/internal/supportservice/clidefinitions/codeassist"
)

func TestAntigravityOAuthObservabilityCollectorCollectQuota(t *testing.T) {
	previousFetchURLs := antigravityFetchAvailableModelsURLs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer access-token"; got != want {
			t.Fatalf("authorization = %q, want %q", got, want)
		}
		switch r.URL.Path {
		case "/load":
			if got, want := r.Header.Get("User-Agent"), codeassist.AntigravityUserAgent; got != want {
				t.Fatalf("load user-agent = %q, want %q", got, want)
			}
			_, _ = w.Write([]byte(`{"cloudaicompanionProject":{"id":"workspacecli-489315"},"paidTier":{"name":"Google AI Pro"}}`))
		case "/models":
			if got, want := r.Header.Get("User-Agent"), "antigravity/1.22.2 darwin/arm64"; got != want {
				t.Fatalf("models user-agent = %q, want %q", got, want)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if got, want := body["project"], "workspacecli-489315"; got != want {
				t.Fatalf("project = %v, want %v", got, want)
			}
			_, _ = w.Write([]byte(`{"models":{"gemini-2.5-pro":{"quotaInfo":{"remainingFraction":0.6,"resetTime":"2026-04-17T06:00:00Z"}},"claude-sonnet-4-6":{"quotaInfo":{"remainingFraction":0.15,"resetTime":"2026-04-17T08:00:00Z"}},"internal-model":{"quotaInfo":{"remainingFraction":1,"resetTime":"2026-04-17T09:00:00Z"}}}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	restoreCodeAssist := codeassist.SetAntigravityURLsForTest(server.URL+"/load", "")
	antigravityFetchAvailableModelsURLs = []string{server.URL + "/models"}
	defer func() {
		restoreCodeAssist()
		antigravityFetchAvailableModelsURLs = previousFetchURLs
	}()

	collector := NewAntigravityOAuthObservabilityCollector()
	result, err := collector.Collect(context.Background(), OAuthObservabilityCollectInput{
		AccessToken:           "access-token",
		HTTPClient:            server.Client(),
		ModelCatalogUserAgent: "antigravity/1.22.2 darwin/arm64",
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got, want := metricRowValueWithLabels(result.GaugeRows, antigravityQuotaRemainingPercentMetric, "model_id", "gemini-2.5-pro"), 60.0; got != want {
		t.Fatalf("gemini remaining percent = %v, want %v", got, want)
	}
	if got, want := metricRowValueWithLabels(result.GaugeRows, antigravityQuotaRemainingPercentMetric, "model_id", "claude-sonnet-4-6"), 15.0; got != want {
		t.Fatalf("claude remaining percent = %v, want %v", got, want)
	}
	if got := metricRowValueWithLabels(result.GaugeRows, antigravityQuotaRemainingPercentMetric, "model_id", "internal-model"); got != 0 {
		t.Fatalf("internal model remaining percent = %v, want 0", got)
	}
	if got, want := result.CredentialBackfillValues[materialKeyProjectID], "workspacecli-489315"; got != want {
		t.Fatalf("secret project_id = %q, want %q", got, want)
	}
	if got, want := result.CredentialBackfillValues[materialKeyTierName], "Google AI Pro"; got != want {
		t.Fatalf("secret tier_name = %q, want %q", got, want)
	}
}

func TestAntigravityOAuthObservabilityCollectorNormalizesGenericTierName(t *testing.T) {
	previousFetchURLs := antigravityFetchAvailableModelsURLs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/load":
			_, _ = w.Write([]byte(`{"currentTier":{"id":"free-tier","name":"Antigravity"},"allowedTiers":[{"id":"free-tier","name":"Antigravity","isDefault":true}],"cloudaicompanionProject":{"id":"workspacecli-489315"}}`))
		case "/models":
			_, _ = w.Write([]byte(`{"models":{"gemini-2.5-pro":{"quotaInfo":{"remainingFraction":0.6,"resetTime":"2026-04-17T06:00:00Z"}}}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	restoreCodeAssist := codeassist.SetAntigravityURLsForTest(server.URL+"/load", "")
	antigravityFetchAvailableModelsURLs = []string{server.URL + "/models"}
	defer func() {
		restoreCodeAssist()
		antigravityFetchAvailableModelsURLs = previousFetchURLs
	}()

	result, err := NewAntigravityOAuthObservabilityCollector().Collect(context.Background(), OAuthObservabilityCollectInput{
		AccessToken: "access-token",
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got, want := result.CredentialBackfillValues[materialKeyTierName], "Free"; got != want {
		t.Fatalf("secret tier_name = %q, want %q", got, want)
	}
}

func metricRowValueWithLabels(rows []OAuthObservabilityMetricRow, metricName string, labelName string, labelValue string) float64 {
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
