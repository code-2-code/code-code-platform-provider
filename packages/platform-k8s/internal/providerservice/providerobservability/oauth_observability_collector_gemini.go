package providerobservability

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"code-code.internal/platform-k8s/internal/supportservice/clidefinitions/codeassist"
)

const (
	geminiProRemainingAmountMetric        = "gen_ai.provider.cli.oauth.gemini.pro.remaining.amount"
	geminiProRemainingPercentMetric       = "gen_ai.provider.cli.oauth.gemini.pro.remaining.fraction.percent"
	geminiProResetTimestampMetric         = "gen_ai.provider.cli.oauth.gemini.pro.reset.timestamp.seconds"
	geminiFlashRemainingAmountMetric      = "gen_ai.provider.cli.oauth.gemini.flash.remaining.amount"
	geminiFlashRemainingPercentMetric     = "gen_ai.provider.cli.oauth.gemini.flash.remaining.fraction.percent"
	geminiFlashResetTimestampMetric       = "gen_ai.provider.cli.oauth.gemini.flash.reset.timestamp.seconds"
	geminiFlashLiteRemainingAmountMetric  = "gen_ai.provider.cli.oauth.gemini.flash.lite.remaining.amount"
	geminiFlashLiteRemainingPercentMetric = "gen_ai.provider.cli.oauth.gemini.flash.lite.remaining.fraction.percent"
	geminiFlashLiteResetTimestampMetric   = "gen_ai.provider.cli.oauth.gemini.flash.lite.reset.timestamp.seconds"
)

// NewGeminiObservabilityCollector creates one Gemini collector.
func NewGeminiObservabilityCollector() ObservabilityCollector {
	return &geminiObservabilityCollector{}
}

func init() {
	registerOAuthCollectorFactory("gemini-cli", NewGeminiObservabilityCollector)
}

type geminiObservabilityCollector struct{}

type geminiQuotaGroupSummary struct {
	remainingAmount  float64
	hasAmount        bool
	remainingPercent float64
	hasPercent       bool
	resetAt          time.Time
	hasResetAt       bool
}

type geminiQuotaGroupDefinition struct {
	name          string
	modelIDs      []string
	amountMetric  string
	percentMetric string
	resetMetric   string
}

var geminiQuotaGroups = []geminiQuotaGroupDefinition{
	{
		name:          "flash-lite",
		modelIDs:      []string{"gemini-2.5-flash-lite"},
		amountMetric:  geminiFlashLiteRemainingAmountMetric,
		percentMetric: geminiFlashLiteRemainingPercentMetric,
		resetMetric:   geminiFlashLiteResetTimestampMetric,
	},
	{
		name:          "flash",
		modelIDs:      []string{"gemini-3-flash-preview", "gemini-2.5-flash"},
		amountMetric:  geminiFlashRemainingAmountMetric,
		percentMetric: geminiFlashRemainingPercentMetric,
		resetMetric:   geminiFlashResetTimestampMetric,
	},
	{
		name:          "pro",
		modelIDs:      []string{"gemini-3.1-pro-preview", "gemini-3-pro-preview", "gemini-2.5-pro"},
		amountMetric:  geminiProRemainingAmountMetric,
		percentMetric: geminiProRemainingPercentMetric,
		resetMetric:   geminiProResetTimestampMetric,
	},
}

func (c *geminiObservabilityCollector) CollectorID() string {
	return "gemini-cli"
}

func (c *geminiObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	if strings.TrimSpace(input.AccessToken) == "" {
		return nil, unauthorizedObservabilityError("gemini access token is empty")
	}
	projectID := strings.TrimSpace(input.MaterialValues[materialKeyProjectID])
	codeAssistPayload, err := codeassist.LoadGeminiCodeAssist(ctx, input.HTTPClient, input.AccessToken, projectID)
	if err != nil {
		return nil, err
	}
	if resolvedProjectID := codeassist.GeminiProjectID(codeAssistPayload); resolvedProjectID != "" {
		projectID = resolvedProjectID
	}
	tierName := codeassist.GeminiTierName(codeAssistPayload)
	if projectID == "" {
		return nil, fmt.Errorf("providerobservability: gemini project id is empty")
	}
	quotaPayload, err := codeassist.LoadGeminiUserQuota(ctx, input.HTTPClient, input.AccessToken, projectID)
	if err != nil {
		return nil, err
	}
	backfillValues := map[string]string{}
	if projectID != "" {
		backfillValues[materialKeyProjectID] = projectID
	}
	if tierName != "" {
		backfillValues[materialKeyTierName] = tierName
	}
	return &ObservabilityCollectResult{
		GaugeRows:                gaugeRows(geminiQuotaGaugeValues(quotaPayload)),
		CredentialBackfillValues: backfillValues,
	}, nil
}

