package providerobservability

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type mistralBillingResponse struct {
	Completion *mistralBillingModelUsageCategory `json:"completion"`
	OCR        *mistralBillingModelUsageCategory `json:"ocr"`
	Connectors *mistralBillingModelUsageCategory `json:"connectors"`
	Libraries  *mistralBillingLibrariesCategory  `json:"libraries_api"`
	FineTuning *mistralBillingFineTuningCategory `json:"fine_tuning"`
	Audio      *mistralBillingModelUsageCategory `json:"audio"`

	VibeUsage      *float64              `json:"vibe_usage"`
	StartDate      string                `json:"start_date"`
	EndDate        string                `json:"end_date"`
	Currency       string                `json:"currency"`
	CurrencySymbol string                `json:"currency_symbol"`
	Prices         []mistralBillingPrice `json:"prices"`
}

type mistralBillingModelUsageCategory struct {
	Models map[string]mistralBillingModelUsageData `json:"models"`
}

type mistralBillingLibrariesCategory struct {
	Pages  *mistralBillingModelUsageCategory `json:"pages"`
	Tokens *mistralBillingModelUsageCategory `json:"tokens"`
}

type mistralBillingFineTuningCategory struct {
	Training map[string]mistralBillingModelUsageData `json:"training"`
	Storage  map[string]mistralBillingModelUsageData `json:"storage"`
}

type mistralBillingModelUsageData struct {
	Input  []mistralBillingUsageEntry `json:"input"`
	Output []mistralBillingUsageEntry `json:"output"`
	Cached []mistralBillingUsageEntry `json:"cached"`
}

type mistralBillingUsageEntry struct {
	UsageType          string   `json:"usage_type"`
	EventType          string   `json:"event_type"`
	BillingMetric      string   `json:"billing_metric"`
	BillingDisplayName string   `json:"billing_display_name"`
	BillingGroup       string   `json:"billing_group"`
	Timestamp          string   `json:"timestamp"`
	Value              *float64 `json:"value"`
	ValuePaid          *float64 `json:"value_paid"`
}

type mistralBillingPrice struct {
	EventType     string `json:"event_type"`
	BillingMetric string `json:"billing_metric"`
	BillingGroup  string `json:"billing_group"`
	Price         string `json:"price"`
}

type mistralBillingLimitsResponse struct {
	Limits *mistralBillingLimits `json:"limits"`
}

type mistralBillingLimits struct {
	Completion *mistralCompletionLimits `json:"completion"`
}

type mistralCompletionLimits struct {
	UsageLimit          *float64                        `json:"usage_limit"`
	NoMonthlyLimit      *bool                           `json:"no_monthly_limit"`
	Usage               *float64                        `json:"usage"`
	VibeUsage           *float64                        `json:"vibe_usage"`
	TotalUsage          *float64                        `json:"total_usage"`
	TokensLimitsByModel map[string]mistralTokenLimits   `json:"tokens_limits_by_model"`
	ModelRequestLimits  map[string]mistralRequestLimits `json:"model_request_limits"`
}

type mistralTokenLimits struct {
	TokensPerMinute *float64 `json:"tokens_per_minute"`
	TokensPerMonth  *float64 `json:"tokens_per_month"`
}

type mistralRequestLimits struct {
	RequestsPerSecond *float64 `json:"requests_per_second"`
}

