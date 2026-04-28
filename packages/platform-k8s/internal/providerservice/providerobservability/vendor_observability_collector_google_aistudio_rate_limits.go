package providerobservability

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type googleAIStudioQuotaModel struct {
	ModelID  string
	Category string
	Preview  bool
	TierCode int
	Limits   []googleAIStudioQuotaLimit
}

type googleAIStudioQuotaLimit struct {
	Resource     string
	Window       string
	QuotaType    string
	Value        float64
	Remaining    float64
	HasRemaining bool
}

const googleAIStudioTextOutputModelCategoryCode = 4
const googleAIStudioTextOutputModelCategory = "text_output"
const googleAIStudioGemmaModelCategory = "gemma"

type googleAIStudioTierHint struct {
	TierCode int
}

type googleAIStudioRateLimitEntry struct {
	ModelID       string
	ModelCategory string
	TierCode      int
	ResourceCode  int
	WindowCode    int
	Value         float64
}

func parseGoogleAIStudioRateLimits(
	payload []any,
	tierHint googleAIStudioTierHint,
	modelMeta map[string]googleAIStudioQuotaModelMeta,
) ([]googleAIStudioQuotaModel, error) {
	rows, err := googleAIStudioPayloadRows(payload)
	if err != nil {
		return nil, err
	}
	entries := make([]googleAIStudioRateLimitEntry, 0, len(rows))
	tierCounts := map[int]int{}
	for _, item := range rows {
		row, ok := googleAIStudioPayloadRow(item)
		if !ok {
			continue
		}
		modelID := googleAIStudioStringAt(row, 0)
		if modelID == "" {
			continue
		}
		meta, ok := modelMeta[modelID]
		if !ok {
			continue
		}
		tierCode, ok := googleAIStudioIntAt(row, 1)
		if !ok {
			continue
		}
		resourceCode, ok := googleAIStudioIntAt(row, 2)
		if !ok {
			continue
		}
		windowCode, ok := googleAIStudioIntAt(row, 3)
		if !ok {
			continue
		}
		categoryCode, hasCategoryCode := googleAIStudioIntAt(row, 5)
		modelCategory, supported := googleAIStudioSupportedModelCategory(meta, modelID, categoryCode, hasCategoryCode)
		if !supported {
			continue
		}
		tierCounts[tierCode]++
		value, ok := googleAIStudioLimitValue(row, 4)
		if !ok || value <= 0 {
			continue
		}
		entries = append(entries, googleAIStudioRateLimitEntry{
			ModelID:       modelID,
			ModelCategory: modelCategory,
			TierCode:      tierCode,
			ResourceCode:  resourceCode,
			WindowCode:    windowCode,
			Value:         value,
		})
	}
	selectedTierCode, selected := googleAIStudioSelectTierCode(tierHint, tierCounts)
	if !selected {
		return nil, errGoogleAIStudioNoRows("ListModelRateLimits")
	}
	modelsByID := map[string]*googleAIStudioQuotaModel{}
	for _, entry := range entries {
		if entry.TierCode != selectedTierCode {
			continue
		}
		meta, ok := modelMeta[entry.ModelID]
		if !ok {
			continue
		}
		model := modelsByID[entry.ModelID]
		if model == nil {
			model = &googleAIStudioQuotaModel{
				ModelID:  entry.ModelID,
				Category: entry.ModelCategory,
				Preview:  meta.Preview,
				TierCode: selectedTierCode,
			}
			modelsByID[entry.ModelID] = model
		}
		model.Limits = append(model.Limits, googleAIStudioQuotaLimit{
			Resource:  googleAIStudioQuotaResource(entry.ResourceCode),
			Window:    googleAIStudioQuotaWindow(entry.WindowCode),
			QuotaType: googleAIStudioQuotaType(entry.ResourceCode, entry.WindowCode),
			Value:     entry.Value,
		})
	}
	if len(modelsByID) == 0 {
		return nil, errGoogleAIStudioNoRows("ListModelRateLimits")
	}
	modelIDs := make([]string, 0, len(modelsByID))
	for modelID := range modelsByID {
		modelIDs = append(modelIDs, modelID)
	}
	sort.Strings(modelIDs)
	models := make([]googleAIStudioQuotaModel, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		model := modelsByID[modelID]
		filteredLimits := model.Limits[:0]
		for _, limit := range model.Limits {
			if limit.QuotaType == "" {
				continue
			}
			filteredLimits = append(filteredLimits, limit)
		}
		model.Limits = filteredLimits
		if len(model.Limits) == 0 {
			continue
		}
		sort.Slice(model.Limits, func(i, j int) bool {
			if model.Limits[i].QuotaType == model.Limits[j].QuotaType {
				return model.Limits[i].Window < model.Limits[j].Window
			}
			return model.Limits[i].QuotaType < model.Limits[j].QuotaType
		})
		models = append(models, *model)
	}
	if len(models) == 0 {
		return nil, errGoogleAIStudioNoRows("ListModelRateLimits")
	}
	return models, nil
}

func googleAIStudioSelectTierCode(tierHint googleAIStudioTierHint, tierCounts map[int]int) (int, bool) {
	if tierHint.TierCode > 0 {
		return tierHint.TierCode, true
	}
	selected := false
	selectedTierCode := 0
	selectedCount := 0
	for tierCode, count := range tierCounts {
		if !selected || count > selectedCount || (count == selectedCount && tierCode < selectedTierCode) {
			selectedTierCode = tierCode
			selectedCount = count
			selected = true
		}
	}
	return selectedTierCode, selected
}

func googleAIStudioSupportedModelCategory(
	meta googleAIStudioQuotaModelMeta,
	modelID string,
	categoryCode int,
	hasCategoryCode bool,
) (string, bool) {
	if googleAIStudioTextOutputCategoryCode(meta, categoryCode, hasCategoryCode) {
		return googleAIStudioTextOutputModelCategory, true
	}
	if googleAIStudioGemmaModel(modelID) {
		return googleAIStudioGemmaModelCategory, true
	}
	return "", false
}

func googleAIStudioTextOutputCategoryCode(meta googleAIStudioQuotaModelMeta, categoryCode int, hasCategoryCode bool) bool {
	if hasCategoryCode {
		return categoryCode == googleAIStudioTextOutputModelCategoryCode
	}
	return meta.CategoryCode == googleAIStudioTextOutputModelCategoryCode
}

func googleAIStudioGemmaModel(modelID string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(modelID)), "gemma-")
}

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

func googleAIStudioLimitValue(row []any, index int) (float64, bool) {
	values, ok := rowValueSlice(row, index)
	if !ok || len(values) == 0 {
		return 0, false
	}
	return numberFromAny(values[0])
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
