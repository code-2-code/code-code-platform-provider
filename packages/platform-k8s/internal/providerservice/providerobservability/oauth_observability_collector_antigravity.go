package providerobservability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"code-code.internal/platform-k8s/internal/supportservice/clidefinitions/codeassist"
)

const (
	antigravityQuotaRemainingPercentMetric = "gen_ai.provider.cli.oauth.antigravity.model.quota.remaining.fraction.percent"
	antigravityQuotaResetTimestampMetric   = "gen_ai.provider.cli.oauth.antigravity.model.quota.reset.timestamp.seconds"
	antigravityQuotaResponseMaxBodySize    = 1 << 20
)

var antigravityFetchAvailableModelsURLs = []string{
	"https://daily-cloudcode-pa.sandbox.googleapis.com/v1internal:fetchAvailableModels",
	"https://daily-cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels",
	"https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels",
}

func NewAntigravityObservabilityCollector() ObservabilityCollector {
	return &antigravityObservabilityCollector{}
}

func init() {
	registerOAuthCollectorFactory("antigravity", NewAntigravityObservabilityCollector)
}

type antigravityObservabilityCollector struct{}

func (c *antigravityObservabilityCollector) CollectorID() string {
	return "antigravity"
}

func (c *antigravityObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	if strings.TrimSpace(input.AccessToken) == "" {
		return nil, unauthorizedObservabilityError("antigravity access token is empty")
	}
	projectID := strings.TrimSpace(input.MaterialValues[materialKeyProjectID])
	codeAssistPayload, err := codeassist.LoadAntigravityCodeAssistWithProject(ctx, input.HTTPClient, input.AccessToken, projectID)
	if err != nil {
		return nil, err
	}
	if resolvedProjectID := codeassist.GeminiProjectID(codeAssistPayload); resolvedProjectID != "" {
		projectID = resolvedProjectID
	}
	tierName := codeassist.AntigravityTierName(codeAssistPayload)
	if projectID == "" {
		if !codeassist.AntigravityShouldOnboard(codeAssistPayload) {
			return nil, codeassist.AntigravityProjectResolutionError(codeAssistPayload)
		}
		projectID, err = codeassist.OnboardAntigravityUserWithProject(ctx, input.HTTPClient, input.AccessToken, codeassist.AntigravityDefaultTierID(codeAssistPayload), projectID)
		if err != nil {
			if codeassist.IsAntigravityOnboardMissingProjectID(err) {
				return nil, codeassist.AntigravityProjectResolutionError(codeAssistPayload)
			}
			return nil, err
		}
	}
	if projectID == "" {
		return nil, fmt.Errorf("providerobservability: antigravity project id is empty")
	}
	payload, err := loadAntigravityAvailableModels(ctx, input.HTTPClient, input.AccessToken, projectID, input.ModelCatalogUserAgent)
	if err != nil {
		return nil, err
	}
	backfillValues := map[string]string{
		materialKeyProjectID: projectID,
	}
	if tierName != "" {
		backfillValues[materialKeyTierName] = tierName
	}
	return &ObservabilityCollectResult{
		GaugeRows:                antigravityQuotaRows(payload),
		CredentialBackfillValues: backfillValues,
	}, nil
}

func loadAntigravityAvailableModels(ctx context.Context, httpClient *http.Client, accessToken, projectID string, userAgent string) (map[string]any, error) {
	body, err := json.Marshal(map[string]string{"project": strings.TrimSpace(projectID)})
	if err != nil {
		return nil, fmt.Errorf("providerobservability: marshal antigravity fetchAvailableModels request: %w", err)
	}
	var lastErr error
	for _, endpoint := range antigravityFetchAvailableModelsURLs {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("providerobservability: create antigravity fetchAvailableModels request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
		request.Header.Set("User-Agent", antigravityModelCatalogUserAgent(userAgent))
		request.Header.Set("X-Goog-Api-Client", codeassist.AntigravityAPIClient)
		request.Header.Set("Client-Metadata", codeassist.AntigravityClientMetadata)
		response, err := httpClient.Do(request)
		if err != nil {
			lastErr = fmt.Errorf("providerobservability: execute antigravity fetchAvailableModels request: %w", err)
			continue
		}
		payload, handled, err := decodeAntigravityQuotaResponse(response)
		if handled {
			return payload, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func decodeAntigravityQuotaResponse(response *http.Response) (map[string]any, bool, error) {
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, antigravityQuotaResponseMaxBodySize))
	if err != nil {
		return nil, true, fmt.Errorf("providerobservability: read antigravity fetchAvailableModels response: %w", err)
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, true, unauthorizedObservabilityError(
			fmt.Sprintf("antigravity fetchAvailableModels unauthorized: status %d %s", response.StatusCode, strings.TrimSpace(string(bodyBytes))),
		)
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		return nil, false, fmt.Errorf("providerobservability: antigravity fetchAvailableModels retryable status %d", response.StatusCode)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, true, fmt.Errorf("providerobservability: antigravity fetchAvailableModels failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	parsed := map[string]any{}
	if len(bytes.TrimSpace(bodyBytes)) == 0 {
		return parsed, true, nil
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, true, fmt.Errorf("providerobservability: decode antigravity fetchAvailableModels response: %w", err)
	}
	return parsed, true, nil
}

func antigravityQuotaRows(payload map[string]any) []ObservabilityMetricRow {
	models, _ := payload["models"].(map[string]any)
	if len(models) == 0 {
		return nil
	}
	rows := make([]ObservabilityMetricRow, 0, len(models)*2)
	for modelID, raw := range models {
		trimmedModelID := strings.TrimSpace(modelID)
		if !antigravityQuotaModelSupported(trimmedModelID) {
			continue
		}
		model, _ := raw.(map[string]any)
		quotaInfo, _ := model["quotaInfo"].(map[string]any)
		if len(quotaInfo) == 0 {
			continue
		}
		labels := map[string]string{"model_id": trimmedModelID}
		if percent, ok := quotaInfo["remainingFraction"].(float64); ok {
			rows = append(rows, ObservabilityMetricRow{
				MetricName: antigravityQuotaRemainingPercentMetric,
				Labels:     labels,
				Value:      clampPercent(percent * 100),
			})
		}
		if resetAt, ok := parseRFC3339Timestamp(quotaInfo["resetTime"]); ok {
			rows = append(rows, ObservabilityMetricRow{
				MetricName: antigravityQuotaResetTimestampMetric,
				Labels:     labels,
				Value:      float64(resetAt.Unix()),
			})
		}
	}
	return rows
}

func antigravityQuotaModelSupported(modelID string) bool {
	normalized := strings.ToLower(strings.TrimSpace(modelID))
	return strings.HasPrefix(normalized, "gemini") ||
		strings.HasPrefix(normalized, "claude") ||
		strings.HasPrefix(normalized, "gpt") ||
		strings.HasPrefix(normalized, "image") ||
		strings.HasPrefix(normalized, "imagen")
}

func parseRFC3339Timestamp(raw any) (time.Time, bool) {
	value, _ := raw.(string)
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

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func antigravityModelCatalogUserAgent(userAgent string) string {
	if trimmed := strings.TrimSpace(userAgent); trimmed != "" {
		return trimmed
	}
	return codeassist.AntigravityUserAgent
}