func parseMistralBillingGaugeRows(body []byte) ([]ObservabilityMetricRow, error) {
	var payload mistralBillingResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("providerobservability: mistral billing: decode response: %w", err)
	}
	if !payload.hasUsageShape() {
		return nil, nil
	}

	currency := normalizeMistralCurrency(payload.Currency)
	currencySymbol := normalizeMistralCurrencySymbol(payload.CurrencySymbol, currency)
	prices := mistralBillingPriceIndex(payload.Prices)
	rows := make([]ObservabilityMetricRow, 0, len(payload.Prices)+8)
	totalCost := 0.0

	appendCategory := func(category string, models map[string]mistralBillingModelUsageData, emitTokens bool) {
		for modelID, modelUsage := range models {
			modelID = strings.TrimSpace(modelID)
			if modelID == "" {
				continue
			}
			if emitTokens {
				rows = append(rows, mistralBillingTokenRows(modelID, modelUsage)...)
			}
			modelCostRows, modelCost := mistralBillingModelCostRows(modelID, category, modelUsage, prices, currency, currencySymbol)
			totalCost += modelCost
			rows = append(rows, modelCostRows...)
		}
	}

	if payload.Completion != nil {
		appendCategory("completion", payload.Completion.Models, true)
	}
	for _, category := range []struct {
		name  string
		value *mistralBillingModelUsageCategory
	}{
		{name: "ocr", value: payload.OCR},
		{name: "connectors", value: payload.Connectors},
		{name: "audio", value: payload.Audio},
	} {
		if category.value != nil {
			appendCategory(category.name, category.value.Models, false)
		}
	}
	if payload.Libraries != nil {
		if payload.Libraries.Pages != nil {
			appendCategory("libraries_api.pages", payload.Libraries.Pages.Models, false)
		}
		if payload.Libraries.Tokens != nil {
			appendCategory("libraries_api.tokens", payload.Libraries.Tokens.Models, false)
		}
	}
	if payload.FineTuning != nil {
		appendCategory("fine_tuning.training", payload.FineTuning.Training, false)
		appendCategory("fine_tuning.storage", payload.FineTuning.Storage, false)
	}
	rows = append(rows, mistralBillingCostRow(totalCost, "total", "all", "", "", "", currency, currencySymbol))
	if resetAt, ok := parseMistralBillingResetTimestamp(payload.EndDate); ok {
		rows = append(rows, mistralQuotaMetricRow(
			providerQuotaResetTimestampMetric,
			"",
			"cost",
			"month",
			"billing_cap",
			float64(resetAt.Unix()),
		))
	}
	return rows, nil
}

func parseMistralBillingLimitsGaugeRows(body []byte) ([]ObservabilityMetricRow, error) {
	var payload mistralBillingLimitsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("providerobservability: mistral limits: decode response: %w", err)
	}
	if payload.Limits == nil || payload.Limits.Completion == nil {
		return nil, nil
	}
	limits := payload.Limits.Completion
	rows := make([]ObservabilityMetricRow, 0, len(limits.TokensLimitsByModel)*3+len(limits.ModelRequestLimits)+3)
	rows = append(rows, mistralBillingCapRows(limits)...)
	for modelID, modelLimits := range limits.TokensLimitsByModel {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if value := positiveFloat(modelLimits.TokensPerMinute); value > 0 {
			rows = append(rows, mistralQuotaMetricRow(providerQuotaLimitMetric, modelID, "tokens", "minute", "", value))
		}
		monthLimit := positiveFloat(modelLimits.TokensPerMonth)
		if monthLimit > 0 {
			rows = append(rows, mistralQuotaMetricRow(providerQuotaLimitMetric, modelID, "tokens", "month", "", monthLimit))
		}
	}
	for modelID, requestLimits := range limits.ModelRequestLimits {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if value := positiveFloat(requestLimits.RequestsPerSecond); value > 0 {
			rows = append(rows, mistralQuotaMetricRow(providerQuotaLimitMetric, modelID, "requests", "second", "", value))
		}
	}
	return rows, nil
}

func mistralBillingCapRows(limits *mistralCompletionLimits) []ObservabilityMetricRow {
	if limits == nil || boolValue(limits.NoMonthlyLimit) {
		return nil
	}
	limit := positiveFloat(limits.UsageLimit)
	if limit <= 0 {
		return nil
	}
	usage := nonNegativeFloat(limits.TotalUsage)
	if usage == 0 {
		usage = nonNegativeFloat(limits.Usage) + nonNegativeFloat(limits.VibeUsage)
	}
	remaining := math.Max(0, limit-usage)
	return []ObservabilityMetricRow{
		mistralQuotaMetricRow(providerQuotaLimitMetric, "", "cost", "month", "billing_cap", limit),
		mistralQuotaMetricRow(providerQuotaUsageMetric, "", "cost", "month", "billing_cap", usage),
		mistralQuotaMetricRow(providerQuotaRemainingMetric, "", "cost", "month", "billing_cap", remaining),
	}
}

func mistralQuotaMetricRow(metricName, modelID, resource, window, poolID string, value float64) ObservabilityMetricRow {
	labels := map[string]string{
		"resource":      resource,
		"window":        window,
		"quota_pool_id": poolID,
	}
	if modelID != "" {
		labels["model_id"] = modelID
	}
	return ObservabilityMetricRow{
		MetricName: metricName,
		Labels:     labels,
		Value:      value,
	}
}

func positiveFloat(value *float64) float64 {
	if value == nil || !isFinitePositive(*value) {
		return 0
	}
	return *value
}

