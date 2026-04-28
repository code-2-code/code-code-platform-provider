package providerobservability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"code-code.internal/platform-k8s/internal/egressauth"
)

const (
	internlmDailyQuotaLimitMetric        = providerQuotaLimitMetric
	internlmDailyQuotaUsageMetric        = providerQuotaUsageMetric
	internlmDailyQuotaRemainingMetric    = providerQuotaRemainingMetric
	internlmDailyQuotaUsagePercentMetric = providerQuotaUsageFractionPercentMetric

	internlmCollectorID = "internlm-quotas"

	internlmStatisticsURL = "https://internlm.intern-ai.org.cn/puyu/statistics/user/api"
)

func init() {
	registerVendorCollectorFactory(internlmCollectorID, NewInternlmObservabilityCollector)
}

// NewInternlmObservabilityCollector returns a collector that probes
// InternLM (书生) daily token quota and usage via the console statistics API.
// Requires one management-plane JWT token resolved from account override or vendor fallback credential.
func NewInternlmObservabilityCollector() ObservabilityCollector {
	return &internlmObservabilityCollector{}
}

type internlmObservabilityCollector struct{}

func (c *internlmObservabilityCollector) CollectorID() string {
	return internlmCollectorID
}

func (c *internlmObservabilityCollector) AuthAdapterID() string {
	return egressauth.AuthAdapterBearerSessionID
}

func (c *internlmObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: internlm quotas: http client is nil")
	}

	token := observabilityCredentialToken(input.ObservabilityCredential)
	if token == "" {
		return nil, unauthorizedObservabilityError("internlm quotas: jwt token is required; configure provider observability authentication")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, internlmStatisticsURL, strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("providerobservability: internlm quotas: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := input.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: internlm quotas: execute request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, observabilityMaxBodyReadSize))

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, unauthorizedObservabilityError(
			fmt.Sprintf("internlm quotas: unauthorized: status %d %s", resp.StatusCode, strings.TrimSpace(string(body))),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("providerobservability: internlm quotas: failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	rows, err := parseInternlmStatisticsGaugeRows(body)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("providerobservability: internlm quotas: no quota data collected")
	}
	return &ObservabilityCollectResult{GaugeRows: rows}, nil
}

// internlmStatisticsResponse represents the statistics API response.
// InternLM currently exposes daily coding quota in month_* fields.
type internlmStatisticsResponse struct {
	Code int                    `json:"code"`
	Data internlmStatisticsData `json:"data"`
}

type internlmStatisticsData struct {
	MonthUsed  internlmUsageBucket `json:"month_used"`
	MonthQuota internlmUsageBucket `json:"month_quota"`
}

type internlmUsageBucket struct {
	Calls        float64 `json:"calls"`
	InputTokens  float64 `json:"input_tokens"`
	OutputTokens float64 `json:"output_tokens"`
}

// parseInternlmStatisticsGaugeRows converts the statistics response into
// metric gauge rows for daily input and output token quota families.
func parseInternlmStatisticsGaugeRows(body []byte) ([]ObservabilityMetricRow, error) {
	var resp internlmStatisticsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("providerobservability: internlm quotas: decode response: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("providerobservability: internlm quotas: api returned error code %d", resp.Code)
	}

	type entry struct {
		tokenType string
		limit     float64
		usage     float64
	}
	entries := []entry{
		{"input", resp.Data.MonthQuota.InputTokens, resp.Data.MonthUsed.InputTokens},
		{"output", resp.Data.MonthQuota.OutputTokens, resp.Data.MonthUsed.OutputTokens},
	}

	var rows []ObservabilityMetricRow
	for _, e := range entries {
		if e.limit <= 0 {
			continue
		}
		remaining := e.limit - e.usage
		if remaining < 0 {
			remaining = 0
		}
		usagePercent := (e.usage / e.limit) * 100
		labels := map[string]string{
			"window":     "day",
			"resource":   "tokens",
			"token_type": e.tokenType,
		}
		rows = append(rows,
			ObservabilityMetricRow{MetricName: internlmDailyQuotaLimitMetric, Labels: labels, Value: e.limit},
			ObservabilityMetricRow{MetricName: internlmDailyQuotaUsageMetric, Labels: labels, Value: e.usage},
			ObservabilityMetricRow{MetricName: internlmDailyQuotaRemainingMetric, Labels: labels, Value: remaining},
			ObservabilityMetricRow{MetricName: internlmDailyQuotaUsagePercentMetric, Labels: labels, Value: usagePercent},
		)
	}
	return rows, nil
}
