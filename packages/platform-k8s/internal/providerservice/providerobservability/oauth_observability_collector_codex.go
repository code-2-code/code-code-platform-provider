package providerobservability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	codexLimitReachedMetric                  = "gen_ai.provider.cli.oauth.codex.limit.reached"
	codexPrimaryWindowUsedPercentMetric      = "gen_ai.provider.cli.oauth.codex.primary.window.used.percent"
	codexPrimaryWindowResetTimestampMetric   = "gen_ai.provider.cli.oauth.codex.primary.window.reset.timestamp.seconds"
	codexPrimaryWindowDurationMetric         = "gen_ai.provider.cli.oauth.codex.primary.window.duration.minutes"
	codexSecondaryWindowUsedPercentMetric    = "gen_ai.provider.cli.oauth.codex.secondary.window.used.percent"
	codexSecondaryWindowResetTimestampMetric = "gen_ai.provider.cli.oauth.codex.secondary.window.reset.timestamp.seconds"
	codexSecondaryWindowDurationMetric       = "gen_ai.provider.cli.oauth.codex.secondary.window.duration.minutes"
	codexPlanTypeCodeMetric                  = "gen_ai.provider.cli.oauth.codex.plan.type.code"
	codexUsageProbeUserAgent                 = "codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal"
)

var codexUsageProbeURL = "https://chatgpt.com/backend-api/wham/usage"

// NewCodexObservabilityCollector creates one Codex collector.
func NewCodexObservabilityCollector() ObservabilityCollector {
	return &codexObservabilityCollector{}
}

func init() {
	registerOAuthCollectorFactory("codex", NewCodexObservabilityCollector)
}

type codexObservabilityCollector struct{}

func (c *codexObservabilityCollector) CollectorID() string {
	return "codex"
}

func (c *codexObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	if strings.TrimSpace(input.AccessToken) == "" {
		return nil, unauthorizedObservabilityError("codex access token is empty")
	}
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: codex oauth observability http client is nil")
	}
	accountID := strings.TrimSpace(input.MaterialValues[materialKeyAccountID])
	if accountID == "" {
		return nil, unauthorizedObservabilityError("codex chatgpt account id is empty")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, codexUsageProbeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: create codex usage operation request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(input.AccessToken))
	request.Header.Set("ChatGPT-Account-Id", accountID)
	request.Header.Set("User-Agent", codexObservabilityUserAgent(input))

	response, err := input.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: execute codex usage operation request: %w", err)
	}
	defer response.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, observabilityMaxBodyReadSize))

	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, unauthorizedObservabilityError(
			fmt.Sprintf("codex usage operation unauthorized: status %d %s", response.StatusCode, strings.TrimSpace(string(bodyBytes))),
		)
	}
	if response.StatusCode == http.StatusTooManyRequests {
		if values, ok := codexUsageLimitGaugeValues("", nil, time.Now().UTC(), bodyBytes); ok {
			return &ObservabilityCollectResult{GaugeRows: gaugeRows(values)}, nil
		}
		return nil, fmt.Errorf("providerobservability: codex usage operation 429 is not usage_limit_reached")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("providerobservability: codex usage operation failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	parsed := codexUsageResponse{}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, fmt.Errorf("providerobservability: decode codex usage operation response: %w", err)
	}
	values, _ := codexUsageLimitGaugeValues(parsed.PlanType, parsed.RateLimit, time.Now().UTC(), nil)
	return &ObservabilityCollectResult{GaugeRows: gaugeRows(values)}, nil
}

type codexUsageResponse struct {
	PlanType  string               `json:"plan_type"`
	RateLimit *codexUsageRateLimit `json:"rate_limit"`
}

type codexUsageRateLimit struct {
	Allowed         bool              `json:"allowed"`
	LimitReached    bool              `json:"limit_reached"`
	PrimaryWindow   *codexUsageWindow `json:"primary_window"`
	SecondaryWindow *codexUsageWindow `json:"secondary_window"`
}

type codexUsageWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	LimitWindowSeconds int64   `json:"limit_window_seconds"`
	ResetAt            int64   `json:"reset_at"`
}

