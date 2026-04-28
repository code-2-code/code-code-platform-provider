package providerobservability

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	observabilityv1 "code-code.internal/go-contract/observability/v1"
)

func TestParseCerebrasOrganizations(t *testing.T) {
	body := []byte(`{
		"data": {
			"ListMyOrganizations": [
				{"id": "org-abc", "name": "My Org", "state": "active"},
				{"id": "org-def", "name": "Other", "state": "suspended"}
			]
		}
	}`)
	got, err := parseCerebrasOrganizations(body)
	if err != nil {
		t.Fatalf("parseCerebrasOrganizations() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(organizations) = %d, want 2", len(got))
	}
	if got[0].ID != "org-abc" {
		t.Errorf("organizations[0].ID = %q, want %q", got[0].ID, "org-abc")
	}
}

func TestParseCerebrasOrganizations_FallbackNonActive(t *testing.T) {
	body := []byte(`{
		"data": {
			"ListMyOrganizations": [
				{"id": "org-only", "name": "Only Org", "state": "pending"}
			]
		}
	}`)
	got, err := parseCerebrasOrganizations(body)
	if err != nil {
		t.Fatalf("parseCerebrasOrganizations() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "org-only" {
		t.Errorf("organizations = %#v, want org-only", got)
	}
}

func TestParseCerebrasOrganizations_Empty(t *testing.T) {
	body := []byte(`{"data": {"ListMyOrganizations": []}}`)
	_, err := parseCerebrasOrganizations(body)
	if err == nil {
		t.Fatal("parseCerebrasOrganizations() error = nil, want error")
	}
}

func TestParseCerebrasGraphQLGaugeRows(t *testing.T) {
	quotaBody := []byte(`{
		"data": {
			"ListOrganizationEffectiveQuotas": [
				{
					"modelId": "llama-4-scout-17b-16e-instruct",
					"regionId": "us-east-1",
					"requestsPerMinute": "30",
					"tokensPerMinute": "60000",
					"requestsPerHour": "-1",
					"tokensPerHour": "-1",
					"requestsPerDay": "1000",
					"tokensPerDay": "1000000",
					"maxSequenceLength": "131072",
					"maxCompletionTokens": "16384"
				}
			]
		}
	}`)
	usageBody := []byte(`{
		"data": {
			"ListOrganizationUsage": [
				{
					"modelId": "llama-4-scout-17b-16e-instruct",
					"regionId": "us-east-1",
					"rpm": "5",
					"tpm": "12000",
					"rph": "0",
					"tph": "0",
					"rpd": "150",
					"tpd": "300000"
				}
			]
		}
	}`)

	rows, err := parseCerebrasGraphQLGaugeRows(quotaBody, usageBody, "org-abc", "Personal")
	if err != nil {
		t.Fatalf("parseCerebrasGraphQLGaugeRows() error = %v", err)
	}

	// 4 windows with positive limits (minute requests, minute tokens, day requests, day tokens)
	// Each window produces 4 rows (limit, usage, remaining, percent)
	// hour has -1 limits so should be skipped
	expectedWindows := 4
	expectedRowsPerWindow := 4
	expectedTotal := expectedWindows * expectedRowsPerWindow
	if len(rows) != expectedTotal {
		t.Errorf("parseCerebrasGraphQLGaugeRows() returned %d rows, want %d", len(rows), expectedTotal)
	}

	// Verify a specific row: day/tokens limit should be 1000000
	found := false
	for _, row := range rows {
		if row.MetricName == cerebrasQuotaLimitMetric &&
			row.Labels["window"] == "day" &&
			row.Labels["resource"] == "tokens" {
			if got, want := row.Labels["org_id"], "org-abc"; got != want {
				t.Fatalf("org_id = %q, want %q", got, want)
			}
			if row.Value != 1000000 {
				t.Errorf("day/tokens limit = %v, want 1000000", row.Value)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("day/tokens limit row not found")
	}

	// Verify usage percent for day/tokens: 300000/1000000 * 100 = 30%
	for _, row := range rows {
		if row.MetricName == cerebrasQuotaUsagePercentMetric &&
			row.Labels["window"] == "day" &&
			row.Labels["resource"] == "tokens" {
			if row.Value != 30.0 {
				t.Errorf("day/tokens usage percent = %v, want 30.0", row.Value)
			}
			break
		}
	}
}

func TestParseCerebrasGraphQLGaugeRows_NoUsage(t *testing.T) {
	quotaBody := []byte(`{
		"data": {
			"ListOrganizationEffectiveQuotas": [
				{
					"modelId": "llama3.1-8b",
					"regionId": "",
					"requestsPerMinute": "30",
					"tokensPerMinute": "60000",
					"requestsPerHour": "",
					"tokensPerHour": "",
					"requestsPerDay": "900",
					"tokensPerDay": "1000000"
				}
			]
		}
	}`)

	rows, err := parseCerebrasGraphQLGaugeRows(quotaBody, nil, "org-abc", "")
	if err != nil {
		t.Fatalf("parseCerebrasGraphQLGaugeRows() error = %v", err)
	}
	if len(rows) == 0 {
		t.Error("parseCerebrasGraphQLGaugeRows() returned 0 rows")
	}
	// Without usage data, usage should be 0 and remaining should equal limit.
	for _, row := range rows {
		if row.MetricName == cerebrasQuotaUsageMetric {
			if row.Value != 0 {
				t.Errorf("usage without usage data = %v, want 0", row.Value)
			}
		}
	}
}

func TestCerebrasParseInt64(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"100", 100},
		{"-1", 0},
		{"", 0},
		{"abc", 0},
		{" 42 ", 42},
	}
	for _, tt := range tests {
		got := cerebrasParseInt64(tt.input)
		if got != tt.want {
			t.Errorf("cerebrasParseInt64(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestCerebrasCollectorRefreshesSessionTokenAcrossRequests(t *testing.T) {
	collector := NewCerebrasObservabilityCollector()
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(request body) error = %v", err)
			}
			body := string(bodyBytes)
			cookie := req.Header.Get("Cookie")
			switch {
			case strings.Contains(body, "ListMyOrganizations"):
				if got, want := cookie, "authjs.session-token=token-1"; got != want {
					t.Fatalf("organization cookie = %q, want %q", got, want)
				}
				return newRoundTripResponse(http.StatusOK, `{"data":{"ListMyOrganizations":[{"id":"org-1","name":"Personal","state":"active"},{"id":"org-2","name":"Team","state":"active"}]}}`, "authjs.session-token=token-2; Path=/"), nil
			case strings.Contains(body, "ListOrganizationEffectiveQuotas"):
				if got, want := cookie, "authjs.session-token=token-1"; got != want {
					t.Fatalf("quota cookie = %q, want %q", got, want)
				}
				if strings.Contains(body, `"organizationId":"org-1"`) {
					return newRoundTripResponse(http.StatusOK, `{"data":{"ListOrganizationEffectiveQuotas":[{"modelId":"llama","regionId":"us-east-1","requestsPerMinute":"30","tokensPerMinute":"60000","requestsPerHour":"-1","tokensPerHour":"-1","requestsPerDay":"900","tokensPerDay":"1000000","totalTokensPerMinute":"120000","totalTokensPerHour":"240000","totalTokensPerDay":"4000000"}]}}`, "authjs.session-token=token-3; Path=/"), nil
				}
				return newRoundTripResponse(http.StatusOK, `{"data":{"ListOrganizationEffectiveQuotas":[{"modelId":"llama","regionId":"us-west-1","requestsPerMinute":"10","tokensPerMinute":"20000","requestsPerHour":"-1","tokensPerHour":"-1","requestsPerDay":"300","tokensPerDay":"500000","totalTokensPerMinute":"40000","totalTokensPerHour":"80000","totalTokensPerDay":"1000000"}]}}`, ""), nil
			case strings.Contains(body, "ListOrganizationUsage"):
				if got, want := cookie, "authjs.session-token=token-1"; got != want {
					t.Fatalf("usage cookie = %q, want %q", got, want)
				}
				if strings.Contains(body, `"organizationId":"org-1"`) {
					return newRoundTripResponse(http.StatusOK, `{"data":{"ListOrganizationUsage":[{"modelId":"llama","regionId":"us-east-1","rpm":"5","tpm":"12000","rph":"0","tph":"0","rpd":"150","tpd":"300000"}]}}`, ""), nil
				}
				return newRoundTripResponse(http.StatusOK, `{"data":{"ListOrganizationUsage":[{"modelId":"llama","regionId":"us-west-1","rpm":"1","tpm":"4000","rph":"0","tph":"0","rpd":"20","tpd":"80000"}]}}`, ""), nil
			default:
				t.Fatalf("unexpected query body: %s", body)
				return nil, nil
			}
		}),
	}

	result, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		ObservabilityCredential: testCerebrasSessionCredential("token-1"),
		CredentialBackfills: []CredentialBackfillRule{{
			RuleID:     "authjs-session-token",
			Source:     observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_HTTP_RESPONSE_COOKIE,
			SourceName: "authjs.session-token",
		}},
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(result.GaugeRows) != 32 {
		t.Fatalf("len(GaugeRows) = %d, want %d", len(result.GaugeRows), 32)
	}
	if got, want := result.CredentialBackfillValues["authjs.session-token"], "token-3"; got != want {
		t.Fatalf("backfilled authjs.session-token = %q, want %q", got, want)
	}
}

