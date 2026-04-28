package providerobservability

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseMinimaxRemainsGaugeRows(body []byte) ([]ObservabilityMetricRow, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("providerobservability: decode minimax remains response: %w", err)
	}
	if statusCode, statusMessage, ok := minimaxBaseResponse(payload); ok && statusCode != 0 {
		if minimaxUnauthorizedMessage(statusMessage) {
			return nil, unauthorizedObservabilityError(
				fmt.Sprintf("minimax remains unauthorized: base_resp status_code=%d status_msg=%q", statusCode, statusMessage),
			)
		}
		return nil, fmt.Errorf("providerobservability: minimax remains base_resp status_code=%d status_msg=%q", statusCode, statusMessage)
	}
	entries := minimaxModelRemainEntries(payload)
	if len(entries) == 0 {
		return nil, fmt.Errorf("providerobservability: minimax remains response does not include model_remains")
	}
	rows := make([]ObservabilityMetricRow, 0, len(entries)*4)
	for _, entry := range entries {
		modelID := minimaxQuotaModelID(entry)
		if modelID == "" {
			continue
		}
		labels := map[string]string{
			"model_id": modelID,
			"resource": "requests",
			"window":   "day",
		}
		remainingCount, hasRemaining := minimaxRemainingCount(entry)
		totalCount, hasTotal := minimaxTotalCount(entry)
		percent, hasPercent := minimaxRemainingPercent(entry, remainingCount, hasRemaining, totalCount, hasTotal)
		resetTimestamp, hasResetTimestamp := minimaxResetTimestamp(entry)

		if hasRemaining {
			rows = append(rows, ObservabilityMetricRow{
				MetricName: minimaxTextRemainingCountMetric,
				Labels:     labels,
				Value:      remainingCount,
			})
		}
		if hasTotal {
			rows = append(rows, ObservabilityMetricRow{
				MetricName: minimaxTextTotalCountMetric,
				Labels:     labels,
				Value:      totalCount,
			})
		}
		if hasPercent {
			rows = append(rows, ObservabilityMetricRow{
				MetricName: minimaxTextRemainingPercentMetric,
				Labels:     labels,
				Value:      percent,
			})
		}
		if hasResetTimestamp {
			rows = append(rows, ObservabilityMetricRow{
				MetricName: minimaxTextResetTimestampMetric,
				Labels:     labels,
				Value:      resetTimestamp,
			})
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("providerobservability: minimax remains does not include supported coding plan model quota fields")
	}
	return rows, nil
}

func minimaxBaseResponse(payload map[string]any) (int64, string, bool) {
	baseRaw, ok := payload["base_resp"]
	if !ok {
		return 0, "", false
	}
	base, ok := baseRaw.(map[string]any)
	if !ok {
		return 0, "", false
	}
	statusCode, _ := numberFromAny(base["status_code"])
	return int64(statusCode), strings.TrimSpace(stringFromAny(base["status_msg"])), true
}

func minimaxUnauthorizedMessage(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "unauthorized") ||
		strings.Contains(normalized, "invalid") ||
		strings.Contains(normalized, "log in again") ||
		strings.Contains(normalized, "cookie")
}
