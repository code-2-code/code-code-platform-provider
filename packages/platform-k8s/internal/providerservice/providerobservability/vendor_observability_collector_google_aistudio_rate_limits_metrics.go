package providerobservability

import (
	"fmt"
	"strconv"
	"time"
)

func googleAIStudioMetricRows(models []googleAIStudioQuotaModel, now time.Time) []ObservabilityMetricRow {
	rows := make([]ObservabilityMetricRow, 0, len(models)*4)
	for _, model := range models {
		for _, limit := range model.Limits {
			labels := map[string]string{
				"model_id":       model.ModelID,
				"model_category": model.Category,
				"resource":       limit.Resource,
				"window":         limit.Window,
				"preview":        strconv.FormatBool(model.Preview),
				"quota_type":     limit.QuotaType,
			}
			rows = append(rows, ObservabilityMetricRow{
				MetricName: googleAIStudioQuotaLimitMetric,
				Labels:     labels,
				Value:      limit.Value,
			})
			if limit.HasRemaining {
				rows = append(rows, ObservabilityMetricRow{
					MetricName: providerQuotaRemainingMetric,
					Labels:     labels,
					Value:      limit.Remaining,
				})
			}
			if resetAt, ok := googleAIStudioQuotaResetTimestamp(limit.Window, now); ok {
				rows = append(rows, ObservabilityMetricRow{
					MetricName: providerQuotaResetTimestampMetric,
					Labels:     labels,
					Value:      resetAt,
				})
			}
		}
	}
	return rows
}

func googleAIStudioQuotaResetTimestamp(window string, now time.Time) (float64, bool) {
	switch window {
	case "minute":
		return float64(now.UTC().Truncate(time.Minute).Add(time.Minute).Unix()), true
	case "day":
		utc := now.UTC()
		nextDayUTC := time.Date(utc.Year(), utc.Month(), utc.Day()+1, 0, 0, 0, 0, time.UTC)
		return float64(nextDayUTC.Unix()), true
	default:
		return 0, false
	}
}

func googleAIStudioQuotaResource(code int) string {
	switch code {
	case 1, 9:
		return "requests"
	case 2, 8:
		return "tokens"
	case 5:
		return "images"
	case 6:
		return "videos"
	default:
		return "other"
	}
}

func googleAIStudioQuotaWindow(code int) string {
	switch code {
	case 1:
		return "minute"
	case 2:
		return "day"
	default:
		return fmt.Sprintf("code_%d", code)
	}
}

func googleAIStudioQuotaType(resourceCode, windowCode int) string {
	resource := googleAIStudioQuotaResource(resourceCode)
	window := googleAIStudioQuotaWindow(windowCode)
	switch {
	case resource == "requests" && window == "minute":
		return "RPM"
	case resource == "requests" && window == "day":
		return "RPD"
	case resource == "tokens" && window == "minute":
		return "TPM"
	case resource == "tokens" && window == "day":
		return "TPD"
	default:
		return ""
	}
}

func errGoogleAIStudioNoRows(method string) error {
	return fmt.Errorf("no supported rows found in %s response", method)
}