func TestCerebrasCollectorReportsInvalidOrExpiredSessionToken(t *testing.T) {
	collector := NewCerebrasObservabilityCollector()
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return newRoundTripResponse(http.StatusUnauthorized, `expired session`, ""), nil
		}),
	}

	_, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		ObservabilityCredential: testCerebrasSessionCredential("token-1"),
		HTTPClient:              client,
	})
	if err == nil {
		t.Fatal("Collect() error = nil, want unauthorized error")
	}
	if !isObservabilityUnauthorizedError(err) {
		t.Fatalf("Collect() unauthorized = false, err=%v", err)
	}
	if got := err.Error(); !strings.Contains(got, "authjs.session-token is invalid or expired") {
		t.Fatalf("error = %q, want invalid or expired prompt", got)
	}
}

func TestCerebrasCollectorSkipsOrganizationsWhoseQuotaQueryFails(t *testing.T) {
	collector := NewCerebrasObservabilityCollector()
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(request body) error = %v", err)
			}
			body := string(bodyBytes)
			switch {
			case strings.Contains(body, "ListMyOrganizations"):
				return newRoundTripResponse(http.StatusOK, `{"data":{"ListMyOrganizations":[{"id":"org-team","name":"Team","state":"active"},{"id":"org-personal","name":"Personal","state":"active"}]}}`, ""), nil
			case strings.Contains(body, "ListOrganizationEffectiveQuotas") && strings.Contains(body, `"organizationId":"org-team"`):
				return newRoundTripResponse(http.StatusBadRequest, `{"errors":[{"message":"Validation error"}]}`, ""), nil
			case strings.Contains(body, "ListOrganizationEffectiveQuotas") && strings.Contains(body, `"organizationId":"org-personal"`):
				return newRoundTripResponse(http.StatusOK, `{"data":{"ListOrganizationEffectiveQuotas":[{"modelId":"gpt-oss-120b","regionId":"us-east-1","requestsPerMinute":"30","tokensPerMinute":"60000","requestsPerHour":"-1","tokensPerHour":"-1","requestsPerDay":"900","tokensPerDay":"1000000","totalTokensPerMinute":"120000","totalTokensPerHour":"240000","totalTokensPerDay":"4000000"}]}}`, ""), nil
			case strings.Contains(body, "ListOrganizationUsage") && strings.Contains(body, `"organizationId":"org-personal"`):
				return newRoundTripResponse(http.StatusOK, `{"data":{"ListOrganizationUsage":[{"modelId":"gpt-oss-120b","regionId":"us-east-1","rpm":"5","tpm":"12000","rph":"0","tph":"0","rpd":"150","tpd":"300000"}]}}`, ""), nil
			default:
				t.Fatalf("unexpected query body: %s", body)
				return nil, nil
			}
		}),
	}

	result, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		ObservabilityCredential: testCerebrasSessionCredential("token-1"),
		HTTPClient:              client,
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(result.GaugeRows) == 0 {
		t.Fatal("len(GaugeRows) = 0, want rows from successful organization")
	}
	for _, row := range result.GaugeRows {
		if got, want := row.Labels["org_id"], "org-personal"; got != want {
			t.Fatalf("org_id = %q, want %q", got, want)
		}
	}
}