type codexUsageLimitErrorBody struct {
	Error codexUsageLimitErrorPayload `json:"error"`
}

type codexUsageLimitErrorPayload struct {
	Type            string `json:"type"`
	ResetsAt        int64  `json:"resets_at"`
	ResetsInSeconds int64  `json:"resets_in_seconds"`
}

func codexUsageLimitGaugeValues(planType string, rateLimit *codexUsageRateLimit, now time.Time, body []byte) (map[string]float64, bool) {
	values := map[string]float64{
		codexLimitReachedMetric:                  0,
		codexPrimaryWindowUsedPercentMetric:      0,
		codexPrimaryWindowResetTimestampMetric:   0,
		codexPrimaryWindowDurationMetric:         0,
		codexSecondaryWindowUsedPercentMetric:    0,
		codexSecondaryWindowResetTimestampMetric: 0,
		codexSecondaryWindowDurationMetric:       0,
		codexPlanTypeCodeMetric:                  float64(codexPlanTypeCode(planType)),
	}
	if rateLimit != nil {
		values[codexLimitReachedMetric] = boolFloat(rateLimit.LimitReached || !rateLimit.Allowed)
		applyCodexUsageWindow(values, rateLimit.PrimaryWindow, codexPrimaryWindowUsedPercentMetric, codexPrimaryWindowResetTimestampMetric, codexPrimaryWindowDurationMetric)
		applyCodexUsageWindow(values, rateLimit.SecondaryWindow, codexSecondaryWindowUsedPercentMetric, codexSecondaryWindowResetTimestampMetric, codexSecondaryWindowDurationMetric)
		return values, true
	}
	retryAfter, ok := parseCodexUsageLimitRetryAfter(body, now)
	if !ok {
		return values, false
	}
	values[codexLimitReachedMetric] = 1
	values[codexPrimaryWindowUsedPercentMetric] = 100
	values[codexPrimaryWindowResetTimestampMetric] = float64(now.Add(retryAfter).Unix())
	return values, true
}

func applyCodexUsageWindow(values map[string]float64, window *codexUsageWindow, usedMetric, resetMetric, durationMetric string) {
	if window == nil {
		return
	}
	values[usedMetric] = window.UsedPercent
	if window.ResetAt > 0 {
		values[resetMetric] = float64(window.ResetAt)
	}
	if window.LimitWindowSeconds > 0 {
		values[durationMetric] = float64(window.LimitWindowSeconds) / 60
	}
}

func parseCodexUsageLimitRetryAfter(body []byte, now time.Time) (time.Duration, bool) {
	if len(body) == 0 {
		return 0, false
	}
	parsed := codexUsageLimitErrorBody{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, false
	}
	if strings.TrimSpace(parsed.Error.Type) != "usage_limit_reached" {
		return 0, false
	}
	if parsed.Error.ResetsAt > 0 {
		resetAt := time.Unix(parsed.Error.ResetsAt, 0).UTC()
		if resetAt.After(now) {
			return resetAt.Sub(now), true
		}
	}
	if parsed.Error.ResetsInSeconds > 0 {
		return time.Duration(parsed.Error.ResetsInSeconds) * time.Second, true
	}
	return 0, false
}

func codexPlanTypeCode(planType string) int {
	switch strings.TrimSpace(strings.ToLower(planType)) {
	case "guest":
		return 1
	case "free":
		return 2
	case "go":
		return 3
	case "plus":
		return 4
	case "pro":
		return 5
	case "prolite":
		return 6
	case "free_workspace":
		return 7
	case "team":
		return 8
	case "self_serve_business_usage_based":
		return 9
	case "business":
		return 10
	case "enterprise_cbp_usage_based":
		return 11
	case "education":
		return 12
	case "quorum":
		return 13
	case "k12":
		return 14
	case "enterprise":
		return 15
	case "edu":
		return 16
	default:
		return 0
	}
}

func boolFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func codexObservabilityUserAgent(input ObservabilityCollectInput) string {
	if value := strings.TrimSpace(input.ObservabilityUserAgent); value != "" {
		return value
	}
	return codexUsageProbeUserAgent
}
