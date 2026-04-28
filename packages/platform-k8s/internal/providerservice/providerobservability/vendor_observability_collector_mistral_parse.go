package providerobservability

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseMistralBillingGaugeRows parses the billing/v2/usage response body into
// metric gauge rows.
//
// NOTE: The exact Mistral billing API response format is not publicly documented.
// This parser targets the most probable JSON shape based on common billing API
// conventions. Update this function once the endpoint has been tested against a
// live Mistral account and the actual schema is confirmed.
//
// Expected shape (best-effort guess):
//
//	{
//	  "data": [
//	    {
//	      "model": "<model-id>",
//	      "input_tokens": <int>,
//	      "output_tokens": <int>,
//	      "total_tokens": <int> // optional source field, not exported as metric
//	    },
//	    ...
//	  ]
//	}
func parseMistralBillingGaugeRows(body []byte) ([]VendorObservabilityMetricRow, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("providerobservability: mistral billing: decode response: %w (body: %s)", err, truncate(string(body), 256))
	}

	entries, ok := mistralBillingEntries(payload)
	if !ok {
		// The response parsed as JSON but doesn't match the expected shape.
		// Return the raw body in the error to aid schema discovery.
		return nil, fmt.Errorf("providerobservability: mistral billing: response does not include expected 'data' array (body: %s)", truncate(string(body), 512))
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("providerobservability: mistral billing: 'data' array is empty")
	}

	rows := make([]VendorObservabilityMetricRow, 0, len(entries)*2)
	for _, entry := range entries {
		modelID := mistralBillingModelID(entry)
		if modelID == "" {
			continue
		}
		baseLabels := map[string]string{
			"model_id": modelID,
			"resource": "tokens",
			"window":   "day",
		}

		if v, ok := mistralBillingInt(entry, "input_tokens"); ok {
			labels := copyStringMap(baseLabels)
			labels["token_type"] = "input"
			rows = append(rows, VendorObservabilityMetricRow{
				MetricName: mistralBillingTokensMetric,
				Labels:     labels,
				Value:      float64(v),
			})
		}
		if v, ok := mistralBillingInt(entry, "output_tokens"); ok {
			labels := copyStringMap(baseLabels)
			labels["token_type"] = "output"
			rows = append(rows, VendorObservabilityMetricRow{
				MetricName: mistralBillingTokensMetric,
				Labels:     labels,
				Value:      float64(v),
			})
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("providerobservability: mistral billing: no token metrics found in response")
	}
	return rows, nil
}

func mistralBillingEntries(payload map[string]any) ([]map[string]any, bool) {
	raw, ok := payload["data"]
	if !ok {
		return nil, false
	}
	slice, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	entries := make([]map[string]any, 0, len(slice))
	for _, item := range slice {
		if m, ok := item.(map[string]any); ok {
			entries = append(entries, m)
		}
	}
	return entries, true
}

func mistralBillingModelID(entry map[string]any) string {
	for _, key := range []string{"model", "model_id"} {
		if v, ok := entry[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func mistralBillingInt(entry map[string]any, key string) (int64, bool) {
	v, ok := entry[key]
	if !ok {
		return 0, false
	}
	n, ok := numberFromAny(v)
	if !ok {
		return 0, false
	}
	return int64(n), true
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}