func nonNegativeFloat(value *float64) float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
		return 0
	}
	return *value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func isFinitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

func (r mistralBillingResponse) hasUsageShape() bool {
	return r.Completion != nil ||
		r.OCR != nil ||
		r.Connectors != nil ||
		r.Libraries != nil ||
		r.FineTuning != nil ||
		r.Audio != nil ||
		r.VibeUsage != nil ||
		strings.TrimSpace(r.Currency) != "" ||
		strings.TrimSpace(r.CurrencySymbol) != "" ||
		len(r.Prices) > 0
}

func mistralBillingTokenRows(modelID string, modelUsage mistralBillingModelUsageData) []ObservabilityMetricRow {
	rows := make([]ObservabilityMetricRow, 0, 3)
	appendTokenRow := func(tokenType string, entries []mistralBillingUsageEntry) {
		for billingModelID, aggregate := range mistralBillingTokenTotalsByModel(modelID, entries) {
			if aggregate.Value <= 0 {
				continue
			}
			labels := mistralBillingModelLabels(billingModelID, aggregate.DisplayModelID, aggregate.ModelLabel)
			labels["gen_ai.provider.name"] = mistralOTelProviderName
			labels["resource"] = "tokens"
			labels["window"] = "month"
			labels["token_type"] = tokenType
			rows = append(rows, ObservabilityMetricRow{
				MetricName: mistralBillingTokensMetric,
				Labels:     labels,
				Value:      float64(int64(math.Round(aggregate.Value))),
			})
		}
	}
	appendTokenRow("input", modelUsage.Input)
	appendTokenRow("output", modelUsage.Output)
	appendTokenRow("cached", modelUsage.Cached)
	return rows
}

func mistralBillingPriceIndex(prices []mistralBillingPrice) map[string]float64 {
	index := map[string]float64{}
	for _, price := range prices {
		metric := strings.TrimSpace(price.BillingMetric)
		group := strings.TrimSpace(price.BillingGroup)
		rawPrice := strings.TrimSpace(price.Price)
		if metric == "" || group == "" || rawPrice == "" {
			continue
		}
		value, err := strconv.ParseFloat(rawPrice, 64)
		if err != nil || !isFinitePositive(value) {
			continue
		}
		index[mistralBillingPriceKey(metric, group)] = value
	}
	return index
}

type mistralBillingUsageAggregate struct {
	Value          float64
	DisplayModelID string
	ModelLabel     string
}

func mistralBillingTokenTotalsByModel(modelID string, entries []mistralBillingUsageEntry) map[string]mistralBillingUsageAggregate {
	groups := map[string]mistralBillingUsageAggregate{}
	for _, entry := range entries {
		billingModelID := mistralBillingEntryModelID(modelID, entry)
		aggregate := groups[billingModelID]
		aggregate.DisplayModelID = mistralBillingDisplayModelID(modelID)
		aggregate.ModelLabel = mistralBillingEntryModelLabel(modelID, entry)
		aggregate.Value += mistralBillingUsageValue(entry)
		groups[billingModelID] = aggregate
	}
	return groups
}

func mistralBillingModelCostRows(
	modelID string,
	category string,
	modelUsage mistralBillingModelUsageData,
	prices map[string]float64,
	currency string,
	currencySymbol string,
) ([]ObservabilityMetricRow, float64) {
	groups := map[string]mistralBillingUsageAggregate{}
	appendEntries := func(entries []mistralBillingUsageEntry) {
		for _, entry := range entries {
			value := mistralBillingEntryCost(entry, prices)
			if value <= 0 {
				continue
			}
			billingModelID := mistralBillingEntryModelID(modelID, entry)
			aggregate := groups[billingModelID]
			aggregate.DisplayModelID = mistralBillingDisplayModelID(modelID)
			aggregate.ModelLabel = mistralBillingEntryModelLabel(modelID, entry)
			aggregate.Value += value
			groups[billingModelID] = aggregate
		}
	}
	appendEntries(modelUsage.Input)
	appendEntries(modelUsage.Output)
	appendEntries(modelUsage.Cached)

	rows := make([]ObservabilityMetricRow, 0, len(groups))
	var total float64
	for billingModelID, aggregate := range groups {
		if aggregate.Value <= 0 {
			continue
		}
		total += aggregate.Value
		rows = append(rows, mistralBillingCostRow(
			aggregate.Value,
			"model",
			category,
			billingModelID,
			aggregate.DisplayModelID,
			aggregate.ModelLabel,
			currency,
			currencySymbol,
		))
	}
	return rows, total
}

