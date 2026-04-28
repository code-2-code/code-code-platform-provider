package providerobservability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const googleAIStudioMetricTimeSeriesLastHourCode = 3

var googleAIStudioMetricTimeSeriesDailyAggregationCode = 2

type googleAIStudioMetricTimeSeriesRequest struct {
	TierCode        int
	QuotaType       string
	ResourceCode    int
	SeriesCode      int
	AggregationCode *int
}

type googleAIStudioMetricTimeSeriesDescriptor struct {
	QuotaType       string
	ResourceCode    int
	SeriesCode      int
	AggregationCode *int
}

var googleAIStudioMetricTimeSeriesDescriptors = []googleAIStudioMetricTimeSeriesDescriptor{
	{
		QuotaType:    "RPM",
		ResourceCode: 1,
		SeriesCode:   2,
	},
	{
		QuotaType:    "TPM",
		ResourceCode: 2,
		SeriesCode:   2,
	},
	{
		QuotaType:       "RPD",
		ResourceCode:    1,
		SeriesCode:      1,
		AggregationCode: &googleAIStudioMetricTimeSeriesDailyAggregationCode,
	},
}

func googleAIStudioMetricTimeSeriesPayload(projectPath string, request googleAIStudioMetricTimeSeriesRequest) ([]byte, error) {
	path := strings.TrimSpace(projectPath)
	if path == "" {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: FetchMetricTimeSeries project path is required")
	}
	if request.TierCode <= 0 {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: FetchMetricTimeSeries tier code is required")
	}
	if request.ResourceCode <= 0 || request.SeriesCode <= 0 {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: FetchMetricTimeSeries metric shape is required")
	}
	var aggregationCode any
	if request.AggregationCode != nil {
		aggregationCode = *request.AggregationCode
	}
	return json.Marshal([]any{
		nil,
		nil,
		nil,
		nil,
		googleAIStudioMetricTimeSeriesLastHourCode,
		nil,
		request.SeriesCode,
		path,
		aggregationCode,
		[]int{request.TierCode},
		[]int{request.ResourceCode},
	})
}

func (c *googleAIStudioObservabilityCollector) enrichGoogleAIStudioMetricTimeSeriesRows(
	ctx context.Context,
	httpClient *http.Client,
	input googleAIStudioRPCCallInput,
	models []googleAIStudioQuotaModel,
) ([]googleAIStudioQuotaModel, error) {
	tierCode := googleAIStudioModelsTierCode(models)
	if tierCode <= 0 {
		return models, nil
	}
	descriptors := googleAIStudioMetricTimeSeriesDescriptorsForModels(models)
	if len(descriptors) == 0 {
		return models, nil
	}
	targetModelIDs := googleAIStudioMetricTimeSeriesTargetModelIDs(models)
	if len(targetModelIDs) == 0 {
		return models, nil
	}
	usageByKey := map[googleAIStudioMetricTimeSeriesUsageKey]float64{}
	fetchedQuotaTypes := map[string]struct{}{}
	for _, descriptor := range descriptors {
		body, err := c.call(ctx, httpClient, googleAIStudioRPCCallInput{
			Method:        "FetchMetricTimeSeries",
			Authorization: input.Authorization,
			AuthUser:      input.AuthUser,
			PageAPIKey:    input.PageAPIKey,
			CookieHeader:  input.CookieHeader,
			Origin:        input.Origin,
			ProjectPath:   input.ProjectPath,
			MetricTimeSeries: googleAIStudioMetricTimeSeriesRequest{
				TierCode:        tierCode,
				QuotaType:       descriptor.QuotaType,
				ResourceCode:    descriptor.ResourceCode,
				SeriesCode:      descriptor.SeriesCode,
				AggregationCode: descriptor.AggregationCode,
			},
		})
		if err != nil {
			return models, err
		}
		payload, err := decodeGoogleAIStudioRPCBody(body)
		if err != nil {
			return nil, fmt.Errorf("providerobservability: google ai studio quotas: decode FetchMetricTimeSeries %s: %w", descriptor.QuotaType, err)
		}
		fetchedQuotaTypes[descriptor.QuotaType] = struct{}{}
		for modelID, value := range parseGoogleAIStudioMetricTimeSeriesUsage(payload, targetModelIDs) {
			usageByKey[newGoogleAIStudioMetricTimeSeriesUsageKey(modelID, descriptor.QuotaType)] = value
		}
	}
	return googleAIStudioApplyMetricTimeSeriesUsage(models, fetchedQuotaTypes, usageByKey), nil
}

func googleAIStudioMetricTimeSeriesTargetModelIDs(models []googleAIStudioQuotaModel) map[string]struct{} {
	targets := make(map[string]struct{}, len(models))
	for _, model := range models {
		modelID := strings.TrimSpace(model.ModelID)
		if modelID == "" {
			continue
		}
		targets[modelID] = struct{}{}
	}
	return targets
}

