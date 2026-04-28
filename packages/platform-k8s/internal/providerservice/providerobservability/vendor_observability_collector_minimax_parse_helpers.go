package providerobservability

import "strings"

func minimaxModelRemainEntries(payload map[string]any) []map[string]any {
	paths := [][]string{
		{"model_remains"},
		{"data", "model_remains"},
		{"result", "model_remains"},
		{"coding_plan", "model_remains"},
		{"remains", "model_remains"},
	}
	for _, path := range paths {
		value, ok := nestedValue(payload, path...)
		if !ok {
			continue
		}
		entries := entriesFromAny(value)
		if len(entries) > 0 {
			return entries
		}
	}
	return nil
}

func nestedValue(root map[string]any, path ...string) (any, bool) {
	current := any(root)
	for _, key := range path {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = currentMap[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func entriesFromAny(value any) []map[string]any {
	switch typed := value.(type) {
	case []any:
		entries := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if ok {
				entries = append(entries, entry)
			}
		}
		return entries
	case map[string]any:
		entries := make([]map[string]any, 0, len(typed))
		for modelID, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if strings.TrimSpace(stringFromAny(entry["model_name"])) == "" {
				entry = copyMap(entry)
				entry["model_name"] = modelID
			}
			entries = append(entries, entry)
		}
		return entries
	default:
		return nil
	}
}

func copyMap(source map[string]any) map[string]any {
	copied := make(map[string]any, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func minimaxQuotaModelID(entry map[string]any) string {
	for _, key := range []string{"model_name", "model_id", "modelId", "model", "name"} {
		name := strings.TrimSpace(stringFromAny(entryValue(entry, key)))
		if name == "" {
			continue
		}
		normalized := strings.ToLower(name)
		if strings.HasPrefix(normalized, "minimax-m") ||
			strings.HasPrefix(normalized, "codex-minimax-m") ||
			normalized == "coding-plan-vlm" ||
			normalized == "coding-plan-search" {
			return name
		}
	}
	return ""
}

func minimaxRemainingCount(entry map[string]any) (float64, bool) {
	if value, ok := minimaxNumber(entry,
		"remaining_count", "remain_count", "remaining_amount", "remainingAmount", "remaining"); ok {
		return value, true
	}
	// MiniMax's current remains payload reports the live remaining count in the
	// usage-shaped field names, so we must not subtract it from total again.
	return minimaxNumber(entry,
		"current_interval_usage_count", "interval_usage_count", "usage_count", "used_count")
}

func minimaxTotalCount(entry map[string]any) (float64, bool) {
	return minimaxNumber(entry,
		"total_count", "limit_count", "total_amount", "totalAmount", "total",
		"current_interval_total_count", "interval_total_count")
}

func minimaxNumber(entry map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := numberFromAny(entryValue(entry, key))
		if ok {
			return value, true
		}
	}
	return 0, false
}

func minimaxRemainingPercent(entry map[string]any, remaining float64, hasRemaining bool, total float64, hasTotal bool) (float64, bool) {
	if value, ok := minimaxNumber(entry,
		"remaining_fraction_percent", "remaining_percent", "remain_percent"); ok {
		return normalizePercent(value)
	}
	if value, ok := minimaxNumber(entry,
		"remaining_fraction", "remain_fraction", "remainingFraction", "remainFraction"); ok {
		return normalizePercent(value)
	}
	if hasRemaining && hasTotal && total > 0 {
		return remaining / total * 100, true
	}
	return 0, false
}
