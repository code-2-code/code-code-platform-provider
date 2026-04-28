package providerobservability

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseMeituanTokenUsageGaugeRows parses the tokenUsage API response body into
// metric gauge rows.
//
// NOTE: The LongCat tokenUsage API response format is not publicly documented.
// This parser targets the most probable JSON shapes based on common patterns
// for Chinese AI platform billing APIs. Update this function once the endpoint
// has been tested against a live LongCat account and the actual schema is confirmed.
//
// Expected shape (best-effort guess):
//
//	{
//	  "code": 200,
//	  "data": [
//	    {
//	      "modelId": "<model-id>",
//	      "inputTokens": <int>,
//	      "outputTokens": <int>,
//	      "totalTokens": <int> // optional source field, not exported as metric
//	    },
//	    ...
//	  ]
//	}
//
// Alternative shape with snake_case:
//
//	{
//	  "code": 200,
//	  "data": {
//	    "usages": [
//	      {
//	        "model_id": "<model-id>",
//	        "input_tokens": <int>,
//	        "output_tokens": <int>,
//	        "total_tokens": <int> // optional source field, not exported as metric
//	      }
//	    ]
//	  }
//	}
func parseMeituanTokenUsageGaugeRows(body []byte) ([]VendorObservabilityMetricRow, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("providerobservability: meituan longcat token usage: decode response: %w (body: %s)", err, truncate(string(body), 256))
	}

	entries, ok := meituanTokenUsageEntries(payload)
	if !ok {
		return nil, fmt.Errorf("providerobservability: meituan longcat token usage: response does not include expected usage array (body: %s)", truncate(string(body), 512))
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("providerobservability: meituan longcat token usage: usage array is empty")
	}

	rows := make([]VendorObservabilityMetricRow, 0, len(entries)*2)
	for _, entry := range entries {
		modelID := meituanTokenUsageModelID(entry)
		if modelID == "" {
			continue
		}
		baseLabels := map[string]string{
			"model_id": modelID,
			"resource": "tokens",
			"window":   "day",
		}

		if v, ok := meituanTokenUsageInt(entry, "inputTokens", "input_tokens"); ok {
			labels := copyStringMap(baseLabels)
			labels["token_type"] = "input"
			rows = append(rows, VendorObservabilityMetricRow{
				MetricName: meituanTokenUsageMetric,
				Labels:     labels,
				Value:      float64(v),
			})
		}
		if v, ok := meituanTokenUsageInt(entry, "outputTokens", "output_tokens"); ok {
			labels := copyStringMap(baseLabels)
			labels["token_type"] = "output"
			rows = append(rows, VendorObservabilityMetricRow{
				MetricName: meituanTokenUsageMetric,
				Labels:     labels,
				Value:      float64(v),
			})
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("providerobservability: meituan longcat token usage: no token metrics found in response")
	}
	return rows, nil
}

// meituanTokenUsageEntries extracts the usage entry array from the payload,
// trying multiple known shapes.
func meituanTokenUsageEntries(payload map[string]any) ([]map[string]any, bool) {
	// Shape 1: {"data": [...]}
	if raw, ok := payload["data"]; ok {
		if slice, ok := raw.([]any); ok {
			return meituanSliceToEntries(slice), true
		}
		// Shape 2: {"data": {"usages": [...]}}
		if obj, ok := raw.(map[string]any); ok {
			for _, key := range []string{"usages", "usage", "items", "list"} {
				if inner, ok := obj[key]; ok {
					if slice, ok := inner.([]any); ok {
						return meituanSliceToEntries(slice), true
					}
				}
			}
		}
	}
	return nil, false
}

func meituanSliceToEntries(slice []any) []map[string]any {
	entries := make([]map[string]any, 0, len(slice))
	for _, item := range slice {
		if m, ok := item.(map[string]any); ok {
			entries = append(entries, m)
		}
	}
	return entries
}

// meituanTokenUsageModelID extracts the model identifier from an entry,
// trying both camelCase and snake_case conventions.
func meituanTokenUsageModelID(entry map[string]any) string {
	for _, key := range []string{"modelId", "model_id", "model", "modelName", "model_name"} {
		if v, ok := entry[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// meituanTokenUsageInt extracts an integer value from an entry,
// trying each candidate key in order.
func meituanTokenUsageInt(entry map[string]any, keys ...string) (int64, bool) {
	for _, key := range keys {
		v, ok := entry[key]
		if !ok {
			continue
		}
		n, ok := numberFromAny(v)
		if !ok {
			continue
		}
		return int64(n), true
	}
	return 0, false
}

func copyStringMap(source map[string]string) map[string]string {
	copied := make(map[string]string, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}