func googleAIStudioMetricTimeSeriesDescriptorsForModels(models []googleAIStudioQuotaModel) []googleAIStudioMetricTimeSeriesDescriptor {
	wantedQuotaTypes := map[string]struct{}{}
	for _, model := range models {
		for _, limit := range model.Limits {
			wantedQuotaTypes[limit.QuotaType] = struct{}{}
		}
	}
	descriptors := make([]googleAIStudioMetricTimeSeriesDescriptor, 0, len(googleAIStudioMetricTimeSeriesDescriptors))
	for _, descriptor := range googleAIStudioMetricTimeSeriesDescriptors {
		if _, ok := wantedQuotaTypes[descriptor.QuotaType]; ok {
			descriptors = append(descriptors, descriptor)
		}
	}
	return descriptors
}

func googleAIStudioModelsTierCode(models []googleAIStudioQuotaModel) int {
	for _, model := range models {
		if model.TierCode > 0 {
			return model.TierCode
		}
	}
	return 0
}

func googleAIStudioApplyMetricTimeSeriesUsage(
	models []googleAIStudioQuotaModel,
	fetchedQuotaTypes map[string]struct{},
	usageByKey map[googleAIStudioMetricTimeSeriesUsageKey]float64,
) []googleAIStudioQuotaModel {
	for modelIndex := range models {
		for limitIndex := range models[modelIndex].Limits {
			limit := &models[modelIndex].Limits[limitIndex]
			if _, ok := fetchedQuotaTypes[limit.QuotaType]; !ok {
				continue
			}
			used := usageByKey[newGoogleAIStudioMetricTimeSeriesUsageKey(models[modelIndex].ModelID, limit.QuotaType)]
			remaining := limit.Value - used
			if remaining < 0 {
				remaining = 0
			}
			limit.Remaining = remaining
			limit.HasRemaining = true
		}
	}
	return models
}

func parseGoogleAIStudioMetricTimeSeriesUsage(payload []any, targetModelIDs map[string]struct{}) map[string]float64 {
	usageByModel := map[string]float64{}
	if len(payload) == 0 {
		return usageByModel
	}
	rows, ok := payload[0].([]any)
	if !ok {
		return usageByModel
	}
	for _, item := range rows {
		row, ok := googleAIStudioPayloadRow(item)
		if !ok || len(row) < 2 {
			continue
		}
		modelID := googleAIStudioStringAt(row, 1)
		if modelID == "" {
			continue
		}
		if _, ok := targetModelIDs[modelID]; !ok {
			continue
		}
		value, ok := googleAIStudioMetricTimeSeriesMaxValue(row[0])
		if !ok {
			continue
		}
		if current, exists := usageByModel[modelID]; !exists || value > current {
			usageByModel[modelID] = value
		}
	}
	return usageByModel
}

func googleAIStudioMetricTimeSeriesMaxValue(value any) (float64, bool) {
	values := []float64{}
	googleAIStudioCollectMetricTimeSeriesValues(value, &values)
	if len(values) == 0 {
		return 0, false
	}
	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue, true
}

func googleAIStudioCollectMetricTimeSeriesValues(value any, values *[]float64) {
	row, ok := value.([]any)
	if !ok {
		return
	}
	if metricValue, ok := googleAIStudioMetricTimeSeriesPointValue(row); ok {
		*values = append(*values, metricValue)
		return
	}
	for _, item := range row {
		googleAIStudioCollectMetricTimeSeriesValues(item, values)
	}
}

func googleAIStudioMetricTimeSeriesPointValue(row []any) (float64, bool) {
	if len(row) != 2 || !googleAIStudioMetricTimeSeriesTimestamp(row[0]) {
		return 0, false
	}
	values, ok := row[1].([]any)
	if !ok || len(values) == 0 {
		return 0, false
	}
	return numberFromAny(values[0])
}

func googleAIStudioMetricTimeSeriesTimestamp(value any) bool {
	row, ok := value.([]any)
	if !ok || len(row) == 0 {
		return false
	}
	timestamp, ok := row[0].(string)
	return ok && strings.TrimSpace(timestamp) != ""
}

type googleAIStudioMetricTimeSeriesUsageKey struct {
	ModelID   string
	QuotaType string
}

func newGoogleAIStudioMetricTimeSeriesUsageKey(modelID string, quotaType string) googleAIStudioMetricTimeSeriesUsageKey {
	return googleAIStudioMetricTimeSeriesUsageKey{
		ModelID:   strings.TrimSpace(modelID),
		QuotaType: strings.TrimSpace(quotaType),
	}
}