func mistralBillingEntryCost(entry mistralBillingUsageEntry, prices map[string]float64) float64 {
	metric := strings.TrimSpace(entry.BillingMetric)
	group := strings.TrimSpace(entry.BillingGroup)
	if metric == "" || group == "" {
		return 0
	}
	price := prices[mistralBillingPriceKey(metric, group)]
	if price <= 0 {
		return 0
	}
	value := mistralBillingBillableValue(entry)
	if value <= 0 {
		return 0
	}
	return value * price
}

func mistralBillingUsageValue(entry mistralBillingUsageEntry) float64 {
	if entry.Value != nil && isFinitePositive(*entry.Value) {
		return *entry.Value
	}
	if entry.ValuePaid != nil && isFinitePositive(*entry.ValuePaid) {
		return *entry.ValuePaid
	}
	return 0
}

func mistralBillingBillableValue(entry mistralBillingUsageEntry) float64 {
	if entry.ValuePaid != nil && isFinitePositive(*entry.ValuePaid) {
		return *entry.ValuePaid
	}
	if entry.Value != nil && isFinitePositive(*entry.Value) {
		return *entry.Value
	}
	return 0
}

func mistralBillingPriceKey(metric, group string) string {
	return strings.TrimSpace(metric) + "::" + strings.TrimSpace(group)
}

func mistralBillingEntryModelID(modelID string, entry mistralBillingUsageEntry) string {
	billingModelID := strings.TrimSpace(entry.BillingMetric)
	if billingModelID != "" {
		return billingModelID
	}
	displayModelID := mistralBillingDisplayModelID(modelID)
	if displayModelID != "" {
		return displayModelID
	}
	return strings.TrimSpace(modelID)
}

func mistralBillingEntryModelLabel(modelID string, entry mistralBillingUsageEntry) string {
	if label := strings.TrimSpace(entry.BillingDisplayName); label != "" {
		return label
	}
	return mistralBillingDisplayModelID(modelID)
}

func mistralBillingDisplayModelID(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if left, _, ok := strings.Cut(modelID, "::"); ok && strings.TrimSpace(left) != "" {
		return strings.TrimSpace(left)
	}
	return modelID
}

func mistralBillingModelLabels(modelID string, displayModelID string, modelLabel string) map[string]string {
	labels := map[string]string{
		"model_id":                  strings.TrimSpace(modelID),
		"mistral_ai.billing_metric": strings.TrimSpace(modelID),
	}
	if displayModelID = strings.TrimSpace(displayModelID); displayModelID != "" {
		labels["gen_ai.request.model"] = displayModelID
	}
	if modelLabel = strings.TrimSpace(modelLabel); modelLabel != "" {
		labels["model_label"] = modelLabel
	}
	return labels
}

func mistralBillingCostRow(
	value float64,
	scope string,
	category string,
	modelID string,
	displayModelID string,
	modelLabel string,
	currency string,
	currencySymbol string,
) ObservabilityMetricRow {
	labels := mistralBillingModelLabels(modelID, displayModelID, modelLabel)
	labels["gen_ai.provider.name"] = mistralOTelProviderName
	labels["resource"] = "cost"
	labels["window"] = "month"
	labels["scope"] = scope
	labels["usage_category"] = category
	labels["currency"] = currency
	labels["currency_symbol"] = currencySymbol
	if strings.TrimSpace(modelID) == "" {
		delete(labels, "model_id")
		delete(labels, "mistral_ai.billing_metric")
	}
	return ObservabilityMetricRow{
		MetricName: mistralBillingCostMetric,
		Labels:     labels,
		Value:      value,
	}
}

func normalizeMistralCurrency(value string) string {
	currency := strings.ToUpper(strings.TrimSpace(value))
	if currency == "" {
		return "EUR"
	}
	return currency
}

func normalizeMistralCurrencySymbol(value string, currency string) string {
	symbol := strings.TrimSpace(value)
	if symbol != "" {
		return symbol
	}
	if strings.EqualFold(currency, "EUR") {
		return "\u20ac"
	}
	if strings.EqualFold(currency, "USD") {
		return "$"
	}
	return currency + " "
}

func parseMistralBillingResetTimestamp(value string) (time.Time, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed.UTC().Add(time.Second), true
		}
	}
	return time.Time{}, false
}
