package providerobservability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type openrouterRewriteTransport struct {
	Target *url.URL
}

func (t *openrouterRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.Target.Scheme
	req.URL.Host = t.Target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestOpenRouterObservabilityCollectorCollect(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")

		response := map[string]any{
			"data": []map[string]any{
				{
					"model":             "openai/gpt-4o",
					"requests":          150,
					"usage":             1.23,
					"prompt_tokens":     12000,
					"completion_tokens": 3000,
				},
				{
					"model":             "anthropic/claude-3-sonnet",
					"requests":          50,
					"usage":             0.45,
					"prompt_tokens":     4000,
					"completion_tokens": 1500,
				},
				{
					"requests": 10,
					"usage":    0.0,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	mockClient := &http.Client{
		Transport: &openrouterRewriteTransport{Target: targetURL},
		Timeout:   2 * time.Second,
	}

	collector := NewOpenRouterObservabilityCollector()
	result, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		OwnerID:                 "openrouter",
		ProviderID:               "acc123",
		ProviderSurfaceBindingID: "inst123",
		APIKey:                   "sk-or-v1-mockapikey123",
		HTTPClient:               mockClient,
	})

	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if got, want := authHeader, "Bearer sk-or-v1-mockapikey123"; got != want {
		t.Fatalf("Authorization header = %q, want %q", got, want)
	}

	// We expect 4 items from the first model, 4 from the second model, and 1 from the third (only requests > 0).
	// Total rows = 4 + 4 + 1 = 9 rows
	if got, want := len(result.GaugeRows), 9; got != want {
		t.Fatalf("metric rows = %d, want %d", got, want)
	}

	var foundGPTCost, foundClaudeInput bool
	for _, row := range result.GaugeRows {
		if row.MetricName == providerUsageCostUSDMetric && row.Labels["model_id"] == "openai/gpt-4o" {
			foundGPTCost = true
			if row.Value != 1.23 {
				t.Errorf("gpt-4o cost = %f, want 1.23", row.Value)
			}
		}
		if row.MetricName == providerUsageTokensMetric &&
			row.Labels["model_id"] == "anthropic/claude-3-sonnet" &&
			row.Labels["token_type"] == "input" {
			foundClaudeInput = true
			if row.Value != 4000 {
				t.Errorf("claude input tokens = %f, want 4000", row.Value)
			}
		}
	}

	if !foundGPTCost {
		t.Errorf("Missing cost metric for openai/gpt-4o")
	}
	if !foundClaudeInput {
		t.Errorf("Missing input token metric for anthropic/claude-3-sonnet")
	}
}
