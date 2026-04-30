package providerobservability

import (
	"context"
)

const (
	// mistralBillingTokensMetric records token usage from the Mistral billing
	// API, labelled by model_id and token_type.
	mistralBillingTokensMetric = providerUsageTokensMetric

	mistralBillingCollectorID = "mistral-billing"

	// mistralBillingURL is the Mistral console billing endpoint.
	mistralBillingURL = "https://console.mistral.ai/billing/v2/usage"
)

func init() {
	registerVendorCollectorFactory(mistralBillingCollectorID, NewMistralObservabilityCollector)
}

// NewMistralObservabilityCollector returns a collector that probes the
// Mistral console billing/v2/usage endpoint.
func NewMistralObservabilityCollector() ObservabilityCollector {
	return &mistralObservabilityCollector{}
}

type mistralObservabilityCollector struct{}

func (c *mistralObservabilityCollector) CollectorID() string {
	return mistralBillingCollectorID
}

func (c *mistralObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	result, err := executeHTTPProbe(ctx, httpProbeSpec{
		CollectorName: "mistral billing",
		URL:           mistralBillingURL,
		HTTPClient:    input.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	rows, err := parseMistralBillingGaugeRows(result.Body)
	if err != nil {
		return nil, err
	}
	return &ObservabilityCollectResult{GaugeRows: rows}, nil
}
