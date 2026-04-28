package providerobservability

import (
	"context"
	"fmt"
	"strings"

	"github.com/OpenRouterTeam/go-sdk"
)

const openrouterCollectorID = "openrouter-quotas"

func init() {
	registerVendorCollectorFactory(openrouterCollectorID, NewOpenRouterObservabilityCollector)
}

func NewOpenRouterObservabilityCollector() ObservabilityCollector {
	return &openrouterObservabilityCollector{}
}

type openrouterObservabilityCollector struct{}

func (c *openrouterObservabilityCollector) CollectorID() string {
	return openrouterCollectorID
}

func (c *openrouterObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: openrouter quotas: http client is nil")
	}

	apiKey := strings.TrimSpace(observabilityCredentialToken(input.ObservabilityCredential))
	if apiKey == "" {
		apiKey = strings.TrimSpace(input.APIKey)
	}

	if apiKey == "" {
		return nil, unauthorizedObservabilityError("openrouter quotas: api key is required; configure provider authentication")
	}

	client := openrouter.New(
		openrouter.WithSecurity(apiKey),
		openrouter.WithClient(input.HTTPClient),
	)

	res, err := client.Analytics.GetUserActivity(ctx, nil, nil, nil)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unauthorized") || strings.Contains(strings.ToLower(err.Error()), "invalid api key") {
			return nil, unauthorizedObservabilityError(err.Error())
		}
		return nil, fmt.Errorf("providerobservability: fetch openrouter user activity: %w", err)
	}

	if res == nil || len(res.Data) == 0 {
		return &ObservabilityCollectResult{}, nil
	}

	var rows []ObservabilityMetricRow
	for _, item := range res.Data {
		labels := map[string]string{
			"window": "30d",
		}
		if item.Model != "" {
			labels["model_id"] = item.Model
		}

		if item.Requests > 0 {
			requestLabels := copyStringMap(labels)
			requestLabels["resource"] = "requests"
			rows = append(rows, ObservabilityMetricRow{
				MetricName: providerUsageRequestsMetric,
				Labels:     requestLabels,
				Value:      float64(item.Requests),
			})
		}
		if item.Usage > 0 {
			costLabels := copyStringMap(labels)
			costLabels["resource"] = "cost"
			rows = append(rows, ObservabilityMetricRow{
				MetricName: providerUsageCostUSDMetric,
				Labels:     costLabels,
				Value:      item.Usage,
			})
		}
		if item.PromptTokens > 0 {
			inputLabels := copyStringMap(labels)
			inputLabels["resource"] = "tokens"
			inputLabels["token_type"] = "input"
			rows = append(rows, ObservabilityMetricRow{
				MetricName: providerUsageTokensMetric,
				Labels:     inputLabels,
				Value:      float64(item.PromptTokens),
			})
		}
		if item.CompletionTokens > 0 {
			outputLabels := copyStringMap(labels)
			outputLabels["resource"] = "tokens"
			outputLabels["token_type"] = "output"
			rows = append(rows, ObservabilityMetricRow{
				MetricName: providerUsageTokensMetric,
				Labels:     outputLabels,
				Value:      float64(item.CompletionTokens),
			})
		}
	}

	return &ObservabilityCollectResult{
		GaugeRows: rows,
	}, nil
}
