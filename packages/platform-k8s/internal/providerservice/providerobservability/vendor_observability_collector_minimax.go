package providerobservability

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

const (
	minimaxTextRemainingCountMetric    = providerQuotaRemainingMetric
	minimaxTextTotalCountMetric        = providerQuotaLimitMetric
	minimaxTextRemainingPercentMetric  = providerQuotaRemainingFractionPercentMetric
	minimaxTextResetTimestampMetric    = providerQuotaResetTimestampMetric
	minimaxRemainsCollectorID          = "minimax-remains"
	defaultMinimaxRemainsCNURL         = "https://www.minimaxi.com/v1/api/openplatform/coding_plan/remains"
	defaultMinimaxRemainsGlobalURL     = "https://www.minimax.io/v1/api/openplatform/coding_plan/remains"
	minimaxUnsupportedHostErrorMessage = "minimax remains is unavailable for surface host"
)

var (
	minimaxRemainsCNURL     = defaultMinimaxRemainsCNURL
	minimaxRemainsGlobalURL = defaultMinimaxRemainsGlobalURL
)

func init() {
	registerVendorCollectorFactory(minimaxRemainsCollectorID, NewMinimaxObservabilityCollector)
}

func NewMinimaxObservabilityCollector() ObservabilityCollector {
	return &minimaxObservabilityCollector{}
}

type minimaxObservabilityCollector struct{}

func (c *minimaxObservabilityCollector) CollectorID() string {
	return minimaxRemainsCollectorID
}

func (c *minimaxObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	remainsURL, err := minimaxRemainsURL(input.SurfaceBaseURL)
	if err != nil {
		return nil, err
	}
	result, err := executeHTTPProbe(ctx, httpProbeSpec{
		CollectorName: "minimax remains",
		URL:           remainsURL,
		HTTPClient:    input.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	rows, err := parseMinimaxRemainsGaugeRows(result.Body)
	if err != nil {
		return nil, err
	}
	return &ObservabilityCollectResult{GaugeRows: rows}, nil
}

func minimaxRemainsURL(surfaceBaseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(surfaceBaseURL))
	if err != nil {
		return "", fmt.Errorf("providerobservability: parse minimax surface base url: %w", err)
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	switch {
	case host == "minimaxi.com", strings.HasSuffix(host, ".minimaxi.com"):
		return minimaxRemainsCNURL, nil
	case host == "minimax.io", strings.HasSuffix(host, ".minimax.io"):
		return minimaxRemainsGlobalURL, nil
	default:
		return "", fmt.Errorf("providerobservability: %s %q", minimaxUnsupportedHostErrorMessage, host)
	}
}
