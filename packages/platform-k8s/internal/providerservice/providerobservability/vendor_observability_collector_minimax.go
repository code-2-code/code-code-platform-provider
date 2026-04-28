package providerobservability

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	registerVendorObservabilityCollectorFactory(minimaxRemainsCollectorID, NewMinimaxVendorObservabilityCollector)
}

func NewMinimaxVendorObservabilityCollector() VendorObservabilityCollector {
	return &minimaxVendorObservabilityCollector{}
}

type minimaxVendorObservabilityCollector struct{}

func (c *minimaxVendorObservabilityCollector) CollectorID() string {
	return minimaxRemainsCollectorID
}

func (c *minimaxVendorObservabilityCollector) Collect(ctx context.Context, input VendorObservabilityCollectInput) (*VendorObservabilityCollectResult, error) {
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey == "" {
		return nil, unauthorizedVendorObservabilityError("minimax api key is empty")
	}
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: minimax vendor observability http client is nil")
	}
	remainsURL, err := minimaxRemainsURL(input.SurfaceBaseURL)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, remainsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: create minimax remains request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response, err := input.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: execute minimax remains request: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, vendorObservabilityMaxBodyReadSize))
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, unauthorizedVendorObservabilityError(
			fmt.Sprintf("minimax remains unauthorized: status %d %s", response.StatusCode, strings.TrimSpace(string(body))),
		)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("providerobservability: minimax remains failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	rows, err := parseMinimaxRemainsGaugeRows(body)
	if err != nil {
		return nil, err
	}
	return &VendorObservabilityCollectResult{GaugeRows: rows}, nil
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
