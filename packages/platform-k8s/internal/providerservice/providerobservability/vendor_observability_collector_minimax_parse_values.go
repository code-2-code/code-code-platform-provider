package providerobservability

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
)

func normalizePercent(value float64) (float64, bool) {
	switch {
	case math.IsNaN(value), math.IsInf(value, 0):
		return 0, false
	case value >= 0 && value <= 1:
		return value * 100, true
	case value > 1 && value <= 100:
		return value, true
	default:
		return 0, false
	}
}

func minimaxResetTimestamp(entry map[string]any) (float64, bool) {
	for _, key := range []string{
		"reset_timestamp", "reset_time", "reset_at", "resetTime", "resetAt",
		"end_time", "interval_end_time", "current_interval_end_time",
	} {
		if value, ok := timestampSecondsFromAny(entryValue(entry, key)); ok {
			return value, true
		}
	}
	return 0, false
}

func entryValue(entry map[string]any, key string) any {
	if entry == nil {
		return nil
	}
	if value, ok := entry[key]; ok {
		return value
	}
	if nested, ok := entry["quotaInfo"].(map[string]any); ok {
		if value, ok := nested[key]; ok {
			return value
		}
	}
	if nested, ok := entry["quota_info"].(map[string]any); ok {
		if value, ok := nested[key]; ok {
			return value
		}
	}
	return nil
}

func timestampSecondsFromAny(value any) (float64, bool) {
	if numeric, ok := numberFromAny(value); ok {
		if numeric > 1e12 {
			numeric = numeric / 1000
		}
		if numeric > 0 {
			return numeric, true
		}
		return 0, false
	}
	raw := strings.TrimSpace(stringFromAny(value))
	if raw == "" {
		return 0, false
	}
	if numeric, err := strconv.ParseFloat(raw, 64); err == nil {
		if numeric > 1e12 {
			numeric = numeric / 1000
		}
		if numeric > 0 {
			return numeric, true
		}
		return 0, false
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return float64(parsed.UTC().Unix()), true
		}
	}
	return 0, false
}

func numberFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func stringFromAny(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}