func geminiQuotaGaugeValues(payload map[string]any) map[string]float64 {
	values := map[string]float64{
		geminiProRemainingAmountMetric:        0,
		geminiProRemainingPercentMetric:       0,
		geminiProResetTimestampMetric:         0,
		geminiFlashRemainingAmountMetric:      0,
		geminiFlashRemainingPercentMetric:     0,
		geminiFlashResetTimestampMetric:       0,
		geminiFlashLiteRemainingAmountMetric:  0,
		geminiFlashLiteRemainingPercentMetric: 0,
		geminiFlashLiteResetTimestampMetric:   0,
	}
	buckets, _ := payload["buckets"].([]any)
	if len(buckets) == 0 {
		return values
	}
	summaries := map[string]geminiQuotaGroupSummary{}
	groupByModel := map[string]geminiQuotaGroupDefinition{}
	for _, group := range geminiQuotaGroups {
		for _, modelID := range group.modelIDs {
			groupByModel[modelID] = group
		}
	}
	for _, raw := range buckets {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		modelID, _ := entry["modelId"].(string)
		group, ok := groupByModel[strings.TrimSpace(modelID)]
		if !ok {
			continue
		}
		summary := summaries[group.name]
		remainingPercent, hasPercent := geminiBucketRemainingPercent(entry)
		remainingAmount, hasAmount := geminiBucketRemainingAmount(entry)
		resetAt, hasResetAt := geminiBucketResetAt(entry)
		if shouldReplaceGeminiQuotaSummary(summary, remainingPercent, hasPercent, remainingAmount, hasAmount) {
			if hasPercent {
				summary.remainingPercent = remainingPercent
				summary.hasPercent = true
			}
			if hasAmount {
				summary.remainingAmount = remainingAmount
				summary.hasAmount = true
			} else {
				summary.remainingAmount = 0
				summary.hasAmount = false
			}
			if hasResetAt {
				summary.resetAt = resetAt
				summary.hasResetAt = true
			} else {
				summary.resetAt = time.Time{}
				summary.hasResetAt = false
			}
		}
		summaries[group.name] = summary
	}
	for _, group := range geminiQuotaGroups {
		summary, ok := summaries[group.name]
		if !ok {
			continue
		}
		if summary.hasAmount {
			values[group.amountMetric] = summary.remainingAmount
		}
		if summary.hasPercent {
			values[group.percentMetric] = summary.remainingPercent
		}
		if summary.hasResetAt {
			values[group.resetMetric] = float64(summary.resetAt.Unix())
		}
	}
	return values
}

func shouldReplaceGeminiQuotaSummary(current geminiQuotaGroupSummary, nextPercent float64, hasNextPercent bool, nextAmount float64, hasNextAmount bool) bool {
	if !current.hasPercent && !current.hasAmount {
		return hasNextPercent || hasNextAmount
	}
	if hasNextPercent {
		return !current.hasPercent || nextPercent < current.remainingPercent
	}
	if hasNextAmount {
		return !current.hasAmount || nextAmount < current.remainingAmount
	}
	return false
}

func geminiBucketRemainingPercent(entry map[string]any) (float64, bool) {
	value, ok := entry["remainingFraction"].(float64)
	if !ok {
		return 0, false
	}
	percent := value * 100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return percent, true
}

func geminiBucketRemainingAmount(entry map[string]any) (float64, bool) {
	switch value := entry["remainingAmount"].(type) {
	case string:
		parsed, err := parseGeminiRemainingAmount(value)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case float64:
		return value, true
	default:
		return 0, false
	}
}

func parseGeminiRemainingAmount(raw string) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("remaining amount is empty")
	}
	return strconv.ParseFloat(trimmed, 64)
}

func geminiBucketResetAt(entry map[string]any) (time.Time, bool) {
	value, _ := entry["resetTime"].(string)
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return time.Time{}, false
		}
	}
	return parsed.UTC(), true
}