func TestCerebrasCollectorFailsWhenEveryOrganizationQuotaQueryFails(t *testing.T) {
	collector := NewCerebrasObservabilityCollector()
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(request body) error = %v", err)
			}
			body := string(bodyBytes)
			switch {
			case strings.Contains(body, "ListMyOrganizations"):
				return newRoundTripResponse(http.StatusOK, `{"data":{"ListMyOrganizations":[{"id":"org-team","name":"Team","state":"active"}]}}`, ""), nil
			case strings.Contains(body, "ListOrganizationEffectiveQuotas"):
				return newRoundTripResponse(http.StatusBadRequest, `{"errors":[{"message":"Validation error"}]}`, ""), nil
			default:
				t.Fatalf("unexpected query body: %s", body)
				return nil, nil
			}
		}),
	}

	_, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		ObservabilityCredential: testCerebrasSessionCredential("token-1"),
		HTTPClient:              client,
	})
	if err == nil {
		t.Fatal("Collect() error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "no metric rows returned") {
		t.Fatalf("error = %q, want no metric rows returned", got)
	}
}

func TestCerebrasGraphQLResponseErrorTreatsSessionErrorsAsUnauthorized(t *testing.T) {
	err := cerebrasGraphQLResponseError([]byte(`{"errors":[{"message":"session expired"}]}`))
	if err == nil {
		t.Fatal("cerebrasGraphQLResponseError() error = nil, want unauthorized error")
	}
	if !isObservabilityUnauthorizedError(err) {
		t.Fatalf("isObservabilityUnauthorizedError() = false, err=%v", err)
	}
}

func TestCerebrasGraphQLResponseErrorTreatsLoggedInErrorsAsUnauthorized(t *testing.T) {
	err := cerebrasGraphQLResponseError([]byte(`{"errors":[{"message":"You must be logged in to access this API."}]}`))
	if err == nil {
		t.Fatal("cerebrasGraphQLResponseError() error = nil, want unauthorized error")
	}
	if !isObservabilityUnauthorizedError(err) {
		t.Fatalf("isObservabilityUnauthorizedError() = false, err=%v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newRoundTripResponse(status int, body string, setCookie string) *http.Response {
	header := http.Header{}
	if strings.TrimSpace(setCookie) != "" {
		header.Set("Set-Cookie", setCookie)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testCerebrasSessionCredential(token string) *credentialv1.ResolvedCredential {
	return &credentialv1.ResolvedCredential{
		Kind: credentialv1.CredentialKind_CREDENTIAL_KIND_SESSION,
		Material: &credentialv1.ResolvedCredential_Session{
			Session: &credentialv1.SessionCredential{
				SchemaId: "cerebras-session",
				Values: map[string]string{
					"authjs_session_token": token,
				},
			},
		},
	}
}
