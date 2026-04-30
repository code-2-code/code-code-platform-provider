package providerobservability

import (
	"context"
)

const (
	// meituanTokenUsageMetric records token usage from the LongCat console
	// tokenUsage API, labelled by model_id and token_type.
	meituanTokenUsageMetric = providerUsageTokensMetric

	meituanLongcatCollectorID = "meituan-longcat-token-usage"

	// meituanTokenUsageURL is the LongCat console token usage endpoint.
	meituanTokenUsageURL = "https://longcat.chat/api/lc-platform/v1/tokenUsage?day=today"
)

func init() {
	registerVendorCollectorFactory(meituanLongcatCollectorID, NewMeituanLongcatObservabilityCollector)
}

// NewMeituanLongcatObservabilityCollector returns a collector that probes
// the LongCat console tokenUsage endpoint.
func NewMeituanLongcatObservabilityCollector() ObservabilityCollector {
	return &meituanLongcatObservabilityCollector{}
}

type meituanLongcatObservabilityCollector struct{}

func (c *meituanLongcatObservabilityCollector) CollectorID() string {
	return meituanLongcatCollectorID
}

func (c *meituanLongcatObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	result, err := executeHTTPProbe(ctx, httpProbeSpec{
		CollectorName: "meituan longcat token usage",
		URL:           meituanTokenUsageURL,
		HTTPClient:    input.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	rows, err := parseMeituanTokenUsageGaugeRows(result.Body)
	if err != nil {
		return nil, err
	}
	return &ObservabilityCollectResult{GaugeRows: rows}, nil
}
