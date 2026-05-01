package providerobservability

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	"code-code.internal/platform-k8s/internal/sessioncookie"
	"google.golang.org/grpc"
)

type mistralRewriteTransport struct {
	Target *url.URL
}

func (t *mistralRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.Target.Scheme
	req.URL.Host = t.Target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestMistralObservabilityCollectorCollectUsesAdminBillingSession(t *testing.T) {
	seenPaths := map[string]bool{}
	var billingQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths[r.URL.Path] = true
		if got, want := r.Header.Get("Cookie"), "ory_session_test=abc; csrftoken=csrf-1"; got != want {
			t.Fatalf("Cookie = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("X-CSRFTOKEN"), "csrf-1"; got != want {
			t.Fatalf("X-CSRFTOKEN = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Origin"), mistralBillingOrigin; got != want {
			t.Fatalf("Origin = %q, want %q", got, want)
		}
		wantReferer := mistralBillingReferer
		if r.URL.Path == mistralLimitsPath {
			wantReferer = mistralLimitsReferer
		}
		if got, want := r.Header.Get("Referer"), wantReferer; got != want {
			t.Fatalf("Referer = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case mistralBillingPath:
			billingQuery = r.URL.Query()
			_, _ = w.Write([]byte(`{
				"completion": {
					"models": {
						"mistral-large-latest::mistral-large-2411": {
							"input": [
								{"billing_metric": "mistral-large-2411", "billing_display_name": "mistral-large-latest", "billing_group": "input", "value": 10},
								{"billing_metric": "mistral-large-2411", "billing_display_name": "mistral-large-latest", "billing_group": "input", "value_paid": 3}
							],
							"output": [{"billing_metric": "mistral-large-2411", "billing_display_name": "mistral-large-latest", "billing_group": "output", "value": 7}],
							"cached": [{"billing_metric": "mistral-large-2411", "billing_display_name": "mistral-large-latest", "billing_group": "cached", "value_paid": 2}]
						}
					}
				},
				"currency": "EUR",
				"currency_symbol": "\u20ac",
				"end_date": "2026-04-30T23:59:59.999Z",
				"prices": [
					{"billing_metric": "mistral-large-2411", "billing_group": "input", "price": "0.1"},
					{"billing_metric": "mistral-large-2411", "billing_group": "output", "price": "0.2"},
					{"billing_metric": "mistral-large-2411", "billing_group": "cached", "price": "0.05"}
				]
			}`))
		case mistralLimitsPath:
			_, _ = w.Write([]byte(`{
				"limits": {
					"completion": {
						"usage_limit": 150,
						"usage": 3,
						"vibe_usage": 2,
						"tokens_limits_by_model": {
							"mistral-large-2411": {
								"tokens_per_minute": 600000,
								"tokens_per_month": 200000000000
							}
						},
						"model_request_limits": {
							"mistral-large-2411": {
								"requests_per_second": 0.43333333333333335
							}
						}
					}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	collector := &mistralObservabilityCollector{
		now: func() time.Time { return time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC) },
	}
	authClient := &fakeMistralBillingAuthClient{values: map[string]string{
		"cookie": "ory_session_test=abc; csrftoken=csrf-1",
	}}
	result, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		ProviderID:   "mistral-provider",
		SurfaceID:    "openai-compatible",
		CredentialID: "mistral-provider-observability",
		Auth:         authClient,
		HTTPClient: &http.Client{
			Transport: &mistralRewriteTransport{Target: targetURL},
			Timeout:   2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if !seenPaths[mistralBillingPath] {
		t.Fatalf("missing billing request path %q", mistralBillingPath)
	}
	if got, want := billingQuery.Get("month"), "4"; got != want {
		t.Fatalf("month = %q, want %q", got, want)
	}
	if got, want := billingQuery.Get("year"), "2026"; got != want {
		t.Fatalf("year = %q, want %q", got, want)
	}

	assertMistralTokenRow(t, result.GaugeRows, "mistral-large-2411", "input", 13)
	assertMistralTokenRow(t, result.GaugeRows, "mistral-large-2411", "output", 7)
	assertMistralTokenRow(t, result.GaugeRows, "mistral-large-2411", "cached", 2)
	assertMistralCostRow(t, result.GaugeRows, "total", "all", "", "EUR", 2.8)
	assertMistralCostRow(t, result.GaugeRows, "model", "completion", "mistral-large-2411", "EUR", 2.8)
	assertMistralQuotaRow(t, result.GaugeRows, providerQuotaLimitMetric, "mistral-large-2411", "tokens", "minute", 600000)
	assertMistralQuotaRow(t, result.GaugeRows, providerQuotaLimitMetric, "mistral-large-2411", "tokens", "month", 200000000000)
	assertMistralQuotaRow(t, result.GaugeRows, providerQuotaLimitMetric, "mistral-large-2411", "requests", "second", 0.43333333333333335)
	assertMistralQuotaRow(t, result.GaugeRows, providerQuotaLimitMetric, "", "cost", "month", 150)
	assertMistralQuotaRow(t, result.GaugeRows, providerQuotaUsageMetric, "", "cost", "month", 5)
	assertMistralQuotaRow(t, result.GaugeRows, providerQuotaRemainingMetric, "", "cost", "month", 145)
	assertMistralQuotaRow(t, result.GaugeRows, providerQuotaResetTimestampMetric, "", "cost", "month", float64(time.Date(2026, 5, 1, 0, 0, 0, 999000000, time.UTC).Unix()))
}

func TestParseMistralBillingGaugeRowsPrefersUsageValueOverPaidValue(t *testing.T) {
	rows, err := parseMistralBillingGaugeRows([]byte(`{
		"completion": {
			"models": {
				"mistral-small-latest::mistral-small-2603": {
					"input": [{
						"billing_display_name": "mistral-small-latest",
						"billing_group": "input",
						"billing_metric": "mistral-small-2603",
						"event_type": "inference",
						"timestamp": "2026-04-30T12:00:00Z",
						"usage_type": "input",
						"value": 56,
						"value_paid": 0
					}],
					"output": [{
						"billing_display_name": "mistral-small-latest",
						"billing_group": "output",
						"billing_metric": "mistral-small-2603",
						"event_type": "inference",
						"timestamp": "2026-04-30T12:00:00Z",
						"usage_type": "output",
						"value": 6,
						"value_paid": 0
					}]
				}
			}
		},
		"currency": "EUR",
		"currency_symbol": "\u20ac",
		"prices": []
	}`))
	if err != nil {
		t.Fatalf("parseMistralBillingGaugeRows() error = %v", err)
	}

	assertMistralTokenRow(t, rows, "mistral-small-2603", "input", 56)
	assertMistralTokenRow(t, rows, "mistral-small-2603", "output", 6)
	assertMistralCostRow(t, rows, "total", "all", "", "EUR", 0)
}

func TestParseMistralBillingGaugeRowsComputesCostAcrossBillingCategories(t *testing.T) {
	rows, err := parseMistralBillingGaugeRows([]byte(`{
		"completion": {
			"models": {
				"mistral-small-latest::mistral-small-2603": {
					"input": [{
						"billing_metric": "mistral-small-2603",
						"billing_group": "input",
						"value": 100,
						"value_paid": 80
					}],
					"output": [{
						"billing_metric": "mistral-small-2603",
						"billing_group": "output",
						"value": 10,
						"value_paid": 10
					}]
				}
			}
		},
		"ocr": {
			"models": {
				"mistral-ocr-latest": {
					"input": [{
						"billing_metric": "mistral-ocr-latest",
						"billing_group": "pages",
						"value": 2
					}]
				}
			}
		},
		"vibe_usage": 1.25,
		"currency": "EUR",
		"currency_symbol": "\u20ac",
		"prices": [
			{"billing_metric": "mistral-small-2603", "billing_group": "input", "price": "0.001"},
			{"billing_metric": "mistral-small-2603", "billing_group": "output", "price": "0.01"},
			{"billing_metric": "mistral-ocr-latest", "billing_group": "pages", "price": "0.03"}
		]
	}`))
	if err != nil {
		t.Fatalf("parseMistralBillingGaugeRows() error = %v", err)
	}

	assertMistralTokenRow(t, rows, "mistral-small-2603", "input", 100)
	assertMistralTokenRow(t, rows, "mistral-small-2603", "output", 10)
	assertMistralCostRow(t, rows, "model", "completion", "mistral-small-2603", "EUR", 0.18)
	assertMistralCostRow(t, rows, "model", "ocr", "mistral-ocr-latest", "EUR", 0.06)
	assertMistralCostRow(t, rows, "total", "all", "", "EUR", 0.24)
}

func TestParseMistralBillingLimitsGaugeRowsAddsModelScopedLimits(t *testing.T) {
	rows, err := parseMistralBillingLimitsGaugeRows([]byte(`{
		"limits": {
			"completion": {
				"usage_limit": 150,
				"usage": 3,
				"vibe_usage": 2,
				"tokens_limits_by_model": {
					"mistral-small-2603": {
						"tokens_per_minute": 1500000,
						"tokens_per_month": 4000000
					},
					"mistral-medium-2508": {
						"tokens_per_minute": 356250,
						"tokens_per_month": 0
					}
				},
				"model_request_limits": {
					"mistral-small-2603": {"requests_per_second": 6.6666666667}
				}
			}
		}
	}`))
	if err != nil {
		t.Fatalf("parseMistralBillingLimitsGaugeRows() error = %v", err)
	}

	assertMistralQuotaRow(t, rows, providerQuotaLimitMetric, "mistral-small-2603", "tokens", "minute", 1500000)
	assertMistralQuotaRow(t, rows, providerQuotaLimitMetric, "mistral-small-2603", "tokens", "month", 4000000)
	assertMistralQuotaRow(t, rows, providerQuotaLimitMetric, "mistral-medium-2508", "tokens", "minute", 356250)
	assertMistralQuotaRow(t, rows, providerQuotaLimitMetric, "mistral-small-2603", "requests", "second", 6.6666666667)
	assertMistralQuotaRow(t, rows, providerQuotaLimitMetric, "", "cost", "month", 150)
	assertMistralQuotaRow(t, rows, providerQuotaUsageMetric, "", "cost", "month", 5)
	assertMistralQuotaRow(t, rows, providerQuotaRemainingMetric, "", "cost", "month", 145)
	if mistralQuotaRowExists(rows, providerQuotaLimitMetric, "mistral-medium-2508", "tokens", "month") {
		t.Fatal("mistral-medium-2508 monthly limit row exists, want omitted")
	}
}

func TestParseMistralBillingGaugeRowsEmptyUsageSucceeds(t *testing.T) {
	rows, err := parseMistralBillingGaugeRows([]byte(`{"completion":{"models":{}}}`))
	if err != nil {
		t.Fatalf("parseMistralBillingGaugeRows() error = %v", err)
	}
	assertMistralCostRow(t, rows, "total", "all", "", "EUR", 0)
}

func TestMistralObservabilityCollectorCollectRequiresCookieMaterial(t *testing.T) {
	collector := &mistralObservabilityCollector{now: time.Now}
	_, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		ProviderID:   "mistral-provider",
		SurfaceID:    "openai-compatible",
		CredentialID: "mistral-provider-observability",
		Auth:         &fakeMistralBillingAuthClient{values: map[string]string{}},
		HTTPClient:   http.DefaultClient,
	})
	if !isObservabilityUnauthorizedError(err) {
		t.Fatalf("Collect() error = %v, want unauthorized observability error", err)
	}
}

func TestMistralObservabilityCollectorCollectTreatsLoginRedirectAsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://auth.mistral.ai/self-service/login/browser", http.StatusFound)
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	collector := &mistralObservabilityCollector{now: time.Now}
	_, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		ProviderID:   "mistral-provider",
		SurfaceID:    "openai-compatible",
		CredentialID: "mistral-provider-observability",
		Auth: &fakeMistralBillingAuthClient{values: map[string]string{
			"cookie": "ory_session_test=expired",
		}},
		HTTPClient: &http.Client{
			Transport: &mistralRewriteTransport{Target: targetURL},
			Timeout:   2 * time.Second,
		},
	})
	if !isObservabilityUnauthorizedError(err) {
		t.Fatalf("Collect() error = %v, want unauthorized observability error", err)
	}
}

func assertMistralTokenRow(t *testing.T, rows []ObservabilityMetricRow, modelID, tokenType string, want float64) {
	t.Helper()
	for _, row := range rows {
		if row.MetricName == mistralBillingTokensMetric &&
			row.Labels["model_id"] == modelID &&
			row.Labels["token_type"] == tokenType {
			if got := row.Value; got != want {
				t.Fatalf("%s %s value = %v, want %v", modelID, tokenType, got, want)
			}
			if got, want := row.Labels["window"], "month"; got != want {
				t.Fatalf("%s %s window = %q, want %q", modelID, tokenType, got, want)
			}
			if got, want := row.Labels["gen_ai.provider.name"], mistralOTelProviderName; got != want {
				t.Fatalf("%s %s gen_ai.provider.name = %q, want %q", modelID, tokenType, got, want)
			}
			return
		}
	}
	t.Fatalf("missing %s %s token row in %#v", modelID, tokenType, rows)
}

func assertMistralCostRow(t *testing.T, rows []ObservabilityMetricRow, scope, category, modelID, currency string, want float64) {
	t.Helper()
	for _, row := range rows {
		if row.MetricName == mistralBillingCostMetric &&
			row.Labels["scope"] == scope &&
			row.Labels["usage_category"] == category &&
			row.Labels["model_id"] == modelID {
			if got := row.Value; math.Abs(got-want) > 0.0000001 {
				t.Fatalf("%s %s %s cost value = %v, want %v", scope, category, modelID, got, want)
			}
			if got := row.Labels["currency"]; got != currency {
				t.Fatalf("%s %s %s currency = %q, want %q", scope, category, modelID, got, currency)
			}
			if got, want := row.Labels["gen_ai.provider.name"], mistralOTelProviderName; got != want {
				t.Fatalf("%s %s %s gen_ai.provider.name = %q, want %q", scope, category, modelID, got, want)
			}
			return
		}
	}
	t.Fatalf("missing %s %s %s cost row in %#v", scope, category, modelID, rows)
}

func assertMistralQuotaRow(t *testing.T, rows []ObservabilityMetricRow, metricName, modelID, resource, window string, want float64) string {
	t.Helper()
	for _, row := range rows {
		if row.MetricName == metricName &&
			row.Labels["model_id"] == modelID &&
			row.Labels["resource"] == resource &&
			row.Labels["window"] == window {
			if got := row.Value; got != want {
				t.Fatalf("%s %s %s value = %v, want %v", modelID, resource, window, got, want)
			}
			return row.Labels["quota_pool_id"]
		}
	}
	t.Fatalf("missing %s %s %s %s row in %#v", metricName, modelID, resource, window, rows)
	return ""
}

func mistralQuotaRowExists(rows []ObservabilityMetricRow, metricName, modelID, resource, window string) bool {
	for _, row := range rows {
		if row.MetricName == metricName &&
			row.Labels["model_id"] == modelID &&
			row.Labels["resource"] == resource &&
			row.Labels["window"] == window {
			return true
		}
	}
	return false
}

type fakeMistralBillingAuthClient struct {
	values map[string]string
}

func (c *fakeMistralBillingAuthClient) GetEgressAuthPolicy(
	_ context.Context,
	request *authv1.GetEgressAuthPolicyRequest,
	_ ...grpc.CallOption,
) (*authv1.GetEgressAuthPolicyResponse, error) {
	if got, want := strings.TrimSpace(request.GetPolicyId()), mistralAdminSessionPolicyID; got != want {
		return nil, fmt.Errorf("policy_id = %q, want %q", got, want)
	}
	return &authv1.GetEgressAuthPolicyResponse{
		PolicyId:           mistralAdminSessionPolicyID,
		AdapterId:          "mistral-admin-session",
		RequestHeaderNames: []string{"cookie", "x-csrftoken"},
		RequestReplacementRules: []*authv1.EgressSimpleReplacementRule{
			{HeaderName: "cookie"},
			{HeaderName: "x-csrftoken"},
		},
	}, nil
}

func (c *fakeMistralBillingAuthClient) ReadCredentialMaterialFields(
	_ context.Context,
	_ *authv1.ReadCredentialMaterialFieldsRequest,
	_ ...grpc.CallOption,
) (*authv1.ReadCredentialMaterialFieldsResponse, error) {
	return nil, fmt.Errorf("ReadCredentialMaterialFields should not be called")
}

func (c *fakeMistralBillingAuthClient) ResolveEgressRequestHeaders(
	_ context.Context,
	request *authv1.ResolveEgressRequestHeadersRequest,
	_ ...grpc.CallOption,
) (*authv1.ResolveEgressRequestHeadersResponse, error) {
	if got, want := strings.TrimSpace(request.GetCredentialId()), "mistral-provider-observability"; got != want {
		return nil, fmt.Errorf("credential_id = %q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(request.GetPolicyId()), mistralAdminSessionPolicyID; got != want {
		return nil, fmt.Errorf("policy_id = %q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(request.GetTargetHost()), "admin.mistral.ai"; got != want {
		return nil, fmt.Errorf("target_host = %q, want %q", got, want)
	}
	cookie := strings.TrimSpace(c.values["cookie"])
	if cookie == "" {
		return &authv1.ResolveEgressRequestHeadersResponse{Skipped: true}, nil
	}
	headers := []*authv1.EgressHeaderMutation{{Name: "cookie", Value: cookie}}
	if csrfToken := sessioncookie.Value(cookie, "csrftoken"); csrfToken != "" {
		headers = append(headers, &authv1.EgressHeaderMutation{Name: "x-csrftoken", Value: csrfToken})
	}
	return &authv1.ResolveEgressRequestHeadersResponse{Headers: headers}, nil
}
